//go:build integration

package integration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/st-mich43l/apexvoid-CRM/internal/app"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
)

func TestCustomizationManagementAndTenantIsolation(t *testing.T) {
	url, cleanup := isolatedDatabaseURL(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	application, err := app.Bootstrap(ctx, config.Config{
		App: config.AppConfig{Name: "customization-test", Environment: "test"},
		Server: config.ServerConfig{Address: ":0"},
		Database: config.DatabaseConfig{URL: url, MaxConns: 10, MinConns: 1},
		Auth: config.AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour, CookieSameSite: "lax", PasswordMinLen: 12, PasswordMaxLen: 128},
		Bootstrap: config.BootstrapConfig{AdminEmail: "admin@localhost", AdminUsername: "admin", AdminPassword: "admin"},
		Logging: config.LoggingConfig{Level: "ERROR"},
	})
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
	primary := setup.Workspace.ID
	var secondary workspace
	admin.must("POST", "/api/v1/workspaces", primary, map[string]string{"name": "Secondary", "timezone": "UTC"}, &secondary, http.StatusCreated)

	// Existing Phase 5 Contacts field types must be translated in the effective schema.
	admin.must("POST", "/api/v1/contacts/fields", primary, map[string]any{"key": "tier", "label": "Tier", "type": "selection", "options": []string{"standard", "premium"}}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/contacts/fields", primary, map[string]any{"key": "lifetime_value", "label": "Lifetime Value", "type": "number"}, nil, http.StatusCreated)
	var contactSchema struct {
		Fields []struct {
			Key string `json:"key"`
			Type string `json:"type"`
		} `json:"fields"`
	}
	admin.must("GET", "/api/v1/customization/schema/contacts.contact", primary, nil, &contactSchema, http.StatusOK)
	mapped := map[string]string{}
	for _, field := range contactSchema.Fields {
		mapped[field.Key] = field.Type
	}
	if mapped["tier"] != "enum" || mapped["lifetime_value"] != "decimal" {
		t.Fatalf("legacy Contacts field types not mapped: %#v", mapped)
	}

	var goodSection, otherEntitySection struct { ID string `json:"id"` }
	admin.must("POST", "/api/v1/customization/sections", primary, map[string]any{"entity": "customization.saved_view", "name": "Details"}, &goodSection, http.StatusCreated)
	admin.must("POST", "/api/v1/customization/sections", primary, map[string]any{"entity": "customization.field_definition", "name": "Other Entity"}, &otherEntitySection, http.StatusCreated)

	var customField struct {
		ID string `json:"id"`
		Label string `json:"label"`
		Required bool `json:"required"`
		DefaultValue string `json:"default_value"`
		Options []string `json:"options"`
		Active bool `json:"active"`
		Visible bool `json:"visible"`
		SectionID string `json:"section_id"`
	}
	admin.must("POST", "/api/v1/customization/fields", primary, map[string]any{
		"entity": "customization.saved_view", "key": "customer_tier", "label": "Customer Tier",
		"type": "enum", "options": []string{"standard", "premium"}, "default_value": "standard",
		"required": true, "section_id": goodSection.ID,
	}, &customField, http.StatusCreated)
	if customField.ID == "" {
		t.Fatal("missing custom-field identifier")
	}
	var updated = customField
	admin.must("PATCH", "/api/v1/customization/fields/"+customField.ID, primary, map[string]any{"label": "Client Tier"}, &updated, http.StatusOK)
	if updated.Label != "Client Tier" || !updated.Required || updated.DefaultValue != "standard" || !updated.Visible || !updated.Active || updated.SectionID != goodSection.ID || len(updated.Options) != 2 {
		t.Fatalf("PATCH reset omitted properties: %#v", updated)
	}
	admin.must("PATCH", "/api/v1/customization/fields/"+customField.ID, primary, map[string]any{"section_id": otherEntitySection.ID}, nil, http.StatusBadRequest)
	admin.must("PATCH", "/api/v1/customization/fields/"+customField.ID, secondary.ID, map[string]any{"label": "Cross-tenant"}, nil, http.StatusNotFound)
	admin.must("PATCH", "/api/v1/customization/fields/"+customField.ID, primary, map[string]any{"active": false}, &updated, http.StatusOK)

	var inventory []struct { ID string `json:"id"`; Active bool `json:"active"` }
	admin.must("GET", "/api/v1/customization/fields/customization.saved_view", primary, nil, &inventory, http.StatusOK)
	if len(inventory) != 1 || inventory[0].ID != customField.ID || inventory[0].Active {
		t.Fatalf("inactive field missing from management inventory: %#v", inventory)
	}
	var otherInventory []struct { ID string `json:"id"` }
	admin.must("GET", "/api/v1/customization/fields/customization.saved_view", secondary.ID, nil, &otherInventory, http.StatusOK)
	if len(otherInventory) != 0 {
		t.Fatalf("workspace isolation failed: %#v", otherInventory)
	}
	admin.must("PATCH", "/api/v1/customization/fields/"+customField.ID, primary, map[string]any{"active": true}, &updated, http.StatusOK)

	var view struct { ID string `json:"id"` }
	admin.must("POST", "/api/v1/customization/views", primary, map[string]any{
		"entity": "customization.saved_view", "name": "My Saved View", "columns": []string{"name"},
		"filters": []any{}, "sort_field": "name", "sort_direction": "asc",
	}, &view, http.StatusCreated)
	var views []struct { ID string `json:"id"` }
	admin.must("GET", "/api/v1/customization/views/customization.saved_view", primary, nil, &views, http.StatusOK)
	if len(views) != 1 || views[0].ID != view.ID {
		t.Fatalf("created view not visible to owner: %#v", views)
	}
	admin.must("GET", "/api/v1/customization/views/customization.saved_view", secondary.ID, nil, &views, http.StatusOK)
	if len(views) != 0 {
		t.Fatalf("views leaked across workspaces: %#v", views)
	}

	// Nested event hooks are discarded on rollback and run only after outermost commit.
	calls := 0
	rollback := errors.New("abort outer transaction")
	err = application.Transactions.WithTransaction(ctx, func(outer context.Context) error {
		database.AfterCommit(outer, func(context.Context) { calls++ })
		if err := application.Transactions.WithTransaction(outer, func(inner context.Context) error {
			database.AfterCommit(inner, func(context.Context) { calls++ })
			return nil
		}); err != nil {
			return err
		}
		if calls != 0 {
			t.Fatalf("event dispatched before outer transaction completed")
		}
		return rollback
	})
	if !errors.Is(err, rollback) || calls != 0 {
		t.Fatalf("rolled-back transaction dispatched events: calls=%d, error=%v", calls, err)
	}
	err = application.Transactions.WithTransaction(ctx, func(outer context.Context) error {
		database.AfterCommit(outer, func(post context.Context) {
			if _, exists := database.TransactionFromContext(post); exists {
				t.Error("post-commit callback retained committed transaction")
			}
			calls++
		})
		return nil
	})
	if err != nil || calls != 1 {
		t.Fatalf("commit callback not invoked: calls=%d, error=%v", calls, err)
	}
}
