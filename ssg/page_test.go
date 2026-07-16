// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package ssg_test

import (
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/hajimehoshi.com/ssg"
)

func TestPagePath(t *testing.T) {
	testCases := []struct {
		Name              string
		In                string
		KeepHTMLExtension bool
		Out               string
	}{
		{
			Name: "root index",
			In:   "index.html",
			Out:  "/",
		},
		{
			Name:              "root index, keeping the extension",
			In:                "index.html",
			KeepHTMLExtension: true,
			Out:               "/",
		},
		{
			Name: "root page",
			In:   "404.html",
			Out:  "/404",
		},
		{
			Name:              "root page, keeping the extension",
			In:                "404.html",
			KeepHTMLExtension: true,
			Out:               "/404.html",
		},
		{
			Name: "nested index",
			In:   "writings/index.html",
			Out:  "/writings/",
		},
		{
			Name:              "nested index, keeping the extension",
			In:                "writings/index.html",
			KeepHTMLExtension: true,
			Out:               "/writings/",
		},
		{
			Name: "nested page",
			In:   "writings/foo.html",
			Out:  "/writings/foo",
		},
		{
			Name:              "nested page, keeping the extension",
			In:                "writings/foo.html",
			KeepHTMLExtension: true,
			Out:               "/writings/foo.html",
		},
		{
			Name: "deeply nested index",
			In:   "ja/writings/index.html",
			Out:  "/ja/writings/",
		},
		{
			Name: "name ending with index.html",
			In:   "myindex.html",
			Out:  "/myindex",
		},
		{
			Name:              "name ending with index.html, keeping the extension",
			In:                "myindex.html",
			KeepHTMLExtension: true,
			Out:               "/myindex.html",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if got, want := ssg.PagePath(filepath.FromSlash(tc.In), tc.KeepHTMLExtension), tc.Out; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}

func TestPageURL(t *testing.T) {
	testCases := []struct {
		Name    string
		SiteURL string
		Path    string
		Out     string
	}{
		{
			Name:    "no site URL",
			SiteURL: "",
			Path:    "/writings/",
			Out:     "",
		},
		{
			Name:    "root",
			SiteURL: "https://example.com",
			Path:    "/",
			Out:     "https://example.com/",
		},
		{
			Name:    "site URL with a trailing slash",
			SiteURL: "https://example.com/",
			Path:    "/",
			Out:     "https://example.com/",
		},
		{
			Name:    "nested index",
			SiteURL: "https://example.com",
			Path:    "/writings/",
			Out:     "https://example.com/writings/",
		},
		{
			Name:    "nested page",
			SiteURL: "https://example.com/",
			Path:    "/writings/foo.html",
			Out:     "https://example.com/writings/foo.html",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if got, want := ssg.PageURL(tc.SiteURL, tc.Path), tc.Out; got != want {
				t.Errorf("got: %q, want: %q", got, want)
			}
		})
	}
}
