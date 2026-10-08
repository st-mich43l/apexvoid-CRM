package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/domain"
)

type ContactReader interface {
	GetByID(context.Context, uuid.UUID, uuid.UUID) (domain.ContactSummary, error)
}
