package httpserver

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	frameworkerrors "github.com/st-mich43l/apexvoid-CRM/internal/framework/errors"
)

func TestWriteErrorUsesStandardEnvelope(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	req = req.WithContext(context.WithValue(req.Context(), requestIDKey{}, "req-test"))
	recorder := httptest.NewRecorder()
	WriteError(recorder, req, 400, "BAD_REQUEST", "invalid request")
	var response APIError
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != "BAD_REQUEST" || response.Error.RequestID != "req-test" {
		t.Fatalf("unexpected error: %+v", response)
	}
}

func TestWriteApplicationErrorMapsWithoutExposingInternalDetails(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()
	WriteApplicationError(recorder, req, frameworkerrors.InternalError("database failed", context.DeadlineExceeded))
	if recorder.Code != 500 || strings.Contains(recorder.Body.String(), "database failed") {
		t.Fatalf("unexpected internal error response: %s", recorder.Body.String())
	}
}
