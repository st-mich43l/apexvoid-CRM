package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/domain"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	usersdomain "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type Handler struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	access        api.WorkspaceAccess
}

func NewHandler(service *application.Service, authenticator usersapi.Authenticator, access api.WorkspaceAccess) *Handler {
	return &Handler{service: service, authenticator: authenticator, access: access}
}

func (h *Handler) RegisterRoutes(routes module.RouteRegistry) error {
	protected := routes.With(usersapi.RequireAuthentication(h.authenticator))
	protected.Get("/setup/status", h.setupStatus)
	protected.Post("/setup/organization", h.setupOrganization)
	protected.Get("/workspaces", h.listWorkspaces)

	current := protected.With(api.RequireWorkspace(h.service))
	current.Get("/workspace", h.currentWorkspace)
	current.With(api.RequireWorkspacePermission(h.access, "organization.organization.read")).Get("/organization", h.getOrganization)
	current.With(api.RequireWorkspacePermission(h.access, "organization.organization.update")).Patch("/organization", h.updateOrganization)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.workspace.update")).Post("/workspaces", h.createWorkspace)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.workspace.update")).Patch("/workspace", h.updateWorkspace)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.read")).Get("/workspace/members", h.listMembers)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.add")).Post("/workspace/members", h.addMember)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.add")).Get("/workspace/user-candidates", h.userCandidates)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.update")).Patch("/workspace/members/{id}", h.updateMember)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.remove")).Delete("/workspace/members/{id}", h.removeMember)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.read")).Get("/workspace/members/{id}/roles", h.memberRoles)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.update")).Put("/workspace/members/{id}/roles", h.replaceMemberRoles)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.read")).Get("/workspace/roles", h.listRoles)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.update")).Post("/workspace/roles", h.createRole)
	current.With(api.RequireWorkspacePermission(h.access, "workspace.member.update")).Put("/workspace/roles/{id}/permissions", h.replaceRolePermissions)
	explicit := current.With(h.verifyWorkspacePath)
	explicit.With(api.RequireWorkspacePermission(h.access, "workspace.workspace.read")).Get("/workspaces/{workspaceID}", h.explicitWorkspace)
	explicit.With(api.RequireWorkspacePermission(h.access, "workspace.workspace.update")).Patch("/workspaces/{workspaceID}", h.updateWorkspace)
	explicit.With(api.RequireWorkspacePermission(h.access, "workspace.member.read")).Get("/workspaces/{workspaceID}/members", h.listMembers)
	explicit.With(api.RequireWorkspacePermission(h.access, "workspace.member.add")).Post("/workspaces/{workspaceID}/members", h.addMember)
	explicit.With(api.RequireWorkspacePermission(h.access, "workspace.member.update")).Patch("/workspaces/{workspaceID}/members/{id}", h.updateMember)
	explicit.With(api.RequireWorkspacePermission(h.access, "workspace.member.remove")).Delete("/workspaces/{workspaceID}/members/{id}", h.removeMember)
	explicit.With(api.RequireWorkspacePermission(h.access, "workspace.member.read")).Get("/workspaces/{workspaceID}/members/{id}/roles", h.memberRoles)
	explicit.With(api.RequireWorkspacePermission(h.access, "workspace.member.update")).Put("/workspaces/{workspaceID}/members/{id}/roles", h.replaceMemberRoles)
	return nil
}

type setupRequest struct {
	OrganizationName string `json:"organization_name"`
	WorkspaceName    string `json:"workspace_name"`
	Timezone         string `json:"timezone"`
}
type organizationUpdateRequest struct {
	Name *string `json:"name"`
}
type workspaceRequest struct {
	Name     string `json:"name"`
	Timezone string `json:"timezone"`
}
type workspaceUpdateRequest struct {
	Name     *string `json:"name"`
	Timezone *string `json:"timezone"`
}
type memberRequest struct {
	UserID string `json:"user_id"`
}
type memberUpdateRequest struct {
	Status string `json:"status"`
}
type roleRequest struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}
type rolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}
type roleIDsRequest struct {
	RoleIDs []string `json:"role_ids"`
}

