//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/st-mich43l/apexvoid-CRM/internal/app"
	crmmigrations "github.com/st-mich43l/apexvoid-CRM/internal/modules/crm/migrations"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

type crmPipeline struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Status      string `json:"status"`
	Version     int    `json:"version"`
}

func TestCRMMigrationRollbackAndReapply(t *testing.T) {
	url, cleanup := isolatedDatabaseURL(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	application, err := app.Bootstrap(ctx, config.Config{App: config.AppConfig{Name: "crm-migration", Environment: "test"}, Server: config.ServerConfig{Address: ":0"}, Database: config.DatabaseConfig{URL: url, MaxConns: 10, MinConns: 1}, Auth: config.AuthConfig{AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour, CookieSameSite: "lax", PasswordMinLen: 12, PasswordMaxLen: 128}, Logging: config.LoggingConfig{Level: "ERROR"}})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close(context.Background())
	runner := database.NewMigrationRunner(application.Database, crmmigrations.All())
	if err := runner.Down(ctx); err != nil {
		t.Fatalf("rollback lifecycle history: %v", err)
	}
	if err := runner.Down(ctx); err != nil {
		t.Fatalf("rollback leads/opportunities: %v", err)
	}
	var leadTable *string
	if err := application.Database.QueryRow(ctx, "SELECT to_regclass(current_schema() || '.crm_leads')").Scan(&leadTable); err != nil {
		t.Fatal(err)
	}
	if leadTable != nil {
		t.Fatalf("crm_leads remained after rollback: %s", *leadTable)
	}
	if err := runner.Up(ctx); err != nil {
		t.Fatalf("reapply CRM migrations: %v", err)
	}
	if err := application.Database.QueryRow(ctx, "SELECT to_regclass(current_schema() || '.crm_lifecycle_history')").Scan(&leadTable); err != nil {
		t.Fatal(err)
	}
	if leadTable == nil {
		t.Fatal("crm_lifecycle_history was not restored")
	}
}

type crmStage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
	Category    string `json:"category"`
	Probability int    `json:"probability"`
	Version     int    `json:"version"`
}
type crmLead struct {
	ID                     string         `json:"id"`
	Status                 string         `json:"status"`
	Version                int            `json:"version"`
	Source                 string         `json:"source"`
	ConvertedOpportunityID string         `json:"converted_opportunity_id"`
	ConvertedAt            *time.Time     `json:"converted_at"`
	CustomValues           map[string]any `json:"custom_values"`
}
type crmOpportunity struct {
	ID                string         `json:"id"`
	Version           int            `json:"version"`
	PipelineID        string         `json:"pipeline_id"`
	OriginalLeadID    string         `json:"original_lead_id"`
	ContactID         string         `json:"contact_id"`
	Description       string         `json:"description"`
	ExpectedCloseDate *string        `json:"expected_close_date"`
	CustomValues      map[string]any `json:"custom_values"`
}

