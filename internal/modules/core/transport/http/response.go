package http

import (
	"encoding/json"
	"net/http"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
)

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
