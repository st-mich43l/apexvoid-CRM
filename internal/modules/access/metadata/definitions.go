package metadata

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

func Register(ctx *module.Context) error {
	for _, item := range []permission.Definition{
		{Name: "access.role.read", Module: "access", DisplayName: "Read Roles", Description: "Allows viewing roles."},
		{Name: "access.role.create", Module: "access", DisplayName: "Create Roles", Description: "Allows creating roles."},
		{Name: "access.role.update", Module: "access", DisplayName: "Update Roles", Description: "Allows updating roles and permissions."},
		{Name: "access.role.delete", Module: "access", DisplayName: "Delete Roles", Description: "Allows deleting non-system roles."},
		{Name: "access.role.assign", Module: "access", DisplayName: "Assign Roles", Description: "Allows assigning roles to users."},
		{Name: "access.permission.read", Module: "access", DisplayName: "Read Permissions", Description: "Allows viewing registered permissions."},
	} {
		if err := ctx.Permissions.Register(item); err != nil {
			return err
		}
	}
	if err := ctx.Entities.Register(entity.Definition{Name: "access.role", DisplayName: "Role", Module: "access", Fields: []field.Definition{
		{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true},
		{Name: "name", DisplayName: "Name", Type: field.String, Required: true},
		{Name: "display_name", DisplayName: "Display Name", Type: field.String, Required: true},
		{Name: "description", DisplayName: "Description", Type: field.Text},
		{Name: "system", DisplayName: "System Role", Type: field.Boolean, ReadOnly: true},
		{Name: "created_at", DisplayName: "Created At", Type: field.DateTime, ReadOnly: true},
		{Name: "updated_at", DisplayName: "Updated At", Type: field.DateTime, ReadOnly: true},
	}}); err != nil {
		return err
	}
	for _, item := range []event.Definition{
		{Name: "access.role.created", Module: "access", Description: "Published when a role is created."},
		{Name: "access.role.updated", Module: "access", Description: "Published when a role is updated."},
		{Name: "access.role.deleted", Module: "access", Description: "Published when a role is deleted."},
		{Name: "access.user_roles.changed", Module: "access", Description: "Published when a user's roles change."},
		{Name: "access.role_permissions.changed", Module: "access", Description: "Published when a role's permissions change."},
	} {
		if err := ctx.Events.Register(item); err != nil {
			return err
		}
	}
	return nil
}
