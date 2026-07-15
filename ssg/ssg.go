// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

package ssg

import "github.com/hajimehoshi/hajimehoshi.com/ssg/internal/gen"

func Run() error {
	if err := gen.Run(); err != nil {
		return err
	}
	return nil
}
