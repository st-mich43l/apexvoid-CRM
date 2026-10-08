package domain

import "github.com/google/uuid"

type ContactCreated struct{ WorkspaceID, ContactID, CreatedBy uuid.UUID }
type ContactUpdated struct{ WorkspaceID, ContactID, UpdatedBy uuid.UUID }
type ContactArchived struct{ WorkspaceID, ContactID, UpdatedBy uuid.UUID }
type RelationshipChanged struct{ WorkspaceID, ContactID uuid.UUID }
type ActivityCreated struct{ WorkspaceID, ActivityID, ContactID uuid.UUID }
type ActivityCompletedEvent struct{ WorkspaceID, ActivityID uuid.UUID }
