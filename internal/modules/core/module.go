package core

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	coreapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core/application"
	coremetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/metadata"
	coremigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/migrations"
	corehttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/transport/http"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
)

type Dependencies struct {
	Metadata      *metadata.Registry
	Authenticator usersapi.Authenticator
	Workspace     organizationapi.WorkspaceResolver
	Access        coreapi.ApplicationAuthorizer
	Pool          *pgxpool.Pool
	Permissions   *permission.Registry
}
type Module struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	workspace     organizationapi.WorkspaceResolver
	external      *application.ExternalStore
	access        coreapi.ApplicationAuthorizer
}

func New(dependencies Dependencies) *Module {
	var external *application.ExternalStore
	if dependencies.Pool != nil && dependencies.Permissions != nil {
		external = application.NewExternalStore(dependencies.Pool, dependencies.Permissions)
	}
	return &Module{service: application.NewService(dependencies.Metadata, dependencies.Access, external), authenticator: dependencies.Authenticator, workspace: dependencies.Workspace, external: external, access: dependencies.Access}
}

func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "core", DisplayName: "ApexVoid Core", Version: "1.0.0"}
}

func (m *Module) Register(ctx *module.Context) error { return coremetadata.Register(ctx) }

func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return corehttp.NewHandler(m.service, m.authenticator, m.workspace, m.access).RegisterRoutes(routes)
}

func (Module) Migrations() []module.Migration { return coremigrations.All() }

// HydrateExternalPermissions restores the persistent external catalog after
// migrations have run. Existing role grants are preserved because permission
// names remain stable in the access database.
func (m *Module) HydrateExternalPermissions(ctx context.Context) error {
	if m.external == nil {
		return nil
	}
	if err := m.external.Hydrate(ctx); err != nil {
		return fmt.Errorf("hydrate external applications: %w", err)
	}
	return nil
}
