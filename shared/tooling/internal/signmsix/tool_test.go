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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
)

// zipOf makes a zip archive of the files (name to content).
func zipOf(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	for name, content := range files {
		file, err := archive.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func checksum(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// serveTool serves data as the release, counts the requests, and points the
// download at it. The first drops connections of dropFirst requests.
func serveTool(t *testing.T, data []byte, dropFirst int32) *atomic.Int32 {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := requests.Add(1)
		if count <= dropFirst {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				_ = conn.Close() // The way this network drops a handshake.
			}
			return
		}
		if r.URL.Path != "/release.zip" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(data)
	}))
	t.Cleanup(server.Close)
	previousSource, previousDelay := source, retryDelay
	source.url, source.sha256, retryDelay = server.URL+"/release.zip", checksum(data), 0
	t.Cleanup(func() { source, retryDelay = previousSource, previousDelay })
	return &requests
}

// release has the layout of the zip of osslsigncode: bin/ with the executable
// and a library next to it.
func release(t *testing.T) []byte {
	return zipOf(t, map[string]string{
		"bin/osslsigncode.exe":                  "MZ the tool",
		"bin/libcrypto-3-x64.dll":               "MZ a library",
		"share/doc/osslsigncode/README.md":      "docs",
		"share/doc/osslsigncode/not-a-file.txt": "docs",
	})
}

func TestEnsureToolDownloadsOnceAndForce(t *testing.T) {
	requests := serveTool(t, release(t), 0)
	directory := t.TempDir()

	tool, err := ensureTool(directory, "", false)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(directory, "tools", "osslsigncode", "bin", "osslsigncode.exe")
	if tool != want {
		t.Errorf("got %s, want %s", tool, want)
	}
	// The libraries are next to the executable, which the tool needs.
	if data, err := os.ReadFile(filepath.Join(filepath.Dir(tool), "libcrypto-3-x64.dll")); err != nil || string(data) != "MZ a library" {
		t.Errorf("the library: %q, %v", data, err)
	}
	if runtime.GOOS != "windows" {
		if info, err := os.Stat(tool); err != nil || info.Mode()&0o111 == 0 {
			t.Errorf("the tool is not executable: %v", err)
		}
	}

	if again, err := ensureTool(directory, "", false); err != nil || again != tool || requests.Load() != 1 {
		t.Errorf("a second call: %s, %v, %d requests", again, err, requests.Load())
	}

	// -force downloads again and starts from nothing.
	stray := filepath.Join(directory, "tools", "osslsigncode", "stray.txt")
	if err := os.WriteFile(stray, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureTool(directory, "", true); err != nil || requests.Load() != 2 {
		t.Errorf("force: %v, %d requests", err, requests.Load())
	}
	if _, err := os.Stat(stray); !os.IsNotExist(err) {
		t.Errorf("an old file is left: %v", err)
	}
}

func TestEnsureToolUsesTheOneThatWasNamed(t *testing.T) {
	requests := serveTool(t, release(t), 0)
	named := filepath.Join(t.TempDir(), "osslsigncode")
	if err := os.WriteFile(named, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if got, err := ensureTool(t.TempDir(), named, false); err != nil || got != named || requests.Load() != 0 {
		t.Errorf("got %s, %v, %d requests", got, err, requests.Load())
	}
	if _, err := ensureTool(t.TempDir(), filepath.Join(t.TempDir(), "missing"), false); err == nil {
		t.Error("a tool that is not there was accepted")
	}
}

func TestEnsureToolRefusesAWrongChecksum(t *testing.T) {
	serveTool(t, release(t), 0)
	source.sha256 = strings.Repeat("0", 64)
	directory := t.TempDir()
	_, err := ensureTool(directory, "", false)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("got %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "tools")); !os.IsNotExist(err) {
		t.Errorf("something was unpacked: %v", err)
	}
}

func TestEnsureToolRetriesADroppedConnection(t *testing.T) {
	requests := serveTool(t, release(t), 2)
	if _, err := ensureTool(t.TempDir(), "", false); err != nil || requests.Load() != 3 {
		t.Fatalf("%v after %d requests", err, requests.Load())
	}
	// Not forever.
	requests = serveTool(t, release(t), 100)
	if _, err := ensureTool(t.TempDir(), "", false); err == nil || requests.Load() != 3 {
		t.Fatalf("%v after %d requests", err, requests.Load())
	}
}

func TestEnsureToolFailures(t *testing.T) {
	data := release(t)
	requests := serveTool(t, data, 0)
	source.url += "/missing"
	if _, err := ensureTool(t.TempDir(), "", false); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("a missing release: %v", err)
	}
	if requests.Load() != 3 {
		t.Errorf("%d requests", requests.Load())
	}

	// The release has no executable.
	empty := zipOf(t, map[string]string{"README.md": "x"})
	serveTool(t, empty, 0)
	if _, err := ensureTool(t.TempDir(), "", false); err == nil || !strings.Contains(err.Error(), "not found after extracting") {
		t.Errorf("no executable: %v", err)
	}

	// Not a zip at all, with the right checksum.
	serveTool(t, []byte("not a zip"), 0)
	if _, err := ensureTool(t.TempDir(), "", false); err == nil {
		t.Error("a file that is not a zip was accepted")
	}
}

func TestUnzipStaysInsideItsDirectory(t *testing.T) {
	for _, name := range []string{"../evil.txt", "bin/../../evil.txt", "/evil.txt"} {
		directory := t.TempDir()
		target := filepath.Join(directory, "tools")
		err := unzip(zipOf(t, map[string]string{name: "x"}), target)
		if err == nil || !strings.Contains(err.Error(), "outside") {
			t.Errorf("%q: got %v", name, err)
		}
		if _, err := os.Stat(filepath.Join(directory, "evil.txt")); !os.IsNotExist(err) {
			t.Errorf("%q: a file was written outside", name)
		}
	}
}

func TestUnzipRefusesLinksAndBombs(t *testing.T) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	header := &zip.FileHeader{Name: "link"}
	header.SetMode(os.ModeSymlink | 0o777)
	file, err := archive.CreateHeader(header)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte("/etc/passwd"))
	_ = archive.Close()
	if err := unzip(buffer.Bytes(), t.TempDir()); err == nil || !strings.Contains(err.Error(), "not a file") {
		t.Errorf("a link: %v", err)
	}

	// Zeros pack to almost nothing and unpack to more than the limit.
	buffer.Reset()
	archive = zip.NewWriter(&buffer)
	file, err = archive.CreateHeader(&zip.FileHeader{Name: "bomb.bin", Method: zip.Deflate})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(make([]byte, maxUnpacked+1)); err != nil {
		t.Fatal(err)
	}
	_ = archive.Close()
	if buffer.Len() > 1<<20 {
		t.Fatalf("the bomb is %d bytes", buffer.Len())
	}
	directory := t.TempDir()
	if err := unzip(buffer.Bytes(), directory); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("a bomb: %v", err)
	}
	if _, err := os.Stat(filepath.Join(directory, "bomb.bin")); !os.IsNotExist(err) {
		t.Errorf("the bomb was unpacked: %v", err)
	}
}
