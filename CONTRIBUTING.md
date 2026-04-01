# Contributing to Spectra 🔮

We welcome contributions! Here's how to get started.

## Development Setup

```bash
# Clone
git clone https://github.com/volumeee/spectra.git
cd spectra

# Requirements
# - Go 1.23+
# - Chromium (for browser plugin testing)

# Build everything (core + CLI + all plugins)
make build-all

# Run server
./bin/spectra

# Run full test suite
bash scripts/install-and-test.sh

# Run unit tests only
make test
```

## Project Structure

```
spectra/
├── cmd/
│   ├── spectra/           → Server entrypoint
│   └── spectra-cli/       → CLI entrypoint
├── internal/
│   ├── domain/            → Pure types, zero dependencies
│   ├── port/              → Interfaces (contracts between layers)
│   ├── adapter/           → Implementations
│   │   ├── browser/       → Chromium pool (warm + recycle)
│   │   ├── plugin/        → Plugin process pool + JSON-RPC
│   │   ├── queue/         → Job queue with health check
│   │   ├── storage/       → SQLite persistence
│   │   ├── monitor/       → CPU/memory health monitoring
│   │   ├── metrics/       → Request metrics collector
│   │   ├── recording/     → CDP screencast recording
│   │   ├── webhook/       → Webhook engine + store
│   │   ├── scheduler/     → Cron scheduler
│   │   └── cdpproxy/      → WebSocket CDP proxy
│   ├── api/
│   │   ├── handler/       → HTTP handlers
│   │   └── middleware/     → Auth, rate limit, logging, request ID
│   ├── mcp/               → MCP server (AI agent integration)
│   └── config/            → YAML + env config loader
├── pkg/protocol/          → JSON-RPC types (public)
├── sdk/go/                → Go Plugin SDK (BrowserOptions, OpenPage)
├── plugins/
│   ├── screenshot/        → Screenshot plugin
│   ├── pdf/               → PDF plugin
│   ├── scrape/            → Scrape plugin
│   ├── stealth/           → Stealth plugin
│   ├── visual-diff/       → Visual diff plugin
│   ├── recorder/          → Session recorder plugin
│   └── ai/                → AI agent plugin (act/extract/observe/execute/plan)
├── client/                → Client SDKs (Go, TypeScript, Python, Rust)
├── docker/                → Dockerfile + docker-compose
├── docs/                  → Documentation
└── scripts/               → Build + test scripts
```

## Architecture Rules

Spectra follows **hexagonal architecture** (ports & adapters):

1. **Dependencies point inward only:** `cmd → api → adapter → port ← domain`
2. **Domain has zero imports** — pure types and errors
3. **Ports define interfaces** — all contracts live in `internal/port/`
4. **Adapters implement ports** — one adapter per external concern
5. **API layer uses ports only** — never imports adapters directly
6. **Plugins are subprocesses** — JSON-RPC over stdin/stdout, crash-isolated

## How to Contribute

### Bug fix or feature

1. Fork the repo
2. Create a branch: `git checkout -b feat/my-feature`
3. Make changes following the architecture rules above
4. Run tests: `bash scripts/install-and-test.sh`
5. Submit PR with clear description

### Build a plugin

The easiest way to contribute. Plugins are standalone binaries.

```go
package main

import (
    "context"
    "encoding/json"
    spectra "github.com/spectra-browser/spectra/sdk/go"
)

type MyParams struct {
    spectra.BrowserOptions
    URL string `json:"url"`
}

func main() {
    p := spectra.NewPlugin("my-plugin")
    p.Handle("my-method", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
        var p MyParams
        json.Unmarshal(params, &p)
        s, _ := spectra.OpenPage(p.URL, p.BrowserOptions)
        defer s.Close()
        return map[string]interface{}{"title": s.Page.MustEval(`() => document.title`).String()}, nil
    })
    p.Run()
}
```

Steps:
1. Read [Plugin Development Guide](docs/plugin-development.md)
2. Create `plugins/my-plugin/main.go` + `plugin.json`
3. Test: `echo '{"jsonrpc":"2.0","id":1,"method":"my-method","params":{"url":"https://example.com"}}' | ./my-plugin`
4. Submit PR

### Add a new port interface

If you need a new capability:
1. Define the interface in `internal/port/`
2. Implement it in `internal/adapter/your-adapter/`
3. Wire it in `cmd/spectra/main.go`
4. Add compile-time check: `var _ port.YourInterface = (*YourAdapter)(nil)`

## Code Style

- Go: `gofmt` + `golangci-lint`
- Keep functions small and focused
- Interfaces in `port/`, implementations in `adapter/`
- Errors should be descriptive and wrapped with `fmt.Errorf("context: %w", err)`
- No global state — everything injected via constructors
- DRY: shared browser logic lives in `sdk/go/browser.go`

## Commit Messages

```
feat: add visual diff plugin
fix: browser pool deadlock on shutdown
docs: update API reference
refactor: extract BrowserOptions to SDK
test: add session persistence E2E test
```

## Testing

```bash
# Full E2E test (build + server + all endpoints)
bash scripts/install-and-test.sh

# With AI plugin tests
OPENAI_API_KEY=sk-... bash scripts/install-and-test.sh

# Unit tests only
go test ./...
```

## Questions?

Open an issue or start a discussion on GitHub.

💜 Thank you for contributing!
