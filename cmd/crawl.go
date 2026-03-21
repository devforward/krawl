package cmd

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/devforward/krawl/internal/fetcher"
	"github.com/devforward/krawl/internal/parser"
	"github.com/devforward/krawl/internal/rules"
)

var crawlCmd = &cobra.Command{
	Use:   "crawl [url]",
	Short: "Spider a site and audit multiple pages",
	Long:  `Crawls a website starting from the given URL, following internal links up to a configurable depth and page limit. Runs SEO audit rules on each page and reports site-wide issues.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runCrawl,
}

func init() {
	rootCmd.AddCommand(crawlCmd)
	crawlCmd.Flags().IntP("max-pages", "n", 20, "Maximum number of pages to crawl")
	crawlCmd.Flags().IntP("depth", "d", 3, "Maximum link depth from start URL")
	crawlCmd.Flags().IntP("concurrency", "c", 5, "Number of concurrent fetches")
	crawlCmd.Flags().BoolP("json", "j", false, "Output results as JSON")
}

// crawlPage holds the result of crawling a single page.
type crawlPage struct {
	URL          string
	Depth        int
	FetchResult  *fetcher.Result
	SEOData      *parser.SEOData
	AuditResults []rules.Result
	Error        string
	InternalURLs []string // discovered internal links
}

// crawlSummary holds site-wide aggregated findings.
type crawlSummary struct {
	TotalPages     int
	TotalErrors    int
	TotalWarnings  int
	TotalPasses    int
	TotalInfos     int
	FailedPages    int
	DuplicateTitles    map[string][]string // title -> list of URLs
	DuplicateDescs     map[string][]string // description -> list of URLs
	MissingTitles      []string
	MissingDescs       []string
	MissingCanonicals  []string
	OrphanPages        []string // pages with no inbound internal links
}

// JSON output types
type crawlJSONOutput struct {
	StartURL string            `json:"start_url"`
	Pages    []crawlJSONPage   `json:"pages"`
	Summary  crawlJSONSummary  `json:"summary"`
	Issues   []crawlJSONIssue  `json:"issues"`
}

type crawlJSONPage struct {
	URL        string `json:"url"`
	Depth      int    `json:"depth"`
	StatusCode int    `json:"status_code,omitempty"`
	Title      string `json:"title,omitempty"`
	Error      string `json:"error,omitempty"`
	Errors     int    `json:"errors"`
	Warnings   int    `json:"warnings"`
}

type crawlJSONSummary struct {
	TotalPages  int `json:"total_pages"`
	FailedPages int `json:"failed_pages"`
	TotalPass   int `json:"total_pass"`
	TotalWarn   int `json:"total_warnings"`
	TotalFail   int `json:"total_errors"`
	TotalInfo   int `json:"total_info"`
}

type crawlJSONIssue struct {
	Type    string   `json:"type"`
	Message string   `json:"message"`
	URLs    []string `json:"urls,omitempty"`
}

func runCrawl(cmd *cobra.Command, args []string) error {
	startURL := args[0]

	timeout := viper.GetDuration("timeout")
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	userAgent := viper.GetString("user-agent")
	if userAgent == "" {
		userAgent = "krawl/1.0"
	}
	maxPages, _ := cmd.Flags().GetInt("max-pages")
	maxDepth, _ := cmd.Flags().GetInt("depth")
	concurrency, _ := cmd.Flags().GetInt("concurrency")
	jsonOutput, _ := cmd.Flags().GetBool("json")

	// Normalize start URL
	parsedStart, err := url.Parse(startURL)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}
	baseHost := strings.ToLower(parsedStart.Host)

	// BFS crawl
	type queueItem struct {
		url   string
		depth int
	}

	visited := make(map[string]bool)
	var pages []crawlPage
	queue := []queueItem{{url: startURL, depth: 0}}
	visited[normalizeURL(startURL)] = true

	// Track inbound links for orphan detection
	inboundLinks := make(map[string]int) // url -> count of pages linking to it

	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex

	if !jsonOutput {
		bold := color.New(color.Bold)
		fmt.Println()
		bold.Println("╔══════════════════════════════════════════════════════════════════╗")
		bold.Printf("║  Crawl: %-53s║\n", truncateStr(startURL, 53))
		bold.Println("╚══════════════════════════════════════════════════════════════════╝")
		fmt.Println()
		fmt.Printf("  Max pages: %d, Max depth: %d, Concurrency: %d\n", maxPages, maxDepth, concurrency)
		fmt.Println()
	}

	for len(queue) > 0 && len(pages) < maxPages {
		// Determine batch size
		batchSize := concurrency
		if batchSize > len(queue) {
			batchSize = len(queue)
		}
		if batchSize > maxPages-len(pages) {
			batchSize = maxPages - len(pages)
		}

		batch := queue[:batchSize]
		queue = queue[batchSize:]

		var wg sync.WaitGroup
		batchResults := make([]crawlPage, batchSize)

		for i, item := range batch {
			wg.Add(1)
			go func(idx int, qi queueItem) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				page := crawlSinglePage(qi.url, qi.depth, timeout, userAgent)
				batchResults[idx] = page
			}(i, item)
		}
		wg.Wait()

		for _, page := range batchResults {
			pages = append(pages, page)

			if !jsonOutput {
				printCrawlProgress(page, len(pages), maxPages, baseHost)
			}

			// Queue discovered internal links
			if page.Error == "" && page.SEOData != nil {
				for _, link := range page.InternalURLs {
					normalized := normalizeURL(link)
					linkParsed, err := url.Parse(link)
					if err != nil {
						continue
					}
					if strings.ToLower(linkParsed.Host) != baseHost {
						continue
					}
					if !visited[normalized] && page.Depth+1 <= maxDepth {
						visited[normalized] = true
						queue = append(queue, queueItem{url: link, depth: page.Depth + 1})
					}

					// Track inbound
					mu.Lock()
					inboundLinks[normalized]++
					mu.Unlock()
				}
			}
		}
	}

	// Build summary
	summary := buildCrawlSummary(pages, inboundLinks, visited)

	if jsonOutput {
		return printCrawlJSON(startURL, pages, summary)
	}

	printCrawlSummary(pages, summary, baseHost)
	return nil
}

func crawlSinglePage(pageURL string, depth int, timeout time.Duration, userAgent string) crawlPage {
	page := crawlPage{URL: pageURL, Depth: depth}

	result, err := fetcher.Fetch(pageURL, timeout, userAgent)
	if err != nil {
		page.Error = err.Error()
		return page
	}
	page.FetchResult = result

	if result.StatusCode >= 400 {
		page.Error = fmt.Sprintf("HTTP %d", result.StatusCode)
		return page
	}

	// Only parse HTML pages
	ct := strings.ToLower(result.ContentType)
	if !strings.Contains(ct, "text/html") && !strings.Contains(ct, "application/xhtml") {
		page.Error = fmt.Sprintf("Not HTML: %s", result.ContentType)
		return page
	}

	seoData, err := parser.ParseWithURL(result.Body, result.FinalURL)
	if err != nil {
		page.Error = fmt.Sprintf("Parse error: %s", err)
		return page
	}
	page.SEOData = seoData
	page.AuditResults = rules.Evaluate(seoData, result)

	// Extract internal links for further crawling
	links, err := parser.ParseLinks(result.Body, result.FinalURL)
	if err == nil {
		for _, link := range links {
			if parser.IsInternal(link.Href, result.FinalURL) {
				page.InternalURLs = append(page.InternalURLs, link.Href)
			}
		}
	}

	return page
}

func buildCrawlSummary(pages []crawlPage, inboundLinks map[string]int, visited map[string]bool) crawlSummary {
	summary := crawlSummary{
		DuplicateTitles: make(map[string][]string),
		DuplicateDescs:  make(map[string][]string),
	}

	titleMap := make(map[string][]string)
	descMap := make(map[string][]string)

	for _, page := range pages {
		summary.TotalPages++

		if page.Error != "" {
			summary.FailedPages++
			continue
		}

		for _, r := range page.AuditResults {
			switch r.Severity {
			case rules.SeverityPass:
				summary.TotalPasses++
			case rules.SeverityWarning:
				summary.TotalWarnings++
			case rules.SeverityError:
				summary.TotalErrors++
			case rules.SeverityInfo:
				summary.TotalInfos++
			}
		}

		if page.SEOData != nil {
			if page.SEOData.Title == "" {
				summary.MissingTitles = append(summary.MissingTitles, page.URL)
			} else {
				titleMap[page.SEOData.Title] = append(titleMap[page.SEOData.Title], page.URL)
			}

			if page.SEOData.MetaDescription == "" {
				summary.MissingDescs = append(summary.MissingDescs, page.URL)
			} else {
				descMap[page.SEOData.MetaDescription] = append(descMap[page.SEOData.MetaDescription], page.URL)
			}

			if page.SEOData.Canonical == "" {
				summary.MissingCanonicals = append(summary.MissingCanonicals, page.URL)
			}
		}
	}

	for title, urls := range titleMap {
		if len(urls) > 1 {
			summary.DuplicateTitles[title] = urls
		}
	}
	for desc, urls := range descMap {
		if len(urls) > 1 {
			summary.DuplicateDescs[desc] = urls
		}
	}

	// Find orphan pages: pages we crawled that have no inbound links from other crawled pages
	// (excluding the start page which naturally has no inbound from the crawl)
	if len(pages) > 1 {
		startNorm := normalizeURL(pages[0].URL)
		for _, page := range pages {
			if page.Error != "" {
				continue
			}
			norm := normalizeURL(page.URL)
			if norm == startNorm {
				continue
			}
			if inboundLinks[norm] == 0 {
				summary.OrphanPages = append(summary.OrphanPages, page.URL)
			}
		}
	}

	return summary
}

// pathOf strips the scheme+host from a URL, returning just the path (or "/" for root).
func pathOf(rawURL, baseHost string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	if strings.ToLower(parsed.Host) == baseHost {
		p := parsed.Path
		if p == "" {
			p = "/"
		}
		if parsed.RawQuery != "" {
			p += "?" + parsed.RawQuery
		}
		return p
	}
	return rawURL
}

func printCrawlProgress(page crawlPage, current, total int, baseHost string) {
	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)
	yellow := color.New(color.FgYellow)
	dim := color.New(color.Faint)

	prefix := fmt.Sprintf("  [%d/%d] ", current, total)
	path := pathOf(page.URL, baseHost)

	if page.Error != "" {
		fmt.Printf("%s%s %s %s\n", prefix, red.Sprint("✗"), path, dim.Sprint(page.Error))
		return
	}

	errors := 0
	warnings := 0
	for _, r := range page.AuditResults {
		if r.Severity == rules.SeverityError {
			errors++
		} else if r.Severity == rules.SeverityWarning {
			warnings++
		}
	}

	var icon string
	if errors > 0 {
		icon = red.Sprint("●")
	} else if warnings > 0 {
		icon = yellow.Sprint("●")
	} else {
		icon = green.Sprint("✓")
	}

	statusParts := []string{}
	if errors > 0 {
		statusParts = append(statusParts, red.Sprintf("%de", errors))
	}
	if warnings > 0 {
		statusParts = append(statusParts, yellow.Sprintf("%dw", warnings))
	}

	if len(statusParts) > 0 {
		fmt.Printf("%s%s %s %s\n", prefix, icon, path, strings.Join(statusParts, " "))
	} else {
		fmt.Printf("%s%s %s\n", prefix, icon, path)
	}
}

func printCrawlSummary(pages []crawlPage, summary crawlSummary, baseHost string) {
	bold := color.New(color.Bold)
	boldWhite := color.New(color.Bold, color.FgWhite)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)
	cyan := color.New(color.FgCyan)
	dim := color.New(color.Faint)

	sectionHeader := func(title string) {
		fmt.Println()
		boldWhite.Printf("┌─ %s ", title)
		boldWhite.Println(strings.Repeat("─", maxVal(0, 62-len(title))))
	}

	pathList := func(urls []string) {
		for _, u := range urls {
			fmt.Printf("│     %s\n", pathOf(u, baseHost))
		}
	}

	// Site-wide issues
	hasIssues := len(summary.DuplicateTitles) > 0 || len(summary.DuplicateDescs) > 0 ||
		len(summary.MissingTitles) > 0 || len(summary.MissingDescs) > 0 ||
		len(summary.MissingCanonicals) > 0 || len(summary.OrphanPages) > 0

	if hasIssues {
		fmt.Println()
		bold.Println("╔══════════════════════════════════════════════════════════════════╗")
		bold.Println("║  Site-Wide Issues                                              ║")
		bold.Println("╚══════════════════════════════════════════════════════════════════╝")

		if len(summary.DuplicateTitles) > 0 {
			sectionHeader(fmt.Sprintf("Duplicate Titles (%d)", len(summary.DuplicateTitles)))
			for title, urls := range summary.DuplicateTitles {
				red.Print("│  ✗ ")
				fmt.Printf("%q\n", truncateStr(title, 70))
				pathList(urls)
			}
		}

		if len(summary.DuplicateDescs) > 0 {
			sectionHeader(fmt.Sprintf("Duplicate Descriptions (%d)", len(summary.DuplicateDescs)))
			for desc, urls := range summary.DuplicateDescs {
				red.Print("│  ✗ ")
				fmt.Printf("%q\n", truncateStr(desc, 70))
				pathList(urls)
			}
		}

		if len(summary.MissingTitles) > 0 {
			sectionHeader(fmt.Sprintf("Missing Titles (%d)", len(summary.MissingTitles)))
			pathList(summary.MissingTitles)
		}

		if len(summary.MissingDescs) > 0 {
			sectionHeader(fmt.Sprintf("Missing Descriptions (%d)", len(summary.MissingDescs)))
			pathList(summary.MissingDescs)
		}

		if len(summary.MissingCanonicals) > 0 {
			sectionHeader(fmt.Sprintf("Missing Canonicals (%d)", len(summary.MissingCanonicals)))
			pathList(summary.MissingCanonicals)
		}

		if len(summary.OrphanPages) > 0 {
			sectionHeader(fmt.Sprintf("Orphan Pages (%d)", len(summary.OrphanPages)))
			pathList(summary.OrphanPages)
		}
	}

	// Per-page breakdown: show pages with errors or warnings, grouped with details
	type pageIssue struct {
		path     string
		errors   []string
		warnings []string
	}
	var issuePages []pageIssue
	for _, page := range pages {
		if page.Error != "" {
			issuePages = append(issuePages, pageIssue{
				path:   pathOf(page.URL, baseHost),
				errors: []string{fmt.Sprintf("Fetch failed: %s", page.Error)},
			})
			continue
		}
		var errs, warns []string
		for _, r := range page.AuditResults {
			msg := fmt.Sprintf("%s: %s", r.Rule, r.Message)
			if r.Severity == rules.SeverityError {
				errs = append(errs, msg)
			} else if r.Severity == rules.SeverityWarning {
				warns = append(warns, msg)
			}
		}
		if len(errs) > 0 || len(warns) > 0 {
			issuePages = append(issuePages, pageIssue{
				path:     pathOf(page.URL, baseHost),
				errors:   errs,
				warnings: warns,
			})
		}
	}

	if len(issuePages) > 0 {
		sectionHeader(fmt.Sprintf("Issues by Page (%d pages)", len(issuePages)))
		for _, ip := range issuePages {
			bold.Printf("│\n│  %s\n", ip.path)
			for _, e := range ip.errors {
				fmt.Printf("│    %s %s\n", red.Sprint("✗"), e)
			}
			for _, w := range ip.warnings {
				fmt.Printf("│    %s %s\n", yellow.Sprint("⚠"), w)
			}
		}
	}

	// Pages with no issues
	cleanPages := 0
	for _, page := range pages {
		if page.Error != "" {
			continue
		}
		hasIssue := false
		for _, r := range page.AuditResults {
			if r.Severity == rules.SeverityError || r.Severity == rules.SeverityWarning {
				hasIssue = true
				break
			}
		}
		if !hasIssue {
			cleanPages++
		}
	}
	if cleanPages > 0 {
		fmt.Printf("│\n│  %s %s\n", green.Sprint("✓"), dim.Sprintf("%d page(s) with no errors or warnings", cleanPages))
	}

	// Final summary
	fmt.Println()
	bold.Println("────────────────────────────────────────────────────────────────────")
	fmt.Printf("  Pages crawled: %d", summary.TotalPages)
	if summary.FailedPages > 0 {
		fmt.Printf(" (%s failed)", red.Sprintf("%d", summary.FailedPages))
	}
	fmt.Println()
	fmt.Printf("  Audit totals: %s passed  %s warnings  %s errors  %s info\n",
		green.Sprintf("%d", summary.TotalPasses),
		yellow.Sprintf("%d", summary.TotalWarnings),
		red.Sprintf("%d", summary.TotalErrors),
		cyan.Sprintf("%d", summary.TotalInfos),
	)

	siteIssues := len(summary.DuplicateTitles) + len(summary.DuplicateDescs) +
		len(summary.MissingTitles) + len(summary.MissingDescs) +
		len(summary.MissingCanonicals) + len(summary.OrphanPages)
	if siteIssues > 0 {
		fmt.Printf("  Site-wide issues: %s\n", yellow.Sprintf("%d", siteIssues))
	}
	bold.Println("────────────────────────────────────────────────────────────────────")
	fmt.Println()
}

func printCrawlJSON(startURL string, pages []crawlPage, summary crawlSummary) error {
	output := crawlJSONOutput{
		StartURL: startURL,
		Summary: crawlJSONSummary{
			TotalPages:  summary.TotalPages,
			FailedPages: summary.FailedPages,
			TotalPass:   summary.TotalPasses,
			TotalWarn:   summary.TotalWarnings,
			TotalFail:   summary.TotalErrors,
			TotalInfo:   summary.TotalInfos,
		},
	}

	for _, page := range pages {
		jp := crawlJSONPage{
			URL:   page.URL,
			Depth: page.Depth,
			Error: page.Error,
		}
		if page.FetchResult != nil {
			jp.StatusCode = page.FetchResult.StatusCode
		}
		if page.SEOData != nil {
			jp.Title = page.SEOData.Title
		}
		for _, r := range page.AuditResults {
			if r.Severity == rules.SeverityError {
				jp.Errors++
			} else if r.Severity == rules.SeverityWarning {
				jp.Warnings++
			}
		}
		output.Pages = append(output.Pages, jp)
	}

	// Site-wide issues
	for title, urls := range summary.DuplicateTitles {
		output.Issues = append(output.Issues, crawlJSONIssue{
			Type:    "duplicate_title",
			Message: fmt.Sprintf("Duplicate title: %q", truncateStr(title, 80)),
			URLs:    urls,
		})
	}
	for desc, urls := range summary.DuplicateDescs {
		output.Issues = append(output.Issues, crawlJSONIssue{
			Type:    "duplicate_description",
			Message: fmt.Sprintf("Duplicate description: %q", truncateStr(desc, 80)),
			URLs:    urls,
		})
	}
	for _, u := range summary.MissingTitles {
		output.Issues = append(output.Issues, crawlJSONIssue{
			Type:    "missing_title",
			Message: "Missing title tag",
			URLs:    []string{u},
		})
	}
	for _, u := range summary.MissingDescs {
		output.Issues = append(output.Issues, crawlJSONIssue{
			Type:    "missing_description",
			Message: "Missing meta description",
			URLs:    []string{u},
		})
	}
	for _, u := range summary.MissingCanonicals {
		output.Issues = append(output.Issues, crawlJSONIssue{
			Type:    "missing_canonical",
			Message: "Missing canonical URL",
			URLs:    []string{u},
		})
	}
	for _, u := range summary.OrphanPages {
		output.Issues = append(output.Issues, crawlJSONIssue{
			Type:    "orphan_page",
			Message: "No inbound internal links from other crawled pages",
			URLs:    []string{u},
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func normalizeURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	// Normalize: lowercase host/scheme, strip fragment, strip trailing slash
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	parsed.Host = strings.ToLower(parsed.Host)
	parsed.Fragment = ""
	p := parsed.Path
	if p == "" || p == "/" {
		parsed.Path = ""
	} else if strings.HasSuffix(p, "/") {
		parsed.Path = strings.TrimRight(p, "/")
	}
	return parsed.String()
}
