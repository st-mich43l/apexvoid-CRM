package application

import (
	"context"
	"testing"

	"github.com/google/uuid"
	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

type metadataReaderStub struct{ snapshot metadata.Snapshot }

func (s metadataReaderStub) Snapshot() metadata.Snapshot           { return s.snapshot }
func (metadataReaderStub) Entity(string) (entity.Definition, bool) { return entity.Definition{}, false }
func (s metadataReaderStub) SnapshotPermission(name string) (permission.Definition, bool) {
	for _, item := range s.snapshot.Permissions {
		if item.Name == name {
			return item, true
		}
	}
	return permission.Definition{}, false
}

type authorizerStub struct {
	platform  map[string]bool
	workspace map[string]bool
}

func (s authorizerStub) Can(_ context.Context, _ uuid.UUID, name string) (bool, error) {
	return s.platform[name], nil
}
func (s authorizerStub) CanInWorkspace(_ context.Context, _ uuid.UUID, _ uuid.UUID, name string) (bool, error) {
	return s.workspace[name], nil
}

func TestApplicationsEvaluatesAllAndAnyPoliciesAtPermissionScope(t *testing.T) {
	snapshot := metadata.Snapshot{
		Permissions: []permission.Definition{
			{Name: "platform.framework.read", Scope: permission.ScopePlatform},
			{Name: "crm.lead.read", Scope: permission.ScopeWorkspace},
			{Name: "crm.opportunity.read", Scope: permission.ScopeWorkspace},
		},
		Applications: []frameworkapplication.Descriptor{
			{ID: "all-policy", Access: frameworkapplication.Access{Entry: frameworkapplication.PermissionPolicy{Match: frameworkapplication.PermissionMatchAll, Permissions: []string{"platform.framework.read", "crm.lead.read"}}}},
			{ID: "any-policy", Access: frameworkapplication.Access{Entry: frameworkapplication.PermissionPolicy{Match: frameworkapplication.PermissionMatchAny, Permissions: []string{"crm.lead.read", "crm.opportunity.read"}}}},
		},
	}
	service := NewService(metadataReaderStub{snapshot: snapshot}, authorizerStub{platform: map[string]bool{"platform.framework.read": true}, workspace: map[string]bool{"crm.opportunity.read": true}})
	applications, err := service.Applications(t.Context(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if applications[0].EntryAuthorized {
		t.Fatal("all-of policy must reject a user missing crm.lead.read")
	}
	if !applications[1].EntryAuthorized {
		t.Fatal("any-of policy must authorize an opportunity-only user")
	}
}

func TestApplicationsEvaluatesSettingsSeparately(t *testing.T) {
	snapshot := metadata.Snapshot{
		Permissions: []permission.Definition{
			{Name: "contacts.contact.read", Scope: permission.ScopeWorkspace},
			{Name: "contacts.field.manage", Scope: permission.ScopeWorkspace},
		},
		Applications: []frameworkapplication.Descriptor{{
			ID: "contacts",
			Access: frameworkapplication.Access{
				Entry:    frameworkapplication.PermissionPolicy{Match: frameworkapplication.PermissionMatchAll, Permissions: []string{"contacts.contact.read"}},
				Settings: &frameworkapplication.PermissionPolicy{Match: frameworkapplication.PermissionMatchAny, Permissions: []string{"contacts.field.manage"}},
			},
		}},
	}
	service := NewService(metadataReaderStub{snapshot: snapshot}, authorizerStub{workspace: map[string]bool{"contacts.contact.read": true}})
	applications, err := service.Applications(t.Context(), uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	if !applications[0].EntryAuthorized || applications[0].SettingsAuthorized {
		t.Fatalf("entry and settings authorization must be independent: %+v", applications[0])
	}
}
