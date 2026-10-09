package erp

import (
	"github.com/jackc/pgx/v5/pgxpool"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/infrastructure/postgres"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/migrations"
	erphttp "github.com/st-mich43l/apexvoid-CRM/internal/modules/erp/transport/http"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
)

type Dependencies struct {
	Pool *pgxpool.Pool
	Authenticator usersapi.Authenticator
	Workspace organizationapi.WorkspaceResolver
	Access organizationapi.WorkspaceAccess
}
type Module struct {
	service *application.Service
	auth usersapi.Authenticator
	workspace organizationapi.WorkspaceResolver
	access organizationapi.WorkspaceAccess
}
func New(d Dependencies) *Module {
	return &Module{service: application.New(postgres.New(d.Pool)),auth:d.Authenticator,workspace:d.Workspace,access:d.Access}
}
func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name:"erp",DisplayName:"ERP",Version:"1.0.0",Dependencies:[]string{"organization","access"}}
}
func (Module) Register(ctx *module.Context) error {
	for _, p := range []permission.Definition{
		{Name:"erp.product.read",Module:"erp",Scope:permission.ScopeWorkspace,DisplayName:"View ERP products"},
		{Name:"erp.product.create",Module:"erp",Scope:permission.ScopeWorkspace,DisplayName:"Create ERP products"},
		{Name:"erp.product.update",Module:"erp",Scope:permission.ScopeWorkspace,DisplayName:"Edit ERP products"},
		{Name:"erp.product.archive",Module:"erp",Scope:permission.ScopeWorkspace,DisplayName:"Archive ERP products"},
	} {
		if err := ctx.Permissions.Register(p); err != nil { return err }
	}
	if err := ctx.Entities.Register(entity.Definition{Name:"erp.product",DisplayName:"Product or Service",Module:"erp",Scope:entity.ScopeWorkspace,Fields:[]field.Definition{
		{Name:"id",DisplayName:"ID",Type:field.UUID,ReadOnly:true},
		{Name:"sku",DisplayName:"SKU",Type:field.String,Required:true},
		{Name:"name",DisplayName:"Name",Type:field.String,Required:true},
		{Name:"description",DisplayName:"Description",Type:field.Text},
		{Name:"kind",DisplayName:"Kind",Type:field.Enum,Required:true},
		{Name:"unit",DisplayName:"Unit",Type:field.Enum,Required:true},
		{Name:"unit_price",DisplayName:"Unit Price",Type:field.Decimal,Required:true},
		{Name:"currency",DisplayName:"Currency",Type:field.String,Required:true},
		{Name:"status",DisplayName:"Status",Type:field.Enum,Required:true},
		{Name:"created_at",DisplayName:"Created At",Type:field.DateTime,ReadOnly:true},
		{Name:"updated_at",DisplayName:"Updated At",Type:field.DateTime,ReadOnly:true},
	}}); err != nil { return err }
	return ctx.Applications.Register(frameworkapplication.Descriptor{
		ID:"erp",DisplayName:"ERP",Description:"Workspace-scoped enterprise products and services catalog.",
		Version:"1.0.0",APIContractVersion:"v1",ModuleDependencies:[]string{"erp"},
		RequiredPermissions:[]string{"erp.product.read"},
		Frontend:frameworkapplication.Frontend{EntryRoute:"/erp",NavigationID:"erp"},
		Access:frameworkapplication.Access{Entry:frameworkapplication.PermissionPolicy{Match:frameworkapplication.PermissionMatchAll,Permissions:[]string{"erp.product.read"}}},
	})
}
func (Module) Migrations() []module.Migration { return migrations.All() }
func (m *Module) RegisterRoutes(r module.RouteRegistry) error {
	return erphttp.New(m.service,m.auth,m.workspace,m.access).RegisterRoutes(r)
}
