package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type externalNavigationAuthenticator struct {
	mustChangePassword bool
}

func (a externalNavigationAuthenticator) AuthenticateAccess(_ context.Context, token string) (Principal, error) {
	if token != "valid" {
		return Principal{}, errors.New("expired or missing session")
	}
	return Principal{UserID: uuid.New(), SessionID: uuid.New(), MustChangePassword: a.mustChangePassword}, nil
}
func (a externalNavigationAuthenticator) AuthenticateSession(_ context.Context, _, _ uuid.UUID) (Principal, error) {
	return Principal{}, errors.New("not used")
}
func (a externalNavigationAuthenticator) Refresh(_ context.Context, _, _ string) (string, string, Principal, error) {
	return "", "", Principal{}, errors.New("not used")
}

func TestExternalApplicationSessionContinuation(t *testing.T) {
	const destination = "/apps/photobooth?workspace_id=603bc0c1-386d-4740-9bf1-2998fabd727e"
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := PrincipalFromContext(r.Context()); !ok {
			t.Error("authorized request missing session principal")
		}
		w.WriteHeader(http.StatusNoContent)
	})
	for _, tc := range []struct {
		name      string
		path      string
		method    string
		accept    string
		dest      string
		mode      string
		token     string
		password  bool
		wantCode  int
		wantRoute string
	}{
		{name: "unauthenticated document", path: destination, method: "GET", accept: "text/html", dest: "document", wantCode: http.StatusSeeOther, wantRoute: "/auth/continue"},
		{name: "expired document cookie", path: destination, method: "GET", accept: "text/html", token: "expired", wantCode: http.StatusSeeOther, wantRoute: "/auth/continue"},
		{name: "valid session launches app", path: destination, method: "GET", accept: "text/html", token: "valid", wantCode: http.StatusNoContent},
		{name: "password change required", path: destination, method: "GET", accept: "text/html", token: "valid", password: true, wantCode: http.StatusSeeOther, wantRoute: "/change-password"},
		{name: "fetch metadata navigation", path: destination, method: "GET", mode: "navigate", wantCode: http.StatusSeeOther, wantRoute: "/auth/continue"},
		{name: "API request stays JSON 401", path: "/api/apps/photobooth/v1/bookings", method: "GET", accept: "application/json", wantCode: http.StatusUnauthorized},
		{name: "asset request stays JSON 401", path: "/apps/photobooth/assets/index.js", method: "GET", accept: "*/*", dest: "script", wantCode: http.StatusUnauthorized},
		{name: "API mutation stays JSON 401", path: "/api/apps/photobooth/v1/bookings", method: "POST", accept: "text/html", wantCode: http.StatusUnauthorized},
		{name: "ordinary auth stays JSON 401", path: "/api/v1/framework/applications", method: "GET", accept: "text/html", wantCode: http.StatusUnauthorized},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(tc.method, tc.path, nil)
			r.Header.Set("Accept", tc.accept)
			if tc.dest != "" {
				r.Header.Set("Sec-Fetch-Dest", tc.dest)
			}
			if tc.mode != "" {
				r.Header.Set("Sec-Fetch-Mode", tc.mode)
			}
			if tc.token != "" {
				r.AddCookie(&http.Cookie{Name: "apexvoid_access_token", Value: tc.token})
			}
			w := httptest.NewRecorder()
			auth := externalNavigationAuthenticator{mustChangePassword: tc.password}
			middleware := RequireExternalApplicationAuthentication(auth)
			if tc.name == "ordinary auth stays JSON 401" {
				middleware = RequireAuthentication(auth)
			}
			middleware(next).ServeHTTP(w, r)
			if w.Code != tc.wantCode {
				t.Fatalf("got HTTP %d, expected %d; response=%s", w.Code, tc.wantCode, w.Body.String())
			}
			if tc.wantRoute != "" {
				location, err := url.Parse(w.Header().Get("Location"))
				if err != nil || location.Path != tc.wantRoute || location.Query().Get("return_to") != destination {
					t.Fatalf("redirect failed to preserve the exact app/workspace: %s (%v)", w.Header().Get("Location"), err)
				}
			} else if tc.wantCode == http.StatusUnauthorized && (!strings.Contains(w.Body.String(), "UNAUTHENTICATED") || w.Header().Get("Location") != "") {
				t.Fatalf("unauthenticated API/asset request must keep JSON 401: %s", w.Body.String())
			}
		})
	}
}
