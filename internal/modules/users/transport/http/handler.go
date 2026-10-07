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
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/users/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/users/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type Handler struct {
	service    *application.Service
	authorizer api.Authorizer
	cfg        config.AuthConfig
}

func NewHandler(service *application.Service, authorizer api.Authorizer, cfg config.AuthConfig) *Handler {
	return &Handler{service: service, authorizer: authorizer, cfg: cfg}
}

func (h *Handler) RegisterRoutes(routes module.RouteRegistry) error {
	routes.Post("/auth/login", h.login)
	routes.Post("/auth/refresh", h.refresh)
	protected := routes.With(api.RequireAuthentication(h.service))
	protected.Post("/auth/logout", h.logout)
	protected.Post("/auth/logout-all", h.logoutAll)
	protected.Get("/auth/me", h.me)
	protected.Post("/auth/change-password", h.changePassword)
	protected.With(api.RequirePermission(h.authorizer, "users.user.read")).Get("/users", h.list)
	protected.With(api.RequirePermission(h.authorizer, "users.user.read")).Get("/users/{id}", h.get)
	protected.With(api.RequirePermission(h.authorizer, "users.user.create")).Post("/users", h.create)
	protected.With(api.RequirePermission(h.authorizer, "users.user.update")).Patch("/users/{id}", h.update)
	protected.With(api.RequirePermission(h.authorizer, "users.user.disable")).Post("/users/{id}/disable", h.disable)
	protected.With(api.RequirePermission(h.authorizer, "users.user.disable")).Post("/users/{id}/enable", h.enable)
	return nil
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type refreshResponse struct {
	User        userResponse `json:"user"`
	Permissions []string     `json:"permissions"`
}
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}
type createRequest struct {
	Email       string `json:"email"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}
type updateRequest struct {
	Username    *string `json:"username"`
	DisplayName *string `json:"display_name"`
}
type userResponse struct {
	ID                 string  `json:"id"`
	Email              string  `json:"email"`
	Username           string  `json:"username,omitempty"`
	DisplayName        string  `json:"display_name"`
	Status             string  `json:"status"`
	MustChangePassword bool    `json:"must_change_password"`
	CreatedAt          string  `json:"created_at"`
	UpdatedAt          string  `json:"updated_at"`
	LastLoginAt        *string `json:"last_login_at,omitempty"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var request loginRequest
	if !decode(w, r, &request) {
		return
	}
	result, err := h.service.Login(r.Context(), request.Email, request.Password, r.UserAgent())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	setCookies(w, result.AccessToken, result.RefreshToken, result.AccessExpiry, h.cfg)
	writeJSON(w, http.StatusOK, h.authResponse(r, result.User))
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("apexvoid_refresh_token")
	if err != nil || cookie.Value == "" {
		httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication is required")
		return
	}
	access, refresh, principal, err := h.service.Refresh(r.Context(), cookie.Value, r.UserAgent())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	user, err := h.service.Find(r.Context(), principal.UserID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	setCookies(w, access, refresh, time.Now().UTC().Add(h.cfg.AccessTokenTTL), h.cfg)
	response := h.authResponse(r, user)
	writeJSON(w, http.StatusOK, refreshResponse{User: response.User, Permissions: response.Permissions})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("apexvoid_access_token"); err == nil {
		_ = h.service.Logout(r.Context(), cookie.Value)
	}
	clearCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) logoutAll(w http.ResponseWriter, r *http.Request) {
	principal, _ := api.PrincipalFromContext(r.Context())
	if err := h.service.LogoutAll(r.Context(), principal.UserID); err != nil {
		writeDomainError(w, r, err)
		return
	}
	clearCookies(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	principal, _ := api.PrincipalFromContext(r.Context())
	user, err := h.service.Find(r.Context(), principal.UserID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, h.authResponse(r, user))
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	var request changePasswordRequest
	if !decode(w, r, &request) {
		return
	}
	principal, _ := api.PrincipalFromContext(r.Context())
	if err := h.service.ChangePassword(r.Context(), principal.UserID, principal.SessionID, request.CurrentPassword, request.NewPassword); err != nil {
		writeDomainError(w, r, err)
		return
	}
	user, err := h.service.Find(r.Context(), principal.UserID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, h.authResponse(r, user))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.List(r.Context())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	result := make([]userResponse, 0, len(users))
	for _, user := range users {
		result = append(result, toUserResponse(user))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeDomainError(w, r, frameworkerrors.ValidationError("invalid user id"))
		return
	}
	user, err := h.service.Find(r.Context(), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var request createRequest
	if !decode(w, r, &request) {
		return
	}
	user, err := h.service.Create(r.Context(), application.CreateInput{Email: request.Email, Username: request.Username, DisplayName: request.DisplayName, Password: request.Password})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toUserResponse(user))
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeDomainError(w, r, frameworkerrors.ValidationError("invalid user id"))
		return
	}
	var request updateRequest
	if !decode(w, r, &request) {
		return
	}
	user, err := h.service.Update(r.Context(), id, application.UpdateInput{Username: request.Username, DisplayName: request.DisplayName})
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

func (h *Handler) disable(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, domain.StatusInactive)
}
func (h *Handler) enable(w http.ResponseWriter, r *http.Request) {
	h.setStatus(w, r, domain.StatusActive)
}
func (h *Handler) setStatus(w http.ResponseWriter, r *http.Request, status domain.Status) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeDomainError(w, r, frameworkerrors.ValidationError("invalid user id"))
		return
	}
	user, err := h.service.SetStatus(r.Context(), id, status)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(user))
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

