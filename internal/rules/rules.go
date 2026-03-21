package rules

import (
	"fmt"
	"strings"

	"github.com/devforward/krawl/internal/fetcher"
	"github.com/devforward/krawl/internal/parser"
)

type Severity int

const (
	SeverityPass Severity = iota
	SeverityInfo
	SeverityWarning
	SeverityError
)

func (s Severity) String() string {
	switch s {
	case SeverityPass:
		return "PASS"
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARN"
	case SeverityError:
		return "FAIL"
	default:
		return "UNKNOWN"
	}
}

type Result struct {
	Category string
	Rule     string
	Severity Severity
	Message  string
}

func Evaluate(data *parser.SEOData, fetchResult *fetcher.Result) []Result {
	var results []Result

	results = append(results, checkTitle(data)...)
	results = append(results, checkDescription(data)...)
	results = append(results, checkCanonical(data)...)
	results = append(results, checkRobots(data)...)
	results = append(results, checkOpenGraph(data)...)
	results = append(results, checkTwitter(data)...)
	results = append(results, checkTechnical(data)...)
	results = append(results, checkHeadings(data)...)
	results = append(results, checkStructuredData(data)...)
	results = append(results, checkImages(data)...)
	results = append(results, checkContent(data)...)
	results = append(results, checkRedirects(fetchResult)...)
	results = append(results, checkLinkMetrics(data)...)
	results = append(results, checkSchemaValidation(data)...)
	results = append(results, checkRichResults(data)...)
	results = append(results, checkLinkQuality(data)...)

	return results
}

func checkTitle(data *parser.SEOData) []Result {
	var results []Result
	cat := "Title"

	if data.Title == "" {
		results = append(results, Result{cat, "Title exists", SeverityError, "Missing <title> tag"})
		return results
	}

	results = append(results, Result{cat, "Title exists", SeverityPass, fmt.Sprintf("Found: %q", truncate(data.Title, 80))})

	n := len(data.Title)
	if n < 30 {
		results = append(results, Result{cat, "Title length", SeverityWarning, fmt.Sprintf("Too short (%d chars). Aim for 30-60.", n)})
	} else if n > 60 {
		results = append(results, Result{cat, "Title length", SeverityWarning, fmt.Sprintf("Too long (%d chars). May be truncated in SERPs. Aim for 30-60.", n)})
	} else {
		results = append(results, Result{cat, "Title length", SeverityPass, fmt.Sprintf("%d chars (30-60 recommended)", n)})
	}

	return results
}

func checkDescription(data *parser.SEOData) []Result {
	var results []Result
	cat := "Description"

	if data.MetaDescription == "" {
		results = append(results, Result{cat, "Meta description exists", SeverityWarning, "Missing meta description"})
		return results
	}

	results = append(results, Result{cat, "Meta description exists", SeverityPass, fmt.Sprintf("Found: %q", truncate(data.MetaDescription, 180))})

	n := len(data.MetaDescription)
	if n < 70 {
		results = append(results, Result{cat, "Description length", SeverityWarning, fmt.Sprintf("Too short (%d chars). Aim for 70-160.", n)})
	} else if n > 160 {
		results = append(results, Result{cat, "Description length", SeverityWarning, fmt.Sprintf("Too long (%d chars). May be truncated. Aim for 70-160.", n)})
	} else {
		results = append(results, Result{cat, "Description length", SeverityPass, fmt.Sprintf("%d chars (70-160 recommended)", n)})
	}

	return results
}

func checkCanonical(data *parser.SEOData) []Result {
	var results []Result
	cat := "Canonical"

	if data.Canonical == "" {
		results = append(results, Result{cat, "Canonical URL exists", SeverityError, "Missing <link rel=\"canonical\"> tag"})
		return results
	}

	results = append(results, Result{cat, "Canonical URL exists", SeverityPass, data.Canonical})

	if !strings.HasPrefix(data.Canonical, "http://") && !strings.HasPrefix(data.Canonical, "https://") {
		results = append(results, Result{cat, "Canonical is absolute URL", SeverityError, "Canonical should be an absolute URL"})
	} else {
		results = append(results, Result{cat, "Canonical is absolute URL", SeverityPass, "Uses absolute URL"})
		if strings.HasPrefix(data.Canonical, "http://") {
			results = append(results, Result{cat, "Canonical uses HTTPS", SeverityWarning, "Canonical uses HTTP instead of HTTPS"})
		} else {
			results = append(results, Result{cat, "Canonical uses HTTPS", SeverityPass, "Uses HTTPS"})
		}
	}

	return results
}

