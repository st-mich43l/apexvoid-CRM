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
	if err := ValidateDefinition(definition); err != nil {
		return err
	}
	definition = normalize(definition)
	if _, exists := r.definitions[definition.Name]; exists {
		return fmt.Errorf("permission %q is already registered", definition.Name)
	}
	r.definitions[definition.Name] = definition
	return nil
}

func ValidateDefinition(definition Definition) error {
	if !namePattern.MatchString(definition.Name) {
		return fmt.Errorf("permission name %q must be lowercase and namespaced", definition.Name)
	}
	if strings.TrimSpace(definition.Module) == "" || strings.TrimSpace(definition.DisplayName) == "" {
		return fmt.Errorf("permission %q requires module and display name", definition.Name)
	}
	if definition.Scope != "" && !definition.Scope.Valid() {
		return fmt.Errorf("permission %q has invalid scope %q", definition.Name, definition.Scope)
	}
	return nil
}
func normalize(definition Definition) Definition {
	if definition.Scope == "" {
		definition.Scope = ScopePlatform
	}
	return definition
}

// RegisterBatch validates every definition before modifying the live catalog.
func (r *Registry) RegisterBatch(definitions []Definition) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	seen := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		if _, duplicate := seen[definition.Name]; duplicate {
			return fmt.Errorf("permission %q is duplicated in batch", definition.Name)
		}
		seen[definition.Name] = struct{}{}
		if err := ValidateDefinition(definition); err != nil {
			return err
		}
		if _, exists := r.definitions[definition.Name]; exists {
			return fmt.Errorf("permission %q is already registered", definition.Name)
		}
	}
	for _, definition := range definitions {
		definition = normalize(definition)
		r.definitions[definition.Name] = definition
	}
	return nil
}

// UnregisterBatch removes matching live definitions. Database grants are kept
// as historical records but no longer become effective through this registry.
func (r *Registry) UnregisterBatch(module string, names []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, name := range names {
		if definition, ok := r.definitions[name]; ok && definition.Module == module {
			delete(r.definitions, name)
		}
	}
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
