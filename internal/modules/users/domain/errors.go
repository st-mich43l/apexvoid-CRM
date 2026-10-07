package domain

import "errors"

var (
	ErrNotFound           = errors.New("user not found")
	ErrDuplicateEmail     = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserDisabled       = errors.New("user is not active")
	ErrSessionInvalid     = errors.New("session is invalid")
	ErrPasswordPolicy     = errors.New("password does not meet policy")
	ErrCurrentPassword    = errors.New("current password is incorrect")
)
