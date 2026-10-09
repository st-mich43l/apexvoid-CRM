package cafe

import (
	"github.com/jackc/pgx/v5/pgxpool"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/cafe/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/cafe/migrations"
	cafehttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/cafe/transport/http"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/api"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Dependencies struct {
	Pool          *pgxpool.Pool
	Transactions  *database.TxManager
	Products      api.ProductReader
	Authenticator usersapi.Authenticator
	Workspace     organizationapi.WorkspaceResolver
	Access        organizationapi.WorkspaceAccess
}
type Module struct {
	service   *application.Service
	auth      usersapi.Authenticator
	workspace organizationapi.WorkspaceResolver
	access    organizationapi.WorkspaceAccess
}

func New(d Dependencies) *Module {
	return &Module{
		service: application.New(d.Pool, d.Transactions, d.Products),
		auth:    d.Authenticator, workspace: d.Workspace, access: d.Access,
	}
}
func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "cafe", DisplayName: "Café & Photo Booth", Version: "1.0.0", Dependencies: []string{"erp", "organization", "access"}}
}
func (Module) Register(ctx *module.Context) error {
	for _, p := range []permission.Definition{
		{Name: "cafe.order.read", Module: "cafe", Scope: permission.ScopeWorkspace, DisplayName: "View café orders"},
		{Name: "cafe.order.create", Module: "cafe", Scope: permission.ScopeWorkspace, DisplayName: "Create café orders"},
		{Name: "cafe.order.manage", Module: "cafe", Scope: permission.ScopeWorkspace, DisplayName: "Complete or cancel café orders"},
		{Name: "cafe.booth.read", Module: "cafe", Scope: permission.ScopeWorkspace, DisplayName: "View photo booths"},
		{Name: "cafe.booth.manage", Module: "cafe", Scope: permission.ScopeWorkspace, DisplayName: "Manage photo booths"},
		{Name: "cafe.booking.read", Module: "cafe", Scope: permission.ScopeWorkspace, DisplayName: "View photo bookings"},
		{Name: "cafe.booking.create", Module: "cafe", Scope: permission.ScopeWorkspace, DisplayName: "Reserve photo booths"},
		{Name: "cafe.booking.manage", Module: "cafe", Scope: permission.ScopeWorkspace, DisplayName: "Manage photo sessions"},
	} {
		if err := ctx.Permissions.Register(p); err != nil {
			return err
		}
	}
	for _, def := range []entity.Definition{
		{Name: "cafe.order", DisplayName: "Café Order", Module: "cafe", Scope: entity.ScopeWorkspace, Fields: []field.Definition{
			{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true},
			{Name: "status", DisplayName: "Status", Type: field.Enum, Required: true},
			{Name: "total", DisplayName: "Total", Type: field.Decimal, ReadOnly: true},
			{Name: "currency", DisplayName: "Currency", Type: field.String, Required: true},
		}},
		{Name: "cafe.booking", DisplayName: "Photo Booking", Module: "cafe", Scope: entity.ScopeWorkspace, Fields: []field.Definition{
			{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true},
			{Name: "booth_id", DisplayName: "Booth", Type: field.UUID, Required: true},
			{Name: "package_product_id", DisplayName: "Package", Type: field.UUID, Required: true},
			{Name: "starts_at", DisplayName: "Start", Type: field.DateTime, Required: true},
			{Name: "ends_at", DisplayName: "End", Type: field.DateTime, Required: true},
		}},
	} {
		if err := ctx.Entities.Register(def); err != nil {
			return err
		}
	}
	return ctx.Applications.Register(frameworkapplication.Descriptor{
		ID: "cafe", DisplayName: "Café & Photo Booth", Description: "In-store café orders and self-photo-booth scheduling.",
		Version: "1.0.0", APIContractVersion: "v1", ModuleDependencies: []string{"cafe", "erp"},
		RequiredPermissions: []string{"cafe.order.read", "cafe.booking.read"},
		Frontend:            frameworkapplication.Frontend{EntryRoute: "/cafe", NavigationID: "cafe"},
		Access:              frameworkapplication.Access{Entry: frameworkapplication.PermissionPolicy{Match: frameworkapplication.PermissionMatchAny, Permissions: []string{"cafe.order.read", "cafe.booking.read"}}},
	})
}
func (Module) Migrations() []module.Migration { return migrations.All() }
func (m *Module) RegisterRoutes(r module.RouteRegistry) error {
	return cafehttp.New(m.service, m.auth, m.workspace, m.access).RegisterRoutes(r)
}
