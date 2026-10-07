package httpserver

import (
	"encoding/json"
	"errors"
	"net/http"

	frameworkerrors "github.com/st-mich43l/apexvoid-CRM/internal/framework/errors"
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

func WriteApplicationError(w http.ResponseWriter, r *http.Request, err error) {
	var frameworkError *frameworkerrors.Error
	if !errors.As(err, &frameworkError) {
		WriteError(w, r, http.StatusInternalServerError, "INTERNAL_ERROR", "An unexpected error occurred")
		return
	}
	status, code := http.StatusInternalServerError, "INTERNAL_ERROR"
	switch frameworkError.Kind {
	case frameworkerrors.Validation:
		status, code = http.StatusBadRequest, "VALIDATION_ERROR"
	case frameworkerrors.NotFound:
		status, code = http.StatusNotFound, "NOT_FOUND"
	case frameworkerrors.Conflict:
		status, code = http.StatusConflict, "CONFLICT"
	case frameworkerrors.Forbidden:
		status, code = http.StatusForbidden, "FORBIDDEN"
	}
	message := frameworkError.Message
	if frameworkError.Kind == frameworkerrors.Internal {
		message = "An unexpected error occurred"
	}
	WriteError(w, r, status, code, message)
}
