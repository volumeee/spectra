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

// SessionHandler manages long-lived browser sessions.
type SessionHandler struct {
	sessions port.SessionManager
}

func NewSessionHandler(sessions port.SessionManager) *SessionHandler {
	return &SessionHandler{sessions: sessions}
}

func (h *SessionHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.sessions == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "sessions require sqlite storage (set storage.driver=sqlite)")
		return
	}
	var body struct {
		ProfileID string `json:"profile_id"`
		TTL       int    `json:"ttl_seconds"`
	}
	json.NewDecoder(r.Body).Decode(&body)

	sess, err := h.sessions.CreateSession(r.Context(), body.ProfileID, body.TTL)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "SESSION_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *SessionHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.sessions == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "sessions require sqlite storage")
		return
	}
	sess, err := h.sessions.GetSession(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeError(w, r, http.StatusNotFound, "SESSION_NOT_FOUND", "session not found")
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *SessionHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.sessions == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"sessions": []interface{}{}, "count": 0})
		return
	}
	list, err := h.sessions.ListSessions(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "SESSION_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"sessions": list, "count": len(list)})
}

func (h *SessionHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.sessions == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "sessions require sqlite storage")
		return
	}
	if err := h.sessions.DeleteSession(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeError(w, r, http.StatusInternalServerError, "SESSION_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ProfileHandler manages browser profiles (fingerprint identities).
type ProfileHandler struct {
	profiles port.ProfileStore
}

func NewProfileHandler(profiles port.ProfileStore) *ProfileHandler {
	return &ProfileHandler{profiles: profiles}
}

func (h *ProfileHandler) Create(w http.ResponseWriter, r *http.Request) {
	if h.profiles == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "profiles require sqlite storage (set storage.driver=sqlite)")
		return
	}
	var p domain.BrowserProfile
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	p.ID = uuid.NewString()
	p.CreatedAt = time.Now()
	if err := h.profiles.CreateProfile(r.Context(), &p); err != nil {
		writeError(w, r, http.StatusInternalServerError, "PROFILE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *ProfileHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.profiles == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"profiles": []interface{}{}, "count": 0})
		return
	}
	list, err := h.profiles.ListProfiles(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "PROFILE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"profiles": list, "count": len(list)})
}

func (h *ProfileHandler) Delete(w http.ResponseWriter, r *http.Request) {
	if h.profiles == nil {
		writeError(w, r, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "profiles require sqlite storage")
		return
	}
	if err := h.profiles.DeleteProfile(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeError(w, r, http.StatusInternalServerError, "PROFILE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
