# Roadmap

Planned features and enhancements for krawl.

## Completed

### Audit Rules — Batch 1 (Body Parsing & Core Rules)
- Heading hierarchy validation (H1-H6 skip detection)
- Image audit (missing alt, missing dimensions)
- Content metrics (word count, thin content, text-to-HTML ratio)
- Redirect chain analysis (>2 hops, mixed HTTP/HTTPS)
- Link metrics (internal/external breakdown, nofollow count, excessive links)

### Audit Rules — Batch 2 (Schema & Link Quality)
- Schema.org validation: required/recommended properties per @type (Article, Product, LocalBusiness, Organization, Person, FAQPage, HowTo, BreadcrumbList, WebSite, WebPage, Event, Recipe, VideoObject)
- Rich result eligibility: FAQ, HowTo, Article, Product, Breadcrumb checks against Google's requirements
- Link quality: generic anchor text detection, external domain concentration, nofollow distribution, empty anchor text

## New Commands

### `krawl crawl <url>`
Spider a site up to N pages. Aggregate audit results across all pages. Detect:
- Orphan pages (no internal links pointing to them)
- Duplicate titles/descriptions across pages
- Inconsistent canonical URLs
- Internal link graph with crawl depth
- Site-wide issue summary with per-page breakdown

Flags: `--max-pages`, `--concurrency`, `--depth`, `--same-host`

### `krawl images <url>`
Deep image audit for a single page:
- Every `<img>` with src, alt, dimensions, file size (via HEAD request), format
- Flag oversized images (e.g., >200KB), missing lazy loading on below-fold images
- Detect next-gen format usage (WebP, AVIF) vs legacy (JPEG, PNG, GIF)
- Check for responsive images (`srcset`, `<picture>`)

### `krawl robots <url>`
Parse and validate a site's robots.txt:
- Syntax validation, unknown directives
- Conflicting allow/disallow rules
- Crawl-delay values
- Sitemap declarations and whether declared sitemaps are reachable
- Check if important pages are accidentally blocked

### `krawl diff <url>`
Compare current audit results against a saved baseline:
- Save baseline: `krawl -j https://example.com > baseline.json`
- Compare: `krawl diff --baseline baseline.json https://example.com`
- Show new issues, resolved issues, and changed values
- Useful for pre/post deploy verification

### `krawl bulk <file>`
Run audits across multiple URLs:
- Read URLs from file (one per line) or stdin
- Concurrent execution with configurable parallelism
- Aggregated summary report: total issues by severity across all pages
- Per-URL pass/fail status
- JSON output with full results for each URL

## DX / Workflow

### CI Integration
- Exit codes by severity: exit 0 on pass/info, exit 1 on errors, configurable for warnings
- `--fail-on` flag: `--fail-on=warning` or `--fail-on=error`
- Machine-readable summary line for CI log parsing

### Config-Driven Rules
- Custom thresholds in `.krawl.yaml` (title length, description length, word count minimum, max links)
- Disable specific rules or categories
- Custom severity overrides (e.g., treat missing og:url as warning instead of error)

### Watch Mode
- `krawl watch <url>` — re-run audit on interval, show diff from previous run
- Useful during local development to see SEO impact of changes in real time
