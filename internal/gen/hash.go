// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2023 Hajime Hoshi

package gen

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"os"
	"sync"
)

var (
	fileHashCache  map[string]string
	fileHashCacheM sync.Mutex
)

func fileHash(path string) (string, error) {
	fileHashCacheM.Lock()
	defer fileHashCacheM.Unlock()

	if h, ok := fileHashCache[path]; ok {
		return h, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, bufio.NewReader(f)); err != nil {
		return "", err
	}
	hash := base64.URLEncoding.EncodeToString(h.Sum(nil))[:10]

	if fileHashCache == nil {
		fileHashCache = map[string]string{}
	}
	fileHashCache[path] = hash

	return hash, nil
}
