// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package main

import (
	"log"
	"os"

	"github.com/hajimehoshi/hajimehoshi.com/internal/gen"
)

func main() {
	if err := gen.Run(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
