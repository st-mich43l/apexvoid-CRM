// Package application defines the compiled-in application contract.
//
// An application is a product-facing composition of one or more modules. It
// is intentionally metadata only: application discovery must never load code
// or execute frontend assets supplied by the database.
package application

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var identifierPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)

// Frontend identifies the compiled frontend entry which implements an
// application. The backend exposes this metadata, but never serves or runs
// arbitrary frontend code from it.
type Frontend struct {
	EntryRoute   string
	NavigationID string
}

// Settings describes an optional, application-owned settings entry point.
// Platform settings remain owned by the platform modules.
type Settings struct {
	Route string
}

// Descriptor is the stable registration contract for a compiled-in
// application. ModuleDependencies, permissions, and capabilities are
// validated by the runtime after every module has registered its metadata.
type Descriptor struct {
	ID                   string
	DisplayName          string
	Description          string
	Version              string
	ModuleDependencies   []string
	RequiredPermissions  []string
	RequiredCapabilities []string
	Frontend             Frontend
	Settings             *Settings
}

type Registry struct{ definitions map[string]Descriptor }

func NewRegistry() *Registry { return &Registry{definitions: make(map[string]Descriptor)} }

func (r *Registry) Register(definition Descriptor) error {
	if !identifierPattern.MatchString(definition.ID) {
		return fmt.Errorf("application id %q must be lowercase and contain only letters, numbers, '.', '_' or '-'", definition.ID)
	}
	if strings.TrimSpace(definition.DisplayName) == "" {
		return fmt.Errorf("application %q display name cannot be empty", definition.ID)
	}
	if strings.TrimSpace(definition.Description) == "" {
		return fmt.Errorf("application %q description cannot be empty", definition.ID)
	}
	if strings.TrimSpace(definition.Version) == "" {
		return fmt.Errorf("application %q version cannot be empty", definition.ID)
	}
	if len(definition.ModuleDependencies) == 0 {
		return fmt.Errorf("application %q must declare at least one module dependency", definition.ID)
	}
	if err := validateIdentifiers("module dependency", definition.ID, definition.ModuleDependencies); err != nil {
		return err
	}
	if err := validateIdentifiers("required permission", definition.ID, definition.RequiredPermissions); err != nil {
		return err
	}
	if err := validateIdentifiers("required capability", definition.ID, definition.RequiredCapabilities); err != nil {
		return err
	}
	if !strings.HasPrefix(definition.Frontend.EntryRoute, "/") {
		return fmt.Errorf("application %q frontend entry route must start with /", definition.ID)
	}
	if !identifierPattern.MatchString(definition.Frontend.NavigationID) {
		return fmt.Errorf("application %q frontend navigation id is invalid", definition.ID)
	}
	if definition.Settings != nil && !strings.HasPrefix(definition.Settings.Route, "/") {
		return fmt.Errorf("application %q settings route must start with /", definition.ID)
	}
	if _, exists := r.definitions[definition.ID]; exists {
		return fmt.Errorf("application %q is already registered", definition.ID)
	}
	r.definitions[definition.ID] = clone(definition)
	return nil
}

func (r *Registry) Get(id string) (Descriptor, bool) {
	definition, ok := r.definitions[id]
	return clone(definition), ok
}

func (r *Registry) List() []Descriptor {
	result := make([]Descriptor, 0, len(r.definitions))
	for _, definition := range r.definitions {
		result = append(result, clone(definition))
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func validateIdentifiers(kind, applicationID string, values []string) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if !identifierPattern.MatchString(value) {
			return fmt.Errorf("application %q has invalid %s %q", applicationID, kind, value)
		}
		if _, exists := seen[value]; exists {
			return fmt.Errorf("application %q declares duplicate %s %q", applicationID, kind, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func clone(definition Descriptor) Descriptor {
	definition.ModuleDependencies = append([]string{}, definition.ModuleDependencies...)
	definition.RequiredPermissions = append([]string{}, definition.RequiredPermissions...)
	definition.RequiredCapabilities = append([]string{}, definition.RequiredCapabilities...)
	if definition.Settings != nil {
		settings := *definition.Settings
		definition.Settings = &settings
	}
	return definition
}
