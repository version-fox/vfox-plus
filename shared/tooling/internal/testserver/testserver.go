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

// Package testserver is a local HTTP server for the offline test plugin of
// shared/test-plugin, following the protocol of shared/test-plugin/README.md:
// a plugin registry, the plugin and two SDK versions. SDK archives are sent
// slowly, in chunks, so that vfox reports download progress. Holding the
// downloads (/control/hold) makes them stop after their first chunk until
// /control/release.
//
// It replaces shared/e2e/test-server.mjs with the same behavior: the same
// endpoints, the same JSON shapes, byte-identical SDK archives and the same
// streaming discipline. Unlike the script it runs in-process: Start returns
// the base URL directly instead of printing it for a parent to parse.
package testserver

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path"
	"regexp"
	"sync"
	"time"
)

// testVersions are the SDK versions the server offers, oldest first. The
// payload seed derives from the position, like the download script.
var testVersions = []string{"1.0.0", "2.0.0"}

// chunkSize and chunkDelay mirror the 64 KiB chunks and the 60 ms pacing of
// the original server: slow enough for vfox to report download progress.
const (
	chunkSize  = 64 * 1024
	chunkDelay = 60 * time.Millisecond
)

// fixedTime keeps every generated archive byte-identical between runs.
var fixedTime = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

// padding is deterministic, incompressible filler so that the downloads are
// long enough to report progress: a 32-bit linear congruential generator,
// the same sequence the Node implementation emits.
func padding(seed uint32, size int) []byte {
	out := make([]byte, size)
	state := seed
	for i := range out {
		state = state*1103515245 + 12345
		out[i] = byte(state >> 24)
	}
	return out
}

// buildArchive packs one SDK version the way the download script does: a
// testsdk-<version>/ tree with an executable bin/testsdk that prints the
// version and a 512 KiB payload of padding, with fixed timestamps.
func buildArchive(version string, seed uint32) ([]byte, error) {
	root := "testsdk-" + version
	script := []byte("#!/bin/sh\necho \"testsdk " + version + "\"\n")
	payload := padding(seed, 512*1024)
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(writer)
	entries := []struct {
		name string
		body []byte
		mode int64
		dir  bool
	}{
		{root + "/", nil, 0o755, true},
		{root + "/bin/", nil, 0o755, true},
		{root + "/bin/testsdk", script, 0o755, false},
		{root + "/payload.bin", payload, 0o644, false},
	}
	for _, entry := range entries {
		header := &tar.Header{
			Name:    entry.name,
			Mode:    entry.mode,
			Uid:     0,
			Gid:     0,
			Uname:   "root",
			Gname:   "root",
			ModTime: fixedTime,
			Format:  tar.FormatUSTAR,
		}
		if entry.dir {
			header.Typeflag = tar.TypeDir
		} else {
			header.Typeflag = tar.TypeReg
			header.Size = int64(len(entry.body))
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, err
		}
		if !entry.dir {
			if _, err := tarWriter.Write(entry.body); err != nil {
				return nil, err
			}
		}
	}
	if err := tarWriter.Close(); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// registryEntry is one row of /registry/index.json, with fields in wire order.
type registryEntry struct {
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Homepage string `json:"homepage"`
}

// manifest is /registry/testsdk.json, with fields in wire order.
type manifest struct {
	Name              string `json:"name"`
	Version           string `json:"version"`
	License           string `json:"license"`
	Author            string `json:"author"`
	DownloadURL       string `json:"downloadUrl"`
	MinRuntimeVersion string `json:"minRuntimeVersion"`
}

var sdkPath = regexp.MustCompile(`^/sdk/testsdk-([0-9.]+)\.tar\.gz$`)

// Server is a running offline test server. Use Start to run one.
type Server struct {
	listener net.Listener
	server   *http.Server
	plugin   []byte
	archives map[string][]byte

	mu      sync.Mutex
	held    bool
	release chan struct{}
}

// baseURL is the URL the server listens on, like the first line the script
// used to print.
func (s *Server) baseURL() string {
	return "http://" + s.listener.Addr().String()
}

// hold stops downloads after their first chunk until release.
func (s *Server) hold() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.held {
		s.held = true
		s.release = make(chan struct{})
	}
}

