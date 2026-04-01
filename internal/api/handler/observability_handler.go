package handler

import (
	"net/http"
	"strconv"

	"github.com/spectra-browser/spectra/internal/port"
)

// PressureHandler returns system load status — useful for load balancers.
// Returns 503 when overloaded, 200 when healthy.
type PressureHandler struct {
	monitor port.SystemMonitor
}

func NewPressureHandler(monitor port.SystemMonitor) *PressureHandler {
	return &PressureHandler{monitor: monitor}
}

func (h *PressureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stats := h.monitor.Stats()
	status := http.StatusOK
	if stats.Overloaded {
		status = http.StatusServiceUnavailable
	}
	writeJSON(w, status, stats)
}

// MetricsHandler returns operational metrics snapshot.
type MetricsHandler struct {
	metrics port.MetricsCollector
}

func NewMetricsHandler(metrics port.MetricsCollector) *MetricsHandler {
	return &MetricsHandler{metrics: metrics}
}

func (h *MetricsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.metrics.Snapshot())
}

// JobsHandler returns job history.
type JobsHandler struct {
	jobs port.JobStore
}

func NewJobsHandler(jobs port.JobStore) *JobsHandler {
	return &JobsHandler{jobs: jobs}
}

func (h *JobsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.jobs == nil {
		writeJSON(w, http.StatusOK, map[string]interface{}{"jobs": []interface{}{}, "count": 0})
		return
	}
	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	records, err := h.jobs.ListJobs(r.Context(), limit)
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "STORAGE_ERROR", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"jobs":  records,
		"count": len(records),
	})
}
