package permission

import (
	"strings"
	"testing"
)

func TestRegistryRejectsInvalidAndDuplicatePermissions(t *testing.T) {
	registry := NewRegistry()
	definition := Definition{Name: "core.record.read", Module: "core", DisplayName: "Read records"}
	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(definition); err == nil {
		t.Fatal("expected duplicate permission error")
	}
	if err := registry.Register(Definition{Name: "Read", Module: "core", DisplayName: "Invalid"}); err == nil || !strings.Contains(err.Error(), "permission name") {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestRegistryUpdatesPresentationWithoutChangingPermissionIdentity(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(Definition{Name: "reports.report.read", Module: "external.reports", Scope: ScopeWorkspace, DisplayName: "Read reports"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.UpdatePresentation("reports.report.read", "Read workspace reports", "Updated safely"); err != nil {
		t.Fatal(err)
	}
	item, ok := registry.Get("reports.report.read")
	if !ok || item.Scope != ScopeWorkspace || item.Module != "external.reports" || item.DisplayName != "Read workspace reports" {
		t.Fatalf("unexpected updated permission: %#v", item)
	}
}
