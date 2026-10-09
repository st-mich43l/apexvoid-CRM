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
	authorizer    usersapi.Authorizer
	issuer        *application.AssertionIssuer
}

func NewHandler(service *application.Service, authenticator usersapi.Authenticator, workspace organizationapi.WorkspaceResolver, authorizer usersapi.Authorizer, issuer *application.AssertionIssuer) *Handler {
	return &Handler{service: service, authenticator: authenticator, workspace: workspace, authorizer: authorizer, issuer: issuer}
}

func (h *Handler) RegisterRoutes(routes module.RouteRegistry) error {
	if h.authenticator == nil || h.workspace == nil {
		return fmt.Errorf("framework application discovery requires authentication and workspace context")
	}
	applications := routes.With(usersapi.RequireAuthentication(h.authenticator), organizationapi.RequireWorkspace(h.workspace))
	applications.Get("/framework/applications", h.applications)
	technical := routes.With(usersapi.RequireAuthentication(h.authenticator), usersapi.RequirePermission(h.authorizer, "core.framework.read"))
	technical.Get("/framework/modules", h.modules)
	technical.Get("/framework/entities", h.entities)
	technical.Get("/framework/entities/{entity}", h.entity)
	technical.Get("/framework/permissions", h.permissions)
	managed := routes.With(usersapi.RequireAuthentication(h.authenticator), usersapi.RequirePermission(h.authorizer, "core.application.manage"))
	managed.Get("/applications/external", h.listExternal)
	managed.Post("/applications/external", h.registerExternal)
	managed.Post("/applications/external/discover", h.discoverExternal)
	managed.Get("/applications/external/installations", h.listExternalInstallations)
	managed.Get("/applications/external/installations/{installation}", h.getExternalInstallation)
	managed.Post("/applications/external/installations/{installation}/approve", h.approveExternalInstallation)
	managed.Post("/applications/external/{application}/updates/check", h.checkExternalUpdate)
	managed.Get("/applications/external/{application}", h.getExternal)
	managed.Patch("/applications/external/{application}", h.updateExternal)
	managed.Delete("/applications/external/{application}", h.unregisterExternal)
	managed.Put("/applications/external/{application}/workspaces/{workspace}", h.setWorkspaceAvailability)
	managed.Post("/applications/external/{application}/credentials/revoke", h.revokeCredential)
	managed.Post("/applications/external/{application}/credentials/rotate", h.rotateCredential)
	managed.Get("/applications/external/{application}/status", h.externalStatus)
	routes.Post("/integrations/v1/session/introspect", h.introspectSession)
	routes.Get("/integrations/v1/applications/{application}/availability", h.integrationAvailability)
	return nil
}

func (h *Handler) RegisterPublicRoutes(routes module.RouteRegistry) error {
	protected := routes.With(usersapi.RequireAuthentication(h.authenticator), organizationapi.RequireWorkspace(h.workspace))
	protected.Get("/apps/{application}", h.proxyExternalFrontend)
	protected.Get("/apps/{application}/*", h.proxyExternalFrontend)
	protected.Get("/api/apps/{application}", h.proxyExternalAPI)
	protected.Get("/api/apps/{application}/*", h.proxyExternalAPI)
	protected.Post("/api/apps/{application}", h.proxyExternalAPI)
	protected.Post("/api/apps/{application}/*", h.proxyExternalAPI)
	protected.Put("/api/apps/{application}", h.proxyExternalAPI)
	protected.Put("/api/apps/{application}/*", h.proxyExternalAPI)
	protected.Patch("/api/apps/{application}", h.proxyExternalAPI)
	protected.Patch("/api/apps/{application}/*", h.proxyExternalAPI)
	protected.Delete("/api/apps/{application}", h.proxyExternalAPI)
	protected.Delete("/api/apps/{application}/*", h.proxyExternalAPI)
	return nil
}
