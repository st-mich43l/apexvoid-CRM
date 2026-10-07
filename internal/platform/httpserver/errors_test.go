package httpserver

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"
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
