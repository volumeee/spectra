# 🧩 Spectra Plugin Protocol

Spectra plugins communicate with the core via **JSON-RPC 2.0 over stdin/stdout**.

## How It Works

1. Core spawns plugin binary as a subprocess (N processes per plugin via `pool_size`)
2. Core writes JSON-RPC requests to plugin's **stdin**
3. Plugin reads stdin, processes, writes JSON-RPC responses to **stdout**
4. Plugin logs go to **stderr** (stdout is reserved for JSON-RPC)
5. If plugin hangs beyond `call_timeout`, core kills the process and restarts on next call

## Message Format

Newline-delimited JSON. One JSON object per line.

### Request (core → plugin)

```json
{"jsonrpc":"2.0","id":1,"method":"capture","params":{"url":"https://example.com","width":1920,"height":1080}}
```

When `browser.share_pool=true`, core injects `_cdp_endpoint` into params:

```json
{"jsonrpc":"2.0","id":1,"method":"capture","params":{"url":"https://example.com","_cdp_endpoint":"ws://127.0.0.1:9222/..."}}
```

### Response (plugin → core)

Success:
```json
{"jsonrpc":"2.0","id":1,"result":{"data":"base64...","size_bytes":45000}}
```

Error:
```json
{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"something went wrong"}}
```

## Plugin Manifest (`plugin.json`)

Place next to the plugin binary as `{binary-name}.json`:

```json
{
  "name": "my-plugin",
  "version": "0.1.0",
  "command": "my-plugin",
  "methods": ["my_method", "another_method"]
}
```

## Error Codes

| Code | Meaning |
|------|---------|
| -32700 | Parse error |
| -32600 | Invalid request |
| -32601 | Method not found |
| -32603 | Internal error |

## Process Lifecycle

- Plugins are **lazy-started** on first call
- Each plugin has a **process pool** of N instances (`plugins.pool_size` in config)
- Requests are routed **round-robin** to healthy processes
- If a process crashes, it's **auto-restarted** on next call
- If a call exceeds `plugins.call_timeout`, the process is **killed** and restarted

## Testing Your Plugin

```bash
echo '{"jsonrpc":"2.0","id":1,"method":"capture","params":{"url":"https://example.com"}}' | ./my-plugin
```
