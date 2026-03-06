# krawl

A CLI tool that fetches web pages and evaluates their SEO metadata.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/devforward/krawl/main/install.sh | sh
```

### Other options

Build from source:

```sh
go install github.com/devforward/krawl@latest
```

## Usage

```sh
krawl https://example.com
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `--no-audit` | Skip SEO audit, show metadata only |
| `--no-meta` | Skip metadata, show audit only |
| `-t, --timeout` | HTTP timeout (default 30s) |
| `-u, --user-agent` | Custom User-Agent string |
| `--config` | Path to config file |

### JSON output

```sh
krawl --json https://example.com
krawl --json https://example.com | jq '.audit.summary'
```

### Config

krawl looks for `.krawl.yaml` in your home directory or current directory. Settings can also be passed via `KRAWL_*` environment variables.

## License

MIT
