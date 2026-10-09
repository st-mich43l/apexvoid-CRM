package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/module"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/runtime"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/database"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/health"
)

type App struct {
	Environment  string
	Logger       *slog.Logger
	Database     *pgxpool.Pool
	Transactions *database.TxManager
	Health       *health.Checker
	Runtime      *runtime.Runtime
}

func (a *App) RegisterRoutes(router chi.Router) error {
	router.Get("/health", a.healthHandler)
	router.Get("/ready", a.readyHandler)
	api := chi.NewRouter()
	api.Get("/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"service": "apexvoid-crm", "version": "v1"})
	})
	if err := a.Runtime.Modules.RegisterRoutes(chiRoutes{router: api}); err != nil {
		return fmt.Errorf("register module routes: %w", err)
	}
	if err := a.Runtime.Modules.RegisterPublicRoutes(chiRoutes{router: router}); err != nil {
		return fmt.Errorf("register public module routes: %w", err)
	}
	router.Mount("/api/v1", api)
	return nil
}

type chiRoutes struct{ router chi.Router }

func (r chiRoutes) Get(path string, handler http.HandlerFunc)    { r.router.Get(path, handler) }
func (r chiRoutes) Post(path string, handler http.HandlerFunc)   { r.router.Post(path, handler) }
func (r chiRoutes) Put(path string, handler http.HandlerFunc)    { r.router.Put(path, handler) }
func (r chiRoutes) Patch(path string, handler http.HandlerFunc)  { r.router.Patch(path, handler) }
func (r chiRoutes) Delete(path string, handler http.HandlerFunc) { r.router.Delete(path, handler) }
func (r chiRoutes) With(middleware ...func(http.Handler) http.Handler) module.RouteRegistry {
	return chiRoutes{router: r.router.With(middleware...)}
}

func (a *App) healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "apexvoid-crm", "environment": a.Environment})
}

func (a *App) readyHandler(w http.ResponseWriter, r *http.Request) {
	status, err := a.Health.Check(r.Context())
	response := map[string]interface{}{"status": "ready", "dependencies": status}
	if err != nil {
		response["status"] = "not_ready"
		writeJSON(w, http.StatusServiceUnavailable, response)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func (a *App) Close(ctx context.Context) error {
	if a.Database != nil {
		a.Database.Close()
	}
	return nil
}
