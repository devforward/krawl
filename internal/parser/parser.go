package parser

import (
	"bytes"
	"encoding/json"
	"net/url"
	"strings"
	"unicode"

	"golang.org/x/net/html"
)

type SEOData struct {
	Title          string
	MetaDescription string
	Canonical      string
	Robots         string
	Charset        string
	Viewport       string
	Lang           string

	// Open Graph
	OGTitle       string
	OGType        string
	OGImage       string
	OGImageAlt    string
	OGImageWidth  string
	OGImageHeight string
	OGURL         string
	OGDescription string
	OGSiteName    string
	OGLocale      string

	// Twitter
	TwitterCard        string
	TwitterTitle       string
	TwitterDescription string
	TwitterImage       string
	TwitterImageAlt    string
	TwitterSite        string
	TwitterCreator     string

	// Other
	Hreflang      []HreflangEntry
	JSONLDBlocks  []map[string]interface{}
	HasFavicon    bool
	H1            []string
	H2            []string

	// Heading hierarchy (ordered as they appear in document)
	Headings []Heading

	// Images
	Images []ImageTag

	// Content metrics
	WordCount        int
	ContentLength    int // visible text bytes
	RawHTMLLength    int // total HTML bytes
	ContentRatio     float64 // ContentLength / RawHTMLLength

	// Link metrics (from body)
	InternalLinks    int
	ExternalLinks    int
	NofollowLinks    int
	TotalLinks       int

	// Raw collections
	MetaTags      []MetaTag
	LinkTags      []LinkTag

	// unexported
	pageURL string
}

type Heading struct {
	Level int    // 1-6
	Text  string
}

type ImageTag struct {
	Src     string
	Alt     string
	HasAlt  bool // distinguishes alt="" from no alt attribute
	Width   string
	Height  string
	Loading string
}

type HreflangEntry struct {
	Lang string
	Href string
}

type MetaTag struct {
	Name     string
	Property string
	Content  string
	Charset  string
	HttpEquiv string
}

type LinkTag struct {
	Rel      string
	Href     string
	Hreflang string
	Type     string
}

func Parse(body []byte) (*SEOData, error) {
	return ParseWithURL(body, "")
}

func ParseWithURL(body []byte, pageURL string) (*SEOData, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	data := &SEOData{
		RawHTMLLength: len(body),
	}
	data.pageURL = pageURL
	parseNode(doc, data)

	if data.RawHTMLLength > 0 {
		data.ContentRatio = float64(data.ContentLength) / float64(data.RawHTMLLength)
	}

	return data, nil
}

