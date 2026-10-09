package metadata

import (
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/capability"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/extension"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

type Snapshot struct {
	Applications []frameworkapplication.Descriptor
	Modules      []module.Descriptor
	Entities     []entity.Definition
	Permissions  []permission.Definition
	Capabilities []capability.Definition
	Events       []event.Definition
	Extensions   []extension.Point
}

type Registry struct {
	applications *frameworkapplication.Registry
	modules      *module.Registry
	entities     *entity.Registry
	permissions  *permission.Registry
	capabilities *capability.Registry
	events       *event.Registry
	extensions   *extension.Registry
}

func NewRegistry(applications *frameworkapplication.Registry, modules *module.Registry, entities *entity.Registry, permissions *permission.Registry, capabilities *capability.Registry, events *event.Registry, extensions *extension.Registry) *Registry {
	return &Registry{applications: applications, modules: modules, entities: entities, permissions: permissions, capabilities: capabilities, events: events, extensions: extensions}
}

func (r *Registry) Snapshot() Snapshot {
	return Snapshot{Applications: r.applications.List(), Modules: r.modules.Descriptors(), Entities: r.entities.List(), Permissions: r.permissions.List(), Capabilities: r.capabilities.List(), Events: r.events.List(), Extensions: r.extensions.Points()}
}

func (r *Registry) Entity(name string) (entity.Definition, bool) { return r.entities.Get(name) }
