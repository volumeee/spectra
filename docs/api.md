# 📡 Spectra API Reference v2

Base URL: `http://localhost:3000`

## Authentication

If auth is enabled: `Authorization: Bearer <api-key>`

## Endpoints

### Health & Observability

```
GET /health          → Liveness check
GET /ready           → Readiness (plugins, browser pool, queue stats)
GET /pressure        → System load (503 when CPU/memory overloaded)
GET /api/metrics     → Request counts, success rates, avg duration per plugin
GET /api/jobs        → Job history (query: ?limit=100)
```

### Plugins

```
GET /api/plugins     → List all plugins and queue stats
```

### Execute Plugin

```
POST /api/{plugin}/{method}
Content-Type: application/json
```

#### Screenshot

```bash
curl -X POST http://localhost:3000/api/screenshot/capture \
  -d '{"url":"https://example.com","width":1280,"height":720,"full_page":true}'
```

#### PDF

```bash
curl -X POST http://localhost:3000/api/pdf/generate \
  -d '{"url":"https://example.com","landscape":false,"print_background":true}'
```

#### Scrape

```bash
curl -X POST http://localhost:3000/api/scrape/extract \
  -d '{"url":"https://example.com","selectors":{"price":".price"},"wait_for":".loaded"}'
```

#### Recorder (Enhanced)

```bash
curl -X POST http://localhost:3000/api/recorder/record \
  -d '{
    "url": "https://example.com",
    "output_mode": "both",
    "steps": [
      {"action":"click","selector":"#login","delay":500},
      {"action":"type","selector":"#email","value":"test@test.com"},
      {"action":"type","selector":"#password","value":"secret"},
      {"action":"click","selector":"button[type=submit]"},
      {"action":"wait_for","selector":".dashboard","timeout":5000},
      {"action":"assert_text","selector":"h1","value":"Welcome"},
      {"action":"screenshot"}
    ]
  }'
```

New actions: `hover`, `select`, `evaluate_js`, `wait_for`, `assert_text`
Output modes: `frames` (PNG per step), `screencast` (WebP video), `both`

#### AI Browse (New)

```bash
curl -X POST http://localhost:3000/api/ai-browse/execute \
  -d '{
    "task": "Go to news.ycombinator.com, find the top 3 stories, return their titles and URLs",
    "openai_api_key": "sk-...",
    "model": "gpt-4o",
    "max_steps": 20
  }'
```

Supports any OpenAI-compatible API (OpenAI, Anthropic via proxy, Ollama, etc).
Set `base_url` to use custom endpoint: `"base_url": "http://localhost:11434/v1"` for Ollama.

### Webhooks

```
POST   /api/webhooks       → Create subscription
GET    /api/webhooks       → List subscriptions
DELETE /api/webhooks/{id}  → Delete subscription
```

Events: `job.completed`, `job.failed`, `plugin.crashed`

### Schedules

```
POST   /api/schedules       → Create scheduled task
GET    /api/schedules       → List tasks
DELETE /api/schedules/{id}  → Delete task
```

### WebSocket CDP

```
GET /cdp → WebSocket upgrade → direct CDP access
```

## Response Format

```json
{"success":true,"data":{...},"meta":{"request_id":"uuid","duration_ms":123}}
{"success":false,"error":{"code":"PLUGIN_TIMEOUT","message":"plugin execution timed out"}}
```

## Error Codes

| Code | HTTP | Description |
|------|------|-------------|
| `PLUGIN_NOT_FOUND` | 404 | Plugin doesn't exist |
| `INVALID_JSON` | 400 | Request body is not valid JSON |
| `QUEUE_FULL` | 503 | Server busy |
| `SYSTEM_OVERLOADED` | 503 | CPU/memory limit exceeded |
| `PLUGIN_TIMEOUT` | 504 | Plugin execution timed out |
| `EXECUTION_ERROR` | 500 | Plugin execution failed |
| `UNAUTHORIZED` | 401 | Invalid API key |
| `RATE_LIMITED` | 429 | Too many requests |
