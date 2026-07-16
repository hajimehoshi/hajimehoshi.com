// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package htmlrewrite

import (
	"bufio"
	"os"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// AddFontPreloads appends a <link rel="preload"> element to the end of the
// document's <head> for every woff2 font used by the document's local
// stylesheets. pageDir is the document's directory relative to the site root,
// used to resolve relative stylesheet URLs.
func AddFontPreloads(node *html.Node, outDir, pageDir string) error {
	head := getElementByName(node, "head")
	if head == nil {
		return nil
	}

	seen := map[string]struct{}{}
	for _, href := range stylesheetHrefs(node) {
		file, ok := localFilePath(href, outDir, pageDir)
		if !ok {
			continue
		}
		urls, err := woff2URLsInCSS(file)
		if err != nil {
			return err
		}
		for _, u := range urls {
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			head.AppendChild(&html.Node{
				Type:     html.ElementNode,
				Data:     "link",
				DataAtom: atom.Link,
				Attr: []html.Attribute{
					{Key: "rel", Val: "preload"},
					{Key: "href", Val: u},
					{Key: "as", Val: "font"},
					{Key: "crossorigin", Val: "anonymous"},
				},
			})
		}
	}
	return nil
}

// stylesheetHrefs returns the href values of the document's
// <link rel="stylesheet"> elements.
func stylesheetHrefs(node *html.Node) []string {
	var hrefs []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href string
			for _, a := range n.Attr {
				switch a.Key {
				case "rel":
					rel = a.Val
				case "href":
					href = a.Val
				}
			}
			if href != "" && slices.ContainsFunc(strings.Fields(rel), func(t string) bool {
				return strings.EqualFold(t, "stylesheet")
			}) {
				hrefs = append(hrefs, href)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(node)
	return hrefs
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
		for _, m := range re.FindAllStringSubmatch(s.Text(), -1) {
			urls = append(urls, m[1])
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return urls, nil
}
