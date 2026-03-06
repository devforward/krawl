package parser

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Sitemap struct {
	IsIndex  bool
	URLs     []SitemapURL
	Sitemaps []SitemapEntry
	RawSize  int
}

type SitemapURL struct {
	Loc        string `xml:"loc"`
	LastMod    string `xml:"lastmod"`
	ChangeFreq string `xml:"changefreq"`
	Priority   string `xml:"priority"`
}

type SitemapEntry struct {
	Loc     string `xml:"loc"`
	LastMod string `xml:"lastmod"`
}

type SitemapIssue struct {
	Severity string // "error", "warn", "info"
	Message  string
}

type SitemapReport struct {
	URL        string
	IsIndex    bool
	URLs       []SitemapURL
	Sitemaps   []SitemapEntry
	Issues     []SitemapIssue
	TotalURLs  int
	RawSize    int
	HasLastMod int
	HasPriority int
	HasChangeFreq int
}

// XML structures for parsing
type xmlURLSet struct {
	XMLName xml.Name     `xml:"urlset"`
	URLs    []SitemapURL `xml:"url"`
}

type xmlSitemapIndex struct {
	XMLName  xml.Name       `xml:"sitemapindex"`
	Sitemaps []SitemapEntry `xml:"sitemap"`
}

func ParseSitemap(body []byte, sitemapURL string) (*SitemapReport, error) {
	report := &SitemapReport{
		URL:     sitemapURL,
		RawSize: len(body),
	}

	// Try sitemap index first
	var index xmlSitemapIndex
	if err := xml.Unmarshal(body, &index); err == nil && len(index.Sitemaps) > 0 {
		report.IsIndex = true
		report.Sitemaps = index.Sitemaps
		validateIndex(report, sitemapURL)
		return report, nil
	}

	// Try regular urlset
	var urlset xmlURLSet
	if err := xml.Unmarshal(body, &urlset); err != nil {
		return nil, fmt.Errorf("invalid XML: %w", err)
	}

	report.URLs = urlset.URLs
	report.TotalURLs = len(urlset.URLs)
	validateURLSet(report, sitemapURL)
	return report, nil
}

func validateIndex(report *SitemapReport, sitemapURL string) {
	if len(report.Sitemaps) > 50000 {
		report.Issues = append(report.Issues, SitemapIssue{"error", fmt.Sprintf("Sitemap index contains %d sitemaps (max 50,000)", len(report.Sitemaps))})
	}

	base, _ := url.Parse(sitemapURL)
	for _, s := range report.Sitemaps {
		if !strings.HasPrefix(s.Loc, "http") {
			report.Issues = append(report.Issues, SitemapIssue{"error", fmt.Sprintf("Non-absolute URL in index: %s", s.Loc)})
		} else if base != nil {
			parsed, err := url.Parse(s.Loc)
			if err == nil && !strings.EqualFold(parsed.Host, base.Host) {
				report.Issues = append(report.Issues, SitemapIssue{"warn", fmt.Sprintf("Cross-domain sitemap: %s", s.Loc)})
			}
		}
		if s.LastMod != "" && !isValidW3CDatetime(s.LastMod) {
			report.Issues = append(report.Issues, SitemapIssue{"warn", fmt.Sprintf("Invalid lastmod format: %s", s.LastMod)})
		}
	}
}

