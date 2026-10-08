//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	neturl "net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/app"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/config"
)

type apiClient struct {
	t       *testing.T
	client  *http.Client
	baseURL string
}

func (c *apiClient) multipartRequest(method, path, workspaceID, filename, contentType string, content []byte, target any) int {
	c.t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		c.t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		c.t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		c.t.Fatal(err)
	}
	req, err := http.NewRequest(method, c.baseURL+path, &body)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if contentType != "" {
		req.Header.Set("X-Test-Content-Type", contentType)
	}
	if workspaceID != "" {
		req.Header.Set("X-ApexVoid-Workspace", workspaceID)
	}
	response, err := c.client.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	if target != nil && len(responseBody) > 0 {
		if err := json.Unmarshal(responseBody, target); err != nil {
			c.t.Fatalf("decode multipart response (%d): %v", response.StatusCode, err)
		}
	}
	return response.StatusCode
}

func newAPIClient(t *testing.T, baseURL string) *apiClient {
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &apiClient{t: t, client: &http.Client{Jar: jar}, baseURL: baseURL}
}

func (c *apiClient) request(method, path string, workspaceID string, body any, target any) int {
	return c.requestContext(context.Background(), method, path, workspaceID, body, target)
}

func (c *apiClient) requestContext(ctx context.Context, method, path string, workspaceID string, body any, target any) int {
	c.t.Helper()
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			c.t.Fatal(err)
		}
		reader = bytes.NewReader(encoded)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		c.t.Fatal(err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if workspaceID != "" {
		req.Header.Set("X-ApexVoid-Workspace", workspaceID)
	}
	response, err := c.client.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer response.Body.Close()
	bodyBytes, err := io.ReadAll(response.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	if target != nil && len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, target); err != nil {
			c.t.Fatalf("decode %s %s response (%d): %v", method, path, response.StatusCode, err)
		}
	}
	if response.StatusCode >= 400 {
		c.t.Logf("%s %s returned %d: %s", method, path, response.StatusCode, string(bodyBytes))
	}
	return response.StatusCode
}

func (c *apiClient) must(method, path, workspaceID string, body any, target any, status int) {
	c.t.Helper()
	got := c.request(method, path, workspaceID, body, target)
	if got != status {
		c.t.Fatalf("expected %s %s to return %d, got %d", method, path, status, got)
	}
}

type authResponse struct {
	User struct {
		ID string `json:"id"`
	} `json:"user"`
	Permissions []string `json:"permissions"`
}

type setupResponse struct {
	Workspace struct {
		ID string `json:"id"`
	} `json:"workspace"`
	Membership struct {
		ID string `json:"id"`
	} `json:"membership"`
}

type workspace struct {
	ID string `json:"id"`
}

type member struct {
	ID string `json:"id"`
}

type role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type workspaceContext struct {
	Permissions []string `json:"permissions"`
}

type contactResponse struct {
	ID string `json:"id"`
}

type tagResponse struct {
	ID string `json:"id"`
}

type fieldResponse struct {
	ID string `json:"id"`
}

type attachmentResponse struct {
	ID string `json:"id"`
}

