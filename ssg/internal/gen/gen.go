// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen

import (
	"bufio"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/evanw/esbuild/pkg/api"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/width"
	"gopkg.in/yaml.v3"
)

// asciiWhitespace is a set of ASCII whitespace characters defined by the HTML spec.
// https://infra.spec.whatwg.org/#ascii-whitespace
var asciiWhitespace = "\t\n\f\r "

func isASCIIWhitespace(r rune) bool {
	for _, w := range asciiWhitespace {
		if r == w {
			return true
		}
	}
	return false
}

func hasASCIIWhitespaceHead(str string) bool {
	return strings.TrimLeft(str, asciiWhitespace) != str
}

func hasASCIIWhitespaceTail(str string) bool {
	return strings.TrimRight(str, asciiWhitespace) != str
}

func hasASCIIWhitespaceWithNewLineHead(str string) bool {
	for _, r := range str {
		if r == '\n' {
			return true
		}
		if !isASCIIWhitespace(r) {
			return false
		}
	}
	return false
}

func hasASCIIWhitespaceWithNewLineTail(str string) bool {
	for {
		r, s := utf8.DecodeLastRuneInString(str)
		if r == '\n' {
			return true
		}
		if !isASCIIWhitespace(r) {
			return false
		}
		str = str[:len(str)-s]
	}
}

