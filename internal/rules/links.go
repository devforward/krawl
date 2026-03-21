package rules

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/devforward/krawl/internal/parser"
)

var genericAnchors = map[string]bool{
	"click here":  true,
	"here":        true,
	"read more":   true,
	"learn more":  true,
	"more":        true,
	"link":        true,
	"this":        true,
	"this link":   true,
	"go":          true,
	"click":       true,
	"continue":    true,
}

func checkLinkQuality(data *parser.SEOData) []Result {
	var results []Result
	cat := "Link Quality"

	if len(data.BodyLinks) == 0 {
		return results
	}

	// Count generic anchor texts
	genericCount := 0
	for _, link := range data.BodyLinks {
		text := strings.TrimSpace(strings.ToLower(link.Text))
		if genericAnchors[text] {
			genericCount++
		}
	}
	if genericCount > 0 {
		results = append(results, Result{cat, "Anchor text quality", SeverityWarning,
			fmt.Sprintf("%d link(s) use generic anchor text (\"click here\", \"read more\", etc.). Use descriptive text.", genericCount)})
	} else {
		results = append(results, Result{cat, "Anchor text quality", SeverityPass,
			"No generic anchor text found"})
	}

	// Check external link domain concentration
	externalDomains := make(map[string]int)
	externalCount := 0
	for _, link := range data.BodyLinks {
		if link.IsInternal {
			continue
		}
		externalCount++
		parsed, err := url.Parse(link.Href)
		if err != nil {
			continue
		}
		externalDomains[strings.ToLower(parsed.Host)]++
	}

	if externalCount > 3 {
		for domain, count := range externalDomains {
			pct := float64(count) / float64(externalCount) * 100
			if pct > 50 && count > 3 {
				results = append(results, Result{cat, "External link concentration", SeverityInfo,
					fmt.Sprintf("%.0f%% of external links (%d/%d) point to %s", pct, count, externalCount, domain)})
			}
		}
	}

	// Check nofollow distribution
	if externalCount > 0 && data.NofollowLinks > 0 {
		externalNofollow := 0
		for _, link := range data.BodyLinks {
			if !link.IsInternal && link.IsNofollow {
				externalNofollow++
			}
		}
		if externalNofollow == externalCount && externalCount > 2 {
			results = append(results, Result{cat, "Nofollow distribution", SeverityWarning,
				"All external links are nofollow. This can look unnatural to search engines."})
		}
	}

	// Check for empty anchor text (non-image links)
	emptyAnchors := 0
	for _, link := range data.BodyLinks {
		if strings.TrimSpace(link.Text) == "" {
			emptyAnchors++
		}
	}
	if emptyAnchors > 0 {
		results = append(results, Result{cat, "Empty anchor text", SeverityWarning,
			fmt.Sprintf("%d link(s) have no anchor text (bad for accessibility and SEO)", emptyAnchors)})
	}

	return results
}
