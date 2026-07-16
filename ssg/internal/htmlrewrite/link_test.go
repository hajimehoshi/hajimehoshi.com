// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package htmlrewrite_test

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/hajimehoshi/hajimehoshi.com/ssg/internal/htmlrewrite"
)

func TestPageHref(t *testing.T) {
	testCases := []struct {
		Name              string
		In                string
		KeepHTMLExtension bool
		Out               string
	}{
		{
			Name: "absolute page",
			In:   "/writings/foo.html",
			Out:  "/writings/foo",
		},
		{
			Name:              "absolute page, keeping the extension",
			In:                "/writings/foo.html",
			KeepHTMLExtension: true,
			Out:               "/writings/foo.html",
		},
		{
			Name: "relative page",
			In:   "foo.html",
			Out:  "foo",
		},
		{
			Name:              "relative page, keeping the extension",
			In:                "foo.html",
			KeepHTMLExtension: true,
			Out:               "foo.html",
		},
		{
			Name: "absolute index",
			In:   "/writings/index.html",
			Out:  "/writings/",
		},
		{
			Name:              "absolute index, keeping the extension",
			In:                "/writings/index.html",
			KeepHTMLExtension: true,
			Out:               "/writings/",
		},
		{
			Name: "root index",
			In:   "/index.html",
			Out:  "/",
		},
		{
			Name: "relative index",
			In:   "index.html",
			Out:  "./",
		},
		{
			Name:              "relative index, keeping the extension",
			In:                "index.html",
			KeepHTMLExtension: true,
			Out:               "./",
		},
		{
			Name: "parent index",
			In:   "../index.html",
			Out:  "../",
		},
		{
			Name: "name ending with index.html",
			In:   "myindex.html",
			Out:  "myindex",
		},
		{
			Name: "fragment",
			In:   "/writings/foo.html#section",
			Out:  "/writings/foo#section",
		},
		{
			Name: "query",
			In:   "/writings/foo.html?a=b",
			Out:  "/writings/foo?a=b",
		},
		{
			Name: "index with a fragment",
			In:   "index.html#section",
			Out:  "./#section",
		},
		{
			Name: "external page",
			In:   "https://example.com/foo.html",
			Out:  "https://example.com/foo.html",
		},
		{
			Name: "protocol-relative page",
			In:   "//example.com/foo.html",
			Out:  "//example.com/foo.html",
		},
		{
			Name: "mailto",
			In:   "mailto:foo@example.com",
			Out:  "mailto:foo@example.com",
		},
		{
			Name: "non-page resource",
			In:   "/style.css",
			Out:  "/style.css",
		},
		{
			Name: "fragment only",
			In:   "#section",
			Out:  "#section",
		},
		{
			Name: "empty",
			In:   "",
			Out:  "",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if got, want := htmlrewrite.PageHref(tc.In, tc.KeepHTMLExtension), tc.Out; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestRewritePageLinks(t *testing.T) {
	testCases := []struct {
		In                string
		KeepHTMLExtension bool
		Out               string
	}{
		{
			In:  `<a href="/writings/foo.html">foo</a>`,
			Out: `<a href="/writings/foo">foo</a>`,
		},
		{
			In:                `<a href="/writings/foo.html">foo</a>`,
			KeepHTMLExtension: true,
			Out:               `<a href="/writings/foo.html">foo</a>`,
		},
		{
			In:                `<a href="/writings/index.html">foo</a>`,
			KeepHTMLExtension: true,
			Out:               `<a href="/writings/">foo</a>`,
		},
		{
			In:  `<a href="https://example.com/foo.html">foo</a>`,
			Out: `<a href="https://example.com/foo.html">foo</a>`,
		},
		{
			In:  `<area href="/writings/foo.html"/>`,
			Out: `<area href="/writings/foo"/>`,
		},
		{
			In:  `<iframe src="/writings/foo.html"></iframe>`,
			Out: `<iframe src="/writings/foo"></iframe>`,
		},
		{
			In:  `<form action="/search.html"></form>`,
			Out: `<form action="/search"></form>`,
		},
		{
			In:  `<object data="/doc.html"></object>`,
			Out: `<object data="/doc"></object>`,
		},
		{
			// A resource attribute must not be rewritten.
			In:  `<img src="/foo.html"/>`,
			Out: `<img src="/foo.html"/>`,
		},
		{
			// A cite attribute references a document rather than links to it.
			In:  `<blockquote cite="/writings/foo.html"></blockquote>`,
			Out: `<blockquote cite="/writings/foo.html"></blockquote>`,
		},
	}
	for _, tc := range testCases {
		nodes, err := html.ParseFragment(bytes.NewBufferString(tc.In), nil)
		if err != nil {
			t.Error(err)
			continue
		}

		node := nodes[0]
		htmlrewrite.RewritePageLinks(node, tc.KeepHTMLExtension)

		var out bytes.Buffer
		if err := html.Render(&out, node); err != nil {
			t.Error(err)
			continue
		}
		got := out.String()
		got = strings.TrimPrefix(got, "<html><head></head><body>")
		got = strings.TrimSuffix(got, "</body></html>")
		want := tc.Out
		if got != want {
			t.Errorf("got: %q, want: %q (in: %q)", got, want, tc.In)
		}
	}
}

func TestRewritePageLinksLinkRel(t *testing.T) {
	testCases := []struct {
		Name              string
		In                string
		KeepHTMLExtension bool
		Out               string
	}{
		{
			Name: "canonical",
			In:   `<link rel="canonical" href="/writings/foo.html"/>`,
			Out:  `<link rel="canonical" href="/writings/foo"/>`,
		},
		{
			Name:              "canonical, keeping the extension",
			In:                `<link rel="canonical" href="/writings/foo.html"/>`,
			KeepHTMLExtension: true,
			Out:               `<link rel="canonical" href="/writings/foo.html"/>`,
		},
		{
			Name: "alternate",
			In:   `<link rel="alternate" hreflang="ja" href="/ja/index.html"/>`,
			Out:  `<link rel="alternate" hreflang="ja" href="/ja/"/>`,
		},
		{
			Name: "prev",
			In:   `<link rel="prev" href="/writings/foo.html"/>`,
			Out:  `<link rel="prev" href="/writings/foo"/>`,
		},
		{
			Name: "absolute URL",
			In:   `<link rel="canonical" href="https://example.com/foo.html"/>`,
			Out:  `<link rel="canonical" href="https://example.com/foo.html"/>`,
		},
		{
			// A resource rel is not a page link, so its href stays as it is
			// even when it ends with .html.
			Name: "stylesheet",
			In:   `<link rel="stylesheet" href="/foo.html"/>`,
			Out:  `<link rel="stylesheet" href="/foo.html"/>`,
		},
		{
			Name: "preload",
			In:   `<link rel="preload" href="/foo.html"/>`,
			Out:  `<link rel="preload" href="/foo.html"/>`,
		},
		{
			// A prefetch target is a page, so its URL has to follow the site's
			// URL style for the prefetched URL to be the one navigated to.
			Name: "prefetch",
			In:   `<link rel="prefetch" href="/writings/foo.html"/>`,
			Out:  `<link rel="prefetch" href="/writings/foo"/>`,
		},
		{
			// An origin is neither a resource nor a page.
			Name: "dns-prefetch",
			In:   `<link rel="dns-prefetch" href="https://res.example.com"/>`,
			Out:  `<link rel="dns-prefetch" href="https://res.example.com"/>`,
		},
		{
			// rel is a case-insensitive token list.
			Name: "shortcut icon",
			In:   `<link rel="Shortcut Icon" href="/foo.html"/>`,
			Out:  `<link rel="Shortcut Icon" href="/foo.html"/>`,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			node := parseLinkElement(t, tc.In)
			htmlrewrite.RewritePageLinks(node, tc.KeepHTMLExtension)
			if got, want := renderNode(t, node), tc.Out; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
