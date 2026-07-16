// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package htmlrewrite

import (
	"golang.org/x/net/html"
)

func PageHref(href string, keepHTMLExtension bool) string {
	return pageHref(href, keepHTMLExtension)
}

func RemoveInterElementWhitespace(node *html.Node) {
	removeInterElementWhitespace(node)
}

func ProcessNewLines(node *html.Node) {
	processNewLines(node)
}
