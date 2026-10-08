package metadata

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

type ModuleMetadata struct {
	Name         string
	DisplayName  string
	Version      string
	Dependencies []string
}

type PermissionMetadata struct {
	Name        string
	Module      string
	Scope       permission.Scope
	DisplayName string
	Description string
}

func ModuleDescriptors(descriptors []module.Descriptor) []ModuleMetadata {
	result := make([]ModuleMetadata, 0, len(descriptors))
	for _, item := range descriptors {
		result = append(result, ModuleMetadata{Name: item.Name, DisplayName: item.DisplayName, Version: item.Version, Dependencies: append([]string{}, item.Dependencies...)})
	}
	return result
}
