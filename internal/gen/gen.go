// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/width"
)

// asciiWhitespace is a set of ASCII whitespace characters defined by the HTML spec.
// https://infra.spec.whatwg.org/#ascii-whitespace
var asciiWhitespace = "\t\n\f\r "

func isASCIIWhitespace(r rune) bool {
	for _, w := range asciiWhitespace {
		if r == w {
			return true
		}
	}
	return false
}

func Run() error {
	const (
		outDir = "_site"
		inDir  = "contents"
	)

	if err := removeHTMLs("_site"); err != nil {
		return err
	}
	if err := generateHTMLs(outDir, inDir); err != nil {
		return err
	}
	return nil
}

func removeHTMLs(outDir string) error {
	var wg errgroup.Group
	if err := filepath.Walk(outDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		wg.Go(func() error {
			return os.Remove(path)
		})
		return nil
	}); err != nil {
		return err
	}

	if err := wg.Wait(); err != nil {
		return err
	}
	return nil
}

func generateHTMLs(outDir, inDir string) error {
	datetime := time.Now().UTC().Format("20060102150405")

	var wg errgroup.Group
	if err := filepath.Walk(inDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		path, err = filepath.Rel(inDir, path)
		if err != nil {
			return err
		}

		wg.Go(func() error {
			return generateHTML(path, outDir, inDir, datetime)
		})
		return nil
	}); err != nil {
		return err
	}

	if err := wg.Wait(); err != nil {
		return err
	}
	return nil
}

