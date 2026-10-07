package api

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type Principal struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
}

type Authenticator interface {
	AuthenticateAccess(ctx context.Context, token string) (Principal, error)
	Refresh(ctx context.Context, token string, userAgent string) (string, string, Principal, error)
}

type UserReader interface {
	FindActiveByID(ctx context.Context, id uuid.UUID) (UserSummary, error)
}

type UserSummary struct {
	ID          uuid.UUID
	Email       string
	DisplayName string
	Status      string
}

type Authorizer interface {
	Can(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
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
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
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
