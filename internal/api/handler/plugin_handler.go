package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/internal/port"
)

type PluginHandler struct {
	plugins   port.PluginManager
	queue     port.JobQueue
	pool      port.BrowserPool // optional: inject CDP endpoint into plugin params
	sharePool bool
}

func NewPluginHandler(plugins port.PluginManager, queue port.JobQueue, pool port.BrowserPool, sharePool bool) *PluginHandler {
	return &PluginHandler{plugins: plugins, queue: queue, pool: pool, sharePool: sharePool}
}

func (h *PluginHandler) Execute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	pluginName := chi.URLParam(r, "plugin")
	method := chi.URLParam(r, "method")

	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "failed to read request body")
		return
	}

	var params json.RawMessage
	if len(body) > 0 {
		if !json.Valid(body) {
			writeError(w, r, http.StatusBadRequest, "INVALID_JSON", "request body is not valid JSON")
			return
		}
		params = body
	}

	// Acquire browser session from pool and inject CDP endpoint into params.
	// Plugins check for _cdp_endpoint and connect to existing browser instead of launching new one.
	var session port.BrowserSession
	if h.sharePool && h.pool != nil {
		session, err = h.pool.Acquire(r.Context())
		if err == nil {
			params = injectCDPEndpoint(params, session.CDPEndpoint())
		}
		// If acquire fails, plugin falls back to launching its own browser
	}

	job := &domain.Job{
		ID:        uuid.NewString(),
		Plugin:    pluginName,
		Method:    method,
		Params:    params,
		Status:    domain.JobStatusPending,
		CreatedAt: time.Now(),
	}

	result, err := h.queue.Enqueue(r.Context(), job, func(ctx context.Context, j *domain.Job) (*domain.JobResult, error) {
		return h.plugins.Execute(ctx, j.Plugin, j.Method, j.Params)
	})

	// Release browser session back to pool after job completes
	if session != nil {
		h.pool.Release(session)
	}

	if err != nil {
		if errors.Is(err, domain.ErrPluginNotFound) {
			writeError(w, r, http.StatusNotFound, "PLUGIN_NOT_FOUND", "plugin '"+pluginName+"' not found")
			return
		}
		if errors.Is(err, domain.ErrQueueFull) {
			writeError(w, r, http.StatusServiceUnavailable, "QUEUE_FULL", "server is busy, try again later")
			return
		}
		if errors.Is(err, domain.ErrPluginTimeout) {
			writeError(w, r, http.StatusGatewayTimeout, "PLUGIN_TIMEOUT", "plugin execution timed out")
			return
		}
		writeError(w, r, http.StatusInternalServerError, "EXECUTION_ERROR", err.Error())
		return
	}

	writeSuccess(w, r, result, start)
}

// injectCDPEndpoint merges _cdp_endpoint into existing JSON params object.
func injectCDPEndpoint(params json.RawMessage, endpoint string) json.RawMessage {
	m := make(map[string]json.RawMessage)
	if len(params) > 0 {
		_ = json.Unmarshal(params, &m)
	}
	ep, _ := json.Marshal(endpoint)
	m["_cdp_endpoint"] = ep
	result, _ := json.Marshal(m)
	return result
}
