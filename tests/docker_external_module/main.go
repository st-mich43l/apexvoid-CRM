// docker_external_module verifies the external gateway with the real Compose
// network, Nginx frontend, Core API, PostgreSQL, and a service fixture.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"
)

const fixtureCredential = "external-module-fixture-credential-0123456789abcdef"

type apiClient struct {
	baseURL string
	client  *http.Client
}

func main() {
	phase := flag.String("phase", "full", "setup, verify, outage, finalize, or full")
	flag.Parse()
	frontendURL := env("APEXVOID_EXTERNAL_TEST_URL", "http://localhost:18386")
	backendURL := env("APEXVOID_EXTERNAL_BACKEND_URL", "http://localhost:16868")
	if *phase == "verify" {
		verifyPersistentRegistration(frontendURL)
		return
	}
	if *phase == "outage" {
		verifyFixtureOutage(frontendURL)
		return
	}
	if *phase == "finalize" {
		finalizeLifecycle(frontendURL, backendURL)
		return
	}
	admin := newClient(frontendURL)
	must(admin.request(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "admin@external.test", "password": "admin"}, http.StatusOK, nil))
	must(admin.request(http.MethodPost, "/api/v1/auth/change-password", map[string]string{"current_password": "admin", "new_password": "admin-password-123"}, http.StatusOK, nil))
	var setup struct {
		Workspace struct {
			ID string `json:"id"`
		} `json:"workspace"`
	}
	must(admin.request(http.MethodPost, "/api/v1/setup/organization", map[string]string{"organization_name": "External integration", "workspace_name": "Primary", "timezone": "UTC"}, http.StatusCreated, &setup))
	workspaceID := setup.Workspace.ID
	registration := map[string]any{
		"id": "reports", "display_name": "Reports", "description": "Docker fixture", "version": "1.0.0", "api_contract_version": "v1", "service_identity": "reports-fixture", "service_endpoint": "http://fixture:8090", "health_endpoint": "http://fixture:8090/health", "frontend_route": "/apps/reports", "access_match": "all", "access_permissions": []string{"reports.report.read"}, "permissions": []map[string]string{{"name": "reports.report.read", "display_name": "Read reports", "scope": "workspace"}}, "service_credential": fixtureCredential,
	}
	must(admin.request(http.MethodPost, "/api/v1/applications/external", registration, http.StatusCreated, nil))
	must(admin.request(http.MethodGet, "/apps/reports", nil, http.StatusOK, nil))
	must(admin.request(http.MethodGet, "/api/apps/reports/records", nil, http.StatusOK, nil))

	user := newClient(frontendURL)
	must(admin.request(http.MethodPost, "/api/v1/users", map[string]string{"email": "member@external.test", "display_name": "Member", "password": "member-password-123"}, http.StatusCreated, nil))
	must(user.request(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "member@external.test", "password": "member-password-123"}, http.StatusOK, nil))
	must(user.request(http.MethodPost, "/api/v1/auth/change-password", map[string]string{"current_password": "member-password-123", "new_password": "member-password-456"}, http.StatusOK, nil))
	must(user.request(http.MethodGet, "/apps/reports", nil, http.StatusForbidden, nil))

	must(admin.request(http.MethodPut, "/api/v1/applications/external/reports/workspaces/"+workspaceID, map[string]bool{"enabled": false}, http.StatusOK, nil))
	must(admin.request(http.MethodGet, "/api/apps/reports/records", nil, http.StatusForbidden, nil))
	must(admin.request(http.MethodPut, "/api/v1/applications/external/reports/workspaces/"+workspaceID, map[string]bool{"enabled": true}, http.StatusOK, nil))
	if *phase == "setup" {
		fmt.Println("external Docker gateway setup: ok")
		return
	}
	finalize(admin, backendURL)
	fmt.Println("external Docker gateway integration: ok")
}

func finalizeLifecycle(frontendURL, backendURL string) {
	admin := newClient(frontendURL)
	must(admin.request(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "admin@external.test", "password": "admin-password-123"}, http.StatusOK, nil))
	finalize(admin, backendURL)
	fmt.Println("external Docker gateway finalization: ok")
}

func finalize(admin *apiClient, backendURL string) {
	var rotated struct {
		ServiceCredential string `json:"service_credential"`
	}
	must(admin.request(http.MethodPost, "/api/v1/applications/external/reports/credentials/rotate", nil, http.StatusOK, &rotated))
	if rotated.ServiceCredential == "" || rotated.ServiceCredential == fixtureCredential {
		panic("credential rotation did not issue a new secret")
	}
	must(serviceRequest(backendURL, fixtureCredential, http.StatusUnauthorized))
	must(admin.request(http.MethodPost, "/api/v1/applications/external/reports/credentials/revoke", nil, http.StatusNoContent, nil))
	must(serviceRequest(backendURL, rotated.ServiceCredential, http.StatusUnauthorized))
	must(admin.request(http.MethodDelete, "/api/v1/applications/external/reports", nil, http.StatusNoContent, nil))
	must(admin.request(http.MethodGet, "/apps/reports", nil, http.StatusNotFound, nil))
}

func verifyPersistentRegistration(frontendURL string) {
	admin := newClient(frontendURL)
	must(admin.request(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "admin@external.test", "password": "admin-password-123"}, http.StatusOK, nil))
	var applications []struct {
		ID string `json:"id"`
	}
	must(admin.request(http.MethodGet, "/api/v1/applications/external", nil, http.StatusOK, &applications))
	found := false
	for _, application := range applications {
		if application.ID == "reports" {
			found = true
		}
	}
	if !found {
		panic("registered external application was not restored after restart")
	}
	must(admin.request(http.MethodGet, "/api/apps/reports/records", nil, http.StatusOK, nil))
	fmt.Println("external Docker gateway restart recovery: ok")
}

func verifyFixtureOutage(frontendURL string) {
	admin := newClient(frontendURL)
	must(admin.request(http.MethodPost, "/api/v1/auth/login", map[string]string{"email": "admin@external.test", "password": "admin-password-123"}, http.StatusOK, nil))
	must(admin.request(http.MethodGet, "/api/apps/reports/records", nil, http.StatusBadGateway, nil))
	fmt.Println("external Docker fixture outage: safe gateway failure")
}

func newClient(baseURL string) *apiClient {
	jar, err := cookiejar.New(nil)
	must(err)
	return &apiClient{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Jar: jar, Timeout: 10 * time.Second}}
}

func (c *apiClient) request(method, path string, body any, expected int, output any) error {
	var reader *bytes.Reader
	if body == nil {
		reader = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != expected {
		return fmt.Errorf("%s %s: got %d, want %d", method, path, res.StatusCode, expected)
	}
	if output != nil {
		return json.NewDecoder(res.Body).Decode(output)
	}
	return nil
}

func serviceRequest(baseURL, credential string, expected int) error {
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(baseURL, "/")+"/api/v1/integrations/v1/applications/reports/availability?workspace_id=00000000-0000-0000-0000-000000000001", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-ApexVoid-Application-ID", "reports")
	req.Header.Set("X-ApexVoid-Service-Credential", credential)
	res, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != expected {
		return fmt.Errorf("service credential: got %d, want %d", res.StatusCode, expected)
	}
	return nil
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
func must(err error) {
	if err != nil {
		panic(err)
	}
}
