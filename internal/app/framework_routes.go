package app

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/entity"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/field"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type moduleMetadata struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Version      string   `json:"version"`
	Dependencies []string `json:"dependencies"`
}

type fieldMetadata struct {
	Name        string          `json:"name"`
	DisplayName string          `json:"display_name"`
	Type        field.Type      `json:"type"`
	Required    bool            `json:"required"`
	ReadOnly    bool            `json:"read_only"`
	Description string          `json:"description,omitempty"`
	Relation    *field.Relation `json:"relation,omitempty"`
}

type entityMetadata struct {
	Name         string          `json:"name"`
	DisplayName  string          `json:"display_name"`
	Module       string          `json:"module"`
	Fields       []fieldMetadata `json:"fields"`
	Capabilities []string        `json:"capabilities"`
}

type permissionMetadata struct {
	Name        string `json:"name"`
	Module      string `json:"module"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
}

func (a *App) registerFrameworkRoutes(router chi.Router) {
	router.Route("/framework", func(r chi.Router) {
		r.Get("/modules", a.frameworkModules)
		r.Get("/entities", a.frameworkEntities)
		r.Get("/entities/{entity}", a.frameworkEntity)
		r.Get("/permissions", a.frameworkPermissions)
	})
}

func (a *App) frameworkModules(w http.ResponseWriter, _ *http.Request) {
	snapshot := a.Metadata.Snapshot()
	result := make([]moduleMetadata, 0, len(snapshot.Modules))
	for _, item := range snapshot.Modules {
		result = append(result, moduleMetadata{Name: item.Name, DisplayName: item.DisplayName, Version: item.Version, Dependencies: append([]string{}, item.Dependencies...)})
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) frameworkEntities(w http.ResponseWriter, _ *http.Request) {
	snapshot := a.Metadata.Snapshot()
	result := make([]entityMetadata, 0, len(snapshot.Entities))
	for _, item := range snapshot.Entities {
		result = append(result, entityResponse(item))
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *App) frameworkEntity(w http.ResponseWriter, r *http.Request) {
	definition, ok := a.Metadata.Entity(chi.URLParam(r, "entity"))
	if !ok {
		httpserver.WriteError(w, r, http.StatusNotFound, "ENTITY_NOT_FOUND", "The requested framework entity was not found")
		return
	}
	writeJSON(w, http.StatusOK, entityResponse(definition))
}

func (a *App) frameworkPermissions(w http.ResponseWriter, _ *http.Request) {
	snapshot := a.Metadata.Snapshot()
	result := make([]permissionMetadata, 0, len(snapshot.Permissions))
	for _, item := range snapshot.Permissions {
		result = append(result, permissionMetadata{Name: item.Name, Module: item.Module, DisplayName: item.DisplayName, Description: item.Description})
	}
	writeJSON(w, http.StatusOK, result)
}

func entityResponse(item entity.Definition) entityMetadata {
	fields := make([]fieldMetadata, 0, len(item.Fields))
	for _, fieldDefinition := range item.Fields {
		fields = append(fields, fieldMetadata{Name: fieldDefinition.Name, DisplayName: fieldDefinition.DisplayName, Type: fieldDefinition.Type, Required: fieldDefinition.Required, ReadOnly: fieldDefinition.ReadOnly, Description: fieldDefinition.Description, Relation: fieldDefinition.Relation})
	}
	return entityMetadata{Name: item.Name, DisplayName: item.DisplayName, Module: item.Module, Fields: fields, Capabilities: append([]string{}, item.Capabilities...)}
}
