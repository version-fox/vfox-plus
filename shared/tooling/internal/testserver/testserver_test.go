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

package testserver

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// startTestServer builds a server with a temporary plugin file.
func startTestServer(t *testing.T) (string, func()) {
	t.Helper()
	plugin := "-- testsdk test plugin\n"
	pluginFile := filepath.Join(t.TempDir(), "testsdk.lua")
	if err := os.WriteFile(pluginFile, []byte(plugin), 0o644); err != nil {
		t.Fatal(err)
	}
	base, stop, err := Start(pluginFile)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(stop)
	if !strings.HasPrefix(base, "http://127.0.0.1:") {
		t.Fatalf("base URL = %q, want http://127.0.0.1:<port>", base)
	}
	return base, stop
}

func get(t *testing.T, url string) (int, []byte, http.Header) {
	t.Helper()
	response, err := http.Get(url) //nolint:gosec // The test server runs on loopback.
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, body, response.Header
}

func TestRegistry(t *testing.T) {
	base, _ := startTestServer(t)

	status, body, header := get(t, base+"/registry/index.json")
	if status != http.StatusOK {
		t.Fatalf("GET /registry/index.json status = %d, want 200", status)
	}
	if contentType := header.Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	var index []map[string]string
	if err := json.Unmarshal(body, &index); err != nil {
		t.Fatal(err)
	}
	if len(index) != 1 || index[0]["name"] != "testsdk" || index[0]["homepage"] != "https://github.com/version-fox/vfox-plus" {
		t.Errorf("registry index = %s, want the testsdk entry", body)
	}

	status, body, _ = get(t, base+"/registry/testsdk.json")
	if status != http.StatusOK {
		t.Fatalf("GET /registry/testsdk.json status = %d, want 200", status)
	}
	var manifest map[string]string
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"name": "testsdk", "version": "0.1.0", "license": "Apache-2.0",
		"author": "vfox-plus", "minRuntimeVersion": "1.0.0",
	}
	for key, value := range want {
		if manifest[key] != value {
			t.Errorf("manifest[%q] = %q, want %q", key, manifest[key], value)
		}
	}
	if !strings.HasPrefix(manifest["downloadUrl"], base+"/plugin/testsdk.lua") {
		t.Errorf("downloadUrl = %q, want it under the base URL", manifest["downloadUrl"])
	}
}

func TestPlugin(t *testing.T) {
	base, _ := startTestServer(t)
	status, body, _ := get(t, base+"/plugin/testsdk.lua")
	if status != http.StatusOK || string(body) != "-- testsdk test plugin\n" {
		t.Errorf("GET /plugin/testsdk.lua = %d %q, want 200 and the plugin source", status, body)
	}
}

func extract(t *testing.T, archive []byte) map[string]*tar.Header {
	t.Helper()
	reader, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	files := map[string]*tar.Header{}
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			return files
		}
		if err != nil {
			t.Fatal(err)
		}
		files[header.Name] = header
		if _, err := io.Copy(io.Discard, tarReader); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSdkArchive(t *testing.T) {
	base, _ := startTestServer(t)
	status, body, _ := get(t, base+"/sdk/testsdk-2.0.0.tar.gz")
	if status != http.StatusOK || len(body) == 0 {
		t.Fatalf("GET /sdk/testsdk-2.0.0.tar.gz status = %d, %d bytes", status, len(body))
	}
	files := extract(t, body)
	script, ok := files["testsdk-2.0.0/bin/testsdk"]
	if !ok {
		t.Fatalf("archive entries = %v, want bin/testsdk", files)
	}
	if script.Mode&0o777 != 0o755 {
		t.Errorf("bin/testsdk mode = %o, want 755", script.Mode)
	}
	payload, ok := files["testsdk-2.0.0/payload.bin"]
	if !ok {
		t.Fatal("archive has no payload.bin")
	}
	if payload.Size != 512*1024 {
		t.Errorf("payload.bin size = %d, want %d", payload.Size, 512*1024)
	}

	// Archives are deterministic: same version, same bytes.
	again, err := buildArchive("2.0.0", 43)
	if err != nil {
		t.Fatal(err)
	}
	once, err := buildArchive("2.0.0", 43)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, once) {
		t.Error("buildArchive is not deterministic")
	}
}

func TestHoldAndRelease(t *testing.T) {
	base, _ := startTestServer(t)
	if status, _, _ := get(t, base+"/control/hold"); status != http.StatusOK {
		t.Fatalf("GET /control/hold status = %d, want 200", status)
	}
	type result struct {
		body   []byte
		status int
		err    error
	}
	done := make(chan result, 1)
	failed := make(chan error, 1)
	go func() {
		var r result
		response, err := http.Get(base + "/sdk/testsdk-1.0.0.tar.gz") //nolint:gosec // Loopback test server.
		if err != nil {
			failed <- err
			return
		}
		defer response.Body.Close()
		r.status = response.StatusCode
		r.body, r.err = io.ReadAll(response.Body)
		done <- r
	}()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("download failed while held: %v", r.err)
		}
		t.Fatal("the download finished while held")
	case err := <-failed:
		t.Fatalf("download failed while held: %v", err)
	case <-time.After(300 * time.Millisecond):
	}
	if status, body, _ := get(t, base+"/control/release"); status != http.StatusOK || string(body) != "released\n" {
		t.Fatalf("GET /control/release = %d %q", status, body)
	}
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("download failed after release: %v", r.err)
		}
		if r.status != http.StatusOK {
			t.Fatalf("released download status = %d, want 200", r.status)
		}
		if len(r.body) == 0 {
			t.Error("the released download is empty")
		}
	case err := <-failed:
		t.Fatalf("download failed after release: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("the download did not finish after release")
	}
}

func TestUnknownPath(t *testing.T) {
	base, _ := startTestServer(t)
	status, body, _ := get(t, base+"/no-such-path")
	if status != http.StatusNotFound {
		t.Errorf("GET /no-such-path status = %d, want 404", status)
	}
	if len(body) != 0 {
		t.Errorf("GET /no-such-path body = %q, want empty", body)
	}
}
