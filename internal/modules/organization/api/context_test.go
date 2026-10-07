package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/organization/domain"
	usersapi "github.com/st-mich43l/apexvoid-CRM/internal/modules/users/api"
)

type resolverStub struct {
	workspace domain.WorkspaceContext
	err       error
}

func (r resolverStub) ResolveWorkspaceContext(context.Context, uuid.UUID, uuid.UUID) (domain.WorkspaceContext, error) {
	return r.workspace, r.err
}
func (r resolverStub) ResolveDefaultWorkspaceContext(context.Context, uuid.UUID) (domain.WorkspaceContext, error) {
	return r.workspace, r.err
}

func TestRequireWorkspaceStoresValidatedContext(t *testing.T) {
	workspaceID := uuid.New()
	userID := uuid.New()
	resolver := resolverStub{workspace: domain.WorkspaceContext{WorkspaceID: workspaceID, UserID: userID}}
	handler := RequireWorkspace(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		contextValue, ok := WorkspaceContextFromContext(r.Context())
		if !ok || contextValue.WorkspaceID != workspaceID || contextValue.UserID != userID {
			t.Fatalf("workspace context was not stored: %+v", contextValue)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspace", nil)
	request.Header.Set("X-ApexVoid-Workspace", workspaceID.String())
	request = request.WithContext(usersapi.WithPrincipal(request.Context(), usersapi.Principal{UserID: userID}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected valid workspace context, got %d", recorder.Code)
	}
}

func TestRequireWorkspaceRejectsResolverFailure(t *testing.T) {
	handler := RequireWorkspace(resolverStub{err: domain.ErrMembershipSuspended})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("handler should not run") }))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspace", nil).WithContext(usersapi.WithPrincipal(context.Background(), usersapi.Principal{UserID: uuid.New()}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected membership failure to be forbidden, got %d", recorder.Code)
	}
}
