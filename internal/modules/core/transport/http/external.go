package http

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core/application"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type externalPermissionRequest struct {
	Name        string           `json:"name"`
	DisplayName string           `json:"display_name"`
	Description string           `json:"description"`
	Scope       permission.Scope `json:"scope"`
}
type externalApplicationRequest struct {
	ID                 string                               `json:"id"`
	DisplayName        string                               `json:"display_name"`
	Description        string                               `json:"description"`
	Version            string                               `json:"version"`
	APIContractVersion string                               `json:"api_contract_version"`
	ServiceIdentity    string                               `json:"service_identity"`
	ServiceEndpoint    string                               `json:"service_endpoint"`
	HealthEndpoint     string                               `json:"health_endpoint"`
	FrontendRoute      string                               `json:"frontend_route"`
	SettingsRoute      string                               `json:"settings_route"`
	AccessMatch        frameworkapplication.PermissionMatch `json:"access_match"`
	AccessPermissions  []string                             `json:"access_permissions"`
	Permissions        []externalPermissionRequest          `json:"permissions"`
	Credential         string                               `json:"service_credential,omitempty"`
}
type workspaceAvailabilityRequest struct {
	Enabled bool `json:"enabled"`
}
type introspectionRequest struct {
	IdentityAssertion string `json:"identity_assertion"`
	Permission        string `json:"permission"`
}

