# 🤖 Spectra MCP Server

Spectra runs as an MCP (Model Context Protocol) server, allowing AI agents like Claude, ChatGPT, and Gemini to use browser capabilities.

## Setup

```yaml
# spectra.yaml
mcp:
  enabled: true
  transport: stdio  # or "sse"
```

## Available Tools

| Tool | Description |
|------|-------------|
| `spectra_screenshot` | Take a screenshot of a web page |
| `spectra_pdf` | Generate a PDF from a web page |
| `spectra_scrape` | Scrape and extract structured data |
| `spectra_record` | Record a multi-step browser session |
| `spectra_ai_act` | Single browser action via natural language |
| `spectra_ai_extract` | Extract structured data using LLM + schema |
| `spectra_ai_observe` | List possible actions on current page |
| `spectra_ai_execute` | Full autonomous AI agent for complex tasks |
| `spectra_ai_plan` | Generate execution plan (dry run) |
| `spectra_execute` | Execute any plugin method |
| `spectra_plugins` | List available plugins |

## Claude Desktop Integration

Add to `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "spectra": {
      "command": "./bin/spectra",
      "args": ["--mcp"]
    }
  }
}
```

Then ask Claude:
- "Take a screenshot of https://example.com"
- "Scrape the top stories from Hacker News"
- "Click the login button on this page" (uses `spectra_ai_act`)
- "Extract all product prices from this page" (uses `spectra_ai_extract`)

## SSE Transport

For remote MCP connections, use SSE transport. MCP server runs on port+1:

```yaml
mcp:
  enabled: true
  transport: sse  # runs on :3001
```

## Custom AI Agent (Python)

```python
from spectra_client import SpectraClient

client = SpectraClient("http://localhost:3000")
result = client.screenshot(url="https://example.com")
data = client.scrape(url="https://news.example.com")
```
