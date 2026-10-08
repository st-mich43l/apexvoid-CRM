package metadata

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

func Register(ctx *module.Context) error {
	for _, item := range []permission.Definition{
		{Name: "customization.schema.read", Module: "customization", Scope: permission.ScopeWorkspace, DisplayName: "View workspace configuration", Description: "Allows reading effective workspace schemas and saved views."},
		{Name: "customization.field.manage", Module: "customization", Scope: permission.ScopeWorkspace, DisplayName: "Manage custom fields", Description: "Allows configuring custom fields and form sections."},
		{Name: "customization.view.manage", Module: "customization", Scope: permission.ScopeWorkspace, DisplayName: "Manage saved views", Description: "Allows creating personal and shared saved views."},
	} {
		if err := ctx.Permissions.Register(item); err != nil {
			return err
		}
	}
	for _, item := range []entity.Definition{
		{Name: "customization.field_definition", DisplayName: "Custom Field", Module: "customization", Scope: entity.ScopeWorkspace, Fields: []field.Definition{{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true}, {Name: "entity_name", DisplayName: "Entity", Type: field.String, Required: true}, {Name: "field_key", DisplayName: "Field Key", Type: field.String, Required: true}, {Name: "label", DisplayName: "Label", Type: field.String, Required: true}, {Name: "field_type", DisplayName: "Type", Type: field.Enum, Required: true}}},
		{Name: "customization.saved_view", DisplayName: "Saved View", Module: "customization", Scope: entity.ScopeWorkspace, Fields: []field.Definition{{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true}, {Name: "name", DisplayName: "Name", Type: field.String, Required: true}, {Name: "shared", DisplayName: "Shared", Type: field.Boolean, Required: true}}},
	} {
		if err := ctx.Entities.Register(item); err != nil {
			return err
		}
	}
	return nil
}
