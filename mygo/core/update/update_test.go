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

package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
)

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{
		{"0.2.0", "0.1.0", 1},
		{"v0.2.0", "0.2.0", 0},
		{"0.10.0", "0.2.0", 1},
		{"0.2.0-rc.1", "0.2.0", -1},
		{"0.1.0", "0.1.0", 0},
	} {
		if got := CompareVersions(c.a, c.b); (got > 0) != (c.want > 0) || (got < 0) != (c.want < 0) {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestCheckReadsGitHubAPI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","name":"0.2.0","html_url":"https://github.com/version-fox/vfox-plus/releases/tag/v0.2.0",` +
			`"assets":[{"name":"vfox-plus_0.2.0_amd64.deb","browser_download_url":"https://github.com/version-fox/vfox-plus/releases/download/v0.2.0/vfox-plus_0.2.0_amd64.deb","size":10}]}`))
	}))
	defer server.Close()
	release, err := Check(context.Background(), server.Client(), server.URL)
	if err != nil || release.Tag != "v0.2.0" || len(release.Assets) != 1 {
		t.Fatalf("got %+v, %v", release, err)
	}
	if !release.NewerThan("0.1.0") || release.NewerThan("0.2.0") {
		t.Errorf("newer check wrong for %+v", release)
	}
	if asset, ok := PlatformAsset(release, "linux", "amd64"); !ok || !strings.HasSuffix(asset.Name, ".deb") {
		t.Errorf("no linux asset: %+v", release.Assets)
	}
}

func TestPlatformAssetPrefersDebAndFallsBackToTarball(t *testing.T) {
	release := Release{Tag: "v0.2.0", Assets: []Asset{
		{Name: "vfox-plus-0.2.0-linux-x64.tar.gz", URL: "https://example.com/a.tar.gz"},
		{Name: "vfox-plus_0.2.0_amd64.deb", URL: "https://example.com/a.deb"},
	}}
	if asset, ok := PlatformAsset(release, "linux", "amd64"); !ok || !strings.HasSuffix(asset.Name, ".deb") {
		t.Errorf("deb is preferred: %+v", asset)
	}
	tarballOnly := Release{Tag: "v0.2.0", Assets: []Asset{
		{Name: "vfox-plus-0.2.0-linux-x64.tar.gz", URL: "https://example.com/a.tar.gz"},
	}}
	if asset, ok := PlatformAsset(tarballOnly, "linux", "amd64"); !ok || !strings.HasSuffix(asset.Name, ".tar.gz") {
		t.Errorf("tarball is the fallback: %+v", asset)
	}
	if _, ok := PlatformAsset(tarballOnly, "linux", "386"); ok {
		t.Error("no 386 asset exists")
	}
	if asset, ok := PlatformAsset(Release{Tag: "v0.2.0", Assets: []Asset{
		{Name: "vfox-plus_0.2.0.msixbundle", URL: "https://example.com/a.msixbundle"},
	}}, "windows", "amd64"); !ok || !strings.HasSuffix(asset.Name, ".msixbundle") {
		t.Errorf("bundle is the windows fallback: %+v", asset)
	}
	// The arm64 files of a release are picked for arm64, and the amd64 ones stay.
	arm := Release{Tag: "v0.2.0", Assets: []Asset{
		{Name: "vfox-plus-0.2.0-linux-x64.tar.gz", URL: "https://example.com/a.tar.gz"},
		{Name: "vfox-plus_0.2.0_amd64.deb", URL: "https://example.com/a.deb"},
		{Name: "vfox-plus-0.2.0-linux-arm64.tar.gz", URL: "https://example.com/arm.tar.gz"},
		{Name: "vfox-plus_0.2.0_arm64.deb", URL: "https://example.com/arm.deb"},
	}}
	if asset, ok := PlatformAsset(arm, "linux", "arm64"); !ok || asset.Name != "vfox-plus_0.2.0_arm64.deb" {
		t.Errorf("the arm64 deb is preferred: %+v", asset)
	}
	armTarballOnly := Release{Tag: "v0.2.0", Assets: []Asset{
		{Name: "vfox-plus-0.2.0-linux-arm64.tar.gz", URL: "https://example.com/arm.tar.gz"},
	}}
	if asset, ok := PlatformAsset(armTarballOnly, "linux", "arm64"); !ok || asset.Name != "vfox-plus-0.2.0-linux-arm64.tar.gz" {
		t.Errorf("the arm64 tarball is the fallback: %+v", asset)
	}
}

func TestMirrorURL(t *testing.T) {
	raw := "https://github.com/version-fox/vfox-plus/releases/download/v0.2.0/vfox-plus_0.2.0_amd64.deb"
	if got := MirrorURL("https://gh-proxy.org/", raw); got != "https://gh-proxy.org/"+raw {
		t.Errorf("got %q", got)
	}
	if got := MirrorURL("", raw); got != raw {
		t.Errorf("got %q", got)
	}
}

