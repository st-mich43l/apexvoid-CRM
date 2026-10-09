package crm

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	contactsapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/application"
	crmpostgres "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/infrastructure/postgres"
	crmmetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/metadata"
	crmmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/migrations"
	crmhttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/transport/http"
	customdomain "github.com/st-mich43l/apexvoid-CRM/internal/modules/customization/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Dependencies struct {
	Pool          *pgxpool.Pool
	Transactions  *database.TxManager
	Authenticator usersapi.Authenticator
	Workspace     organizationapi.WorkspaceResolver
	Access        organizationapi.WorkspaceAccess
	Contacts      interface {
		contactsapi.ContactReader
		contactsapi.ContactCreator
	}
	Customization interface {
		ValidateCustomValues(context.Context, uuid.UUID, string, map[string]any) error
		ListViews(context.Context, uuid.UUID, uuid.UUID, string) ([]customdomain.SavedView, error)
		EffectiveSchema(context.Context, uuid.UUID, string) (customdomain.EffectiveSchema, error)
	}
	Events *event.Bus
	Logger *slog.Logger
}

type Module struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	workspace     organizationapi.WorkspaceResolver
	access        organizationapi.WorkspaceAccess
}

func New(d Dependencies) *Module {
	return &Module{service: application.NewWithDependencies(application.Dependencies{Repository: crmpostgres.New(d.Pool), Transactions: d.Transactions, CustomValues: d.Customization, Contacts: d.Contacts, Workspace: d.Workspace, Access: d.Access, Events: d.Events, Logger: d.Logger}), authenticator: d.Authenticator, workspace: d.Workspace, access: d.Access}
}
func (m *Module) Service() *application.Service { return m.service }
func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "crm", DisplayName: "CRM", Version: "1.0.0", Dependencies: []string{"customization", "contacts"}}
}
func (Module) Register(ctx *module.Context) error {
	if err := crmmetadata.Register(ctx); err != nil {
		return err
	}
	return ctx.Applications.Register(frameworkapplication.Descriptor{
		ID:                  "crm",
		DisplayName:         "CRM",
		Description:         "Workspace-scoped lead, opportunity, and pipeline collaboration.",
		Version:             "1.0.0",
		ModuleDependencies:  []string{"crm", "contacts"},
		RequiredPermissions: []string{"crm.lead.read"},
		Frontend:            frameworkapplication.Frontend{EntryRoute: "/crm/leads", NavigationID: "crm"},
		Settings:            &frameworkapplication.Settings{Route: "/crm/settings/pipelines"},
	})
}
func (Module) Migrations() []module.Migration { return crmmigrations.All() }
func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return crmhttp.New(m.service, m.authenticator, m.workspace, m.access).RegisterRoutes(routes)
}
