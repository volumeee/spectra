package handler

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/spectra-browser/spectra/internal/port"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// LiveViewHandler streams browser state via WebSocket for real-time viewing.
// GET /api/sessions/:id/live
// Sends: {"type":"heartbeat","timestamp":ms,"endpoint":"ws://..."}
type LiveViewHandler struct {
	pool port.BrowserPool
}

func NewLiveViewHandler(pool port.BrowserPool) *LiveViewHandler {
	return &LiveViewHandler{pool: pool}
}

func (h *LiveViewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = chi.URLParam(r, "id") // TODO: resolve session → specific browser

	sess, err := h.pool.Acquire(r.Context())
	if err != nil {
		http.Error(w, "no browser available", http.StatusServiceUnavailable)
		return
	}
	defer h.pool.Release(sess)

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		slog.Error("live view ws accept failed", "error", err)
		return
	}
	defer conn.CloseNow()

	slog.Info("live view connected", "client", r.RemoteAddr)

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := wsjson.Write(ctx, conn, map[string]interface{}{
				"type":      "heartbeat",
				"timestamp": time.Now().UnixMilli(),
				"endpoint":  sess.CDPEndpoint(),
			}); err != nil {
				return
			}
		}
	}
}
