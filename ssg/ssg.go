// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

package ssg

import (
	"fmt"

	"github.com/hajimehoshi/hajimehoshi.com/ssg/internal/gen"
)

// RunOptions is options for Run.
type RunOptions struct {
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

func Run(options *RunOptions) error {
	if options == nil || options.SiteName == "" {
		return fmt.Errorf("ssg: SiteName must not be empty")
	}
	if err := gen.Run(gen.Options{
		SiteName:          options.SiteName,
		SiteURL:           options.SiteURL,
		KeepHTMLExtension: options.KeepHTMLExtension,
	}); err != nil {
		return err
	}
	return nil
}
