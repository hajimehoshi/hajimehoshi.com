// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package ssg

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/sync/errgroup"

	"github.com/hajimehoshi/hajimehoshi.com/ssg/internal/htmlrewrite"
)

func generateHTMLs(outDir, inDir string, options *GenerateOptions) error {
	// templateFile is the base name of each directory's HTML template. Its
	// leading underscore keeps it out of the generated site (see isIgnoredFile).
	const templateFile = "_tmpl.html"

	// templates maps a directory to the template it defines. Building it in the
	// walk (which is single-threaded) lets the concurrent generation below read
	// it without locking.
	templates := map[string]*template.Template{}
	var contentPaths []string

	if err := filepath.Walk(inDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".html" {
			return nil
		}
		if filepath.Base(path) == templateFile {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			tmpl, err := template.New(templateFile).Parse(string(data))
			if err != nil {
				return err
			}
			templates[filepath.Dir(path)] = tmpl
			return nil
		}
		if isIgnoredFile(path) {
			return nil
		}
		rel, err := filepath.Rel(inDir, path)
		if err != nil {
			return err
		}
		contentPaths = append(contentPaths, rel)
		return nil
	}); err != nil {
		return err
	}

	var wg errgroup.Group
	for _, path := range contentPaths {
		tmpl := closestTemplate(templates, inDir, filepath.Dir(filepath.Join(inDir, path)))
		if tmpl == nil {
			return fmt.Errorf("ssg: no %s found for %s", templateFile, path)
		}
		wg.Go(func() error {
			return generateHTML(path, tmpl, outDir, inDir, options)
		})
	}
	return wg.Wait()
}

// closestTemplate returns the template defined in dir or its nearest ancestor up
// to inDir, or nil if none of them define one.
func closestTemplate(templates map[string]*template.Template, inDir, dir string) *template.Template {
	for {
		if tmpl, ok := templates[dir]; ok {
			return tmpl
		}
		if dir == inDir {
			return nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

// siteData is the site-wide data available to templates as .Site.
type siteData struct {
	Name string
	URL  string
}

// pageData is the per-page data available to templates as .Page.
type pageData struct {
	Lang    string
	Path    string
	URL     string
	Meta    map[string]any
	Content template.HTML
}

// pagePath returns the site-root-absolute path of the content file at relPath,
// which is relative to the content root. A trailing index.html is dropped so
// that the path denotes the directory the browser requests; any other .html
// extension is dropped unless keepHTMLExtension is set.
func pagePath(relPath string, keepHTMLExtension bool) string {
	p := "/" + filepath.ToSlash(relPath)
	if strings.HasSuffix(p, "/index.html") {
		return strings.TrimSuffix(p, "index.html")
	}
	if !keepHTMLExtension {
		p = strings.TrimSuffix(p, ".html")
	}
	return p
}

// pageURL returns the absolute URL of the page at the site-root-absolute path,
// or an empty string when siteURL is empty.
func pageURL(siteURL, path string) string {
	if siteURL == "" {
		return ""
	}
	return strings.TrimSuffix(siteURL, "/") + path
}

func generateHTML(path string, tmpl *template.Template, outDir, inDir string, options *GenerateOptions) error {
	inPath := filepath.Join(inDir, path)
	outPath := filepath.Join(outDir, path)

	content, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}

	meta, content, err := extractMetadataFromHTML(content)
	if err != nil {
		return fmt.Errorf("ssg: extracting metadata in %s failed: %w", inPath, err)
	}

	lang := "en"
	if dir := filepath.Dir(path); dir != "." {
		if ts := strings.Split(dir, string(filepath.Separator)); len(ts) > 0 {
			lang = ts[0]
		}
	}

	urlPath := pagePath(path, options.KeepHTMLExtension)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct {
		Site siteData
		Page pageData
	}{
		Site: siteData{
			Name: options.SiteName,
			URL:  options.SiteURL,
		},
		Page: pageData{
			Lang:    lang,
			Path:    urlPath,
			URL:     pageURL(options.SiteURL, urlPath),
			Meta:    meta,
			Content: template.HTML(content),
		},
	}); err != nil {
		return err
	}

	node, err := html.Parse(&buf)
	if err != nil {
		return err
	}

	htmlrewrite.SetMissingTitle(node, options.SiteName)

	if err := htmlrewrite.AddFontPreloads(node, outDir, filepath.Dir(path)); err != nil {
		return err
	}

	if err := htmlrewrite.AddResourceVersions(node, outDir, filepath.Dir(path)); err != nil {
		return err
	}

	htmlrewrite.RewritePageLinks(node, options.KeepHTMLExtension)

	htmlrewrite.Minify(node)

	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}

	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	w := bufio.NewWriter(out)
	if err := html.Render(w, node); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}
