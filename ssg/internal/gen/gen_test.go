// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen_test

import (
	"bytes"
	"maps"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/hajimehoshi/hajimehoshi.com/ssg/internal/gen"
)

func TestExtractMetadataFromHTML(t *testing.T) {
	testCases := []struct {
		Name    string
		In      string
		Meta    map[string]string
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
			Meta:    map[string]string{"title": "Foo", "description": "Bar baz"},
			Content: "\n<article></article>\n",
		},
		{
			Name:    "leading whitespace",
			In:      "\n  \t<script type=\"application/yaml\">title: Foo</script><article></article>",
			Meta:    map[string]string{"title": "Foo"},
			Content: "<article></article>",
		},
		{
			Name:    "empty data block",
			In:      "<script type=\"application/yaml\"></script><article></article>",
			Meta:    map[string]string{},
			Content: "<article></article>",
		},
		{
			Name:    "extra attribute",
			In:      "<script id=\"meta\" type=\"application/yaml\">title: Foo</script><article></article>",
			Meta:    map[string]string{"title": "Foo"},
			Content: "<article></article>",
		},
		{
			Name:    "value with markup",
			In:      "<script type=\"application/yaml\">title: \"<b> & -->\"</script><article></article>",
			Meta:    map[string]string{"title": "<b> & -->"},
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
			if !maps.Equal(meta, tc.Meta) {
				t.Errorf("meta: got: %v, want: %v", meta, tc.Meta)
			}
			if got, want := string(content), tc.Content; got != want {
				t.Errorf("content: got: %q, want: %q", got, want)
			}
		})
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
