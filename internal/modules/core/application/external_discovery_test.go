package application

import (
	"testing"

	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

func discoveryManifest(version, bundle string, migrations []ExternalMigration) ExternalManifest {
	return ExternalManifest{
		ManifestVersion: SupportedExternalContractVersion,
		Application:     ManifestApplication{ID: "photobooth", DisplayName: "ApexVoid Photobooth", Version: version, APIContractVersion: SupportedExternalContractVersion},
		Service:         ManifestService{Identity: "photobooth-service", HealthPath: "/health", EnrollmentPath: "/enroll", FrontendRoute: "/apps/photobooth", APIRoute: "/api"},
		Database:        ManifestDatabase{Name: "apexvoid_photobooth", Schema: "photobooth", Role: "apexvoid_photobooth", MigrationBundleVersion: bundle},
		Permissions:     []ExternalPermission{{Name: "photobooth.booking.read", DisplayName: "Read bookings", Scope: permission.ScopeWorkspace}},
		Access:          frameworkapplication.PermissionPolicy{Match: frameworkapplication.PermissionMatchAll, Permissions: []string{"photobooth.booking.read"}},
		Migrations:      migrations,
	}
}

func TestUpdateDiscoveryStatusReportsOnlyValidNewVersions(t *testing.T) {
	installed := discoveryManifest("0.1.0", "0.1.0", nil)
	updated := discoveryManifest("0.2.0", "0.2.0", []ExternalMigration{{Version: 1, Path: "/.well-known/apexvoid/migrations/001.sql", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}})
	available, err := updateDiscoveryStatus(installed, updated)
	if err != nil || !available {
		t.Fatalf("expected valid update to be available, available=%v err=%v", available, err)
	}

	available, err = updateDiscoveryStatus(updated, updated)
	if err != nil || available {
		t.Fatalf("expected equal versions to be up to date, available=%v err=%v", available, err)
	}

	invalid := discoveryManifest("0.3.0", "0.3.0", nil)
	invalid.Application.APIContractVersion = "v2"
	available, err = updateDiscoveryStatus(installed, invalid)
	if !available || err == nil {
		t.Fatalf("expected newer invalid manifest to remain visible with a diagnostic, available=%v err=%v", available, err)
	}
}
