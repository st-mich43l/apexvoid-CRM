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

	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/domain"
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

func (r *Repository) CountOrganizations(ctx context.Context) (int, error) {
	var count int
	if err := r.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM organization_organizations`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count organizations: %w", err)
	}
	return count, nil
}

func (r *Repository) GetOrganization(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	var organization domain.Organization
	err := r.db(ctx).QueryRow(ctx, `SELECT id, name, slug, status, created_at, updated_at FROM organization_organizations WHERE id = $1`, id).Scan(&organization.ID, &organization.Name, &organization.Slug, &organization.Status, &organization.CreatedAt, &organization.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get organization: %w", err)
	}
	return &organization, nil
}

func (r *Repository) GetOrganizationForWorkspace(ctx context.Context, workspaceID uuid.UUID) (*domain.Organization, error) {
	var organization domain.Organization
	err := r.db(ctx).QueryRow(ctx, `SELECT o.id, o.name, o.slug, o.status, o.created_at, o.updated_at FROM organization_organizations o JOIN workspace_workspaces w ON w.organization_id = o.id WHERE w.id = $1`, workspaceID).Scan(&organization.ID, &organization.Name, &organization.Slug, &organization.Status, &organization.CreatedAt, &organization.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get organization for workspace: %w", err)
	}
	return &organization, nil
}

func (r *Repository) CreateOrganization(ctx context.Context, organization *domain.Organization) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO organization_organizations (id, name, slug, status, is_default, created_at, updated_at) VALUES ($1,$2,$3,$4,TRUE,$5,$6)`, organization.ID, organization.Name, organization.Slug, organization.Status, organization.CreatedAt, organization.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicateSlug
	}
	if err != nil {
		return fmt.Errorf("create organization: %w", err)
	}
	return nil
}

func (r *Repository) UpdateOrganization(ctx context.Context, organization *domain.Organization) error {
	result, err := r.db(ctx).Exec(ctx, `UPDATE organization_organizations SET name = $2, slug = $3, status = $4, updated_at = $5 WHERE id = $1`, organization.ID, organization.Name, organization.Slug, organization.Status, organization.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicateSlug
	}
	if err != nil {
		return fmt.Errorf("update organization: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT w.id, w.organization_id, w.name, w.slug, w.timezone, w.status, w.created_at, w.updated_at FROM workspace_workspaces w JOIN workspace_memberships m ON m.workspace_id = w.id JOIN organization_organizations o ON o.id = w.organization_id WHERE m.user_id = $1 AND m.status = 'active' AND w.status = 'active' AND o.status = 'active' ORDER BY w.name`, userID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()
	result := []domain.Workspace{}
	for rows.Next() {
		var workspace domain.Workspace
		if err := rows.Scan(&workspace.ID, &workspace.OrganizationID, &workspace.Name, &workspace.Slug, &workspace.Timezone, &workspace.Status, &workspace.CreatedAt, &workspace.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, workspace)
	}
	return result, rows.Err()
}

func (r *Repository) GetWorkspace(ctx context.Context, id uuid.UUID) (*domain.Workspace, error) {
	var workspace domain.Workspace
	err := r.db(ctx).QueryRow(ctx, `SELECT id, organization_id, name, slug, timezone, status, created_at, updated_at FROM workspace_workspaces WHERE id = $1`, id).Scan(&workspace.ID, &workspace.OrganizationID, &workspace.Name, &workspace.Slug, &workspace.Timezone, &workspace.Status, &workspace.CreatedAt, &workspace.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get workspace: %w", err)
	}
	return &workspace, nil
}

func (r *Repository) CreateWorkspace(ctx context.Context, workspace *domain.Workspace) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO workspace_workspaces (id, organization_id, name, slug, timezone, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, workspace.ID, workspace.OrganizationID, workspace.Name, workspace.Slug, workspace.Timezone, workspace.Status, workspace.CreatedAt, workspace.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicateSlug
	}
	if err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}
	return nil
}

