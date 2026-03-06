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
krawl https://devforward.com
```

### Example output

```
╔══════════════════════════════════════════════════════════════════╗
║  krawl: https://devforward.com                                 ║
╚══════════════════════════════════════════════════════════════════╝

┌─ HTTP Response ─────────────────────────────────────────────────
│ Status Code              200
│ Final URL                https://devforward.com
│ Content-Type             text/html; charset=utf-8
│ Content-Length           41.8 KB

┌─ Timing ────────────────────────────────────────────────────────
│ DNS Lookup               69.18ms
│ TCP Connect              15.88ms
│ TLS Handshake            26.36ms
│ Time to First Byte       170.85ms
│ Total Time               170.88ms

┌─ Notable Headers ───────────────────────────────────────────────
│ Cache-Control            public,max-age=10,s-maxage=86400
│ Server                   cloudflare

┌─ Page Metadata ─────────────────────────────────────────────────
│ Title                    Dev Forward - Founder-Led Product Studio
│ Description              Dev Forward is a founder-led AI product studio. We build and operate our own software products, and partner selectively with founders as a strategic CTO and build partner.
│ Canonical                https://devforward.com
│ Charset                  utf-8
│ Viewport                 width=device-width, initial-scale=1
│ Language                 en

┌─ Open Graph ────────────────────────────────────────────────────
│ og:title                 Dev Forward - Founder-Led Product Studio
│ og:type                  website
│ og:image                 https://devforward.com/og-image.png
│ og:url                   (missing)
│ og:description           Dev Forward is a founder-led product studio building AI-enabled software and partnering with ambitious startups to design, build, and scale technology businesses.

┌─ Twitter Card ──────────────────────────────────────────────────
│ twitter:card             summary_large_image
│ twitter:title            Dev Forward - Founder-Led Product Studio
│ twitter:image            https://devforward.com/og-image.png

┌─ Structured Data (JSON-LD) ─────────────────────────────────────
│ Block #1                 (unknown type)

╔══════════════════════════════════════════════════════════════════╗
║  SEO Audit Results                                             ║
╚══════════════════════════════════════════════════════════════════╝

┌─ Title ─────────────────────────────────────────────────────────
  ✓ Title exists                   Found: "Dev Forward - Founder-Led Product Studio"
  ✓ Title length                   40 chars (30-60 recommended)

┌─ Description ───────────────────────────────────────────────────
  ✓ Meta description exists        Found: "Dev Forward is a founder-led AI product studio. We build and operate our own software products, and partner selectively ..."
  ⚠ Description length             Too long (171 chars). May be truncated. Aim for 70-160.

┌─ Canonical ─────────────────────────────────────────────────────
  ✓ Canonical URL exists           https://devforward.com
  ✓ Canonical is absolute URL      Uses absolute URL
  ✓ Canonical uses HTTPS           Uses HTTPS

┌─ Open Graph ────────────────────────────────────────────────────
  ✓ og:title exists                Dev Forward - Founder-Led Product Studio
  ✓ og:type exists                 website
  ✓ og:image exists                https://devforward.com/og-image.png
  ✗ og:url exists                  Missing og:url
  ✓ og:description exists          Dev Forward is a founder-led product studio building AI-enabled software and partnering with ambitious startups to desig...
  ⚠ og:image:alt exists            Missing og:image:alt (accessibility)

┌─ Twitter Card ──────────────────────────────────────────────────
  ✓ twitter:card exists            summary_large_image
  ✓ twitter:title exists           Dev Forward - Founder-Led Product Studio
  ⚠ twitter:image:alt exists       Missing twitter:image:alt (accessibility)

┌─ Technical ─────────────────────────────────────────────────────
  ✓ Charset declared               utf-8
  ✓ Viewport meta tag              width=device-width, initial-scale=1
  ✓ HTML lang attribute            en
  ✓ Favicon                        Favicon declared

┌─ Headings ──────────────────────────────────────────────────────
  ✓ H1 tag exists                  Dev Forward

┌─ Structured Data ───────────────────────────────────────────────
  ✓ JSON-LD exists                 Found 1 block(s)
  ⚠ JSON-LD #1 @type               Missing @type

────────────────────────────────────────────────────────────────────
  Summary: 18 passed  4 warnings  1 errors  1 info
────────────────────────────────────────────────────────────────────
```

### JSON output

Pipe to `jq` or feed directly into scripts and AI tools:

```sh
krawl --json https://devforward.com
krawl --json https://devforward.com | jq '.audit.summary'
```

```json
{
  "pass": 18,
  "warn": 4,
  "fail": 1,
  "info": 1
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
