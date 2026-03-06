package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/devforward/krawl/internal/fetcher"
	"github.com/devforward/krawl/internal/parser"
)

var sitemapCmd = &cobra.Command{
	Use:   "sitemap [url]",
	Short: "Fetch and validate an XML sitemap",
	Long:  `Fetches an XML sitemap (or sitemap index), validates it against the sitemaps.org protocol, and reports any issues.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runSitemap,
}

func init() {
	rootCmd.AddCommand(sitemapCmd)
	sitemapCmd.Flags().BoolP("json", "j", false, "Output results as JSON")
	sitemapCmd.Flags().Bool("check-urls", false, "Check each URL for HTTP status (slow for large sitemaps)")
}

type sitemapJSONOutput struct {
	URL     string              `json:"url"`
	IsIndex bool                `json:"is_index"`
	Summary sitemapJSONSummary  `json:"summary"`
	Issues  []sitemapJSONIssue  `json:"issues"`
	URLs    []sitemapJSONURL    `json:"urls,omitempty"`
	Sitemaps []sitemapJSONEntry `json:"sitemaps,omitempty"`
}

type sitemapJSONSummary struct {
	TotalURLs     int    `json:"total_urls"`
	TotalSitemaps int    `json:"total_sitemaps,omitempty"`
	FileSize      string `json:"file_size"`
	FileSizeBytes int    `json:"file_size_bytes"`
	HasLastMod    int    `json:"has_lastmod"`
	HasPriority   int    `json:"has_priority"`
	HasChangeFreq int    `json:"has_changefreq"`
	Errors        int    `json:"errors"`
	Warnings      int    `json:"warnings"`
	Info          int    `json:"info"`
}

type sitemapJSONIssue struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type sitemapJSONURL struct {
	Loc        string `json:"loc"`
	LastMod    string `json:"lastmod,omitempty"`
	ChangeFreq string `json:"changefreq,omitempty"`
	Priority   string `json:"priority,omitempty"`
	Status     int    `json:"status,omitempty"`
}

type sitemapJSONEntry struct {
	Loc     string `json:"loc"`
	LastMod string `json:"lastmod,omitempty"`
}

func runSitemap(cmd *cobra.Command, args []string) error {
	sitemapURL := args[0]

	timeout := viper.GetDuration("timeout")
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	userAgent := viper.GetString("user-agent")
	if userAgent == "" {
		userAgent = "krawl/1.0"
	}

	result, err := fetcher.Fetch(sitemapURL, timeout, userAgent)
	if err != nil {
		return fmt.Errorf("failed to fetch sitemap: %w", err)
	}

	if result.StatusCode != 200 {
		return fmt.Errorf("sitemap returned HTTP %d", result.StatusCode)
	}

	report, err := parser.ParseSitemap(result.Body, result.FinalURL)
	if err != nil {
		return fmt.Errorf("failed to parse sitemap: %w", err)
	}

	// Check robots.txt for sitemap reference
	checkRobotsTxt(report, result.FinalURL, timeout, userAgent)

	jsonOutput, _ := cmd.Flags().GetBool("json")

	if jsonOutput {
		return printSitemapJSON(report)
	}

	printSitemapFormatted(report)
	return nil
}

func checkRobotsTxt(report *parser.SitemapReport, sitemapURL string, timeout time.Duration, userAgent string) {
	// Extract base URL for robots.txt
	parts := strings.SplitN(sitemapURL, "/", 4)
	if len(parts) < 3 {
		return
	}
	robotsURL := parts[0] + "//" + parts[2] + "/robots.txt"

	result, err := fetcher.Fetch(robotsURL, timeout, userAgent)
	if err != nil || result.StatusCode != 200 {
		report.Issues = append(report.Issues, parser.SitemapIssue{Severity: "warn", Message: "Could not fetch robots.txt to verify sitemap declaration"})
		return
	}

	body := strings.ToLower(string(result.Body))
	sitemapLower := strings.ToLower(sitemapURL)
	if !strings.Contains(body, "sitemap:") {
		report.Issues = append(report.Issues, parser.SitemapIssue{Severity: "warn", Message: "robots.txt does not declare any Sitemap"})
	} else if !strings.Contains(body, strings.ToLower(sitemapLower)) {
		report.Issues = append(report.Issues, parser.SitemapIssue{Severity: "info", Message: "robots.txt declares a sitemap but not this specific URL"})
	}
}

func printSitemapJSON(report *parser.SitemapReport) error {
	output := sitemapJSONOutput{
		URL:     report.URL,
		IsIndex: report.IsIndex,
		Summary: sitemapJSONSummary{
			TotalURLs:     report.TotalURLs,
			FileSize:      formatSitemapSize(report.RawSize),
			FileSizeBytes: report.RawSize,
			HasLastMod:    report.HasLastMod,
			HasPriority:   report.HasPriority,
			HasChangeFreq: report.HasChangeFreq,
		},
	}

	if report.IsIndex {
		output.Summary.TotalSitemaps = len(report.Sitemaps)
		for _, s := range report.Sitemaps {
			output.Sitemaps = append(output.Sitemaps, sitemapJSONEntry{Loc: s.Loc, LastMod: s.LastMod})
		}
	} else {
		for _, u := range report.URLs {
			output.URLs = append(output.URLs, sitemapJSONURL{
				Loc:        u.Loc,
				LastMod:    u.LastMod,
				ChangeFreq: u.ChangeFreq,
				Priority:   u.Priority,
			})
		}
	}

	for _, issue := range report.Issues {
		output.Issues = append(output.Issues, sitemapJSONIssue{Severity: issue.Severity, Message: issue.Message})
		switch issue.Severity {
		case "error":
			output.Summary.Errors++
		case "warn":
			output.Summary.Warnings++
		case "info":
			output.Summary.Info++
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func printSitemapFormatted(report *parser.SitemapReport) {
	bold := color.New(color.Bold)
	boldWhite := color.New(color.Bold, color.FgWhite)
	green := color.New(color.FgGreen)
	yellow := color.New(color.FgYellow)
	red := color.New(color.FgRed)
	cyan := color.New(color.FgCyan)
	dim := color.New(color.Faint)

	printRow := func(label, value string) {
		cyan.Printf("│ %-24s", label)
		fmt.Printf(" %s\n", value)
	}

	sectionHeader := func(title string) {
		boldWhite.Printf("┌─ %s ", title)
		boldWhite.Println(strings.Repeat("─", maxVal(0, 62-len(title))))
	}

	fmt.Println()
	bold.Println("╔══════════════════════════════════════════════════════════════════╗")
	bold.Printf("║  Sitemap: %-55s║\n", truncateStr(report.URL, 55))
	bold.Println("╚══════════════════════════════════════════════════════════════════╝")

	fmt.Println()
	sectionHeader("Summary")

	if report.IsIndex {
		printRow("Type", "Sitemap Index")
		printRow("Child Sitemaps", fmt.Sprintf("%d", len(report.Sitemaps)))
	} else {
		printRow("Type", "URL Set")
		printRow("Total URLs", fmt.Sprintf("%d", report.TotalURLs))
	}

	printRow("File Size", formatSitemapSize(report.RawSize))

	if !report.IsIndex && report.TotalURLs > 0 {
		printRow("Has lastmod", fmt.Sprintf("%d/%d URLs", report.HasLastMod, report.TotalURLs))
		if report.HasChangeFreq > 0 {
			printRow("Has changefreq", dim.Sprintf("%d URLs (ignored by Google)", report.HasChangeFreq))
		}
		if report.HasPriority > 0 {
			printRow("Has priority", dim.Sprintf("%d URLs (ignored by Google)", report.HasPriority))
		}
	}

	// Show entries
	if report.IsIndex {
		fmt.Println()
		sectionHeader("Child Sitemaps")
		for _, s := range report.Sitemaps {
			loc := s.Loc
			if len(loc) > 65 {
				loc = loc[:62] + "..."
			}
			if s.LastMod != "" {
				fmt.Printf("│   %-60s %s\n", loc, dim.Sprint(s.LastMod))
			} else {
				fmt.Printf("│   %s\n", loc)
			}
		}
	} else if report.TotalURLs > 0 {
		fmt.Println()
		sectionHeader(fmt.Sprintf("URLs (%d)", report.TotalURLs))

		hasAnyLastMod := report.HasLastMod > 0
		if hasAnyLastMod {
			dim.Printf("│   %-60s %s\n", "URL", "Last Modified")
			dim.Printf("│   %-60s %s\n", strings.Repeat("─", 58), "─────────────")
		}

		showCount := report.TotalURLs
		if showCount > 60 {
			showCount = 60
		}
		for _, u := range report.URLs[:showCount] {
			loc := u.Loc
			if len(loc) > 65 {
				loc = loc[:62] + "..."
			}
			if hasAnyLastMod {
				lastmod := u.LastMod
				if lastmod == "" {
					lastmod = dim.Sprint("(not set)")
				}
				fmt.Printf("│   %-60s %s\n", loc, dim.Sprint(lastmod))
			} else {
				fmt.Printf("│   %s\n", loc)
			}
		}
		if report.TotalURLs > 60 {
			dim.Printf("│   ... and %d more URLs\n", report.TotalURLs-60)
		}
	}

	// Issues
	var errors, warnings, infos int
	for _, issue := range report.Issues {
		switch issue.Severity {
		case "error":
			errors++
		case "warn":
			warnings++
		case "info":
			infos++
		}
	}

	if len(report.Issues) > 0 {
		fmt.Println()
		bold.Println("╔══════════════════════════════════════════════════════════════════╗")
		bold.Println("║  Validation Results                                            ║")
		bold.Println("╚══════════════════════════════════════════════════════════════════╝")
		fmt.Println()

		for _, issue := range report.Issues {
			var icon string
			switch issue.Severity {
			case "error":
				icon = red.Sprint("  ✗ ")
			case "warn":
				icon = yellow.Sprint("  ⚠ ")
			case "info":
				icon = cyan.Sprint("  ℹ ")
			}
			fmt.Printf("%s%s\n", icon, issue.Message)
		}
	}

	fmt.Println()
	bold.Println("────────────────────────────────────────────────────────────────────")
	if errors == 0 && warnings == 0 {
		fmt.Printf("  Summary: %s\n", green.Sprint("No issues found"))
	} else {
		fmt.Printf("  Summary: %s errors  %s warnings  %s info\n",
			red.Sprintf("%d", errors),
			yellow.Sprintf("%d", warnings),
			cyan.Sprintf("%d", infos),
		)
	}
	bold.Println("────────────────────────────────────────────────────────────────────")
	fmt.Println()
}

func formatSitemapSize(n int) string {
	if n < 1024 {
		return fmt.Sprintf("%d B", n)
	}
	if n < 1024*1024 {
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
}

func maxVal(a, b int) int {
	if a > b {
		return a
	}
	return b
}
