// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

// Package ssg generates a static website from a directory of contents.
package ssg

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	outDir = "dist"
	inDir  = "contents"
)

// GenerateOptions is options for Generate.
type GenerateOptions struct {
	// SiteName is the name of the website, used e.g. in page titles.
	SiteName string

	// SiteURL is the absolute URL of the website root, used when a page
	// needs an absolute URL. This can be empty.
	SiteURL string

	// KeepHTMLExtension keeps the .html extension in page URLs. By default a
	// page URL omits it, which requires the server to resolve an extensionless
	// URL to its .html file.
	KeepHTMLExtension bool
}

func Generate(options *GenerateOptions) error {
	if options == nil || options.SiteName == "" {
		return fmt.Errorf("ssg: SiteName must not be empty")
	}

	if err := os.RemoveAll(outDir); err != nil {
		return err
	}
	if err := copyNonHTMLFiles(outDir, inDir); err != nil {
		return err
	}
	if err := generateHTMLs(outDir, inDir, options); err != nil {
		return err
	}
	return nil
}

func isIgnoredFile(path string) bool {
	if strings.HasPrefix(filepath.Base(path), "#") {
		return true
	}
	if strings.HasPrefix(filepath.Base(path), "_") {
		return true
	}
	if strings.HasSuffix(path, "~") {
		return true
	}
	return false
}
