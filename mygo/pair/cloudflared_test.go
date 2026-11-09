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

package pair

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTheTunnelProgramOfEveryPlatformIsPinned(t *testing.T) {
	for _, platform := range []struct{ goos, goarch, name string }{
		{"linux", "amd64", "cloudflared-linux-amd64"},
		{"linux", "arm64", "cloudflared-linux-arm64"},
		{"windows", "amd64", "cloudflared-windows-amd64.exe"},
		// Windows on ARM runs the x64 program.
		{"windows", "arm64", "cloudflared-windows-amd64.exe"},
	} {
		a, err := cloudflaredFor(platform.goos, platform.goarch)
		if err != nil {
			t.Errorf("%s/%s: %v", platform.goos, platform.goarch, err)
			continue
		}
		if a.name != platform.name {
			t.Errorf("%s/%s uses %s, want %s", platform.goos, platform.goarch, a.name, platform.name)
		}
		if len(a.sha256) != 64 {
			t.Errorf("%s/%s: checksum %q", platform.goos, platform.goarch, a.sha256)
		}
		if want := cloudflaredDownloadBase + "/" + cloudflaredVersion + "/" + platform.name; a.URLs()[0] != want {
			t.Errorf("%s/%s: URL %q, want %q", platform.goos, platform.goarch, a.URLs()[0], want)
		}
		if !strings.HasPrefix(a.URLs()[1], "https://api.github.com/repos/cloudflare/cloudflared/releases/assets/") {
			t.Errorf("%s/%s: API URL %q", platform.goos, platform.goarch, a.URLs()[1])
		}
	}
	if _, err := cloudflaredFor("plan9", "mips"); err == nil {
		t.Error("a platform without a release is an error")
	}
}

func TestTheProgramNameIsTheOneTheApplicationLooksFor(t *testing.T) {
	if cloudflaredName("windows") != "cloudflared.exe" || cloudflaredName("linux") != "cloudflared" {
		t.Errorf("names: %q, %q", cloudflaredName("windows"), cloudflaredName("linux"))
	}
}

func TestDownloadProgramChecksTheChecksum(t *testing.T) {
	body := []byte("the program")
	sum := sha256.Sum256(body)
	want := hex.EncodeToString(sum[:])
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer server.Close()

	path := filepath.Join(t.TempDir(), "cloudflared")
	if err := downloadProgram(context.Background(), []string{server.URL}, "cloudflared-linux-amd64", want, path); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Errorf("the downloaded program is not executable: %v", info.Mode())
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != string(body) { //nolint:gosec // The path is a temporary file of the test.
		t.Fatalf("content %q, %v", got, err)
	}
	if !matches(path, want) {
		t.Error("the file matches its checksum")
	}

	// Other bytes are refused, and leave nothing behind.
	bad := filepath.Join(t.TempDir(), "cloudflared")
	err = downloadProgram(context.Background(), []string{server.URL}, "cloudflared-linux-amd64", strings.Repeat("0", 64), bad)
	if err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Errorf("a mismatched checksum is an error, got %v", err)
	}
	if _, statErr := os.Stat(bad); !os.IsNotExist(statErr) {
		t.Errorf("a refused download leaves no file: %v", statErr)
	}
}

// When the release download fails, the API of the same asset is tried.
func TestDownloadProgramFallsBackToTheAPI(t *testing.T) {
	body := []byte("the program")
	sum := sha256.Sum256(body)
	want := hex.EncodeToString(sum[:])
	failing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "gateway timeout", http.StatusGatewayTimeout)
	}))
	defer failing.Close()
	serving := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer serving.Close()

	path := filepath.Join(t.TempDir(), "cloudflared")
	if err := downloadProgram(context.Background(), []string{failing.URL, serving.URL}, "cloudflared-linux-amd64", want, path); err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(path); err != nil || string(got) != string(body) { //nolint:gosec // The path is a temporary file of the test.
		t.Fatalf("content %q, %v", got, err)
	}
}

func TestAFailedDownloadIsAnError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not here", http.StatusNotFound)
	}))
	defer server.Close()
	path := filepath.Join(t.TempDir(), "cloudflared")
	if err := downloadProgram(context.Background(), []string{server.URL}, "cloudflared-linux-amd64", strings.Repeat("0", 64), path); err == nil {
		t.Error("a missing asset is an error")
	}
}

// A program in the cache is the one that a download would produce, so the
// settings find it without asking again.
func TestTheCacheIsSearchedBeforeThePath(t *testing.T) {
	cache := t.TempDir()
	path := filepath.Join(cache, cloudflaredName(runtime.GOOS))
	if err := os.WriteFile(path, []byte("program"), 0o755); err != nil { //nolint:gosec // A stand-in program is executable.
		t.Fatal(err)
	}
	t.Setenv("VFOX_PLUS_CLOUDFLARED", "")
	t.Setenv("VFOX_PLUS_CLOUDFLARED_CACHE", cache)
	t.Setenv("PATH", filepath.Join(t.TempDir(), "no-programs"))
	if got := CloudflaredPath(); got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}
