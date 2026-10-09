package app

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/runtime"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core"
	organizationdomain "github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/domain"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
)

type frameworkAuthStub struct{}

func (frameworkAuthStub) AuthenticateAccess(context.Context, string) (usersapi.Principal, error) {
	return usersapi.Principal{}, errors.New("not used")
}

func TestHealthReportsSafeRuntimeEnvironment(t *testing.T) {
	application := &App{Environment: "production"}
	recorder := httptest.NewRecorder()
	application.healthHandler(recorder, httptest.NewRequest(http.MethodGet, "/health", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), `"environment":"production"`) {
		t.Fatalf("health response did not report runtime environment: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
func (frameworkAuthStub) Refresh(context.Context, string, string) (string, string, usersapi.Principal, error) {
	return "", "", usersapi.Principal{}, errors.New("not used")
}

type frameworkWorkspaceStub struct{}

func (frameworkWorkspaceStub) ResolveWorkspaceContext(context.Context, uuid.UUID, uuid.UUID) (organizationdomain.WorkspaceContext, error) {
	return organizationdomain.WorkspaceContext{}, errors.New("not used")
}
func (frameworkWorkspaceStub) ResolveDefaultWorkspaceContext(context.Context, uuid.UUID) (organizationdomain.WorkspaceContext, error) {
	return organizationdomain.WorkspaceContext{}, errors.New("not used")
}

type frameworkAccessStub struct{}

func (frameworkAccessStub) Can(context.Context, uuid.UUID, string) (bool, error) { return false, nil }
func (frameworkAccessStub) CanInWorkspace(context.Context, uuid.UUID, uuid.UUID, string) (bool, error) {
	return false, nil
}

func TestFrameworkDiscoveryEndpoints(t *testing.T) {
	framework := runtime.New()
	if err := framework.Modules.Register(core.New(core.Dependencies{Metadata: framework.Metadata, Authenticator: frameworkAuthStub{}, Workspace: frameworkWorkspaceStub{}, Access: frameworkAccessStub{}})); err != nil {
		t.Fatal(err)
	}
	if err := framework.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	application := &App{Runtime: framework}
	router := chi.NewRouter()
	if err := application.RegisterRoutes(router); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/v1/framework/modules", "/api/v1/framework/entities", "/api/v1/framework/permissions", "/api/v1/framework/entities/core.example"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, recorder.Code)
		}
	}
	applications := httptest.NewRecorder()
	router.ServeHTTP(applications, httptest.NewRequest(http.MethodGet, "/api/v1/framework/applications", nil))
	if applications.Code != http.StatusUnauthorized {
		t.Fatalf("application discovery must require authentication, got %d", applications.Code)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/framework/entities/missing.entity", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown entity returned %d", recorder.Code)
	}
}
