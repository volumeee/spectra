<div align="center">

🔮

# Spectra

**Open-source headless browser infrastructure — pluggable, AI-native, production-grade.**

[![Go](https://img.shields.io/badge/Go-00ADD8?style=flat&logo=go&logoColor=white)]()
[![Rust](https://img.shields.io/badge/Rust-000000?style=flat&logo=rust&logoColor=white)]()
[![TypeScript](https://img.shields.io/badge/TypeScript-3178C6?style=flat&logo=typescript&logoColor=white)]()
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-ready-2496ED?logo=docker)]()

📸 Screenshot · 📄 PDF · 🕷️ Scrape · 🤖 AI Agent · 👻 Stealth · 🔍 Visual Diff · 🎥 Recorder

[Quick Start](#-quick-start) · [Plugins](#-built-in-plugins) · [API](#-api) · [AI Agent](#-ai-agent) · [Plugin SDK](#-plugin-sdk) · [Configuration](#-configuration) · [Contributing](#-contributing)

</div>

---

## 🚀 Quick Start

### Docker

```bash
git clone https://github.com/spectra-browser/spectra.git
cd spectra
docker compose -f docker/docker-compose.yml up
```

### Build from source

```bash
git clone https://github.com/spectra-browser/spectra.git
cd spectra
make build-all    # builds core + CLI + all plugins
./bin/spectra     # starts server on :3000
```

### Try it

```bash
# Screenshot at 1920x1080 (default)
curl -X POST http://localhost:3000/api/screenshot/capture \
  -H "Content-Type: application/json" \
  -d '{"url": "https://example.com"}'

# AI agent — autonomous browser task
curl -X POST http://localhost:3000/api/ai/execute \
  -H "Content-Type: application/json" \
  -d '{"task": "Find top 3 HN stories", "openai_api_key": "sk-..."}'

# Multi-step query (SpectraQL)
curl -X POST http://localhost:3000/api/query \
  -H "Content-Type: application/json" \
  -d '{"steps": [{"action":"goto","url":"https://example.com"},{"action":"screenshot"}]}'
```

---

## 🌐 7 Ways to Use Spectra

| # | Method | For | Example |
|---|--------|-----|---------|
| 📡 | **REST API** | Backend services, curl | `POST /api/screenshot/capture` |
| 🔗 | **SpectraQL** | Multi-step workflows in one request | `POST /api/query` |
| 💻 | **CLI Tool** | Terminal, shell scripts, cron | `spectra-cli screenshot https://example.com` |
| 📦 | **Client SDK** | Go, TypeScript, Python, Rust | `client.Screenshot(ctx, params)` |
| 🤖 | **MCP Server** | AI agents (Claude, ChatGPT, Gemini) | AI: "screenshot this page" |
| 🔌 | **WebSocket CDP** | Puppeteer, Playwright direct | `puppeteer.connect({browserWSEndpoint})` |
| 🔔 | **Webhooks + Scheduler** | Event-driven, cron jobs | "Every hour, screenshot dashboard" |

---

## 🏗️ Architecture

```
  Request ──→ 🔮 Spectra Core (Go)
                    │
         ┌──────────┼──────────┐
         ▼          ▼          ▼
    Health Check  Job Queue  Browser Pool
    (CPU/Memory)  (semaphore) (warm + recycle)
                    │
            Plugin Process Pool
            (N concurrent per plugin)
                    │
              JSON-RPC stdio
                    │
    ┌────────┬──────┼──────┬─────────┐
    ▼        ▼      ▼      ▼         ▼
 screenshot  pdf   scrape  ai     recorder
                        (act/extract
                         /observe
                         /execute)
                    │
              Chromium (CDP)
```

**Clean Architecture:** Domain → Ports → Adapters → API  
**Plugin System:** JSON-RPC over stdin/stdout — write plugins in any language  
**Shared Browser Pool:** Core injects CDP endpoint into plugins — 5-10x faster  
**Warm Pool:** Pre-launched browsers = zero cold-start latency

---

## 🧩 Built-in Plugins

| Plugin | Methods | Description |
|--------|---------|-------------|
| 📸 **screenshot** | `capture` | Full-page or viewport screenshots, PNG/JPEG |
| 📄 **pdf** | `generate` | Generate PDFs with custom margins, landscape |
| 🕷️ **scrape** | `extract` | Title, text, links, images, meta, custom selectors |
| 👻 **stealth** | `navigate`, `screenshot`, `scrape` | Bypass bot detection (webdriver, WebGL, permissions) |
| 🔍 **visual-diff** | `compare` | Pixel-by-pixel diff between two URLs |
| 🎥 **recorder** | `record` | Multi-step session recording with per-step screenshots |
| 🤖 **ai** | `act`, `extract`, `observe`, `execute`, `plan` | AI browser agent with planning + self-correction |

All plugins share `BrowserOptions` — viewport, headless, proxy, and CDP pool are configured once.

---

## 📡 API

### Core Endpoints

```
GET  /health                    → Liveness check
GET  /ready                     → Readiness (pool, queue, plugin stats)
GET  /pressure                  → System load — returns 503 when CPU/memory overloaded
GET  /api/plugins               → List all loaded plugins
GET  /api/metrics               → Request counts, success rates, avg duration per plugin
GET  /api/jobs                  → Job history (requires storage.driver=sqlite for persistence)
GET  /cdp                       → WebSocket CDP proxy (Puppeteer/Playwright direct)
GET  /api/sessions/:id/live     → Live browser view via WebSocket
```

### Plugin Execution

```bash
# Any plugin, any method
POST /api/{plugin}/{method}

# Examples
curl -X POST http://localhost:3000/api/screenshot/capture \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","full_page":true}'

curl -X POST http://localhost:3000/api/pdf/generate \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","landscape":true}'

curl -X POST http://localhost:3000/api/scrape/extract \
  -H "Content-Type: application/json" \
  -d '{"url":"https://example.com","selectors":{"price":".product-price"}}'
```

### BrowserOptions — shared across all plugins

Every plugin accepts these optional fields for dynamic viewport and browser control:

```json
{
  "url": "https://example.com",
  "width": 1920,
  "height": 1080,
  "headless": true,
  "extra_flags": {
    "proxy-server": "http://proxy:8080",
    "user-agent": "Mozilla/5.0 Custom"
  }
}
```

| Field | Default | Description |
|-------|---------|-------------|
| `width` | `1920` | Viewport width in pixels |
| `height` | `1080` | Viewport height in pixels |
| `headless` | `true` | Run browser without GUI |
| `extra_flags` | `{}` | Any Chromium CLI flag |

### SpectraQL — multi-step in one request

```bash
curl -X POST http://localhost:3000/api/query \
  -H "Content-Type: application/json" \
  -d '{
    "steps": [
      {"action": "goto", "url": "https://example.com"},
      {"action": "wait_for", "selector": ".content"},
      {"action": "click", "selector": "#load-more"},
      {"action": "screenshot"}
    ]
  }'
```

Actions: `goto`, `click`, `type`, `scroll`, `wait_for`, `evaluate_js`, `screenshot`, `extract`

### Recorder — multi-step with per-step screenshots

```bash
curl -X POST http://localhost:3000/api/recorder/record \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://example.com",
    "output_mode": "both",
    "steps": [
      {"action": "type", "selector": "#email", "value": "user@example.com"},
      {"action": "click", "selector": "button[type=submit]"},
      {"action": "wait_for", "selector": ".dashboard", "timeout": 5000},
      {"action": "assert_text", "selector": "h1", "value": "Welcome"}
    ]
  }'
```

Actions: `navigate`, `click`, `type`, `hover`, `select`, `scroll`, `evaluate_js`, `wait_for`, `assert_text`, `screenshot`, `wait`  
Output modes: `frames` (PNG per step) · `both`

### Sessions (requires `storage.driver: sqlite`)

Persistent browser sessions that maintain cookies and localStorage across requests.

```bash
# Create session
curl -X POST http://localhost:3000/api/sessions \
  -H "Content-Type: application/json" \
  -d '{"ttl_seconds": 3600}'

# List / Get / Delete
curl http://localhost:3000/api/sessions
curl http://localhost:3000/api/sessions/{id}
curl -X DELETE http://localhost:3000/api/sessions/{id}
```

### Browser Profiles (requires `storage.driver: sqlite`)

Persistent browser identities for consistent fingerprinting across sessions.

```bash
curl -X POST http://localhost:3000/api/profiles \
  -H "Content-Type: application/json" \
  -d '{"name":"us-chrome","locale":"en-US","timezone":"America/New_York","stealth_level":"advanced"}'
```

### Webhooks

```bash
curl -X POST http://localhost:3000/api/webhooks \
  -H "Content-Type: application/json" \
  -d '{"event":"job.completed","target_url":"https://your-server.com/hook","secret":"hmac-secret"}'
```

Events: `job.completed` · `job.failed` · `plugin.crashed`

### Schedules

```bash
curl -X POST http://localhost:3000/api/schedules \
  -H "Content-Type: application/json" \
  -d '{"cron":"0 * * * *","plugin":"screenshot","method":"capture","params":{"url":"https://example.com"}}'
```

---

## 🤖 AI Agent

The `ai` plugin provides 5 methods — from single actions to full autonomous agents:

### `act` — single action via accessibility tree

```bash
curl -X POST http://localhost:3000/api/ai/act \
  -H "Content-Type: application/json" \
  -d '{"instruction":"click the login button","openai_api_key":"sk-..."}'
```

Uses **accessibility tree** (not full DOM) — 10-50x fewer tokens. Results are **cached** per domain.

### `extract` — structured data extraction

```bash
curl -X POST http://localhost:3000/api/ai/extract \
  -H "Content-Type: application/json" \
  -d '{"instruction":"get all product prices","schema":{"products":[{"name":"string","price":"number"}]},"openai_api_key":"sk-..."}'
```

### `observe` — preview possible actions (no execute)

```bash
curl -X POST http://localhost:3000/api/ai/observe \
  -H "Content-Type: application/json" \
  -d '{"openai_api_key":"sk-..."}'
```

### `execute` — full autonomous agent

```bash
curl -X POST http://localhost:3000/api/ai/execute \
  -H "Content-Type: application/json" \
  -d '{
    "task": "Login to GitHub and star the repo spectra-browser/spectra",
    "url": "https://github.com",
    "openai_api_key": "sk-...",
    "model": "gpt-4o",
    "max_steps": 20,
    "config": {"planning": true, "self_correction": true, "memory": true}
  }'
```

Features: **planning** (step-by-step plan before executing) · **self-correction** (recovers from errors) · **memory** (caches learned selectors, gets cheaper over time)

Supports any OpenAI-compatible API. Use `base_url` for custom endpoints:
```json
{"base_url": "http://localhost:11434/v1", "model": "llama3"}
```

### `plan` — dry run (no browser needed)

```bash
curl -X POST http://localhost:3000/api/ai/plan \
  -H "Content-Type: application/json" \
  -d '{"task":"Scrape all products from page 1-5","openai_api_key":"sk-..."}'
```

### Cost comparison

| Method | LLM Calls | Tokens/call | Cost (8 steps) |
|--------|-----------|-------------|----------------|
| Simple loop (full DOM) | 8 | ~2000 | ~$0.08 |
| `act` (a11y tree) | 8 | ~200 | ~$0.008 |
| `execute` + cache | 1-3 | ~500 | ~$0.003 |

---

## ⚙️ Configuration

See [spectra.example.yaml](spectra.example.yaml) for all options.

```yaml
server:
  port: 3000

browser:
  max_instances: 5
  warm_pool_size: 2       # pre-launch browsers (zero cold-start)
  recycle_after: 50       # recycle after N uses (prevent memory drift)
  share_pool: true        # inject CDP endpoint into plugins (5-10x faster)

plugins:
  pool_size: 3            # concurrent processes per plugin
  call_timeout: 60s       # kills hung plugins automatically

health:
  enabled: true
  cpu_limit: 90           # reject requests when CPU > 90%
  memory_limit: 85

storage:
  driver: sqlite          # "memory" or "sqlite" — sqlite persists sessions, profiles, webhooks, schedules, job history
  sqlite_path: ./spectra.db

auth:
  enabled: false
  api_key: ""

webhook:
  enabled: false
scheduler:
  enabled: false
mcp:
  enabled: false
```

Environment variables override config: `SPECTRA_SERVER_PORT=8080`, `SPECTRA_BROWSER_SHARE_POOL=true`, etc.

---

## 🧩 Plugin SDK

Build plugins in any language. The Go SDK provides `BrowserOptions` for zero-boilerplate browser access.

### Go (recommended)

```go
package main

import (
    "context"
    "encoding/json"
    spectra "github.com/spectra-browser/spectra/sdk/go"
)

type MyParams struct {
    spectra.BrowserOptions          // width, height, headless, proxy — free
    URL     string `json:"url"`
}

func main() {
    p := spectra.NewPlugin("my-plugin")
    p.Handle("my-method", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
        var p MyParams
        json.Unmarshal(params, &p)

        s, err := spectra.OpenPage(p.URL, p.BrowserOptions)
        if err != nil { return nil, err }
        defer s.Close()

        return map[string]interface{}{
            "title": s.Page.MustEval(`() => document.title`).String(),
        }, nil
    })
    p.Run()
}
```

Three levels of browser access:
- `spectra.OpenPage(url, opts)` — opens URL, sets viewport, waits for load
- `spectra.OpenBlankPage(opts)` — blank page for manual navigation (stealth, multi-step)
- `spectra.ConnectOrLaunch(opts)` — direct `*rod.Browser` access (multi-page, CDP events)

### TypeScript

```typescript
import { createPlugin } from '@spectra/plugin-sdk';
const plugin = createPlugin('my-plugin');
plugin.handle('my-method', async (params) => ({ result: 'done' }));
plugin.run();
```

### Python (no SDK needed)

```python
import json, sys
for line in sys.stdin:
    req = json.loads(line)
    print(json.dumps({"jsonrpc":"2.0","id":req["id"],"result":{"done":True}}), flush=True)
```

See [Plugin Development Guide](docs/plugin-development.md) for full documentation.

---

## 🧪 Testing

```bash
# Run full test suite (build + unit tests + E2E)
bash scripts/install-and-test.sh

# With AI plugin tests (requires OpenAI API key + Chromium)
OPENAI_API_KEY=sk-... bash scripts/install-and-test.sh
```

---

## 🤝 Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## 📄 License

MIT License — see [LICENSE](LICENSE) for details.

💜 Built with love by the Spectra community.
