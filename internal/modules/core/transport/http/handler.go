package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/st-mich43l/apexvoid-CRM/internal/platform/httpserver"
)

func (h *Handler) applications(w http.ResponseWriter, r *http.Request) {
	items := h.service.Applications(r.Context())
	result := make([]applicationResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toApplicationResponse(item))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) modules(w http.ResponseWriter, r *http.Request) {
	items := h.service.Modules(r.Context())
	result := make([]moduleResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toModuleResponse(item))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) entities(w http.ResponseWriter, r *http.Request) {
	items := h.service.Entities(r.Context())
	result := make([]entityResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toEntityResponse(item))
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) entity(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.Entity(r.Context(), chi.URLParam(r, "entity"))
	if err != nil {
		httpserver.WriteApplicationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toEntityResponse(item))
}

func (h *Handler) permissions(w http.ResponseWriter, r *http.Request) {
	items := h.service.Permissions(r.Context())
	result := make([]permissionResponse, 0, len(items))
	for _, item := range items {
		result = append(result, toPermissionResponse(item))
	}
	writeJSON(w, http.StatusOK, result)
}
