package core

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core/application"
	coremetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/metadata"
	corehttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/transport/http"
)

type Dependencies struct{ Metadata *metadata.Registry }
type Module struct{ service *application.Service }

func New(dependencies Dependencies) *Module {
	return &Module{service: application.NewService(dependencies.Metadata)}
}

func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "core", DisplayName: "ApexVoid Core", Version: "1.0.0"}
}

func (m *Module) Register(ctx *module.Context) error { return coremetadata.Register(ctx) }

func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return corehttp.NewHandler(m.service).RegisterRoutes(routes)
}

func (Module) Migrations() []module.Migration { return nil }