func (h *Handler) store(w http.ResponseWriter, r *http.Request) *application.ExternalStore {
	store := h.service.External()
	if store == nil {
		httpserver.WriteError(w, r, http.StatusServiceUnavailable, "INTEGRATION_UNAVAILABLE", "External integrations are not configured")
		return nil
	}
	return store
}
func decodeExternal(w http.ResponseWriter, r *http.Request) (externalApplicationRequest, bool) {
	var request externalApplicationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid application payload")
		return request, false
	}
	return request, true
}
func toExternal(request externalApplicationRequest) application.ExternalApplication {
	permissions := make([]application.ExternalPermission, 0, len(request.Permissions))
	for _, item := range request.Permissions {
		permissions = append(permissions, application.ExternalPermission{Name: item.Name, DisplayName: item.DisplayName, Description: item.Description, Scope: item.Scope})
	}
	return application.ExternalApplication{ID: request.ID, DisplayName: request.DisplayName, Description: request.Description, Version: request.Version, APIContractVersion: request.APIContractVersion, ServiceIdentity: request.ServiceIdentity, ServiceEndpoint: request.ServiceEndpoint, HealthEndpoint: request.HealthEndpoint, FrontendRoute: request.FrontendRoute, SettingsRoute: request.SettingsRoute, Access: frameworkapplication.PermissionPolicy{Match: request.AccessMatch, Permissions: request.AccessPermissions}, Permissions: permissions}
}
func externalResponse(item application.ExternalApplication, health string) map[string]any {
	permissions := make([]map[string]any, 0, len(item.Permissions))
	for _, p := range item.Permissions {
		permissions = append(permissions, map[string]any{"name": p.Name, "display_name": p.DisplayName, "description": p.Description, "scope": p.Scope})
	}
	return map[string]any{"id": item.ID, "deployment": "external", "display_name": item.DisplayName, "description": item.Description, "version": item.Version, "api_contract_version": item.APIContractVersion, "service_identity": item.ServiceIdentity, "service_endpoint": item.ServiceEndpoint, "health_endpoint": item.HealthEndpoint, "frontend_route": item.FrontendRoute, "settings_route": item.SettingsRoute, "enabled": item.Enabled, "credential_revoked": item.CredentialRevoked, "status": item.Status, "access_match": item.Access.Match, "access_permissions": item.Access.Permissions, "permissions": permissions, "health": health}
}
func externalError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, application.ErrExternalNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "External application was not found")
	case errors.Is(err, application.ErrExternalDuplicate):
		httpserver.WriteError(w, r, http.StatusConflict, "CONFLICT", "External application or permission already exists")
	case errors.Is(err, application.ErrInvalidExternalModule):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	default:
		httpserver.WriteApplicationError(w, r, err)
	}
}
func (h *Handler) registerExternal(w http.ResponseWriter, r *http.Request) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	request, ok := decodeExternal(w, r)
	if !ok {
		return
	}
	item, credential, err := store.Register(r.Context(), application.RegisterExternalInput{Application: toExternal(request), Credential: request.Credential})
	if err != nil {
		externalError(w, r, err)
		return
	}
	response := externalResponse(item, "unknown")
	response["service_credential"] = credential
	writeJSON(w, http.StatusCreated, response)
}
func (h *Handler) listExternal(w http.ResponseWriter, r *http.Request) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	items, err := store.List(r.Context())
	if err != nil {
		externalError(w, r, err)
		return
	}
	workspaceID, workspaceRequested := uuid.Nil, false
	if raw := r.URL.Query().Get("workspace_id"); raw != "" {
		var parseErr error
		workspaceID, parseErr = uuid.Parse(raw)
		if parseErr != nil {
			httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid workspace identifier")
			return
		}
		workspaceRequested = true
	}
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		response := externalResponse(item, store.Health(r.Context(), item))
		if workspaceRequested {
			enabled, enabledErr := store.EnabledInWorkspace(r.Context(), item.ID, workspaceID)
			if enabledErr != nil {
				externalError(w, r, enabledErr)
				return
			}
			response["workspace_enabled"] = enabled
		}
		result = append(result, response)
	}
	writeJSON(w, http.StatusOK, result)
}
func (h *Handler) getExternal(w http.ResponseWriter, r *http.Request) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	item, err := store.Get(r.Context(), chi.URLParam(r, "application"))
	if err != nil {
		externalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, externalResponse(item, store.Health(r.Context(), item)))
}
func (h *Handler) updateExternal(w http.ResponseWriter, r *http.Request) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	request, ok := decodeExternal(w, r)
	if !ok {
		return
	}
	request.ID = chi.URLParam(r, "application")
	item, err := store.Update(r.Context(), toExternal(request))
	if err != nil {
		externalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, externalResponse(item, "unknown"))
}
func (h *Handler) unregisterExternal(w http.ResponseWriter, r *http.Request) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	if err := store.Unregister(r.Context(), chi.URLParam(r, "application")); err != nil {
		externalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) setWorkspaceAvailability(w http.ResponseWriter, r *http.Request) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	var request workspaceAvailabilityRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid availability payload")
		return
	}
	workspaceID, err := uuid.Parse(chi.URLParam(r, "workspace"))
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid workspace identifier")
		return
	}
	if err = store.SetWorkspaceEnabled(r.Context(), chi.URLParam(r, "application"), workspaceID, request.Enabled); err != nil {
		externalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": request.Enabled})
}
func (h *Handler) revokeCredential(w http.ResponseWriter, r *http.Request) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	if err := store.RevokeCredential(r.Context(), chi.URLParam(r, "application")); err != nil {
		externalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) rotateCredential(w http.ResponseWriter, r *http.Request) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	credential, err := store.RotateCredential(r.Context(), chi.URLParam(r, "application"))
	if err != nil {
		externalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"service_credential": credential})
}
func (h *Handler) externalStatus(w http.ResponseWriter, r *http.Request) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	item, err := store.Get(r.Context(), chi.URLParam(r, "application"))
	if err != nil {
		externalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"id": item.ID, "health": store.Health(r.Context(), item), "api_contract_version": item.APIContractVersion, "credential_revoked": item.CredentialRevoked})
}
func (h *Handler) serviceRequest(w http.ResponseWriter, r *http.Request) (application.ExternalApplication, bool) {
	store := h.store(w, r)
	if store == nil {
		return application.ExternalApplication{}, false
	}
	item, err := store.Authenticate(r.Context(), r.Header.Get("X-ApexVoid-Application-ID"), r.Header.Get("X-ApexVoid-Service-Credential"))
	if err != nil {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "SERVICE_UNAUTHENTICATED", "Valid external service credentials are required")
		return application.ExternalApplication{}, false
	}
	return item, true
}
func (h *Handler) introspectSession(w http.ResponseWriter, r *http.Request) {
	app, ok := h.serviceRequest(w, r)
	if !ok {
		return
	}
	var request introspectionRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil || strings.TrimSpace(request.IdentityAssertion) == "" || strings.TrimSpace(request.Permission) == "" {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Identity assertion and permission are required")
		return
	}
	if !h.service.OwnsExternalPermission(app, request.Permission) {
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Application does not own the requested permission")
		return
	}
	claims, err := h.issuer.Verify(request.IdentityAssertion, app.ID)
	if err != nil {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "INVALID_ASSERTION", "A valid gateway identity assertion is required")
		return
	}
	principal, err := h.authenticator.AuthenticateAccess(r.Context(), claims.AccessToken)
	if err != nil || principal.MustChangePassword {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Active session is required")
		return
	}
	if principal.UserID != claims.UserID || principal.SessionID != claims.SessionID {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "INVALID_ASSERTION", "Assertion session is no longer valid")
		return
	}
	workspaceID := claims.WorkspaceID
	if _, err = h.workspace.ResolveWorkspaceContext(r.Context(), principal.UserID, workspaceID); err != nil {
		httpserver.WriteError(w, r, http.StatusForbidden, "WORKSPACE_FORBIDDEN", "User does not have access to this workspace")
		return
	}
	enabled, err := h.service.External().EnabledInWorkspace(r.Context(), app.ID, workspaceID)
	if err != nil || !enabled {
		httpserver.WriteError(w, r, http.StatusForbidden, "APPLICATION_DISABLED", "Application is not enabled for this workspace")
		return
	}
	allowed, err := h.service.EvaluatePermission(r.Context(), principal.UserID, workspaceID, request.Permission)
	if err != nil {
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Permission is not available")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": principal.UserID, "workspace_id": workspaceID, "permission": request.Permission, "allowed": allowed})
}
func (h *Handler) integrationAvailability(w http.ResponseWriter, r *http.Request) {
	app, ok := h.serviceRequest(w, r)
	if !ok {
		return
	}
	if app.ID != chi.URLParam(r, "application") {
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "Service identity does not match application")
		return
	}
	workspaceID, err := uuid.Parse(r.URL.Query().Get("workspace_id"))
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid workspace identifier")
		return
	}
	enabled, err := h.service.External().EnabledInWorkspace(r.Context(), app.ID, workspaceID)
	if err != nil {
		externalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"application_id": app.ID, "workspace_id": workspaceID, "enabled": enabled})
}

