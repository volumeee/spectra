package handler

import (
	"net/http"
	"time"

	"github.com/spectra-browser/spectra/internal/port"
)

type InfoHandler struct {
	plugins port.PluginManager
	queue   port.JobQueue
}

func NewInfoHandler(plugins port.PluginManager, queue port.JobQueue) *InfoHandler {
	return &InfoHandler{plugins: plugins, queue: queue}
}

func (h *InfoHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	plugins := h.plugins.List()
	data := map[string]interface{}{
		"plugins": plugins,
		"queue":   h.queue.Stats(),
	}
	writeSuccess(w, r, data, start)
}
