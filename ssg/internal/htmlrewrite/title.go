// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package htmlrewrite

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// SetMissingTitle sets the document's title to the first <h1> text combined
// with the site name when the document has no non-empty <title>. This is a
// safety net for pages whose template does not emit a title, e.g. because the
// page has no title metadata.
func SetMissingTitle(node *html.Node, siteName string) {
	title := getElementByName(node, "title")
	if title != nil && title.FirstChild != nil && strings.Trim(title.FirstChild.Data, asciiWhitespace) != "" {
		return
	}

	text := siteName
	if h1 := getElementByName(node, "h1"); h1 != nil && h1.FirstChild != nil && h1.FirstChild.Type == html.TextNode {
		if t := strings.Trim(h1.FirstChild.Data, asciiWhitespace); t != "" {
			text = t + " – " + siteName
		}
	}

	if title == nil {
		head := getElementByName(node, "head")
		if head == nil {
			return
		}
		title = &html.Node{
			Type:     html.ElementNode,
			Data:     "title",
			DataAtom: atom.Title,
		}
		head.AppendChild(title)
	}
	for title.FirstChild != nil {
		title.RemoveChild(title.FirstChild)
	}
	title.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: text,
	})
}
