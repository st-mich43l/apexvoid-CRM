package domain

import "github.com/google/uuid"

// Event is the typed payload for CRM lifecycle notifications. Detailed history is
// persisted on the business record; this lightweight event is emitted post-commit.
type Event struct {
	WorkspaceID uuid.UUID
	ResourceID  uuid.UUID
}
