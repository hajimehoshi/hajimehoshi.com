// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package main

import (
	"context"
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

	options := &ssg.GenerateOptions{
		SiteName: "hajimehoshi.com",
		SiteURL:  "https://hajimehoshi.com",
	}
	if *httpAddr == "" {
		if err := ssg.Generate(options); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}

	if err := ssg.ServeSite(context.Background(), &ssg.ServeSiteOptions{
		Addr:            *httpAddr,
		GenerateOptions: *options,
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
