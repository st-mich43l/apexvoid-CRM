// External-module is a deliberately tiny Docker fixture used to exercise the
// ApexVoid external integration boundary. It is not a product application.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/st-mich43l/apexvoid-CRM/integration"
)

func main() {
	address := os.Getenv("FIXTURE_ADDR")
	if address == "" {
		address = ":8090"
	}
	client, err := integration.NewClient(integration.Config{PlatformURL: fixtureEnv("APEXVOID_URL", "http://backend:6868"), ApplicationID: fixtureEnv("APEXVOID_APPLICATION_ID", "reports"), ServiceCredential: fixtureEnv("APEXVOID_SERVICE_CREDENTIAL", ""), Timeout: 3 * time.Second, RetryAttempts: 1})
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/records", func(w http.ResponseWriter, r *http.Request) {
		assertion, err := integration.IdentityAssertionFromRequest(r)
		if err != nil {
			http.Error(w, "gateway identity assertion is required", http.StatusUnauthorized)
			return
		}
		decision, err := client.Introspect(r.Context(), assertion, fixtureEnv("APEXVOID_FIXTURE_PERMISSION", "reports.report.read"))
		if err != nil || !decision.Allowed {
			http.Error(w, "permission denied", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"fixture": "authorized", "user_id": decision.UserID, "workspace_id": decision.WorkspaceID})
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<!doctype html><title>External module fixture</title><main>external module fixture</main>"))
	})
	_ = http.ListenAndServe(address, mux)
}

func fixtureEnv(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
