package runtime

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/capability"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/extension"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

type Runtime struct {
	Modules      *module.Registry
	Entities     *entity.Registry
	Permissions  *permission.Registry
	Capabilities *capability.Registry
	Events       *event.Bus
	Extensions   *extension.Registry
	Metadata     *metadata.Registry
}

func New() *Runtime {
	modules := module.NewRegistry()
	entities := entity.NewRegistry()
	permissions := permission.NewRegistry()
	capabilities := capability.NewRegistry()
	events := event.NewRegistry()
	extensions := extension.NewRegistry()
	return &Runtime{Modules: modules, Entities: entities, Permissions: permissions, Capabilities: capabilities, Events: event.NewBus(events), Extensions: extensions, Metadata: metadata.NewRegistry(modules, entities, permissions, capabilities, events, extensions)}
}

func (r *Runtime) Initialize(ctx context.Context) error {
	_, err := r.Modules.Initialize(ctx, &module.Context{Entities: r.Entities, Permissions: r.Permissions, Capabilities: r.Capabilities, Events: r.Events, Extensions: r.Extensions})
	if err != nil {
		return err
	}
	return r.Validate()
}

func (r *Runtime) Validate() error {
	if err := r.Entities.ValidateCapabilities(r.Capabilities.Contains); err != nil {
		return err
	}
	if err := r.Entities.ValidateRelations(func(name string) bool { _, ok := r.Entities.Get(name); return ok }); err != nil {
		return err
	}
	return nil
}

func (r *Runtime) LogStartup(logger *slog.Logger) {
	snapshot := r.Metadata.Snapshot()
	logger.Info("framework runtime created", "modules", len(snapshot.Modules), "entities", len(snapshot.Entities), "permissions", len(snapshot.Permissions), "events", len(snapshot.Events), "extensions", len(snapshot.Extensions))
	logger.Info("module dependency resolution completed", "modules", snapshot.Modules)
	logger.Info("framework runtime validated")
}

func RequireEntity(r *Runtime, name string) (entity.Definition, error) {
	definition, ok := r.Entities.Get(name)
	if !ok {
		return entity.Definition{}, fmt.Errorf("unknown entity %q", name)
	}
	return definition, nil
}
