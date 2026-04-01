package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/internal/port"
)

type WebhookHandler struct {
	store port.WebhookStore
}

func NewWebhookHandler(store port.WebhookStore) *WebhookHandler {
	return &WebhookHandler{store: store}
}

func (h *WebhookHandler) Create(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var req struct {
		Event     domain.WebhookEvent `json:"event"`
		TargetURL string              `json:"target_url"`
		Secret    string              `json:"secret"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}

	sub := &domain.WebhookSubscription{
		ID:        uuid.NewString(),
		Event:     req.Event,
		TargetURL: req.TargetURL,
		Secret:    req.Secret,
		Active:    true,
		CreatedAt: time.Now(),
	}
	if err := h.store.CreateWebhook(r.Context(), sub); err != nil {
		writeError(w, r, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	writeSuccess(w, r, sub, start)
}

func (h *WebhookHandler) List(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	subs, err := h.store.ListWebhooks(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	writeSuccess(w, r, subs, start)
}

func (h *WebhookHandler) Delete(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	id := chi.URLParam(r, "id")
	if err := h.store.DeleteWebhook(r.Context(), id); err != nil {
		writeError(w, r, http.StatusInternalServerError, "STORE_ERROR", err.Error())
		return
	}
	writeSuccess(w, r, map[string]string{"deleted": id}, start)
}
