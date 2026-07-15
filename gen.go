// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/hajimehoshi.com/ssg"
)

func main() {
	if err := ssg.Run(&ssg.RunOptions{
		SiteName: "hajimehoshi.com",
		SiteURL:  "https://hajimehoshi.com",
	}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