func checkRobots(data *parser.SEOData) []Result {
	var results []Result
	cat := "Robots"

	if data.Robots == "" {
		results = append(results, Result{cat, "Robots meta tag", SeverityInfo, "No robots meta tag (defaults to index, follow)"})
		return results
	}

	results = append(results, Result{cat, "Robots meta tag", SeverityPass, data.Robots})

	lower := strings.ToLower(data.Robots)
	if strings.Contains(lower, "noindex") {
		results = append(results, Result{cat, "Indexability", SeverityWarning, "Page is set to noindex"})
		if data.Canonical != "" {
			results = append(results, Result{cat, "noindex + canonical", SeverityWarning, "Page has both noindex and canonical (contradictory signals)"})
		}
	}

	return results
}

func checkOpenGraph(data *parser.SEOData) []Result {
	var results []Result
	cat := "Open Graph"

	check := func(name, value string, required bool, maxLen int) {
		if value == "" {
			sev := SeverityWarning
			if required {
				sev = SeverityError
			}
			results = append(results, Result{cat, name + " exists", sev, "Missing " + name})
		} else {
			results = append(results, Result{cat, name + " exists", SeverityPass, truncate(value, maxLen)})
		}
	}

	check("og:title", data.OGTitle, true, 80)
	check("og:type", data.OGType, true, 80)
	check("og:image", data.OGImage, true, 120)
	check("og:url", data.OGURL, true, 120)
	check("og:description", data.OGDescription, false, 220)

	if data.OGTitle != "" && len(data.OGTitle) > 60 {
		results = append(results, Result{cat, "og:title length", SeverityWarning, fmt.Sprintf("Too long (%d chars). Aim for under 60.", len(data.OGTitle))})
	}
	if data.OGDescription != "" && len(data.OGDescription) > 200 {
		results = append(results, Result{cat, "og:description length", SeverityWarning, fmt.Sprintf("Too long (%d chars). Aim for under 200.", len(data.OGDescription))})
	}
	if data.OGImage != "" {
		if !strings.HasPrefix(data.OGImage, "http") {
			results = append(results, Result{cat, "og:image absolute URL", SeverityError, "og:image should be an absolute URL"})
		}
		if data.OGImageAlt == "" {
			results = append(results, Result{cat, "og:image:alt exists", SeverityWarning, "Missing og:image:alt (accessibility)"})
		}
	}
	if data.OGURL != "" && !strings.HasPrefix(data.OGURL, "http") {
		results = append(results, Result{cat, "og:url absolute URL", SeverityError, "og:url should be an absolute URL"})
	}

	return results
}

func checkTwitter(data *parser.SEOData) []Result {
	var results []Result
	cat := "Twitter Card"

	if data.TwitterCard == "" {
		results = append(results, Result{cat, "twitter:card exists", SeverityWarning, "Missing twitter:card tag"})
	} else {
		validCards := map[string]bool{"summary": true, "summary_large_image": true, "app": true, "player": true}
		if !validCards[data.TwitterCard] {
			results = append(results, Result{cat, "twitter:card value", SeverityError, fmt.Sprintf("Invalid value %q. Must be summary, summary_large_image, app, or player.", data.TwitterCard)})
		} else {
			results = append(results, Result{cat, "twitter:card exists", SeverityPass, data.TwitterCard})
		}
	}

	if data.TwitterTitle == "" && data.OGTitle == "" {
		results = append(results, Result{cat, "twitter:title exists", SeverityWarning, "Missing twitter:title (no og:title fallback either)"})
	} else if data.TwitterTitle != "" {
		results = append(results, Result{cat, "twitter:title exists", SeverityPass, truncate(data.TwitterTitle, 70)})
		if len(data.TwitterTitle) > 70 {
			results = append(results, Result{cat, "twitter:title length", SeverityWarning, fmt.Sprintf("Too long (%d chars). May be truncated at 70.", len(data.TwitterTitle))})
		}
	}

	if data.TwitterImage != "" && !strings.HasPrefix(data.TwitterImage, "https://") {
		results = append(results, Result{cat, "twitter:image HTTPS", SeverityWarning, "twitter:image should use HTTPS"})
	}

	if data.TwitterImage != "" && data.TwitterImageAlt == "" {
		results = append(results, Result{cat, "twitter:image:alt exists", SeverityWarning, "Missing twitter:image:alt (accessibility)"})
	}

	return results
}

