package users

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/users/application"
	userspostgres "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/infrastructure/postgres"
	usersmetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/metadata"
	usersmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/migrations"
	usershttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/transport/http"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Dependencies struct {
	Pool         *pgxpool.Pool
	Transactions *database.TxManager
	Auth         config.AuthConfig
}

type Module struct {
	service    *application.Service
	authorizer api.Authorizer
	auth       config.AuthConfig
}

func New(dependencies Dependencies) *Module {
	repository := userspostgres.NewRepository(dependencies.Pool)
	return &Module{service: application.NewService(application.Dependencies{Users: repository, Sessions: repository, Transactions: dependencies.Transactions, PasswordMinLen: dependencies.Auth.PasswordMinLen, PasswordMaxLen: dependencies.Auth.PasswordMaxLen, AccessTokenTTL: dependencies.Auth.AccessTokenTTL, RefreshTokenTTL: dependencies.Auth.RefreshTokenTTL}), auth: dependencies.Auth}
}

func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "users", DisplayName: "Users", Version: "1.0.0", Dependencies: []string{"core"}}
}
func (m *Module) Service() *application.Service            { return m.service }
func (m *Module) SetAuthorizer(authorizer api.Authorizer)  { m.authorizer = authorizer }
func (m *Module) SetStatusGuard(guard api.UserStatusGuard) { m.service.SetStatusGuard(guard) }
func (m *Module) Register(ctx *module.Context) error       { return usersmetadata.Register(ctx) }
func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return usershttp.NewHandler(m.service, m.authorizer, m.auth).RegisterRoutes(routes)
}
func (Module) Migrations() []module.Migration { return usersmigrations.All() }
