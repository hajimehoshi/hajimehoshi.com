// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

//go:build ignore

package main

import (
	"fmt"
	"os"

	"github.com/hajimehoshi/hajimehoshi.com/internal/gen"
)

func main() {
	if err := gen.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
