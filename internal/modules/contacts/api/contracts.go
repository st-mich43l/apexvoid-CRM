package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/domain"
)

type ContactReader interface {
	GetByID(context.Context, uuid.UUID, uuid.UUID) (domain.ContactSummary, error)
}

// CustomFieldReader exposes the established Contacts field store to shared
// configuration consumers without creating a second source of truth.
type CustomFieldReader interface {
	ListCustomFields(context.Context, uuid.UUID, bool) ([]domain.CustomFieldDefinition, error)
}
