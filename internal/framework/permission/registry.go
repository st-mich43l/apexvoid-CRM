package permission

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Definition struct {
	Name        string
	Module      string
	DisplayName string
	Description string
}
type Registry struct{ definitions map[string]Definition }

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)

func NewRegistry() *Registry { return &Registry{definitions: make(map[string]Definition)} }
func (r *Registry) Register(definition Definition) error {
	if !namePattern.MatchString(definition.Name) {
		return fmt.Errorf("permission name %q must be lowercase and namespaced", definition.Name)
	}
	if strings.TrimSpace(definition.Module) == "" || strings.TrimSpace(definition.DisplayName) == "" {
		return fmt.Errorf("permission %q requires module and display name", definition.Name)
	}
	if !strings.HasPrefix(definition.Name, definition.Module+".") {
		return fmt.Errorf("permission %q must be owned by module %q", definition.Name, definition.Module)
	}
	if _, exists := r.definitions[definition.Name]; exists {
		return fmt.Errorf("permission %q is already registered", definition.Name)
	}
	r.definitions[definition.Name] = definition
	return nil
}
func (r *Registry) List() []Definition {
	result := make([]Definition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		result = append(result, definition)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (r *Registry) Get(name string) (Definition, bool) {
	definition, ok := r.definitions[name]
	return definition, ok
}
func (r *Registry) Contains(name string) bool { _, ok := r.definitions[name]; return ok }
