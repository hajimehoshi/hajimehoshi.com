// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package htmlrewrite

import (
	"net/url"

	"golang.org/x/net/html"
)

// AddResourceVersions appends a ?v=<hash> query to every local resource URL in
// the document so that updated files bypass stale caches. pageDir is the
// document's directory relative to the site root, used to resolve relative URLs.
func AddResourceVersions(node *html.Node, outDir, pageDir string) error {
	if node.Type == html.ElementNode {
		for i := range node.Attr {
			if !isResourceAttr(node, node.Attr[i].Key) {
				continue
			}
			v, err := versionedURL(node.Attr[i].Val, outDir, pageDir)
			if err != nil {
				return err
			}
			node.Attr[i].Val = v
		}
	}
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if err := AddResourceVersions(n, outDir, pageDir); err != nil {
			return err
		}
	}
	return nil
}

// versionedURL returns rawURL with a ?v=<hash> cache-busting query. URLs that
// do not point at a local file under outDir are returned unchanged.
func versionedURL(rawURL, outDir, pageDir string) (string, error) {
	file, ok := localFilePath(rawURL, outDir, pageDir)
	if !ok {
		return rawURL, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, nil
	}
	h, err := fileHash(file)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("v", h)
	u.RawQuery = q.Encode()
	return u.String(), nil
}