// unhold resumes held downloads.
func (s *Server) unhold() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.held {
		s.held = false
		close(s.release)
		s.release = nil
	}
}

// awaitRelease reports whether downloads are currently held; while held it
// blocks until release or the request ends. It mirrors the `held?.promise`
// branch of the original streaming loop.
func (s *Server) awaitRelease(r *http.Request) bool {
	s.mu.Lock()
	if !s.held {
		s.mu.Unlock()
		return false
	}
	release := s.release
	s.mu.Unlock()
	select {
	case <-release:
		return true
	case <-r.Context().Done():
		return false
	}
}

// writeJSON answers with a JSON body, like the original json() helper.
func writeJSON(w http.ResponseWriter, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// Start builds the SDK archives and serves the registry on 127.0.0.1 with an
// ephemeral port. It returns the base URL and a stop function.
func Start(pluginFile string) (baseURL string, stop func(), err error) {
	plugin, err := os.ReadFile(pluginFile)
	if err != nil {
		return "", nil, fmt.Errorf("read test plugin: %w", err)
	}
	archives := make(map[string][]byte, len(testVersions))
	for index, version := range testVersions {
		archive, err := buildArchive(version, uint32(42+index))
		if err != nil {
			return "", nil, fmt.Errorf("build testsdk %s: %w", version, err)
		}
		archives[version] = archive
	}
	server := &Server{plugin: plugin, archives: archives}
	mux := http.NewServeMux()
	mux.HandleFunc("/registry/index.json", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, []registryEntry{{
			Name:     "testsdk",
			Desc:     "An offline SDK for the vfox-plus end-to-end tests.",
			Homepage: "https://github.com/version-fox/vfox-plus",
		}})
	})
	mux.HandleFunc("/registry/testsdk.json", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, manifest{
			Name:              "testsdk",
			Version:           "0.1.0",
			License:           "Apache-2.0",
			Author:            "vfox-plus",
			DownloadURL:       "http://" + r.Host + "/plugin/testsdk.lua",
			MinRuntimeVersion: "1.0.0",
		})
	})
	mux.HandleFunc("/plugin/testsdk.lua", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(server.plugin)
	})
	mux.HandleFunc("/sdk/", func(w http.ResponseWriter, r *http.Request) {
		match := sdkPath.FindStringSubmatch(path.Clean(r.URL.Path))
		if match == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		archive, ok := server.archives[match[1]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/gzip")
		w.Header().Set("Content-Length", fmt.Sprint(len(archive)))
		w.WriteHeader(http.StatusOK)
		for offset := 0; offset < len(archive); {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			end := offset + chunkSize
			if end > len(archive) {
				end = len(archive)
			}
			if _, err := w.Write(archive[offset:end]); err != nil {
				return
			}
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			offset = end
			if offset >= len(archive) {
				break
			}
			if server.awaitRelease(r) {
				continue
			}
			select {
			case <-time.After(chunkDelay):
			case <-r.Context().Done():
				return
			}
		}
	})
	mux.HandleFunc("/control/hold", func(w http.ResponseWriter, _ *http.Request) {
		server.hold()
		_, _ = w.Write([]byte("held\n"))
	})
	mux.HandleFunc("/control/release", func(w http.ResponseWriter, _ *http.Request) {
		server.unhold()
		_, _ = w.Write([]byte("released\n"))
	})
	// Anything else is a bare 404 like the original: empty body, no
	// framework error page.
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	server.listener = listener
	server.server = &http.Server{Handler: mux}
	go func() { _ = server.server.Serve(listener) }()
	return server.baseURL(), func() { _ = server.server.Close() }, nil
}
