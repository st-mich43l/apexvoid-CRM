package customization

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	contactsapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/application"
	customizationpostgres "github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/infrastructure/postgres"
	customizationmetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/metadata"
	customizationmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/migrations"
	customizationhttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/transport/http"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Dependencies struct {
	Pool          *pgxpool.Pool
	Transactions  *database.TxManager
	Metadata      *metadata.Registry
	Contacts      contactsapi.CustomFieldReader
	Access        organizationapi.WorkspaceAccess
	Workspace     organizationapi.WorkspaceResolver
	Authenticator usersapi.Authenticator
}

type Module struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	workspace     organizationapi.WorkspaceResolver
	access        organizationapi.WorkspaceAccess
}

func New(d Dependencies) *Module {
	return &Module{service: application.NewService(application.Dependencies{Repository: customizationpostgres.NewRepository(d.Pool), Metadata: d.Metadata, Contacts: d.Contacts, Transactions: d.Transactions}), authenticator: d.Authenticator, workspace: d.Workspace, access: d.Access}
}

func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "customization", DisplayName: "Workspace Customization", Version: "1.0.0", Dependencies: []string{"contacts"}}
}
func (m *Module) Service() *application.Service      { return m.service }
func (m *Module) Register(ctx *module.Context) error { return customizationmetadata.Register(ctx) }
func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return customizationhttp.NewHandler(m.service, m.authenticator, m.workspace, m.access).RegisterRoutes(routes)
}
func (Module) Migrations() []module.Migration { return customizationmigrations.All() }