func parseNode(n *html.Node, data *SEOData) {
	switch n.Type {
	case html.ElementNode:
		switch n.Data {
		case "html":
			for _, a := range n.Attr {
				if a.Key == "lang" {
					data.Lang = a.Val
				}
			}
		case "title":
			if n.FirstChild != nil {
				data.Title = strings.TrimSpace(n.FirstChild.Data)
			}
		case "meta":
			parseMeta(n, data)
		case "link":
			parseLink(n, data)
		case "script":
			parseScript(n, data)
		case "h1":
			data.H1 = append(data.H1, getTextContent(n))
		case "h2":
			data.H2 = append(data.H2, getTextContent(n))
		case "body":
			parseBody(n, data)
			return
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parseNode(c, data)
	}
}

var headingLevels = map[string]int{
	"h1": 1, "h2": 2, "h3": 3, "h4": 4, "h5": 5, "h6": 6,
}

func parseBody(n *html.Node, data *SEOData) {
	if n.Type == html.TextNode {
		// Skip text inside script/style tags (handled by caller skipping those subtrees)
		text := strings.TrimSpace(n.Data)
		if text != "" {
			data.ContentLength += len(n.Data)
			data.WordCount += countWords(text)
		}
	}

	if n.Type == html.ElementNode {
		// Skip script and style content
		if n.Data == "script" || n.Data == "style" || n.Data == "noscript" {
			return
		}

		if level, ok := headingLevels[n.Data]; ok {
			text := getTextContent(n)
			data.Headings = append(data.Headings, Heading{Level: level, Text: text})
			switch n.Data {
			case "h1":
				data.H1 = append(data.H1, text)
			case "h2":
				data.H2 = append(data.H2, text)
			}
		}

		if n.Data == "img" {
			parseImage(n, data)
		}

		if n.Data == "a" {
			parseBodyLink(n, data)
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parseBody(c, data)
	}
}

func parseImage(n *html.Node, data *SEOData) {
	img := ImageTag{}
	for _, a := range n.Attr {
		switch a.Key {
		case "src":
			img.Src = a.Val
		case "alt":
			img.Alt = a.Val
			img.HasAlt = true
		case "width":
			img.Width = a.Val
		case "height":
			img.Height = a.Val
		case "loading":
			img.Loading = a.Val
		}
	}
	data.Images = append(data.Images, img)
}

func parseBodyLink(n *html.Node, data *SEOData) {
	var href, rel string
	for _, a := range n.Attr {
		switch a.Key {
		case "href":
			href = a.Val
		case "rel":
			rel = strings.ToLower(a.Val)
		}
	}

	if href == "" || strings.HasPrefix(href, "mailto:") || strings.HasPrefix(href, "tel:") || strings.HasPrefix(href, "javascript:") || href == "#" {
		return
	}

	data.TotalLinks++
	if strings.Contains(rel, "nofollow") {
		data.NofollowLinks++
	}

	if data.pageURL != "" {
		if isInternalLink(href, data.pageURL) {
			data.InternalLinks++
		} else {
			data.ExternalLinks++
		}
	}
}

func isInternalLink(href, pageURL string) bool {
	parsed, err := url.Parse(href)
	if err != nil {
		return false
	}
	// Relative URLs are internal
	if parsed.Host == "" {
		return true
	}
	base, err := url.Parse(pageURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, base.Host)
}

func countWords(s string) int {
	count := 0
	inWord := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}

func parseMeta(n *html.Node, data *SEOData) {
	tag := MetaTag{}
	for _, a := range n.Attr {
		switch a.Key {
		case "name":
			tag.Name = strings.ToLower(a.Val)
		case "property":
			tag.Property = strings.ToLower(a.Val)
		case "content":
			tag.Content = a.Val
		case "charset":
			tag.Charset = a.Val
			data.Charset = a.Val
		case "http-equiv":
			tag.HttpEquiv = strings.ToLower(a.Val)
		}
	}
	data.MetaTags = append(data.MetaTags, tag)

	content := tag.Content

	switch tag.Name {
	case "description":
		data.MetaDescription = content
	case "robots":
		data.Robots = content
	case "viewport":
		data.Viewport = content
	case "twitter:card":
		data.TwitterCard = content
	case "twitter:title":
		data.TwitterTitle = content
	case "twitter:description":
		data.TwitterDescription = content
	case "twitter:image":
		data.TwitterImage = content
	case "twitter:image:alt":
		data.TwitterImageAlt = content
	case "twitter:site":
		data.TwitterSite = content
	case "twitter:creator":
		data.TwitterCreator = content
	}

	switch tag.Property {
	case "og:title":
		data.OGTitle = content
	case "og:type":
		data.OGType = content
	case "og:image":
		data.OGImage = content
	case "og:image:alt":
		data.OGImageAlt = content
	case "og:image:width":
		data.OGImageWidth = content
	case "og:image:height":
		data.OGImageHeight = content
	case "og:url":
		data.OGURL = content
	case "og:description":
		data.OGDescription = content
	case "og:site_name":
		data.OGSiteName = content
	case "og:locale":
		data.OGLocale = content
	}

	if tag.HttpEquiv == "content-type" && data.Charset == "" {
		if strings.Contains(strings.ToLower(content), "utf-8") {
			data.Charset = "UTF-8"
		}
	}
}

func parseLink(n *html.Node, data *SEOData) {
	tag := LinkTag{}
	for _, a := range n.Attr {
		switch a.Key {
		case "rel":
			tag.Rel = strings.ToLower(a.Val)
		case "href":
			tag.Href = a.Val
		case "hreflang":
			tag.Hreflang = a.Val
		case "type":
			tag.Type = a.Val
		}
	}
	data.LinkTags = append(data.LinkTags, tag)

	switch tag.Rel {
	case "canonical":
		data.Canonical = tag.Href
	case "alternate":
		if tag.Hreflang != "" {
			data.Hreflang = append(data.Hreflang, HreflangEntry{
				Lang: tag.Hreflang,
				Href: tag.Href,
			})
		}
	case "icon", "shortcut icon", "apple-touch-icon":
		data.HasFavicon = true
	}
}

func parseScript(n *html.Node, data *SEOData) {
	isJSONLD := false
	for _, a := range n.Attr {
		if a.Key == "type" && a.Val == "application/ld+json" {
			isJSONLD = true
		}
	}
	if !isJSONLD || n.FirstChild == nil {
		return
	}

	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(n.FirstChild.Data), &obj); err == nil {
		data.JSONLDBlocks = append(data.JSONLDBlocks, obj)
	}
}

func getTextContent(n *html.Node) string {
	var buf strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			buf.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.TrimSpace(buf.String())
}
