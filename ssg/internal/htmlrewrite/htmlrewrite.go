// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

// Package htmlrewrite provides the transforms a generated page's parsed HTML
// document goes through before it is rendered. Every transform rewrites the
// document in place.
package htmlrewrite

import (
	"net/url"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// asciiWhitespace is a set of ASCII whitespace characters defined by the HTML spec.
// https://infra.spec.whatwg.org/#ascii-whitespace
var asciiWhitespace = "\t\n\f\r "

func getElement(node *html.Node, f func(*html.Node) bool) *html.Node {
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if n.Type != html.ElementNode {
			continue
		}
		if f(n) {
			return n
		}
		if n := getElement(n, f); n != nil {
			return n
		}
	}
	return nil
}

func getElementByName(node *html.Node, name string) *html.Node {
	return getElement(node, func(n *html.Node) bool {
		return n.Data == name
	})
}

// localFilePath resolves rawURL to a file path under outDir. It reports false
// for URLs that do not point at a local file, such as external URLs (with a
// scheme or host, including protocol-relative ones). pageDir is the document's
// directory relative to the site root, used to resolve relative URLs.
func localFilePath(rawURL, outDir, pageDir string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if u.Scheme != "" || u.Host != "" || u.Path == "" {
		return "", false
	}
	if strings.HasPrefix(u.Path, "/") {
		// A leading slash denotes the site root; resolve under outDir.
		return filepath.Join(outDir, filepath.FromSlash(u.Path[1:])), true
	}
	return filepath.Join(outDir, pageDir, filepath.FromSlash(u.Path)), true
}
