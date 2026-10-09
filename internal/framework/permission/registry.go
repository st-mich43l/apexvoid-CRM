package permission

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

type Definition struct {
	Name        string
	Module      string
	Scope       Scope
	DisplayName string
	Description string
}
type Registry struct {
	mu          sync.RWMutex
	definitions map[string]Definition
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)

func NewRegistry() *Registry { return &Registry{definitions: make(map[string]Definition)} }
func (r *Registry) Register(definition Definition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !namePattern.MatchString(definition.Name) {
		return fmt.Errorf("permission name %q must be lowercase and namespaced", definition.Name)
	}
	if strings.TrimSpace(definition.Module) == "" || strings.TrimSpace(definition.DisplayName) == "" {
		return fmt.Errorf("permission %q requires module and display name", definition.Name)
	}
	if definition.Scope == "" {
		definition.Scope = ScopePlatform
	}
	if !definition.Scope.Valid() {
		return fmt.Errorf("permission %q has invalid scope %q", definition.Name, definition.Scope)
	}
	if _, exists := r.definitions[definition.Name]; exists {
		return fmt.Errorf("permission %q is already registered", definition.Name)
	}
	r.definitions[definition.Name] = definition
	return nil
}
func (r *Registry) List() []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Definition, 0, len(r.definitions))
	for _, definition := range r.definitions {
		result = append(result, definition)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (r *Registry) Get(name string) (Definition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	definition, ok := r.definitions[name]
	return definition, ok
}

// UpdatePresentation changes non-authoritative catalog text without changing a
// permission name or scope. Existing role grants therefore remain intact.
func (r *Registry) UpdatePresentation(name, displayName, description string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	definition, ok := r.definitions[name]
	if !ok {
		return fmt.Errorf("permission %q is not registered", name)
	}
	if strings.TrimSpace(displayName) == "" {
		return fmt.Errorf("permission %q requires display name", name)
	}
	definition.DisplayName = displayName
	definition.Description = description
	r.definitions[name] = definition
	return nil
}
func (r *Registry) Contains(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.definitions[name]
	return ok
}