func (h *Handler) setupStatus(w http.ResponseWriter, r *http.Request) {
	principal, ok := usersapi.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
		return
	}
	status, err := h.service.Status(r.Context(), principal.UserID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) setupOrganization(w http.ResponseWriter, r *http.Request) {
	principal, ok := usersapi.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
		return
	}
	allowed, err := h.access.Can(r.Context(), principal.UserID, "organization.organization.update")
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	if !allowed {
		writeError(w, r, http.StatusForbidden, "FORBIDDEN", "Only a platform administrator can complete initial setup")
		return
	}
	var request setupRequest
	if !decode(w, r, &request) {
		return
	}
	organization, workspace, membership, err := h.service.Setup(r.Context(), principal.UserID, application.SetupInput{OrganizationName: request.OrganizationName, WorkspaceName: request.WorkspaceName, Timezone: request.Timezone})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"organization": toOrganizationResponse(organization), "workspace": toWorkspaceResponse(workspace), "membership": toMembershipResponse(membership)})
}

func (h *Handler) listWorkspaces(w http.ResponseWriter, r *http.Request) {
	principal, ok := usersapi.PrincipalFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
		return
	}
	workspaces, err := h.service.ListWorkspaces(r.Context(), principal.UserID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	result := make([]workspaceResponse, 0, len(workspaces))
	for _, workspace := range workspaces {
		result = append(result, toWorkspaceResponse(workspace))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) currentWorkspace(w http.ResponseWriter, r *http.Request) {
	workspace, ok := api.WorkspaceContextFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "WORKSPACE_FORBIDDEN", "A workspace is required")
		return
	}
	organization, err := h.service.GetOrganization(r.Context(), workspace.WorkspaceID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	item, err := h.service.GetWorkspace(r.Context(), workspace.WorkspaceID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	permissions, _ := h.access.EffectivePermissionsForWorkspace(r.Context(), workspace.UserID, workspace.WorkspaceID)
	writeJSON(w, http.StatusOK, map[string]any{"organization": toOrganizationResponse(organization), "workspace": toWorkspaceResponse(item), "permissions": permissions})
}

func (h *Handler) explicitWorkspace(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	workspace, err := h.service.GetWorkspace(r.Context(), current.WorkspaceID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toWorkspaceResponse(workspace))
}

func (h *Handler) verifyWorkspacePath(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current, ok := api.WorkspaceContextFromContext(r.Context())
		workspaceID, err := uuid.Parse(chi.URLParam(r, "workspaceID"))
		if !ok || err != nil || current.WorkspaceID != workspaceID {
			writeError(w, r, http.StatusForbidden, "WORKSPACE_FORBIDDEN", "You do not have access to this workspace")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) getOrganization(w http.ResponseWriter, r *http.Request) {
	workspace, _ := api.WorkspaceContextFromContext(r.Context())
	organization, err := h.service.GetOrganization(r.Context(), workspace.WorkspaceID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toOrganizationResponse(organization))
}

func (h *Handler) updateOrganization(w http.ResponseWriter, r *http.Request) {
	workspace, _ := api.WorkspaceContextFromContext(r.Context())
	var request organizationUpdateRequest
	if !decode(w, r, &request) {
		return
	}
	organization, err := h.service.UpdateOrganization(r.Context(), workspace.WorkspaceID, request.Name)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toOrganizationResponse(organization))
}

func (h *Handler) createWorkspace(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	var request workspaceRequest
	if !decode(w, r, &request) {
		return
	}
	workspace, err := h.service.CreateWorkspace(r.Context(), current, request.Name, request.Timezone)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toWorkspaceResponse(workspace))
}

func (h *Handler) updateWorkspace(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	var request workspaceUpdateRequest
	if !decode(w, r, &request) {
		return
	}
	workspace, err := h.service.UpdateWorkspace(r.Context(), current.WorkspaceID, request.Name, request.Timezone)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toWorkspaceResponse(workspace))
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	members, err := h.service.ListMembers(r.Context(), current.WorkspaceID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	result := make([]membershipResponse, 0, len(members))
	for _, member := range members {
		result = append(result, toMembershipResponse(member))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	var request memberRequest
	if !decode(w, r, &request) {
		return
	}
	userID, err := uuid.Parse(request.UserID)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid user id")
		return
	}
	membership, err := h.service.AddMember(r.Context(), current.WorkspaceID, userID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMembershipResponse(membership))
}

func (h *Handler) userCandidates(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUserCandidates(r.Context())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *Handler) updateMember(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid membership id")
		return
	}
	var request memberUpdateRequest
	if !decode(w, r, &request) {
		return
	}
	membership, err := h.service.UpdateMembership(r.Context(), current.WorkspaceID, id, domain.Status(request.Status))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toMembershipResponse(membership))
}

