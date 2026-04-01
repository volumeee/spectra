package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/internal/port"
)

type ScheduleHandler struct {
	scheduler port.Scheduler
}

func NewScheduleHandler(scheduler port.Scheduler) *ScheduleHandler {
	return &ScheduleHandler{scheduler: scheduler}
}

func (h *ScheduleHandler) Create(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var task domain.ScheduledTask
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if err := h.scheduler.Add(r.Context(), &task); err != nil {
		writeError(w, r, http.StatusBadRequest, "SCHEDULE_ERROR", err.Error())
		return
	}
	writeSuccess(w, r, task, start)
}

func (h *ScheduleHandler) List(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	tasks, err := h.scheduler.List(r.Context())
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "SCHEDULE_ERROR", err.Error())
		return
	}
	writeSuccess(w, r, tasks, start)
}

func (h *ScheduleHandler) Delete(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	id := chi.URLParam(r, "id")
	if err := h.scheduler.Remove(r.Context(), id); err != nil {
		writeError(w, r, http.StatusInternalServerError, "SCHEDULE_ERROR", err.Error())
		return
	}
	writeSuccess(w, r, map[string]string{"deleted": id}, start)
}
