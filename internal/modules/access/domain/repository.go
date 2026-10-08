package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreateRole(ctx context.Context, role *Role) error
	FindRole(ctx context.Context, id uuid.UUID) (*Role, error)
	FindRoleByName(ctx context.Context, name string) (*Role, error)
	ListRoles(ctx context.Context) ([]Role, error)
	ListWorkspaceRoles(ctx context.Context, workspaceID uuid.UUID) ([]Role, error)
	UpdateRole(ctx context.Context, role *Role) error
	DeleteRole(ctx context.Context, id uuid.UUID) error
	ReplaceRolePermissions(ctx context.Context, roleID uuid.UUID, permissions []string) error
	RolePermissions(ctx context.Context, roleID uuid.UUID) ([]string, error)
	ReplaceUserRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
	UserRoleIDs(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error)
	EffectivePermissions(ctx context.Context, userID uuid.UUID) ([]string, bool, error)
	EnsureAdministrator(ctx context.Context) (Role, error)
	FindRoleByWorkspaceAndName(ctx context.Context, workspaceID *uuid.UUID, name string) (*Role, error)
	ReplaceMembershipRoles(ctx context.Context, membershipID uuid.UUID, roleIDs []uuid.UUID) error
	MembershipRoleIDs(ctx context.Context, membershipID uuid.UUID) ([]uuid.UUID, error)
	EffectivePermissionsForWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) ([]string, []string, bool, bool, error)
	CountActiveWorkspaceAdministrators(ctx context.Context, workspaceID uuid.UUID) (int, error)
	IsWorkspaceAdministrator(ctx context.Context, membershipID, workspaceID uuid.UUID) (bool, error)
	TouchRole(ctx context.Context, id uuid.UUID, now time.Time) error
}
