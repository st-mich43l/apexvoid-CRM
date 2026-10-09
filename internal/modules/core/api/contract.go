package api

import (
	"context"

	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
)

// MetadataReader is the intentionally small public contract exposed by the
// core module. Other modules can depend on this contract without importing
// core's application or transport implementation.
type MetadataReader interface {
	Snapshot() metadata.Snapshot
	Entity(name string) (entity.Definition, bool)
}

type FrameworkReader interface {
	Applications(context.Context) []frameworkapplication.Descriptor
	Modules(context.Context) []metadata.ModuleMetadata
	Entities(context.Context) []entity.Definition
	Permissions(context.Context) []metadata.PermissionMetadata
	Entity(context.Context, string) (entity.Definition, error)
}
