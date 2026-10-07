package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, user *User) error
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	List(ctx context.Context) ([]User, error)
	Count(ctx context.Context) (int, error)
	Update(ctx context.Context, user *User) error
	SetPassword(ctx context.Context, id uuid.UUID, hash string, now time.Time) error
}

type Session struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	AccessTokenHash  []byte
	RefreshTokenHash []byte
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
	RevokedAt        *time.Time
	CreatedAt        time.Time
	LastUsedAt       *time.Time
	UserAgent        string
}

type SessionRepository interface {
	CreateSession(ctx context.Context, session Session) error
	FindByAccessHash(ctx context.Context, hash []byte) (*Session, *User, error)
	FindByRefreshHash(ctx context.Context, hash []byte) (*Session, *User, error)
	RotateSession(ctx context.Context, id uuid.UUID, accessHash, refreshHash []byte, accessExpiresAt, refreshExpiresAt, lastUsedAt time.Time) error
	RevokeSession(ctx context.Context, id uuid.UUID, now time.Time) error
	RevokeUserSessions(ctx context.Context, userID uuid.UUID, except *uuid.UUID, now time.Time) error
}
