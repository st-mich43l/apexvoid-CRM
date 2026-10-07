package organization

import (
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/application"
	organizationpostgres "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/infrastructure/postgres"
	organizationmetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/metadata"
	organizationmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/migrations"
	organizationhttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/transport/http"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Dependencies struct {
	Pool          *pgxpool.Pool
	Transactions  *database.TxManager
	Authenticator usersapi.Authenticator
	Users         usersapi.UserReader
	Directory     usersapi.UserDirectory
}

type Module struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	access        organizationapi.WorkspaceAccess
}

func New(dependencies Dependencies) *Module {
	return &Module{service: application.NewService(application.Dependencies{Repository: organizationpostgres.NewRepository(dependencies.Pool), Transactions: dependencies.Transactions, Users: dependencies.Users, Directory: dependencies.Directory}), authenticator: dependencies.Authenticator}
}

func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "organization", DisplayName: "Organization & Workspace", Version: "1.0.0", Dependencies: []string{"users"}}
}

func (m *Module) Service() *application.Service { return m.service }
func (m *Module) SetAccess(access organizationapi.WorkspaceAccess) {
	m.access = access
	m.service.SetAccess(access)
}
func (m *Module) Register(ctx *module.Context) error { return organizationmetadata.Register(ctx) }
func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return organizationhttp.NewHandler(m.service, m.authenticator, m.access).RegisterRoutes(routes)
}
func (Module) Migrations() []module.Migration { return organizationmigrations.All() }

var _ organizationapi.WorkspaceResolver = (*application.Service)(nil)
