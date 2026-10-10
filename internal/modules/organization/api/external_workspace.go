package api

import (
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/domain"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

const externalWorkspaceCookie = "apexvoid.external_workspace"

// RequireExternalWorkspace resolves the workspace for top-level application
// documents and subsequent assets/API requests. Browser navigation cannot send
// X-ApexVoid-Workspace headers; an explicit launch query sets a short-lived,
// HttpOnly workspace selector. Every request still checks live membership.
// The normal platform API continues to use RequireWorkspace unchanged.
func RequireExternalWorkspace(resolver WorkspaceResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := usersapi.PrincipalFromContext(r.Context())
			if !ok {
				httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
				return
			}
			selected := strings.TrimSpace(r.Header.Get("X-ApexVoid-Workspace"))
			launchWorkspace := ""
			if selected == "" && r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/apps/") {
				launchWorkspace = strings.TrimSpace(r.URL.Query().Get("workspace_id"))
				selected = launchWorkspace
			}
			if selected == "" {
				if cookie, err := r.Cookie(externalWorkspaceCookie); err == nil {
					selected = cookie.Value
				}
			}
			var workspaceID uuid.UUID
			if selected != "" {
				var err error
				workspaceID, err = uuid.Parse(selected)
				if err != nil {
					httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid workspace identifier")
					return
				}
			}
			var err error
			var workspaceContext domain.WorkspaceContext
			if selected == "" {
				workspaceContext, err = resolver.ResolveDefaultWorkspaceContext(r.Context(), principal.UserID)
			} else {
				workspaceContext, err = resolver.ResolveWorkspaceContext(r.Context(), principal.UserID, workspaceID)
			}
			if err != nil {
				httpserver.WriteError(w, r, http.StatusForbidden, "WORKSPACE_FORBIDDEN", "You do not have access to this workspace")
				return
			}
			if launchWorkspace != "" {
				http.SetCookie(w, &http.Cookie{
					Name: externalWorkspaceCookie,
					Value: workspaceID.String(),
					Path: "/",
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
					Secure: r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https"),
					MaxAge: 60 * 60 * 12,
				})
			}
			next.ServeHTTP(w, r.WithContext(WithWorkspaceContext(r.Context(), workspaceContext)))
		})
	}
}
