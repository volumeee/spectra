package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/spectra-browser/spectra/internal/api/middleware"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
	Meta    *APIMeta    `json:"meta,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type APIMeta struct {
	RequestID  string `json:"request_id"`
	DurationMs int64  `json:"duration_ms"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeSuccess(w http.ResponseWriter, r *http.Request, data interface{}, start time.Time) {
	writeJSON(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    data,
		Meta: &APIMeta{
			RequestID:  middleware.GetRequestID(r.Context()),
			DurationMs: time.Since(start).Milliseconds(),
		},
	})
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, APIResponse{
		Success: false,
		Error:   &APIError{Code: code, Message: message},
		Meta:    &APIMeta{RequestID: middleware.GetRequestID(r.Context())},
	})
}
