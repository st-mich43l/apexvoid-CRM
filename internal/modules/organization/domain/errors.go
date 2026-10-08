package domain

import "errors"

var (
	ErrNotFound                   = errors.New("organization resource not found")
	ErrSetupComplete              = errors.New("initial organization setup is already complete")
	ErrDuplicateSlug              = errors.New("organization or workspace slug already exists")
	ErrDuplicateMembership        = errors.New("user is already a member of this workspace")
	ErrMembershipNotFound         = errors.New("workspace membership not found")
	ErrMembershipSuspended        = errors.New("workspace membership is suspended")
	ErrWorkspaceInactive          = errors.New("workspace is inactive")
	ErrOrganizationInactive       = errors.New("organization is inactive")
	ErrInvalidStatus              = errors.New("invalid organization status")
	ErrLastWorkspaceAdministrator = errors.New("a workspace must have at least one active administrator")
)
