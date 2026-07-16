// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package ssg

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/sync/errgroup"
)

func copyNonHTMLFiles(outDir, inDir string) error {
	var wg errgroup.Group
	if err := filepath.Walk(inDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".html" {
			return nil
		}
		if isIgnoredFile(path) {
			return nil
		}
		wg.Go(func() error {
			inRelPath, err := filepath.Rel(inDir, path)
			if err != nil {
				return err
			}
			outPath := filepath.Join(outDir, inRelPath)
			if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
				return err
			}

			out, err := os.Create(outPath)
			if err != nil {
				return err
			}
			defer out.Close()

			in, err := os.Open(path)
			if err != nil {
				return err
			}
			defer in.Close()

			switch filepath.Ext(path) {
			case ".css":
				outbuf := bufio.NewWriter(out)
				if err := minifyCSS(outbuf, bufio.NewReader(in)); err != nil {
					return err
				}
				if err := outbuf.Flush(); err != nil {
					return err
				}
			case ".js":
				outbuf := bufio.NewWriter(out)
				if err := minifyJS(outbuf, bufio.NewReader(in)); err != nil {
					return err
				}
				if err := outbuf.Flush(); err != nil {
					return err
				}
			default:
				if _, err := io.Copy(out, in); err != nil {
					return err
				}
			}
			return nil
		})
		return nil
	}); err != nil {
		return err
	}
	if err := wg.Wait(); err != nil {
		return err
	}
	return nil
}

func minifyCSS(out io.Writer, in io.Reader) error {
	css, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	r := api.Transform(string(css), api.TransformOptions{
		Loader:            api.LoaderCSS,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})
	if len(r.Errors) > 0 {
		var msgs []string
		for _, e := range r.Errors {
			msgs = append(msgs, e.Text)
		}
		return fmt.Errorf("ssg: minifying CSS failed: %s", strings.Join(msgs, ", "))
	}
	if _, err := out.Write(bytes.TrimSpace(r.Code)); err != nil {
		return err
	}
	return nil
}

func minifyJS(out io.Writer, in io.Reader) error {
	js, err := io.ReadAll(in)
	if err != nil {
		return err
	}
	r := api.Transform(string(js), api.TransformOptions{
		Loader:            api.LoaderJS,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
	})
	if len(r.Errors) > 0 {
		var msgs []string
		for _, e := range r.Errors {
			msgs = append(msgs, e.Text)
		}
		return fmt.Errorf("ssg: minifying JS failed: %s", strings.Join(msgs, ", "))
	}
	if _, err := out.Write(bytes.TrimSpace(r.Code)); err != nil {
		return err
	}
	return nil
}
