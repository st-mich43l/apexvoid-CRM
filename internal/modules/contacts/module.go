package contacts

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/application"
	contactspostgres "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/infrastructure/postgres"
	contactsstorage "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/infrastructure/storage"
	contactsmetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/metadata"
	contactsmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/migrations"
	contactshttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/contacts/transport/http"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Dependencies struct {
	Pool           *pgxpool.Pool
	Transactions   *database.TxManager
	Permissions    *permission.Registry
	Access         organizationapi.WorkspaceAccess
	Workspace      organizationapi.WorkspaceResolver
	Authenticator  usersapi.Authenticator
	Events         *event.Bus
	Logger         *slog.Logger
	UploadDir      string
	MaxUploadBytes int64
}
type Module struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	workspace     organizationapi.WorkspaceResolver
	access        organizationapi.WorkspaceAccess
}

func New(d Dependencies) (*Module, error) {
	if d.UploadDir == "" {
		d.UploadDir = filepath.Join(os.TempDir(), "apexvoid-contacts")
	}
	store, err := contactsstorage.NewFileSystem(d.UploadDir)
	if err != nil {
		return nil, err
	}
	repository := contactspostgres.NewRepository(d.Pool)
	service := application.NewService(application.Dependencies{Repository: repository, Transactions: d.Transactions, Access: d.Access, Events: d.Events, Logger: d.Logger, Store: store, MaxUploadBytes: d.MaxUploadBytes})
	return &Module{service: service, authenticator: d.Authenticator, workspace: d.Workspace, access: d.Access}, nil
}
func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "contacts", DisplayName: "Contacts", Version: "1.0.0", Dependencies: []string{"access", "organization"}}
}
func (m *Module) Service() *application.Service { return m.service }
func (m *Module) Register(ctx *module.Context) error {
	if err := contactsmetadata.Register(ctx); err != nil {
		return err
	}
	return ctx.Applications.Register(frameworkapplication.Descriptor{
		ID:                  "contacts",
		DisplayName:         "Contacts",
		Description:         "Workspace-scoped people, companies, activities, and shared relationship records.",
		Version:             "1.0.0",
		ModuleDependencies:  []string{"contacts"},
		RequiredPermissions: []string{"contacts.contact.read"},
		Frontend:            frameworkapplication.Frontend{EntryRoute: "/contacts/people", NavigationID: "contacts"},
		Settings:            &frameworkapplication.Settings{Route: "/contacts/settings"},
	})
}
func (m *Module) RegisterRoutes(routes module.RouteRegistry) error {
	return contactshttp.NewHandler(m.service, m.authenticator, m.workspace, m.access).RegisterRoutes(routes)
}
func (Module) Migrations() []module.Migration { return contactsmigrations.All() }
