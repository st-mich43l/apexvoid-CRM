package capability

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

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)

func NewRegistry() *Registry { return &Registry{definitions: make(map[string]Definition)} }

func (r *Registry) Register(definition Definition) error {
	if !namePattern.MatchString(definition.Name) {
		return fmt.Errorf("invalid capability name %q", definition.Name)
	}
	if strings.TrimSpace(definition.Module) == "" || strings.TrimSpace(definition.DisplayName) == "" {
		return fmt.Errorf("capability %q requires module and display name", definition.Name)
	}
	if _, exists := r.definitions[definition.Name]; exists {
		return fmt.Errorf("capability %q is already registered", definition.Name)
	}
	r.definitions[definition.Name] = definition
	return nil
}

func (r *Registry) Contains(name string) bool { _, ok := r.definitions[name]; return ok }
func (r *Registry) List() []Definition {
	result := make([]Definition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		result = append(result, definition)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