// payloadDigest is the digest GitHub publishes for a file whose content is "payload".
var payloadDigest = func() string {
	sum := sha256.Sum256([]byte("payload"))
	return "sha256:" + hex.EncodeToString(sum[:])
}()

func TestDownloadFallsBackToDirect(t *testing.T) {
	direct := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer direct.Close()
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusBadGateway)
	}))
	defer mirror.Close()
	// The mirror prefix points at a failing server; download falls back to direct.
	directory := t.TempDir()
	asset := Asset{Name: "file.deb", URL: direct.URL + "/file.deb", Digest: payloadDigest}
	path, err := Download(context.Background(), direct.Client(), asset, mirror.URL+"/", directory, nil)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path) //nolint:gosec // The path is the downloaded file.
	if err != nil || string(content) != "payload" {
		t.Fatalf("got %q, %v", content, err)
	}
	if !strings.HasPrefix(filepath.Base(path), "file.deb") {
		t.Errorf("keeps the file name: %q", path)
	}
}

func TestDownloadWithoutMirrorAsksOnce(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.Error(w, "gone", http.StatusInternalServerError)
	}))
	defer server.Close()
	asset := Asset{Name: "file.deb", URL: server.URL + "/file.deb", Digest: payloadDigest}
	if _, err := Download(context.Background(), server.Client(), asset, "", t.TempDir(), nil); err == nil {
		t.Fatal("a failing server downloaded a file")
	}
	if n := requests.Load(); n != 1 {
		t.Fatalf("the server was asked %d times with no mirror, want once", n)
	}
}

func TestDownloadRefusesAnAssetWithoutAUsableDigest(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("payload"))
	}))
	defer server.Close()
	for _, digest := range []string{"", "md5:0123", "sha256:xyz", "sha256:" + strings.Repeat("0", 63)} {
		asset := Asset{Name: "file.deb", URL: server.URL + "/file.deb", Digest: digest}
		if path, err := Download(context.Background(), server.Client(), asset, "", t.TempDir(), nil); err == nil {
			t.Errorf("digest %q: downloaded %s, want a refusal", digest, path)
		}
	}
	if n := requests.Load(); n != 0 {
		t.Fatalf("the server was asked %d times for assets without a digest, want none", n)
	}
}

func TestDownloadKeepsOnlyAFileThatMatchesTheDigest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("tampered"))
	}))
	defer server.Close()
	directory := t.TempDir()
	asset := Asset{Name: "file.deb", URL: server.URL + "/file.deb", Digest: payloadDigest}
	if path, err := Download(context.Background(), server.Client(), asset, "", directory, nil); err == nil {
		t.Fatalf("downloaded %s from a server that serves other bytes", path)
	}
	entries, err := os.ReadDir(directory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("the file that did not match was kept: %v, %v", entries, err)
	}
}

func TestDownloadDoesNotKeepATamperedMirror(t *testing.T) {
	direct := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("payload"))
	}))
	defer direct.Close()
	mirror := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A mirror that answers 200 with other bytes must not be kept.
		_, _ = w.Write([]byte("tampered"))
	}))
	defer mirror.Close()
	directory := t.TempDir()
	asset := Asset{Name: "file.deb", URL: direct.URL + "/file.deb", Digest: payloadDigest}
	path, err := Download(context.Background(), direct.Client(), asset, mirror.URL+"/", directory, nil)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path) //nolint:gosec // The path is the downloaded file.
	if err != nil || string(content) != "payload" {
		t.Fatalf("got %q, %v; want the direct download", content, err)
	}
	if entries, _ := os.ReadDir(directory); len(entries) != 1 {
		t.Fatalf("%d files are left in the directory, want the one download", len(entries))
	}
}

func TestCheckReadsTheDigestOfEachAsset(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0","assets":[{"name":"vfox-plus_0.2.0_amd64.deb",` +
			`"browser_download_url":"https://github.com/version-fox/vfox-plus/releases/download/v0.2.0/vfox-plus_0.2.0_amd64.deb",` +
			`"size":12,"digest":"sha256:` + strings.Repeat("ab", 32) + `"}]}`))
	}))
	defer server.Close()
	release, err := Check(context.Background(), server.Client(), server.URL)
	if err != nil || len(release.Assets) != 1 {
		t.Fatalf("got %+v, %v", release, err)
	}
	if got, want := release.Assets[0].Digest, "sha256:"+strings.Repeat("ab", 32); got != want {
		t.Fatalf("digest = %q, want %q", got, want)
	}
}
