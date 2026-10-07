package module

import "testing"

type migrationModule struct {
	descriptor Descriptor
	migrations []Migration
}

func (m migrationModule) Descriptor() Descriptor  { return m.descriptor }
func (m migrationModule) Register(*Context) error { return nil }
func (m migrationModule) Migrations() []Migration { return m.migrations }

func TestMigrationsFollowModuleDependencyOrder(t *testing.T) {
	registry := NewRegistry()
	if err := registry.Register(migrationModule{descriptor: Descriptor{Name: "sales", DisplayName: "Sales", Version: "1", Dependencies: []string{"contacts"}}, migrations: []Migration{{Module: "sales", Version: 1, Name: "orders"}}}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(migrationModule{descriptor: Descriptor{Name: "contacts", DisplayName: "Contacts", Version: "1"}, migrations: []Migration{{Module: "contacts", Version: 2, Name: "indexes"}, {Module: "contacts", Version: 1, Name: "contacts"}}}); err != nil {
		t.Fatal(err)
	}
	items, err := registry.Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || items[0].Module != "contacts" || items[0].Version != 1 || items[1].Version != 2 || items[2].Module != "sales" {
		t.Fatalf("unexpected migration order: %+v", items)
	}
}
