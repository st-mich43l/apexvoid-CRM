package entity

import (
	"strings"
	"testing"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
)

func TestRegistryValidatesEntitiesAndFields(t *testing.T) {
	registry := NewRegistry()
	definition := Definition{Name: "core.record", DisplayName: "Record", Module: "core", Fields: []field.Definition{{Name: "name", DisplayName: "Name", Type: field.String, Required: true}}}
	if err := registry.Register(definition); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(definition); err == nil {
		t.Fatal("expected duplicate entity error")
	}
	bad := Definition{Name: "core.bad", DisplayName: "Bad", Module: "core", Fields: []field.Definition{{Name: "State", DisplayName: "State", Type: field.String}}}
	if err := registry.Register(bad); err == nil || !strings.Contains(err.Error(), "field name") {
		t.Fatalf("unexpected field validation error: %v", err)
	}
	relation := Definition{Name: "core.link", DisplayName: "Link", Module: "core", Fields: []field.Definition{{Name: "target", DisplayName: "Target", Type: field.RelationField, Relation: &field.Relation{Kind: field.ManyToOne, Target: "missing.record"}}}}
	if err := registry.Register(relation); err != nil {
		t.Fatal(err)
	}
	if err := registry.ValidateRelations(func(name string) bool { return name == "core.record" }); err == nil || !strings.Contains(err.Error(), "unknown relation target") {
		t.Fatalf("unexpected relation validation error: %v", err)
	}
}
