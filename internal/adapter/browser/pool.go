package browser

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/go-rod/rod"
	"github.com/spectra-browser/spectra/internal/port"
)

// session wraps a browser entry acquired from the pool.
// It carries the use count so Release() can track recycling.
type session struct {
	browser  *rod.Browser
	endpoint string
	uses     int // inherited from browserEntry on acquire
}

func (s *session) CDPEndpoint() string { return s.endpoint }
func (s *session) Close()              {}

// browserEntry is the pool's internal representation of a browser instance.
type browserEntry struct {
	browser  *rod.Browser
	endpoint string
	uses     int
}

// Pool manages a pool of Chromium browser instances.
// warm_size pre-launches browsers for zero cold-start latency.
// recycle_after recycles a browser after N uses to prevent memory drift.
type Pool struct {
	maxInstances int
	warmSize     int
	recycleAfter int
	pool         chan *browserEntry
	active       atomic.Int32
	total        atomic.Int32
	mu           sync.Mutex
	closed       bool
}

func NewPool(maxInstances, warmSize, recycleAfter int) *Pool {
	if warmSize > maxInstances {
		warmSize = maxInstances
	}
	if recycleAfter <= 0 {
		recycleAfter = 100
	}
	return &Pool{
		maxInstances: maxInstances,
		warmSize:     warmSize,
		recycleAfter: recycleAfter,
		pool:         make(chan *browserEntry, maxInstances),
	}
}

// WarmUp pre-launches warmSize browsers so they're ready immediately.
func (p *Pool) WarmUp(_ context.Context) {
	for i := 0; i < p.warmSize; i++ {
		entry, err := p.launch()
		if err != nil {
			slog.Warn("warm-up browser failed", "error", err)
			return
		}
		select {
		case p.pool <- entry:
		default:
			entry.browser.MustClose()
			p.total.Add(-1)
		}
	}
	if p.warmSize > 0 {
		slog.Info("browser pool warmed up", "warm_size", p.warmSize)
	}
}

func (p *Pool) Acquire(ctx context.Context) (port.BrowserSession, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("browser pool is closed")
	}
	p.mu.Unlock()

	// Try idle pool first
	select {
	case entry := <-p.pool:
		p.active.Add(1)
		return &session{browser: entry.browser, endpoint: entry.endpoint, uses: entry.uses}, nil
	default:
	}

	// Launch new if under limit
	if int(p.total.Load()) < p.maxInstances {
		entry, err := p.launch()
		if err != nil {
			return nil, err
		}
		p.active.Add(1)
		return &session{browser: entry.browser, endpoint: entry.endpoint}, nil
	}

	// Wait for one to be released
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("acquire browser: %w", ctx.Err())
	case entry := <-p.pool:
		p.active.Add(1)
		return &session{browser: entry.browser, endpoint: entry.endpoint, uses: entry.uses}, nil
	}
}

func (p *Pool) Release(s port.BrowserSession) {
	sess, ok := s.(*session)
	if !ok {
		return
	}
	p.active.Add(-1)

	sess.uses++

	// Recycle after N uses to prevent memory drift
	if sess.uses >= p.recycleAfter {
		slog.Debug("recycling browser", "uses", sess.uses)
		sess.browser.MustClose()
		p.total.Add(-1)
		if fresh, err := p.launch(); err == nil {
			select {
			case p.pool <- fresh:
			default:
				fresh.browser.MustClose()
				p.total.Add(-1)
			}
		}
		return
	}

	entry := &browserEntry{browser: sess.browser, endpoint: sess.endpoint, uses: sess.uses}
	select {
	case p.pool <- entry:
	default:
		sess.browser.MustClose()
		p.total.Add(-1)
	}
}

func (p *Pool) Stats() port.PoolStats {
	return port.PoolStats{
		Active: int(p.active.Load()),
		Idle:   len(p.pool),
		Total:  int(p.total.Load()),
		Max:    p.maxInstances,
	}
}

func (p *Pool) Shutdown(_ context.Context) error {
	p.mu.Lock()
	p.closed = true
	p.mu.Unlock()

	close(p.pool)
	for entry := range p.pool {
		entry.browser.MustClose()
	}
	slog.Info("browser pool shut down")
	return nil
}

func (p *Pool) launch() (*browserEntry, error) {
	cfg := DefaultLaunchConfig()
	u, err := cfg.LaunchURL()
	if err != nil {
		return nil, fmt.Errorf("launch chromium: %w", err)
	}
	b := rod.New().ControlURL(u)
	if err := b.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}
	p.total.Add(1)
	slog.Info("browser launched", "total", p.total.Load())
	return &browserEntry{browser: b, endpoint: u}, nil
}
