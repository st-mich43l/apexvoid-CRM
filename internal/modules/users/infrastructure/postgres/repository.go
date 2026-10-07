package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/users/domain"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Repository struct{ pool *pgxpool.Pool }

func NewRepository(pool *pgxpool.Pool) *Repository { return &Repository{pool: pool} }

type querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (r *Repository) db(ctx context.Context) querier {
	if tx, ok := database.TransactionFromContext(ctx); ok {
		return tx
	}
	return r.pool
}

func (r *Repository) Create(ctx context.Context, user *domain.User) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO users_users (id, email, username, display_name, password_hash, status, created_at, updated_at, last_login_at) VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9)`, user.ID, user.Email, user.Username, user.DisplayName, user.PasswordHash, user.Status, user.CreatedAt, user.UpdatedAt, user.LastLoginAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicateEmail
	}
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return r.find(ctx, `WHERE id = $1`, id)
}

func (r *Repository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.find(ctx, `WHERE email = $1`, email)
}

func (r *Repository) find(ctx context.Context, condition string, arg any) (*domain.User, error) {
	row := r.db(ctx).QueryRow(ctx, `SELECT id, email, COALESCE(username,''), display_name, password_hash, status, created_at, updated_at, last_login_at FROM users_users `+condition, arg)
	var user domain.User
	var status string
	if err := row.Scan(&user.ID, &user.Email, &user.Username, &user.DisplayName, &user.PasswordHash, &status, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt); errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	user.Status = domain.Status(status)
	return &user, nil
}

func (r *Repository) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT id, email, COALESCE(username,''), display_name, password_hash, status, created_at, updated_at, last_login_at FROM users_users ORDER BY display_name, email`)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	users := []domain.User{}
	for rows.Next() {
		var user domain.User
		var status string
		if err := rows.Scan(&user.ID, &user.Email, &user.Username, &user.DisplayName, &user.PasswordHash, &status, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		user.Status = domain.Status(status)
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	if err := r.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM users_users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (r *Repository) Update(ctx context.Context, user *domain.User) error {
	result, err := r.db(ctx).Exec(ctx, `UPDATE users_users SET username = NULLIF($2,''), display_name = $3, status = $4, updated_at = $5, last_login_at = $6 WHERE id = $1`, user.ID, user.Username, user.DisplayName, user.Status, user.UpdatedAt, user.LastLoginAt)
	if err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) SetPassword(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	result, err := r.db(ctx).Exec(ctx, `UPDATE users_users SET password_hash = $2, updated_at = $3 WHERE id = $1`, id, hash, now)
	if err != nil {
		return fmt.Errorf("set user password: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateSession(ctx context.Context, session domain.Session) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO users_sessions (id, user_id, access_token_hash, refresh_token_hash, access_expires_at, refresh_expires_at, created_at, last_used_at, user_agent) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, session.ID, session.UserID, session.AccessTokenHash, session.RefreshTokenHash, session.AccessExpiresAt, session.RefreshExpiresAt, session.CreatedAt, session.LastUsedAt, session.UserAgent)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (r *Repository) FindByAccessHash(ctx context.Context, hash []byte) (*domain.Session, *domain.User, error) {
	return r.findSession(ctx, `s.access_token_hash = $1`, hash)
}

func (r *Repository) FindByRefreshHash(ctx context.Context, hash []byte) (*domain.Session, *domain.User, error) {
	return r.findSession(ctx, `s.refresh_token_hash = $1`, hash)
}

func (r *Repository) findSession(ctx context.Context, condition string, arg any) (*domain.Session, *domain.User, error) {
	row := r.db(ctx).QueryRow(ctx, `SELECT s.id, s.user_id, s.access_token_hash, s.refresh_token_hash, s.access_expires_at, s.refresh_expires_at, s.revoked_at, s.created_at, s.last_used_at, s.user_agent, u.email, COALESCE(u.username,''), u.display_name, u.password_hash, u.status, u.created_at, u.updated_at, u.last_login_at FROM users_sessions s JOIN users_users u ON u.id = s.user_id WHERE `+condition, arg)
	var session domain.Session
	var user domain.User
	var status string
	if err := row.Scan(&session.ID, &session.UserID, &session.AccessTokenHash, &session.RefreshTokenHash, &session.AccessExpiresAt, &session.RefreshExpiresAt, &session.RevokedAt, &session.CreatedAt, &session.LastUsedAt, &session.UserAgent, &user.Email, &user.Username, &user.DisplayName, &user.PasswordHash, &status, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt); errors.Is(err, pgx.ErrNoRows) {
		return nil, nil, domain.ErrSessionInvalid
	} else if err != nil {
		return nil, nil, fmt.Errorf("find session: %w", err)
	}
	user.ID = session.UserID
	user.Status = domain.Status(status)
	return &session, &user, nil
}

func (r *Repository) RotateSession(ctx context.Context, id uuid.UUID, accessHash, refreshHash []byte, accessExpiresAt, refreshExpiresAt, lastUsedAt time.Time) error {
	result, err := r.db(ctx).Exec(ctx, `UPDATE users_sessions SET access_token_hash = $2, refresh_token_hash = $3, access_expires_at = $4, refresh_expires_at = $5, last_used_at = $6 WHERE id = $1 AND revoked_at IS NULL`, id, accessHash, refreshHash, accessExpiresAt, refreshExpiresAt, lastUsedAt)
	if err != nil {
		return fmt.Errorf("rotate session: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrSessionInvalid
	}
	return nil
}

func (r *Repository) RevokeSession(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := r.db(ctx).Exec(ctx, `UPDATE users_sessions SET revoked_at = COALESCE(revoked_at, $2) WHERE id = $1`, id, now)
	return err
}

func (r *Repository) RevokeUserSessions(ctx context.Context, userID uuid.UUID, except *uuid.UUID, now time.Time) error {
	if except == nil {
		_, err := r.db(ctx).Exec(ctx, `UPDATE users_sessions SET revoked_at = COALESCE(revoked_at, $2) WHERE user_id = $1`, userID, now)
		return err
	}
	_, err := r.db(ctx).Exec(ctx, `UPDATE users_sessions SET revoked_at = COALESCE(revoked_at, $3) WHERE user_id = $1 AND id <> $2`, userID, *except, now)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
