//go:build integration

package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/app"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
)

func TestERPProductCatalogLifecycle(t *testing.T) {
	databaseURL, cleanup := isolatedDatabaseURL(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	platform, err := app.Bootstrap(ctx, config.Config{
		App: config.AppConfig{Name: "erp-test", Environment: "test"},
		Server: config.ServerConfig{Address: ":0"},
		Database: config.DatabaseConfig{URL: databaseURL, MaxConns: 10, MinConns: 1},
		Auth: config.AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: time.Hour, CookieSameSite: "lax", PasswordMinLen: 12, PasswordMaxLen: 128},
		Bootstrap: config.BootstrapConfig{AdminEmail: "admin@localhost", AdminUsername: "admin", AdminPassword: "admin"},
		Logging: config.LoggingConfig{Level: "ERROR"},
	})
	if err != nil { t.Fatal(err) }
	defer platform.Close(context.Background())
	router := chi.NewRouter()
	if err := platform.RegisterRoutes(router); err != nil { t.Fatal(err) }
	server := httptest.NewServer(router)
	defer server.Close()

	guest := newAPIClient(t, server.URL)
	guest.must(http.MethodGet, "/api/v1/erp/products", "", nil, nil, http.StatusUnauthorized)
	admin := newAPIClient(t, server.URL)
	admin.must(http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "admin@localhost", "password": "admin"}, nil, http.StatusOK)
	admin.must(http.MethodPost, "/api/v1/auth/change-password", "", map[string]string{"current_password": "admin", "new_password": "admin-password-123"}, nil, http.StatusOK)
	var setup setupResponse
	admin.must(http.MethodPost, "/api/v1/setup/organization", "", map[string]string{"organization_name": "ERP test", "workspace_name": "Primary", "timezone": "UTC"}, &setup, http.StatusCreated)
	workspaceID := setup.Workspace.ID

	var applications []struct { ID string `json:"id"` }
	admin.must(http.MethodGet, "/api/v1/framework/applications", workspaceID, nil, &applications, http.StatusOK)
	crmFound, erpFound := false, false
	for _, item := range applications {
		if item.ID == "crm" { crmFound = true }
		if item.ID == "erp" { erpFound = true }
	}
	if !crmFound || !erpFound { t.Fatal("CRM and ERP must both remain registered") }

	product := map[string]any{
		"sku": "SKU-100", "name": "Office chair", "description": "Ergonomic chair",
		"kind": "good", "unit": "unit", "unit_price": "129.1234", "currency": "USD",
	}
	var created struct {
		ID uuid.UUID `json:"id"`
		Version int `json:"version"`
		UnitPrice string `json:"unit_price"`
	}
	admin.must(http.MethodPost, "/api/v1/erp/products", workspaceID, product, &created, http.StatusCreated)
	if created.ID == uuid.Nil || created.Version != 1 || created.UnitPrice != "129.1234" {
		t.Fatalf("unexpected decimal or created product: %#v", created)
	}
	admin.must(http.MethodPost, "/api/v1/erp/products", workspaceID, product, nil, http.StatusConflict)
	badPrice := map[string]any{
		"sku": "SKU-101", "name": "Bad Price", "kind": "good",
		"unit": "unit", "unit_price": "1.12345", "currency": "USD",
	}
	admin.must(http.MethodPost, "/api/v1/erp/products", workspaceID, badPrice, nil, http.StatusBadRequest)

	var listed struct { Total int `json:"total"`; Items []struct { ID uuid.UUID `json:"id"` } `json:"items"` }
	admin.must(http.MethodGet, "/api/v1/erp/products", workspaceID, nil, &listed, http.StatusOK)
	if listed.Total != 1 || len(listed.Items) != 1 || listed.Items[0].ID != created.ID {
		t.Fatalf("unexpected catalog response %#v", listed)
	}
	forgedWorkspace := uuid.New().String()
	admin.must(http.MethodGet, "/api/v1/erp/products", forgedWorkspace, nil, nil, http.StatusForbidden)
	admin.must(http.MethodGet, "/api/v1/erp/products/"+created.ID.String(), forgedWorkspace, nil, nil, http.StatusForbidden)

	product["name"] = "Office chair updated"
	product["version"] = created.Version
	var updated struct { Version int `json:"version"` }
	admin.must(http.MethodPut, "/api/v1/erp/products/"+created.ID.String(), workspaceID, product, &updated, http.StatusOK)
	if updated.Version != 2 { t.Fatalf("product version was not incremented: %#v", updated) }
	admin.must(http.MethodPut, "/api/v1/erp/products/"+created.ID.String(), workspaceID, product, nil, http.StatusConflict)

	var archived struct { Status string `json:"status"`; Version int `json:"version"` }
	admin.must(http.MethodPost, "/api/v1/erp/products/"+created.ID.String()+"/archive", workspaceID, map[string]any{"version": updated.Version}, &archived, http.StatusOK)
	if archived.Status != "archived" { t.Fatalf("archive did not persist: %#v", archived) }
	admin.must(http.MethodGet, "/api/v1/erp/products", workspaceID, nil, &listed, http.StatusOK)
	if listed.Total != 0 { t.Fatal("archived product unexpectedly in active catalog") }
	admin.must(http.MethodPost, "/api/v1/erp/products/"+created.ID.String()+"/restore", workspaceID, map[string]any{"version": archived.Version}, nil, http.StatusOK)
	admin.must(http.MethodGet, "/api/v1/erp/products", workspaceID, nil, &listed, http.StatusOK)
	if listed.Total != 1 { t.Fatal("restored product missing from catalog") }
}
