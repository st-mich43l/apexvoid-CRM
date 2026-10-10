package http

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	coreapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/api"
)

type applicationResponse struct {
	ID                   string            `json:"id"`
	DisplayName          string            `json:"display_name"`
	Description          string            `json:"description"`
	Version              string            `json:"version"`
	APIContractVersion   string            `json:"api_contract_version"`
	ModuleDependencies   []string          `json:"module_dependencies"`
	RequiredPermissions  []string          `json:"required_permissions"`
	RequiredCapabilities []string          `json:"required_capabilities"`
	Frontend             frontendResponse  `json:"frontend"`
	Settings             *settingsResponse `json:"settings,omitempty"`
	EntryAuthorized      bool              `json:"entry_authorized"`
	SettingsAuthorized   bool              `json:"settings_authorized"`
	Deployment           string            `json:"deployment"`
	FrontendExternal     bool              `json:"frontend_external"`
	UpdateAvailable      bool              `json:"update_available,omitempty"`
	AvailableVersion     string            `json:"available_version,omitempty"`
	UpdateCheckedAt      *time.Time        `json:"update_checked_at,omitempty"`
	UpdateCheckError     string            `json:"update_check_error,omitempty"`
}

type frontendResponse struct {
	EntryRoute   string `json:"entry_route"`
	NavigationID string `json:"navigation_id"`
}

type settingsResponse struct {
	Route string `json:"route"`
}

type moduleResponse struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Version      string   `json:"version"`
	Dependencies []string `json:"dependencies"`
}
type permissionResponse struct {
	Name        string `json:"name"`
	Module      string `json:"module"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
}

type fieldResponse struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	Type        field.Type      `json:"type"`
	Required    bool            `json:"required"`
	ReadOnly    bool            `json:"read_only"`
	Description string          `json:"description,omitempty"`
	Relation    *field.Relation `json:"relation,omitempty"`
}

type entityResponse struct {
	Name         string          `json:"name"`
	DisplayName  string          `json:"display_name"`
	Module       string          `json:"module"`
	Fields       []fieldResponse `json:"fields"`
	Capabilities []string        `json:"capabilities"`
}

func toModuleResponse(item metadata.ModuleMetadata) moduleResponse {
	return moduleResponse{Name: item.Name, DisplayName: item.DisplayName, Version: item.Version, Dependencies: append([]string{}, item.Dependencies...)}
}
func toApplicationResponse(item coreapi.DiscoveredApplication) applicationResponse {
	definition := item.Descriptor
	result := applicationResponse{ID: definition.ID, DisplayName: definition.DisplayName, Description: definition.Description, Version: definition.Version, APIContractVersion: definition.APIContractVersion, ModuleDependencies: append([]string{}, definition.ModuleDependencies...), RequiredPermissions: append([]string{}, definition.RequiredPermissions...), RequiredCapabilities: append([]string{}, definition.RequiredCapabilities...), Frontend: frontendResponse{EntryRoute: definition.Frontend.EntryRoute, NavigationID: definition.Frontend.NavigationID}, EntryAuthorized: item.EntryAuthorized, SettingsAuthorized: item.SettingsAuthorized, Deployment: string(definition.Deployment), FrontendExternal: definition.Deployment == "external" && definition.Frontend.EntryRoute != "", UpdateAvailable: item.UpdateAvailable, AvailableVersion: item.AvailableVersion, UpdateCheckedAt: item.UpdateCheckedAt, UpdateCheckError: item.UpdateCheckError}
	if definition.Settings != nil {
		result.Settings = &settingsResponse{Route: definition.Settings.Route}
	}
	return result
}
func toPermissionResponse(item metadata.PermissionMetadata) permissionResponse {
	return permissionResponse{Name: item.Name, Module: item.Module, DisplayName: item.DisplayName, Description: item.Description}
}

func toEntityResponse(item entity.Definition) entityResponse {
	fields := make([]fieldResponse, 0, len(item.Fields))
	for _, fieldDefinition := range item.Fields {
		fields = append(fields, fieldResponse{Name: fieldDefinition.Name, DisplayName: fieldDefinition.DisplayName, Type: fieldDefinition.Type, Required: fieldDefinition.Required, ReadOnly: fieldDefinition.ReadOnly, Description: fieldDefinition.Description, Relation: fieldDefinition.Relation})
	}
	return entityResponse{Name: item.Name, DisplayName: item.DisplayName, Module: item.Module, Fields: fields, Capabilities: append([]string{}, item.Capabilities...)}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
