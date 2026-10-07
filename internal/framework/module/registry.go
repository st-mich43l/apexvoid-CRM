package module

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/capability"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/event"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/extension"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$`)

type Descriptor struct {
	Name         string
	DisplayName  string
	Version      string
	Dependencies []string
}

type Module interface {
	Descriptor() Descriptor
	Register(*Context) error
}

type Context struct {
	Entities     *entity.Registry
	Permissions  *permission.Registry
	Capabilities *capability.Registry
	Events       *event.Bus
	Extensions   *extension.Registry
}

type Registry struct {
	modules map[string]Module
}

func NewRegistry() *Registry { return &Registry{modules: make(map[string]Module)} }

func (r *Registry) Register(m Module) error {
	if m == nil {
		return fmt.Errorf("module cannot be nil")
	}
	descriptor := m.Descriptor()
	if err := ValidateName(descriptor.Name); err != nil {
		return fmt.Errorf("invalid module name: %w", err)
	}
	if strings.TrimSpace(descriptor.DisplayName) == "" {
		return fmt.Errorf("module %q display name cannot be empty", descriptor.Name)
	}
	if strings.TrimSpace(descriptor.Version) == "" {
		return fmt.Errorf("module %q version cannot be empty", descriptor.Name)
	}
	if _, exists := r.modules[descriptor.Name]; exists {
		return fmt.Errorf("module %q is already registered", descriptor.Name)
	}
	seen := make(map[string]struct{}, len(descriptor.Dependencies))
	for _, dependency := range descriptor.Dependencies {
		if err := ValidateName(dependency); err != nil {
			return fmt.Errorf("module %q has invalid dependency %q: %w", descriptor.Name, dependency, err)
		}
		if dependency == descriptor.Name {
			return fmt.Errorf("module %q cannot depend on itself", descriptor.Name)
		}
		if _, exists := seen[dependency]; exists {
			return fmt.Errorf("module %q declares duplicate dependency %q", descriptor.Name, dependency)
		}
		seen[dependency] = struct{}{}
	}
	r.modules[descriptor.Name] = m
	return nil
}

func (r *Registry) Resolve() ([]Module, error) {
	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}
	sort.Strings(names)
	state := make(map[string]uint8, len(names))
	order := make([]Module, 0, len(names))
	stack := []string{}
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case 1:
			cycleStart := 0
			for i, item := range stack {
				if item == name {
					cycleStart = i
					break
				}
			}
			cycle := append(append([]string{}, stack[cycleStart:]...), name)
			return fmt.Errorf("module dependency cycle: %s", strings.Join(cycle, " -> "))
		case 2:
			return nil
		}
		m, exists := r.modules[name]
		if !exists {
			return fmt.Errorf("module dependency missing: module %q requires missing dependency %q", stack[len(stack)-1], name)
		}
		state[name] = 1
		stack = append(stack, name)
		dependencies := append([]string{}, m.Descriptor().Dependencies...)
		sort.Strings(dependencies)
		for _, dependency := range dependencies {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		stack = stack[:len(stack)-1]
		state[name] = 2
		order = append(order, m)
		return nil
	}
	for _, name := range names {
		if err := visit(name); err != nil {
			return nil, err
		}
	}
	return order, nil
}

func (r *Registry) Initialize(ctx context.Context, services *Context) ([]Module, error) {
	order, err := r.Resolve()
	if err != nil {
		return nil, err
	}
	for _, m := range order {
		if err := m.Register(services); err != nil {
			return nil, fmt.Errorf("register module %q: %w", m.Descriptor().Name, err)
		}
	}
	return order, nil
}

func (r *Registry) Descriptors() []Descriptor {
	order, err := r.Resolve()
	if err != nil {
		return nil
	}
	result := make([]Descriptor, 0, len(order))
	for _, m := range order {
		result = append(result, m.Descriptor())
	}
	return result
}

func ValidateName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("%q must be lowercase and contain only letters, numbers, '.', '_' or '-'", name)
	}
	return nil
}
