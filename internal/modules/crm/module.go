package crm

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/application"
	crmpostgres "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/infrastructure/postgres"
	crmmetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/metadata"
	crmmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/migrations"
	crmhttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/transport/http"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Module struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	workspace     organizationapi.WorkspaceResolver
	access        organizationapi.WorkspaceAccess
}

func New(pool *pgxpool.Pool, tx *database.TxManager, authenticator usersapi.Authenticator, workspace organizationapi.WorkspaceResolver, access organizationapi.WorkspaceAccess) *Module {
	return &Module{service: application.New(crmpostgres.New(pool), tx), authenticator: authenticator, workspace: workspace, access: access}
}
func (m *Module) Service() *application.Service { return m.service }
func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "crm", DisplayName: "CRM", Version: "1.0.0", Dependencies: []string{"customization", "contacts"}}
}
func (Module) Register(ctx *module.Context) error { return crmmetadata.Register(ctx) }
func (Module) Migrations() []module.Migration     { return crmmigrations.All() }
func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return crmhttp.New(m.service, m.authenticator, m.workspace, m.access).RegisterRoutes(routes)
}