func validateURLSet(report *SitemapReport, sitemapURL string) {
	if report.RawSize > 52428800 {
		report.Issues = append(report.Issues, SitemapIssue{"error", fmt.Sprintf("Sitemap exceeds 50MB uncompressed (%d bytes)", report.RawSize)})
	}

	if len(report.URLs) > 50000 {
		report.Issues = append(report.Issues, SitemapIssue{"error", fmt.Sprintf("Sitemap contains %d URLs (max 50,000)", len(report.URLs))})
	}

	base, _ := url.Parse(sitemapURL)
	seen := make(map[string]bool)
	validChangeFreqs := map[string]bool{
		"always": true, "hourly": true, "daily": true, "weekly": true,
		"monthly": true, "yearly": true, "never": true,
	}

	for _, u := range report.URLs {
		// Count optional fields
		if u.LastMod != "" {
			report.HasLastMod++
		}
		if u.Priority != "" {
			report.HasPriority++
		}
		if u.ChangeFreq != "" {
			report.HasChangeFreq++
		}

		// Duplicate check
		if seen[u.Loc] {
			report.Issues = append(report.Issues, SitemapIssue{"warn", fmt.Sprintf("Duplicate URL: %s", truncateSitemapURL(u.Loc, 80))})
		}
		seen[u.Loc] = true

		// Absolute URL check
		if !strings.HasPrefix(u.Loc, "http://") && !strings.HasPrefix(u.Loc, "https://") {
			report.Issues = append(report.Issues, SitemapIssue{"error", fmt.Sprintf("Non-absolute URL: %s", u.Loc)})
			continue
		}

		// URL length
		if len(u.Loc) > 2048 {
			report.Issues = append(report.Issues, SitemapIssue{"warn", fmt.Sprintf("URL exceeds 2048 chars: %s", truncateSitemapURL(u.Loc, 80))})
		}

		// Domain match
		if base != nil {
			parsed, err := url.Parse(u.Loc)
			if err == nil && !strings.EqualFold(parsed.Host, base.Host) {
				report.Issues = append(report.Issues, SitemapIssue{"warn", fmt.Sprintf("Cross-domain URL: %s", truncateSitemapURL(u.Loc, 80))})
			}
			// Protocol match
			if err == nil && parsed.Scheme != base.Scheme {
				report.Issues = append(report.Issues, SitemapIssue{"warn", fmt.Sprintf("Protocol mismatch (%s vs %s): %s", base.Scheme, parsed.Scheme, truncateSitemapURL(u.Loc, 60))})
			}
		}

		// lastmod validation
		if u.LastMod != "" {
			if !isValidW3CDatetime(u.LastMod) {
				report.Issues = append(report.Issues, SitemapIssue{"warn", fmt.Sprintf("Invalid lastmod %q for %s", u.LastMod, truncateSitemapURL(u.Loc, 60))})
			} else if isFutureDate(u.LastMod) {
				report.Issues = append(report.Issues, SitemapIssue{"warn", fmt.Sprintf("Future lastmod %s for %s", u.LastMod, truncateSitemapURL(u.Loc, 60))})
			}
		}

		// changefreq validation
		if u.ChangeFreq != "" && !validChangeFreqs[strings.ToLower(u.ChangeFreq)] {
			report.Issues = append(report.Issues, SitemapIssue{"error", fmt.Sprintf("Invalid changefreq %q for %s", u.ChangeFreq, truncateSitemapURL(u.Loc, 60))})
		}

		// priority validation
		if u.Priority != "" {
			p, err := strconv.ParseFloat(u.Priority, 64)
			if err != nil || p < 0.0 || p > 1.0 {
				report.Issues = append(report.Issues, SitemapIssue{"error", fmt.Sprintf("Invalid priority %q for %s", u.Priority, truncateSitemapURL(u.Loc, 60))})
			}
		}
	}

	// Warnings for missing optional fields
	if report.HasLastMod == 0 && len(report.URLs) > 0 {
		report.Issues = append(report.Issues, SitemapIssue{"warn", "No URLs have lastmod set (recommended for crawl prioritization)"})
	} else if report.HasLastMod > 0 && report.HasLastMod < len(report.URLs) {
		report.Issues = append(report.Issues, SitemapIssue{"info", fmt.Sprintf("Only %d/%d URLs have lastmod set", report.HasLastMod, len(report.URLs))})
	}

	if report.HasChangeFreq > 0 {
		report.Issues = append(report.Issues, SitemapIssue{"info", "changefreq is present but ignored by Google"})
	}

	if report.HasPriority > 0 {
		report.Issues = append(report.Issues, SitemapIssue{"info", "priority is present but ignored by Google"})
	}
}

var w3cDatePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\d{4}$`),
	regexp.MustCompile(`^\d{4}-\d{2}$`),
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`),
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:\d{2})$`),
}

func isValidW3CDatetime(s string) bool {
	for _, p := range w3cDatePatterns {
		if p.MatchString(s) {
			return true
		}
	}
	return false
}

func isFutureDate(s string) bool {
	formats := []string{
		"2006",
		"2006-01",
		"2006-01-02",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04Z07:00",
		"2006-01-02T15:04:05.000Z07:00",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.After(time.Now())
		}
	}
	return false
}

func truncateSitemapURL(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
