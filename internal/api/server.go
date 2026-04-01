package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/spectra-browser/spectra/internal/adapter/cdpproxy"
	"github.com/spectra-browser/spectra/internal/api/handler"
	"github.com/spectra-browser/spectra/internal/api/middleware"
	"github.com/spectra-browser/spectra/internal/config"
	"github.com/spectra-browser/spectra/internal/port"
)

type ServerDeps struct {
	Config    *config.Config
	Plugins   port.PluginManager
	Queue     port.JobQueue
	Webhooks  port.WebhookStore
	Scheduler port.Scheduler
	Pool      port.BrowserPool
	Monitor   port.SystemMonitor
	Metrics   port.MetricsCollector
	Jobs      port.JobStore
	Sessions  port.SessionManager
	Profiles  port.ProfileStore
}

func NewServer(deps ServerDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)
	r.Use(middleware.RequestID)
	r.Use(middleware.Logging)
	r.Use(middleware.Auth(deps.Config.Auth.APIKey, deps.Config.Auth.Enabled))
	rl := middleware.NewRateLimiter(100, 200)
	r.Use(rl.Middleware)

	pluginH := handler.NewPluginHandler(deps.Plugins, deps.Queue, deps.Pool, deps.Config.Browser.SharePool)
	healthH := handler.NewHealthHandler(deps.Plugins, deps.Pool, deps.Queue)
	infoH := handler.NewInfoHandler(deps.Plugins, deps.Queue)
	queryH := handler.NewQueryHandler(deps.Plugins, deps.Queue, deps.Pool)

	// Core
	r.Get("/health", healthH.Liveness)
	r.Get("/ready", healthH.Readiness)
	r.Get("/api/plugins", infoH.ListPlugins)
	r.Post("/api/{plugin}/{method}", pluginH.Execute)
	r.Post("/api/query", queryH.Execute)

	// Observability — always registered
	if deps.Monitor != nil {
		r.Get("/pressure", handler.NewPressureHandler(deps.Monitor).ServeHTTP)
	}
	if deps.Metrics != nil {
		r.Get("/api/metrics", handler.NewMetricsHandler(deps.Metrics).ServeHTTP)
	}
	// /api/jobs always registered; returns empty list when no persistent store
	r.Get("/api/jobs", handler.NewJobsHandler(deps.Jobs).ServeHTTP)

	// CDP proxy + live view
	if deps.Pool != nil {
		r.Get("/cdp", cdpproxy.NewProxy(deps.Pool, 5*time.Minute).ServeHTTP)
		r.Get("/api/sessions/{id}/live", handler.NewLiveViewHandler(deps.Pool).ServeHTTP)
	}

	// Sessions — always registered; returns 503 when no persistent store
	r.Post("/api/sessions", handler.NewSessionHandler(deps.Sessions).Create)
	r.Get("/api/sessions", handler.NewSessionHandler(deps.Sessions).List)
	r.Get("/api/sessions/{id}", handler.NewSessionHandler(deps.Sessions).Get)
	r.Delete("/api/sessions/{id}", handler.NewSessionHandler(deps.Sessions).Delete)

	// Profiles — always registered; returns 503 when no persistent store
	r.Post("/api/profiles", handler.NewProfileHandler(deps.Profiles).Create)
	r.Get("/api/profiles", handler.NewProfileHandler(deps.Profiles).List)
	r.Delete("/api/profiles/{id}", handler.NewProfileHandler(deps.Profiles).Delete)

	// Webhooks
	if deps.Config.Webhook.Enabled && deps.Webhooks != nil {
		webhookH := handler.NewWebhookHandler(deps.Webhooks)
		r.Post("/api/webhooks", webhookH.Create)
		r.Get("/api/webhooks", webhookH.List)
		r.Delete("/api/webhooks/{id}", webhookH.Delete)
	}

	// Scheduler
	if deps.Config.Scheduler.Enabled && deps.Scheduler != nil {
		scheduleH := handler.NewScheduleHandler(deps.Scheduler)
		r.Post("/api/schedules", scheduleH.Create)
		r.Get("/api/schedules", scheduleH.List)
		r.Delete("/api/schedules/{id}", scheduleH.Delete)
	}

	return r
}
