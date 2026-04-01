# 🏗️ Architecture

## Clean Architecture Layers

```
┌─────────────────────────────────────┐
│           cmd/ (entrypoint)         │  Wire & boot
├─────────────────────────────────────┤
│           api/ (HTTP layer)         │  Chi router, handlers, middleware
├─────────────────────────────────────┤
│         adapter/ (implementations)  │  Plugin pool, browser pool, queue, storage, monitor, metrics, recording
├─────────────────────────────────────┤
│          port/ (interfaces)         │  Contracts between layers
├─────────────────────────────────────┤
│         domain/ (core types)        │  Pure types, zero dependencies
└─────────────────────────────────────┘
```

Dependencies point **inward only**. Domain has zero imports. Ports define interfaces. Adapters implement them. API uses ports.

## Data Flow

```
HTTP Request
  → Chi Router (middleware: auth, rate limit, logging, request ID)
    → Plugin Handler
      → Acquire browser from pool (inject _cdp_endpoint into params)
        → Job Queue (concurrency control + health check)
          → Plugin Manager
            → Process Pool (N concurrent plugin processes)
              → Plugin Process (JSON-RPC over stdin/stdout, ctx timeout)
                → Plugin logic (reuses existing browser via CDP endpoint)
              ← JSON-RPC response (or timeout → kill → restart)
            ← Job result
          ← Release browser back to pool
        ← Queue event (webhook + metrics + job store)
      ← API response
  ← HTTP Response
```

## Key Design Decisions

1. **Interface-based DI** — All components defined as interfaces in `port/`. Implementations in `adapter/`. Wired in `cmd/`. Fully testable with mocks.

2. **Plugin = subprocess pool** — Each plugin has N processes (configurable `pool_size`). Crash isolation. Language agnostic. JSON-RPC over stdin/stdout. Context-based timeout kills hung processes.

3. **Shared browser pool** — Core acquires browser from pool, injects CDP endpoint into plugin params. Plugins connect to existing browser instead of launching new one. 5-10x faster, dramatically lower memory usage.

4. **Health monitoring** — System monitor checks CPU/memory before accepting jobs. Returns 503 when overloaded. `/pressure` endpoint for load balancers.

5. **Pluggable storage** — `storage.driver = "memory" | "sqlite"`. Interface-based, easy to add Postgres. Webhooks, schedules, and job history all persist.

6. **Metrics & observability** — `/api/metrics` for request counts, success rates, avg duration per plugin. `/api/jobs` for job history.

7. **AI agent plugin** — `ai-browse` plugin uses OpenAI-compatible API to autonomously control browser via natural language. Agent loop: LLM decides action → execute → observe → repeat.

8. **CDP recording** — `recorder` plugin captures frames via `Page.startScreencast`. Output: video (WebP), frames (PNG), or both.

## Adapters

| Adapter | Interface | Description |
|---------|-----------|-------------|
| `adapter/plugin` | `PluginManager` | ProcessPool per plugin, round-robin routing, auto-restart |
| `adapter/browser` | `BrowserPool` | Chromium pool, acquire/release, idle timeout |
| `adapter/queue` | `JobQueue` | Semaphore concurrency, health check gate |
| `adapter/webhook` | `WebhookStore` | Memory or SQLite |
| `adapter/scheduler` | `Scheduler` | Cron-based task scheduler |
| `adapter/storage` | `WebhookStore`, `JobStore` | SQLite persistence |
| `adapter/monitor` | `SystemMonitor` | /proc/stat + /proc/meminfo |
| `adapter/metrics` | `MetricsCollector` | In-memory atomic counters |
| `adapter/recording` | `Recorder` | CDP screencast frame capture |
| `adapter/cdpproxy` | — | WebSocket proxy to browser |
