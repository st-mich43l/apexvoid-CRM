package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/domain"
)

type ContactReader interface {
	GetByID(context.Context, uuid.UUID, uuid.UUID) (domain.ContactSummary, error)
}

// ContactCreator is intentionally narrow: CRM conversion can create an explicit
// contact inside the caller's transaction, without reaching into Contacts storage.
type ContactCreator interface {
	CreateContact(context.Context, domain.Contact) (domain.Contact, error)
}

// CustomFieldReader exposes the established Contacts field store to shared
// configuration consumers without creating a second source of truth.
type CustomFieldReader interface {
	ListCustomFields(context.Context, uuid.UUID, bool) ([]domain.CustomFieldDefinition, error)
}
