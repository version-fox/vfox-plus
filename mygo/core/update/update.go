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

// Package update checks for new releases of vfox-plus itself and downloads
// them, from GitHub or through a mirror prefix such as https://gh-proxy.org/.
//
// The source of truth is the GitHub releases of version-fox/vfox-plus:
// fetching https://api.github.com/repos/version-fox/vfox-plus/releases/latest
// gives the newest tag and its assets. Asset downloads go to
// https://github.com/version-fox/vfox-plus/releases/download/<tag>/<asset>,
// optionally prefixed with the mirror (for example
// https://gh-proxy.org/https://github.com/...), which is how the application
// updates where github.com is slow. Nothing here installs: the .deb is left
// for dpkg and the .msixbundle for the installer, so the GUI only downloads
// and verifies.
//
// No release exists yet, so the asset names below are the convention the
// packaging uses (see shared/tooling/internal/builddeb and buildmsix):
// vfox-plus_<version>_amd64.deb, vfox-plus_<version>_arm64.deb,
// vfox-plus-<version>-linux-x64.tar.gz, vfox-plus-<version>-linux-arm64.tar.gz,
// vfox-plus_<version>_x64.msix, vfox-plus_<version>_arm64.msix and
// vfox-plus_<version>.msixbundle. On Linux the .deb is preferred and the
// .tar.gz is the fallback; on Windows the per-architecture .msix is
// preferred and the .msixbundle is the fallback.
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Repository is the GitHub repository releases are read from.
const Repository = "version-fox/vfox-plus"

// LatestAPI is the endpoint of the newest release.
const LatestAPI = "https://api.github.com/repos/" + Repository + "/releases/latest"

// Asset is one downloadable file of a release.
type Asset struct {
	// File name, for example vfox-plus_0.2.0_amd64.deb.
	Name string
	// Download URL on github.com.
	URL string
	// Size in bytes, when the API reports one.
	Size int64
}

// Release is the newest release the API reports.
type Release struct {
	// Tag such as v0.2.0.
	Tag string
	// Title of the release.
	Name string
	// Page of the release on github.com.
	Page string
	// Assets of the release.
	Assets []Asset
}

// Version strips the leading v of the tag: v0.2.0 becomes 0.2.0.
func (r Release) Version() string { return strings.TrimPrefix(r.Tag, "v") }

// NewerThan reports whether the release is newer than current, comparing
// dot-separated numbers. A tag that is not a version never counts as newer.
func (r Release) NewerThan(current string) bool {
	return CompareVersions(r.Version(), current) > 0
}

