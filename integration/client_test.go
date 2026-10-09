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

func TestClientRefusesRedirectsBeforeSendingCredentialToAnotherService(t *testing.T) {
	var redirected atomic.Bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirected.Store(true)
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	platform := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(ServiceCredentialHeader) != "private-service-credential" {
			t.Error("missing service credential on first request")
		}
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))
	defer platform.Close()

	customClient := &http.Client{}
	client, err := NewClient(Config{
		PlatformURL: platform.URL, ApplicationID: "reports",
		ServiceCredential: "private-service-credential", HTTPClient: customClient,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Availability(context.Background(), "workspace")
	apiErr, ok := err.(*Error)
	if !ok || apiErr.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("redirect should fail without being followed: %v", err)
	}
	if redirected.Load() {
		t.Fatal("service credential was exposed by following an upstream redirect")
	}
	if customClient.CheckRedirect != nil {
		t.Fatal("the caller's HTTP client was mutated")
	}
}

func TestClientRejectsUnsafePlatformURLs(t *testing.T) {
	for _, endpoint := range []string{
		"http://user:password@localhost:8080",
		"http://localhost:8080/other-service",
		"http://localhost:8080?redirect=untrusted",
		"http://localhost:8080/#fragment",
		"file:///tmp/platform",
	} {
		t.Run(endpoint, func(t *testing.T) {
			if _, err := NewClient(Config{PlatformURL: endpoint, ApplicationID: "reports", ServiceCredential: "secret"}); err == nil {
				t.Fatal("unsafe platform URL must be rejected")
			}
		})
	}
}
