package application

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

func validManifestForTest() ExternalManifest {
	return ExternalManifest{
		ManifestVersion: "v1",
		Application:     ManifestApplication{ID: "reports", DisplayName: "Reports", Description: "Reports", Version: "1.0.0", APIContractVersion: "v1"},
		Service:         ManifestService{Identity: "reports-service", HealthPath: "/health", EnrollmentPath: "/.well-known/apexvoid/enroll", FrontendRoute: "/apps/reports", APIRoute: "/api"},
		Database:        ManifestDatabase{Name: "apexvoid_reports", Schema: "reports", Role: "apexvoid_reports", MigrationBundleVersion: "1.0.0"},
		Permissions:     []ExternalPermission{{Name: "reports.report.read", DisplayName: "Read reports", Scope: permission.ScopeWorkspace}},
		Access:          application.PermissionPolicy{Match: application.PermissionMatchAll, Permissions: []string{"reports.report.read"}},
		Migrations:      []ExternalMigration{{Version: 1, Path: "/.well-known/apexvoid/migrations/001-init.sql", SHA256: strings.Repeat("a", 64)}},
	}
}

func TestValidateManifestAcceptsPinnedContract(t *testing.T) {
	if err := validateManifest(validManifestForTest()); err != nil {
		t.Fatalf("validateManifest() error = %v", err)
	}
}

func TestDecodeManifestRejectsUnknownFields(t *testing.T) {
	raw, err := json.Marshal(struct {
		ExternalManifest
		Unexpected string `json:"unexpected"`
	}{ExternalManifest: validManifestForTest(), Unexpected: "nope"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = decodeManifest(raw); err == nil {
		t.Fatal("decodeManifest() accepted an unknown field")
	}
}

func TestValidateManifestRejectsTraversalAndUnpinnedMigration(t *testing.T) {
	manifest := validManifestForTest()
	manifest.Migrations[0].Path = "/.well-known/apexvoid/migrations/../secret.sql"
	if err := validateManifest(manifest); err == nil {
		t.Fatal("accepted migration traversal")
	}
	manifest = validManifestForTest()
	manifest.Migrations[0].SHA256 = "not-a-checksum"
	if err := validateManifest(manifest); err == nil {
		t.Fatal("accepted an unpinned migration")
	}
}

func TestMigrationPolicyRejectsPrivilegedSQL(t *testing.T) {
	for _, sql := range []string{"DROP DATABASE app", "CREATE ROLE app", "GRANT ALL ON TABLE users TO public", "COPY data FROM PROGRAM 'curl example.com'"} {
		if !migrationForbiddenPattern.MatchString(sql) {
			t.Fatalf("policy accepted %q", sql)
		}
	}
	if migrationForbiddenPattern.MatchString("CREATE TABLE reports (id uuid primary key)") {
		t.Fatal("policy rejected an ordinary table")
	}
}
