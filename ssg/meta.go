// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package ssg

import (
	"bytes"
	"fmt"
	"strings"

	"golang.org/x/net/html"
	"gopkg.in/yaml.v3"
)

// asciiWhitespace is a set of ASCII whitespace characters defined by the HTML spec.
// https://infra.spec.whatwg.org/#ascii-whitespace
var asciiWhitespace = "\t\n\f\r "

// extractMetadataFromHTML parses the metadata data block at the beginning of
// an HTML content file and returns the metadata and the content with the data
// block removed. The data block is a <script type="application/yaml"> element
// preceded by nothing but whitespace, holding a YAML mapping with string keys.
// Each value keeps the type YAML resolves it to. A content file without a data
// block yields a nil map and unmodified content.
func extractMetadataFromHTML(content []byte) (map[string]any, []byte, error) {
	// metadataScriptType is the type attribute value that identifies a
	// metadata data block.
	const metadataScriptType = "application/yaml"

	z := html.NewTokenizer(bytes.NewReader(content))

	// offset tracks how many bytes of content the accepted tokens cover, so
	// that the remainder after the data block can be sliced off.
	var offset int

	// Find the data block's start tag, allowing only whitespace before it.
loop:
	for {
		switch z.Next() {
		case html.TextToken:
			if strings.Trim(string(z.Raw()), asciiWhitespace) != "" {
				return nil, content, nil
			}
			offset += len(z.Raw())
		case html.StartTagToken:
			break loop
		default:
			return nil, content, nil
		}
	}

	// TagName and TagAttr may overwrite the raw token buffer, so record the
	// start tag's length first.
	rawLen := len(z.Raw())
	name, hasAttr := z.TagName()
	if string(name) != "script" {
		return nil, content, nil
	}
	var isMetadata bool
	for hasAttr {
		var key, val []byte
		key, val, hasAttr = z.TagAttr()
		if string(key) == "type" && strings.EqualFold(string(val), metadataScriptType) {
			isMetadata = true
		}
	}
	if !isMetadata {
		return nil, content, nil
	}
	offset += rawLen

	var yamlSrc []byte
	for {
		switch z.Next() {
		case html.TextToken:
			yamlSrc = append(yamlSrc, z.Raw()...)
			offset += len(z.Raw())
		case html.EndTagToken:
			// In the raw text state, only </script> ends the element, so this
			// must be the data block's end tag.
			offset += len(z.Raw())
			meta := map[string]any{}
			if err := yaml.Unmarshal(yamlSrc, &meta); err != nil {
				return nil, nil, err
			}
			return meta, content[offset:], nil
		default:
			return nil, nil, fmt.Errorf("ssg: metadata element is not closed")
		}
	}
}
