package metadata

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

func Register(ctx *module.Context) error {
	for _, p := range []permission.Definition{{Name: "crm.pipeline.read", Module: "crm", Scope: permission.ScopeWorkspace, DisplayName: "View CRM pipelines", Description: "Allows viewing sales pipeline configuration."}, {Name: "crm.pipeline.manage", Module: "crm", Scope: permission.ScopeWorkspace, DisplayName: "Manage CRM pipelines", Description: "Allows creating and maintaining pipelines."}, {Name: "crm.stage.manage", Module: "crm", Scope: permission.ScopeWorkspace, DisplayName: "Manage CRM stages", Description: "Allows configuring CRM pipeline stages."}} {
		if err := ctx.Permissions.Register(p); err != nil {
			return err
		}
	}
	for _, e := range []entity.Definition{{Name: "crm.pipeline", DisplayName: "Sales Pipeline", Module: "crm", Scope: entity.ScopeWorkspace, Fields: []field.Definition{{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true}, {Name: "name", DisplayName: "Name", Type: field.String, Required: true}, {Name: "status", DisplayName: "Status", Type: field.Enum, Required: true}}}, {Name: "crm.stage", DisplayName: "Pipeline Stage", Module: "crm", Scope: entity.ScopeWorkspace, Fields: []field.Definition{{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true}, {Name: "pipeline_id", DisplayName: "Pipeline", Type: field.RelationField, Required: true, Relation: &field.Relation{Kind: field.ManyToOne, Target: "crm.pipeline"}}, {Name: "name", DisplayName: "Name", Type: field.String, Required: true}, {Name: "category", DisplayName: "Category", Type: field.Enum, Required: true}}}} {
		if err := ctx.Entities.Register(e); err != nil {
			return err
		}
	}
	return nil
}
