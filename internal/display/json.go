package display

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/devforward/krawl/internal/fetcher"
	"github.com/devforward/krawl/internal/parser"
	"github.com/devforward/krawl/internal/rules"
)

type JSONOutput struct {
	HTTP     JSONHTTPInfo     `json:"http"`
	SEO      JSONSEOData      `json:"seo"`
	Audit    *JSONAudit       `json:"audit,omitempty"`
}

type JSONHTTPInfo struct {
	URL           string            `json:"url"`
	FinalURL      string            `json:"final_url"`
	StatusCode    int               `json:"status_code"`
	ContentType   string            `json:"content_type"`
	ContentLength int64             `json:"content_length_bytes"`
	Redirects     []JSONRedirect    `json:"redirects,omitempty"`
	Timing        JSONTiming        `json:"timing_ms"`
	Headers       map[string]string `json:"notable_headers,omitempty"`
}

type JSONRedirect struct {
	URL        string `json:"url"`
	StatusCode int    `json:"status_code"`
}

type JSONTiming struct {
	DNS     float64 `json:"dns"`
	Connect float64 `json:"connect"`
	TLS     float64 `json:"tls"`
	TTFB    float64 `json:"ttfb"`
	Total   float64 `json:"total"`
}

type JSONSEOData struct {
	Title           string `json:"title"`
	MetaDescription string `json:"meta_description"`
	Canonical       string `json:"canonical"`
	Robots          string `json:"robots"`
	Charset         string `json:"charset"`
	Viewport        string `json:"viewport"`
	Lang            string `json:"lang"`

	OpenGraph   JSONOpenGraph   `json:"open_graph"`
	TwitterCard JSONTwitterCard `json:"twitter_card"`

	Hreflang       []JSONHreflang           `json:"hreflang,omitempty"`
	JSONLD         []map[string]interface{} `json:"json_ld,omitempty"`
	HasFavicon     bool                     `json:"has_favicon"`
	H1             []string                 `json:"h1,omitempty"`
	Images         *JSONImageStats          `json:"images,omitempty"`
	Content        *JSONContentStats        `json:"content,omitempty"`
	Links          *JSONLinkStats           `json:"links,omitempty"`
}

type JSONImageStats struct {
	Total             int `json:"total"`
	MissingAlt        int `json:"missing_alt"`
	MissingDimensions int `json:"missing_dimensions"`
}

type JSONContentStats struct {
	WordCount     int     `json:"word_count"`
	ContentBytes  int     `json:"content_bytes"`
	HTMLBytes     int     `json:"html_bytes"`
	TextToHTML    float64 `json:"text_to_html_ratio"`
}

type JSONLinkStats struct {
	Total    int `json:"total"`
	Internal int `json:"internal"`
	External int `json:"external"`
	Nofollow int `json:"nofollow"`
}

type JSONOpenGraph struct {
	Title       string `json:"title"`
	Type        string `json:"type"`
	Image       string `json:"image"`
	ImageAlt    string `json:"image_alt"`
	URL         string `json:"url"`
	Description string `json:"description"`
	SiteName    string `json:"site_name"`
	Locale      string `json:"locale"`
}

