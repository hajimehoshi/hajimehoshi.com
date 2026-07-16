// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

package ssg_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/hajimehoshi.com/ssg"
)

func newTestSite(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	for _, path := range []string{
		"index.html",
		"404.html",
		"writings/index.html",
		"writings/foo.html",
		"style.css",
	} {
		path = filepath.Join(dir, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(path), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestHandler(t *testing.T) {
	testCases := []struct {
		Name              string
		In                string
		KeepHTMLExtension bool
		Code              int
		Location          string
	}{
		{
			Name: "root",
			In:   "/",
			Code: http.StatusOK,
		},
		{
			Name: "page",
			In:   "/writings/foo",
			Code: http.StatusOK,
		},
		{
			Name:              "page, keeping the extension",
			In:                "/writings/foo",
			KeepHTMLExtension: true,
			Code:              http.StatusNotFound,
		},
		{
			Name:     "page with the extension",
			In:       "/writings/foo.html",
			Code:     http.StatusMovedPermanently,
			Location: "/writings/foo",
		},
		{
			Name:     "page with the extension and a query",
			In:       "/writings/foo.html?a=b",
			Code:     http.StatusMovedPermanently,
			Location: "/writings/foo?a=b",
		},
		{
			Name:              "page with the extension, keeping the extension",
			In:                "/writings/foo.html",
			KeepHTMLExtension: true,
			Code:              http.StatusOK,
		},
		{
			Name:     "index",
			In:       "/writings/index",
			Code:     http.StatusMovedPermanently,
			Location: "/writings/",
		},
		{
			Name:              "index, keeping the extension",
			In:                "/writings/index",
			KeepHTMLExtension: true,
			Code:              http.StatusNotFound,
		},
		{
			Name:     "index with the extension",
			In:       "/writings/index.html",
			Code:     http.StatusMovedPermanently,
			Location: "./",
		},
		{
			Name:              "index with the extension, keeping the extension",
			In:                "/writings/index.html",
			KeepHTMLExtension: true,
			Code:              http.StatusMovedPermanently,
			Location:          "./",
		},
		{
			Name:     "root index with the extension",
			In:       "/index.html",
			Code:     http.StatusMovedPermanently,
			Location: "./",
		},
		{
			Name: "non-page file",
			In:   "/style.css",
			Code: http.StatusOK,
		},
		{
			Name: "missing page",
			In:   "/writings/bar",
			Code: http.StatusNotFound,
		},
		{
			Name:     "missing page with the extension",
			In:       "/writings/bar.html",
			Code:     http.StatusMovedPermanently,
			Location: "/writings/bar",
		},
	}

	dir := newTestSite(t)
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			h := ssg.NewHandler(dir, tc.KeepHTMLExtension)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, tc.In, nil))

			if got, want := w.Code, tc.Code; got != want {
				t.Errorf("code: got: %d, want: %d", got, want)
			}
			if got, want := w.Header().Get("Location"), tc.Location; got != want {
				t.Errorf("location: got: %q, want: %q", got, want)
			}
		})
	}
}
