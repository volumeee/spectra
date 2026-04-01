package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"sync/atomic"

	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/pkg/protocol"
)

type callResult struct {
	resp *protocol.Response
	err  error
}

type PluginProcess struct {
	manifest domain.PluginManifest
	cmd      *exec.Cmd
	codec    *JSONRPCCodec
	reqID    atomic.Int64
	mu       sync.Mutex
	running  bool
	dir      string
}

func NewPluginProcess(manifest domain.PluginManifest, dir string) *PluginProcess {
	return &PluginProcess{manifest: manifest, dir: dir}
}

func (p *PluginProcess) Start(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.running {
		return nil
	}

	cmdPath := p.dir + "/" + p.manifest.Name
	p.cmd = exec.Command(cmdPath)
	p.cmd.Stderr = os.Stderr

	stdin, err := p.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}
	stdout, err := p.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("start plugin %s: %w", p.manifest.Name, err)
	}

	p.codec = NewJSONRPCCodec(stdout, stdin)
	p.running = true

	go func() {
		_ = p.cmd.Wait()
		p.mu.Lock()
		p.running = false
		p.mu.Unlock()
		slog.Warn("plugin process exited", "name", p.manifest.Name)
	}()

	slog.Info("plugin started", "name", p.manifest.Name, "pid", p.cmd.Process.Pid)
	return nil
}

// Call sends a JSON-RPC request and waits for response, respecting ctx deadline.
// If ctx is cancelled/timed out, the plugin process is killed and will restart on next call.
func (p *PluginProcess) Call(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil, domain.ErrPluginCrashed
	}

	id := p.reqID.Add(1)
	req := &protocol.Request{
		JSONRPC: protocol.JSONRPCVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := p.codec.WriteRequest(req); err != nil {
		p.running = false
		p.mu.Unlock()
		return nil, fmt.Errorf("write request: %w", err)
	}

	// Read response in goroutine so we can select on ctx
	ch := make(chan callResult, 1)
	go func() {
		resp, err := p.codec.ReadResponse()
		ch <- callResult{resp: resp, err: err}
	}()

	p.mu.Unlock()

	select {
	case <-ctx.Done():
		slog.Warn("plugin call timeout, killing process", "name", p.manifest.Name, "method", method)
		p.kill()
		return nil, fmt.Errorf("%w: %v", domain.ErrPluginTimeout, ctx.Err())
	case res := <-ch:
		if res.err != nil {
			p.mu.Lock()
			p.running = false
			p.mu.Unlock()
			return nil, fmt.Errorf("read response: %w", res.err)
		}
		if res.resp.Error != nil {
			return nil, fmt.Errorf("plugin error [%d]: %s", res.resp.Error.Code, res.resp.Error.Message)
		}
		return res.resp.Result, nil
	}
}

func (p *PluginProcess) kill() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.running && p.cmd != nil && p.cmd.Process != nil {
		p.running = false
		_ = p.cmd.Process.Kill()
	}
}

func (p *PluginProcess) Stop(_ context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running || p.cmd == nil || p.cmd.Process == nil {
		return nil
	}

	p.running = false
	_ = p.cmd.Process.Kill()
	_ = p.cmd.Wait()
	slog.Info("plugin stopped", "name", p.manifest.Name)
	return nil
}

func (p *PluginProcess) Healthy() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.running
}
