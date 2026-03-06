package parser

import (
	"bytes"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type PageLink struct {
	Href string
	Text string
}

func ParseLinks(body []byte, baseURL string) ([]PageLink, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)
	var links []PageLink

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			var href string
			for _, a := range n.Attr {
				if a.Key == "href" {
					href = a.Val
				}
			}
			if href != "" {
				resolved := resolveURL(href, base)
				if resolved != "" && !seen[resolved] {
					seen[resolved] = true
					text := strings.TrimSpace(getTextContent(n))
					links = append(links, PageLink{Href: resolved, Text: text})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return links, nil
}

func resolveURL(href string, base *url.URL) string {
	href = strings.TrimSpace(href)

	// Skip non-http links
	if strings.HasPrefix(href, "mailto:") ||
		strings.HasPrefix(href, "tel:") ||
		strings.HasPrefix(href, "javascript:") ||
		href == "#" || href == "" {
		return ""
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return ""
	}

	resolved := base.ResolveReference(parsed)
	// Strip fragment
	resolved.Fragment = ""
	return resolved.String()
}

func IsInternal(link string, baseURL string) bool {
	linkParsed, err := url.Parse(link)
	if err != nil {
		return false
	}
	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return false
	}
	return strings.EqualFold(linkParsed.Host, baseParsed.Host)
}
