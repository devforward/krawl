# Roadmap

Planned features and enhancements for krawl.

## Audit Rule Enhancements (Batch 2)

Extend the existing rules engine with deeper analysis:

### Schema.org Validation
- Validate JSON-LD against schema.org required/recommended properties per `@type`
- Article: `headline`, `datePublished`, `author`, `image`
- Product: `name`, `offers` (with `price`, `priceCurrency`)
- LocalBusiness: `name`, `address`, `telephone`
- FAQPage: `mainEntity` with `Question`/`acceptedAnswer` pairs
- BreadcrumbList: `itemListElement` with `position`, `name`, `item`

### Rich Result Eligibility
- Check if structured data meets Google's specific requirements for rich results
- FAQ rich result: valid FAQPage schema with properly nested Q&A
- HowTo rich result: `step` elements with `text` or `name`+`itemListElement`
- Article rich result: required fields plus `dateModified`, `publisher` with logo
- Product rich result: `aggregateRating` or `review` for star ratings in SERPs

### Link Quality
- Outbound link audit: count links by domain, flag concentration to single external domain
- Nofollow distribution: warn if all external links are nofollow (looks unnatural)
- Anchor text analysis: flag generic anchors ("click here", "read more") that waste link context

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
