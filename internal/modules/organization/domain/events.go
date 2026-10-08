package domain

import "github.com/google/uuid"

type OrganizationCreated struct{ OrganizationID uuid.UUID }
type OrganizationUpdated struct{ OrganizationID uuid.UUID }
type WorkspaceCreated struct {
	WorkspaceID    uuid.UUID
	OrganizationID uuid.UUID
}
type WorkspaceUpdated struct {
	WorkspaceID    uuid.UUID
	OrganizationID uuid.UUID
}
type MemberAdded struct {
	WorkspaceID  uuid.UUID
	MembershipID uuid.UUID
	UserID       uuid.UUID
}
type MemberUpdated struct {
	WorkspaceID  uuid.UUID
	MembershipID uuid.UUID
	UserID       uuid.UUID
}
type MemberRemoved struct {
	WorkspaceID  uuid.UUID
	MembershipID uuid.UUID
	UserID       uuid.UUID
}
