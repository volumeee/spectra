package cdpproxy

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/spectra-browser/spectra/internal/port"
	"nhooyr.io/websocket"
)

type Proxy struct {
	pool        port.BrowserPool
	idleTimeout time.Duration
}

func NewProxy(pool port.BrowserPool, idleTimeout time.Duration) *Proxy {
	return &Proxy{pool: pool, idleTimeout: idleTimeout}
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	session, err := p.pool.Acquire(r.Context())
	if err != nil {
		http.Error(w, "no browser available", http.StatusServiceUnavailable)
		return
	}
	defer p.pool.Release(session)

	cdpURL := session.CDPEndpoint()
	if cdpURL == "" {
		http.Error(w, "no cdp endpoint", http.StatusInternalServerError)
		return
	}

	clientConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"},
	})
	if err != nil {
		slog.Error("ws accept failed", "error", err)
		return
	}
	defer clientConn.CloseNow()

	browserConn, _, err := websocket.Dial(r.Context(), cdpURL, nil)
	if err != nil {
		slog.Error("ws dial browser failed", "error", err)
		clientConn.Close(websocket.StatusInternalError, "cannot connect to browser")
		return
	}
	defer browserConn.CloseNow()

	slog.Info("cdp proxy connected", "client", r.RemoteAddr)

	done := make(chan struct{}, 2)
	go func() {
		proxyWS(r.Context(), clientConn, browserConn)
		done <- struct{}{}
	}()
	go func() {
		proxyWS(r.Context(), browserConn, clientConn)
		done <- struct{}{}
	}()

	<-done
	slog.Info("cdp proxy disconnected", "client", r.RemoteAddr)
}

func proxyWS(ctx context.Context, src, dst *websocket.Conn) {
	for {
		typ, data, err := src.Read(ctx)
		if err != nil {
			if err != io.EOF {
				slog.Debug("ws read error", "error", err)
			}
			return
		}
		if err := dst.Write(ctx, typ, data); err != nil {
			return
		}
	}
}