type JSONTwitterCard struct {
	Card        string `json:"card"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Image       string `json:"image"`
	ImageAlt    string `json:"image_alt"`
	Site        string `json:"site"`
	Creator     string `json:"creator"`
}

type JSONHreflang struct {
	Lang string `json:"lang"`
	Href string `json:"href"`
}

type JSONAudit struct {
	Results []JSONAuditResult `json:"results"`
	Summary JSONAuditSummary  `json:"summary"`
}

type JSONAuditResult struct {
	Category string `json:"category"`
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

type JSONAuditSummary struct {
	Pass int `json:"pass"`
	Warn int `json:"warn"`
	Fail int `json:"fail"`
	Info int `json:"info"`
}

func PrintJSON(fetchResult *fetcher.Result, seoData *parser.SEOData, auditResults []rules.Result) error {
	output := JSONOutput{
		HTTP: buildHTTPInfo(fetchResult),
		SEO:  buildSEOData(seoData),
	}

	if auditResults != nil {
		output.Audit = buildAudit(auditResults)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return fmt.Errorf("encoding JSON: %w", err)
	}
	return nil
}

func buildHTTPInfo(r *fetcher.Result) JSONHTTPInfo {
	info := JSONHTTPInfo{
		URL:           r.URL,
		FinalURL:      r.FinalURL,
		StatusCode:    r.StatusCode,
		ContentType:   r.ContentType,
		ContentLength: r.ContentLength,
		Timing: JSONTiming{
			DNS:     r.DNSTime.Seconds() * 1000,
			Connect: r.ConnectTime.Seconds() * 1000,
			TLS:     r.TLSTime.Seconds() * 1000,
			TTFB:    r.TTFB.Seconds() * 1000,
			Total:   r.TotalTime.Seconds() * 1000,
		},
	}

	for _, rd := range r.Redirects {
		info.Redirects = append(info.Redirects, JSONRedirect{
			URL:        rd.URL,
			StatusCode: rd.StatusCode,
		})
	}

	relevant := []string{
		"X-Robots-Tag", "Cache-Control", "Strict-Transport-Security",
		"Content-Security-Policy", "X-Frame-Options", "X-Content-Type-Options",
		"Link", "Server",
	}
	headers := make(map[string]string)
	for _, h := range relevant {
		if v := r.Headers.Get(h); v != "" {
			headers[h] = v
		}
	}
	if len(headers) > 0 {
		info.Headers = headers
	}

	return info
}

func buildSEOData(d *parser.SEOData) JSONSEOData {
	seo := JSONSEOData{
		Title:           d.Title,
		MetaDescription: d.MetaDescription,
		Canonical:       d.Canonical,
		Robots:          d.Robots,
		Charset:         d.Charset,
		Viewport:        d.Viewport,
		Lang:            d.Lang,
		OpenGraph: JSONOpenGraph{
			Title:       d.OGTitle,
			Type:        d.OGType,
			Image:       d.OGImage,
			ImageAlt:    d.OGImageAlt,
			URL:         d.OGURL,
			Description: d.OGDescription,
			SiteName:    d.OGSiteName,
			Locale:      d.OGLocale,
		},
		TwitterCard: JSONTwitterCard{
			Card:        d.TwitterCard,
			Title:       d.TwitterTitle,
			Description: d.TwitterDescription,
			Image:       d.TwitterImage,
			ImageAlt:    d.TwitterImageAlt,
			Site:        d.TwitterSite,
			Creator:     d.TwitterCreator,
		},
		HasFavicon: d.HasFavicon,
		H1:         d.H1,
		JSONLD:     d.JSONLDBlocks,
	}

	for _, h := range d.Hreflang {
		seo.Hreflang = append(seo.Hreflang, JSONHreflang{Lang: h.Lang, Href: h.Href})
	}

	if len(d.Images) > 0 {
		missingAlt := 0
		missingDim := 0
		for _, img := range d.Images {
			if !img.HasAlt {
				missingAlt++
			}
			if img.Width == "" && img.Height == "" {
				missingDim++
			}
		}
		seo.Images = &JSONImageStats{
			Total:             len(d.Images),
			MissingAlt:        missingAlt,
			MissingDimensions: missingDim,
		}
	}

	if d.WordCount > 0 || d.RawHTMLLength > 0 {
		seo.Content = &JSONContentStats{
			WordCount:    d.WordCount,
			ContentBytes: d.ContentLength,
			HTMLBytes:    d.RawHTMLLength,
			TextToHTML:   d.ContentRatio,
		}
	}

	if d.TotalLinks > 0 {
		seo.Links = &JSONLinkStats{
			Total:    d.TotalLinks,
			Internal: d.InternalLinks,
			External: d.ExternalLinks,
			Nofollow: d.NofollowLinks,
		}
	}

	return seo
}

func buildAudit(results []rules.Result) *JSONAudit {
	audit := &JSONAudit{}
	for _, r := range results {
		audit.Results = append(audit.Results, JSONAuditResult{
			Category: r.Category,
			Rule:     r.Rule,
			Severity: r.Severity.String(),
			Message:  r.Message,
		})
		switch r.Severity {
		case rules.SeverityPass:
			audit.Summary.Pass++
		case rules.SeverityWarning:
			audit.Summary.Warn++
		case rules.SeverityError:
			audit.Summary.Fail++
		case rules.SeverityInfo:
			audit.Summary.Info++
		}
	}
	return audit
}
