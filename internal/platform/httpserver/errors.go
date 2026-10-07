package httpserver

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Error APIErrorBody `json:"error"`
}

type APIErrorBody struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(APIError{Error: APIErrorBody{Code: code, Message: message, RequestID: RequestIDFromContext(r.Context())}})
}
