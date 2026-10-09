package runtime

import (
	"context"
	"fmt"
	"log/slog"

	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/capability"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/extension"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

type Runtime struct {
	Applications *frameworkapplication.Registry
	Modules      *module.Registry
	Entities     *entity.Registry
	Permissions  *permission.Registry
	Capabilities *capability.Registry
	Events       *event.Bus
	Extensions   *extension.Registry
	Metadata     *metadata.Registry
}

func New() *Runtime {
	applications := frameworkapplication.NewRegistry()
	modules := module.NewRegistry()
	entities := entity.NewRegistry()
	permissions := permission.NewRegistry()
	capabilities := capability.NewRegistry()
	events := event.NewRegistry()
	extensions := extension.NewRegistry()
	return &Runtime{Applications: applications, Modules: modules, Entities: entities, Permissions: permissions, Capabilities: capabilities, Events: event.NewBus(events), Extensions: extensions, Metadata: metadata.NewRegistry(applications, modules, entities, permissions, capabilities, events, extensions)}
}

func (r *Runtime) Initialize(ctx context.Context) error {
	_, err := r.Modules.Initialize(ctx, &module.Context{Applications: r.Applications, Entities: r.Entities, Permissions: r.Permissions, Capabilities: r.Capabilities, Events: r.Events, Extensions: r.Extensions})
	if err != nil {
		return err
	}
	return r.Validate()
}

func (r *Runtime) Validate() error {
	modules := make(map[string]struct{})
	for _, descriptor := range r.Modules.Descriptors() {
		modules[descriptor.Name] = struct{}{}
	}
	for _, definition := range r.Entities.List() {
		if _, ok := modules[definition.Module]; !ok {
			return fmt.Errorf("entity %q references unregistered module %q", definition.Name, definition.Module)
		}
	}
	for _, definition := range r.Permissions.List() {
		if _, ok := modules[definition.Module]; !ok {
			return fmt.Errorf("permission %q references unregistered module %q", definition.Name, definition.Module)
		}
	}
	for _, definition := range r.Metadata.Snapshot().Events {
		if _, ok := modules[definition.Module]; !ok {
			return fmt.Errorf("event %q references unregistered module %q", definition.Name, definition.Module)
		}
	}
	for _, definition := range r.Applications.List() {
		for _, dependency := range definition.ModuleDependencies {
			if _, ok := modules[dependency]; !ok {
				return fmt.Errorf("application %q requires unregistered module %q", definition.ID, dependency)
			}
		}
		for _, required := range definition.RequiredPermissions {
			if !r.Permissions.Contains(required) {
				return fmt.Errorf("application %q requires unregistered permission %q", definition.ID, required)
			}
		}
		for _, required := range definition.RequiredCapabilities {
			if !r.Capabilities.Contains(required) {
				return fmt.Errorf("application %q requires unregistered capability %q", definition.ID, required)
			}
		}
	}
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
	logger.Info("framework runtime created", "modules", len(snapshot.Modules), "applications", len(snapshot.Applications), "entities", len(snapshot.Entities), "permissions", len(snapshot.Permissions), "events", len(snapshot.Events), "extensions", len(snapshot.Extensions))
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
