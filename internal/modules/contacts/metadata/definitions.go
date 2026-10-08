package metadata

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

func Register(ctx *module.Context) error {
	permissions := []permission.Definition{
		{Name: "contacts.contact.read", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "View Contacts", Description: "Allows viewing contacts."},
		{Name: "contacts.contact.create", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "Create Contacts", Description: "Allows creating contacts."},
		{Name: "contacts.contact.update", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "Update Contacts", Description: "Allows editing contacts."},
		{Name: "contacts.contact.archive", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "Archive Contacts", Description: "Allows archiving and restoring contacts."},
		{Name: "contacts.tag.manage", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "Manage Contact Tags", Description: "Allows managing tags."},
		{Name: "contacts.note.read", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "View Contact Notes", Description: "Allows viewing contact notes."},
		{Name: "contacts.note.manage", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "Manage Contact Notes", Description: "Allows creating and editing notes."},
		{Name: "contacts.activity.read", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "View Activities", Description: "Allows viewing activities."},
		{Name: "contacts.activity.manage", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "Manage Activities", Description: "Allows managing activities."},
		{Name: "contacts.attachment.read", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "View Attachments", Description: "Allows viewing attachments."},
		{Name: "contacts.attachment.manage", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "Manage Attachments", Description: "Allows managing attachments."},
		{Name: "contacts.field.manage", Module: "contacts", Scope: permission.ScopeWorkspace, DisplayName: "Manage Contact Fields", Description: "Allows managing custom contact fields."},
	}
	for _, item := range permissions {
		if err := ctx.Permissions.Register(item); err != nil {
			return err
		}
	}
	if err := ctx.Entities.Register(entity.Definition{Name: "contacts.contact", DisplayName: "Contact", Module: "contacts", Scope: entity.ScopeWorkspace, Fields: []field.Definition{
		{Name: "id", DisplayName: "ID", Type: field.UUID, ReadOnly: true},
		{Name: "kind", DisplayName: "Kind", Type: field.Enum, Required: true},
		{Name: "display_name", DisplayName: "Display Name", Type: field.String, Required: true},
		{Name: "email", DisplayName: "Email", Type: field.String}, {Name: "phone", DisplayName: "Phone", Type: field.String},
		{Name: "website", DisplayName: "Website", Type: field.String}, {Name: "description", DisplayName: "Description", Type: field.Text},
		{Name: "status", DisplayName: "Status", Type: field.Enum, Required: true}, {Name: "custom_values", DisplayName: "Custom Values", Type: field.Text},
		{Name: "created_at", DisplayName: "Created At", Type: field.DateTime, ReadOnly: true}, {Name: "updated_at", DisplayName: "Updated At", Type: field.DateTime, ReadOnly: true},
	}}); err != nil {
		return err
	}
	for _, item := range []event.Definition{
		{Name: "contacts.contact.created", Module: "contacts", Description: "Published when a contact is created."},
		{Name: "contacts.contact.updated", Module: "contacts", Description: "Published when a contact is updated."},
		{Name: "contacts.contact.archived", Module: "contacts", Description: "Published when a contact is archived."},
		{Name: "contacts.relationship.changed", Module: "contacts", Description: "Published when a contact relationship changes."},
		{Name: "contacts.activity.created", Module: "contacts", Description: "Published when an activity is created."},
		{Name: "contacts.activity.completed", Module: "contacts", Description: "Published when an activity is completed."},
	} {
		if err := ctx.Events.Register(item); err != nil {
			return err
		}
	}
	return nil
}
