package application

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

func upgradedManifestForTest(t *testing.T) (ExternalManifest, ExternalManifest) {
	t.Helper()
	previous := validManifestForTest()
	raw, err := json.Marshal(previous)
	if err != nil {
		t.Fatal(err)
	}
	var next ExternalManifest
	if err := json.Unmarshal(raw, &next); err != nil {
		t.Fatal(err)
	}
	next.Application.Version = "1.1.0"
	next.Database.MigrationBundleVersion = "1.1.0"
	next.Permissions = append(next.Permissions, ExternalPermission{
		Name: "reports.export.read", DisplayName: "Export reports", Scope: permission.ScopeWorkspace,
	})
	next.Migrations = append(next.Migrations, ExternalMigration{
		Version: 2, Path: "/.well-known/apexvoid/migrations/002-export.sql", SHA256: strings.Repeat("b", 64),
	})
	return previous, next
}

func TestUpgradeAcceptsOnlyAdditiveChanges(t *testing.T) {
	previous, next := upgradedManifestForTest(t)
	added, migrations, err := validateUpgrade(previous, next)
	if err != nil {
		t.Fatal(err)
	}
	if len(added) != 1 || added[0].Name != "reports.export.read" {
		t.Fatalf("added permissions: %#v", added)
	}
	if len(migrations) != 1 || migrations[0].Version != 2 {
		t.Fatalf("pending migrations: %#v", migrations)
	}
}

func TestUpgradeRejectsRewrittenMigrationsAndScopeChanges(t *testing.T) {
	previous, next := upgradedManifestForTest(t)
	next.Migrations[0].SHA256 = strings.Repeat("c", 64)
	if _, _, err := validateUpgrade(previous, next); err == nil {
		t.Fatal("approved a changed migration checksum")
	}
	_, next = upgradedManifestForTest(t)
	next.Permissions[0].Scope = permission.ScopePlatform
	if _, _, err := validateUpgrade(previous, next); err == nil {
		t.Fatal("approved a changed permission scope")
	}
	_, next = upgradedManifestForTest(t)
	next.Permissions = next.Permissions[1:]
	if _, _, err := validateUpgrade(previous, next); err == nil {
		t.Fatal("approved a removed permission")
	}
}

func TestUpgradeRequiresIncreasingStableVersions(t *testing.T) {
	previous, next := upgradedManifestForTest(t)
	next.Application.Version = "1.0.0"
	if _, _, err := validateUpgrade(previous, next); err == nil {
		t.Fatal("approved a non-increasing release")
	}
	_, next = upgradedManifestForTest(t)
	next.Database.MigrationBundleVersion = "1.0.0"
	if _, _, err := validateUpgrade(previous, next); err == nil {
		t.Fatal("approved new migration without new bundle release")
	}
	if _, err := compareUpgradeVersion("1.0.0", "0.9.9"); err != nil {
		t.Fatal(err)
	}
	if order, err := compareUpgradeVersion("1.10.0", "1.9.9"); err != nil || order <= 0 {
		t.Fatalf("version comparison = %d, %v", order, err)
	}
	if _, err := compareUpgradeVersion("1.0.0-rc.1", "1.1.0"); err == nil {
		t.Fatal("accepted unsupported prerelease comparison")
	}
}

func TestUpgradeRejectsDatabaseOwnershipAndRouteChanges(t *testing.T) {
	previous, next := upgradedManifestForTest(t)
	next.Database.Name = "apexvoid_other"
	if _, _, err := validateUpgrade(previous, next); err == nil {
		t.Fatal("approved database takeover")
	}
	_, next = upgradedManifestForTest(t)
	next.Service.FrontendRoute = "/apps/reports/admin"
	if _, _, err := validateUpgrade(previous, next); err == nil {
		t.Fatal("approved frontend route replacement")
	}
}