// Options is options for Run.
type Options struct {
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

func Run(options Options) error {
	const (
		outDir = "dist"
		inDir  = "contents"
	)

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

func generateHTMLs(outDir, inDir string, options Options) error {
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
			return fmt.Errorf("gen: no %s found for %s", templateFile, path)
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

func generateHTML(path string, tmpl *template.Template, outDir, inDir string, options Options) error {
	inPath := filepath.Join(inDir, path)
	outPath := filepath.Join(outDir, path)

	content, err := os.ReadFile(inPath)
	if err != nil {
		return err
	}

	meta, content, err := extractMetadataFromHTML(content)
	if err != nil {
		return fmt.Errorf("gen: extracting metadata in %s failed: %w", inPath, err)
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

	setMissingTitle(node, options.SiteName)

	if err := addFontPreloads(node, outDir, filepath.Dir(path)); err != nil {
		return err
	}

	if err := addResourceVersions(node, outDir, filepath.Dir(path)); err != nil {
		return err
	}

	rewritePageLinks(node, options.KeepHTMLExtension)

	removeHeadWhitespace(node)
	removeComments(node)
	removeInterElementWhitespace(node)
	processNewLines(node)

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

func getElement(node *html.Node, f func(*html.Node) bool) *html.Node {
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if n.Type != html.ElementNode {
			continue
		}
		if f(n) {
			return n
		}
		if n := getElement(n, f); n != nil {
			return n
		}
	}
	return nil
}

func getElementByName(node *html.Node, name string) *html.Node {
	return getElement(node, func(n *html.Node) bool {
		return n.Data == name
	})
}

// extractMetadataFromHTML parses the metadata data block at the beginning of
// an HTML content file and returns the metadata and the content with the data
// block removed. The data block is a <script type="application/yaml"> element
// preceded by nothing but whitespace, holding a YAML mapping with string keys.
// Each value keeps the type YAML resolves it to. A content file without a data
// block yields a nil map and unmodified content.
func extractMetadataFromHTML(content []byte) (map[string]any, []byte, error) {
	// metadataScriptType is the type attribute value that identifies a
	// metadata data block.
	const metadataScriptType = "application/yaml"

	z := html.NewTokenizer(bytes.NewReader(content))

	// offset tracks how many bytes of content the accepted tokens cover, so
	// that the remainder after the data block can be sliced off.
	var offset int

	// Find the data block's start tag, allowing only whitespace before it.
loop:
	for {
		switch z.Next() {
		case html.TextToken:
			if strings.Trim(string(z.Raw()), asciiWhitespace) != "" {
				return nil, content, nil
			}
			offset += len(z.Raw())
		case html.StartTagToken:
			break loop
		default:
			return nil, content, nil
		}
	}

	// TagName and TagAttr may overwrite the raw token buffer, so record the
	// start tag's length first.
	rawLen := len(z.Raw())
	name, hasAttr := z.TagName()
	if string(name) != "script" {
		return nil, content, nil
	}
	var isMetadata bool
	for hasAttr {
		var key, val []byte
		key, val, hasAttr = z.TagAttr()
		if string(key) == "type" && strings.EqualFold(string(val), metadataScriptType) {
			isMetadata = true
		}
	}
	if !isMetadata {
		return nil, content, nil
	}
	offset += rawLen

	var yamlSrc []byte
	for {
		switch z.Next() {
		case html.TextToken:
			yamlSrc = append(yamlSrc, z.Raw()...)
			offset += len(z.Raw())
		case html.EndTagToken:
			// In the raw text state, only </script> ends the element, so this
			// must be the data block's end tag.
			offset += len(z.Raw())
			meta := map[string]any{}
			if err := yaml.Unmarshal(yamlSrc, &meta); err != nil {
				return nil, nil, err
			}
			return meta, content[offset:], nil
		default:
			return nil, nil, fmt.Errorf("gen: metadata element is not closed")
		}
	}
}

// setMissingTitle sets the document's title to the first <h1> text combined
// with the site name when the document has no non-empty <title>. This is a
// safety net for pages whose template does not emit a title, e.g. because the
// page has no title metadata.
func setMissingTitle(node *html.Node, siteName string) {
	title := getElementByName(node, "title")
	if title != nil && title.FirstChild != nil && strings.Trim(title.FirstChild.Data, asciiWhitespace) != "" {
		return
	}

	text := siteName
	if h1 := getElementByName(node, "h1"); h1 != nil && h1.FirstChild != nil && h1.FirstChild.Type == html.TextNode {
		if t := strings.Trim(h1.FirstChild.Data, asciiWhitespace); t != "" {
			text = t + " – " + siteName
		}
	}

	if title == nil {
		head := getElementByName(node, "head")
		if head == nil {
			return
		}
		title = &html.Node{
			Type:     html.ElementNode,
			Data:     "title",
			DataAtom: atom.Title,
		}
		head.AppendChild(title)
	}
	for title.FirstChild != nil {
		title.RemoveChild(title.FirstChild)
	}
	title.AppendChild(&html.Node{
		Type: html.TextNode,
		Data: text,
	})
}

// elemAttr identifies an element attribute by the element and attribute names.
type elemAttr struct {
	elem string
	attr string
}

// resourceAttrs is the set of attributes whose value is a single URL of a
// resource the browser fetches, as opposed to a hyperlink (see hyperlinkAttrs).
// These are the attributes to version. srcset attributes (img, source) are
// excluded: their value is a URL list, whereas versionedURL handles only a
// single URL. link[href] is excluded: it is either kind depending on rel, and
// isResourceAttr classifies it.
var resourceAttrs = map[elemAttr]struct{}{
	{"img", "src"}:      {},
	{"script", "src"}:   {},
	{"audio", "src"}:    {},
	{"video", "src"}:    {},
	{"video", "poster"}: {},
	{"source", "src"}:   {},
	{"embed", "src"}:    {},
	{"track", "src"}:    {},
}

// hyperlinkAttrs is the set of attributes whose value can be a single URL of
// another page, as opposed to a resource the browser fetches (see
// resourceAttrs). cite attributes (blockquote, q, ins, del) are excluded: they
// reference a source document rather than link to one. link[href] is excluded:
// it is either kind depending on rel, and isHyperlinkAttr classifies it.
var hyperlinkAttrs = map[elemAttr]struct{}{
	{"a", "href"}:      {},
	{"area", "href"}:   {},
	{"iframe", "src"}:  {},
	{"form", "action"}: {},
	{"object", "data"}: {},
}

// resourceRels is the set of rel keywords denoting a <link> whose href is a
// resource the browser fetches. A <link> with any other rel is treated as a
// hyperlink, as versioning a page URL would hash an output file that the
// concurrent generation has possibly not written yet. prefetch and prerender
// are absent for that reason: their target is a page. A keyword whose href is
// an origin or an endpoint, such as preconnect or pingback, needs neither
// treatment and carries a host that both of them ignore.
var resourceRels = []string{
	"apple-touch-icon",
	"apple-touch-startup-image",
	"icon",
	"manifest",
	"mask-icon",
	"modulepreload",
	"preload",
	"stylesheet",
}

// isResourceLink reports whether the <link> element node points at a resource
// the browser fetches rather than at a page.
func isResourceLink(node *html.Node) bool {
	for _, a := range node.Attr {
		if a.Key != "rel" {
			continue
		}
		for t := range strings.FieldsSeq(a.Val) {
			if slices.ContainsFunc(resourceRels, func(r string) bool {
				return strings.EqualFold(t, r)
			}) {
				return true
			}
		}
	}
	return false
}

// isResourceAttr reports whether the node's attr holds a resource URL to
// version. A <link> is classified by its rel keywords, as the element name
// alone does not tell the two kinds apart.
func isResourceAttr(node *html.Node, attr string) bool {
	if node.Data == "link" && attr == "href" {
		return isResourceLink(node)
	}
	_, ok := resourceAttrs[elemAttr{node.Data, attr}]
	return ok
}

// isHyperlinkAttr reports whether the node's attr holds a URL that can point at
// another page. A <link> is classified by its rel keywords, as the element name
// alone does not tell the two kinds apart.
func isHyperlinkAttr(node *html.Node, attr string) bool {
	if node.Data == "link" && attr == "href" {
		return !isResourceLink(node)
	}
	_, ok := hyperlinkAttrs[elemAttr{node.Data, attr}]
	return ok
}

// pageHref returns href with its path adjusted to match the URL of the page it
// points at: a trailing index.html is dropped, and unless keepHTMLExtension is
// set, so is any other .html extension. An href that does not point at a local
// page is returned unchanged.
func pageHref(href string, keepHTMLExtension bool) string {
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	if u.Scheme != "" || u.Host != "" || u.Path == "" {
		return href
	}
	switch {
	case u.Path == "index.html":
		// An empty path would denote the current page rather than its
		// directory.
		u.Path = "./"
	case strings.HasSuffix(u.Path, "/index.html"):
		u.Path = strings.TrimSuffix(u.Path, "index.html")
	case !keepHTMLExtension && strings.HasSuffix(u.Path, ".html"):
		u.Path = strings.TrimSuffix(u.Path, ".html")
	default:
		return href
	}
	return u.String()
}

// rewritePageLinks adjusts every hyperlink to a local page so that it matches
// the URL that page is served at.
func rewritePageLinks(node *html.Node, keepHTMLExtension bool) {
	if node.Type == html.ElementNode {
		for i := range node.Attr {
			if !isHyperlinkAttr(node, node.Attr[i].Key) {
				continue
			}
			node.Attr[i].Val = pageHref(node.Attr[i].Val, keepHTMLExtension)
		}
	}
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		rewritePageLinks(n, keepHTMLExtension)
	}
}

// addResourceVersions appends a ?v=<hash> query to every local resource URL in
// the document so that updated files bypass stale caches. pageDir is the
// document's directory relative to the site root, used to resolve relative URLs.
func addResourceVersions(node *html.Node, outDir, pageDir string) error {
	if node.Type == html.ElementNode {
		for i := range node.Attr {
			if !isResourceAttr(node, node.Attr[i].Key) {
				continue
			}
			v, err := versionedURL(node.Attr[i].Val, outDir, pageDir)
			if err != nil {
				return err
			}
			node.Attr[i].Val = v
		}
	}
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if err := addResourceVersions(n, outDir, pageDir); err != nil {
			return err
		}
	}
	return nil
}

// localFilePath resolves rawURL to a file path under outDir. It reports false
// for URLs that do not point at a local file, such as external URLs (with a
// scheme or host, including protocol-relative ones). pageDir is the document's
// directory relative to the site root, used to resolve relative URLs.
func localFilePath(rawURL, outDir, pageDir string) (string, bool) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	if u.Scheme != "" || u.Host != "" || u.Path == "" {
		return "", false
	}
	if strings.HasPrefix(u.Path, "/") {
		// A leading slash denotes the site root; resolve under outDir.
		return filepath.Join(outDir, filepath.FromSlash(u.Path[1:])), true
	}
	return filepath.Join(outDir, pageDir, filepath.FromSlash(u.Path)), true
}

// versionedURL returns rawURL with a ?v=<hash> cache-busting query. URLs that
// do not point at a local file under outDir are returned unchanged.
func versionedURL(rawURL, outDir, pageDir string) (string, error) {
	file, ok := localFilePath(rawURL, outDir, pageDir)
	if !ok {
		return rawURL, nil
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL, nil
	}
	h, err := fileHash(file)
	if err != nil {
		return "", err
	}

	q := u.Query()
	q.Set("v", h)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// removeHeadWhitespace drops the formatting whitespace between the <head>'s
// child elements. The head holds only metadata elements, so inter-element
// whitespace is insignificant and would otherwise survive as stray spaces.
func removeHeadWhitespace(node *html.Node) {
	head := getElementByName(node, "head")
	if head == nil {
		return
	}
	var next *html.Node
	for n := head.FirstChild; n != nil; n = next {
		next = n.NextSibling
		if n.Type == html.TextNode && strings.Trim(n.Data, asciiWhitespace) == "" {
			head.RemoveChild(n)
		}
	}
}

func removeComments(node *html.Node) {
	var next *html.Node
	for n := node.FirstChild; n != nil; n = next {
		next = n.NextSibling

		if n.Type != html.CommentNode {
			removeComments(n)
			continue
		}

		prev := n.PrevSibling
		n.Parent.RemoveChild(n)

		// Merge two adjenct text nodes.
		if prev != nil && prev.Type == html.TextNode && next != nil && next.Type == html.TextNode {
			prev.Data += next.Data
			next2 := next.NextSibling
			next.Parent.RemoveChild(next)
			next = next2
		}
	}
}

func removeInterElementWhitespace(node *html.Node) {
	if node.Type == html.ElementNode {
		if isMetadataElementName(node.Data) {
			return
		}
		if node.Data == "pre" {
			return
		}
	}

	var next *html.Node
	for n := node.FirstChild; n != nil; n = next {
		next = n.NextSibling

		if n.Type != html.TextNode {
			removeInterElementWhitespace(n)
			continue
		}

		if strings.Trim(n.Data, asciiWhitespace) != "" {
			continue
		}

		// Replace a whitespace text with one space character.
		n.Data = " "

		if n.PrevSibling == nil && n.NextSibling == nil {
			continue
		}

		// If a node is in between phrasing elements, reserve this.
		if n.PrevSibling != nil && n.PrevSibling.Type == html.ElementNode && isPhrasingElementName(n.PrevSibling.Data) &&
			n.NextSibling != nil && n.NextSibling.Type == html.ElementNode && isPhrasingElementName(n.NextSibling.Data) {
			continue
		}

		n.Parent.RemoveChild(n)
	}
}

func processNewLines(node *html.Node) {
	if node.Type == html.ElementNode {
		if isMetadataElementName(node.Data) {
			return
		}
		if node.Data == "pre" {
			return
		}
	}

	reNewLineAndSpace := regexp.MustCompile(`[\t\n\f\r ]*\n[\t\n\f\r ]*`)
	reSpace := regexp.MustCompile(`[\t\n\f\r ]+`)

	// Insert dummy empty text nodes between two elements.
	var next *html.Node
	for n := node.FirstChild; n != nil; n = next {
		next = n.NextSibling

		if n.Type != html.ElementNode {
			continue
		}
		if n.NextSibling == nil {
			continue
		}
		if n.NextSibling.Type != html.ElementNode {
			continue
		}
		n.InsertBefore(&html.Node{
			Type: html.TextNode,
		}, n.NextSibling)
	}

	// Process child text nodes first.
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if n.Type != html.TextNode {
			continue
		}

		prev := prevVisibleTextNode(n)
		next := nextVisibleTextNode(n)

		var data string
		if len(n.Data) > 0 && (strings.Trim(n.Data, asciiWhitespace) != "" || !strings.Contains(n.Data, "\n")) {
			if prev != nil && (hasASCIIWhitespaceTail(prev.Data) || hasASCIIWhitespaceHead(n.Data)) {
				if shouldReserveSpaceBetweenTexts(prev.Data, n.Data) {
					data += " "
				}
			}
			for _, t := range reNewLineAndSpace.Split(n.Data, -1) {
				if len(data) > 0 && t != "" {
					r0, _ := utf8.DecodeLastRuneInString(data)
					r1, _ := utf8.DecodeRuneInString(t)
					if shouldReserveSpaceBetweenRunes(r0, r1) {
						data += " "
					}
				}
				data += t
			}
			if next != nil && (hasASCIIWhitespaceTail(n.Data) || hasASCIIWhitespaceHead(next.Data)) {
				if shouldReserveSpaceBetweenTexts(n.Data, next.Data) {
					data += " "
				}
			}
		} else if prev != nil && next != nil && (hasASCIIWhitespaceTail(prev.Data) || hasASCIIWhitespaceHead(next.Data) || len(n.Data) > 0) {
			if shouldReserveSpaceBetweenTexts(prev.Data, next.Data) {
				data += " "
			}
		}

		data = reSpace.ReplaceAllString(data, " ")

		n.Data = data
	}

	// Process child element nodes next.
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.TextNode {
			continue
		}
		processNewLines(n)
	}

	// Remove dummy empty text nodes.
	for n := node.FirstChild; n != nil; n = next {
		next = n.NextSibling

		if n.Type != html.TextNode {
			continue
		}
		if len(n.Data) > 0 {
			continue
		}
		n.Parent.RemoveChild(n)
	}
}

