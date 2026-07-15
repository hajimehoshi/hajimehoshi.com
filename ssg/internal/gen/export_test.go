// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen

import (
	"golang.org/x/net/html"
)

func ExtractMetadataFromHTML(content []byte) (map[string]string, []byte, error) {
	return extractMetadataFromHTML(content)
}

func RemoveInterElementWhitespace(node *html.Node) {
	removeInterElementWhitespace(node)
}

func ProcessNewLines(node *html.Node) {
	processNewLines(node)
}
