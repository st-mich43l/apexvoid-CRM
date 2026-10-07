package api

import (
	"context"
	"github.com/google/uuid"
)

type Authorizer interface {
	Can(ctx context.Context, userID uuid.UUID, permission string) (bool, error)
}
