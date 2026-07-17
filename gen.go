// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hajimehoshi/ssg"
)

var (
	httpAddr = flag.String("http", "", "HTTP address to serve the generated website at (e.g. :8000)")
)

func main() {
	flag.Parse()

	if err := ssg.Generate(&ssg.GenerateOptions{
		SiteName: "hajimehoshi.com",
		SiteURL:  "https://hajimehoshi.com",
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *httpAddr == "" {
		return
	}

	if err := ssg.ServeSite(&ssg.ServeSiteOptions{
		Addr: *httpAddr,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
