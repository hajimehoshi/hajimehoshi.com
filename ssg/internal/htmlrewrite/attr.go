// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package htmlrewrite

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
)

// elemAttr identifies an element attribute by the element and attribute names.
type elemAttr struct {
	elem string
	attr string
}

// resourceAttrs is the set of attributes whose value is a single URL of a
// resource the browser fetches, as opposed to a hyperlink (see hyperlinkAttrs).
// These are the attributes to version. srcset attributes (img, source) are
// excluded: their value is a URL list, whereas versionedURL handles only a
// single URL. link[href] is excluded: it is either kind depending on rel, and
// isResourceAttr classifies it.
var resourceAttrs = map[elemAttr]struct{}{
	{"img", "src"}:      {},
	{"script", "src"}:   {},
	{"audio", "src"}:    {},
	{"video", "src"}:    {},
	{"video", "poster"}: {},
	{"source", "src"}:   {},
	{"embed", "src"}:    {},
	{"track", "src"}:    {},
}

// hyperlinkAttrs is the set of attributes whose value can be a single URL of
// another page, as opposed to a resource the browser fetches (see
// resourceAttrs). cite attributes (blockquote, q, ins, del) are excluded: they
// reference a source document rather than link to one. link[href] is excluded:
// it is either kind depending on rel, and isHyperlinkAttr classifies it.
var hyperlinkAttrs = map[elemAttr]struct{}{
	{"a", "href"}:      {},
	{"area", "href"}:   {},
	{"iframe", "src"}:  {},
	{"form", "action"}: {},
	{"object", "data"}: {},
}

// resourceRels is the set of rel keywords denoting a <link> whose href is a
// resource the browser fetches. A <link> with any other rel is treated as a
// hyperlink, as versioning a page URL would hash an output file that the
// concurrent generation has possibly not written yet. prefetch and prerender
// are absent for that reason: their target is a page. A keyword whose href is
// an origin or an endpoint, such as preconnect or pingback, needs neither
// treatment and carries a host that both of them ignore.
var resourceRels = []string{
	"apple-touch-icon",
	"apple-touch-startup-image",
	"icon",
	"manifest",
	"mask-icon",
	"modulepreload",
	"preload",
	"stylesheet",
}

// isResourceLink reports whether the <link> element node points at a resource
// the browser fetches rather than at a page.
func isResourceLink(node *html.Node) bool {
	for _, a := range node.Attr {
		if a.Key != "rel" {
			continue
		}
		for t := range strings.FieldsSeq(a.Val) {
			if slices.ContainsFunc(resourceRels, func(r string) bool {
				return strings.EqualFold(t, r)
			}) {
				return true
			}
		}
	}
	return false
}

// isResourceAttr reports whether the node's attr holds a resource URL to
// version. A <link> is classified by its rel keywords, as the element name
// alone does not tell the two kinds apart.
func isResourceAttr(node *html.Node, attr string) bool {
	if node.Data == "link" && attr == "href" {
		return isResourceLink(node)
	}
	_, ok := resourceAttrs[elemAttr{node.Data, attr}]
	return ok
}

// isHyperlinkAttr reports whether the node's attr holds a URL that can point at
// another page. A <link> is classified by its rel keywords, as the element name
// alone does not tell the two kinds apart.
func isHyperlinkAttr(node *html.Node, attr string) bool {
	if node.Data == "link" && attr == "href" {
		return !isResourceLink(node)
	}
	_, ok := hyperlinkAttrs[elemAttr{node.Data, attr}]
	return ok
}
