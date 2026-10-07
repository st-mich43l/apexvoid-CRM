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
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/access/domain"
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

func (r *Repository) CreateRole(ctx context.Context, role *domain.Role) error {
	_, err := r.db(ctx).Exec(ctx, `INSERT INTO access_roles (id, workspace_id, name, display_name, description, system, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, role.ID, role.WorkspaceID, role.Name, role.DisplayName, role.Description, role.System, role.CreatedAt, role.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicateRole
	}
	if err != nil {
		return fmt.Errorf("create role: %w", err)
	}
	return nil
}

func (r *Repository) FindRole(ctx context.Context, id uuid.UUID) (*domain.Role, error) {
	return r.findRole(ctx, `WHERE id = $1`, id)
}

func (r *Repository) FindRoleByName(ctx context.Context, name string) (*domain.Role, error) {
	return r.findRole(ctx, `WHERE name = $1`, name)
}

func (r *Repository) FindRoleByWorkspaceAndName(ctx context.Context, workspaceID *uuid.UUID, name string) (*domain.Role, error) {
	if workspaceID == nil {
		return r.findRole(ctx, `WHERE workspace_id IS NULL AND name = $1`, name)
	}
	return r.findRole(ctx, `WHERE workspace_id = $1 AND name = $2`, *workspaceID, name)
}

func (r *Repository) findRole(ctx context.Context, condition string, args ...any) (*domain.Role, error) {
	var role domain.Role
	if err := r.db(ctx).QueryRow(ctx, `SELECT id, workspace_id, name, display_name, description, system, created_at, updated_at FROM access_roles `+condition, args...).Scan(&role.ID, &role.WorkspaceID, &role.Name, &role.DisplayName, &role.Description, &role.System, &role.CreatedAt, &role.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("find role: %w", err)
	}
	return &role, nil
}

func (r *Repository) ListRoles(ctx context.Context) ([]domain.Role, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT id, workspace_id, name, display_name, description, system, created_at, updated_at FROM access_roles ORDER BY system DESC, name`)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	defer rows.Close()
	roles := []domain.Role{}
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.WorkspaceID, &role.Name, &role.DisplayName, &role.Description, &role.System, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *Repository) ListWorkspaceRoles(ctx context.Context, workspaceID uuid.UUID) ([]domain.Role, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT id, workspace_id, name, display_name, description, system, created_at, updated_at FROM access_roles WHERE workspace_id = $1 ORDER BY name`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace roles: %w", err)
	}
	defer rows.Close()
	roles := []domain.Role{}
	for rows.Next() {
		var role domain.Role
		if err := rows.Scan(&role.ID, &role.WorkspaceID, &role.Name, &role.DisplayName, &role.Description, &role.System, &role.CreatedAt, &role.UpdatedAt); err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, rows.Err()
}

func (r *Repository) UpdateRole(ctx context.Context, role *domain.Role) error {
	result, err := r.db(ctx).Exec(ctx, `UPDATE access_roles SET display_name = $2, description = $3, updated_at = $4 WHERE id = $1`, role.ID, role.DisplayName, role.Description, role.UpdatedAt)
	if isUniqueViolation(err) {
		return domain.ErrDuplicateRole
	}
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteRole(ctx context.Context, id uuid.UUID) error {
	result, err := r.db(ctx).Exec(ctx, `DELETE FROM access_roles WHERE id = $1 AND system = FALSE`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return domain.ErrSystemRole
	}
	return nil
}

func (r *Repository) ReplaceRolePermissions(ctx context.Context, roleID uuid.UUID, permissions []string) error {
	db := r.db(ctx)
	if _, err := db.Exec(ctx, `DELETE FROM access_role_permissions WHERE role_id = $1`, roleID); err != nil {
		return err
	}
	for _, permission := range permissions {
		if _, err := db.Exec(ctx, `INSERT INTO access_role_permissions (role_id, permission_name) VALUES ($1,$2)`, roleID, permission); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) RolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT permission_name FROM access_role_permissions WHERE role_id = $1 ORDER BY permission_name`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var permission string
		if err := rows.Scan(&permission); err != nil {
			return nil, err
		}
		result = append(result, permission)
	}
	return result, rows.Err()
}

