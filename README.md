# krawl

A fast, single-binary CLI tool for SEO analysis. Fetch any URL and instantly see its meta tags, Open Graph data, Twitter Card tags, structured data, and more — with a built-in audit that flags common SEO issues.

Use it to check title tag length, missing meta descriptions, broken canonical URLs, missing og:image tags, image alt text, heading hierarchy, content quality, redirect chains, link metrics, and 30+ other SEO rules. Outputs both human-readable tables and machine-readable JSON for use in scripts, CI pipelines, and AI workflows.

## Install

```sh
curl -fsSL https://raw.githubusercontent.com/devforward/krawl/main/install.sh | sh
```

Or build from source:

```sh
go install github.com/devforward/krawl@latest
```

Upgrade to the latest version:

```sh
krawl upgrade
```

## Commands

| Command | Description |
|---------|-------------|
| `krawl <url>` | SEO audit — meta tags, Open Graph, Twitter Cards, structured data |
| `krawl links <url>` | Check all internal and external links for broken URLs |
| `krawl sitemap <url>` | Fetch and validate an XML sitemap |
| `krawl upgrade` | Self-update to the latest release |

## SEO Audit

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

┌─ Page Metadata ─────────────────────────────────────────────────
│ Title                    Dev Forward - Founder-Led Product Studio
│ Description              Dev Forward is a founder-led AI product studio. We build and
│                          operate our own software products, and partner selectively
│                          with founders as a strategic CTO and build partner.
│ Canonical                https://devforward.com
│ Viewport                 width=device-width, initial-scale=1
│ Language                 en

┌─ Open Graph ────────────────────────────────────────────────────
│ og:title                 Dev Forward - Founder-Led Product Studio
│ og:type                  website
│ og:image                 https://devforward.com/og-image.png
│ og:url                   (missing)
│ og:description           Dev Forward is a founder-led product studio building
│                          AI-enabled software and partnering with ambitious startups
│                          to design, build, and scale technology businesses.

┌─ Twitter Card ──────────────────────────────────────────────────
│ twitter:card             summary_large_image
│ twitter:title            Dev Forward - Founder-Led Product Studio
│ twitter:image            https://devforward.com/og-image.png

┌─ Structured Data (JSON-LD) ─────────────────────────────────────
│ Block #1 (@graph)        Organization, WebSite, WebPage, ProfessionalService

╔══════════════════════════════════════════════════════════════════╗
║  SEO Audit Results                                             ║
╚══════════════════════════════════════════════════════════════════╝

┌─ Title ─────────────────────────────────────────────────────────
  ✓ Title exists                   Found: "Dev Forward - Founder-Led Product Studio"
  ✓ Title length                   40 chars (30-60 recommended)

┌─ Description ───────────────────────────────────────────────────
  ✓ Meta description exists        Found: "Dev Forward is a founder-led AI product
                                   studio. We build and operate our own software
                                   products, and partner selectively with founders as
                                   a strategic CTO and build partner."
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
  ✓ og:description exists          Dev Forward is a founder-led product studio building
                                   AI-enabled software and partnering with ambitious
                                   startups to design, build, and scale technology
                                   businesses.
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
  ⚠ Heading hierarchy              Skipped heading level: H2 → H4 (found "Navigate")

┌─ Structured Data ───────────────────────────────────────────────
  ✓ JSON-LD exists                 Found 1 block(s)
  ✓ JSON-LD #1 @graph              4 item(s): Organization, WebSite, WebPage, ProfessionalService

┌─ Images ────────────────────────────────────────────────────────
  ✓ Images found                   2 image(s)
  ✓ Alt text                       All images have alt text
  ⚠ Image dimensions               2 image(s) missing width/height (causes layout shift)

┌─ Content ───────────────────────────────────────────────────────
  ℹ Word count                     612 words
  ⚠ Text-to-HTML ratio             9.7% (low). Heavy markup relative to visible content.

┌─ Links ─────────────────────────────────────────────────────────
  ✓ Links found                    21 total (16 internal, 5 external)

────────────────────────────────────────────────────────────────────
  Summary: 22 passed  6 warnings  1 errors  2 info
────────────────────────────────────────────────────────────────────
```

## Link Checker

```sh
krawl links https://devforward.com
```

### Example output

```
╔══════════════════════════════════════════════════════════════════╗
║  Link Check: https://devforward.com                            ║
╚══════════════════════════════════════════════════════════════════╝

  Found 9 links (4 internal, 5 external). Checking...

┌─ Internal Links (4) ──────────────────────────────────────────
  ✓ 200   https://devforward.com/                                      Dev Forward
  ✓ 200   https://devforward.com/work-with-us                          Work With Us
  ✓ 200   https://devforward.com                                       Explore the Studio
  ✓ 200   https://devforward.com/cdn-cgi/l/email-protection            [email protected]