func (h *Handler) removeMember(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid membership id")
		return
	}
	if err := h.service.RemoveMember(r.Context(), current.WorkspaceID, id); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) memberRoles(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid membership id")
		return
	}
	if err := h.service.EnsureMembership(r.Context(), current.WorkspaceID, id); err != nil {
		writeDomainError(w, r, err)
		return
	}
	roles, err := h.access.MembershipRoleIDs(r.Context(), id)
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

func (h *Handler) replaceMemberRoles(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid membership id")
		return
	}
	if err := h.service.EnsureMembership(r.Context(), current.WorkspaceID, id); err != nil {
		writeDomainError(w, r, err)
		return
	}
	var request roleIDsRequest
	if !decode(w, r, &request) {
		return
	}
	roleIDs := make([]uuid.UUID, 0, len(request.RoleIDs))
	for _, raw := range request.RoleIDs {
		roleID, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid role id")
			return
		}
		roleIDs = append(roleIDs, roleID)
	}
	if err := h.access.ReplaceMembershipRoles(r.Context(), id, current.WorkspaceID, roleIDs); err != nil {
		writeDomainError(w, r, err)
		return
	}
	h.memberRoles(w, r)
}

func (h *Handler) listRoles(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	roles, err := h.access.ListWorkspaceRoles(r.Context(), current.WorkspaceID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, roles)
}

func (h *Handler) createRole(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	var request roleRequest
	if !decode(w, r, &request) {
		return
	}
	role, err := h.access.CreateWorkspaceRole(r.Context(), current.WorkspaceID, request.Name, request.DisplayName, request.Description)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, role)
}

func (h *Handler) replaceRolePermissions(w http.ResponseWriter, r *http.Request) {
	current, _ := api.WorkspaceContextFromContext(r.Context())
	roleID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid role id")
		return
	}
	var request rolePermissionsRequest
	if !decode(w, r, &request) {
		return
	}
	if err := h.access.ReplaceWorkspaceRolePermissions(r.Context(), roleID, current.WorkspaceID, request.Permissions); err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "updated"})
}

type organizationResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
type workspaceResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organization_id"`
	Name           string `json:"name"`
	Slug           string `json:"slug"`
	Timezone       string `json:"timezone"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}
type membershipResponse struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	UserID      string `json:"user_id"`
	Email       string `json:"email,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Status      string `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toOrganizationResponse(item domain.Organization) organizationResponse {
	return organizationResponse{ID: item.ID.String(), Name: item.Name, Slug: item.Slug, Status: string(item.Status), CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
}
func toWorkspaceResponse(item domain.Workspace) workspaceResponse {
	return workspaceResponse{ID: item.ID.String(), OrganizationID: item.OrganizationID.String(), Name: item.Name, Slug: item.Slug, Timezone: item.Timezone, Status: string(item.Status), CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
}
func toMembershipResponse(item domain.Membership) membershipResponse {
	return membershipResponse{ID: item.ID.String(), WorkspaceID: item.WorkspaceID.String(), UserID: item.UserID.String(), Email: item.Email, DisplayName: item.DisplayName, Status: string(item.Status), CreatedAt: item.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: item.UpdatedAt.UTC().Format(time.RFC3339)}
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(target); err != nil {
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid request body")
		return false
	}
	return true
}
func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	httpserver.WriteError(w, r, status, code, message)
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrSetupComplete):
		writeError(w, r, http.StatusConflict, "SETUP_COMPLETE", "Organization setup is already complete")
	case errors.Is(err, domain.ErrDuplicateSlug), errors.Is(err, domain.ErrDuplicateMembership):
		writeError(w, r, http.StatusConflict, "CONFLICT", "The requested resource already exists")
	case errors.Is(err, domain.ErrNotFound), errors.Is(err, domain.ErrMembershipNotFound):
		writeError(w, r, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	case errors.Is(err, domain.ErrWorkspaceInactive), errors.Is(err, domain.ErrOrganizationInactive), errors.Is(err, domain.ErrMembershipSuspended):
		writeError(w, r, http.StatusForbidden, "WORKSPACE_FORBIDDEN", "This workspace is not available")
	case errors.Is(err, usersdomain.ErrUserDisabled):
		writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "The selected user is not active")
	default:
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		if strings.Contains(err.Error(), "role") || strings.Contains(err.Error(), "permission") {
			writeError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		writeError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}
