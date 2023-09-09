// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen_test

import (
	"bytes"
	"strings"
	"testing"

	"golang.org/x/net/html"

	"github.com/hajimehoshi/hajimehoshi.com/internal/gen"
)

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
			t.Errorf("got: %s, want: %s", got, want)
		}
	}
}

func TestInsertNodeBetweenWideAndNarrow(t *testing.T) {
	testCases := []struct {
		In  string
		Out string
	}{
		{
			In:  "<p>foo</p>",
			Out: "<p>foo</p>",
		},
		{
			In:  "<p>fooあ</p>",
			Out: "<p>foo<dummy-space></dummy-space>あ</p>",
		},
		{
			In:  "<p>foo あ</p>",
			Out: "<p>foo あ</p>",
		},
		{
			In:  "<p>foo<b>あ</b></p>",
			Out: "<p>foo<dummy-space></dummy-space><b>あ</b></p>",
		},
		{
			In:  "<p>foo<b>あ</b>bar</p>",
			Out: "<p>foo<dummy-space></dummy-space><b>あ<dummy-space></dummy-space></b>bar</p>",
		},
		{
			In:  "<p>foo<b>あ</b><b><i>bar</i></b></p>",
			Out: "<p>foo<dummy-space></dummy-space><b>あ<dummy-space></dummy-space></b><b><i>bar</i></b></p>",
		},
		{
			In:  "<p><b><i>foo</i></b><b>あ</b><b><i>bar</i></b></p>",
			Out: "<p><b><i>foo<dummy-space></dummy-space></i></b><b>あ<dummy-space></dummy-space></b><b><i>bar</i></b></p>",
		},
		{
			In:  "<ul><li>foo</li><li>あ</li></ul>",
			Out: "<ul><li>foo</li><li>あ</li></ul>",
		},
	}
	for _, tc := range testCases {
		nodes, err := html.ParseFragment(bytes.NewBufferString(tc.In), nil)
		if err != nil {
			t.Error(err)
			continue
		}

		node := nodes[0]
		gen.InsertNodeBetweenWideAndNarrow(node, &html.Node{
			Type: html.ElementNode,
			Data: "dummy-space",
		})

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
			t.Errorf("got: %s, want: %s", got, want)
		}
	}
}
