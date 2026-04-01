# 💻 Spectra CLI Reference

## Installation

```bash
go install github.com/volumeee/spectra/cmd/spectra-cli@latest
```

Or build from source:

```bash
make build-cli
./bin/spectra-cli --help
```

## Global Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--server` | `http://localhost:3000` | Spectra server URL |
| `--api-key` | | API key for authentication |
| `-o, --output` | stdout | Output file path |

## Commands

### screenshot

```bash
spectra-cli screenshot https://example.com
spectra-cli screenshot https://example.com --width 1920 --height 1080 --full-page
spectra-cli screenshot https://example.com -o screenshot.png
```

### pdf

```bash
spectra-cli pdf https://example.com
spectra-cli pdf https://example.com --landscape -o output.pdf
```

### scrape

```bash
spectra-cli scrape https://example.com
```

### plugins

```bash
spectra-cli plugins
```

### health

```bash
spectra-cli health
```

### exec (any plugin, any method)

```bash
spectra-cli exec screenshot capture '{"url":"https://example.com"}'
spectra-cli exec ai plan '{"task":"scrape HN","openai_api_key":"sk-..."}'
spectra-cli exec recorder record '{"url":"https://example.com","steps":[{"action":"screenshot"}]}'
```
