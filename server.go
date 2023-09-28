// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

//go:build ignore

package main

import (
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

var (
	httpAddr = flag.String("http", ":8000", "HTTP address")
)

var rootPath = ""

func init() {
	flag.Parse()
	dir := flag.Arg(0)
	if dir == "" {
		dir = "."
	}
	rootPath = dir
}

type handler struct{}

func (handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(rootPath, r.URL.Path[1:])
	f, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			notFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if f.IsDir() {
		path = filepath.Join(path, "index.html")
		if _, err := os.Stat(path); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				notFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	http.ServeFile(w, r, path)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)

	f, err := os.Open(filepath.Join(rootPath, "404.html"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	io.Copy(w, f)
}

func main() {
	http.Handle("/", handler{})
	log.Fatal(http.ListenAndServe(*httpAddr, nil))
}