func (r *Repository) ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	db := r.db(ctx)
	if _, err := db.Exec(ctx, `DELETE FROM access_user_roles WHERE user_id = $1`, userID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if _, err := db.Exec(ctx, `INSERT INTO access_user_roles (user_id, role_id) VALUES ($1,$2)`, userID, roleID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) UserRoleIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT role_id FROM access_user_roles WHERE user_id = $1 ORDER BY role_id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (r *Repository) EffectivePermissions(ctx context.Context, userID uuid.UUID) ([]string, bool, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT r.name, rp.permission_name FROM access_user_roles ur JOIN access_roles r ON r.id = ur.role_id LEFT JOIN access_role_permissions rp ON rp.role_id = r.id WHERE ur.user_id = $1 ORDER BY r.name, rp.permission_name`, userID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	set := map[string]struct{}{}
	admin := false
	for rows.Next() {
		var roleName string
		var permission *string
		if err := rows.Scan(&roleName, &permission); err != nil {
			return nil, false, err
		}
		if roleName == "administrator" {
			admin = true
		}
		if permission != nil {
			set[*permission] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for permission := range set {
		result = append(result, permission)
	}
	return result, admin, rows.Err()
}

func (r *Repository) EffectivePermissionsForWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) ([]string, bool, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT r.name, rp.permission_name FROM access_user_roles ur JOIN access_roles r ON r.id = ur.role_id LEFT JOIN access_role_permissions rp ON rp.role_id = r.id WHERE ur.user_id = $1
UNION ALL
SELECT r.name, rp.permission_name FROM workspace_memberships m JOIN access_workspace_membership_roles wmr ON wmr.membership_id = m.id JOIN access_roles r ON r.id = wmr.role_id LEFT JOIN access_role_permissions rp ON rp.role_id = r.id WHERE m.user_id = $1 AND m.workspace_id = $2 AND m.status = 'active' AND r.workspace_id = $2`, userID, workspaceID)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	set := map[string]struct{}{}
	platformAdmin := false
	for rows.Next() {
		var roleName string
		var permission *string
		if err := rows.Scan(&roleName, &permission); err != nil {
			return nil, false, err
		}
		if roleName == "administrator" {
			platformAdmin = true
		}
		if permission != nil {
			set[*permission] = struct{}{}
		}
	}
	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	return result, platformAdmin, rows.Err()
}

func (r *Repository) ReplaceMembershipRoles(ctx context.Context, membershipID uuid.UUID, roleIDs []uuid.UUID) error {
	db := r.db(ctx)
	if _, err := db.Exec(ctx, `DELETE FROM access_workspace_membership_roles WHERE membership_id = $1`, membershipID); err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if _, err := db.Exec(ctx, `INSERT INTO access_workspace_membership_roles (membership_id, role_id) VALUES ($1,$2)`, membershipID, roleID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) MembershipRoleIDs(ctx context.Context, membershipID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db(ctx).Query(ctx, `SELECT role_id FROM access_workspace_membership_roles WHERE membership_id = $1 ORDER BY role_id`, membershipID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		result = append(result, id)
	}
	return result, rows.Err()
}

func (r *Repository) EnsureAdministrator(ctx context.Context) (domain.Role, error) {
	role, err := r.FindRoleByName(ctx, "administrator")
	if err == nil {
		return *role, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return domain.Role{}, err
	}
	now := time.Now().UTC()
	role = &domain.Role{ID: uuid.New(), Name: "administrator", DisplayName: "Administrator", Description: "Grants all registered permissions.", System: true, CreatedAt: now, UpdatedAt: now}
	if err := r.CreateRole(ctx, role); err != nil {
		if errors.Is(err, domain.ErrDuplicateRole) {
			role, err = r.FindRoleByName(ctx, "administrator")
			if err == nil {
				return *role, nil
			}
		}
		return domain.Role{}, err
	}
	return *role, nil
}

func (r *Repository) TouchRole(ctx context.Context, id uuid.UUID, now time.Time) error {
	_, err := r.db(ctx).Exec(ctx, `UPDATE access_roles SET updated_at = $2 WHERE id = $1`, id, now)
	return err
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