func generateHTML(path string, outDir, inDir string, datetime string) error {
	inPath := filepath.Join(inDir, path)
	outPath := filepath.Join(outDir, path)

	in, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer in.Close()

	node, err := html.Parse(bufio.NewReader(in))
	if err != nil {
		return err
	}

	htmle := getElementByName(node, "html")
	head := getElementByName(htmle, "head")
	if getElement(head, func(n *html.Node) bool {
		if n.Data != "meta" {
			return false
		}
		for _, a := range n.Attr {
			if a.Key == "name" && a.Val == "description" {
				return true
			}
		}
		return false
	}) == nil {
		// TODO: Generate a good description.
		head.AppendChild(&html.Node{
			Type: html.ElementNode,
			Data: "meta",
			Attr: []html.Attribute{
				{
					Key: "name",
					Val: "description",
				},
				{
					Key: "content",
					Val: "Hajime Hoshi is a software engineer in Tokyo",
				},
			},
		})
	}
	head.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "meta",
		Attr: []html.Attribute{
			{
				Key: "name",
				Val: "viewport",
			},
			{
				Key: "content",
				Val: "width=device-width, initial-scale=1",
			},
		},
	})
	head.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "meta",
		Attr: []html.Attribute{
			{
				Key: "charset",
				Val: "utf-8",
			},
		},
	})
	head.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "link",
		Attr: []html.Attribute{
			{
				Key: "rel",
				Val: "stylesheet",
			},
			{
				Key: "href",
				Val: fmt.Sprintf("/style.css?%s", datetime),
			},
		},
	})
	head.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "link",
		Attr: []html.Attribute{
			{
				Key: "rel",
				Val: "icon",
			},
			{
				Key: "href",
				Val: "/hajimehoshi.png",
			},
			{
				Key: "type",
				Val: "image/png",
			},
		},
	})
	titleStr := "hajimehoshi.com"
	if path != "index.html" {
		title := getElementByName(htmle, "h1").FirstChild.Data
		titleStr = fmt.Sprintf("%s - %s", title, titleStr)
	}
	head.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "title",
		FirstChild: &html.Node{
			Type: html.TextNode,
			Data: titleStr,
		},
	})
	style := &html.Node{
		Type: html.ElementNode,
		Data: "style",
	}
	style.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: `.thin-space:after {
  content: '\2005';
}`,
	})
	head.AppendChild(style)

	removeInterElementWhitespace(node)
	processNewLines(node)
	insertNodeBetweenWideAndNarrow(node, &html.Node{
		Type: html.ElementNode,
		Data: "span",
		Attr: []html.Attribute{
			{
				Key: "class",
				Val: "thin-space",
			},
		},
	})

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	w := bufio.NewWriter(out)
	if err := html.Render(w, node); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

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

	// Process child text nodes first.
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if n.Type != html.TextNode {
			continue
		}

		var data string

		if shouldReserveSpaceAtHead(n) {
			data += " "
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

		if shouldReserveSpaceAtTail(n) {
			data += " "
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
}

func insertNodeBetweenWideAndNarrow(node *html.Node, insertingNode *html.Node) {
	if node.Type == html.ElementNode {
		if isMetadataElementName(node.Data) {
			return
		}
		if node.Data == "pre" {
			return
		}
	}

	// Insert dummy empty text nodes between two elements. This might be replaced with insertingNode later.
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
	for n := node.FirstChild; n != nil; n = next {
		next = n.NextSibling

		if n.Type != html.TextNode {
			continue
		}

		var tokens []string
		var lastI int
		for i, r := range n.Data {
			if i == 0 {
				continue
			}
			if prevR, _ := utf8.DecodeLastRuneInString(n.Data[:i]); shouldHaveThinSpace(prevR, r) {
				tokens = append(tokens, n.Data[lastI:i])
				lastI = i
			}
		}
		tokens = append(tokens, n.Data[lastI:])

		prevR := lastRuneBefore(n)
		nextR := firstRuneAfter(n)

		parent := n.Parent
		parent.RemoveChild(n)

		insertSpan := func() {
			node := &html.Node{
				Type:      insertingNode.Type,
				DataAtom:  insertingNode.DataAtom,
				Data:      insertingNode.Data,
				Namespace: insertingNode.Namespace,
			}
			node.Attr = make([]html.Attribute, len(insertingNode.Attr))
			copy(node.Attr, insertingNode.Attr)
			parent.InsertBefore(node, next)
		}

		if len(tokens) > 0 {
			if r, _ := utf8.DecodeRuneInString(tokens[0]); shouldHaveThinSpace(prevR, r) {
				insertSpan()
			}
			for i, t := range tokens {
				parent.InsertBefore(&html.Node{
					Type: html.TextNode,
					Data: t,
				}, next)
				if i == len(tokens)-1 {
					continue
				}
				insertSpan()
			}
			if r, _ := utf8.DecodeLastRuneInString(tokens[len(tokens)-1]); shouldHaveThinSpace(r, nextR) {
				insertSpan()
			}
		} else {
			if shouldHaveThinSpace(prevR, nextR) {
				insertSpan()
			}
		}
	}

	// Process child element nodes next.
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.TextNode {
			continue
		}
		insertNodeBetweenWideAndNarrow(n, insertingNode)
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

func firstRuneAfter(node *html.Node) rune {
	node = nextVisibleTextNode(node)
	if node == nil {
		return -1
	}

	r, _ := utf8.DecodeRuneInString(node.Data)
	return r
}

func lastRuneBefore(node *html.Node) rune {
	node = prevVisibleTextNode(node)
	if node == nil {
		return -1
	}

	r, _ := utf8.DecodeLastRuneInString(node.Data)
	return r
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
		// Skip if the element is not visible.
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
		// Skip if the element is not visible.
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

func shouldReserveSpaceAtHead(node *html.Node) bool {
	if node.Type != html.TextNode {
		panic("gen: node must be a text")
	}

	if node.Data == "" {
		return false
	}

	prevN := prevVisibleTextNode(node)
	if prevN == nil {
		return false
	}

	prev := prevN.Data
	current := node.Data
	if !isASCIIWhitespace(rune(prev[len(prev)-1])) && !isASCIIWhitespace(rune(current[0])) {
		return false
	}

	if hasNewLineRight(prev) {
		prev = strings.TrimRight(prev, asciiWhitespace)
	}
	if hasNewLineLeft(current) {
		current = strings.TrimLeft(current, asciiWhitespace)
	}

	r0, _ := utf8.DecodeLastRuneInString(prev)
	r1, _ := utf8.DecodeRuneInString(current)
	return shouldReserveSpaceBetweenRunes(r0, r1)
}

func shouldReserveSpaceAtTail(node *html.Node) bool {
	if node.Type != html.TextNode {
		panic("gen: node must be a text")
	}

	if node.Data == "" {
		return false
	}

	nextN := nextVisibleTextNode(node)
	if nextN == nil {
		return false
	}

	current := node.Data
	next := nextN.Data
	if !isASCIIWhitespace(rune(current[len(current)-1])) && !isASCIIWhitespace(rune(next[0])) {
		return false
	}

	if hasNewLineRight(current) {
		current = strings.TrimRight(current, asciiWhitespace)
	}
	if hasNewLineLeft(next) {
		next = strings.TrimLeft(next, asciiWhitespace)
	}

	r0, _ := utf8.DecodeLastRuneInString(current)
	r1, _ := utf8.DecodeRuneInString(next)
	return shouldReserveSpaceBetweenRunes(r0, r1)
}

func shouldHaveThinSpace(r0, r1 rune) bool {
	if r0 == -1 || r1 == -1 {
		return false
	}
	if unicode.IsSpace(r0) || unicode.IsSpace(r1) {
		return false
	}
	if unicode.IsPunct(r0) || unicode.IsPunct(r1) {
		return false
	}

	k0 := width.LookupRune(r0).Kind()
	k1 := width.LookupRune(r1).Kind()
	w0 := k0 == width.EastAsianWide || k0 == width.EastAsianFullwidth
	w1 := k1 == width.EastAsianWide || k1 == width.EastAsianFullwidth
	return w0 != w1
}

func isMetadataElementName(name string) bool {
	for _, n := range []string{"base", "link", "meta", "noscript", "script", "style", "template", "title"} {
		if name == n {
			return true
		}
	}
	return false
}

func isPhrasingElementName(name string) bool {
	for _, n := range []string{"a", "abbr", "area", "audio", "b", "bdi", "bdo", "br", "button", "canvas", "cite", "code", "data", "datalist", "del", "dfn", "em", "embed", "i", "iframe", "img", "input", "ins", "kbd", "label", "link", "map", "mark", "math", "meta", "meter", "noscript", "object", "output", "picture", "progress", "q", "ruby", "s", "samp", "script", "select", "slot", "small", "span", "strong", "sub", "sup", "svg", "template", "textarea", "time", "u", "var", "video", "wbr"} {
		if name == n {
			return true
		}
	}
	return false
}

func hasNewLineLeft(str string) bool {
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

func hasNewLineRight(str string) bool {
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
	return false
}