func toUserResponse(user domain.User) userResponse {
	response := userResponse{ID: user.ID.String(), Email: user.Email, Username: user.Username, DisplayName: user.DisplayName, Status: string(user.Status), MustChangePassword: user.MustChangePassword, CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339), UpdatedAt: user.UpdatedAt.UTC().Format(time.RFC3339)}
	if user.LastLoginAt != nil {
		value := user.LastLoginAt.UTC().Format(time.RFC3339)
		response.LastLoginAt = &value
	}
	return response
}

type authResponse struct {
	User        userResponse `json:"user"`
	Permissions []string     `json:"permissions"`
}

func (h *Handler) authResponse(r *http.Request, user domain.User) authResponse {
	permissions := []string{}
	if reader, ok := h.authorizer.(api.PermissionReader); ok {
		permissions, _ = reader.EffectivePermissions(r.Context(), user.ID)
	}
	return authResponse{User: toUserResponse(user), Permissions: permissions}
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials), errors.Is(err, domain.ErrSessionInvalid), errors.Is(err, domain.ErrUserDisabled):
		httpserver.WriteError(w, r, http.StatusUnauthorized, "UNAUTHENTICATED", "Authentication failed")
	case errors.Is(err, domain.ErrDuplicateEmail):
		httpserver.WriteError(w, r, http.StatusConflict, "CONFLICT", "A user with that email already exists")
	case errors.Is(err, domain.ErrCurrentPassword), errors.Is(err, domain.ErrPasswordPolicy):
		httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", "The password is invalid")
	case errors.Is(err, domain.ErrNotFound):
		httpserver.WriteError(w, r, http.StatusNotFound, "NOT_FOUND", "User not found")
	default:
		if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "invalid") {
			httpserver.WriteError(w, r, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
			return
		}
		httpserver.WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
	}
}

func setCookies(w http.ResponseWriter, access, refresh string, accessExpiry time.Time, cfg config.AuthConfig) {
	maxAge := int(time.Until(accessExpiry).Seconds())
	if maxAge < 1 {
		maxAge = 1
	}
	http.SetCookie(w, &http.Cookie{Name: "apexvoid_access_token", Value: access, Path: "/", HttpOnly: true, Secure: cfg.CookieSecure, SameSite: sameSite(cfg.CookieSameSite), MaxAge: maxAge})
	refreshAge := int(cfg.RefreshTokenTTL.Seconds())
	http.SetCookie(w, &http.Cookie{Name: "apexvoid_refresh_token", Value: refresh, Path: "/api/v1/auth", HttpOnly: true, Secure: cfg.CookieSecure, SameSite: sameSite(cfg.CookieSameSite), MaxAge: refreshAge})
}

func clearCookies(w http.ResponseWriter) {
	for _, item := range []struct{ name, path string }{{"apexvoid_access_token", "/"}, {"apexvoid_refresh_token", "/api/v1/auth"}} {
		http.SetCookie(w, &http.Cookie{Name: item.name, Value: "", Path: item.path, MaxAge: -1, HttpOnly: true})
	}
}

func sameSite(value string) http.SameSite {
	switch strings.ToLower(value) {
	case "strict":
		return http.SameSiteStrictMode
	case "none":
		return http.SameSiteNoneMode
	default:
		return http.SameSiteLaxMode
	}
}
