// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package ssg

import (
	"net/http"
)

func ExtractMetadataFromHTML(content []byte) (map[string]any, []byte, error) {
	return extractMetadataFromHTML(content)
}

func NewHandler(rootPath string, keepHTMLExtension bool) http.Handler {
	return handler{
		rootPath:          rootPath,
		keepHTMLExtension: keepHTMLExtension,
	}
}

func PagePath(relPath string, keepHTMLExtension bool) string {
	return pagePath(relPath, keepHTMLExtension)
}

func PageURL(siteURL, path string) string {
	return pageURL(siteURL, path)
}