func TestOrganizationWorkspaceAccessAndLastAdministrator(t *testing.T) {
	databaseURL, cleanup := isolatedDatabaseURL(t)
	defer cleanup()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	application, err := app.Bootstrap(ctx, config.Config{
		App:       config.AppConfig{Name: "integration", Environment: "test"},
		Server:    config.ServerConfig{Address: ":0"},
		Database:  config.DatabaseConfig{URL: databaseURL, MaxConns: 10, MinConns: 1},
		Auth:      config.AuthConfig{AccessTokenTTL: 15 * time.Minute, RefreshTokenTTL: 24 * time.Hour, CookieSameSite: "lax", PasswordMinLen: 12, PasswordMaxLen: 128},
		Bootstrap: config.BootstrapConfig{AdminEmail: "admin@localhost", AdminUsername: "admin", AdminPassword: "admin"},
		Logging:   config.LoggingConfig{Level: "ERROR"},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer application.Close(context.Background())

	var users int
	if err := application.Database.QueryRow(ctx, "SELECT COUNT(*) FROM users_users").Scan(&users); err != nil {
		t.Fatal(err)
	}
	if users != 1 {
		t.Fatalf("isolated integration database contains unexpected users: %d", users)
	}

	router := chi.NewRouter()
	if err := application.RegisterRoutes(router); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(router)
	defer server.Close()

	admin := newAPIClient(t, server.URL)
	adminLogin := authResponse{}
	admin.must("POST", "/api/v1/auth/login", "", map[string]string{"email": "admin@localhost", "password": "admin"}, &adminLogin, http.StatusOK)
	if !contains(adminLogin.Permissions, "organization.organization.update") {
		t.Fatalf("bootstrap administrator permissions did not include organization update: %#v", adminLogin.Permissions)
	}
	admin.must("POST", "/api/v1/auth/change-password", "", map[string]string{"current_password": "admin", "new_password": "admin-password-123"}, &authResponse{}, http.StatusOK)

	setup := setupResponse{}
	admin.must("POST", "/api/v1/setup/organization", "", map[string]string{"organization_name": "Integration Org", "workspace_name": "Primary", "timezone": "UTC"}, &setup, http.StatusCreated)
	primaryWorkspaceID := setup.Workspace.ID
	primaryMembershipID := setup.Membership.ID
	admin.must("POST", "/api/v1/setup/organization", "", map[string]string{"organization_name": "Duplicate", "workspace_name": "Duplicate", "timezone": "UTC"}, nil, http.StatusConflict)
	var organizations, workspaces, memberships int
	if err := application.Database.QueryRow(ctx, "SELECT COUNT(*) FROM organization_organizations").Scan(&organizations); err != nil {
		t.Fatal(err)
	}
	if err := application.Database.QueryRow(ctx, "SELECT COUNT(*) FROM workspace_workspaces").Scan(&workspaces); err != nil {
		t.Fatal(err)
	}
	if err := application.Database.QueryRow(ctx, "SELECT COUNT(*) FROM workspace_memberships").Scan(&memberships); err != nil {
		t.Fatal(err)
	}
	if organizations != 1 || workspaces != 1 || memberships != 1 {
		t.Fatalf("idempotent setup created duplicate records: organizations=%d workspaces=%d memberships=%d", organizations, workspaces, memberships)
	}

	var secondary workspace
	admin.must("POST", "/api/v1/workspaces", primaryWorkspaceID, map[string]string{"name": "Secondary", "timezone": "UTC"}, &secondary, http.StatusCreated)
	var secondaryRole role
	admin.must("POST", "/api/v1/workspace/roles", secondary.ID, map[string]string{"name": "secondary_role", "display_name": "Secondary Role"}, &secondaryRole, http.StatusCreated)
	admin.must("POST", "/api/v1/workspace/roles", secondary.ID, map[string]string{"name": "administrator", "display_name": "Spoofed Administrator"}, nil, http.StatusConflict)
	admin.must("POST", "/api/v1/access/roles", "", map[string]string{"name": "administrator", "display_name": "Spoofed Administrator"}, nil, http.StatusConflict)
	admin.must("PUT", fmt.Sprintf("/api/v1/workspace/roles/%s/permissions", secondaryRole.ID), primaryWorkspaceID, map[string][]string{"permissions": []string{"workspace.member.read"}}, nil, http.StatusBadRequest)

	var primaryPerson, primaryCompany, secondaryPerson contactResponse
	admin.must("POST", "/api/v1/contacts", primaryWorkspaceID, map[string]any{"kind": "person", "display_name": "Primary Person", "email": "same@example.com"}, &primaryPerson, http.StatusCreated)
	admin.must("POST", "/api/v1/contacts", primaryWorkspaceID, map[string]any{"kind": "company", "display_name": "Primary Company", "email": "same@example.com"}, &primaryCompany, http.StatusCreated)
	admin.must("POST", "/api/v1/contacts", secondary.ID, map[string]any{"kind": "person", "display_name": "Secondary Person"}, &secondaryPerson, http.StatusCreated)
	var customField fieldResponse
	admin.must("POST", "/api/v1/contacts/fields", primaryWorkspaceID, map[string]any{"key": "tier", "label": "Tier", "type": "selection", "options": []string{"gold", "silver"}}, &customField, http.StatusCreated)
	var tagged contactResponse
	admin.must("PATCH", "/api/v1/contacts/"+primaryPerson.ID, primaryWorkspaceID, map[string]any{"kind": "person", "display_name": "Primary Person", "custom_values": map[string]any{"tier": "gold"}}, &tagged, http.StatusOK)
	var tag tagResponse
	admin.must("POST", "/api/v1/contacts/tags", primaryWorkspaceID, map[string]string{"name": "VIP", "color": "#7c3aed"}, &tag, http.StatusCreated)
	admin.must("PUT", "/api/v1/contacts/"+primaryPerson.ID+"/tags", primaryWorkspaceID, []string{tag.ID}, nil, http.StatusNoContent)
	admin.must("PUT", "/api/v1/contacts/"+primaryPerson.ID+"/relationships", primaryWorkspaceID, []map[string]any{{"person_id": primaryPerson.ID, "company_id": primaryCompany.ID, "relationship_type": "employee", "job_title": "Founder", "is_primary": true}}, nil, http.StatusNoContent)
	admin.must("POST", "/api/v1/contacts/"+primaryPerson.ID+"/notes", primaryWorkspaceID, map[string]string{"content": "Workspace-scoped note"}, nil, http.StatusCreated)
	admin.must("POST", "/api/v1/activities", primaryWorkspaceID, map[string]any{"title": "Welcome call", "activity_type": "call", "related_contact_id": primaryPerson.ID, "assigned_user_id": adminLogin.User.ID}, nil, http.StatusCreated)
	var attachment attachmentResponse
	if got := admin.multipartRequest("POST", "/api/v1/contacts/"+primaryPerson.ID+"/attachments", primaryWorkspaceID, "brief.pdf", "application/pdf", []byte("%PDF-1.7\nattachment"), &attachment); got != http.StatusCreated {
		t.Fatalf("expected attachment upload to return %d, got %d", http.StatusCreated, got)
	}
	downloadRequest, err := http.NewRequest("GET", server.URL+"/api/v1/contacts/"+primaryPerson.ID+"/attachments/"+attachment.ID, nil)
	if err != nil {
		t.Fatal(err)
	}
	downloadRequest.Header.Set("X-ApexVoid-Workspace", primaryWorkspaceID)
	downloadResponse, err := admin.client.Do(downloadRequest)
	if err != nil {
		t.Fatal(err)
	}
	downloadBody, err := io.ReadAll(downloadResponse.Body)
	downloadResponse.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if downloadResponse.StatusCode != http.StatusOK || string(downloadBody) != "%PDF-1.7\nattachment" || downloadResponse.Header.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unexpected attachment download: status=%d body=%q nosniff=%q", downloadResponse.StatusCode, string(downloadBody), downloadResponse.Header.Get("X-Content-Type-Options"))
	}
	admin.must("GET", "/api/v1/contacts/"+primaryPerson.ID+"/attachments/"+attachment.ID, secondary.ID, nil, nil, http.StatusNotFound)
	admin.must("DELETE", "/api/v1/contacts/"+primaryPerson.ID+"/attachments/"+attachment.ID, primaryWorkspaceID, nil, nil, http.StatusNoContent)
	admin.must("GET", "/api/v1/contacts/"+secondaryPerson.ID, primaryWorkspaceID, nil, nil, http.StatusNotFound)
	admin.must("GET", "/api/v1/contacts/"+primaryPerson.ID+"/relationships", secondary.ID, nil, nil, http.StatusNotFound)

	var createdUser struct {
		ID string `json:"id"`
	}
	admin.must("POST", "/api/v1/users", "", map[string]string{"email": "member@localhost", "username": "member", "display_name": "Workspace Member", "password": "member-password-123"}, &createdUser, http.StatusCreated)
	var platformRoles []role
	admin.must("GET", "/api/v1/access/roles", "", nil, &platformRoles, http.StatusOK)
	var platformAdministratorID string
	for _, item := range platformRoles {
		if item.Name == "administrator" {
			platformAdministratorID = item.ID
		}
	}
	if platformAdministratorID == "" {
		t.Fatal("platform administrator role was not provisioned")
	}
	var secondPlatformAdministrator struct {
		ID string `json:"id"`
	}
	admin.must("POST", "/api/v1/users", "", map[string]string{"email": "platform2@localhost", "username": "platform2", "display_name": "Second Platform Administrator", "password": "platform2-password-123"}, &secondPlatformAdministrator, http.StatusCreated)
	admin.must("PUT", fmt.Sprintf("/api/v1/users/%s/roles", secondPlatformAdministrator.ID), "", map[string][]string{"role_ids": []string{platformAdministratorID}}, nil, http.StatusOK)
	admin.must("POST", fmt.Sprintf("/api/v1/users/%s/disable", secondPlatformAdministrator.ID), "", nil, nil, http.StatusOK)
	admin.must("POST", fmt.Sprintf("/api/v1/users/%s/disable", adminLogin.User.ID), "", nil, nil, http.StatusConflict)

	var added member
	admin.must("POST", "/api/v1/workspace/members", primaryWorkspaceID, map[string]string{"user_id": createdUser.ID}, &added, http.StatusCreated)

	var roles []role
	admin.must("GET", "/api/v1/workspace/roles", primaryWorkspaceID, nil, &roles, http.StatusOK)
	var workspaceAdministratorID string
	for _, item := range roles {
		if item.Name == "workspace_administrator" {
			workspaceAdministratorID = item.ID
		}
	}
	if workspaceAdministratorID == "" {
		t.Fatal("workspace administrator role was not provisioned")
	}
	var primaryRoles struct {
		RoleIDs []string `json:"role_ids"`
	}
	admin.must("GET", fmt.Sprintf("/api/v1/workspace/members/%s/roles", primaryMembershipID), primaryWorkspaceID, nil, &primaryRoles, http.StatusOK)
	if !contains(primaryRoles.RoleIDs, workspaceAdministratorID) {
		t.Fatalf("initial workspace administrator role was not assigned: %#v", primaryRoles.RoleIDs)
	}

	admin.must("PATCH", fmt.Sprintf("/api/v1/workspace/members/%s", primaryMembershipID), primaryWorkspaceID, map[string]string{"status": "suspended"}, nil, http.StatusConflict)
	admin.must("PUT", fmt.Sprintf("/api/v1/workspace/members/%s/roles", primaryMembershipID), primaryWorkspaceID, map[string][]string{"role_ids": []string{}}, nil, http.StatusConflict)
	admin.must("DELETE", fmt.Sprintf("/api/v1/workspace/members/%s", primaryMembershipID), primaryWorkspaceID, nil, nil, http.StatusConflict)
	admin.must("PUT", fmt.Sprintf("/api/v1/workspace/members/%s/roles", added.ID), primaryWorkspaceID, map[string][]string{"role_ids": []string{workspaceAdministratorID}}, nil, http.StatusOK)
	admin.must("PUT", fmt.Sprintf("/api/v1/workspace/members/%s/roles", added.ID), primaryWorkspaceID, map[string][]string{"role_ids": []string{secondaryRole.ID}}, nil, http.StatusBadRequest)
	malformedRoleID := uuid.New()
	if _, err := application.Database.Exec(ctx, `ALTER TABLE access_roles DISABLE TRIGGER access_protected_role_identity`); err != nil {
		t.Fatal(err)
	}
	_, insertErr := application.Database.Exec(ctx, `INSERT INTO access_roles (id, workspace_id, name, display_name, description, system, created_at, updated_at) VALUES ($1,$2,'administrator','Malformed Administrator','',TRUE,NOW(),NOW())`, malformedRoleID, secondary.ID)
	_, enableErr := application.Database.Exec(ctx, `ALTER TABLE access_roles ENABLE TRIGGER access_protected_role_identity`)
	if insertErr != nil || enableErr != nil {
		t.Fatalf("create malformed role fixture: insert=%v enable=%v", insertErr, enableErr)
	}
	if _, err := application.Database.Exec(ctx, `INSERT INTO access_user_roles (user_id, role_id) VALUES ($1,$2)`, createdUser.ID, malformedRoleID); err != nil {
		t.Fatal(err)
	}

	memberClient := newAPIClient(t, server.URL)
	memberClient.must("POST", "/api/v1/auth/login", "", map[string]string{"email": "member@localhost", "password": "member-password-123"}, &authResponse{}, http.StatusOK)
	var memberWorkspaces []workspace
	memberClient.must("GET", "/api/v1/workspaces", "", nil, &memberWorkspaces, http.StatusOK)
	if len(memberWorkspaces) != 1 || memberWorkspaces[0].ID != primaryWorkspaceID {
		t.Fatalf("member can access unexpected workspaces: %#v", memberWorkspaces)
	}
	memberClient.must("GET", "/api/v1/workspace", secondary.ID, nil, nil, http.StatusForbidden)
	var current workspaceContext
	memberClient.must("GET", "/api/v1/workspace", primaryWorkspaceID, nil, &current, http.StatusOK)
	if !contains(current.Permissions, "workspace.member.read") {
		t.Fatalf("workspace administrator did not receive dynamic workspace permissions: %#v", current.Permissions)
	}
	memberClient.must("GET", "/api/v1/users", "", nil, nil, http.StatusForbidden)

	concurrentCtx, concurrentCancel := context.WithTimeout(ctx, 5*time.Second)
	defer concurrentCancel()
	results := make(chan int, 2)
	go func() {
		results <- admin.requestContext(concurrentCtx, "DELETE", fmt.Sprintf("/api/v1/workspace/members/%s", primaryMembershipID), primaryWorkspaceID, nil, nil)
	}()
	go func() {
		results <- admin.requestContext(concurrentCtx, "DELETE", fmt.Sprintf("/api/v1/workspace/members/%s", added.ID), primaryWorkspaceID, nil, nil)
	}()
	first, second := <-results, <-results
	if !((first == http.StatusNoContent && second == http.StatusConflict) || (first == http.StatusConflict && second == http.StatusNoContent)) {
		t.Fatalf("concurrent administrator removals returned %d and %d", first, second)
	}
	var remainingAdministrators int
	if err := application.Database.QueryRow(ctx, `SELECT COUNT(DISTINCT m.id) FROM workspace_memberships m JOIN access_workspace_membership_roles wmr ON wmr.membership_id = m.id JOIN access_roles r ON r.id = wmr.role_id WHERE m.workspace_id = $1 AND m.status = 'active' AND r.workspace_id = $1 AND r.name = 'workspace_administrator' AND r.system = TRUE`, primaryWorkspaceID).Scan(&remainingAdministrators); err != nil {
		t.Fatal(err)
	}
	if remainingAdministrators != 1 {
		t.Fatalf("concurrent removals left %d administrators, want exactly one", remainingAdministrators)
	}
}

func isolatedDatabaseURL(t *testing.T) (string, func()) {
	url := os.Getenv("APEXVOID_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set APEXVOID_TEST_DATABASE_URL to run PostgreSQL integration tests")
	}
	adminPool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	schema := "apexvoid_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := adminPool.Exec(context.Background(), `CREATE SCHEMA `+schema); err != nil {
		adminPool.Close()
		t.Fatal(err)
	}
	parsed, err := neturl.Parse(url)
	if err != nil {
		adminPool.Close()
		t.Fatal(err)
	}
	query := parsed.Query()
	query.Set("options", "-c search_path="+schema+",public")
	parsed.RawQuery = query.Encode()
	return parsed.String(), func() {
		defer adminPool.Close()
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = adminPool.Exec(cleanupCtx, `DROP SCHEMA `+schema+` CASCADE`)
	}
}

func contains(items []string, expected string) bool {
	for _, item := range items {
		if item == expected {
			return true
		}
	}
	return false
}
