// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen_test

import (
	"bytes"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/net/html"

	"github.com/hajimehoshi/hajimehoshi.com/ssg/internal/gen"
)

func TestExtractMetadataFromHTML(t *testing.T) {
	testCases := []struct {
		Name    string
		In      string
		Meta    map[string]any
		Content string
		Err     bool
	}{
		{
			Name:    "no metadata",
			In:      "<article>\n<h1>Title</h1>\n</article>\n",
			Meta:    nil,
			Content: "<article>\n<h1>Title</h1>\n</article>\n",
		},
		{
			Name:    "metadata",
			In:      "<script type=\"application/yaml\">\ntitle: Foo\ndescription: Bar baz\n</script>\n<article></article>\n",
			Meta:    map[string]any{"title": "Foo", "description": "Bar baz"},
			Content: "\n<article></article>\n",
		},
		{
			Name:    "leading whitespace",
			In:      "\n  \t<script type=\"application/yaml\">title: Foo</script><article></article>",
			Meta:    map[string]any{"title": "Foo"},
			Content: "<article></article>",
		},
		{
			Name:    "empty data block",
			In:      "<script type=\"application/yaml\"></script><article></article>",
			Meta:    map[string]any{},
			Content: "<article></article>",
		},
		{
			Name:    "extra attribute",
			In:      "<script id=\"meta\" type=\"application/yaml\">title: Foo</script><article></article>",
			Meta:    map[string]any{"title": "Foo"},
			Content: "<article></article>",
		},
		{
			Name:    "value with markup",
			In:      "<script type=\"application/yaml\">title: \"<b> & -->\"</script><article></article>",
			Meta:    map[string]any{"title": "<b> & -->"},
			Content: "<article></article>",
		},
		{
			Name: "non-string values",
			In:   "<script type=\"application/yaml\">\ndate: 2026-07-16\ndraft: true\nweight: 3\ntags: [go, ssg]\n</script><article></article>",
			Meta: map[string]any{
				"date":   time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC),
				"draft":  true,
				"weight": 3,
				"tags":   []any{"go", "ssg"},
			},
			Content: "<article></article>",
		},
		{
			Name:    "nested mapping",
			In:      "<script type=\"application/yaml\">\nauthor:\n  name: Hajime\n</script><article></article>",
			Meta:    map[string]any{"author": map[string]any{"name": "Hajime"}},
			Content: "<article></article>",
		},
		{
			Name:    "quoted date is a string",
			In:      "<script type=\"application/yaml\">date: \"2026-07-16\"</script><article></article>",
			Meta:    map[string]any{"date": "2026-07-16"},
			Content: "<article></article>",
		},
		{
			Name:    "non-metadata script",
			In:      "<script src=\"/script.js\"></script><article></article>",
			Meta:    nil,
			Content: "<script src=\"/script.js\"></script><article></article>",
		},
		{
			Name:    "metadata not at the beginning",
			In:      "<article></article><script type=\"application/yaml\">title: Foo</script>",
			Meta:    nil,
			Content: "<article></article><script type=\"application/yaml\">title: Foo</script>",
		},
		{
			Name:    "text before metadata",
			In:      "foo <script type=\"application/yaml\">title: Foo</script>",
			Meta:    nil,
			Content: "foo <script type=\"application/yaml\">title: Foo</script>",
		},
		{
			Name:    "comment before metadata",
			In:      "<!-- foo --><script type=\"application/yaml\">title: Foo</script>",
			Meta:    nil,
			Content: "<!-- foo --><script type=\"application/yaml\">title: Foo</script>",
		},
		{
			Name: "invalid YAML",
			In:   "<script type=\"application/yaml\">: : :</script><article></article>",
			Err:  true,
		},
		{
			Name: "non-mapping YAML",
			In:   "<script type=\"application/yaml\">just a scalar</script><article></article>",
			Err:  true,
		},
		{
			Name: "unclosed data block",
			In:   "<script type=\"application/yaml\">title: Foo",
			Err:  true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			meta, content, err := gen.ExtractMetadataFromHTML([]byte(tc.In))
			if tc.Err {
				if err == nil {
					t.Fatal("expected an error but got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(meta, tc.Meta) {
				t.Errorf("meta: got: %v, want: %v", meta, tc.Meta)
			}
			if got, want := string(content), tc.Content; got != want {
				t.Errorf("content: got: %q, want: %q", got, want)
			}
		})
	}
}