func nextVisibleTextNode(node *html.Node) *html.Node {
	for {
		node = nextVisibleNode(node)
		if node == nil {
			return nil
		}

		if node.Type == html.TextNode && len(node.Data) > 0 {
			return node
		}
	}
}

func prevVisibleTextNode(node *html.Node) *html.Node {
	for {
		node = prevVisibleNode(node)
		if node == nil {
			return nil
		}

		if node.Type == html.TextNode && len(node.Data) > 0 {
			return node
		}
	}
}

func nextVisibleNode(node *html.Node) *html.Node {
	if node.NextSibling == nil {
		return nil
	}
	node = node.NextSibling

	// Search the first visible descendant.
	for {
		if node.Type == html.ElementNode {
			if !isPhrasingElementName(node.Data) {
				return nil
			}
		}

		if node.FirstChild == nil {
			break
		}
		node = node.FirstChild
	}

	return node
}

func prevVisibleNode(node *html.Node) *html.Node {
	if node.PrevSibling == nil {
		return nil
	}
	node = node.PrevSibling

	// Search the last visible descendant.
	for {
		if node.Type == html.ElementNode {
			if !isPhrasingElementName(node.Data) {
				return nil
			}
		}

		if node.LastChild == nil {
			break
		}
		node = node.LastChild
	}

	return node
}