func checkTechnical(data *parser.SEOData) []Result {
	var results []Result
	cat := "Technical"

	if data.Charset == "" {
		results = append(results, Result{cat, "Charset declared", SeverityWarning, "Missing charset declaration"})
	} else if strings.EqualFold(data.Charset, "UTF-8") {
		results = append(results, Result{cat, "Charset declared", SeverityPass, data.Charset})
	} else {
		results = append(results, Result{cat, "Charset declared", SeverityWarning, fmt.Sprintf("Charset is %q, UTF-8 recommended", data.Charset)})
	}

	if data.Viewport == "" {
		results = append(results, Result{cat, "Viewport meta tag", SeverityError, "Missing viewport meta tag (critical for mobile SEO)"})
	} else {
		if !strings.Contains(data.Viewport, "width=device-width") {
			results = append(results, Result{cat, "Viewport meta tag", SeverityWarning, "Viewport missing width=device-width"})
		} else {
			results = append(results, Result{cat, "Viewport meta tag", SeverityPass, data.Viewport})
		}
		if strings.Contains(data.Viewport, "user-scalable=no") || strings.Contains(data.Viewport, "maximum-scale=1") {
			results = append(results, Result{cat, "Viewport accessibility", SeverityWarning, "Viewport restricts zoom (user-scalable=no or maximum-scale=1)"})
		}
	}

	if data.Lang == "" {
		results = append(results, Result{cat, "HTML lang attribute", SeverityWarning, "Missing lang attribute on <html>"})
	} else {
		results = append(results, Result{cat, "HTML lang attribute", SeverityPass, data.Lang})
	}

	if !data.HasFavicon {
		results = append(results, Result{cat, "Favicon", SeverityInfo, "No favicon link tag found"})
	} else {
		results = append(results, Result{cat, "Favicon", SeverityPass, "Favicon declared"})
	}

	return results
}

func checkHeadings(data *parser.SEOData) []Result {
	var results []Result
	cat := "Headings"

	if len(data.H1) == 0 {
		results = append(results, Result{cat, "H1 tag exists", SeverityWarning, "No <h1> tag found on page"})
	} else if len(data.H1) == 1 {
		results = append(results, Result{cat, "H1 tag exists", SeverityPass, truncate(data.H1[0], 60)})
	} else {
		results = append(results, Result{cat, "H1 tag count", SeverityWarning, fmt.Sprintf("Multiple H1 tags found (%d). Use exactly one.", len(data.H1))})
	}

	// Check heading hierarchy for skipped levels
	if len(data.Headings) > 0 {
		skipped := false
		prevLevel := 0
		for _, h := range data.Headings {
			if prevLevel > 0 && h.Level > prevLevel+1 {
				results = append(results, Result{cat, "Heading hierarchy", SeverityWarning,
					fmt.Sprintf("Skipped heading level: H%d → H%d (found %q)", prevLevel, h.Level, truncate(h.Text, 40))})
				skipped = true
			}
			prevLevel = h.Level
		}
		if !skipped {
			results = append(results, Result{cat, "Heading hierarchy", SeverityPass, "No skipped heading levels"})
		}
	}

	return results
}

func checkStructuredData(data *parser.SEOData) []Result {
	var results []Result
	cat := "Structured Data"

	if len(data.JSONLDBlocks) == 0 {
		results = append(results, Result{cat, "JSON-LD exists", SeverityInfo, "No JSON-LD structured data found"})
		return results
	}

	results = append(results, Result{cat, "JSON-LD exists", SeverityPass, fmt.Sprintf("Found %d block(s)", len(data.JSONLDBlocks))})

	for i, block := range data.JSONLDBlocks {
		ctx, _ := block["@context"].(string)
		if ctx == "" {
			results = append(results, Result{cat, fmt.Sprintf("JSON-LD #%d @context", i+1), SeverityWarning, "Missing @context"})
		}

		typ, _ := block["@type"].(string)
		graph, hasGraph := block["@graph"].([]interface{})

		if typ != "" {
			results = append(results, Result{cat, fmt.Sprintf("JSON-LD #%d @type", i+1), SeverityPass, typ})
		} else if hasGraph {
			var types []string
			for _, item := range graph {
				if obj, ok := item.(map[string]interface{}); ok {
					if t, ok := obj["@type"].(string); ok {
						types = append(types, t)
					}
				}
			}
			if len(types) > 0 {
				results = append(results, Result{cat, fmt.Sprintf("JSON-LD #%d @graph", i+1), SeverityPass, fmt.Sprintf("%d item(s): %s", len(types), strings.Join(types, ", "))})
			} else {
				results = append(results, Result{cat, fmt.Sprintf("JSON-LD #%d @graph", i+1), SeverityWarning, "Graph contains no typed items"})
			}
		} else {
			results = append(results, Result{cat, fmt.Sprintf("JSON-LD #%d @type", i+1), SeverityWarning, "Missing @type"})
		}
	}

	return results
}

