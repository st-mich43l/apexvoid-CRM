package crm

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/application"
	crmpostgres "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/infrastructure/postgres"
	crmmetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/metadata"
	crmmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/migrations"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type Module struct{ service *application.Service }

func New(pool *pgxpool.Pool, tx *database.TxManager) *Module {
	return &Module{service: application.New(crmpostgres.New(pool), tx)}
}
func (m *Module) Service() *application.Service { return m.service }
func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "crm", DisplayName: "CRM", Version: "1.0.0", Dependencies: []string{"customization", "contacts"}}
}
func (Module) Register(ctx *module.Context) error { return crmmetadata.Register(ctx) }
func (Module) Migrations() []module.Migration     { return crmmigrations.All() }
