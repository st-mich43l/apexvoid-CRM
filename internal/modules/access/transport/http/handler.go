package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	frameworkerrors "github.com/st-mich43l/apexvoid-CRM/internal/framework/errors"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/access/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/access/domain"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type Handler struct {
	service       *application.Service
	authenticator usersapi.Authenticator
}

func NewHandler(service *application.Service, authenticator usersapi.Authenticator) *Handler {
	return &Handler{service: service, authenticator: authenticator}
}

func (h *Handler) RegisterRoutes(routes module.RouteRegistry) error {
	protected := routes.With(usersapi.RequireAuthentication(h.authenticator))
	readRoles := protected.With(usersapi.RequirePermission(h.service, "access.role.read"))
	readRoles.Get("/access/roles", h.listRoles)
	readRoles.Get("/access/roles/{id}", h.getRole)
	protected.With(usersapi.RequirePermission(h.service, "access.role.create")).Post("/access/roles", h.createRole)
	protected.With(usersapi.RequirePermission(h.service, "access.role.update")).Patch("/access/roles/{id}", h.updateRole)
	protected.With(usersapi.RequirePermission(h.service, "access.role.delete")).Delete("/access/roles/{id}", h.deleteRole)
	protected.With(usersapi.RequirePermission(h.service, "access.role.update")).Put("/access/roles/{id}/permissions", h.replaceRolePermissions)
	protected.With(usersapi.RequirePermission(h.service, "access.permission.read")).Get("/access/permissions", h.permissions)
	protected.With(usersapi.RequirePermission(h.service, "access.role.read")).Get("/users/{id}/roles", h.userRoles)
	protected.With(usersapi.RequirePermission(h.service, "access.role.assign")).Put("/users/{id}/roles", h.replaceUserRoles)
	return nil
}

type roleRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}
type roleUpdateRequest struct {
	DisplayName *string `json:"display_name"`
	Description *string `json:"description"`
}
type permissionsRequest struct {
	Permissions []string `json:"permissions"`
}
type rolesRequest struct {
	RoleIDs []string `json:"role_ids"`
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := h.service.ListRoles(r.Context())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	result := make([]roleResponse, 0, len(roles))
	for _, role := range roles {
		result = append(result, h.toRoleResponse(r, role))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) getRole(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	role, err := h.service.GetRole(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toRoleResponse(r, role))
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	var request roleRequest
	if !decode(w, r, &request) {
		return
	}
	role, err := h.service.CreateRole(r.Context(), application.CreateRoleInput{Name: request.Name, DisplayName: request.DisplayName, Description: request.Description})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, h.toRoleResponse(r, role))
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	var request roleUpdateRequest
	if !decode(w, r, &request) {
		return
	}
	role, err := h.service.UpdateRole(r.Context(), id, application.UpdateRoleInput{DisplayName: request.DisplayName, Description: request.Description})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toRoleResponse(r, role))
}

func (h *Handler) deleteRole(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	if err := h.service.DeleteRole(r.Context(), id); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) replaceRolePermissions(w http.ResponseWriter, r *http.Request) {
	id, err := parseID(r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	var request permissionsRequest
	if !decode(w, r, &request) {
		return
	}
	if err := h.service.ReplaceRolePermissions(r.Context(), id, request.Permissions); err != nil {
		writeDomainError(w, r, err)
		return
	}
	role, err := h.service.GetRole(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, h.toRoleResponse(r, role))
}

func (h *Handler) permissions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.service.Permissions(r.Context()))
}

func (h *Handler) userRoles(w http.ResponseWriter, r *http.Request) {
	id, err := parseUserID(r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	roles, err := h.service.UserRoleIDs(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	result := make([]string, 0, len(roles))
	for _, role := range roles {
		result = append(result, role.String())
	}
	writeJSON(w, http.StatusOK, map[string]any{"role_ids": result})
}

func (h *Handler) replaceUserRoles(w http.ResponseWriter, r *http.Request) {
	id, err := parseUserID(r)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	var request rolesRequest
	if !decode(w, r, &request) {
		return
	}
	roleIDs := make([]uuid.UUID, 0, len(request.RoleIDs))
	for _, raw := range request.RoleIDs {
		roleID, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			writeDomainError(w, r, frameworkerrors.ValidationError("invalid role id"))
			return
		}
		roleIDs = append(roleIDs, roleID)
	}
	if err := h.service.ReplaceUserRoles(r.Context(), id, roleIDs); err != nil {
		writeDomainError(w, r, err)
		return
	}
	h.userRoles(w, r)
}

type roleResponse struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	Description string   `json:"description"`
	System      bool     `json:"system"`
	Permissions []string `json:"permissions"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

func (h *Handler) toRoleResponse(r *http.Request, role domain.Role) roleResponse {
	permissions, _ := h.service.RolePermissions(r.Context(), role.ID)
	return roleResponse{ID: role.ID.String(), Name: role.Name, DisplayName: role.DisplayName, Description: role.Description, System: role.System, Permissions: permissions, CreatedAt: role.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: role.UpdatedAt.UTC().Format(time.RFC3339)}
}

func parseID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, frameworkerrors.ValidationError("invalid role id")
	}
	return id, nil
}
func parseUserID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, frameworkerrors.ValidationError("invalid user id")
	}
	return id, nil
}
func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(target); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	case errors.Is(err, domain.ErrDuplicateRole):
		httpserver.WriteError(w, r, http.StatusConflict, "CONFLICT", "A role with that name already exists")
	case errors.Is(err, domain.ErrSystemRole), errors.Is(err, domain.ErrAdministratorRole):
		httpserver.WriteError(w, r, http.StatusConflict, "CONFLICT", "System roles are protected")
	case errors.Is(err, domain.ErrUnknownPermission):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "One or more permissions are not registered")
	case errors.Is(err, domain.ErrPermissionScope):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Permission scope does not match the role scope")
	case errors.Is(err, domain.ErrLastWorkspaceAdministrator):
		httpserver.WriteError(w, r, http.StatusConflict, "LAST_WORKSPACE_ADMINISTRATOR", "A workspace must have at least one active administrator. Assign another administrator before removing this access.")
	default:
		if strings.Contains(err.Error(), "required") {
			httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}
