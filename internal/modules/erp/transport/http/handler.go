package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type Handler struct {
	service   *application.Service
	auth      usersapi.Authenticator
	workspace organizationapi.WorkspaceResolver
	access    organizationapi.WorkspaceAccess
}

func New(service *application.Service, auth usersapi.Authenticator, workspace organizationapi.WorkspaceResolver, access organizationapi.WorkspaceAccess) *Handler {
	return &Handler{service: service, auth: auth, workspace: workspace, access: access}
}
func (h *Handler) RegisterRoutes(r module.RouteRegistry) error {
	protected := r.With(usersapi.RequireAuthentication(h.auth), organizationapi.RequireWorkspace(h.workspace))
	read := protected.With(organizationapi.RequireWorkspacePermission(h.access, "erp.product.read"))
	read.Get("/erp/products", h.list)
	read.Get("/erp/products/{id}", h.get)
	protected.With(organizationapi.RequireWorkspacePermission(h.access, "erp.product.create")).Post("/erp/products", h.create)
	protected.With(organizationapi.RequireWorkspacePermission(h.access, "erp.product.update")).Put("/erp/products/{id}", h.update)
	archive := protected.With(organizationapi.RequireWorkspacePermission(h.access, "erp.product.archive"))
	archive.Post("/erp/products/{id}/archive", h.archive)
	archive.Post("/erp/products/{id}/restore", h.restore)
	return nil
}
func contextIDs(r *http.Request) (uuid.UUID, uuid.UUID) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	actor, _ := usersapi.PrincipalFromContext(r.Context())
	return workspace.WorkspaceID, actor.UserID
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(value); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid product payload")
		return false
	}
	var trailing any
	if err := d.Decode(&trailing); err != io.EOF {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Only one JSON object is accepted")
		return false
	}
	return true
}
func pathID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid product identifier")
		return uuid.Nil, false
	}
	return id, true
}
func fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid product fields")
	case errors.Is(err, domain.ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Product was not found in this workspace")
	case errors.Is(err, domain.ErrConflict):
		httpserver.WriteError(w, r, http.StatusConflict, "CONFLICT", "Product SKU conflicts or the version is stale")
	default:
		httpserver.WriteApplicationError(w, r, err)
	}
}
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	workspace, _ := contextIDs(r)
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, total, err := h.service.List(r.Context(), workspace, r.URL.Query().Get("search"), r.URL.Query().Get("include_archived") == "true", page, limit)
	if err != nil {
		fail(w, r, err)
		return
	}
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 25
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total, "page": page, "limit": limit})
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	workspace, _ := contextIDs(r)
	item, err := h.service.Get(r.Context(), workspace, id)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var input domain.Input
	if !decode(w, r, &input) {
		return
	}
	workspace, actor := contextIDs(r)
	item, err := h.service.Create(r.Context(), workspace, actor, input)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var input struct {
		domain.Input
		Version int `json:"version"`
	}
	if !decode(w, r, &input) {
		return
	}
	workspace, actor := contextIDs(r)
	item, err := h.service.Update(r.Context(), workspace, actor, id, input.Version, input.Input)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
func (h *Handler) archive(w http.ResponseWriter, r *http.Request) { h.setArchive(w, r, true) }
func (h *Handler) restore(w http.ResponseWriter, r *http.Request) { h.setArchive(w, r, false) }
func (h *Handler) setArchive(w http.ResponseWriter, r *http.Request, archived bool) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var input struct {
		Version int `json:"version"`
	}
	if !decode(w, r, &input) {
		return
	}
	workspace, actor := contextIDs(r)
	item, err := h.service.SetArchived(r.Context(), workspace, actor, id, input.Version, archived)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}
