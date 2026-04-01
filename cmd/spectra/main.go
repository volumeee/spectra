package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spectra-browser/spectra/internal/adapter/browser"
	"github.com/spectra-browser/spectra/internal/adapter/metrics"
	"github.com/spectra-browser/spectra/internal/adapter/monitor"
	pluginAdapter "github.com/spectra-browser/spectra/internal/adapter/plugin"
	"github.com/spectra-browser/spectra/internal/adapter/queue"
	"github.com/spectra-browser/spectra/internal/adapter/scheduler"
	"github.com/spectra-browser/spectra/internal/adapter/storage"
	"github.com/spectra-browser/spectra/internal/adapter/webhook"
	"github.com/spectra-browser/spectra/internal/api"
	"github.com/spectra-browser/spectra/internal/config"
	"github.com/spectra-browser/spectra/internal/domain"
	"github.com/spectra-browser/spectra/internal/port"
	spectraMCP "github.com/spectra-browser/spectra/internal/mcp"

	mcpserver "github.com/mark3labs/mcp-go/server"
)

const version = "0.3.0"

func main() {
	cfgPath := "spectra.yaml"
	if v := os.Getenv("SPECTRA_CONFIG"); v != "" {
		cfgPath = v
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	// Logger
	opts := &slog.HandlerOptions{Level: parseLogLevel(cfg.Log.Level)}
	var logHandler slog.Handler
	if cfg.Log.Format == "json" {
		logHandler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(logHandler))

	// System monitor
	var sysMonitor port.SystemMonitor
	if cfg.Health.Enabled {
		sysMonitor = monitor.New(cfg.Health.CPULimit, cfg.Health.MemoryLimit)
	}

	// Metrics
	metricsCollector := metrics.New()

	// Browser pool with warm-up
	pool := browser.NewPool(cfg.Browser.MaxInstances, cfg.Browser.WarmPoolSize, cfg.Browser.RecycleAfter)

	// Plugin manager
	pluginMgr := pluginAdapter.NewManager(cfg.Plugins.Dir, cfg.Plugins.PoolSize)

	// Job queue
	jobQueue := queue.NewMemoryQueue(cfg.Queue.MaxConcurrent, cfg.Queue.MaxPending, sysMonitor)

	// Storage
	var webhookStore port.WebhookStore
	var jobStore port.JobStore
	var sessionMgr port.SessionManager
	var profileStore port.ProfileStore
	var sqliteStore *storage.SQLiteStore

	if cfg.Storage.Driver == "sqlite" {
		sqliteStore, err = storage.NewSQLiteStore(cfg.Storage.SQLitePath)
		if err != nil {
			slog.Error("failed to open SQLite", "error", err)
			os.Exit(1)
		}
		webhookStore = sqliteStore
		jobStore = sqliteStore
		sessionMgr = sqliteStore
		profileStore = sqliteStore
		slog.Info("storage: sqlite", "path", cfg.Storage.SQLitePath)
	} else {
		webhookStore = webhook.NewMemoryStore()
		slog.Info("storage: memory")
	}

	// Webhook engine + job callbacks
	webhookEngine := webhook.NewEngine(webhookStore, cfg.Webhook.MaxRetries, cfg.Webhook.RetryInterval)

	jobQueue.OnJobCompleted(func(job *domain.Job, result *domain.JobResult) {
		if cfg.Webhook.Enabled {
			webhookEngine.HandleJobCompleted(job, result)
		}
		if jobStore != nil {
			_ = jobStore.Save(context.Background(), job, result)
		}
		metricsCollector.RecordRequest(job.Plugin, job.Method, result.DurationMs, result.Error == "")
	})
	jobQueue.OnJobFailed(func(job *domain.Job, result *domain.JobResult) {
		if cfg.Webhook.Enabled {
			webhookEngine.HandleJobFailed(job, result)
		}
		if jobStore != nil && result != nil {
			_ = jobStore.Save(context.Background(), job, result)
		}
		metricsCollector.RecordRequest(job.Plugin, job.Method, 0, false)
	})

	// Scheduler
	var cronScheduler *scheduler.CronScheduler
	if cfg.Scheduler.Enabled {
		cronScheduler = scheduler.NewCronScheduler(pluginMgr, jobQueue)
	}

	// Discover plugins
	ctx := context.Background()
	if err := pluginMgr.Discover(ctx); err != nil {
		slog.Error("plugin discovery failed", "error", err)
	}

	// Warm up browser pool
	pool.WarmUp(ctx)

	// Build HTTP server
	deps := api.ServerDeps{
		Config:    cfg,
		Plugins:   pluginMgr,
		Queue:     jobQueue,
		Webhooks:  webhookStore,
		Pool:      pool,
		Monitor:   sysMonitor,
		Metrics:   metricsCollector,
		Jobs:      jobStore,
		Sessions:  sessionMgr,
		Profiles:  profileStore,
	}
	if cfg.Scheduler.Enabled && cronScheduler != nil {
		deps.Scheduler = cronScheduler
	}

	router := api.NewServer(deps)
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	if cronScheduler != nil {
		cronScheduler.Start(ctx)
	}

	// MCP server
	if cfg.MCP.Enabled {
		mcpSrv := spectraMCP.NewMCPServer(pluginMgr)
		go func() {
			slog.Info("MCP server starting", "transport", cfg.MCP.Transport)
			if cfg.MCP.Transport == "sse" {
				sseSrv := mcpserver.NewSSEServer(mcpSrv)
				if err := sseSrv.Start(fmt.Sprintf(":%d", cfg.Server.Port+1)); err != nil {
					slog.Error("MCP SSE server error", "error", err)
				}
			} else {
				stdioSrv := mcpserver.NewStdioServer(mcpSrv)
				if err := stdioSrv.Listen(ctx, os.Stdin, os.Stdout); err != nil {
					slog.Error("MCP stdio server error", "error", err)
				}
			}
		}()
	}

	// Startup banner
	plugins := pluginMgr.List()
	pluginNames := make([]string, len(plugins))
	for i, p := range plugins {
		pluginNames[i] = p.Manifest.Name
	}
	fmt.Println()
	fmt.Println("  🔮 Spectra v" + version)
	fmt.Printf("  ├── Port:        %d\n", cfg.Server.Port)
	fmt.Printf("  ├── Browsers:    0/%d (max, warm=%d, recycle_after=%d)\n", cfg.Browser.MaxInstances, cfg.Browser.WarmPoolSize, cfg.Browser.RecycleAfter)
	fmt.Printf("  ├── Plugins:     %d loaded %v (pool=%d)\n", len(plugins), pluginNames, cfg.Plugins.PoolSize)
	fmt.Printf("  ├── Storage:     %s\n", cfg.Storage.Driver)
	fmt.Printf("  ├── Health:      %v (cpu=%d%% mem=%d%%)\n", cfg.Health.Enabled, cfg.Health.CPULimit, cfg.Health.MemoryLimit)
	fmt.Printf("  ├── Sessions:    %v\n", sessionMgr != nil)
	fmt.Printf("  ├── CDP Proxy:   ws://localhost:%d/cdp\n", cfg.Server.Port)
	fmt.Printf("  ├── Webhooks:    %v\n", cfg.Webhook.Enabled)
	fmt.Printf("  ├── Scheduler:   %v\n", cfg.Scheduler.Enabled)
	fmt.Printf("  ├── MCP:         %v\n", cfg.MCP.Enabled)
	fmt.Printf("  └── Ready\n")
	fmt.Println()

	go func() {
		slog.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_ = srv.Shutdown(shutdownCtx)
	if cronScheduler != nil {
		cronScheduler.Stop(shutdownCtx)
	}
	_ = jobQueue.Shutdown(shutdownCtx)
	_ = pluginMgr.Shutdown(shutdownCtx)
	_ = pool.Shutdown(shutdownCtx)
	if sqliteStore != nil {
		_ = sqliteStore.Close()
	}

	slog.Info("shutdown complete")
}

func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