func shouldReserveSpaceBetweenRunes(r0, r1 rune) bool {
	if r0 == -1 || r1 == -1 {
		return false
	}
	k0 := width.LookupRune(r0).Kind()
	k1 := width.LookupRune(r1).Kind()
	w0 := k0 == width.EastAsianWide || k0 == width.EastAsianFullwidth
	w1 := k1 == width.EastAsianWide || k1 == width.EastAsianFullwidth
	return !w0 && !w1
}

func shouldReserveSpaceBetweenTexts(d0, d1 string) bool {
	if d0 == "" && d1 == "" {
		return false
	}

	if hasASCIIWhitespaceWithNewLineTail(d0) {
		d0 = strings.TrimRight(d0, asciiWhitespace)
	}
	if hasASCIIWhitespaceWithNewLineHead(d1) {
		d1 = strings.TrimLeft(d1, asciiWhitespace)
	}

	r0, _ := utf8.DecodeLastRuneInString(d0)
	r1, _ := utf8.DecodeRuneInString(d1)
	return shouldReserveSpaceBetweenRunes(r0, r1)
}

func isMetadataElementName(name string) bool {
	return slices.Contains([]string{"base", "link", "meta", "noscript", "script", "style", "template", "title"}, name)
}

func isPhrasingElementName(name string) bool {
	return slices.Contains([]string{"a", "abbr", "area", "audio", "b", "bdi", "bdo", "br", "button", "canvas", "cite", "code", "data", "datalist", "del", "dfn", "em", "embed", "i", "iframe", "img", "input", "ins", "kbd", "label", "link", "map", "mark", "math", "meta", "meter", "noscript", "object", "output", "picture", "progress", "q", "ruby", "s", "samp", "script", "select", "slot", "small", "span", "strong", "sub", "sup", "svg", "template", "textarea", "time", "u", "var", "video", "wbr"}, name)
}

