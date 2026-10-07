package module

import (
	"fmt"
	"sort"
)

type Migration struct {
	Module  string
	Version uint64
	Name    string
	UpSQL   string
	DownSQL string
}
type MigrationProvider interface{ Migrations() []Migration }

func (r *Registry) Migrations() ([]Migration, error) {
	order, err := r.Resolve()
	if err != nil {
		return nil, err
	}
	result := []Migration{}
	seen := map[string]struct{}{}
	for _, item := range order {
		provider, ok := item.(MigrationProvider)
		if !ok {
			continue
		}
		for _, migration := range provider.Migrations() {
			if migration.Module != item.Descriptor().Name {
				return nil, fmt.Errorf("migration %q is owned by %q but registered by module %q", migration.Name, migration.Module, item.Descriptor().Name)
			}
			if migration.Version == 0 || migration.Name == "" {
				return nil, fmt.Errorf("module %q has invalid migration metadata", migration.Module)
			}
			key := fmt.Sprintf("%s:%d", migration.Module, migration.Version)
			if _, exists := seen[key]; exists {
				return nil, fmt.Errorf("duplicate migration %s", key)
			}
			seen[key] = struct{}{}
			result = append(result, migration)
		}
	}
	moduleOrder := map[string]int{}
	for i, item := range order {
		moduleOrder[item.Descriptor().Name] = i
	}
	sort.SliceStable(result, func(i, j int) bool {
		if moduleOrder[result[i].Module] != moduleOrder[result[j].Module] {
			return moduleOrder[result[i].Module] < moduleOrder[result[j].Module]
		}
		return result[i].Version < result[j].Version
	})
	return result, nil
}
