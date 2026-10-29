// Copyright 2026 Han Li and contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package signmsix

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// osslsigncode signs APPX and MSIX on any operating system. It is pinned, so
// that every run uses the same tool, and checked by its SHA-256 (the one of
// the zip that the release publishes). No Windows SDK (SignTool) is needed.
const (
	toolVersion = "2.14"
	toolSHA256  = "9a1722aaf62a27852c4eb9c35749a0248065052d0ae0a93d4ed6bb49def027f2"
	toolURL     = "https://github.com/mtrojnar/osslsigncode/releases/download/" + toolVersion + "/osslsigncode-" + toolVersion + "-windows-x64-mingw.zip"

	toolExecutable = "osslsigncode.exe"
	// maxDownload and maxUnpacked bound what a download may be, and what a file
	// in it may unpack to.
	maxDownload = 64 << 20
	maxUnpacked = 64 << 20
)

// source is where the tool is downloaded from, and the checksum that the
// download must have. The tests point it at a server of their own.
var source = struct{ url, sha256 string }{toolURL, toolSHA256}

// retryDelay is the time between the attempts of a download.
var retryDelay = 2 * time.Second

// ensureTool returns the osslsigncode to sign with: the one that the caller
// named, or the one below directory/tools (downloaded when it is not there, or
// when force says so).
func ensureTool(directory, named string, force bool) (string, error) {
	if named != "" {
		if _, err := os.Stat(named); err != nil {
			return "", fmt.Errorf("osslsigncode: %w", err)
		}
		return named, nil
	}
	tools := filepath.Join(directory, "tools")
	if !force {
		if found := findTool(tools); found != "" {
			fmt.Println("Using", found)
			return found, nil
		}
	}
	fmt.Printf("Downloading osslsigncode %s...\n", toolVersion)
	archive, err := download(source.url, source.sha256)
	if err != nil {
		return "", err
	}
	target := filepath.Join(tools, "osslsigncode")
	if err := os.RemoveAll(target); err != nil {
		return "", err
	}
	if err := unzip(archive, target); err != nil {
		return "", err
	}
	found := findTool(tools)
	if found == "" {
		return "", errors.New(toolExecutable + " not found after extracting")
	}
	fmt.Println("Using", found)
	return found, nil
}

// findTool looks for the executable below dir.
func findTool(dir string) string {
	found := ""
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && !entry.IsDir() && entry.Name() == toolExecutable {
			found = path
			return fs.SkipAll
		}
		return nil
	})
	return found
}

// download fetches url, tries again after a failed connection (the network may
// drop a TLS handshake) and checks the SHA-256 of what came.
func download(url, wantSum string) ([]byte, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	var data []byte
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		if data, err = fetch(client, url); err == nil {
			break
		}
		if attempt < 3 {
			fmt.Printf("Download failed (%v); trying again...\n", err)
			time.Sleep(retryDelay)
		}
	}
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); !strings.EqualFold(got, wantSum) {
		return nil, fmt.Errorf("osslsigncode checksum mismatch: got %s", got)
	}
	return data, nil
}

func fetch(client *http.Client, url string) ([]byte, error) {
	response, err := client.Get(url) //nolint:noctx // A download with a timeout of its own.
	if err != nil {
		return nil, err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, response.Status)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxDownload+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxDownload {
		return nil, fmt.Errorf("GET %s: more than %d bytes", url, maxDownload)
	}
	return data, nil
}

// unzip extracts a zip archive below dest. It refuses an entry that would
// land outside (a name with .. or a drive), a link, and a file that unpacks to
// more than it should.
func unzip(data []byte, dest string) error {
	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, file := range archive.File {
		name := filepath.FromSlash(file.Name)
		if !filepath.IsLocal(name) {
			return fmt.Errorf("the archive has an entry outside its directory: %q", file.Name)
		}
		target := filepath.Join(dest, name)
		switch {
		case file.FileInfo().IsDir():
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		case !file.Mode().IsRegular():
			return fmt.Errorf("the archive has an entry that is not a file: %q", file.Name)
		case file.UncompressedSize64 > maxUnpacked:
			return fmt.Errorf("the archive entry %q is too large", file.Name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		if err := unpack(file, target); err != nil {
			return err
		}
	}
	return nil
}

func unpack(file *zip.File, target string) error {
	in, err := file.Open()
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755) //nolint:gosec // The tool and its libraries must be executable.
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, io.LimitReader(in, maxUnpacked))
	if closeErr := out.Close(); copyErr == nil {
		copyErr = closeErr
	}
	return copyErr
}