func TestPagePath(t *testing.T) {
	testCases := []struct {
		Name              string
		In                string
		KeepHTMLExtension bool
		Out               string
	}{
		{
			Name: "root index",
			In:   "index.html",
			Out:  "/",
		},
		{
			Name:              "root index, keeping the extension",
			In:                "index.html",
			KeepHTMLExtension: true,
			Out:               "/",
		},
		{
			Name: "root page",
			In:   "404.html",
			Out:  "/404",
		},
		{
			Name:              "root page, keeping the extension",
			In:                "404.html",
			KeepHTMLExtension: true,
			Out:               "/404.html",
		},
		{
			Name: "nested index",
			In:   "writings/index.html",
			Out:  "/writings/",
		},
		{
			Name:              "nested index, keeping the extension",
			In:                "writings/index.html",
			KeepHTMLExtension: true,
			Out:               "/writings/",
		},
		{
			Name: "nested page",
			In:   "writings/foo.html",
			Out:  "/writings/foo",
		},
		{
			Name:              "nested page, keeping the extension",
			In:                "writings/foo.html",
			KeepHTMLExtension: true,
			Out:               "/writings/foo.html",
		},
		{
			Name: "deeply nested index",
			In:   "ja/writings/index.html",
			Out:  "/ja/writings/",
		},
		{
			Name: "name ending with index.html",
			In:   "myindex.html",
			Out:  "/myindex",
		},
		{
			Name:              "name ending with index.html, keeping the extension",
			In:                "myindex.html",
			KeepHTMLExtension: true,
			Out:               "/myindex.html",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if got, want := gen.PagePath(filepath.FromSlash(tc.In), tc.KeepHTMLExtension), tc.Out; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestPageURL(t *testing.T) {
	testCases := []struct {
		Name    string
		SiteURL string
		Path    string
		Out     string
	}{
		{
			Name:    "no site URL",
			SiteURL: "",
			Path:    "/writings/",
			Out:     "",
		},
		{
			Name:    "root",
			SiteURL: "https://example.com",
			Path:    "/",
			Out:     "https://example.com/",
		},
		{
			Name:    "site URL with a trailing slash",
			SiteURL: "https://example.com/",
			Path:    "/",
			Out:     "https://example.com/",
		},
		{
			Name:    "nested index",
			SiteURL: "https://example.com",
			Path:    "/writings/",
			Out:     "https://example.com/writings/",
		},
		{
			Name:    "nested page",
			SiteURL: "https://example.com/",
			Path:    "/writings/foo.html",
			Out:     "https://example.com/writings/foo.html",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if got, want := gen.PageURL(tc.SiteURL, tc.Path), tc.Out; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

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
			if got, want := gen.PageHref(tc.In, tc.KeepHTMLExtension), tc.Out; got != want {
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
		gen.RewritePageLinks(node, tc.KeepHTMLExtension)

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

func TestRemoveInterElementWhitespace(t *testing.T) {
	testCases := []struct {
		In  string
		Out string
	}{
		{
			In:  "<p>foo</p>",
			Out: "<p>foo</p>",
		},
		{
			In:  "<p>foo </p>",
			Out: "<p>foo </p>",
		},
		{
			In:  "<p>foo<b>bar</b>baz</p>",
			Out: "<p>foo<b>bar</b>baz</p>",
		},
		{
			In:  "<p>foo <b>bar</b> baz</p>",
			Out: "<p>foo <b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo <b>  </b> baz</p>",
			Out: "<p>foo <b> </b> baz</p>",
		},
		{
			In:  "<p>  <b>  </b> baz</p>",
			Out: "<p><b> </b> baz</p>",
		},
		{
			In:  "<p>foo <b>  </b> </p>",
			Out: "<p>foo <b> </b></p>",
		},
		{
			In:  "<p><b> bar</b> <b>bar </b></p>",
			Out: "<p><b> bar</b> <b>bar </b></p>",
		},
		{
			In:  "<ul><li> bar</li> <li>bar </li></ul>",
			Out: "<ul><li> bar</li><li>bar </li></ul>",
		},
	}
	for _, tc := range testCases {
		nodes, err := html.ParseFragment(bytes.NewBufferString(tc.In), nil)
		if err != nil {
			t.Error(err)
			continue
		}

		node := nodes[0]
		gen.RemoveInterElementWhitespace(node)

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

func TestProcessNewLines(t *testing.T) {
	testCases := []struct {
		In  string
		Out string
	}{
		// Simple
		{
			In:  "<p>foo</p>",
			Out: "<p>foo</p>",
		},
		{
			In:  "<p>foo </p>",
			Out: "<p>foo </p>",
		},
		{
			In:  "<p>foo \n </p>",
			Out: "<p>foo</p>",
		},
		// 1 English node
		{
			In:  "<p>foo <b> bar </b> baz</p>",
			Out: "<p>foo <b> bar </b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n bar \n </b> baz</p>",
			Out: "<p>foo <b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo<b> \n bar \n </b>baz</p>",
			Out: "<p>foo <b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> bar </b> \n baz</p>",
			Out: "<p>foo <b> bar </b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n bar \n </b> \n baz</p>",
			Out: "<p>foo <b>bar</b> baz</p>",
		},
		{
			In:  "<p>a<a>a</a>a</p>",
			Out: "<p>a<a>a</a>a</p>",
		},
		{
			In:  "<p>(<a>a</a>)</p>",
			Out: "<p>(<a>a</a>)</p>",
		},
		// 2 English nodes
		{
			In:  "<p>foo <b> bar </b><b> bar </b> baz</p>",
			Out: "<p>foo <b> bar </b> <b> bar </b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n bar \n </b><b> \n bar \n </b> baz</p>",
			Out: "<p>foo <b>bar</b> <b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo<b> \n bar \n </b><b> \n bar \n </b>baz</p>",
			Out: "<p>foo <b>bar</b> <b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n bar</b> <b>bar \n </b> \n baz</p>",
			Out: "<p>foo <b>bar</b> <b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> bar </b><b> bar </b> \n baz</p>",
			Out: "<p>foo <b> bar </b> <b> bar </b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n bar \n </b><b> \n bar \n </b> \n baz</p>",
			Out: "<p>foo <b>bar</b> <b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n bar</b><b>bar \n </b> \n baz</p>",
			Out: "<p>foo <b>bar</b><b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n bar</b> <b>bar \n </b> \n baz</p>",
			Out: "<p>foo <b>bar</b> <b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n bar</b> \n <b>bar \n </b> \n baz</p>",
			Out: "<p>foo <b>bar</b> <b>bar</b> baz</p>",
		},
		// 1 Japanese node
		{
			In:  "<p>foo <b> あ </b> baz</p>",
			Out: "<p>foo <b> あ </b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n あ \n </b> baz</p>",
			Out: "<p>foo <b>あ</b> baz</p>",
		},
		{
			In:  "<p>foo<b> \n あ \n </b>baz</p>",
			Out: "<p>foo<b>あ</b>baz</p>",
		},
		{
			In:  "<p>foo \n <b> あ </b> \n baz</p>",
			Out: "<p>foo <b> あ </b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n あ \n </b> \n baz</p>",
			Out: "<p>foo<b>あ</b>baz</p>",
		},
		// 2 Japanese node
		{
			In:  "<p>foo <b> あ </b><b> い </b> baz</p>",
			Out: "<p>foo <b> あ </b> <b> い </b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n あ \n </b><b> \n い \n </b> baz</p>",
			Out: "<p>foo <b>あ</b><b>い</b> baz</p>",
		},
		{
			In:  "<p>foo<b> \n あ \n </b><b> \n い \n </b>baz</p>",
			Out: "<p>foo<b>あ</b><b>い</b>baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n あ</b> <b>い \n </b> \n baz</p>",
			Out: "<p>foo<b>あ</b> <b>い</b>baz</p>",
		},
		{
			In:  "<p>foo \n <b> あ </b><b> い </b> \n baz</p>",
			Out: "<p>foo <b> あ </b> <b> い </b> baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n あ \n </b><b> \n い \n </b> \n baz</p>",
			Out: "<p>foo<b>あ</b><b>い</b>baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n あ</b><b>い \n </b> \n baz</p>",
			Out: "<p>foo<b>あ</b><b>い</b>baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n あ</b> <b>い \n </b> \n baz</p>",
			Out: "<p>foo<b>あ</b> <b>い</b>baz</p>",
		},
		{
			In:  "<p>foo \n <b> \n あ</b> \n <b>い \n </b> \n baz</p>",
			Out: "<p>foo<b>あ</b><b>い</b>baz</p>",
		},
		// 1 Japanese and 1 English nodes
		{
			In:  "<p>foo <b> \n あ \n </b><b> \n bar \n </b> baz</p>",
			Out: "<p>foo <b>あ</b><b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n あ \n </b> \n <b> \n bar \n </b> baz</p>",
			Out: "<p>foo <b>あ</b><b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n あ</b><b>bar \n </b> baz</p>",
			Out: "<p>foo <b>あ</b><b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n あ</b> <b>bar \n </b> baz</p>",
			Out: "<p>foo <b>あ</b> <b>bar</b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n あ</b> \n <b>bar \n </b> baz</p>",
			Out: "<p>foo <b>あ</b><b>bar</b> baz</p>",
		},
		// 1 English and 1 Japanese nodes
		{
			In:  "<p>foo <b> \n bar \n </b><b> \n あ \n </b> baz</p>",
			Out: "<p>foo <b>bar</b><b>あ</b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n bar \n </b> \n <b> \n あ \n </b> baz</p>",
			Out: "<p>foo <b>bar</b><b>あ</b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n bar</b><b>あ \n </b> baz</p>",
			Out: "<p>foo <b>bar</b><b>あ</b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n bar</b> <b>あ \n </b> baz</p>",
			Out: "<p>foo <b>bar</b> <b>あ</b> baz</p>",
		},
		{
			In:  "<p>foo <b> \n bar</b> \n <b>あ \n </b> baz</p>",
			Out: "<p>foo <b>bar</b><b>あ</b> baz</p>",
		},
		// List
		{
			In:  "<ul><li> foo </li><li> bar </li></ul>",
			Out: "<ul><li> foo </li><li> bar </li></ul>",
		},
		{
			In:  "<ul><li> foo</li><li>bar </li></ul>",
			Out: "<ul><li> foo</li><li>bar </li></ul>",
		},
		{
			In:  "<ul><li> あ </li><li> い </li></ul>",
			Out: "<ul><li> あ </li><li> い </li></ul>",
		},
		{
			In:  "<ul><li> あ</li><li>い </li></ul>",
			Out: "<ul><li> あ</li><li>い </li></ul>",
		},
		{
			In:  "<ul><li> あ</li> <li>い </li></ul>",
			Out: "<ul><li> あ</li> <li>い </li></ul>",
		},
		{
			In:  "<ul><li> あ</li> \n <li>い </li></ul>",
			Out: "<ul><li> あ</li><li>い </li></ul>",
		},
	}
	for _, tc := range testCases {
		nodes, err := html.ParseFragment(bytes.NewBufferString(tc.In), nil)
		if err != nil {
			t.Error(err)
			continue
		}

		node := nodes[0]
		gen.ProcessNewLines(node)

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
