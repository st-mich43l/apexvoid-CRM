package errors

import "fmt"

type Kind string

const (
	Validation Kind = "validation"
	NotFound   Kind = "not_found"
	Conflict   Kind = "conflict"
	Forbidden  Kind = "forbidden"
	Internal   Kind = "internal"
)

type Error struct {
	Kind    Kind
	Message string
	Cause   error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Cause }
func New(kind Kind, message string, cause error) *Error {
	return &Error{Kind: kind, Message: message, Cause: cause}
}
func ValidationError(message string) error { return New(Validation, message, nil) }
func NotFoundError(resource string) error {
	return New(NotFound, fmt.Sprintf("%s was not found", resource), nil)
}
func ConflictError(message string) error              { return New(Conflict, message, nil) }
func ForbiddenError(message string) error             { return New(Forbidden, message, nil) }
func InternalError(message string, cause error) error { return New(Internal, message, cause) }
