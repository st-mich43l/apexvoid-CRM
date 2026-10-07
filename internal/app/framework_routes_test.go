package app

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/runtime"
	"github.com/st-mich43l/apexvoid-CRM/internal/modules/core"
)

func TestFrameworkDiscoveryEndpoints(t *testing.T) {
	framework := runtime.New()
	if err := framework.Modules.Register(core.New()); err != nil {
		t.Fatal(err)
	}
	if err := framework.Initialize(t.Context()); err != nil {
		t.Fatal(err)
	}
	application := &App{Metadata: framework.Metadata}
	router := chi.NewRouter()
	application.RegisterRoutes(router)
	for _, path := range []string{"/api/v1/framework/modules", "/api/v1/framework/entities", "/api/v1/framework/permissions", "/api/v1/framework/entities/core.example"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, recorder.Code)
		}
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/framework/entities/missing.entity", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unknown entity returned %d", recorder.Code)
	}
}
