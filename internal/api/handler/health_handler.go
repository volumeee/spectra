package handler

import (
	"net/http"

	"github.com/spectra-browser/spectra/internal/port"
)

type HealthHandler struct {
	plugins port.PluginManager
	pool    port.BrowserPool
	queue   port.JobQueue
}

func NewHealthHandler(plugins port.PluginManager, pool port.BrowserPool, queue port.JobQueue) *HealthHandler {
	return &HealthHandler{plugins: plugins, pool: pool, queue: queue}
}

func (h *HealthHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	plugins := h.plugins.List()
	resp := map[string]interface{}{
		"status":  "ok",
		"plugins": len(plugins),
	}
	if h.pool != nil {
		resp["browser_pool"] = h.pool.Stats()
	}
	if h.queue != nil {
		resp["queue"] = h.queue.Stats()
	}
	writeJSON(w, http.StatusOK, resp)
}
