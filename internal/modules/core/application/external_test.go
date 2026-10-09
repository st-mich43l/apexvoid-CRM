package application

import (
	"testing"

	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

func externalFixture() ExternalApplication {
	return ExternalApplication{ID: "reports", DisplayName: "Reports", Description: "Reporting service", Version: "1.0.0", APIContractVersion: "v1", ServiceIdentity: "reports-service", ServiceEndpoint: "http://reports:8090", HealthEndpoint: "http://reports:8090/health", FrontendRoute: "/apps/reports", Access: frameworkapplication.PermissionPolicy{Match: frameworkapplication.PermissionMatchAll, Permissions: []string{"reports.report.read"}}, Permissions: []ExternalPermission{{Name: "reports.report.read", DisplayName: "Read reports", Scope: permission.ScopeWorkspace}}}
}

func TestExternalContractRequiresOwnedPermissionsAndGatewayRoute(t *testing.T) {
	app := externalFixture()
	if err := validateExternal(&app); err != nil {
		t.Fatal(err)
	}
	app.Permissions[0].Name = "other.report.read"
	if err := validateExternal(&app); err == nil {
		t.Fatal("foreign permission must be rejected")
	}
	app = externalFixture()
	app.FrontendRoute = "/reports"
	if err := validateExternal(&app); err == nil {
		t.Fatal("external frontend must use the gateway route")
	}
}

func TestExternalDescriptorKeepsSettingsUnderTheSameAccessPolicy(t *testing.T) {
	app := externalFixture()
	app.SettingsRoute = "/apps/reports/settings"
	descriptor := (&ExternalStore{}).Descriptor(app)
	if descriptor.Deployment != frameworkapplication.DeploymentExternal || descriptor.Access.Settings == nil || descriptor.Access.Settings.Permissions[0] != "reports.report.read" {
		t.Fatalf("unexpected external descriptor: %#v", descriptor)
	}
}
