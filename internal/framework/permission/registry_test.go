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
