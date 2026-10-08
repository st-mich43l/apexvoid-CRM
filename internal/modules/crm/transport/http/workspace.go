package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/domain"
	organizationapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/api"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

type leadCreateRequest struct {
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	ContactName    string         `json:"contact_name"`
	CompanyName    string         `json:"company_name"`
	Email          string         `json:"email"`
	Phone          string         `json:"phone"`
	Source         string         `json:"source"`
	AssignedUserID *uuid.UUID     `json:"assigned_user_id"`
	ContactID      *uuid.UUID     `json:"contact_id"`
	CustomValues   map[string]any `json:"custom_values"`
}
type leadPatchRequest struct {
	Version        int             `json:"version"`
	Title          *string         `json:"title"`
	Description    *string         `json:"description"`
	ContactName    *string         `json:"contact_name"`
	CompanyName    *string         `json:"company_name"`
	Email          *string         `json:"email"`
	Phone          *string         `json:"phone"`
	Source         *string         `json:"source"`
	AssignedUserID json.RawMessage `json:"assigned_user_id"`
	ContactID      json.RawMessage `json:"contact_id"`
	CustomValues   json.RawMessage `json:"custom_values"`
}
type opportunityCreateRequest struct {
	Title             string         `json:"title"`
	Description       string         `json:"description"`
	PipelineID        uuid.UUID      `json:"pipeline_id"`
	StageID           uuid.UUID      `json:"stage_id"`
	ContactID         *uuid.UUID     `json:"contact_id"`
	CompanyID         *uuid.UUID     `json:"company_id"`
	AssignedUserID    *uuid.UUID     `json:"assigned_user_id"`
	ExpectedRevenue   string         `json:"expected_revenue"`
	Currency          string         `json:"currency"`
	ExpectedCloseDate *dateOnly      `json:"expected_close_date"`
	CustomValues      map[string]any `json:"custom_values"`
}

// dateOnly is the transport contract for business dates. It deliberately does
// not accept timestamps, which avoids browser timezone shifts at the API edge.
type dateOnly struct{ time.Time }

func (d *dateOnly) UnmarshalJSON(raw []byte) error {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return err
	}
	d.Time = parsed
	return nil
}

func dateOnlyValue(value *dateOnly) *time.Time {
	if value == nil {
		return nil
	}
	return &value.Time
}

type opportunityPatchRequest struct {
	Version           int             `json:"version"`
	Title             *string         `json:"title"`
	Description       *string         `json:"description"`
	ExpectedRevenue   *string         `json:"expected_revenue"`
	Currency          *string         `json:"currency"`
	ExpectedCloseDate json.RawMessage `json:"expected_close_date"`
	AssignedUserID    json.RawMessage `json:"assigned_user_id"`
	ContactID         json.RawMessage `json:"contact_id"`
	CompanyID         json.RawMessage `json:"company_id"`
	CustomValues      json.RawMessage `json:"custom_values"`
}

func (h *Handler) registerWorkspaceRoutes(r module.RouteRegistry) {
	c := r.With(usersapi.RequireAuthentication(h.a), organizationapi.RequireWorkspace(h.w))
	leadRead := c.With(organizationapi.RequireWorkspacePermission(h.x, "crm.lead.read"))
	leadRead.Get("/crm/leads", h.listLeads)
	leadRead.Get("/crm/leads/{id}", h.getLead)
	leadRead.Get("/crm/leads/{id}/history", h.leadHistory)
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
	oppRead.Get("/crm/opportunities/{id}/history", h.opportunityHistory)
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
func opportunityOutput(item domain.Opportunity) map[string]any {
	encoded, _ := json.Marshal(item)
	result := map[string]any{}
	_ = json.Unmarshal(encoded, &result)
	if item.ExpectedCloseDate == nil {
		result["expected_close_date"] = nil
	} else {
		result["expected_close_date"] = item.ExpectedCloseDate.Format("2006-01-02")
	}
	return result
}
func opportunityListOutput(items []domain.Opportunity) []map[string]any {
	result := make([]map[string]any, 0, len(items))
	for _, item := range items {
		result = append(result, opportunityOutput(item))
	}
	return result
}
func writeOpportunity(w http.ResponseWriter, status int, item domain.Opportunity) {
	out(w, status, opportunityOutput(item))
}
func nullableUUID(raw json.RawMessage) (*uuid.UUID, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil
	}
	if string(raw) == "null" {
		return nil, true, nil
	}
	var value uuid.UUID
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, true, err
	}
	return &value, true, nil
}
func nullableDate(raw json.RawMessage) (*time.Time, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil
	}
	if string(raw) == "null" {
		return nil, true, nil
	}
	var value dateOnly
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, true, err
	}
	return &value.Time, true, nil
}
func patchValues(raw json.RawMessage) (map[string]any, bool, error) {
	if len(raw) == 0 {
		return nil, false, nil
	}
	if string(raw) == "null" {
		return map[string]any{}, true, nil
	}
	var values map[string]any
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, true, err
	}
	return values, true, nil
}
func patchError(w http.ResponseWriter, r *http.Request) {
	httpserver.WriteError(w, r, 400, "VALIDATION_ERROR", "Patch value is invalid")
}

