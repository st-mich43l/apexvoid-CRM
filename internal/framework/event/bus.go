package event

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Definition struct {
	Name        string
	Module      string
	Description string
}
type Registry struct{ definitions map[string]Definition }
type subscription struct {
	subscriber string
	handle     func(context.Context, any) error
}
type Bus struct {
	registry      *Registry
	subscriptions map[string][]subscription
}

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9_]*(?:\.[a-z][a-z0-9_]*)+$`)

func NewRegistry() *Registry { return &Registry{definitions: make(map[string]Definition)} }
func NewBus(registry *Registry) *Bus {
	return &Bus{registry: registry, subscriptions: make(map[string][]subscription)}
}
func (b *Bus) Register(definition Definition) error { return b.registry.Register(definition) }
func (r *Registry) Register(definition Definition) error {
	if !namePattern.MatchString(definition.Name) {
		return fmt.Errorf("event name %q must be lowercase and namespaced", definition.Name)
	}
	if strings.TrimSpace(definition.Module) == "" {
		return fmt.Errorf("event %q must have an owning module", definition.Name)
	}
	if _, exists := r.definitions[definition.Name]; exists {
		return fmt.Errorf("event %q is already registered", definition.Name)
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
func (r *Registry) Has(name string) bool { _, ok := r.definitions[name]; return ok }

func Subscribe[T any](bus *Bus, name, subscriber string, handler func(context.Context, T) error) error {
	if !bus.registry.Has(name) {
		return fmt.Errorf("event %q is not registered", name)
	}
	if strings.TrimSpace(subscriber) == "" {
		return fmt.Errorf("subscriber cannot be empty")
	}
	for _, item := range bus.subscriptions[name] {
		if item.subscriber == subscriber {
			return fmt.Errorf("subscriber %q is already registered for event %q", subscriber, name)
		}
	}
	bus.subscriptions[name] = append(bus.subscriptions[name], subscription{subscriber: subscriber, handle: func(ctx context.Context, payload any) error {
		typed, ok := payload.(T)
		if !ok {
			return fmt.Errorf("event %q payload type mismatch for subscriber %q", name, subscriber)
		}
		return handler(ctx, typed)
	}})
	return nil
}

func Publish[T any](bus *Bus, ctx context.Context, name string, payload T) error {
	if !bus.registry.Has(name) {
		return fmt.Errorf("event %q is not registered", name)
	}
	for _, subscriber := range bus.subscriptions[name] {
		if err := subscriber.handle(ctx, payload); err != nil {
			return fmt.Errorf("event %q subscriber %q: %w", name, subscriber.subscriber, err)
		}
	}
	return nil
}
