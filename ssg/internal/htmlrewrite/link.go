// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package htmlrewrite

import (
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

// pageHref returns href with its path adjusted to match the URL of the page it
// points at: a trailing index.html is dropped, and unless keepHTMLExtension is
// set, so is any other .html extension. An href that does not point at a local
// page is returned unchanged.
func pageHref(href string, keepHTMLExtension bool) string {
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	if u.Scheme != "" || u.Host != "" || u.Path == "" {
		return href
	}
	switch {
	case u.Path == "index.html":
		// An empty path would denote the current page rather than its
		// directory.
		u.Path = "./"
	case strings.HasSuffix(u.Path, "/index.html"):
		u.Path = strings.TrimSuffix(u.Path, "index.html")
	case !keepHTMLExtension && strings.HasSuffix(u.Path, ".html"):
		u.Path = strings.TrimSuffix(u.Path, ".html")
	default:
		return href
	}
	return u.String()
}

// RewritePageLinks adjusts every hyperlink to a local page so that it matches
// the URL that page is served at.
func RewritePageLinks(node *html.Node, keepHTMLExtension bool) {
	if node.Type == html.ElementNode {
		for i := range node.Attr {
			if !isHyperlinkAttr(node, node.Attr[i].Key) {
				continue
			}
			node.Attr[i].Val = pageHref(node.Attr[i].Val, keepHTMLExtension)
		}
	}
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		RewritePageLinks(n, keepHTMLExtension)
	}
}
