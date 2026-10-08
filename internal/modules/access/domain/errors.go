package domain

import "errors"

var (
	ErrNotFound                   = errors.New("role not found")
	ErrDuplicateRole              = errors.New("role already exists")
	ErrReservedRoleName           = errors.New("reserved administrator role name")
	ErrUnknownPermission          = errors.New("unknown permission")
	ErrPermissionScope            = errors.New("permission scope does not match role scope")
	ErrSystemRole                 = errors.New("system role cannot be deleted or renamed")
	ErrAdministratorRole          = errors.New("administrator role cannot be removed")
	ErrAssignmentNotAllowed       = errors.New("assignment is not allowed")
	ErrWorkspaceRole              = errors.New("role does not belong to this workspace")
	ErrLastWorkspaceAdministrator = errors.New("a workspace must have at least one active administrator")
	ErrLastPlatformAdministrator  = errors.New("a platform must have at least one active administrator")
)
