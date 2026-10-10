package api

import (
	"context"
	"net/http"
	"net/url"
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

// RequireExternalApplicationAuthentication preserves JSON 401 responses for
// API and asset requests, but browser document navigations use the Enterprise
// session-continuation page to recover an expired access token. Refresh tokens
// remain scoped to /api/v1/auth and are never forwarded to external services.
func RequireExternalApplicationAuthentication(auth Authenticator) func(http.Handler) http.Handler {
	return requireAuthentication(auth, true)
}

func RequireAuthentication(auth Authenticator) func(http.Handler) http.Handler {
	return requireAuthentication(auth, false)
}

func requireAuthentication(auth Authenticator, externalNavigation bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("Authorization")
			if len(token) > 7 && token[:7] == "Bearer " {
				token = token[7:]
			} else if cookie, err := r.Cookie("apexvoid_access_token"); err == nil {
				token = cookie.Value
			}
			unauthenticated := func() {
				if externalNavigation && externalDocumentNavigation(r) {
					redirectToExternalContinuation(w, r, "/auth/continue")
					return
				}
				httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
			}
			if token == "" {
				unauthenticated()
				return
			}
			principal, err := auth.AuthenticateAccess(r.Context(), token)
			if err != nil {
				unauthenticated()
				return
			}
			if principal.MustChangePassword && !passwordChangeAllowed(r.URL.Path) {
				if externalNavigation && externalDocumentNavigation(r) {
					redirectToExternalContinuation(w, r, "/change-password")
					return
				}
				httpserver.WriteError(w, r, http.StatusForbidden, "PASSWORD_CHANGE_REQUIRED", "Change your password before continuing")
				return
			}
			next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
		})
	}
}

func externalDocumentNavigation(r *http.Request) bool {
	if r.Method != http.MethodGet || !strings.HasPrefix(r.URL.Path, "/apps/") {
		return false
	}
	mode := strings.ToLower(r.Header.Get("Sec-Fetch-Mode"))
	if mode == "navigate" {
		return true
	}
	if mode != "" {
		return false
	}
	// Compatibility with browsers that do not yet send Fetch Metadata.
	// API/fetch requests and static assets must keep their 401 JSON response.
	dest := strings.ToLower(r.Header.Get("Sec-Fetch-Dest"))
	return strings.Contains(strings.ToLower(r.Header.Get("Accept")), "text/html") &&
		(dest == "" || dest == "document")
}

func redirectToExternalContinuation(w http.ResponseWriter, r *http.Request, page string) {
	// Construct the destination exclusively from the already-matched /apps/
	// route, never from Host, Origin, Referer, or a user supplied redirect URL.
	query := url.Values{"return_to": {r.URL.RequestURI()}}
	http.Redirect(w, r, page+"?"+query.Encode(), http.StatusSeeOther)
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
