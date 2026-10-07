package extension

import "testing"

func TestRegistryProvidesDeterministicExplicitExtensions(t *testing.T) {
	registry := NewRegistry()
	if err := registry.RegisterPoint(Point{Name: "core.navigation", Module: "core", DisplayName: "Navigation"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register("core.navigation", Implementation{Name: "sales", Module: "sales", Value: "sales-nav"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register("core.navigation", Implementation{Name: "contacts", Module: "contacts", Value: "contacts-nav"}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register("core.navigation", Implementation{Name: "contacts", Module: "contacts", Value: "duplicate"}); err == nil {
		t.Fatal("expected duplicate extension error")
	}
	items := registry.Implementations("core.navigation")
	if len(items) != 2 || items[0].Name != "contacts" || items[1].Name != "sales" {
		t.Fatalf("unexpected extension order: %+v", items)
	}
}
