# krawl

A fast, single-binary CLI tool for SEO analysis. Fetch any URL and instantly see its meta tags, Open Graph data, Twitter Card tags, structured data, and more — with a built-in audit that flags common SEO issues.

Use it to check title tag length, missing meta descriptions, broken canonical URLs, missing og:image tags, and 18+ other SEO rules. Outputs both human-readable tables and machine-readable JSON for use in scripts, CI pipelines, and AI workflows.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/devforward/krawl/main/install.sh | sh
```

Or build from source:

```sh
go install github.com/devforward/krawl@latest
```

## Usage

```sh
krawl https://example.com
```

### Example output

```
╔══════════════════════════════════════════════════════════════════╗
║  krawl: https://example.com                                    ║
╚══════════════════════════════════════════════════════════════════╝

┌─ HTTP Response ─────────────────────────────────────────────────
│ Status Code              200
│ Final URL                https://example.com
│ Content-Type             text/html
│ Content-Length           528 B

┌─ Timing ────────────────────────────────────────────────────────
│ DNS Lookup               22.45ms
│ TCP Connect              14.69ms
│ TLS Handshake            24.02ms
│ Time to First Byte       94.10ms
│ Total Time               94.15ms

┌─ Page Metadata ─────────────────────────────────────────────────
│ Title                    Example Domain
│ Description              (missing)
│ Canonical                (missing)
│ Robots                   (not set - defaults to index, follow)
│ Viewport                 width=device-width, initial-scale=1
│ Language                 en

┌─ Open Graph ────────────────────────────────────────────────────
│ og:title                 (missing)
│ og:type                  (missing)
│ og:image                 (missing)
│ og:url                   (missing)
│ og:description           (missing)

┌─ Twitter Card ──────────────────────────────────────────────────
│ twitter:card             (missing)
│ twitter:title            (falls back to og:title)
│ twitter:description      (falls back to og:description)
│ twitter:image            (falls back to og:image)

╔══════════════════════════════════════════════════════════════════╗
║  SEO Audit Results                                             ║
╚══════════════════════════════════════════════════════════════════╝

┌─ Title ─────────────────────────────────────────────────────────
  ✓ Title exists                   Found: "Example Domain"
  ⚠ Title length                   Too short (14 chars). Aim for 30-60.

┌─ Description ───────────────────────────────────────────────────
  ⚠ Meta description exists        Missing meta description

┌─ Canonical ─────────────────────────────────────────────────────
  ✗ Canonical URL exists           Missing <link rel="canonical"> tag

┌─ Open Graph ────────────────────────────────────────────────────
  ✗ og:title exists                Missing og:title
  ✗ og:type exists                 Missing og:type
  ✗ og:image exists                Missing og:image
  ✗ og:url exists                  Missing og:url
  ⚠ og:description exists          Missing og:description

┌─ Technical ─────────────────────────────────────────────────────
  ⚠ Charset declared               Missing charset declaration
  ✓ Viewport meta tag              width=device-width, initial-scale=1
  ✓ HTML lang attribute            en

┌─ Headings ──────────────────────────────────────────────────────
  ✓ H1 tag exists                  Example Domain

────────────────────────────────────────────────────────────────────
  Summary: 4 passed  6 warnings  5 errors  3 info
────────────────────────────────────────────────────────────────────
```

### JSON output

Pipe to `jq` or feed directly into scripts and AI tools:

```sh
krawl --json https://example.com
krawl --json https://example.com | jq '.audit.summary'
```

```json
{
  "pass": 4,
  "warn": 6,
  "fail": 5,
  "info": 3
}
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

### Config

krawl looks for `.krawl.yaml` in your home directory or current directory. Settings can also be passed via `KRAWL_*` environment variables.

## What it checks

**HTTP & Performance** — status code, redirects, DNS/TCP/TLS/TTFB timing, content size, notable security headers

**Meta Tags** — title tag (length validation), meta description (length validation), canonical URL (absolute, HTTPS), robots directives, charset, viewport, lang attribute

**Open Graph** — og:title, og:type, og:image, og:url, og:description, og:image:alt (accessibility)

**Twitter Cards** — twitter:card (valid type), twitter:title, twitter:description, twitter:image (HTTPS), twitter:image:alt

**Structured Data** — JSON-LD detection, @context and @type validation

**Headings** — H1 existence and count (flags multiple H1s)

**Technical** — favicon, viewport zoom restrictions (accessibility), charset encoding

## License

MIT
