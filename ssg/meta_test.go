// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package ssg_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/hajimehoshi/hajimehoshi.com/ssg"
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
			meta, content, err := ssg.ExtractMetadataFromHTML([]byte(tc.In))
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
