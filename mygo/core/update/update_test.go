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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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
	path, err := Download(context.Background(), direct.Client(), direct.URL+"/file.deb", mirror.URL+"/", directory, nil)
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
