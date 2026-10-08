package entity

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
)

type Definition struct {
	Name         string
	DisplayName  string
	Module       string
	Scope        Scope
	Fields       []field.Definition
	Capabilities []string
}

type Registry struct{ definitions map[string]Definition }

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)

func NewRegistry() *Registry { return &Registry{definitions: make(map[string]Definition)} }

func (r *Registry) Register(definition Definition) error {
	if !identifierPattern.MatchString(definition.Name) {
		return fmt.Errorf("entity name %q must be a lowercase namespaced identifier", definition.Name)
	}
	if strings.TrimSpace(definition.DisplayName) == "" {
		return fmt.Errorf("entity %q display name cannot be empty", definition.Name)
	}
	if strings.TrimSpace(definition.Module) == "" {
		return fmt.Errorf("entity %q module cannot be empty", definition.Name)
	}
	if definition.Scope == "" {
		definition.Scope = ScopeGlobal
	}
	if !definition.Scope.Valid() {
		return fmt.Errorf("entity %q has invalid scope %q", definition.Name, definition.Scope)
	}
	if _, exists := r.definitions[definition.Name]; exists {
		return fmt.Errorf("entity %q is already registered", definition.Name)
	}
	seen := make(map[string]struct{}, len(definition.Fields))
	for _, fieldDefinition := range definition.Fields {
		if err := fieldDefinition.Validate(); err != nil {
			return err
		}
		if _, exists := seen[fieldDefinition.Name]; exists {
			return fmt.Errorf("entity %q has duplicate field %q", definition.Name, fieldDefinition.Name)
		}
		seen[fieldDefinition.Name] = struct{}{}
	}
	capabilities := make(map[string]struct{}, len(definition.Capabilities))
	for _, item := range definition.Capabilities {
		if !identifierPattern.MatchString(item) {
			return fmt.Errorf("entity %q has invalid capability %q", definition.Name, item)
		}
		if _, exists := capabilities[item]; exists {
			return fmt.Errorf("entity %q has duplicate capability %q", definition.Name, item)
		}
		capabilities[item] = struct{}{}
	}
	r.definitions[definition.Name] = cloneDefinition(definition)
	return nil
}

func (r *Registry) Get(name string) (Definition, bool) {
	definition, ok := r.definitions[name]
	return cloneDefinition(definition), ok
}

func (r *Registry) List() []Definition {
	result := make([]Definition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		result = append(result, cloneDefinition(definition))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (r *Registry) ValidateCapabilities(known func(string) bool) error {
	for _, definition := range r.definitions {
		for _, item := range definition.Capabilities {
			if !known(item) {
				return fmt.Errorf("entity %q references unknown capability %q", definition.Name, item)
			}
		}
	}
	return nil
}

func (r *Registry) ValidateRelations(known func(string) bool) error {
	for _, definition := range r.definitions {
		for _, fieldDefinition := range definition.Fields {
			if fieldDefinition.Relation != nil && !known(fieldDefinition.Relation.Target) {
				return fmt.Errorf("entity %q field %q references unknown relation target %q", definition.Name, fieldDefinition.Name, fieldDefinition.Relation.Target)
			}
		}
	}
	return nil
}

func cloneDefinition(definition Definition) Definition {
	definition.Fields = append([]field.Definition{}, definition.Fields...)
	definition.Capabilities = append([]string{}, definition.Capabilities...)
	return definition
}
