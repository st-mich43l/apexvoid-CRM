package runtime

import (
	"context"
	"strings"
	"testing"

	frameworkapplication "github.com/st-mich43l/apexvoid-CRM/internal/framework/application"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/capability"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/permission"
)

type applicationFixtureModule struct{ invalidPermission bool }

func (applicationFixtureModule) Descriptor() module.Descriptor {
	return module.Descriptor{Name: "fixture", DisplayName: "Framework Fixture", Version: "1.0.0"}
}

func (m applicationFixtureModule) Register(ctx *module.Context) error {
	if err := ctx.Capabilities.Register(capability.Definition{Name: "fixture.auditable", Module: "fixture", DisplayName: "Auditable"}); err != nil {
		return err
	}
	if err := ctx.Permissions.Register(permission.Definition{Name: "fixture.record.read", Module: "fixture", DisplayName: "Read fixture records"}); err != nil {
		return err
	}
	requiredPermission := "fixture.record.read"
	if m.invalidPermission {
		requiredPermission = "fixture.record.manage"
	}
	return ctx.Applications.Register(frameworkapplication.Descriptor{
		ID: "fixture", DisplayName: "Framework Fixture", Description: "A non-business framework registration fixture.", Version: "1.0.0", APIContractVersion: "v1",
		ModuleDependencies: []string{"fixture"}, RequiredPermissions: []string{requiredPermission}, RequiredCapabilities: []string{"fixture.auditable"},
		Frontend: frameworkapplication.Frontend{EntryRoute: "/fixture", NavigationID: "fixture-home"},
		Access:   frameworkapplication.Access{Entry: frameworkapplication.PermissionPolicy{Match: frameworkapplication.PermissionMatchAll, Permissions: []string{requiredPermission}}},
	})
}

func TestRuntimeValidatesAndDiscoversCompiledApplicationFixture(t *testing.T) {
	framework := New()
	if err := framework.Modules.Register(applicationFixtureModule{}); err != nil {
		t.Fatal(err)
	}
	if err := framework.Initialize(context.Background()); err != nil {
		t.Fatal(err)
	}
	applications := framework.Metadata.Snapshot().Applications
	if len(applications) != 1 || applications[0].ID != "fixture" || applications[0].Frontend.EntryRoute != "/fixture" {
		t.Fatalf("unexpected application discovery result: %+v", applications)
	}
}

func TestRuntimeRejectsApplicationWithUnknownRegisteredContract(t *testing.T) {
	framework := New()
	if err := framework.Modules.Register(applicationFixtureModule{invalidPermission: true}); err != nil {
		t.Fatal(err)
	}
	err := framework.Initialize(context.Background())
	if err == nil || !strings.Contains(err.Error(), "unregistered permission") {
		t.Fatalf("expected unknown application permission error, got %v", err)
	}
}
