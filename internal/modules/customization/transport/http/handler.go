package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type Handler struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	workspace     organizationapi.WorkspaceResolver
	access        organizationapi.WorkspaceAccess
}

func NewHandler(service *application.Service, auth usersapi.Authenticator, workspace organizationapi.WorkspaceResolver, access organizationapi.WorkspaceAccess) *Handler {
	return &Handler{service: service, authenticator: auth, workspace: workspace, access: access}
}

func (h *Handler) RegisterRoutes(routes module.RouteRegistry) error {
	current := routes.With(usersapi.RequireAuthentication(h.authenticator), organizationapi.RequireWorkspace(h.workspace))
	current.With(organizationapi.RequireWorkspacePermission(h.access, "customization.schema.read")).Get("/customization/schema/{entity}", h.schema)
	fields := current.With(organizationapi.RequireWorkspacePermission(h.access, "customization.field.manage"))
	fields.Post("/customization/fields", h.createField)
	fields.Get("/customization/fields/{entity}", h.listFields)
	fields.Patch("/customization/fields/{fieldID}", h.updateField)
	fields.Post("/customization/sections", h.createSection)
	views := current.With(organizationapi.RequireWorkspacePermission(h.access, "customization.schema.read"))
	views.Get("/customization/views/{entity}", h.listViews)
	current.With(organizationapi.RequireWorkspacePermission(h.access, "customization.view.manage")).Post("/customization/views", h.createView)
	return nil
}

type fieldRequest struct {
	Entity       string     `json:"entity"`
	Key          string     `json:"key"`
	Label        string     `json:"label"`
	Type         field.Type `json:"type"`
	Description  string     `json:"description"`
	Required     bool       `json:"required"`
	DefaultValue any        `json:"default_value"`
	Options      []string   `json:"options"`
	Visible      *bool      `json:"visible"`
	DisplayOrder int        `json:"display_order"`
	SectionID    *uuid.UUID `json:"section_id"`
	Active       *bool      `json:"active"`
}

type sectionRequest struct {
	Entity       string `json:"entity"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	DisplayOrder int    `json:"display_order"`
}

type viewRequest struct {
	Entity        string          `json:"entity"`
	Name          string          `json:"name"`
	Shared        bool            `json:"shared"`
	Filters       []domain.Filter `json:"filters"`
	Columns       []string        `json:"columns"`
	SortField     string          `json:"sort_field"`
	SortDirection string          `json:"sort_direction"`
}

func (h *Handler) schema(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	item, err := h.service.EffectiveSchema(r.Context(), workspace.WorkspaceID, chi.URLParam(r, "entity"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) createField(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	var input fieldRequest
	if !decode(w, r, &input) {
		return
	}
	visible, active := true, true
	if input.Visible != nil {
		visible = *input.Visible
	}
	if input.Active != nil {
		active = *input.Active
	}
	item, err := h.service.CreateField(r.Context(), domain.RuntimeField{WorkspaceID: workspace.WorkspaceID, Entity: input.Entity, Key: input.Key, Label: input.Label, Type: input.Type, Description: input.Description, Required: input.Required, DefaultValue: input.DefaultValue, Options: input.Options, Visible: visible, DisplayOrder: input.DisplayOrder, SectionID: input.SectionID, Active: active})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

// listFields is an administrator inventory, not a consumer effective schema.
func (h *Handler) listFields(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	items, err := h.service.ListFields(r.Context(), workspace.WorkspaceID, chi.URLParam(r, "entity"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// PATCH deliberately accepts only editable attributes, never entity/key/type.
type fieldPatchRequest struct {
	Label *string `json:"label"`
	Description *string `json:"description"`
	Required *bool `json:"required"`
	DefaultValue json.RawMessage `json:"default_value"`
	Options *[]string `json:"options"`
	Visible *bool `json:"visible"`
	DisplayOrder *int `json:"display_order"`
	SectionID json.RawMessage `json:"section_id"`
	Active *bool `json:"active"`
}

func (h *Handler) updateField(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	id, ok := parseUUID(w, r, "fieldID")
	if !ok {
		return
	}
	var input fieldPatchRequest
	if !decode(w, r, &input) {
		return
	}
	patch := domain.FieldPatch{
		Label: input.Label, Description: input.Description,
		Required: input.Required, Options: input.Options,
		Visible: input.Visible, DisplayOrder: input.DisplayOrder, Active: input.Active,
	}
	if len(input.DefaultValue) > 0 {
		patch.SetDefault = true
		if err := json.Unmarshal(input.DefaultValue, &patch.DefaultValue); err != nil {
			httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Default value is invalid")
			return
		}
	}
	if len(input.SectionID) > 0 {
		patch.SetSection = true
		if string(input.SectionID) != "null" {
			var sectionID uuid.UUID
			if err := json.Unmarshal(input.SectionID, &sectionID); err != nil {
				httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Section ID is invalid")
				return
			}
			patch.SectionID = &sectionID
		}
	}
	item, err := h.service.PatchField(r.Context(), workspace.WorkspaceID, id, patch)
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (h *Handler) createSection(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	var input sectionRequest
	if !decode(w, r, &input) {
		return
	}
	item, err := h.service.CreateSection(r.Context(), domain.FormSection{WorkspaceID: workspace.WorkspaceID, Entity: input.Entity, Name: input.Name, Description: input.Description, DisplayOrder: input.DisplayOrder})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (h *Handler) listViews(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	principal, _ := usersapi.PrincipalFromContext(r.Context())
	items, err := h.service.ListViews(r.Context(), workspace.WorkspaceID, principal.UserID, chi.URLParam(r, "entity"))
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) createView(w http.ResponseWriter, r *http.Request) {
	workspace, _ := organizationapi.WorkspaceContextFromContext(r.Context())
	principal, _ := usersapi.PrincipalFromContext(r.Context())
	var input viewRequest
	if !decode(w, r, &input) {
		return
	}
	item, err := h.service.CreateView(r.Context(), domain.SavedView{WorkspaceID: workspace.WorkspaceID, OwnerUserID: principal.UserID, Entity: input.Entity, Name: input.Name, Shared: input.Shared, Filters: input.Filters, Columns: input.Columns, SortField: input.SortField, SortDirection: input.SortDirection})
	if err != nil {
		writeError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Request body is invalid")
		return false
	}
	return true
}

func parseUUID(w http.ResponseWriter, r *http.Request, parameter string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, parameter))
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Identifier is invalid")
		return uuid.Nil, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, domain.ErrConflict):
		httpserver.WriteError(w, r, http.StatusConflict, "CONFLICT", err.Error())
	default:
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	}
}
