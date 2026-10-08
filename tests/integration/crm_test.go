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
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
)

type crmPipeline struct {
	ID string `json:"id"`
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
	admin.must("POST", "/api/v1/crm/leads/"+lead.ID+"/convert", workspaceID, map[string]any{"pipeline_id": pipeline.ID, "stage_id": openStage, "contact_id": contact.ID, "create_contact": false, "expected_revenue": "1250.50", "currency": "USD", "version": lead.Version}, nil, http.StatusConflict)
	admin.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "stage_id": openStage}, nil, http.StatusBadRequest)
	admin.must("PATCH", "/api/v1/crm/opportunities/"+opportunity.ID, workspaceID, map[string]any{"version": opportunity.Version, "description": "Updated without moving stage"}, &opportunity, http.StatusOK)
	if opportunity.Description != "Updated without moving stage" {
		t.Fatalf("ordinary opportunity patch did not persist: %#v", opportunity)
	}
}
