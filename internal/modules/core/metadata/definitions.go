package metadata

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/capability"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/extension"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

func Register(ctx *module.Context) error {
	if err := ctx.Capabilities.Register(capability.Definition{Name: "core.auditable", Module: "core", DisplayName: "Auditable", Description: "Marks an entity as eligible for audit history."}); err != nil {
		return err
	}
	if err := ctx.Entities.Register(entity.Definition{Name: "core.example", DisplayName: "Example Record", Module: "core", Fields: []field.Definition{{Name: "name", DisplayName: "Name", Type: field.String, Required: true}}, Capabilities: []string{"core.auditable"}}); err != nil {
		return err
	}
	if err := ctx.Permissions.Register(permission.Definition{Name: "core.example.read", Module: "core", Scope: permission.ScopePlatform, DisplayName: "Read Example Records", Description: "Allows reading example records."}); err != nil {
		return err
	}
	if err := ctx.Permissions.Register(permission.Definition{Name: "core.framework.read", Module: "core", Scope: permission.ScopePlatform, DisplayName: "Read Framework Metadata", Description: "Allows viewing technical framework metadata."}); err != nil {
		return err
	}
	if err := ctx.Permissions.Register(permission.Definition{Name: "core.application.manage", Module: "core", Scope: permission.ScopePlatform, DisplayName: "Manage External Applications", Description: "Allows managing trusted external application integrations."}); err != nil {
		return err
	}
	if err := ctx.Events.Register(event.Definition{Name: "core.example.created", Module: "core", Description: "Published when an example record is created."}); err != nil {
		return err
	}
	return ctx.Extensions.RegisterPoint(extension.Point{Name: "core.navigation", Module: "core", DisplayName: "Navigation Contributors", Description: "Allows modules to contribute navigation metadata."})
}
