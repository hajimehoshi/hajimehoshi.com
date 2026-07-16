// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package htmlrewrite

import (
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/text/width"
)

// Minify drops the document parts that do not affect rendering: comments, and
// the whitespace the source's formatting introduces.
func Minify(node *html.Node) {
	removeHeadWhitespace(node)
	removeComments(node)
	removeInterElementWhitespace(node)
	processNewLines(node)
}

func isASCIIWhitespace(r rune) bool {
	for _, w := range asciiWhitespace {
		if r == w {
			return true
		}
	}
	return false
}

func hasASCIIWhitespaceHead(str string) bool {
	return strings.TrimLeft(str, asciiWhitespace) != str
}

func hasASCIIWhitespaceTail(str string) bool {
	return strings.TrimRight(str, asciiWhitespace) != str
}

func hasASCIIWhitespaceWithNewLineHead(str string) bool {
	for _, r := range str {
		if r == '\n' {
			return true
		}
		if !isASCIIWhitespace(r) {
			return false
		}
	}
	return false
}

func hasASCIIWhitespaceWithNewLineTail(str string) bool {
	for {
		r, s := utf8.DecodeLastRuneInString(str)
		if r == '\n' {
			return true
		}
		if !isASCIIWhitespace(r) {
			return false
		}
		str = str[:len(str)-s]
	}
}

// removeHeadWhitespace drops the formatting whitespace between the <head>'s
// child elements. The head holds only metadata elements, so inter-element
// whitespace is insignificant and would otherwise survive as stray spaces.
func removeHeadWhitespace(node *html.Node) {
	head := getElementByName(node, "head")
	if head == nil {
		return
	}
	var next *html.Node
	for n := head.FirstChild; n != nil; n = next {
		next = n.NextSibling
		if n.Type == html.TextNode && strings.Trim(n.Data, asciiWhitespace) == "" {
			head.RemoveChild(n)
		}
	}
}

func removeComments(node *html.Node) {
	var next *html.Node
	for n := node.FirstChild; n != nil; n = next {
		next = n.NextSibling

		if n.Type != html.CommentNode {
			removeComments(n)
			continue
		}

		prev := n.PrevSibling
		n.Parent.RemoveChild(n)

		// Merge two adjenct text nodes.
		if prev != nil && prev.Type == html.TextNode && next != nil && next.Type == html.TextNode {
			prev.Data += next.Data
			next2 := next.NextSibling
			next.Parent.RemoveChild(next)
			next = next2
		}
	}
}

func removeInterElementWhitespace(node *html.Node) {
	if node.Type == html.ElementNode {
		if isMetadataElementName(node.Data) {
			return
		}
		if node.Data == "pre" {
			return
		}
	}

	var next *html.Node
	for n := node.FirstChild; n != nil; n = next {
		next = n.NextSibling

		if n.Type != html.TextNode {
			removeInterElementWhitespace(n)
			continue
		}

		if strings.Trim(n.Data, asciiWhitespace) != "" {
			continue
		}

		// Replace a whitespace text with one space character.
		n.Data = " "

		if n.PrevSibling == nil && n.NextSibling == nil {
			continue
		}

		// If a node is in between phrasing elements, reserve this.
		if n.PrevSibling != nil && n.PrevSibling.Type == html.ElementNode && isPhrasingElementName(n.PrevSibling.Data) &&
			n.NextSibling != nil && n.NextSibling.Type == html.ElementNode && isPhrasingElementName(n.NextSibling.Data) {
			continue
		}

		n.Parent.RemoveChild(n)
	}
}

