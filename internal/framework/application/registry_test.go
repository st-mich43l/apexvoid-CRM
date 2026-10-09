package application

import (
	"strings"
	"testing"
)

func fixture() Descriptor {
	return Descriptor{
		ID:                   "fixture",
		DisplayName:          "Framework Fixture",
		Description:          "A non-business registration fixture.",
		Version:              "1.0.0",
		ModuleDependencies:   []string{"fixture"},
		RequiredPermissions:  []string{"fixture.record.read"},
		RequiredCapabilities: []string{"fixture.auditable"},
		Frontend:             Frontend{EntryRoute: "/fixture", NavigationID: "fixture-home"},
	}
}

func TestRegistryRejectsInvalidAndDuplicateApplicationContracts(t *testing.T) {
	registry := NewRegistry()
	definition := fixture()
	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(definition); err == nil {
		t.Fatal("expected duplicate application error")
	}
	definition = fixture()
	definition.Frontend.EntryRoute = "fixture"
	if err := NewRegistry().Register(definition); err == nil || !strings.Contains(err.Error(), "entry route") {
		t.Fatalf("unexpected invalid route error: %v", err)
	}
	definition = fixture()
	definition.ModuleDependencies = []string{"fixture", "fixture"}
	if err := NewRegistry().Register(definition); err == nil || !strings.Contains(err.Error(), "duplicate module dependency") {
		t.Fatalf("unexpected duplicate dependency error: %v", err)
	}
}

func TestRegistryReturnsImmutableSortedDescriptors(t *testing.T) {
	registry := NewRegistry()
	alpha := fixture()
	alpha.ID = "alpha"
	alpha.ModuleDependencies = []string{"alpha"}
	alpha.Frontend.NavigationID = "alpha-home"
	if err := registry.Register(fixture()); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(alpha); err != nil {
		t.Fatal(err)
	}
	items := registry.List()
	if len(items) != 2 || items[0].ID != "alpha" {
		t.Fatalf("unexpected application order: %+v", items)
	}
	items[0].ModuleDependencies[0] = "mutated"
	stored, ok := registry.Get("alpha")
	if !ok || stored.ModuleDependencies[0] != "alpha" {
		t.Fatalf("application registry leaked mutable state: %+v", stored)
	}
}
