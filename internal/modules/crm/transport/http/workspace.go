package http

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

func (h *Handler) registerWorkspaceRoutes(r module.RouteRegistry) {
	c := r.With(usersapi.RequireAuthentication(h.a), organizationapi.RequireWorkspace(h.w))
	leadRead := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.lead.read"))
	leadRead.Get("/crm/leads", h.listLeads)
	leadRead.Get("/crm/leads/{id}", h.getLead)
	leadCreate := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.lead.create"))
	leadCreate.Post("/crm/leads", h.createLead)
	leadUpdate := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.lead.update"))
	leadUpdate.Patch("/crm/leads/{id}", h.updateLead)
	leadUpdate.Post("/crm/leads/{id}/contact", h.contactLead)
	leadUpdate.Post("/crm/leads/{id}/qualify", h.qualifyLead)
	leadUpdate.Post("/crm/leads/{id}/disqualify", h.disqualifyLead)
	leadUpdate.Post("/crm/leads/{id}/reopen", h.reopenLead)
	leadConvert := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.lead.convert"))
	leadConvert.Post("/crm/leads/{id}/convert", h.convertLead)
	oppRead := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.opportunity.read"))
	oppRead.Get("/crm/opportunities", h.listOpportunities)
	oppRead.Get("/crm/opportunities/{id}", h.getOpportunity)
	oppCreate := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.opportunity.create"))
	oppCreate.Post("/crm/opportunities", h.createOpportunity)
	oppUpdate := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.opportunity.update"))
	oppUpdate.Patch("/crm/opportunities/{id}", h.updateOpportunity)
	oppTransition := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.opportunity.transition"))
	oppTransition.Post("/crm/opportunities/{id}/move-stage", h.moveOpportunity)
	oppTransition.Post("/crm/opportunities/{id}/mark-won", h.markWon)
	oppTransition.Post("/crm/opportunities/{id}/mark-lost", h.markLost)
	oppTransition.Post("/crm/opportunities/{id}/reopen", h.reopenOpportunity)
}

func page(r *http.Request) (int, int) {
	p, _ := strconv.Atoi(r.URL.Query().Get("page"))
	l, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if p < 1 {
		p = 1
	}
	if l < 1 || l > 100 {
		l = 25
	}
	return p, l
}
func optionalID(w http.ResponseWriter, r *http.Request, name string) *uuid.UUID {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		httpserver.WriteError(w, r, 400, "VALIDATION_ERROR", "Invalid "+name)
		return nil
	}
	return &parsed
}
func routeID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	value, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpserver.WriteError(w, r, 400, "VALIDATION_ERROR", "Identifier is invalid")
		return uuid.Nil, false
	}
	return value, true
}
func listOut[T any](w http.ResponseWriter, items []T, page, limit, total int) {
	out(w, 200, map[string]any{"items": items, "page": page, "limit": limit, "total": total})
}

