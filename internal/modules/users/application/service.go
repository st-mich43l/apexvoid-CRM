package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/users/domain"
)

type Service struct {
	users           domain.Repository
	sessions        domain.SessionRepository
	hasher          PasswordHasher
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

type Dependencies struct {
	Users           domain.Repository
	Sessions        domain.SessionRepository
	PasswordMinLen  int
	PasswordMaxLen  int
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func NewService(dependencies Dependencies) *Service {
	return &Service{users: dependencies.Users, sessions: dependencies.Sessions, hasher: PasswordHasher{MinLength: dependencies.PasswordMinLen, MaxLength: dependencies.PasswordMaxLen}, accessTokenTTL: dependencies.AccessTokenTTL, refreshTokenTTL: dependencies.RefreshTokenTTL}
}

type CreateInput struct {
	Email       string
	Username    string
	DisplayName string
	Password    string
}

type UpdateInput struct {
	Username    *string
	DisplayName *string
}

type LoginResult struct {
	User         domain.User
	AccessToken  string
	RefreshToken string
	Principal    api.Principal
	AccessExpiry time.Time
}

func (s *Service) Create(ctx context.Context, input CreateInput) (domain.User, error) {
	email := domain.NormalizeEmail(input.Email)
	existing, err := s.users.FindByEmail(ctx, email)
	if err == nil && existing != nil {
		return domain.User{}, domain.ErrDuplicateEmail
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return domain.User{}, err
	}
	hash, err := s.hasher.Hash(input.Password)
	if err != nil {
		return domain.User{}, domain.ErrPasswordPolicy
	}
	now := time.Now().UTC()
	user := domain.User{ID: uuid.New(), Email: email, Username: strings.TrimSpace(input.Username), DisplayName: strings.TrimSpace(input.DisplayName), PasswordHash: hash, Status: domain.StatusActive, CreatedAt: now, UpdatedAt: now}
	if err := user.Validate(); err != nil {
		return domain.User{}, err
	}
	if err := s.users.Create(ctx, &user); err != nil {
		return domain.User{}, err
	}
	return user, nil
}

func (s *Service) BootstrapAdmin(ctx context.Context, email, password string) (domain.User, bool, error) {
	count, err := s.users.Count(ctx)
	if err != nil {
		return domain.User{}, false, err
	}
	if count != 0 || strings.TrimSpace(email) == "" || password == "" {
		return domain.User{}, false, nil
	}
	user, err := s.Create(ctx, CreateInput{Email: email, DisplayName: "Administrator", Password: password})
	return user, err == nil, err
}

func (s *Service) Find(ctx context.Context, id uuid.UUID) (domain.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	return *user, nil
}

func (s *Service) FindActiveByID(ctx context.Context, id uuid.UUID) (api.UserSummary, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return api.UserSummary{}, err
	}
	if !user.IsUsable() {
		return api.UserSummary{}, domain.ErrUserDisabled
	}
	return toSummary(*user), nil
}

func (s *Service) List(ctx context.Context) ([]domain.User, error) { return s.users.List(ctx) }

func (s *Service) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (domain.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if input.Username != nil {
		user.Username = strings.TrimSpace(*input.Username)
	}
	if input.DisplayName != nil {
		user.DisplayName = strings.TrimSpace(*input.DisplayName)
	}
	user.UpdatedAt = time.Now().UTC()
	if err := user.Validate(); err != nil {
		return domain.User{}, err
	}
	if err := s.users.Update(ctx, user); err != nil {
		return domain.User{}, err
	}
	return *user, nil
}

func (s *Service) SetStatus(ctx context.Context, id uuid.UUID, status domain.Status) (domain.User, error) {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	user.Status = status
	user.UpdatedAt = time.Now().UTC()
	if err := s.users.Update(ctx, user); err != nil {
		return domain.User{}, err
	}
	if status != domain.StatusActive {
		_ = s.sessions.RevokeUserSessions(ctx, id, nil, time.Now().UTC())
	}
	return *user, nil
}

func (s *Service) Login(ctx context.Context, email, password, userAgent string) (LoginResult, error) {
	user, err := s.users.FindByEmail(ctx, domain.NormalizeEmail(email))
	if err != nil || user == nil || !user.IsUsable() || !s.hasher.Verify(user.PasswordHash, password) {
		return LoginResult{}, domain.ErrInvalidCredentials
	}
	now := time.Now().UTC()
	accessToken, err := randomToken()
	if err != nil {
		return LoginResult{}, err
	}
	refreshToken, err := randomToken()
	if err != nil {
		return LoginResult{}, err
	}
	session := domain.Session{ID: uuid.New(), UserID: user.ID, AccessTokenHash: hashToken(accessToken), RefreshTokenHash: hashToken(refreshToken), AccessExpiresAt: now.Add(s.accessTokenTTL), RefreshExpiresAt: now.Add(s.refreshTokenTTL), CreatedAt: now, UserAgent: strings.TrimSpace(userAgent)}
	if err := s.sessions.CreateSession(ctx, session); err != nil {
		return LoginResult{}, err
	}
	user.LastLoginAt = &now
	user.UpdatedAt = now
	if err := s.users.Update(ctx, user); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{User: *user, AccessToken: accessToken, RefreshToken: refreshToken, Principal: api.Principal{UserID: user.ID, SessionID: session.ID}, AccessExpiry: session.AccessExpiresAt}, nil
}

func (s *Service) AuthenticateAccess(ctx context.Context, token string) (api.Principal, error) {
	session, user, err := s.sessions.FindByAccessHash(ctx, hashToken(token))
	if err != nil || session == nil || user == nil || session.RevokedAt != nil || time.Now().UTC().After(session.AccessExpiresAt) || !user.IsUsable() {
		return api.Principal{}, domain.ErrSessionInvalid
	}
	return api.Principal{UserID: user.ID, SessionID: session.ID}, nil
}

func (s *Service) Refresh(ctx context.Context, token, userAgent string) (string, string, api.Principal, error) {
	session, user, err := s.sessions.FindByRefreshHash(ctx, hashToken(token))
	if err != nil || session == nil || user == nil || session.RevokedAt != nil || time.Now().UTC().After(session.RefreshExpiresAt) || !user.IsUsable() {
		return "", "", api.Principal{}, domain.ErrSessionInvalid
	}
	accessToken, err := randomToken()
	if err != nil {
		return "", "", api.Principal{}, err
	}
	refreshToken, err := randomToken()
	if err != nil {
		return "", "", api.Principal{}, err
	}
	now := time.Now().UTC()
	if err := s.sessions.RotateSession(ctx, session.ID, hashToken(accessToken), hashToken(refreshToken), now.Add(s.accessTokenTTL), now.Add(s.refreshTokenTTL), now); err != nil {
		return "", "", api.Principal{}, err
	}
	return accessToken, refreshToken, api.Principal{UserID: user.ID, SessionID: session.ID}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	session, _, err := s.sessions.FindByAccessHash(ctx, hashToken(token))
	if err != nil || session == nil {
		return nil
	}
	return s.sessions.RevokeSession(ctx, session.ID, time.Now().UTC())
}

func (s *Service) LogoutAll(ctx context.Context, id uuid.UUID) error {
	return s.sessions.RevokeUserSessions(ctx, id, nil, time.Now().UTC())
}

func (s *Service) ChangePassword(ctx context.Context, id uuid.UUID, current, next string) error {
	user, err := s.users.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if !s.hasher.Verify(user.PasswordHash, current) {
		return domain.ErrCurrentPassword
	}
	hash, err := s.hasher.Hash(next)
	if err != nil {
		return domain.ErrPasswordPolicy
	}
	if err := s.users.SetPassword(ctx, id, hash, time.Now().UTC()); err != nil {
		return err
	}
	return s.LogoutAll(ctx, id)
}

func (s *Service) Summary(user domain.User) api.UserSummary { return toSummary(user) }

func toSummary(user domain.User) api.UserSummary {
	return api.UserSummary{ID: user.ID, Email: user.Email, DisplayName: user.DisplayName, Status: string(user.Status)}
}

func randomToken() (string, error) {
	value := make([]byte, 32)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func hashToken(token string) []byte {
	hash := sha256.Sum256([]byte(token))
	return hash[:]
}
