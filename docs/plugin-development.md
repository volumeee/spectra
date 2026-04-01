# 🧩 Plugin Development Guide

Build a Spectra plugin in any language. All you need: read stdin, write stdout, JSON-RPC 2.0.

## Go SDK (recommended)

The Go SDK provides `BrowserOptions` — a single shared struct for all browser configuration. Embed it in your params struct and get viewport, headless, proxy, and CDP pool support for free.

```go
package main

import (
    "context"
    "encoding/json"
    spectra "github.com/spectra-browser/spectra/sdk/go"
)

type MyParams struct {
    spectra.BrowserOptions          // width, height, headless, extra_flags, _cdp_endpoint
    URL     string `json:"url"`
    // Add any custom fields here
}

func main() {
    p := spectra.NewPlugin("my-plugin")

    p.Handle("my-method", func(ctx context.Context, params json.RawMessage) (interface{}, error) {
        var p MyParams
        if err := json.Unmarshal(params, &p); err != nil {
            return nil, err
        }

        // OpenPage: connects to pool browser or launches new one, sets viewport
        s, err := spectra.OpenPage(p.URL, p.BrowserOptions)
        if err != nil {
            return nil, err
        }
        defer s.Close() // closes page; closes browser only if we launched it

        // s.Page is a *rod.Page ready at ViewportWidth x ViewportHeight
        title := s.Page.MustEval(`() => document.title`).String()
        return map[string]interface{}{"title": title}, nil
    })

    p.Run()
}
```

### BrowserOptions fields

| Field | Default | Description |
|-------|---------|-------------|
| `width` | `1920` | Viewport width |
| `height` | `1080` | Viewport height |
| `headless` | `true` | Headless mode |
| `extra_flags` | `{}` | Any Chromium CLI flag |
| `_cdp_endpoint` | auto | Injected by core when `share_pool=true` |

### Helper functions

```go
// OpenPage — opens URL, sets viewport, waits for load
s, err := spectra.OpenPage(url, opts)
defer s.Close()
// s.Page, s.Browser available

// OpenBlankPage — blank page for manual navigation (stealth, multi-step)
s, err := spectra.OpenBlankPage(opts)
defer s.Close()

// ConnectOrLaunch — low-level, for multi-page or CDP event plugins
browser, owned, err := spectra.ConnectOrLaunch(opts)
defer func() { if owned { browser.MustClose() } }()
page1, _ := browser.Page(...)
page2, _ := browser.Page(...)
```

### Dynamic viewport example

```bash
# Default 1920x1080
curl -X POST http://localhost:3000/api/my-plugin/my-method \
  -d '{"url":"https://example.com"}'

# Custom viewport
curl -X POST http://localhost:3000/api/my-plugin/my-method \
  -d '{"url":"https://example.com","width":375,"height":812}'

# With proxy
curl -X POST http://localhost:3000/api/my-plugin/my-method \
  -d '{"url":"https://example.com","extra_flags":{"proxy-server":"http://proxy:8080"}}'

# Headless off (GUI mode)
curl -X POST http://localhost:3000/api/my-plugin/my-method \
  -d '{"url":"https://example.com","headless":false}'
```

## TypeScript SDK

```typescript
import { createPlugin } from '@spectra/plugin-sdk';

const plugin = createPlugin('my-plugin');

plugin.handle('my-method', async (params) => {
    return { message: 'Hello from TypeScript!' };
});

plugin.run();
```

## Rust SDK

```rust
use spectra_plugin_sdk::Plugin;
use serde_json::json;

fn main() {
    let mut plugin = Plugin::new("my-plugin");
    plugin.handle("my-method", |_params| {
        Ok(json!({"message": "Hello from Rust!"}))
    });
    plugin.run();
}
```

## Python (no SDK needed)

```python
import json, sys

for line in sys.stdin:
    req = json.loads(line)
    if req["method"] == "my-method":
        resp = {"jsonrpc": "2.0", "id": req["id"], "result": {"message": "Hello from Python!"}}
    else:
        resp = {"jsonrpc": "2.0", "id": req["id"], "error": {"code": -32601, "message": "not found"}}
    print(json.dumps(resp), flush=True)
```

## Steps to create a plugin

1. Create `plugins/my-plugin/` directory
2. Write plugin code (embed `BrowserOptions` if using browser)
3. Create `plugin.json` manifest
4. Build: `go build -o bin/plugins/my-plugin ./plugins/my-plugin/`
5. Restart Spectra — plugin auto-discovered
6. Call: `POST /api/my-plugin/my-method`

## plugin.json manifest

```json
{
  "name": "my-plugin",
  "version": "0.1.0",
  "command": "my-plugin",
  "methods": ["my-method"]
}
```

## Testing

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"my-method","params":{"url":"https://example.com"}}' \
  | ./bin/plugins/my-plugin
```
