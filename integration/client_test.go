package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestClientUsesServiceHeadersAndTypedResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(ApplicationIDHeader) != "reports" || r.Header.Get(ServiceCredentialHeader) != "secret" {
			t.Fatal("service authentication headers were not supplied")
		}
		if r.URL.Path != "/api/v1/integrations/v1/session/introspect" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(Decision{UserID: "user", WorkspaceID: "workspace", Permission: "reports.read", Allowed: true})
	}))
	defer server.Close()
	client, err := NewClient(Config{PlatformURL: server.URL, ApplicationID: "reports", ServiceCredential: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	decision, err := client.Introspect(context.Background(), "opaque-assertion", "reports.read")
	if err != nil || !decision.Allowed {
		t.Fatalf("unexpected result: %#v %v", decision, err)
	}
}

func TestClientReturnsSafeTypedAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"FORBIDDEN","message":"Denied","request_id":"req-1"}}`))
	}))
	defer server.Close()
	client, err := NewClient(Config{PlatformURL: server.URL, ApplicationID: "reports", ServiceCredential: "never-log-this"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Availability(context.Background(), "workspace")
	apiErr, ok := err.(*Error)
	if !ok || apiErr.Code != "FORBIDDEN" || apiErr.RequestID != "req-1" || apiErr.Error() == "never-log-this" {
		t.Fatalf("unexpected safe error: %#v", err)
	}
}

func TestIdentityAssertionFromRequest(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(IdentityAssertionHeader, "assertion")
	assertion, err := IdentityAssertionFromRequest(request)
	if err != nil || assertion != "assertion" {
		t.Fatalf("unexpected assertion %q: %v", assertion, err)
	}
}

func TestClientRetriesOnlyTransientFailures(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(w).Encode(Availability{ApplicationID: "reports", WorkspaceID: "workspace", Enabled: true})
	}))
	defer server.Close()
	client, err := NewClient(Config{PlatformURL: server.URL, ApplicationID: "reports", ServiceCredential: "secret", RetryAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	availability, err := client.Availability(context.Background(), "workspace")
	if err != nil || !availability.Enabled || attempts.Load() != 2 {
		t.Fatalf("unexpected retry result %#v %v attempts=%d", availability, err, attempts.Load())
	}
}
