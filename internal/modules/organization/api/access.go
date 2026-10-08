package api

import (
	"context"

	"github.com/google/uuid"
)

type WorkspaceRole struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID *uuid.UUID `json:"workspace_id,omitempty"`
	Name        string     `json:"name"`
	DisplayName string     `json:"display_name"`
	Description string     `json:"description"`
	System      bool       `json:"system"`
	Permissions []string   `json:"permissions"`
}

type WorkspaceAccess interface {
	WorkspaceProvisioner
	Can(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
	CanInWorkspace(ctx context.Context, userID, workspaceID uuid.UUID, permission string) (bool, error)
	EffectivePermissionsForWorkspace(ctx context.Context, userID, workspaceID uuid.UUID) ([]string, error)
	ListWorkspaceRoles(ctx context.Context, workspaceID uuid.UUID) ([]WorkspaceRole, error)
	CreateWorkspaceRole(ctx context.Context, workspaceID uuid.UUID, name, displayName, description string) (WorkspaceRole, error)
	ReplaceWorkspaceRolePermissions(ctx context.Context, roleID, workspaceID uuid.UUID, permissions []string) error
	MembershipRoleIDs(ctx context.Context, membershipID uuid.UUID) ([]uuid.UUID, error)
	ReplaceMembershipRoles(ctx context.Context, membershipID, workspaceID uuid.UUID, roleIDs []uuid.UUID) error
	CountActiveWorkspaceAdministrators(ctx context.Context, workspaceID uuid.UUID) (int, error)
	IsWorkspaceAdministrator(ctx context.Context, membershipID, workspaceID uuid.UUID) (bool, error)
	LockWorkspaceAdministratorState(ctx context.Context, workspaceID uuid.UUID) error
}

type WorkspaceProvisioner interface {
	EnsureWorkspaceAdministrator(ctx context.Context, workspaceID uuid.UUID) error
	AssignWorkspaceAdministrator(ctx context.Context, membershipID, workspaceID uuid.UUID) error
}
