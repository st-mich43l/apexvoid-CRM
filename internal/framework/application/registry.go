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

// DeploymentType identifies whether an application is compiled into ApexVoid
// or represented by a trusted, remotely deployed service contract.
type DeploymentType string

const (
	DeploymentInternal DeploymentType = "internal"
	DeploymentExternal DeploymentType = "external"
)

// ExternalService is metadata for a service that ApexVoid integrates with. It
// is deliberately a contract only: the registry never loads or executes code
// from this data.
type ExternalService struct {
	ServiceIdentity string
	Endpoint        string
	HealthEndpoint  string
}

// Settings describes an optional, application-owned settings entry point.
// Platform settings remain owned by the platform modules.
type Settings struct {
	Route string
}

// PermissionMatch declares whether every permission or at least one
// permission in a policy is required. Policies are intentionally small and
// typed; they are not a general purpose authorization-expression language.
type PermissionMatch string

const (
	PermissionMatchAll PermissionMatch = "all"
	PermissionMatchAny PermissionMatch = "any"
)

// PermissionPolicy controls access to an application entry or settings route.
// Permission scope is resolved from the registered permission definition at
// runtime, so platform and workspace permissions retain their normal
// authorization boundaries.
type PermissionPolicy struct {
	Match       PermissionMatch
	Permissions []string
}

type Access struct {
	Entry    PermissionPolicy
	Settings *PermissionPolicy
}

// Descriptor is the stable registration contract for a compiled-in
// application. ModuleDependencies, permissions, and capabilities are
// validated by the runtime after every module has registered its metadata.
type Descriptor struct {
	ID                 string
	DisplayName        string
	Description        string
	Version            string
	APIContractVersion string
	ModuleDependencies []string
	// RequiredPermissions are static framework dependencies validated during
	// startup. They are not the user-authorization policy for opening an app.
	RequiredPermissions  []string
	RequiredCapabilities []string
	Frontend             Frontend
	Settings             *Settings
	Access               Access
	Deployment           DeploymentType
	External             *ExternalService
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
	if !regexp.MustCompile(`^v[1-9][0-9]*$`).MatchString(definition.APIContractVersion) {
		return fmt.Errorf("application %q API contract version must use the form vN", definition.ID)
	}
	if definition.Deployment == "" {
		definition.Deployment = DeploymentInternal
	}
	if definition.Deployment != DeploymentInternal && definition.Deployment != DeploymentExternal {
		return fmt.Errorf("application %q has invalid deployment type %q", definition.ID, definition.Deployment)
	}
	if definition.Deployment == DeploymentInternal && definition.External != nil {
		return fmt.Errorf("internal application %q cannot declare an external service", definition.ID)
	}
	if definition.Deployment == DeploymentExternal {
		if definition.External == nil || strings.TrimSpace(definition.External.ServiceIdentity) == "" || strings.TrimSpace(definition.External.Endpoint) == "" || strings.TrimSpace(definition.External.HealthEndpoint) == "" {
			return fmt.Errorf("external application %q requires service identity, endpoint and health endpoint", definition.ID)
		}
	}
	if definition.Deployment == DeploymentInternal && len(definition.ModuleDependencies) == 0 {
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
	if definition.Frontend.EntryRoute != "" && !strings.HasPrefix(definition.Frontend.EntryRoute, "/") {
		return fmt.Errorf("application %q frontend entry route must start with /", definition.ID)
	}
	if definition.Frontend.NavigationID != "" && !identifierPattern.MatchString(definition.Frontend.NavigationID) {
		return fmt.Errorf("application %q frontend navigation id is invalid", definition.ID)
	}
	if definition.Deployment == DeploymentInternal && (definition.Frontend.EntryRoute == "" || definition.Frontend.NavigationID == "") {
		return fmt.Errorf("internal application %q requires frontend route and navigation id", definition.ID)
	}
	if definition.Settings != nil && !strings.HasPrefix(definition.Settings.Route, "/") {
		return fmt.Errorf("application %q settings route must start with /", definition.ID)
	}
	if err := validatePolicy(definition.ID, "entry", definition.Access.Entry); err != nil {
		return err
	}
	if definition.Settings == nil && definition.Access.Settings != nil {
		return fmt.Errorf("application %q declares settings access without a settings route", definition.ID)
	}
	if definition.Settings != nil {
		if definition.Access.Settings == nil {
			return fmt.Errorf("application %q settings route requires an access policy", definition.ID)
		}
		if err := validatePolicy(definition.ID, "settings", *definition.Access.Settings); err != nil {
			return err
		}
	}
	if _, exists := r.definitions[definition.ID]; exists {
		return fmt.Errorf("application %q is already registered", definition.ID)
	}
	r.definitions[definition.ID] = clone(definition)
	return nil
}

func validatePolicy(applicationID, name string, policy PermissionPolicy) error {
	if policy.Match != PermissionMatchAll && policy.Match != PermissionMatchAny {
		return fmt.Errorf("application %q %s access policy has invalid permission match %q", applicationID, name, policy.Match)
	}
	if len(policy.Permissions) == 0 {
		return fmt.Errorf("application %q %s access policy must declare permissions", applicationID, name)
	}
	return validateIdentifiers(name+" access permission", applicationID, policy.Permissions)
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
	definition.Access.Entry.Permissions = append([]string{}, definition.Access.Entry.Permissions...)
	if definition.Access.Settings != nil {
		policy := *definition.Access.Settings
		policy.Permissions = append([]string{}, policy.Permissions...)
		definition.Access.Settings = &policy
	}
	if definition.Settings != nil {
		settings := *definition.Settings
		definition.Settings = &settings
	}
	if definition.External != nil {
		external := *definition.External
		definition.External = &external
	}
	return definition
}
