// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen

import (
	"bufio"
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/width"
)

//go:embed *.html
var htmlFiles embed.FS

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

func Run() error {
	const (
		outDir = "_site"
		inDir  = "contents"
	)

	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	if err := copyNonHTMLFiles(outDir, inDir); err != nil {
		return err
	}
	if err := generateHTMLs(outDir, inDir); err != nil {
		return err
	}
	return nil
}

func isIgnoredFile(path string) bool {
	if strings.HasPrefix(filepath.Base(path), "#") {
		return true
	}
	if strings.HasPrefix(filepath.Base(path), "_") {
		return true
	}
	if strings.HasSuffix(path, "~") {
		return true
	}
	return false
}

func copyNonHTMLFiles(outDir, inDir string) error {
	var wg errgroup.Group
	if err := filepath.Walk(inDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".html" {
			return nil
		}
		if isIgnoredFile(path) {
			return nil
		}
		wg.Go(func() error {
			inRelPath, err := filepath.Rel(inDir, path)
			if err != nil {
				return err
			}
			outPath := filepath.Join(outDir, inRelPath)
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}

			out, err := os.Create(outPath)
			if err != nil {
				return err
			}
			defer out.Close()

			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()

			switch filepath.Ext(path) {
			case ".css":
				outbuf := bufio.NewWriter(out)
				if err := minifyCSS(outbuf, bufio.NewReader(in)); err != nil {
					return err
				}
				if err := outbuf.Flush(); err != nil {
					return err
				}
			case ".js":
				outbuf := bufio.NewWriter(out)
				if err := minifyJS(outbuf, bufio.NewReader(in)); err != nil {
					return err
				}
				if err := outbuf.Flush(); err != nil {
					return err
				}
			default:
				if _, err := io.Copy(out, in); err != nil {
					return err
				}
			}
			return nil
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
		if isIgnoredFile(path) {
			return nil
		}
		path, err = filepath.Rel(inDir, path)
		if err != nil {
			return err
		}

		wg.Go(func() error {
			return generateHTML(path, outDir, inDir)
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

func generateHTML(path string, outDir, inDir string) error {
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
	if _, ok := getAttribute(htmle, "lang"); !ok {
		lang := "en"
		if dir := filepath.Dir(path); dir != "." {
			if ts := strings.Split(dir, string(filepath.Separator)); len(ts) > 0 {
				lang = ts[0]
			}
		}
		htmle.Attr = append(htmle.Attr, html.Attribute{
			Key: "lang",
			Val: lang,
		})
	}

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
	// Preload woff2 files.
	urls, err := woff2URLsInCSS(filepath.Join(outDir, "style.css"))
	if err != nil {
		return err
	}
	if len(urls) == 0 {
		return fmt.Errorf("gen: no woff2 files")
	}
	for _, url := range urls {
		head.AppendChild(&html.Node{
			Type: html.ElementNode,
			Data: "link",
			Attr: []html.Attribute{
				{
					Key: "rel",
					Val: "preload",
				},
				{
					Key: "href",
					Val: url,
				},
				{
					Key: "as",
					Val: "font",
				},
				{
					Key: "crossorigin",
					Val: "anonymous",
				},
			},
		})
	}
	h, err := fileHash(filepath.Join(outDir, "style.css"))
	if err != nil {
		return err
	}
	head.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "link",
		Attr: []html.Attribute{
			{
				Key: "preload",
				Val: "stylesheet",
			},
			{
				Key: "href",
				Val: fmt.Sprintf("/style.css?%s", h),
			},
			{
				Key: "as",
				Val: "style",
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
				Val: fmt.Sprintf("/style.css?%s", h),
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
				Val: "/favicon.webp?20251129",
			},
			{
				Key: "type",
				Val: "image/webp",
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
	var cssBuf bytes.Buffer
	if err := minifyCSS(&cssBuf, bytes.NewReader([]byte(`.thin-space:after {
	content: '\2006';
}`))); err != nil {
		return err
	}
	style.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: cssBuf.String(),
	})
	head.AppendChild(style)

	if err := addHeader(node); err != nil {
		return err
	}

	h, err = fileHash(filepath.Join(outDir, "script.js"))
	if err != nil {
		return err
	}
	head.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "script",
		Attr: []html.Attribute{
			{
				Key: "src",
				Val: fmt.Sprintf("/script.js?%s", h),
			},
			{
				Key: "defer",
			},
		},
	})

	removeComments(node)
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

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

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

func getAttribute(node *html.Node, key string) (html.Attribute, bool) {
	for _, a := range node.Attr {
		if a.Key == key {
			return a, true
		}
	}
	return html.Attribute{}, false
}

func addHeader(node *html.Node) error {
	body := getElementByName(node, "body")
	main := getElementByName(node, "main")

	f, err := htmlFiles.Open("header.html")
	if err != nil {
		return err
	}
	defer f.Close()

	header, err := html.Parse(f)
	if err != nil {
		return err
	}
	body.InsertBefore(header, main)
	return nil
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

func shouldHaveThinSpace(r0, r1 rune) bool {
	if r0 == -1 || r1 == -1 {
		return false
	}

	if unicode.IsSpace(r0) || unicode.IsSpace(r1) {
		return false
	}

	k0 := width.LookupRune(r0).Kind()
	k1 := width.LookupRune(r1).Kind()
	w0 := k0 == width.EastAsianWide || k0 == width.EastAsianFullwidth
	w1 := k1 == width.EastAsianWide || k1 == width.EastAsianFullwidth
	if w0 == w1 {
		return false
	}

	return (w0 && !unicode.IsPunct(r0)) != (w1 && !unicode.IsPunct(r1))
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

func woff2URLsInCSS(cssFile string) ([]string, error) {
	f, err := os.Open(cssFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	re := regexp.MustCompile(`url\((https://.+?woff2)\)`)

	var urls []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		m := re.FindStringSubmatch(s.Text())
		if m == nil {
			continue
		}
		urls = append(urls, m[1])
	}
	return urls, nil
}

func minifyCSS(out io.Writer, in io.Reader) error {
	css, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	r := api.Transform(string(css), api.TransformOptions{
		Loader:            api.LoaderCSS,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})
	if len(r.Errors) > 0 {
		var msgs []string
		for _, e := range r.Errors {
			msgs = append(msgs, e.Text)
		}
		return fmt.Errorf("gen: minifying CSS failed: %s", strings.Join(msgs, ", "))
	}
	if _, err := out.Write(bytes.TrimSpace(r.Code)); err != nil {
		return err
	}
	return nil
}

func minifyJS(out io.Writer, in io.Reader) error {
	js, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	r := api.Transform(string(js), api.TransformOptions{
		Loader:            api.LoaderJS,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})
	if len(r.Errors) > 0 {
		var msgs []string
		for _, e := range r.Errors {
			msgs = append(msgs, e.Text)
		}
		return fmt.Errorf("gen: minifying JS failed: %s", strings.Join(msgs, ", "))
	}
	if _, err := out.Write(bytes.TrimSpace(r.Code)); err != nil {
		return err
	}
	return nil
}
