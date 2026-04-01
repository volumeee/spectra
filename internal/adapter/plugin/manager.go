package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spectra-browser/spectra/internal/domain"
)

type Manager struct {
	pluginsDir string
	poolSize   int
	plugins    map[string]*pluginEntry
	mu         sync.RWMutex
}

type pluginEntry struct {
	manifest domain.PluginManifest
	pool     *ProcessPool
}

func NewManager(pluginsDir string, poolSize int) *Manager {
	if poolSize < 1 {
		poolSize = 1
	}
	return &Manager{
		pluginsDir: pluginsDir,
		poolSize:   poolSize,
		plugins:    make(map[string]*pluginEntry),
	}
}

func (m *Manager) Discover(_ context.Context) error {
	entries, err := os.ReadDir(m.pluginsDir)
	if err != nil {
		if os.IsNotExist(err) {
			slog.Warn("plugins directory not found", "dir", m.pluginsDir)
			return nil
		}
		return fmt.Errorf("read plugins dir: %w", err)
	}

	manifests := make(map[string]domain.PluginManifest)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(m.pluginsDir, e.Name()))
		if err != nil {
			continue
		}
		var manifest domain.PluginManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			slog.Error("invalid plugin manifest", "file", e.Name(), "error", err)
			continue
		}
		manifests[manifest.Name] = manifest
	}

	for _, e := range entries {
		if e.IsDir() || strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.Mode()&0111 == 0 {
			continue
		}

		name := e.Name()
		manifest, hasManifest := manifests[name]
		if !hasManifest {
			manifest = domain.PluginManifest{Name: name, Version: "0.0.0", Command: name, Methods: []string{}}
		}

		m.mu.Lock()
		m.plugins[manifest.Name] = &pluginEntry{
			manifest: manifest,
			pool:     NewProcessPool(manifest, m.pluginsDir, m.poolSize),
		}
		m.mu.Unlock()

		slog.Info("plugin discovered", "name", manifest.Name, "version", manifest.Version, "pool_size", m.poolSize)
	}
	return nil
}

func (m *Manager) Get(name string) (*domain.PluginInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	entry, ok := m.plugins[name]
	if !ok {
		return nil, domain.ErrPluginNotFound
	}

	status := domain.PluginStatusStopped
	if entry.pool.Healthy() {
		status = domain.PluginStatusRunning
	}
	return &domain.PluginInfo{Manifest: entry.manifest, Status: status}, nil
}

func (m *Manager) List() []domain.PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]domain.PluginInfo, 0, len(m.plugins))
	for _, entry := range m.plugins {
		status := domain.PluginStatusStopped
		if entry.pool.Healthy() {
			status = domain.PluginStatusRunning
		}
		list = append(list, domain.PluginInfo{Manifest: entry.manifest, Status: status})
	}
	return list
}

func (m *Manager) Execute(ctx context.Context, pluginName, method string, params json.RawMessage) (*domain.JobResult, error) {
	m.mu.RLock()
	entry, ok := m.plugins[pluginName]
	m.mu.RUnlock()

	if !ok {
		return nil, domain.ErrPluginNotFound
	}

	start := time.Now()
	result, err := entry.pool.Call(ctx, method, params)
	duration := time.Since(start).Milliseconds()

	if err != nil {
		return &domain.JobResult{Error: err.Error(), DurationMs: duration}, err
	}
	return &domain.JobResult{Data: result, DurationMs: duration}, nil
}

func (m *Manager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, entry := range m.plugins {
		if err := entry.pool.Stop(ctx); err != nil {
			slog.Error("failed to stop plugin pool", "name", name, "error", err)
		}
	}
	return nil
}