func (r *Repository) UpdateWorkspace(ctx context.Context, workspace *domain.Workspace) error {
	result, err := r.db(ctx).Exec(ctx, `UPDATE workspace_workspaces SET name = $2, slug = $3, timezone = $4, status = $5, updated_at = $6 WHERE id = $1`, workspace.ID, workspace.Name, workspace.Slug, workspace.Timezone, workspace.Status, workspace.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicateSlug
	}
	if err != nil {
		return fmt.Errorf("update workspace: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) CreateMembership(ctx context.Context, membership *domain.Membership) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO workspace_memberships (id, workspace_id, user_id, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6)`, membership.ID, membership.WorkspaceID, membership.UserID, membership.Status, membership.CreatedAt, membership.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicateMembership
	}
	if err != nil {
		return fmt.Errorf("create membership: %w", err)
	}
	return nil
}

func (r *Repository) FindMembership(ctx context.Context, userID, workspaceID uuid.UUID) (*domain.Membership, error) {
	return r.findMembership(ctx, `WHERE m.user_id = $1 AND m.workspace_id = $2`, userID, workspaceID)
}

func (r *Repository) FindMembershipByID(ctx context.Context, id uuid.UUID) (*domain.Membership, error) {
	return r.findMembership(ctx, `WHERE m.id = $1`, id)
}

func (r *Repository) findMembership(ctx context.Context, condition string, args ...any) (*domain.Membership, error) {
	var membership domain.Membership
	err := r.db(ctx).QueryRow(ctx, `SELECT m.id, m.workspace_id, m.user_id, u.email, u.display_name, m.status, m.created_at, m.updated_at FROM workspace_memberships m JOIN users_users u ON u.id = m.user_id `+condition, args...).Scan(&membership.ID, &membership.WorkspaceID, &membership.UserID, &membership.Email, &membership.DisplayName, &membership.Status, &membership.CreatedAt, &membership.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrMembershipNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find membership: %w", err)
	}
	return &membership, nil
}

func (r *Repository) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT m.id, m.workspace_id, m.user_id, u.email, u.display_name, m.status, m.created_at, m.updated_at FROM workspace_memberships m JOIN users_users u ON u.id = m.user_id WHERE m.workspace_id = $1 ORDER BY u.display_name, u.email`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()
	result := []domain.Membership{}
	for rows.Next() {
		var membership domain.Membership
		if err := rows.Scan(&membership.ID, &membership.WorkspaceID, &membership.UserID, &membership.Email, &membership.DisplayName, &membership.Status, &membership.CreatedAt, &membership.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, membership)
	}
	return result, rows.Err()
}

func (r *Repository) UpdateMembership(ctx context.Context, membership *domain.Membership) error {
	result, err := r.db(ctx).Exec(ctx, `UPDATE workspace_memberships SET status = $2, updated_at = $3 WHERE id = $1`, membership.ID, membership.Status, membership.UpdatedAt)
	if err != nil {
		return fmt.Errorf("update membership: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrMembershipNotFound
	}
	return nil
}

func (r *Repository) DeleteMembership(ctx context.Context, id uuid.UUID) error {
	result, err := r.db(ctx).Exec(ctx, `DELETE FROM workspace_memberships WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete membership: %w", err)
	}
	if result.RowsAffected() == 0 {
		return domain.ErrMembershipNotFound
	}
	return nil
}

func (r *Repository) CountWorkspaces(ctx context.Context, organizationID uuid.UUID) (int, error) {
	var count int
	if err := r.db(ctx).QueryRow(ctx, `SELECT COUNT(*) FROM workspace_workspaces WHERE organization_id = $1`, organizationID).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

func (r *Repository) Touch(ctx context.Context, table string, id uuid.UUID, now time.Time) error {
	if table != "organization_organizations" && table != "workspace_workspaces" {
		return fmt.Errorf("invalid organization table")
	}
	_, err := r.db(ctx).Exec(ctx, `UPDATE `+table+` SET updated_at = $2 WHERE id = $1`, id, now)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