func checkImages(data *parser.SEOData) []Result {
	var results []Result
	cat := "Images"

	if len(data.Images) == 0 {
		results = append(results, Result{cat, "Images found", SeverityInfo, "No <img> tags found on page"})
		return results
	}

	results = append(results, Result{cat, "Images found", SeverityPass, fmt.Sprintf("%d image(s)", len(data.Images))})

	missingAlt := 0
	emptyAlt := 0
	missingDimensions := 0
	for _, img := range data.Images {
		if !img.HasAlt {
			missingAlt++
		} else if img.Alt == "" {
			emptyAlt++
		}
		if img.Width == "" && img.Height == "" {
			missingDimensions++
		}
	}

	if missingAlt > 0 {
		results = append(results, Result{cat, "Alt text", SeverityError,
			fmt.Sprintf("%d image(s) missing alt attribute (accessibility & SEO)", missingAlt)})
	} else if emptyAlt > 0 {
		results = append(results, Result{cat, "Alt text", SeverityPass,
			fmt.Sprintf("All images have alt attributes (%d decorative with empty alt)", emptyAlt)})
	} else {
		results = append(results, Result{cat, "Alt text", SeverityPass, "All images have alt text"})
	}

	if missingDimensions > 0 {
		results = append(results, Result{cat, "Image dimensions", SeverityWarning,
			fmt.Sprintf("%d image(s) missing width/height (causes layout shift)", missingDimensions)})
	} else {
		results = append(results, Result{cat, "Image dimensions", SeverityPass, "All images have width and height"})
	}

	return results
}

func checkContent(data *parser.SEOData) []Result {
	var results []Result
	cat := "Content"

	results = append(results, Result{cat, "Word count", SeverityInfo, fmt.Sprintf("%d words", data.WordCount)})

	if data.WordCount < 50 {
		results = append(results, Result{cat, "Thin content", SeverityWarning, "Very low word count. Pages with thin content may rank poorly."})
	}

	if data.ContentRatio > 0 {
		pct := data.ContentRatio * 100
		if pct < 10 {
			results = append(results, Result{cat, "Text-to-HTML ratio", SeverityWarning,
				fmt.Sprintf("%.1f%% (low). Heavy markup relative to visible content.", pct)})
		} else {
			results = append(results, Result{cat, "Text-to-HTML ratio", SeverityPass,
				fmt.Sprintf("%.1f%%", pct)})
		}
	}

	return results
}

func checkRedirects(r *fetcher.Result) []Result {
	var results []Result
	cat := "Redirects"

	if len(r.Redirects) == 0 {
		return results
	}

	if len(r.Redirects) <= 2 {
		results = append(results, Result{cat, "Redirect chain", SeverityPass,
			fmt.Sprintf("%d redirect(s)", len(r.Redirects))})
	} else {
		results = append(results, Result{cat, "Redirect chain", SeverityWarning,
			fmt.Sprintf("%d redirects in chain (aim for ≤2). Longer chains waste crawl budget.", len(r.Redirects))})
	}

	// Check for mixed HTTP/HTTPS in the chain
	for _, rd := range r.Redirects {
		if strings.HasPrefix(rd.URL, "http://") && strings.HasPrefix(r.FinalURL, "https://") {
			results = append(results, Result{cat, "Mixed protocol redirect", SeverityWarning,
				"Redirect chain includes HTTP → HTTPS hop. Set up direct HTTPS redirect."})
			break
		}
	}

	return results
}

func checkLinkMetrics(data *parser.SEOData) []Result {
	var results []Result
	cat := "Links"

	if data.TotalLinks == 0 {
		results = append(results, Result{cat, "Links found", SeverityWarning, "No links found on page"})
		return results
	}

	results = append(results, Result{cat, "Links found", SeverityPass,
		fmt.Sprintf("%d total (%d internal, %d external)", data.TotalLinks, data.InternalLinks, data.ExternalLinks)})

	if data.TotalLinks > 200 {
		results = append(results, Result{cat, "Link count", SeverityWarning,
			fmt.Sprintf("%d links on page. Excessive links may dilute PageRank and hurt crawlability.", data.TotalLinks)})
	}

	if data.NofollowLinks > 0 {
		results = append(results, Result{cat, "Nofollow links", SeverityInfo,
			fmt.Sprintf("%d link(s) marked nofollow", data.NofollowLinks)})
	}

	if data.InternalLinks == 0 && data.TotalLinks > 0 {
		results = append(results, Result{cat, "Internal links", SeverityWarning,
			"No internal links found. Internal linking helps search engines discover and rank pages."})
	}

	return results
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