func (h *Handler) listLeads(w http.ResponseWriter, r *http.Request) {
	workspace, _ := ctx(r)
	p, l := page(r)
	owner := optionalID(w, r, "owner_id")
	if r.URL.Query().Get("owner_id") != "" && owner == nil {
		return
	}
	items, total, err := h.s.ListLeads(r.Context(), workspace, domain.LeadFilter{Search: r.URL.Query().Get("search"), Status: domain.LeadStatus(r.URL.Query().Get("status")), OwnerID: owner, Page: p, Limit: l, Sort: r.URL.Query().Get("sort"), Desc: r.URL.Query().Get("desc") == "true"})
	if err != nil {
		fail(w, r, err)
		return
	}
	listOut(w, items, p, l, total)
}
func (h *Handler) getLead(w http.ResponseWriter, r *http.Request) {
	workspace, _ := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	item, err := h.s.GetLead(r.Context(), workspace, identifier)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, item)
}
func (h *Handler) createLead(w http.ResponseWriter, r *http.Request) {
	var item domain.Lead
	if !decode(w, r, &item) {
		return
	}
	workspace, actor := ctx(r)
	item.WorkspaceID, item.CreatedBy, item.UpdatedBy = workspace, actor, actor
	created, err := h.s.CreateLead(r.Context(), item)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 201, created)
}
func (h *Handler) updateLead(w http.ResponseWriter, r *http.Request) {
	var item domain.Lead
	if !decode(w, r, &item) {
		return
	}
	workspace, actor := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	item.ID, item.WorkspaceID, item.UpdatedBy = identifier, workspace, actor
	updated, err := h.s.UpdateLead(r.Context(), item, item.Version)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, updated)
}
func (h *Handler) leadTransition(w http.ResponseWriter, r *http.Request, next domain.LeadStatus) {
	var body struct {
		Version int    `json:"version"`
		Reason  string `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	workspace, actor := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	item, err := h.s.TransitionLead(r.Context(), workspace, actor, identifier, next, body.Reason, body.Version)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, item)
}
func (h *Handler) contactLead(w http.ResponseWriter, r *http.Request) {
	h.leadTransition(w, r, domain.LeadContacted)
}
func (h *Handler) qualifyLead(w http.ResponseWriter, r *http.Request) {
	h.leadTransition(w, r, domain.LeadQualified)
}
func (h *Handler) disqualifyLead(w http.ResponseWriter, r *http.Request) {
	h.leadTransition(w, r, domain.LeadDisqualified)
}
func (h *Handler) reopenLead(w http.ResponseWriter, r *http.Request) {
	h.leadTransition(w, r, domain.LeadNew)
}
func (h *Handler) convertLead(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PipelineID      uuid.UUID  `json:"pipeline_id"`
		StageID         uuid.UUID  `json:"stage_id"`
		ContactID       *uuid.UUID `json:"contact_id"`
		CreateContact   bool       `json:"create_contact"`
		ExpectedRevenue string     `json:"expected_revenue"`
		Currency        string     `json:"currency"`
		Version         int        `json:"version"`
	}
	if !decode(w, r, &body) {
		return
	}
	workspace, actor := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	item, err := h.s.ConvertLead(r.Context(), workspace, actor, identifier, application.ConvertInput{PipelineID: body.PipelineID, StageID: body.StageID, ContactID: body.ContactID, CreateContact: body.CreateContact, ExpectedRevenue: body.ExpectedRevenue, Currency: body.Currency}, body.Version)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 201, item)
}
func (h *Handler) listOpportunities(w http.ResponseWriter, r *http.Request) {
	workspace, _ := ctx(r)
	p, l := page(r)
	pipeline, stage, owner := optionalID(w, r, "pipeline_id"), optionalID(w, r, "stage_id"), optionalID(w, r, "owner_id")
	if (r.URL.Query().Get("pipeline_id") != "" && pipeline == nil) || (r.URL.Query().Get("stage_id") != "" && stage == nil) || (r.URL.Query().Get("owner_id") != "" && owner == nil) {
		return
	}
	items, total, err := h.s.ListOpportunities(r.Context(), workspace, domain.OpportunityFilter{Search: r.URL.Query().Get("search"), PipelineID: pipeline, StageID: stage, OwnerID: owner, Outcome: domain.OpportunityOutcome(r.URL.Query().Get("outcome")), Page: p, Limit: l, Sort: r.URL.Query().Get("sort"), Desc: r.URL.Query().Get("desc") == "true"})
	if err != nil {
		fail(w, r, err)
		return
	}
	listOut(w, items, p, l, total)
}
func (h *Handler) getOpportunity(w http.ResponseWriter, r *http.Request) {
	workspace, _ := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	item, err := h.s.GetOpportunity(r.Context(), workspace, identifier)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, item)
}
func (h *Handler) createOpportunity(w http.ResponseWriter, r *http.Request) {
	var item domain.Opportunity
	if !decode(w, r, &item) {
		return
	}
	workspace, actor := ctx(r)
	item.WorkspaceID, item.CreatedBy, item.UpdatedBy = workspace, actor, actor
	created, err := h.s.CreateOpportunity(r.Context(), item)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 201, created)
}
func (h *Handler) updateOpportunity(w http.ResponseWriter, r *http.Request) {
	var item domain.Opportunity
	if !decode(w, r, &item) {
		return
	}
	workspace, actor := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	item.ID, item.WorkspaceID, item.UpdatedBy = identifier, workspace, actor
	updated, err := h.s.UpdateOpportunity(r.Context(), item, item.Version)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, updated)
}
func (h *Handler) moveOpportunity(w http.ResponseWriter, r *http.Request) {
	var body struct {
		PipelineID uuid.UUID `json:"pipeline_id"`
		StageID    uuid.UUID `json:"stage_id"`
		Version    int       `json:"version"`
	}
	if !decode(w, r, &body) {
		return
	}
	workspace, actor := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	item, err := h.s.MoveOpportunity(r.Context(), workspace, actor, identifier, body.PipelineID, body.StageID, body.Version)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, item)
}
func (h *Handler) markWon(w http.ResponseWriter, r *http.Request)  { h.closeOpportunity(w, r, true) }
func (h *Handler) markLost(w http.ResponseWriter, r *http.Request) { h.closeOpportunity(w, r, false) }
func (h *Handler) closeOpportunity(w http.ResponseWriter, r *http.Request, won bool) {
	var body struct {
		Version int    `json:"version"`
		Reason  string `json:"reason"`
	}
	if !decode(w, r, &body) {
		return
	}
	workspace, actor := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	item, err := h.s.CloseOpportunity(r.Context(), workspace, actor, identifier, won, body.Reason, body.Version)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, item)
}
func (h *Handler) reopenOpportunity(w http.ResponseWriter, r *http.Request) {
	var body struct {
		StageID uuid.UUID `json:"stage_id"`
		Version int       `json:"version"`
	}
	if !decode(w, r, &body) {
		return
	}
	workspace, actor := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	item, err := h.s.ReopenOpportunity(r.Context(), workspace, actor, identifier, body.StageID, body.Version)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, item)
}
