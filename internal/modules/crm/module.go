package crm

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	crmmetadata "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/metadata"
	crmmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/migrations"
)

type Module struct{}

func New() *Module { return &Module{} }
func (Module) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "crm", DisplayName: "CRM", Version: "1.0.0", Dependencies: []string{"customization", "contacts"}}
}
func (Module) Register(ctx *module.Context) error { return crmmetadata.Register(ctx) }
func (Module) Migrations() []module.Migration     { return crmmigrations.All() }
