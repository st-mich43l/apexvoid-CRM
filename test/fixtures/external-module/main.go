// External-module is a deliberately tiny Docker fixture used to exercise the
// ApexVoid external integration boundary. It is not a product application.
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	address := os.Getenv("FIXTURE_ADDR")
	if address == "" {
		address = ":8090"
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	mux.HandleFunc("/records", func(w http.ResponseWriter, r *http.Request) {
		assertion := r.Header.Get("X-ApexVoid-Identity-Assertion")
		if assertion == "" {
			http.Error(w, "gateway identity assertion is required", http.StatusUnauthorized)
			return
		}
		payload, _ := json.Marshal(map[string]string{"identity_assertion": assertion, "permission": fixtureEnv("APEXVOID_FIXTURE_PERMISSION", "reports.report.read")})
		request, err := http.NewRequestWithContext(r.Context(), http.MethodPost, strings.TrimRight(fixtureEnv("APEXVOID_URL", "http://backend:6868"), "/")+"/api/v1/integrations/v1/session/introspect", bytes.NewReader(payload))
		if err != nil {
			http.Error(w, "could not create introspection request", http.StatusBadGateway)
			return
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-ApexVoid-Application-ID", fixtureEnv("APEXVOID_APPLICATION_ID", "reports"))
		request.Header.Set("X-ApexVoid-Service-Credential", fixtureEnv("APEXVOID_SERVICE_CREDENTIAL", ""))
		response, err := (&http.Client{Timeout: 3 * time.Second}).Do(request)
		if err != nil {
			http.Error(w, "introspection unavailable", http.StatusBadGateway)
			return
		}
		defer response.Body.Close()
		var decision struct {
			UserID      string `json:"user_id"`
			WorkspaceID string `json:"workspace_id"`
			Allowed     bool   `json:"allowed"`
		}
		if response.StatusCode != http.StatusOK || json.NewDecoder(response.Body).Decode(&decision) != nil || !decision.Allowed {
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
