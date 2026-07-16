// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package ssg

func ExtractMetadataFromHTML(content []byte) (map[string]any, []byte, error) {
	return extractMetadataFromHTML(content)
}

func PagePath(relPath string, keepHTMLExtension bool) string {
	return pagePath(relPath, keepHTMLExtension)
}

func PageURL(siteURL, path string) string {
	return pageURL(siteURL, path)
}
