package access

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/access/application"
	accesspostgres "github.com/st-mich43l/apexvoid-CRM/internal/modules/access/infrastructure/postgres"
	accessmetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/access/metadata"
	accessmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/access/migrations"
	accesshttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/access/transport/http"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Dependencies struct {
	Pool          *pgxpool.Pool
	Transactions  *database.TxManager
	Permissions   *permission.Registry
	Users         usersapi.UserReader
	Authenticator usersapi.Authenticator
}

type Module struct {
	service       *application.Service
	authenticator usersapi.Authenticator
}

func New(dependencies Dependencies) *Module {
	repository := accesspostgres.NewRepository(dependencies.Pool)
	return &Module{service: application.NewService(application.Dependencies{Repository: repository, Permissions: dependencies.Permissions, Users: dependencies.Users, Transactions: dependencies.Transactions}), authenticator: dependencies.Authenticator}
}

func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "access", DisplayName: "Access Control", Version: "1.0.0", Dependencies: []string{"users", "organization"}}
}
func (m *Module) Service() *application.Service      { return m.service }
func (m *Module) Register(ctx *module.Context) error { return accessmetadata.Register(ctx) }
func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return accesshttp.NewHandler(m.service, m.authenticator).RegisterRoutes(routes)
}
func (Module) Migrations() []module.Migration { return accessmigrations.All() }
