package display

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/kmazanec/seocheck/internal/fetcher"
	"github.com/kmazanec/seocheck/internal/parser"
	"github.com/kmazanec/seocheck/internal/rules"
)

var (
	bold      = color.New(color.Bold)
	green     = color.New(color.FgGreen)
	yellow    = color.New(color.FgYellow)
	red       = color.New(color.FgRed)
	cyan      = color.New(color.FgCyan)
	dim       = color.New(color.Faint)
	boldWhite = color.New(color.Bold, color.FgWhite)
)

func PrintHTTPInfo(r *fetcher.Result) {
	fmt.Println()
	bold.Println("╔══════════════════════════════════════════════════════════════════╗")
	bold.Printf("║  SEO Check: %-53s║\n", truncateDisplay(r.URL, 53))
	bold.Println("╚══════════════════════════════════════════════════════════════════╝")

	fmt.Println()
	sectionHeader("HTTP Response")
	printRow("Status Code", formatStatusCode(r.StatusCode))
	printRow("Final URL", r.FinalURL)
	printRow("Content-Type", r.ContentType)
	printRow("Content-Length", formatBytes(r.ContentLength))

	if len(r.Redirects) > 0 {
		fmt.Println()
		sectionHeader("Redirect Chain")
		for i, rd := range r.Redirects {
			printRow(fmt.Sprintf("  %d. [%d]", i+1, rd.StatusCode), rd.URL)
		}
		printRow(fmt.Sprintf("  %d. [%d]", len(r.Redirects)+1, r.StatusCode), r.FinalURL)
	}

	fmt.Println()
	sectionHeader("Timing")
	printRow("DNS Lookup", r.DNSTime.String())
	printRow("TCP Connect", r.ConnectTime.String())
	printRow("TLS Handshake", r.TLSTime.String())
	printRow("Time to First Byte", r.TTFB.String())
	printRow("Total Time", r.TotalTime.String())

	printRelevantHeaders(r)
}

func PrintSEOData(data *parser.SEOData) {
	fmt.Println()
	sectionHeader("Page Metadata")

	printRow("Title", valueOrMissing(data.Title))
	printRow("Description", valueOrMissing(data.MetaDescription))
	printRow("Canonical", valueOrMissing(data.Canonical))
	printRow("Robots", valueOrEmpty(data.Robots, "(not set - defaults to index, follow)"))
	printRow("Charset", valueOrMissing(data.Charset))
	printRow("Viewport", valueOrMissing(data.Viewport))
	printRow("Language", valueOrMissing(data.Lang))

	fmt.Println()
	sectionHeader("Open Graph")
	printRow("og:title", valueOrMissing(data.OGTitle))
	printRow("og:type", valueOrMissing(data.OGType))
	printRow("og:image", valueOrMissing(data.OGImage))
	printRow("og:url", valueOrMissing(data.OGURL))
	printRow("og:description", valueOrMissing(data.OGDescription))
	printRow("og:site_name", valueOrEmpty(data.OGSiteName, "(not set)"))
	printRow("og:locale", valueOrEmpty(data.OGLocale, "(not set)"))
	if data.OGImageAlt != "" {
		printRow("og:image:alt", data.OGImageAlt)
	}

	fmt.Println()
	sectionHeader("Twitter Card")
	printRow("twitter:card", valueOrMissing(data.TwitterCard))
	printRow("twitter:title", valueOrEmpty(data.TwitterTitle, "(falls back to og:title)"))
	printRow("twitter:description", valueOrEmpty(data.TwitterDescription, "(falls back to og:description)"))
	printRow("twitter:image", valueOrEmpty(data.TwitterImage, "(falls back to og:image)"))
	if data.TwitterSite != "" {
		printRow("twitter:site", data.TwitterSite)
	}

	if len(data.Hreflang) > 0 {
		fmt.Println()
		sectionHeader("Hreflang Tags")
		for _, h := range data.Hreflang {
			printRow(h.Lang, h.Href)
		}
	}

	if len(data.JSONLDBlocks) > 0 {
		fmt.Println()
		sectionHeader("Structured Data (JSON-LD)")
		for i, block := range data.JSONLDBlocks {
			typ, _ := block["@type"].(string)
			if typ == "" {
				typ = "(unknown type)"
			}
			printRow(fmt.Sprintf("Block #%d", i+1), typ)
		}
	}

	if len(data.H1) > 0 {
		fmt.Println()
		sectionHeader("Headings")
		for i, h := range data.H1 {
			printRow(fmt.Sprintf("H1 #%d", i+1), h)
		}
	}
}

func PrintRules(results []rules.Result) {
	fmt.Println()
	bold.Println("╔══════════════════════════════════════════════════════════════════╗")
	bold.Println("║  SEO Audit Results                                             ║")
	bold.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	var passes, warnings, errors, infos int
	currentCat := ""

	for _, r := range results {
		if r.Category != currentCat {
			if currentCat != "" {
				fmt.Println()
			}
			sectionHeader(r.Category)
			currentCat = r.Category
		}

		var icon string
		switch r.Severity {
		case rules.SeverityPass:
			icon = green.Sprint("  ✓ ")
			passes++
		case rules.SeverityInfo:
			icon = cyan.Sprint("  ℹ ")
			infos++
		case rules.SeverityWarning:
			icon = yellow.Sprint("  ⚠ ")
			warnings++
		case rules.SeverityError:
			icon = red.Sprint("  ✗ ")
			errors++
		}

		fmt.Printf("%s%-30s %s\n", icon, r.Rule, dim.Sprint(r.Message))
	}

	fmt.Println()
	bold.Println("────────────────────────────────────────────────────────────────────")
	fmt.Printf("  Summary: %s passed  %s warnings  %s errors  %s info\n",
		green.Sprintf("%d", passes),
		yellow.Sprintf("%d", warnings),
		red.Sprintf("%d", errors),
		cyan.Sprintf("%d", infos),
	)
	bold.Println("────────────────────────────────────────────────────────────────────")
	fmt.Println()
}

func printRelevantHeaders(r *fetcher.Result) {
	relevant := []string{
		"X-Robots-Tag",
		"Cache-Control",
		"Strict-Transport-Security",
		"Content-Security-Policy",
		"X-Frame-Options",
		"X-Content-Type-Options",
		"Link",
		"Server",
	}

	var found []string
	for _, h := range relevant {
		if v := r.Headers.Get(h); v != "" {
			found = append(found, h)
		}
	}

	if len(found) > 0 {
		fmt.Println()
		sectionHeader("Notable Headers")
		for _, h := range found {
			printRow(h, r.Headers.Get(h))
		}
	}
}

func sectionHeader(title string) {
	boldWhite.Printf("┌─ %s ", title)
	boldWhite.Println(strings.Repeat("─", max(0, 62-len(title))))
}

func printRow(label, value string) {
	cyan.Printf("│ %-24s", label)
	fmt.Printf(" %s\n", value)
}

func formatStatusCode(code int) string {
	s := fmt.Sprintf("%d", code)
	if code >= 200 && code < 300 {
		return green.Sprint(s)
	} else if code >= 300 && code < 400 {
		return yellow.Sprint(s)
	}
	return red.Sprint(s)
}

func formatBytes(n int64) string {
	if n < 0 {
		return "unknown"
	}
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}

func valueOrMissing(v string) string {
	if v == "" {
		return red.Sprint("(missing)")
	}
	return v
}

func valueOrEmpty(v, fallback string) string {
	if v == "" {
		return dim.Sprint(fallback)
	}
	return v
}

func truncateDisplay(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-3] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
