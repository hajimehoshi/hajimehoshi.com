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
			Out: "<p>foo<dummy-space></dummy-space><b>あ</b><dummy-space></dummy-space>bar</p>",
		},
		{
			In:  "<p>foo<b>あ</b><b><i>bar</i></b></p>",
			Out: "<p>foo<dummy-space></dummy-space><b>あ</b><dummy-space></dummy-space><b><i>bar</i></b></p>",
		},
		{
			In:  "<p><b><i>foo</i></b><b>あ</b><b><i>bar</i></b></p>",
			Out: "<p><b><i>foo</i></b><dummy-space></dummy-space><b>あ</b><dummy-space></dummy-space><b><i>bar</i></b></p>",
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
			t.Errorf("got: %q, want: %q (in: %q)", got, want, tc.In)
		}
	}
}
