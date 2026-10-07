package module

import (
	"context"
	"strings"
	"testing"
)

type testModule struct {
	descriptor  Descriptor
	initialized *[]string
}

func (m testModule) Descriptor() Descriptor { return m.descriptor }
func (m testModule) Register(_ *Context) error {
	if m.initialized != nil {
		*m.initialized = append(*m.initialized, m.descriptor.Name)
	}
	return nil
}

func TestRegistryResolvesDependenciesDeterministically(t *testing.T) {
	initialized := []string{}
	registry := NewRegistry()
	for _, item := range []testModule{{descriptor: Descriptor{Name: "sales", DisplayName: "Sales", Version: "1.0.0", Dependencies: []string{"crm"}}, initialized: &initialized}, {descriptor: Descriptor{Name: "crm", DisplayName: "CRM", Version: "1.0.0", Dependencies: []string{"contacts"}}, initialized: &initialized}, {descriptor: Descriptor{Name: "contacts", DisplayName: "Contacts", Version: "1.0.0"}, initialized: &initialized}} {
		if err := registry.Register(item); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := registry.Initialize(context.Background(), &Context{}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(initialized, ",") != "contacts,crm,sales" {
		t.Fatalf("unexpected initialization order: %v", initialized)
	}
}

func TestRegistryRejectsDuplicateMissingAndCyclicDependencies(t *testing.T) {
	registry := NewRegistry()
	a := testModule{descriptor: Descriptor{Name: "a", DisplayName: "A", Version: "1.0.0", Dependencies: []string{"b"}}}
	if err := registry.Register(a); err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(a); err == nil {
		t.Fatal("expected duplicate error")
	}
	if _, err := registry.Resolve(); err == nil || !strings.Contains(err.Error(), "missing dependency") {
		t.Fatalf("unexpected missing dependency error: %v", err)
	}
	cycle := NewRegistry()
	if err := cycle.Register(testModule{descriptor: Descriptor{Name: "a", DisplayName: "A", Version: "1.0.0", Dependencies: []string{"b"}}}); err != nil {
		t.Fatal(err)
	}
	if err := cycle.Register(testModule{descriptor: Descriptor{Name: "b", DisplayName: "B", Version: "1.0.0", Dependencies: []string{"a"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := cycle.Resolve(); err == nil || !strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("unexpected cycle error: %v", err)
	}
}
