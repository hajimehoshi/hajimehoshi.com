// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package htmlrewrite_test

import (
	"bytes"
	"testing"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// parseLinkElement parses src, which must be a single <link> element, in the
// head context so that the element itself is returned rather than a document.
func parseLinkElement(t *testing.T, src string) *html.Node {
	t.Helper()
	nodes, err := html.ParseFragment(bytes.NewBufferString(src), &html.Node{
		Type:     html.ElementNode,
		Data:     "head",
		DataAtom: atom.Head,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1 (in: %q)", len(nodes), src)
	}
	return nodes[0]
}

func renderNode(t *testing.T, node *html.Node) string {
	t.Helper()
	var buf bytes.Buffer
	if err := html.Render(&buf, node); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
