package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spectra-browser/spectra/internal/domain"
)

// ProcessPool manages N concurrent PluginProcess instances for a single plugin.
// Requests are routed round-robin to idle processes; semaphore limits concurrency.
type ProcessPool struct {
	manifest domain.PluginManifest
	size     int
	procs    []*PluginProcess
	sem      chan struct{}
	idx      atomic.Uint64
	mu       sync.Mutex
}

func NewProcessPool(manifest domain.PluginManifest, dir string, size int) *ProcessPool {
	if size < 1 {
		size = 1
	}
	procs := make([]*PluginProcess, size)
	for i := range procs {
		procs[i] = NewPluginProcess(manifest, dir)
	}
	return &ProcessPool{
		manifest: manifest,
		size:     size,
		procs:    procs,
		sem:      make(chan struct{}, size),
	}
}

// Call acquires a semaphore slot, picks a process, auto-starts if needed, executes.
func (p *ProcessPool) Call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	select {
	case p.sem <- struct{}{}:
	case <-ctx.Done():
		return nil, fmt.Errorf("pool acquire: %w", ctx.Err())
	}
	defer func() { <-p.sem }()

	proc := p.pick()

	if !proc.Healthy() {
		startCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := proc.Start(startCtx); err != nil {
			return nil, fmt.Errorf("start plugin: %w", err)
		}
	}

	return proc.Call(ctx, method, params)
}

// pick returns a healthy process via round-robin, falling back to any process.
func (p *ProcessPool) pick() *PluginProcess {
	n := uint64(p.size)
	base := p.idx.Add(1) % n
	for i := uint64(0); i < n; i++ {
		proc := p.procs[(base+i)%n]
		if proc.Healthy() {
			return proc
		}
	}
	return p.procs[base]
}

func (p *ProcessPool) Healthy() bool {
	for _, proc := range p.procs {
		if proc.Healthy() {
			return true
		}
	}
	return false
}

func (p *ProcessPool) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, proc := range p.procs {
		if err := proc.Start(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (p *ProcessPool) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, proc := range p.procs {
		if err := proc.Stop(ctx); err != nil {
			slog.Error("failed to stop plugin process", "name", p.manifest.Name, "error", err)
		}
	}
	return nil
}