// addFontPreloads appends a <link rel="preload"> element to the end of the
// document's <head> for every woff2 font used by the document's local
// stylesheets. pageDir is the document's directory relative to the site root,
// used to resolve relative stylesheet URLs.
func addFontPreloads(node *html.Node, outDir, pageDir string) error {
	head := getElementByName(node, "head")
	if head == nil {
		return nil
	}

	seen := map[string]struct{}{}
	for _, href := range stylesheetHrefs(node) {
		file, ok := localFilePath(href, outDir, pageDir)
		if !ok {
			continue
		}
		urls, err := woff2URLsInCSS(file)
		if err != nil {
			return err
		}
		for _, u := range urls {
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			head.AppendChild(&html.Node{
				Type:     html.ElementNode,
				Data:     "link",
				DataAtom: atom.Link,
				Attr: []html.Attribute{
					{Key: "rel", Val: "preload"},
					{Key: "href", Val: u},
					{Key: "as", Val: "font"},
					{Key: "crossorigin", Val: "anonymous"},
				},
			})
		}
	}
	return nil
}

// stylesheetHrefs returns the href values of the document's
// <link rel="stylesheet"> elements.
func stylesheetHrefs(node *html.Node) []string {
	var hrefs []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var rel, href string
			for _, a := range n.Attr {
				switch a.Key {
				case "rel":
					rel = a.Val
				case "href":
					href = a.Val
				}
			}
			if href != "" && slices.ContainsFunc(strings.Fields(rel), func(t string) bool {
				return strings.EqualFold(t, "stylesheet")
			}) {
				hrefs = append(hrefs, href)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(node)
	return hrefs
}

func woff2URLsInCSS(cssFile string) ([]string, error) {
	f, err := os.Open(cssFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	re := regexp.MustCompile(`url\((https://.+?woff2)\)`)

	var urls []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		for _, m := range re.FindAllStringSubmatch(s.Text(), -1) {
			urls = append(urls, m[1])
		}
	}
	if err := s.Err(); err != nil {
		return nil, err
	}
	return urls, nil
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
		return fmt.Errorf("gen: minifying CSS failed: %s", strings.Join(msgs, ", "))
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
		return fmt.Errorf("gen: minifying JS failed: %s", strings.Join(msgs, ", "))
	}
	if _, err := out.Write(bytes.TrimSpace(r.Code)); err != nil {
		return err
	}
	return nil
}