┌─ External Links (5) ──────────────────────────────────────────
  ✓ 200   https://costops.dev                                          ■ Visit CostOps
  ✓ 200   https://electrac.app                                         ■ Visit Electrac
  ✓ 200   https://slopdrop.net                                         ■ Visit SlopDrop
  ✓ 200   https://rentbutter.com                                       ■ Visit RentButter
  ✓ 200   https://joincamino.com                                       ■ Visit Camino

────────────────────────────────────────────────────────────────────
  Summary: 9 ok  0 redirected  0 broken  (9 total)
────────────────────────────────────────────────────────────────────
```

## Sitemap Validator

```sh
krawl sitemap https://www.apple.com/sitemap.xml
```

### Example output

```
╔══════════════════════════════════════════════════════════════════╗
║  Sitemap: https://www.apple.com/sitemap.xml                    ║
╚══════════════════════════════════════════════════════════════════╝

┌─ Summary ───────────────────────────────────────────────────────
│ Type                     URL Set
│ Total URLs               838
│ File Size                70.1 KB
│ Has lastmod              0/838 URLs

┌─ URLs (838) ────────────────────────────────────────────────────
│   https://www.apple.com/
│   https://www.apple.com/accessibility/
│   https://www.apple.com/airpods-4/
│   https://www.apple.com/apple-intelligence/
│   https://www.apple.com/iphone/
│   ... and 833 more URLs

╔══════════════════════════════════════════════════════════════════╗
║  Validation Results                                            ║
╚══════════════════════════════════════════════════════════════════╝

  ⚠ No URLs have lastmod set (recommended for crawl prioritization)
  ℹ robots.txt declares a sitemap but not this specific URL

────────────────────────────────────────────────────────────────────
  Summary: 0 errors  1 warnings  1 info
────────────────────────────────────────────────────────────────────
```

Validates against the sitemaps.org protocol:
- URL limits (50,000 per file) and file size (50MB)
- Absolute URLs, domain/protocol matching, duplicates
- `lastmod` format (W3C datetime), future dates
- `changefreq` and `priority` values
- Sitemap index support (nested sitemaps)
- robots.txt sitemap declaration

## JSON Output

All commands support `-j` / `--json` for machine-readable output:

```sh
krawl -j https://devforward.com
krawl -j https://devforward.com | jq '.audit.summary'
krawl links -j https://devforward.com | jq '.summary'
krawl sitemap -j https://example.com/sitemap.xml
```

## Schema Detail

Use `-s` / `--schema` to see a full breakdown of JSON-LD structured data with nested entities, plus schema.org property validation and rich result eligibility checks:

```sh
krawl -s https://devforward.com
```

## Flags

### `krawl <url>`

| Flag | Description |
|------|-------------|
| `-j, --json` | Output as JSON |
| `-s, --schema` | Show only detailed JSON-LD structured data |
| `--no-audit` | Skip SEO audit, show metadata only |
| `--no-meta` | Skip metadata, show audit only |
| `-t, --timeout` | HTTP timeout (default 30s) |
| `-u, --user-agent` | Custom User-Agent string |
| `--config` | Path to config file |

### `krawl links <url>`

| Flag | Description |
|------|-------------|
| `-j, --json` | Output as JSON |
| `-c, --concurrency` | Number of concurrent link checks (default 10) |

### `krawl sitemap <url>`

| Flag | Description |
|------|-------------|
| `-j, --json` | Output as JSON |

## Config

krawl looks for `.krawl.yaml` in your home directory or current directory. Settings can also be passed via `KRAWL_*` environment variables.

## What it checks

**HTTP & Performance** — status code, redirects, DNS/TCP/TLS/TTFB timing, content size, notable security headers

**Meta Tags** — title tag (length validation), meta description (length validation), canonical URL (absolute, HTTPS), robots directives, charset, viewport, lang attribute

**Open Graph** — og:title, og:type, og:image, og:url, og:description, og:image:alt (accessibility)

**Twitter Cards** — twitter:card (valid type), twitter:title, twitter:description, twitter:image (HTTPS), twitter:image:alt

**Structured Data** — JSON-LD detection, @context and @type validation, @graph traversal, schema.org property validation (required/recommended per type), rich result eligibility checks (FAQ, HowTo, Article, Product, Breadcrumb)

**Headings** — H1 existence and count (flags multiple H1s), heading hierarchy validation (flags skipped levels like H2 → H4)

**Technical** — favicon, viewport zoom restrictions (accessibility), charset encoding

**Images** — alt text audit (missing vs decorative empty alt), width/height dimensions (layout shift prevention)

**Content** — word count, thin content detection, text-to-HTML ratio

**Redirects** — redirect chain length (flags >2 hops), mixed HTTP/HTTPS redirect detection

**Links** — internal/external link breakdown, nofollow audit, excessive link count warning, generic anchor text detection, external domain concentration, empty anchor text, internal and external link checking with concurrent HEAD requests

**Sitemaps** — XML sitemap validation, sitemap index support, URL/lastmod/changefreq/priority checks, robots.txt declaration

## License

MIT
