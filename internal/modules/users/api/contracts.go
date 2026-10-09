package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type Principal struct {
	UserID             uuid.UUID
	SessionID          uuid.UUID
	MustChangePassword bool
}

type Authenticator interface {
	AuthenticateAccess(ctx context.Context, token string) (Principal, error)
	AuthenticateSession(ctx context.Context, userID, sessionID uuid.UUID) (Principal, error)
	Refresh(ctx context.Context, token string, userAgent string) (string, string, Principal, error)
}

type UserReader interface {
	FindActiveByID(ctx context.Context, id uuid.UUID) (UserSummary, error)
}

type UserDirectory interface {
	ListActive(ctx context.Context) ([]UserSummary, error)
}

type UserSummary struct {
	ID                 uuid.UUID `json:"id"`
	Email              string    `json:"email"`
	DisplayName        string    `json:"display_name"`
	Status             string    `json:"status"`
	MustChangePassword bool      `json:"must_change_password"`
}

type Authorizer interface {
	Can(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}

// UserStatusGuard lets the access/organization boundary validate a deactivation
// before the users module commits the identity change.
type UserStatusGuard interface {
	ValidateUserStatusChange(ctx context.Context, userID uuid.UUID, status string) error
}

type PermissionReader interface {
	EffectivePermissions(ctx context.Context, userID uuid.UUID) ([]string, error)
}

type contextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, contextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(contextKey{}).(Principal)
	return principal, ok
}

func RequireAuthentication(auth Authenticator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			} else {
				if cookie, err := r.Cookie("apexvoid_access_token"); err == nil {
					token = cookie.Value
				}
			}
			if token == "" {
				httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
				return
			}
			principal, err := auth.AuthenticateAccess(r.Context(), token)
			if err != nil {
				httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
				return
			}
			if principal.MustChangePassword && !passwordChangeAllowed(r.URL.Path) {
				httpserver.WriteError(w, r, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED", "Change your password before continuing")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

func passwordChangeAllowed(path string) bool {
	return strings.HasSuffix(path, "/auth/me") || strings.HasSuffix(path, "/auth/change-password") || strings.HasSuffix(path, "/auth/logout") || strings.HasSuffix(path, "/auth/logout-all")
}

func RequirePermission(authorizer Authorizer, permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
				return
			}
			if authorizer == nil {
				httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "Authorization is not configured")
				return
			}
			allowed, err := authorizer.Can(r.Context(), principal.UserID, permission)
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
