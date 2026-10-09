package http

import (
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core/application"
)

type Handler struct{ service *application.Service }

func NewHandler(service *application.Service) *Handler { return &Handler{service: service} }

func (h *Handler) RegisterRoutes(routes module.RouteRegistry) error {
	routes.Get("/framework/applications", h.applications)
	routes.Get("/framework/modules", h.modules)
	routes.Get("/framework/entities", h.entities)
	routes.Get("/framework/entities/{entity}", h.entity)
	routes.Get("/framework/permissions", h.permissions)
	return nil
}
