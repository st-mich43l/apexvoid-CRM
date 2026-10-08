package http

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
	"net/http"
)

type Handler struct {
	s *application.Service
	a usersapi.Authenticator
	w organizationapi.WorkspaceResolver
	x organizationapi.WorkspaceAccess
}

func New(s *application.Service, a usersapi.Authenticator, w organizationapi.WorkspaceResolver, x organizationapi.WorkspaceAccess) *Handler {
	return &Handler{s, a, w, x}
}
func (h *Handler) RegisterRoutes(r module.RouteRegistry) error {
	c := r.With(usersapi.RequireAuthentication(h.a), organizationapi.RequireWorkspace(h.w))
	read := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.pipeline.read"))
	read.Get("/crm/pipeline-templates", h.templates)
	read.Get("/crm/pipelines", h.list)
	read.Get("/crm/pipelines/{id}", h.get)
	read.Get("/crm/pipelines/{id}/stages", h.stages)
	manage := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.pipeline.manage"))
	manage.Post("/crm/pipelines", h.create)
	manage.Post("/crm/pipelines/initialize", h.initialize)
	manage.Post("/crm/pipelines/{id}/set-default", h.defaultPipeline)
	stage := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.stage.manage"))
	stage.Post("/crm/pipelines/{id}/stages", h.addStage)
	stage.Post("/crm/pipelines/{id}/stages/reorder", h.reorder)
	h.registerWorkspaceRoutes(r)
	return nil
}
func ctx(r *http.Request) (uuid.UUID, uuid.UUID) {
	w, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	u, _ := usersapi.PrincipalFromContext(r.Context())
	return w.WorkspaceID, u.UserID
}
func id(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	x, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		httpserver.WriteError(w, r, 400, "VALIDATION_ERROR", "Identifier is invalid")
		return uuid.Nil, false
	}
	return x, true
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if d.Decode(v) != nil {
		httpserver.WriteError(w, r, 400, "VALIDATION_ERROR", "Request body is invalid")
		return false
	}
	return true
}
func out(w http.ResponseWriter, s int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(s)
	_ = json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, r *http.Request, e error) {
	if errors.Is(e, domain.ErrNotFound) {
		httpserver.WriteError(w, r, 404, "NOT_FOUND", e.Error())
	} else if errors.Is(e, domain.ErrConflict) {
		httpserver.WriteError(w, r, 409, "CONFLICT", e.Error())
	} else {
		httpserver.WriteError(w, r, 400, "VALIDATION_ERROR", e.Error())
	}
}
func (h *Handler) templates(w http.ResponseWriter, r *http.Request) { out(w, 200, h.s.Templates()) }
func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	x, _ := ctx(r)
	v, e := h.s.List(r.Context(), x, r.URL.Query().Get("include_archived") == "true")
	if e != nil {
		fail(w, r, e)
		return
	}
	out(w, 200, v)
}
func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	x, _ := ctx(r)
	i, ok := id(w, r)
	if !ok {
		return
	}
	v, e := h.s.Get(r.Context(), x, i)
	if e != nil {
		fail(w, r, e)
		return
	}
	out(w, 200, v)
}
func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"color"`
	}
	if !decode(w, r, &v) {
		return
	}
	x, u := ctx(r)
	p, e := h.s.Create(r.Context(), x, u, v.Name, v.Description, v.Color)
	if e != nil {
		fail(w, r, e)
		return
	}
	out(w, 201, p)
}
func (h *Handler) initialize(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Template string `json:"template"`
	}
	if !decode(w, r, &v) {
		return
	}
	x, u := ctx(r)
	p, e := h.s.Initialize(r.Context(), x, u, v.Template)
	if e != nil {
		fail(w, r, e)
		return
	}
	out(w, 201, p)
}
func (h *Handler) defaultPipeline(w http.ResponseWriter, r *http.Request) {
	x, _ := ctx(r)
	i, ok := id(w, r)
	if !ok {
		return
	}
	if e := h.s.SetDefault(r.Context(), x, i); e != nil {
		fail(w, r, e)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) stages(w http.ResponseWriter, r *http.Request) {
	x, _ := ctx(r)
	i, ok := id(w, r)
	if !ok {
		return
	}
	v, e := h.s.Stages(r.Context(), x, i, r.URL.Query().Get("include_archived") == "true")
	if e != nil {
		fail(w, r, e)
		return
	}
	out(w, 200, v)
}
func (h *Handler) addStage(w http.ResponseWriter, r *http.Request) {
	var v struct {
		Key, Name   string
		Probability int `json:"probability"`
	}
	if !decode(w, r, &v) {
		return
	}
	x, _ := ctx(r)
	i, ok := id(w, r)
	if !ok {
		return
	}
	s, e := h.s.AddStage(r.Context(), x, i, v.Key, v.Name, v.Probability)
	if e != nil {
		fail(w, r, e)
		return
	}
	out(w, 201, s)
}
func (h *Handler) reorder(w http.ResponseWriter, r *http.Request) {
	var v struct {
		StageIDs []uuid.UUID `json:"stage_ids"`
	}
	if !decode(w, r, &v) {
		return
	}
	x, _ := ctx(r)
	i, ok := id(w, r)
	if !ok {
		return
	}
	if e := h.s.Reorder(r.Context(), x, i, v.StageIDs); e != nil {
		fail(w, r, e)
		return
	}
	w.WriteHeader(204)
}
