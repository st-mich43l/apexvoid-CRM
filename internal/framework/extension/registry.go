package extension

import (
	"fmt"
	"sort"
	"strings"
)

type Point struct {
	Name        string
	Module      string
	DisplayName string
	Description string
}
type Implementation struct {
	Name   string
	Module string
	Value  any
}
type Registry struct {
	points          map[string]Point
	implementations map[string][]Implementation
}

func NewRegistry() *Registry {
	return &Registry{points: make(map[string]Point), implementations: make(map[string][]Implementation)}
}
func (r *Registry) RegisterPoint(point Point) error {
	if strings.TrimSpace(point.Name) == "" || strings.TrimSpace(point.Module) == "" || strings.TrimSpace(point.DisplayName) == "" {
		return fmt.Errorf("extension point requires name, module, and display name")
	}
	if _, exists := r.points[point.Name]; exists {
		return fmt.Errorf("extension point %q is already registered", point.Name)
	}
	r.points[point.Name] = point
	return nil
}
func (r *Registry) Register(pointName string, implementation Implementation) error {
	if _, exists := r.points[pointName]; !exists {
		return fmt.Errorf("extension point %q is not registered", pointName)
	}
	if strings.TrimSpace(implementation.Name) == "" || strings.TrimSpace(implementation.Module) == "" {
		return fmt.Errorf("extension implementation requires name and module")
	}
	for _, item := range r.implementations[pointName] {
		if item.Name == implementation.Name {
			return fmt.Errorf("extension %q is already registered for point %q", implementation.Name, pointName)
		}
	}
	r.implementations[pointName] = append(r.implementations[pointName], implementation)
	return nil
}
func (r *Registry) Points() []Point {
	result := make([]Point, 0, len(r.points))
	for _, point := range r.points {
		result = append(result, point)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
func (r *Registry) Implementations(pointName string) []Implementation {
	result := append([]Implementation{}, r.implementations[pointName]...)
	sort.SliceStable(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
