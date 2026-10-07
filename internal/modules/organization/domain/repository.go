package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CountOrganizations(ctx context.Context) (int, error)
	GetOrganization(ctx context.Context, id uuid.UUID) (*Organization, error)
	GetOrganizationForWorkspace(ctx context.Context, workspaceID uuid.UUID) (*Organization, error)
	CreateOrganization(ctx context.Context, organization *Organization) error
	UpdateOrganization(ctx context.Context, organization *Organization) error
	ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]Workspace, error)
	GetWorkspace(ctx context.Context, id uuid.UUID) (*Workspace, error)
	CreateWorkspace(ctx context.Context, workspace *Workspace) error
	UpdateWorkspace(ctx context.Context, workspace *Workspace) error
	CreateMembership(ctx context.Context, membership *Membership) error
	FindMembership(ctx context.Context, userID, workspaceID uuid.UUID) (*Membership, error)
	FindMembershipByID(ctx context.Context, id uuid.UUID) (*Membership, error)
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]Membership, error)
	UpdateMembership(ctx context.Context, membership *Membership) error
	DeleteMembership(ctx context.Context, id uuid.UUID) error
	CountWorkspaces(ctx context.Context, organizationID uuid.UUID) (int, error)
	Touch(ctx context.Context, table string, id uuid.UUID, now time.Time) error
}
