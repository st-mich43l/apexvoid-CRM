package domain

import (
	"context"
	"github.com/google/uuid"
	"io"
)

type Repository interface {
	CreateContact(context.Context, *Contact) error
	GetContact(context.Context, uuid.UUID, uuid.UUID) (*Contact, error)
	ListContacts(context.Context, uuid.UUID, ListFilter) ([]Contact, int, error)
	UpdateContact(context.Context, *Contact) error
	SetContactStatus(context.Context, uuid.UUID, uuid.UUID, ContactStatus, uuid.UUID) error
	ListRelationships(context.Context, uuid.UUID, uuid.UUID) ([]Relationship, error)
	ReplaceRelationships(context.Context, uuid.UUID, uuid.UUID, []Relationship) error
	ListTags(context.Context, uuid.UUID, bool) ([]Tag, error)
	CreateTag(context.Context, *Tag) error
	UpdateTag(context.Context, *Tag) error
	ReplaceContactTags(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID) error
	ContactTagIDs(context.Context, uuid.UUID, uuid.UUID) ([]uuid.UUID, error)
	ListNotes(context.Context, uuid.UUID, uuid.UUID) ([]Note, error)
	CreateNote(context.Context, *Note) error
	UpdateNote(context.Context, *Note) error
	ListActivities(context.Context, uuid.UUID, *uuid.UUID, *uuid.UUID, ActivityStatus) ([]Activity, error)
	CreateActivity(context.Context, *Activity) error
	UpdateActivity(context.Context, *Activity) error
	ListAttachments(context.Context, uuid.UUID, uuid.UUID) ([]Attachment, error)
	CreateAttachment(context.Context, *Attachment, string) error
	GetAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Attachment, string, error)
	DeleteAttachment(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error
	ListCustomFields(context.Context, uuid.UUID, bool) ([]CustomFieldDefinition, error)
	CreateCustomField(context.Context, *CustomFieldDefinition) error
	UpdateCustomField(context.Context, *CustomFieldDefinition) error
	IsActiveWorkspaceMember(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

type FileStore interface {
	Save(context.Context, string, io.Reader, int64) (string, int64, error)
	Open(context.Context, string) (io.ReadCloser, error)
	Delete(context.Context, string) error
}
type ContactReader interface {
	GetByID(context.Context, uuid.UUID, uuid.UUID) (ContactSummary, error)
}
type ContactSummary struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Kind        ContactKind
	DisplayName string
	Email       string
}