func (h *Handler) listLeads(w http.ResponseWriter, r *http.Request) {
	workspace, actor := ctx(r)
	p, l := page(r)
	viewID := optionalID(w, r, "view_id")
	if r.URL.Query().Get("view_id") != "" && viewID == nil { return }
	owner := optionalID(w, r, "owner_id")
	if r.URL.Query().Get("owner_id") != "" && owner == nil {
		return
	}
	items, total, err := h.s.ListLeads(r.Context(), workspace, domain.LeadFilter{Search: r.URL.Query().Get("search"), ViewID: viewID, ViewUserID: actor, Status: domain.LeadStatus(r.URL.Query().Get("status")), OwnerID: owner, Page: p, Limit: l, Sort: r.URL.Query().Get("sort"), Desc: r.URL.Query().Get("desc") == "true"})
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
func (h *Handler) leadHistory(w http.ResponseWriter, r *http.Request) {
	workspace, _ := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	items, err := h.s.LeadHistory(r.Context(), workspace, identifier)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, items)
}
func (h *Handler) createLead(w http.ResponseWriter, r *http.Request) {
	var body leadCreateRequest
	if !decode(w, r, &body) {
		return
	}
	workspace, actor := ctx(r)
	item := domain.Lead{WorkspaceID: workspace, Title: body.Title, Description: body.Description, ContactName: body.ContactName, CompanyName: body.CompanyName, Email: body.Email, Phone: body.Phone, Source: body.Source, AssignedUserID: body.AssignedUserID, ContactID: body.ContactID, CustomValues: body.CustomValues, CreatedBy: actor, UpdatedBy: actor}
	created, err := h.s.CreateLead(r.Context(), item)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 201, created)
}
func (h *Handler) updateLead(w http.ResponseWriter, r *http.Request) {
	var body leadPatchRequest
	if !decode(w, r, &body) {
		return
	}
	workspace, actor := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	assigned, setAssigned, err := nullableUUID(body.AssignedUserID)
	if err != nil {
		patchError(w, r)
		return
	}
	contact, setContact, err := nullableUUID(body.ContactID)
	if err != nil {
		patchError(w, r)
		return
	}
	custom, setCustom, err := patchValues(body.CustomValues)
	if err != nil {
		patchError(w, r)
		return
	}
	updated, err := h.s.PatchLead(r.Context(), workspace, actor, identifier, body.Version, application.LeadPatch{Title: body.Title, Description: body.Description, ContactName: body.ContactName, CompanyName: body.CompanyName, Email: body.Email, Phone: body.Phone, Source: body.Source, AssignedUserID: assigned, ContactID: contact, SetAssignedUser: setAssigned, SetContact: setContact, CustomValues: custom, SetCustomValues: setCustom})
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
	writeOpportunity(w, 201, item)
}
func (h *Handler) listOpportunities(w http.ResponseWriter, r *http.Request) {
	workspace, actor := ctx(r)
	p, l := page(r)
	viewID := optionalID(w, r, "view_id")
	if r.URL.Query().Get("view_id") != "" && viewID == nil { return }
	pipeline, stage, owner := optionalID(w, r, "pipeline_id"), optionalID(w, r, "stage_id"), optionalID(w, r, "owner_id")
	if (r.URL.Query().Get("pipeline_id") != "" && pipeline == nil) || (r.URL.Query().Get("stage_id") != "" && stage == nil) || (r.URL.Query().Get("owner_id") != "" && owner == nil) {
		return
	}
	items, total, err := h.s.ListOpportunities(r.Context(), workspace, domain.OpportunityFilter{Search: r.URL.Query().Get("search"), ViewID: viewID, ViewUserID: actor, PipelineID: pipeline, StageID: stage, OwnerID: owner, Outcome: domain.OpportunityOutcome(r.URL.Query().Get("outcome")), Page: p, Limit: l, Sort: r.URL.Query().Get("sort"), Desc: r.URL.Query().Get("desc") == "true"})
	if err != nil {
		fail(w, r, err)
		return
	}
	listOut(w, opportunityListOutput(items), p, l, total)
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
	writeOpportunity(w, 200, *item)
}
func (h *Handler) opportunityHistory(w http.ResponseWriter, r *http.Request) {
	workspace, _ := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	items, err := h.s.OpportunityHistory(r.Context(), workspace, identifier)
	if err != nil {
		fail(w, r, err)
		return
	}
	out(w, 200, items)
}
func (h *Handler) createOpportunity(w http.ResponseWriter, r *http.Request) {
	var body opportunityCreateRequest
	if !decode(w, r, &body) {
		return
	}
	workspace, actor := ctx(r)
	item := domain.Opportunity{WorkspaceID: workspace, Title: body.Title, Description: body.Description, PipelineID: body.PipelineID, StageID: body.StageID, ContactID: body.ContactID, CompanyID: body.CompanyID, AssignedUserID: body.AssignedUserID, ExpectedRevenue: body.ExpectedRevenue, Currency: body.Currency, ExpectedCloseDate: dateOnlyValue(body.ExpectedCloseDate), CustomValues: body.CustomValues, CreatedBy: actor, UpdatedBy: actor}
	created, err := h.s.CreateOpportunity(r.Context(), item)
	if err != nil {
		fail(w, r, err)
		return
	}
	writeOpportunity(w, 201, created)
}
func (h *Handler) updateOpportunity(w http.ResponseWriter, r *http.Request) {
	var body opportunityPatchRequest
	if !decode(w, r, &body) {
		return
	}
	workspace, actor := ctx(r)
	identifier, ok := routeID(w, r)
	if !ok {
		return
	}
	closeDate, setCloseDate, err := nullableDate(body.ExpectedCloseDate)
	if err != nil {
		patchError(w, r)
		return
	}
	assigned, setAssigned, err := nullableUUID(body.AssignedUserID)
	if err != nil {
		patchError(w, r)
		return
	}
	contact, setContact, err := nullableUUID(body.ContactID)
	if err != nil {
		patchError(w, r)
		return
	}
	company, setCompany, err := nullableUUID(body.CompanyID)
	if err != nil {
		patchError(w, r)
		return
	}
	custom, setCustom, err := patchValues(body.CustomValues)
	if err != nil {
		patchError(w, r)
		return
	}
	updated, err := h.s.PatchOpportunity(r.Context(), workspace, actor, identifier, body.Version, application.OpportunityPatch{Title: body.Title, Description: body.Description, ExpectedRevenue: body.ExpectedRevenue, Currency: body.Currency, ExpectedCloseDate: closeDate, SetExpectedCloseDate: setCloseDate, AssignedUserID: assigned, SetAssignedUser: setAssigned, ContactID: contact, SetContact: setContact, CompanyID: company, SetCompany: setCompany, CustomValues: custom, SetCustomValues: setCustom})
	if err != nil {
		fail(w, r, err)
		return
	}
	writeOpportunity(w, 200, updated)
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
	writeOpportunity(w, 200, item)
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
	writeOpportunity(w, 200, item)
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
	writeOpportunity(w, 200, item)
}
