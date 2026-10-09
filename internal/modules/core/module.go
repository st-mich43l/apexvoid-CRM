package core

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	coreapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core/application"
	coremetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/metadata"
	corehttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/transport/http"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
)

type Dependencies struct {
	Metadata      *metadata.Registry
	Authenticator usersapi.Authenticator
	Workspace     organizationapi.WorkspaceResolver
	Access        coreapi.ApplicationAuthorizer
}
type Module struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	workspace     organizationapi.WorkspaceResolver
}

func New(dependencies Dependencies) *Module {
	return &Module{service: application.NewService(dependencies.Metadata, dependencies.Access), authenticator: dependencies.Authenticator, workspace: dependencies.Workspace}
}

func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "core", DisplayName: "ApexVoid Core", Version: "1.0.0"}
}

func (m *Module) Register(ctx *module.Context) error { return coremetadata.Register(ctx) }

func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return corehttp.NewHandler(m.service, m.authenticator, m.workspace).RegisterRoutes(routes)
}

func (Module) Migrations() []module.Migration { return nil }