func processNewLines(node *html.Node) {
	if node.Type == html.ElementNode {
		if isMetadataElementName(node.Data) {
			return
		}
		if node.Data == "pre" {
			return
		}
	}

	reNewLineAndSpace := regexp.MustCompile(`[\t\n\f\r ]*\n[\t\n\f\r ]*`)
	reSpace := regexp.MustCompile(`[\t\n\f\r ]+`)

	// Insert dummy empty text nodes between two elements.
	var next *html.Node
	for n := node.FirstChild; n != nil; n = next {
		next = n.NextSibling

		if n.Type != html.ElementNode {
			continue
		}
		if n.NextSibling == nil {
			continue
		}
		if n.NextSibling.Type != html.ElementNode {
			continue
		}
		n.InsertBefore(&html.Node{
			Type: html.TextNode,
		}, n.NextSibling)
	}

	// Process child text nodes first.
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if n.Type != html.TextNode {
			continue
		}

		prev := prevVisibleTextNode(n)
		next := nextVisibleTextNode(n)

		var data string
		if len(n.Data) > 0 && (strings.Trim(n.Data, asciiWhitespace) != "" || !strings.Contains(n.Data, "\n")) {
			if prev != nil && (hasASCIIWhitespaceTail(prev.Data) || hasASCIIWhitespaceHead(n.Data)) {
				if shouldReserveSpaceBetweenTexts(prev.Data, n.Data) {
					data += " "
				}
			}
			for _, t := range reNewLineAndSpace.Split(n.Data, -1) {
				if len(data) > 0 && t != "" {
					r0, _ := utf8.DecodeLastRuneInString(data)
					r1, _ := utf8.DecodeRuneInString(t)
					if shouldReserveSpaceBetweenRunes(r0, r1) {
						data += " "
					}
				}
				data += t
			}
			if next != nil && (hasASCIIWhitespaceTail(n.Data) || hasASCIIWhitespaceHead(next.Data)) {
				if shouldReserveSpaceBetweenTexts(n.Data, next.Data) {
					data += " "
				}
			}
		} else if prev != nil && next != nil && (hasASCIIWhitespaceTail(prev.Data) || hasASCIIWhitespaceHead(next.Data) || len(n.Data) > 0) {
			if shouldReserveSpaceBetweenTexts(prev.Data, next.Data) {
				data += " "
			}
		}

		data = reSpace.ReplaceAllString(data, " ")

		n.Data = data
	}

	// Process child element nodes next.
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.TextNode {
			continue
		}
		processNewLines(n)
	}

	// Remove dummy empty text nodes.
	for n := node.FirstChild; n != nil; n = next {
		next = n.NextSibling

		if n.Type != html.TextNode {
			continue
		}
		if len(n.Data) > 0 {
			continue
		}
		n.Parent.RemoveChild(n)
	}
}

func nextVisibleTextNode(node *html.Node) *html.Node {
	for {
		node = nextVisibleNode(node)
		if node == nil {
			return nil
		}

		if node.Type == html.TextNode && len(node.Data) > 0 {
			return node
		}
	}
}

func prevVisibleTextNode(node *html.Node) *html.Node {
	for {
		node = prevVisibleNode(node)
		if node == nil {
			return nil
		}

		if node.Type == html.TextNode && len(node.Data) > 0 {
			return node
		}
	}
}

func nextVisibleNode(node *html.Node) *html.Node {
	if node.NextSibling == nil {
		return nil
	}
	node = node.NextSibling

	// Search the first visible descendant.
	for {
		if node.Type == html.ElementNode {
			if !isPhrasingElementName(node.Data) {
				return nil
			}
		}

		if node.FirstChild == nil {
			break
		}
		node = node.FirstChild
	}

	return node
}

func prevVisibleNode(node *html.Node) *html.Node {
	if node.PrevSibling == nil {
		return nil
	}
	node = node.PrevSibling

	// Search the last visible descendant.
	for {
		if node.Type == html.ElementNode {
			if !isPhrasingElementName(node.Data) {
				return nil
			}
		}

		if node.LastChild == nil {
			break
		}
		node = node.LastChild
	}

	return node
}

func shouldReserveSpaceBetweenRunes(r0, r1 rune) bool {
	if r0 == -1 || r1 == -1 {
		return false
	}
	k0 := width.LookupRune(r0).Kind()
	k1 := width.LookupRune(r1).Kind()
	w0 := k0 == width.EastAsianWide || k0 == width.EastAsianFullwidth
	w1 := k1 == width.EastAsianWide || k1 == width.EastAsianFullwidth
	return !w0 && !w1
}

func shouldReserveSpaceBetweenTexts(d0, d1 string) bool {
	if d0 == "" && d1 == "" {
		return false
	}

	if hasASCIIWhitespaceWithNewLineTail(d0) {
		d0 = strings.TrimRight(d0, asciiWhitespace)
	}
	if hasASCIIWhitespaceWithNewLineHead(d1) {
		d1 = strings.TrimLeft(d1, asciiWhitespace)
	}

	r0, _ := utf8.DecodeLastRuneInString(d0)
	r1, _ := utf8.DecodeRuneInString(d1)
	return shouldReserveSpaceBetweenRunes(r0, r1)
}

func isMetadataElementName(name string) bool {
	return slices.Contains([]string{"base", "link", "meta", "noscript", "script", "style", "template", "title"}, name)
}

func isPhrasingElementName(name string) bool {
	return slices.Contains([]string{"a", "abbr", "area", "audio", "b", "bdi", "bdo", "br", "button", "canvas", "cite", "code", "data", "datalist", "del", "dfn", "em", "embed", "i", "iframe", "img", "input", "ins", "kbd", "label", "link", "map", "mark", "math", "meta", "meter", "noscript", "object", "output", "picture", "progress", "q", "ruby", "s", "samp", "script", "select", "slot", "small", "span", "strong", "sub", "sup", "svg", "template", "textarea", "time", "u", "var", "video", "wbr"}, name)
}
