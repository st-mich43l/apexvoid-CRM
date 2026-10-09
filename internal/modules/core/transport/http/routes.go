package http

import (
	"fmt"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core/application"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
)

type Handler struct {
	service       *application.Service
	authenticator usersapi.Authenticator
	workspace     organizationapi.WorkspaceResolver
}

func NewHandler(service *application.Service, authenticator usersapi.Authenticator, workspace organizationapi.WorkspaceResolver) *Handler {
	return &Handler{service: service, authenticator: authenticator, workspace: workspace}
}

func (h *Handler) RegisterRoutes(routes module.RouteRegistry) error {
	if h.authenticator == nil || h.workspace == nil {
		return fmt.Errorf("framework application discovery requires authentication and workspace context")
	}
	applications := routes.With(usersapi.RequireAuthentication(h.authenticator), organizationapi.RequireWorkspace(h.workspace))
	applications.Get("/framework/applications", h.applications)
	routes.Get("/framework/modules", h.modules)
	routes.Get("/framework/entities", h.entities)
	routes.Get("/framework/entities/{entity}", h.entity)
	routes.Get("/framework/permissions", h.permissions)
	return nil
}
