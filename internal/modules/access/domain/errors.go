package domain

import "errors"

var (
	ErrNotFound             = errors.New("role not found")
	ErrDuplicateRole        = errors.New("role already exists")
	ErrUnknownPermission    = errors.New("unknown permission")
	ErrSystemRole           = errors.New("system role cannot be deleted or renamed")
	ErrAdministratorRole    = errors.New("administrator role cannot be removed")
	ErrAssignmentNotAllowed = errors.New("assignment is not allowed")
)
