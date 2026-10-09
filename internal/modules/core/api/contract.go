package api

import (
	"context"

	"github.com/google/uuid"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

// MetadataReader is the intentionally small public contract exposed by the
// core module. Other modules can depend on this contract without importing
// core's application or transport implementation.
type MetadataReader interface {
	Snapshot() metadata.Snapshot
	Entity(name string) (entity.Definition, bool)
	SnapshotPermission(name string) (permission.Definition, bool)
}

// SnapshotPermission is kept separate so the external integration surface can
// make a single, scope-aware authorization decision without exposing roles.
type PermissionMetadataReader interface {
	SnapshotPermission(string) (permission.Definition, bool)
}

type FrameworkReader interface {
	Applications(context.Context, uuid.UUID, uuid.UUID) ([]DiscoveredApplication, error)
	Modules(context.Context) []metadata.ModuleMetadata
	Entities(context.Context) []entity.Definition
	Permissions(context.Context) []metadata.PermissionMetadata
	Entity(context.Context, string) (entity.Definition, error)
}

// ApplicationAuthorizer combines existing platform and workspace authorization
// contracts. The core module never reads role state directly.
type ApplicationAuthorizer interface {
	Can(context.Context, uuid.UUID, string) (bool, error)
	CanInWorkspace(context.Context, uuid.UUID, uuid.UUID, string) (bool, error)
}

type DiscoveredApplication struct {
	Descriptor         frameworkapplication.Descriptor
	EntryAuthorized    bool
	SettingsAuthorized bool
}
