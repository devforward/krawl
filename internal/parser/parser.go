package parser

import (
	"bytes"
	"encoding/json"
	"strings"

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

	// Raw collections
	MetaTags      []MetaTag
	LinkTags      []LinkTag
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
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	data := &SEOData{}
	parseNode(doc, data)
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
			// Only parse h1/h2 from body, skip deep traversal for perf
			parseHeadings(n, data)
			return
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parseNode(c, data)
	}
}

func parseHeadings(n *html.Node, data *SEOData) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "h1":
			data.H1 = append(data.H1, getTextContent(n))
		case "h2":
			data.H2 = append(data.H2, getTextContent(n))
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		parseHeadings(c, data)
	}
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
