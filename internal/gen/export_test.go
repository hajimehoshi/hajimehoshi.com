// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen

import (
	"golang.org/x/net/html"
)

func RemoveInterElementWhitespace(node *html.Node) {
	removeInterElementWhitespace(node)
}

func ProcessNewLines(node *html.Node) {
	processNewLines(node)
}

func InsertNodeBetweenWideAndNarrow(node *html.Node, insertingNode *html.Node) {
	insertNodeBetweenWideAndNarrow(node, insertingNode)
}
