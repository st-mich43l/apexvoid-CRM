//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/app"
	coreapplication "github.com/st-mich43l/apexvoid-CRM/internal/modules/core/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
)

func TestExternalModuleIntegrationLifecycle(t *testing.T) {
	databaseURL, cleanup := isolatedDatabaseURL(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	const assertionSecret = "test-external-assertion-secret-which-is-long-enough"
	platform, err := app.Bootstrap(ctx, config.Config{App: config.AppConfig{Name: "external-module-test", Environment: "test"}, Server: config.ServerConfig{Address: ":0"}, Database: config.DatabaseConfig{URL: databaseURL, MaxConns: 10, MinConns: 1}, Auth: config.AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: time.Hour, CookieSameSite: "lax", PasswordMinLen: 12, PasswordMaxLen: 128}, Integrations: config.IntegrationsConfig{AssertionSecret: assertionSecret, AllowedServiceHosts: []string{"fixture"}}, Bootstrap: config.BootstrapConfig{AdminEmail: "admin@localhost", AdminUsername: "admin", AdminPassword: "admin"}, Logging: config.LoggingConfig{Level: "ERROR"}})
	if err != nil {
		t.Fatal(err)
	}
	defer platform.Close(context.Background())
	router := chi.NewRouter()
	if err := platform.RegisterRoutes(router); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(router)
	defer server.Close()
	unauthenticated := newAPIClient(t, server.URL)
	unauthenticated.must(http.MethodPost, "/api/v1/applications/external", "", map[string]any{}, nil, http.StatusUnauthorized)
	admin := newAPIClient(t, server.URL)
	var identity authResponse
	admin.must(http.MethodPost, "/api/v1/auth/login", "", map[string]string{"email": "admin@localhost", "password": "admin"}, &identity, http.StatusOK)
	admin.must(http.MethodPost, "/api/v1/auth/change-password", "", map[string]string{"current_password": "admin", "new_password": "admin-password-123"}, nil, http.StatusOK)
	var setup setupResponse
	admin.must(http.MethodPost, "/api/v1/setup/organization", "", map[string]string{"organization_name": "External modules", "workspace_name": "Primary", "timezone": "UTC"}, &setup, http.StatusCreated)
	workspaceID := setup.Workspace.ID
	registration := map[string]any{"id": "reports", "display_name": "Reports", "description": "Trusted fixture", "version": "1.0.0", "api_contract_version": "v1", "service_identity": "reports-fixture", "service_endpoint": "http://fixture:8090", "health_endpoint": "http://fixture:8090/health", "frontend_route": "/apps/reports", "access_match": "all", "access_permissions": []string{"reports.report.read"}, "permissions": []map[string]string{{"name": "reports.report.read", "display_name": "Read reports", "scope": "workspace"}}}
	internalCollision := cloneRegistration(registration)
	internalCollision["id"] = "crm"
	internalCollision["service_identity"] = "crm-fixture"
	internalCollision["frontend_route"] = "/apps/crm"
	internalCollision["access_permissions"] = []string{"crm.report.read"}
	internalCollision["permissions"] = []map[string]string{{"name": "crm.report.read", "display_name": "Read CRM reports", "scope": "workspace"}}
	admin.must(http.MethodPost, "/api/v1/applications/external", "", internalCollision, nil, http.StatusConflict)
	invalidPermission := cloneRegistration(registration)
	invalidPermission["permissions"] = []map[string]string{{"name": "reports.invalid-name", "display_name": "Invalid", "scope": "workspace"}}
	admin.must(http.MethodPost, "/api/v1/applications/external", "", invalidPermission, nil, http.StatusBadRequest)
	var persistedAfterFailures int
	if err := platform.Database.QueryRow(ctx, `SELECT COUNT(*) FROM core_external_applications`).Scan(&persistedAfterFailures); err != nil {
		t.Fatal(err)
	}
	if persistedAfterFailures != 0 {
		t.Fatalf("failed registration persisted %d external applications", persistedAfterFailures)
	}
	var registered struct {
		ServiceCredential string `json:"service_credential"`
	}
	admin.must(http.MethodPost, "/api/v1/applications/external", "", registration, &registered, http.StatusCreated)
	if len(registered.ServiceCredential) < 32 {
		t.Fatalf("registration did not return a service credential")
	}
	admin.must(http.MethodPost, "/api/v1/applications/external", "", registration, nil, http.StatusConflict)

	var discovered []struct {
		ID              string `json:"id"`
		EntryAuthorized bool   `json:"entry_authorized"`
	}
	admin.must(http.MethodGet, "/api/v1/framework/applications", workspaceID, nil, &discovered, http.StatusOK)
	if !externalModuleDiscovered(discovered, "reports") {
		t.Fatal("enabled external module was not discovered for its authorized workspace")
	}
	userID, _ := uuid.Parse(identity.User.ID)
	workspaceUUID, _ := uuid.Parse(workspaceID)
	var sessionID uuid.UUID
	if err := platform.Database.QueryRow(ctx, `SELECT id FROM users_sessions WHERE user_id=$1 ORDER BY created_at DESC LIMIT 1`, userID).Scan(&sessionID); err != nil {
		t.Fatal(err)
	}
	issuer, err := coreapplication.NewAssertionIssuer(assertionSecret)
	if err != nil {
		t.Fatal(err)
	}
	assertion, err := issuer.Issue("reports", userID, sessionID, workspaceUUID)
	if err != nil {
		t.Fatal(err)
	}
	introspection := serviceRequest(t, admin, server.URL, http.MethodPost, "/api/v1/integrations/v1/session/introspect", registered.ServiceCredential, map[string]string{"identity_assertion": assertion, "permission": "reports.report.read"})
	if introspection.StatusCode != http.StatusOK {
		t.Fatalf("introspection returned %d", introspection.StatusCode)
	}
	var decision struct {
		Allowed bool `json:"allowed"`
	}
	if err := json.NewDecoder(introspection.Body).Decode(&decision); err != nil {
		t.Fatal(err)
	}
	_ = introspection.Body.Close()
	if !decision.Allowed {
		t.Fatal("workspace administrator was denied a declared external permission")
	}
	foreignPermission := serviceRequest(t, admin, server.URL, http.MethodPost, "/api/v1/integrations/v1/session/introspect", registered.ServiceCredential, map[string]string{"identity_assertion": assertion, "permission": "core.application.manage"})
	if foreignPermission.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign platform permission introspection returned %d", foreignPermission.StatusCode)
	}
	_ = foreignPermission.Body.Close()
	admin.must(http.MethodPut, "/api/v1/applications/external/reports/workspaces/"+workspaceID, "", map[string]bool{"enabled": false}, nil, http.StatusOK)
	admin.must(http.MethodGet, "/api/v1/framework/applications", workspaceID, nil, &discovered, http.StatusOK)
	if externalModuleDiscovered(discovered, "reports") {
		t.Fatal("disabled external module remained in workspace discovery")
	}
	disabled := serviceRequest(t, admin, server.URL, http.MethodPost, "/api/v1/integrations/v1/session/introspect", registered.ServiceCredential, map[string]string{"identity_assertion": assertion, "permission": "reports.report.read"})
	if disabled.StatusCode != http.StatusForbidden {
		t.Fatalf("disabled module introspection returned %d", disabled.StatusCode)
	}
	_ = disabled.Body.Close()
	var rotated struct {
		ServiceCredential string `json:"service_credential"`
	}
	admin.must(http.MethodPost, "/api/v1/applications/external/reports/credentials/rotate", "", nil, &rotated, http.StatusOK)
	if rotated.ServiceCredential == "" || rotated.ServiceCredential == registered.ServiceCredential {
		t.Fatal("credential rotation did not issue a fresh secret")
	}
	oldCredential := serviceRequest(t, admin, server.URL, http.MethodGet, "/api/v1/integrations/v1/applications/reports/availability?workspace_id="+workspaceID, registered.ServiceCredential, nil)
	if oldCredential.StatusCode != http.StatusUnauthorized {
		t.Fatalf("old rotated credential returned %d", oldCredential.StatusCode)
	}
	_ = oldCredential.Body.Close()
	admin.must(http.MethodPost, "/api/v1/applications/external/reports/credentials/revoke", "", nil, nil, http.StatusNoContent)
	revoked := serviceRequest(t, admin, server.URL, http.MethodGet, "/api/v1/integrations/v1/applications/reports/availability?workspace_id="+workspaceID, rotated.ServiceCredential, nil)
	if revoked.StatusCode != http.StatusUnauthorized {
		t.Fatalf("revoked credential returned %d", revoked.StatusCode)
	}
	_ = revoked.Body.Close()
	admin.must(http.MethodDelete, "/api/v1/applications/external/reports", "", nil, nil, http.StatusNoContent)
	admin.must(http.MethodPost, "/api/v1/applications/external", "", registration, nil, http.StatusConflict)
}

func cloneRegistration(input map[string]any) map[string]any {
	result := make(map[string]any, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func externalModuleDiscovered(items []struct {
	ID              string `json:"id"`
	EntryAuthorized bool   `json:"entry_authorized"`
}, id string) bool {
	for _, item := range items {
		if item.ID == id {
			return item.EntryAuthorized
		}
	}
	return false
}

func serviceRequest(t *testing.T, client *apiClient, baseURL, method, path, credential string, body any) *http.Response {
	t.Helper()
	var payload *bytes.Reader
	if body == nil {
		payload = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		payload = bytes.NewReader(raw)
	}
	request, err := http.NewRequest(method, baseURL+path, payload)
	if err != nil {
		t.Fatal(err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("X-ApexVoid-Application-ID", "reports")
	request.Header.Set("X-ApexVoid-Service-Credential", credential)
	response, err := client.client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	return response
}
