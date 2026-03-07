# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What is krawl

krawl is a single-binary CLI tool for SEO analysis written in Go. It fetches URLs and analyzes meta tags, Open Graph, Twitter Cards, structured data (JSON-LD), links, and XML sitemaps. It outputs human-readable tables (with color) or JSON (`-j`).

## Build & Run

```sh
go build -o krawl .                    # build
go run . https://example.com           # run directly
go run . links https://example.com     # link checker
go run . sitemap https://example.com/sitemap.xml  # sitemap validator
```

Version is stamped at build time via `-ldflags "-X github.com/devforward/krawl/cmd.Version=vX.Y.Z"` (see `cmd/upgrade.go:14`).

There are no tests, no linter config, and no Makefile.

## Architecture

The CLI uses [cobra](https://github.com/spf13/cobra) for commands and [viper](https://github.com/spf13/viper) for config (`.krawl.yaml` or `KRAWL_*` env vars).

### Data flow (root command)

```
fetcher.Fetch(url) → parser.ParseWithURL(body, url) → rules.Evaluate(seoData, fetchResult) → display.Print*(...)
```

### Packages

- **`cmd/`** — Cobra command definitions. Each command file (`root.go`, `links.go`, `sitemap.go`, `upgrade.go`) wires together the internal packages. Commands handle their own JSON output formatting inline.
- **`internal/fetcher/`** — HTTP client with `httptrace` timing (DNS, TCP, TLS, TTFB). Returns `fetcher.Result` with body, headers, timing, and redirect chain.
- **`internal/parser/`** — Three parsers, each in its own file:
  - `parser.go` — HTML parsing → `SEOData` struct. Walks `<head>` fully (meta tags, OG, Twitter, JSON-LD, favicons). `parseBody` walks `<body>` for headings (H1-H6 with hierarchy), images, content metrics (word count, text-to-HTML ratio), and link stats.
  - `links.go` — Extracts `<a>` hrefs, resolves relative URLs, deduplicates. `IsInternal()` compares hosts.
  - `sitemap.go` — XML sitemap/index parsing with validation (URL limits, lastmod format, duplicates, cross-domain, priority/changefreq values).
- **`internal/rules/`** — SEO audit engine. `Evaluate()` takes `SEOData` + `fetcher.Result` and runs all rule functions, each returning `[]Result` with category, severity (Pass/Info/Warning/Error), and message. Rule categories: Title, Description, Canonical, Robots, Open Graph, Twitter Card, Technical, Headings, Structured Data, Images, Content, Redirects, Links.
- **`internal/display/`** — Terminal output (`display.go`) with box-drawing and color via `fatih/color`. JSON output (`json.go`) with dedicated struct types that mirror the display layout.

### Key patterns

- All commands follow the same pattern: fetch → parse → (optionally evaluate) → display or JSON encode.
- The `links` command does concurrent HEAD requests with a semaphore for concurrency control, falling back to GET if HEAD is rejected.
- Severity levels in `rules` package: `SeverityPass` (0), `SeverityInfo` (1), `SeverityWarning` (2), `SeverityError` (3).
- JSON output structs are separate from internal data types — see `display/json.go` for the full JSON schema.