// CompareVersions compares dot-separated versions with an optional leading
// v: 0.2.0 is newer than 0.1.0, v0.2.0 equals 0.2.0. Non-numeric tails are
// compared as text after the numbers, so 0.2.0-rc.1 is older than 0.2.0.
func CompareVersions(a, b string) int {
	pa, ta := splitVersion(a)
	pb, tb := splitVersion(b)
	for i := 0; i < len(pa) || i < len(pb); i++ {
		var x, y int
		if i < len(pa) {
			x = pa[i]
		}
		if i < len(pb) {
			y = pb[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	// A tail such as -rc.1 marks a pre-release, older than the release.
	if ta != "" && tb == "" {
		return -1
	}
	if ta == "" && tb != "" {
		return 1
	}
	return strings.Compare(ta, tb)
}

func splitVersion(version string) ([]int, string) {
	version = strings.TrimSpace(strings.TrimPrefix(version, "v"))
	var numbers []int
	i := 0
	for i < len(version) && (version[i] >= '0' && version[i] <= '9' || version[i] == '.' || version[i] == ' ' || version[i] == '_') {
		i++
	}
	head, tail := version[:i], strings.TrimSpace(version[i:])
	head = strings.ReplaceAll(strings.ReplaceAll(head, " ", "."), "_", ".")
	for _, part := range strings.Split(head, ".") {
		if part == "" {
			continue
		}
		n := 0
		for _, r := range part {
			if r < '0' || r > '9' {
				n = -1
				break
			}
			n = n*10 + int(r-'0')
		}
		if n < 0 {
			break
		}
		numbers = append(numbers, n)
	}
	return numbers, tail
}

// MirrorURL prefixes rawURL with the mirror prefix, for example
// https://gh-proxy.org/ + https://github.com/... An empty prefix returns
// the URL unchanged.
func MirrorURL(prefix, rawURL string) string {
	prefix = strings.TrimSpace(prefix)
	rawURL = strings.TrimSpace(rawURL)
	if prefix == "" || rawURL == "" {
		return rawURL
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix + rawURL
}

// Check asks the GitHub API for the newest release. apiURL defaults to
// LatestAPI when empty; client defaults to http.DefaultClient.
func Check(ctx context.Context, client *http.Client, apiURL string) (Release, error) {
	if client == nil {
		client = http.DefaultClient
	}
	if strings.TrimSpace(apiURL) == "" {
		apiURL = LatestAPI
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return Release{}, err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	response, err := client.Do(request)
	if err != nil {
		return Release{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return Release{}, fmt.Errorf("github api: %s", response.Status)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return Release{}, err
	}
	var raw struct {
		TagName string `json:"tag_name"`
		Name    string `json:"name"`
		HTMLURL string `json:"html_url"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Size               int64  `json:"size"`
		} `json:"assets"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return Release{}, err
	}
	if strings.TrimSpace(raw.TagName) == "" {
		return Release{}, fmt.Errorf("github api: no tag_name")
	}
	release := Release{Tag: raw.TagName, Name: raw.Name, Page: raw.HTMLURL}
	for _, asset := range raw.Assets {
		if asset.Name == "" || asset.BrowserDownloadURL == "" {
			continue
		}
		release.Assets = append(release.Assets, Asset{Name: asset.Name, URL: asset.BrowserDownloadURL, Size: asset.Size})
	}
	return release, nil
}

// PlatformAsset picks the asset for the platform: a .deb on Linux (the
// .tar.gz is the fallback) and an .msix on Windows, matching the
// architecture. The bundle is the fallback on Windows when no
// per-architecture package matches.
func PlatformAsset(release Release, goos, goarch string) (Asset, bool) {
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}
	debArch := ""
	switch goarch {
	case "amd64":
		debArch = "amd64"
	case "arm64":
		debArch = "arm64"
	}
	msixArch := ""
	switch goarch {
	case "amd64":
		msixArch = "x64"
	case "arm64":
		msixArch = "arm64"
	}
	switch goos {
	case "linux":
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.HasSuffix(name, ".deb") && debArch != "" && strings.Contains(name, "_"+debArch+".deb") {
				return asset, true
			}
		}
		tarArch := ""
		switch goarch {
		case "amd64":
			tarArch = "x64"
		case "arm64":
			tarArch = "arm64"
		}
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.HasSuffix(name, ".tar.gz") && tarArch != "" && strings.Contains(name, "linux-"+tarArch+".tar.gz") {
				return asset, true
			}
		}
	case "windows":
		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.HasSuffix(name, ".msix") && !strings.HasSuffix(name, ".msixbundle") && msixArch != "" && strings.Contains(name, "_"+msixArch+".msix") {
				return asset, true
			}
		}
		for _, asset := range release.Assets {
			if strings.HasSuffix(strings.ToLower(asset.Name), ".msixbundle") {
				return asset, true
			}
		}
	}
	return Asset{}, false
}

// Download fetches url (optionally through mirrorPrefix) into a temporary
// file in directory and returns its path. Progress reports the fraction
// downloaded when the size is known, -1 otherwise; it may be nil.
func Download(ctx context.Context, client *http.Client, url, mirrorPrefix, directory string, progress func(float64)) (string, error) {
	if client == nil {
		client = http.DefaultClient
	}
	target := url
	if mirrored := MirrorURL(mirrorPrefix, url); mirrored != "" {
		// Try the mirror first when one is configured; fall back to direct.
		if path, err := downloadOne(ctx, client, mirrored, directory, progress); err == nil {
			return path, nil
		}
	}
	return downloadOne(ctx, client, target, directory, progress)
}

func downloadOne(ctx context.Context, client *http.Client, url, directory string, progress func(float64)) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", fmt.Errorf("empty download url")
	}
	if directory == "" {
		directory = os.TempDir()
	}
	if err := os.MkdirAll(directory, 0o750); err != nil {
		return "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("download: %s", response.Status)
	}
	base := filepath.Base(request.URL.Path)
	if base == "" || base == "/" || base == "." {
		base = "vfox-plus-update"
	}
	file, err := os.CreateTemp(directory, filepath.Base(base)+"-*")
	if err != nil {
		return "", err
	}
	name := file.Name()
	total := response.ContentLength
	var done int64
	buffer := make([]byte, 128*1024)
	_, err = func() (int64, error) {
		defer file.Close()
		for {
			n, readErr := response.Body.Read(buffer)
			if n > 0 {
				if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
					return done, writeErr
				}
				done += int64(n)
				if progress != nil {
					if total > 0 {
						progress(float64(done) / float64(total))
					} else {
						progress(-1)
					}
				}
			}
			if readErr != nil {
				if readErr == io.EOF {
					return done, nil
				}
				return done, readErr
			}
		}
	}()
	if err != nil {
		_ = os.Remove(name)
		return "", err
	}
	return name, nil
}
