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
		{Name: "users.user.read", Module: "users", Scope: permission.ScopePlatform, DisplayName: "Read Users", Description: "Allows viewing users."},
		{Name: "users.user.create", Module: "users", Scope: permission.ScopePlatform, DisplayName: "Create Users", Description: "Allows creating users."},
		{Name: "users.user.update", Module: "users", Scope: permission.ScopePlatform, DisplayName: "Update Users", Description: "Allows updating users."},
		{Name: "users.user.disable", Module: "users", Scope: permission.ScopePlatform, DisplayName: "Enable or Disable Users", Description: "Allows changing user status."},
	} {
		if err := ctx.Permissions.Register(item); err != nil {
			return err
		}
	}
	if err := ctx.Entities.Register(entity.Definition{Name: "users.user", DisplayName: "User", Module: "users", Fields: []field.Definition{
		{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true},
		{Name: "email", DisplayName: "Email", Type: field.String, Required: true},
		{Name: "display_name", DisplayName: "Display Name", Type: field.String, Required: true},
		{Name: "status", DisplayName: "Status", Type: field.Enum, Required: true},
		{Name: "created_at", DisplayName: "Created At", Type: field.DateTime, ReadOnly: true},
		{Name: "updated_at", DisplayName: "Updated At", Type: field.DateTime, ReadOnly: true},
	}}); err != nil {
		return err
	}
	for _, item := range []event.Definition{
		{Name: "users.user.created", Module: "users", Description: "Published when a user is created."},
		{Name: "users.user.updated", Module: "users", Description: "Published when a user is updated."},
		{Name: "users.user.disabled", Module: "users", Description: "Published when a user is disabled."},
		{Name: "users.user.enabled", Module: "users", Description: "Published when a user is enabled."},
		{Name: "users.user.logged_in", Module: "users", Description: "Published when a user logs in."},
	} {
		if err := ctx.Events.Register(item); err != nil {
			return err
		}
	}
	return nil
}
