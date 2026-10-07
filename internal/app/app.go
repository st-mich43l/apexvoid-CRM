package app

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/st-mich43l/apexvoid-CRM/internal/framework/metadata"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/health"
)

type App struct {
	Logger   *slog.Logger
	Database *pgxpool.Pool
	Health   *health.Checker
	Metadata *metadata.Registry
}

func (a *App) RegisterRoutes(router chi.Router) {
	router.Get("/health", a.healthHandler)
	router.Get("/ready", a.readyHandler)
	router.Route("/api/v1", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, map[string]string{"service": "apexvoid-crm", "version": "v1"})
		})
		a.registerFrameworkRoutes(r)
	})
}

func (a *App) healthHandler(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "apexvoid-crm"})
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
