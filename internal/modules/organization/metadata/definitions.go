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
		{Name: "organization.organization.read", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "View Organization", Description: "Allows viewing organization details."},
		{Name: "organization.organization.update", Module: "organization", Scope: permission.ScopePlatform, DisplayName: "Update Organization", Description: "Allows changing organization details."},
		{Name: "workspace.workspace.read", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "View Workspace", Description: "Allows viewing workspace details."},
		{Name: "workspace.workspace.create", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "Create Workspaces", Description: "Allows creating workspaces in the organization."},
		{Name: "workspace.workspace.update", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "Update Workspace", Description: "Allows changing workspace details."},
		{Name: "workspace.member.read", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "View Members", Description: "Allows viewing workspace members."},
		{Name: "workspace.member.add", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "Add Members", Description: "Allows adding existing users to a workspace."},
		{Name: "workspace.member.update", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "Update Members", Description: "Allows changing workspace membership status."},
		{Name: "workspace.member.remove", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "Remove Members", Description: "Allows removing users from a workspace."},
		{Name: "workspace.role.read", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "View Workspace Roles", Description: "Allows viewing workspace roles."},
		{Name: "workspace.role.manage", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "Manage Workspace Roles", Description: "Allows creating roles and changing their permissions."},
		{Name: "workspace.role.assign", Module: "organization", Scope: permission.ScopeWorkspace, DisplayName: "Assign Workspace Roles", Description: "Allows assigning roles to workspace members."},
	} {
		if err := ctx.Permissions.Register(item); err != nil {
			return err
		}
	}
	if err := ctx.Entities.Register(entity.Definition{Name: "organization.organization", DisplayName: "Organization", Module: "organization", Scope: entity.ScopeOrganization, Fields: []field.Definition{
		{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true},
		{Name: "name", DisplayName: "Name", Type: field.String, Required: true},
		{Name: "slug", DisplayName: "Slug", Type: field.String, ReadOnly: true},
		{Name: "status", DisplayName: "Status", Type: field.Enum, Required: true},
		{Name: "created_at", DisplayName: "Created At", Type: field.DateTime, ReadOnly: true},
		{Name: "updated_at", DisplayName: "Updated At", Type: field.DateTime, ReadOnly: true},
	}}); err != nil {
		return err
	}
	if err := ctx.Entities.Register(entity.Definition{Name: "workspace.workspace", DisplayName: "Workspace", Module: "organization", Scope: entity.ScopeOrganization, Fields: []field.Definition{
		{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true},
		{Name: "organization_id", DisplayName: "Organization", Type: field.UUID, Required: true},
		{Name: "name", DisplayName: "Name", Type: field.String, Required: true},
		{Name: "slug", DisplayName: "Slug", Type: field.String, ReadOnly: true},
		{Name: "timezone", DisplayName: "Timezone", Type: field.String, Required: true},
		{Name: "status", DisplayName: "Status", Type: field.Enum, Required: true},
		{Name: "created_at", DisplayName: "Created At", Type: field.DateTime, ReadOnly: true},
		{Name: "updated_at", DisplayName: "Updated At", Type: field.DateTime, ReadOnly: true},
	}}); err != nil {
		return err
	}
	if err := ctx.Entities.Register(entity.Definition{Name: "workspace.membership", DisplayName: "Workspace Membership", Module: "organization", Scope: entity.ScopeWorkspace, Fields: []field.Definition{
		{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true},
		{Name: "workspace_id", DisplayName: "Workspace", Type: field.UUID, Required: true},
		{Name: "user_id", DisplayName: "User", Type: field.UUID, Required: true},
		{Name: "status", DisplayName: "Status", Type: field.Enum, Required: true},
		{Name: "created_at", DisplayName: "Created At", Type: field.DateTime, ReadOnly: true},
		{Name: "updated_at", DisplayName: "Updated At", Type: field.DateTime, ReadOnly: true},
	}}); err != nil {
		return err
	}
	for _, item := range []event.Definition{
		{Name: "organization.organization.created", Module: "organization", Description: "Published when an organization is created."},
		{Name: "organization.organization.updated", Module: "organization", Description: "Published when an organization is updated."},
		{Name: "workspace.workspace.created", Module: "organization", Description: "Published when a workspace is created."},
		{Name: "workspace.workspace.updated", Module: "organization", Description: "Published when a workspace is updated."},
		{Name: "workspace.member.added", Module: "organization", Description: "Published when a member joins a workspace."},
		{Name: "workspace.member.updated", Module: "organization", Description: "Published when a membership changes."},
		{Name: "workspace.member.removed", Module: "organization", Description: "Published when a member leaves a workspace."},
	} {
		if err := ctx.Events.Register(item); err != nil {
			return err
		}
	}
	return nil
}
