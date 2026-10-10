package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/domain"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
)

type externalWorkspaceResolverStub struct {
	lastWorkspace uuid.UUID
	fail          error
}

func (r *externalWorkspaceResolverStub) ResolveWorkspaceContext(_ context.Context, userID, workspaceID uuid.UUID) (domain.WorkspaceContext, error) {
	r.lastWorkspace = workspaceID
	return domain.WorkspaceContext{WorkspaceID: workspaceID, UserID: userID}, r.fail
}

func (r *externalWorkspaceResolverStub) ResolveDefaultWorkspaceContext(_ context.Context, userID uuid.UUID) (domain.WorkspaceContext, error) {
	return domain.WorkspaceContext{UserID: userID}, r.fail
}

func TestExternalWorkspaceLaunchSelectsAndPersistsWorkspace(t *testing.T) {
	resolver := &externalWorkspaceResolverStub{}
	selected := uuid.New()
	principal := usersapi.Principal{UserID: uuid.New()}
	handler := RequireExternalWorkspace(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		workspace, ok := WorkspaceContextFromContext(r.Context())
		if !ok || workspace.WorkspaceID != selected {
			t.Fatalf("expected selected workspace, got %+v", workspace)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/apps/photobooth/?workspace_id="+selected.String(), nil)
	request = request.WithContext(usersapi.WithPrincipal(request.Context(), principal))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("launch status %d", response.Code)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Value != selected.String() || !cookies[0].HttpOnly {
		t.Fatalf("expected secured workspace selection cookie, got %+v", cookies)
	}
	next := httptest.NewRequest(http.MethodGet, "/api/apps/photobooth/v1/orders", nil)
	next.AddCookie(cookies[0])
	next = next.WithContext(usersapi.WithPrincipal(next.Context(), principal))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, next)
	if response.Code != http.StatusNoContent || resolver.lastWorkspace != selected {
		t.Fatalf("workspace was not carried into app API request: %d", response.Code)
	}
}

func TestExternalWorkspaceLaunchChecksMembershipAndRejectsInvalidSelector(t *testing.T) {
	resolver := &externalWorkspaceResolverStub{fail: errors.New("membership suspended")}
	handler := RequireExternalWorkspace(resolver)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("request must not be routed")
	}))
	principal := usersapi.Principal{UserID: uuid.New()}
	request := httptest.NewRequest(http.MethodGet, "/apps/photobooth/?workspace_id="+uuid.NewString(), nil)
	request = request.WithContext(usersapi.WithPrincipal(request.Context(), principal))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("expected membership denial, got %d", response.Code)
	}
	request = httptest.NewRequest(http.MethodGet, "/apps/photobooth/?workspace_id=invalid", nil)
	request = request.WithContext(usersapi.WithPrincipal(request.Context(), principal))
	response = httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid selector rejection, got %d", response.Code)
	}
}
