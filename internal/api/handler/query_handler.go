package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/internal/port"
)

// QueryStep is a single step in a SpectraQL query.
type QueryStep struct {
	Action   string            `json:"action"`             // goto|wait_for|click|type|extract|screenshot|scroll|evaluate_js
	URL      string            `json:"url,omitempty"`      // for goto
	Selector string            `json:"selector,omitempty"` // for wait_for, click, type, extract
	Value    string            `json:"value,omitempty"`    // for type, evaluate_js
	Selectors map[string]string `json:"selectors,omitempty"` // for extract (key→selector map)
	WaitUntil string           `json:"wait_until,omitempty"` // load|networkIdle|domContentLoaded
	Timeout  int               `json:"timeout,omitempty"`
}

// QueryRequest is the body for POST /api/query
type QueryRequest struct {
	Steps     []QueryStep `json:"steps"`
	SessionID string      `json:"session_id,omitempty"`
	Width     int         `json:"width,omitempty"`
	Height    int         `json:"height,omitempty"`
}

// QueryHandler executes multi-step browser workflows in a single request (SpectraQL).
type QueryHandler struct {
	plugins port.PluginManager
	queue   port.JobQueue
	pool    port.BrowserPool
}

func NewQueryHandler(plugins port.PluginManager, queue port.JobQueue, pool port.BrowserPool) *QueryHandler {
	return &QueryHandler{plugins: plugins, queue: queue, pool: pool}
}

func (h *QueryHandler) Execute(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, r, http.StatusBadRequest, "INVALID_JSON", err.Error())
		return
	}
	if len(req.Steps) == 0 {
		writeError(w, r, http.StatusBadRequest, "INVALID_BODY", "steps is required")
		return
	}

	// Encode as recorder-compatible params and delegate to recorder plugin
	params, _ := json.Marshal(map[string]interface{}{
		"url":         findFirstURL(req.Steps),
		"steps":       convertSteps(req.Steps),
		"output_mode": "frames",
		"width":       req.Width,
		"height":      req.Height,
	})

	job := &domain.Job{
		ID:        uuid.NewString(),
		Plugin:    "recorder",
		Method:    "record",
		Params:    params,
		Status:    domain.JobStatusPending,
		CreatedAt: time.Now(),
	}

	// Inject CDP endpoint if pool available
	if h.pool != nil {
		if sess, err := h.pool.Acquire(r.Context()); err == nil {
			params = injectCDPEndpoint(params, sess.CDPEndpoint())
			job.Params = params
			defer h.pool.Release(sess)
		}
	}

	result, err := h.queue.Enqueue(r.Context(), job, func(ctx context.Context, j *domain.Job) (*domain.JobResult, error) {
		return h.plugins.Execute(ctx, j.Plugin, j.Method, j.Params)
	})
	if err != nil {
		writeError(w, r, http.StatusInternalServerError, "QUERY_ERROR", err.Error())
		return
	}

	writeSuccess(w, r, result, start)
}

func findFirstURL(steps []QueryStep) string {
	for _, s := range steps {
		if s.Action == "goto" && s.URL != "" {
			return s.URL
		}
	}
	return ""
}

// convertSteps maps SpectraQL steps to recorder plugin steps.
func convertSteps(steps []QueryStep) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(steps))
	for _, s := range steps {
		step := map[string]interface{}{"action": s.Action}
		switch s.Action {
		case "goto":
			step["action"] = "navigate"
			step["value"] = s.URL
		default:
			if s.Selector != "" {
				step["selector"] = s.Selector
			}
			if s.Value != "" {
				step["value"] = s.Value
			}
			if len(s.Selectors) > 0 {
				step["selectors"] = s.Selectors
			}
		}
		if s.Timeout > 0 {
			step["timeout"] = s.Timeout
		}
		out = append(out, step)
	}
	return out
}
