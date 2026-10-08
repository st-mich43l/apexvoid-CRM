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
	ID string `json:"id"`
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
	ID       string `json:"id"`
	Category string `json:"category"`
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
	ID             string `json:"id"`
	Version        int    `json:"version"`
	OriginalLeadID string `json:"original_lead_id"`
	ContactID      string `json:"contact_id"`
	Description    string `json:"description"`
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
	var contact contactResponse
	admin.must("POST", "/api/v1/contacts", workspaceID, map[string]any{"kind": "person", "display_name": "Existing Contact", "email": "existing@example.com"}, &contact, http.StatusCreated)
	// System-managed lifecycle fields are rejected by strict create DTOs.
	admin.must("POST", "/api/v1/crm/leads", workspaceID, map[string]any{"title": "Forged", "status": "converted"}, nil, http.StatusBadRequest)
	admin.must("POST", "/api/v1/customization/fields", workspaceID, map[string]any{"entity": "crm.lead", "key": "segment", "label": "Segment", "type": "enum", "options": []string{"mid", "enterprise"}}, nil, http.StatusCreated)
	var lead crmLead
	admin.must("POST", "/api/v1/crm/leads", workspaceID, map[string]any{"title": "Acme lead", "contact_name": "Existing Contact", "email": "existing@example.com", "source": "web", "custom_values": map[string]any{"segment": "mid"}}, &lead, http.StatusCreated)
	admin.must("PATCH", "/api/v1/crm/leads/"+lead.ID, workspaceID, map[string]any{"version": lead.Version, "title": "Acme qualified"}, &lead, http.StatusOK)
	if lead.Source != "web" || lead.CustomValues["segment"] != "mid" {
		t.Fatalf("sparse lead patch erased data: %#v", lead)
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
	restricted.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "description": "Allowed ordinary edit"}, &opportunity, http.StatusOK)
	restricted.must("POST", "/api/v1/crm/opportunities/"+opportunity.ID+"/move-stage", workspaceID, map[string]any{"pipeline_id": pipeline.ID, "stage_id": openStage, "version": opportunity.Version}, nil, http.StatusForbidden)
}
