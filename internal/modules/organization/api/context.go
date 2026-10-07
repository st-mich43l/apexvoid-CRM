package api

import (
	"context"
	"github.com/google/uuid"
	"net/http"
	"strings"

	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/domain"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type WorkspaceResolver interface {
	ResolveWorkspaceContext(ctx context.Context, userID, workspaceID uuid.UUID) (domain.WorkspaceContext, error)
	ResolveDefaultWorkspaceContext(ctx context.Context, userID uuid.UUID) (domain.WorkspaceContext, error)
}

type WorkspaceAuthorizer interface {
	CanInWorkspace(ctx context.Context, userID, workspaceID uuid.UUID, permission string) (bool, error)
}

type contextKey struct{}

func WithWorkspaceContext(ctx context.Context, workspace domain.WorkspaceContext) context.Context {
	return context.WithValue(ctx, contextKey{}, workspace)
}

func WorkspaceContextFromContext(ctx context.Context) (domain.WorkspaceContext, bool) {
	workspace, ok := ctx.Value(contextKey{}).(domain.WorkspaceContext)
	return workspace, ok
}

func RequireWorkspace(resolver WorkspaceResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := usersapi.PrincipalFromContext(r.Context())
			if !ok {
				httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
				return
			}
			raw := strings.TrimSpace(r.Header.Get("X-ApexVoid-Workspace"))
			var workspace domain.WorkspaceContext
			var err error
			if raw == "" {
				workspace, err = resolver.ResolveDefaultWorkspaceContext(r.Context(), principal.UserID)
			} else {
				workspaceID, parseErr := uuid.Parse(raw)
				if parseErr != nil {
					httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid workspace identifier")
					return
				}
				workspace, err = resolver.ResolveWorkspaceContext(r.Context(), principal.UserID, workspaceID)
			}
			if err != nil {
				httpserver.WriteError(w, r, http.StatusForbidden, "WORKSPACE_FORBIDDEN", "You do not have access to this workspace")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithWorkspaceContext(r.Context(), workspace)))
		})
	}
}

func RequireWorkspacePermission(authorizer WorkspaceAuthorizer, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, principalOK := usersapi.PrincipalFromContext(r.Context())
			workspace, workspaceOK := WorkspaceContextFromContext(r.Context())
			if !principalOK || !workspaceOK {
				httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Workspace authentication is required")
				return
			}
			allowed, err := authorizer.CanInWorkspace(r.Context(), principal.UserID, workspace.WorkspaceID, permission)
			if err != nil {
				httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
				return
			}
			if !allowed {
				httpserver.WriteError(w, r, http.StatusForbidden, "FORBIDDEN", "You do not have permission to perform this action")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
