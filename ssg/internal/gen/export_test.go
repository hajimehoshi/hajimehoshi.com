// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen

import (
	"golang.org/x/net/html"
)

func ExtractMetadataFromHTML(content []byte) (map[string]any, []byte, error) {
	return extractMetadataFromHTML(content)
}

func PagePath(relPath string, keepHTMLExtension bool) string {
	return pagePath(relPath, keepHTMLExtension)
}

func PageURL(siteURL, path string) string {
	return pageURL(siteURL, path)
}

func PageHref(href string, keepHTMLExtension bool) string {
	return pageHref(href, keepHTMLExtension)
}

func RewritePageLinks(node *html.Node, keepHTMLExtension bool) {
	rewritePageLinks(node, keepHTMLExtension)
}

func AddResourceVersions(node *html.Node, outDir, pageDir string) error {
	return addResourceVersions(node, outDir, pageDir)
}

func RemoveInterElementWhitespace(node *html.Node) {
	removeInterElementWhitespace(node)
}

func ProcessNewLines(node *html.Node) {
	processNewLines(node)
}