func (h *Handler) proxyExternalFrontend(w http.ResponseWriter, r *http.Request) {
	h.proxyExternal(w, r, false)
}
func (h *Handler) proxyExternalAPI(w http.ResponseWriter, r *http.Request) {
	h.proxyExternal(w, r, true)
}

// proxyExternal is a gateway for administrator-approved same-origin services.
// It strips browser credentials and replaces them with a one-minute encrypted
// assertion that can only be opened by Core during service introspection.
func (h *Handler) proxyExternal(w http.ResponseWriter, r *http.Request, api bool) {
	store := h.store(w, r)
	if store == nil {
		return
	}
	applicationID := chi.URLParam(r, "application")
	app, err := store.Get(r.Context(), applicationID)
	if err != nil {
		externalError(w, r, err)
		return
	}
	if !api && (app.FrontendRoute == "" || !strings.HasPrefix(app.FrontendRoute, "/apps/"+applicationID)) {
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "External application has no gateway frontend")
		return
	}
	workspace, ok := organizationapi.WorkspaceContextFromContext(r.Context())
	if !ok {
		httpserver.WriteError(w, r, http.StatusForbidden, "WORKSPACE_FORBIDDEN", "Workspace context is required")
		return
	}
	enabled, err := store.EnabledInWorkspace(r.Context(), applicationID, workspace.WorkspaceID)
	if err != nil || !enabled {
		httpserver.WriteError(w, r, http.StatusForbidden, "APPLICATION_DISABLED", "Application is not enabled for this workspace")
		return
	}
	principal, principalOK := usersapi.PrincipalFromContext(r.Context())
	accessToken, tokenOK := usersapi.AccessTokenFromContext(r.Context())
	if !principalOK || !tokenOK {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
		return
	}
	allowed, err := h.service.AuthorizeExternalEntry(r.Context(), principal.UserID, workspace.WorkspaceID, app)
	if err != nil {
		httpserver.WriteApplicationError(w, r, err)
		return
	}
	if !allowed {
		httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "You do not have permission to open this application")
		return
	}
	assertion, err := h.issuer.Issue(applicationID, principal.UserID, principal.SessionID, workspace.WorkspaceID, accessToken)
	if err != nil {
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Could not establish external application identity")
		return
	}
	target, err := url.Parse(app.ServiceEndpoint)
	if err != nil {
		httpserver.WriteError(w, r, http.StatusBadGateway, "SERVICE_UNAVAILABLE", "Application service endpoint is unavailable")
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	original := proxy.Director
	proxy.Director = func(request *http.Request) {
		original(request)
		request.URL.Path = "/" + strings.TrimPrefix(chi.URLParam(r, "*"), "/")
		request.Header.Del("Authorization")
		request.Header.Del("Cookie")
		for _, header := range []string{"X-User-ID", "X-Workspace-ID", "X-Role", "X-Permissions", "X-ApexVoid-Gateway", "X-ApexVoid-Identity-Assertion"} {
			request.Header.Del(header)
		}
		request.Header.Set("X-ApexVoid-Gateway", "external-application")
		request.Header.Set("X-ApexVoid-Identity-Assertion", assertion)
	}
	proxy.ModifyResponse = func(response *http.Response) error {
		response.Header.Del("Set-Cookie")
		response.Header.Set("X-Content-Type-Options", "nosniff")
		response.Header.Set("Cache-Control", "no-store")
		return nil
	}
	proxy.ErrorHandler = func(response http.ResponseWriter, _ *http.Request, _ error) {
		httpserver.WriteError(response, r, http.StatusBadGateway, "SERVICE_UNAVAILABLE", "Application service is unavailable")
	}
	proxy.ServeHTTP(w, r)
}