func TestCRMLeadConversionAndSparsePatch(t *testing.T) {
	url, cleanup := isolatedDatabaseURL(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	application, err := app.Bootstrap(ctx, config.Config{App: config.AppConfig{Name: "crm-test", Environment: "test"}, Server: config.ServerConfig{Address: ":0"}, Database: config.DatabaseConfig{URL: url, MaxConns: 10, MinConns: 1}, Auth: config.AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour, CookieSameSite: "lax", PasswordMinLen: 12, PasswordMaxLen: 128}, Bootstrap: config.BootstrapConfig{AdminEmail: "admin@localhost", AdminUsername: "admin", AdminPassword: "admin"}, Logging: config.LoggingConfig{Level: "ERROR"}})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close(context.Background())
	router := chi.NewRouter()
	if err := application.RegisterRoutes(router); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	admin := newAPIClient(t, server.URL)
	admin.must("POST", "/api/v1/auth/login", "", map[string]string{"email": "admin@localhost", "password": "admin"}, nil, http.StatusOK)
	admin.must("POST", "/api/v1/auth/change-password", "", map[string]string{"current_password": "admin", "new_password": "admin-password-123"}, nil, http.StatusOK)
	var setup setupResponse
	admin.must("POST", "/api/v1/setup/organization", "", map[string]string{"organization_name": "CRM", "workspace_name": "Primary", "timezone": "UTC"}, &setup, http.StatusCreated)
	workspaceID := setup.Workspace.ID
	var pipeline crmPipeline
	admin.must("POST", "/api/v1/crm/pipelines/initialize", workspaceID, map[string]string{"template": "standard_b2b"}, &pipeline, http.StatusCreated)
	admin.must("PATCH", "/api/v1/crm/pipelines/"+pipeline.ID, workspaceID, map[string]any{"name": "Revenue pipeline", "description": "Configured in CRM", "color": "#4f46e5", "version": pipeline.Version}, &pipeline, http.StatusOK)
	if pipeline.Name != "Revenue pipeline" || pipeline.Description != "Configured in CRM" || pipeline.Color != "#4f46e5" {
		t.Fatalf("pipeline edit did not persist: %#v", pipeline)
	}
	var stages []crmStage
	admin.must("GET", "/api/v1/crm/pipelines/"+pipeline.ID+"/stages", workspaceID, nil, &stages, http.StatusOK)
	var openStage string
	for _, stage := range stages {
		if stage.Category == "open" {
			openStage = stage.ID
			break
		}
	}
	if openStage == "" {
		t.Fatal("pipeline did not create an open stage")
	}
	var blankPipeline crmPipeline
	admin.must("POST", "/api/v1/crm/pipelines/initialize", workspaceID, map[string]string{"template": "blank"}, &blankPipeline, http.StatusCreated)
	var blankStages []crmStage
	admin.must("GET", "/api/v1/crm/pipelines/"+blankPipeline.ID+"/stages", workspaceID, nil, &blankStages, http.StatusOK)
	for _, stage := range blankStages {
		if stage.Category == "open" {
			admin.must("POST", "/api/v1/crm/pipelines/"+blankPipeline.ID+"/stages/"+stage.ID+"/archive", workspaceID, nil, nil, http.StatusBadRequest)
		}
	}
	var addedStage crmStage
	admin.must("POST", "/api/v1/crm/pipelines/"+pipeline.ID+"/stages", workspaceID, map[string]any{"key": "discovery", "name": "Discovery", "probability": 20}, &addedStage, http.StatusCreated)
	admin.must("PATCH", "/api/v1/crm/pipelines/"+pipeline.ID+"/stages/"+addedStage.ID, workspaceID, map[string]any{"name": "Discovery complete", "description": "A qualified discovery", "color": "#0ea5e9", "probability": 30, "version": addedStage.Version}, &addedStage, http.StatusOK)
	if addedStage.Name != "Discovery complete" || addedStage.Probability != 30 {
		t.Fatalf("stage edit did not persist: %#v", addedStage)
	}
	var clonedPipeline crmPipeline
	admin.must("POST", "/api/v1/crm/pipelines/"+pipeline.ID+"/clone", workspaceID, map[string]string{"name": "Revenue pipeline copy"}, &clonedPipeline, http.StatusCreated)
	admin.must("POST", "/api/v1/crm/pipelines/"+clonedPipeline.ID+"/archive", workspaceID, map[string]int{"version": clonedPipeline.Version}, nil, http.StatusNoContent)
	var contact contactResponse
	admin.must("POST", "/api/v1/contacts", workspaceID, map[string]any{"kind": "person", "display_name": "Existing Contact", "email": "existing@example.com"}, &contact, http.StatusCreated)
	// System-managed lifecycle fields are rejected by strict create DTOs.
	admin.must("POST", "/api/v1/crm/leads", workspaceID, map[string]any{"title": "Forged", "status": "converted"}, nil, http.StatusBadRequest)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.lead", "key": "segment", "label": "Segment", "type": "enum", "options": []string{"mid", "enterprise"}}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.lead", "key": "annual_revenue", "label": "Annual revenue", "type": "decimal"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.lead", "key": "employee_count", "label": "Employee count", "type": "integer"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.lead", "key": "priority", "label": "Priority", "type": "boolean"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.lead", "key": "follow_up_date", "label": "Follow-up date", "type": "date"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.opportunity", "key": "forecast_score", "label": "Forecast score", "type": "decimal"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.opportunity", "key": "seat_count", "label": "Seat count", "type": "integer"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.opportunity", "key": "segment", "label": "Segment", "type": "enum", "options": []string{"mid", "enterprise"}}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.opportunity", "key": "priority", "label": "Priority", "type": "boolean"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.opportunity", "key": "follow_up_date", "label": "Follow-up date", "type": "date"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/crm/leads", workspaceID, map[string]any{"title": "Invalid numeric lead", "custom_values": map[string]any{"employee_count": 1.5}}, nil, http.StatusBadRequest)
	admin.must("POST", "/api/v1/crm/leads", workspaceID, map[string]any{"title": "Invalid decimal lead", "custom_values": map[string]any{"annual_revenue": "not-a-number"}}, nil, http.StatusBadRequest)
	admin.must("POST", "/api/v1/crm/leads", workspaceID, map[string]any{"title": "Invalid enum lead", "custom_values": map[string]any{"segment": "unknown"}}, nil, http.StatusBadRequest)
	var lead crmLead
	admin.must("POST", "/api/v1/crm/leads", workspaceID, map[string]any{"title": "Acme lead", "contact_name": "Existing Contact", "email": "existing@example.com", "source": "web", "custom_values": map[string]any{"segment": "mid", "annual_revenue": 125000.5, "employee_count": 42, "priority": true, "follow_up_date": "2026-10-25"}}, &lead, http.StatusCreated)
	if revenue, ok := lead.CustomValues["annual_revenue"].(float64); !ok || revenue != 125000.5 {
		t.Fatalf("decimal custom value was not serialized as a number: %#v", lead.CustomValues["annual_revenue"])
	}
	if employees, ok := lead.CustomValues["employee_count"].(float64); !ok || employees != 42 || lead.CustomValues["priority"] != true || lead.CustomValues["follow_up_date"] != "2026-10-25" {
		t.Fatalf("typed custom values were not preserved: %#v", lead.CustomValues)
	}
	admin.must("PATCH", "/api/v1/crm/leads/"+lead.ID, workspaceID, map[string]any{"version": lead.Version, "title": "Acme qualified"}, &lead, http.StatusOK)
	if lead.Source != "web" || lead.CustomValues["segment"] != "mid" {
		t.Fatalf("sparse lead patch erased data: %#v", lead)
	}
	admin.must("PATCH", "/api/v1/crm/leads/"+lead.ID, workspaceID, map[string]any{"version": lead.Version, "custom_values": map[string]any{"segment": nil}}, &lead, http.StatusOK)
	if _, exists := lead.CustomValues["segment"]; exists {
		t.Fatalf("cleared custom field is still present: %#v", lead.CustomValues)
	}
	if lead.CustomValues["annual_revenue"] != 125000.5 {
		t.Fatalf("clearing one custom field removed unrelated values: %#v", lead.CustomValues)
	}
	admin.must("GET", "/api/v1/crm/leads/"+lead.ID, workspaceID, nil, &lead, http.StatusOK)
	if _, exists := lead.CustomValues["segment"]; exists {
		t.Fatal("cleared custom field was restored after reload")
	}
	admin.must("POST", "/api/v1/crm/leads/"+lead.ID+"/contact", workspaceID, map[string]any{"version": lead.Version}, &lead, http.StatusOK)
	admin.must("POST", "/api/v1/crm/leads/"+lead.ID+"/qualify", workspaceID, map[string]any{"version": lead.Version}, &lead, http.StatusOK)
	var opportunity crmOpportunity
	admin.must("POST", "/api/v1/crm/leads/"+lead.ID+"/convert", workspaceID, map[string]any{"pipeline_id": pipeline.ID, "stage_id": openStage, "contact_id": contact.ID, "create_contact": false, "expected_revenue": "1250.50", "currency": "USD", "version": lead.Version}, &opportunity, http.StatusCreated)
	admin.must("GET", "/api/v1/crm/leads/"+lead.ID, workspaceID, nil, &lead, http.StatusOK)
	if lead.Status != "converted" || lead.ConvertedOpportunityID != opportunity.ID || lead.ConvertedAt == nil {
		t.Fatalf("lead conversion link was not persisted: %#v", lead)
	}
	if opportunity.OriginalLeadID != lead.ID || opportunity.ContactID != contact.ID {
		t.Fatalf("opportunity conversion link was not persisted: %#v", opportunity)
	}
	var savedOpportunityView struct {
		ID string `json:"id"`
	}
	admin.must("POST", "/api/v1/customization/views", workspaceID, map[string]any{
		"entity": "crm.opportunity", "name": "Revenue pipeline opportunities", "columns": []string{"title", "pipeline_id"},
		"filters": []map[string]any{{"field": "pipeline_id", "operator": "eq", "value": pipeline.ID}, {"field": "title", "operator": "contains", "value": "Acme"}}, "sort_field": "title", "sort_direction": "desc",
	}, &savedOpportunityView, http.StatusCreated)
	var filteredOpportunities struct {
		Items []crmOpportunity `json:"items"`
	}
	admin.must("POST", "/api/v1/crm/opportunities", workspaceID, map[string]any{"title": "Beta deal", "pipeline_id": pipeline.ID, "stage_id": openStage, "expected_revenue": "100.00", "currency": "USD"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/crm/opportunities", workspaceID, map[string]any{"title": "Acme expansion", "pipeline_id": pipeline.ID, "stage_id": openStage, "expected_revenue": "200.00", "currency": "USD"}, nil, http.StatusCreated)
	admin.must("GET", "/api/v1/crm/opportunities?view_id="+savedOpportunityView.ID, workspaceID, nil, &filteredOpportunities, http.StatusOK)
	if len(filteredOpportunities.Items) != 2 {
		t.Fatalf("saved view did not apply both conditions: %#v", filteredOpportunities.Items)
	}
	if filteredOpportunities.Items[0].ID != opportunity.ID {
		t.Fatalf("saved view did not order by title descending: %#v", filteredOpportunities.Items)
	}
	admin.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "custom_values": map[string]any{"forecast_score": 87.5, "seat_count": 12, "segment": "enterprise", "priority": true, "follow_up_date": "2026-11-01"}}, &opportunity, http.StatusOK)
	if score, ok := opportunity.CustomValues["forecast_score"].(float64); !ok || score != 87.5 || opportunity.CustomValues["seat_count"] != float64(12) || opportunity.CustomValues["segment"] != "enterprise" || opportunity.CustomValues["priority"] != true || opportunity.CustomValues["follow_up_date"] != "2026-11-01" {
		t.Fatalf("opportunity numeric custom values were not preserved: %#v", opportunity.CustomValues)
	}
	var history []struct {
		EventType string `json:"event_type"`
	}
	admin.must("GET", "/api/v1/crm/opportunities/"+opportunity.ID+"/history", workspaceID, nil, &history, http.StatusOK)
	if len(history) != 1 || history[0].EventType != "lead.converted" {
		t.Fatalf("conversion history was not persisted: %#v", history)
	}
	admin.must("POST", "/api/v1/crm/leads/"+lead.ID+"/convert", workspaceID, map[string]any{"pipeline_id": pipeline.ID, "stage_id": openStage, "contact_id": contact.ID, "create_contact": false, "expected_revenue": "1250.50", "currency": "USD", "version": lead.Version}, nil, http.StatusConflict)
	admin.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "stage_id": openStage}, nil, http.StatusBadRequest)
	admin.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "description": "Updated without moving stage"}, &opportunity, http.StatusOK)
	if opportunity.Description != "Updated without moving stage" {
		t.Fatalf("ordinary opportunity patch did not persist: %#v", opportunity)
	}
	admin.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "expected_close_date": "2026-10-25"}, &opportunity, http.StatusOK)
	if opportunity.ExpectedCloseDate == nil || *opportunity.ExpectedCloseDate != "2026-10-25" {
		t.Fatalf("date-only opportunity value was not preserved: %#v", opportunity.ExpectedCloseDate)
	}
	admin.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "expected_close_date": nil}, &opportunity, http.StatusOK)
	if opportunity.ExpectedCloseDate != nil {
		t.Fatalf("date-only opportunity value was not cleared: %#v", opportunity.ExpectedCloseDate)
	}
	admin.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "expected_close_date": "2026-02-30"}, nil, http.StatusBadRequest)
	var restrictedUser struct {
		ID string `json:"id"`
	}
	admin.must("POST", "/api/v1/users", "", map[string]string{"email": "crm-editor@localhost", "username": "crm-editor", "display_name": "CRM Editor", "password": "crm-editor-password-123"}, &restrictedUser, http.StatusCreated)
	var membership member
	admin.must("POST", "/api/v1/workspace/members", workspaceID, map[string]string{"user_id": restrictedUser.ID}, &membership, http.StatusCreated)
	var editorRole role
	admin.must("POST", "/api/v1/workspace/roles", workspaceID, map[string]string{"name": "crm_editor", "display_name": "CRM editor"}, &editorRole, http.StatusCreated)
	admin.must("PUT", "/api/v1/workspace/roles/"+editorRole.ID+"/permissions", workspaceID, map[string][]string{"permissions": {"crm.opportunity.read", "crm.opportunity.update"}}, nil, http.StatusOK)
	admin.must("PUT", "/api/v1/workspace/members/"+membership.ID+"/roles", workspaceID, map[string][]string{"role_ids": {editorRole.ID}}, nil, http.StatusOK)
	restricted := newAPIClient(t, server.URL)
	restricted.must("POST", "/api/v1/auth/login", "", map[string]string{"email": "crm-editor@localhost", "password": "crm-editor-password-123"}, nil, http.StatusOK)
	restricted.must("GET", "/api/v1/crm/opportunities?view_id="+savedOpportunityView.ID, workspaceID, nil, nil, http.StatusNotFound)
	restricted.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "description": "Allowed ordinary edit"}, &opportunity, http.StatusOK)
	restricted.must("POST", "/api/v1/crm/opportunities/"+opportunity.ID+"/move-stage", workspaceID, map[string]any{"pipeline_id": pipeline.ID, "stage_id": openStage, "version": opportunity.Version}, nil, http.StatusForbidden)
}
