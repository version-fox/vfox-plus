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
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// cloudflaredVersion is the cloudflared release that the public tunnel runs.
// The checksums below are the ones of that release: change them together.
const cloudflaredVersion = "2026.10.7"

// cloudflaredDownloadBase is where Cloudflare publishes the release assets.
const cloudflaredDownloadBase = "https://github.com/cloudflare/cloudflared/releases/download"

// cloudflaredAsset is one published program and the SHA-256 of its bytes.
type cloudflaredAsset struct {
	name   string
	sha256 string
	// id is the asset number of the GitHub API, the way to the same bytes
	// when the release download is slow to answer.
	id string
}

// cloudflaredAssets holds the program of every platform the application runs
// on. Cloudflare publishes no Windows arm64 build: Windows 11 on ARM runs the
// x64 one, so it stands in for both.
var cloudflaredAssets = map[string]cloudflaredAsset{
	"linux/amd64":   {"cloudflared-linux-amd64", "1f99fb518b83dec2ba8d870a7cc991d82a77e6196295a3d2754ba17ab9f91872", "692817008"},
	"linux/arm64":   {"cloudflared-linux-arm64", "3df43aedc9ff291d1735b5cb576cb97e0b157488796db2e32b28aecabb445775", "692818763"},
	"windows/amd64": {"cloudflared-windows-amd64.exe", "2493dbc168923bb3e59e5e7716f71deced92992eea9acd6e9d74f84ac7751343", "692820426"},
	"windows/arm64": {"cloudflared-windows-amd64.exe", "2493dbc168923bb3e59e5e7716f71deced92992eea9acd6e9d74f84ac7751343", "692820426"},
}

// cloudflaredName is the file name of the program on a platform.
func cloudflaredName(goos string) string {
	if goos == "windows" {
		return "cloudflared.exe"
	}
	return "cloudflared"
}

// cloudflaredFor returns the asset of a Go platform.
func cloudflaredFor(goos, goarch string) (cloudflaredAsset, error) {
	a, ok := cloudflaredAssets[goos+"/"+goarch]
	if !ok {
		return cloudflaredAsset{}, fmt.Errorf("no tunnel program for %s/%s", goos, goarch)
	}
	return a, nil
}

// URLs are where the asset can be fetched, the release download first and the
// API of the same asset second.
func (a cloudflaredAsset) URLs() []string {
	return []string{
		cloudflaredDownloadBase + "/" + cloudflaredVersion + "/" + a.name,
		"https://api.github.com/repos/cloudflare/cloudflared/releases/assets/" + a.id,
	}
}

// CloudflaredPath is the tunnel program that can run now: the one that
// VFOX_PLUS_CLOUDFLARED names, then a copy beside the application, then the
// one downloaded for this version, then one on the PATH. Empty when there is
// none, which is when the settings offer to download it.
func CloudflaredPath() string {
	if path := os.Getenv("VFOX_PLUS_CLOUDFLARED"); path != "" {
		if info, err := os.Stat(path); err == nil && !info.IsDir() { //nolint:gosec // The user names the program to run.
			return path
		}
	}
	name := cloudflaredName(runtime.GOOS)
	if executable, err := os.Executable(); err == nil {
		path := filepath.Join(filepath.Dir(executable), name)
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			return path
		}
	}
	if cache, err := cloudflaredCachePath(); err == nil {
		if info, err := os.Stat(cache); err == nil && !info.IsDir() {
			return cache
		}
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// DownloadCloudflared makes sure that the tunnel program of this platform is
// on the machine, downloading the pinned release and checking its SHA-256
// when it is not, and returns its path.
func DownloadCloudflared(ctx context.Context) (string, error) {
	if path := CloudflaredPath(); path != "" {
		return path, nil
	}
	a, err := cloudflaredFor(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return "", err
	}
	dest, err := cloudflaredCachePath()
	if err != nil {
		return "", err
	}
	// A copy of the right bytes is a download already made; anything else is
	// replaced by the program of the release.
	if matches(dest, a.sha256) {
		return dest, nil
	}
	if err := downloadProgram(ctx, a.URLs(), a.name, a.sha256, dest); err != nil {
		return "", err
	}
	return dest, nil
}

// cloudflaredCachePath is where a downloaded program waits, one per version.
func cloudflaredCachePath() (string, error) {
	if dir := os.Getenv("VFOX_PLUS_CLOUDFLARED_CACHE"); dir != "" {
		return filepath.Join(dir, cloudflaredName(runtime.GOOS)), nil
	}
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "vfox-plus", "cloudflared", cloudflaredVersion, cloudflaredName(runtime.GOOS)), nil
}

// matches reports whether path holds the bytes with this checksum.
func matches(path, want string) bool {
	got, err := fileChecksum(path)
	return err == nil && got == want
}

// fileChecksum returns the SHA-256 of a file in hex.
func fileChecksum(path string) (string, error) {
	file, err := os.Open(path) //nolint:gosec // The path is the cache file of the pinned program.
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(sum.Sum(nil)), nil
}

// errChecksum means that the bytes were not the pinned program: no retry can
// change that, unlike a network failure.
var errChecksum = errors.New("checksum mismatch")

// statusError is an HTTP answer that was not OK. A client error is final; a
// server error may pass.
type statusError struct {
	code int
	url  string
}

func (e statusError) Error() string {
	return fmt.Sprintf("download %s: %d %s", e.url, e.code, http.StatusText(e.code))
}

// downloadProgram fetches a program to path from the first URL that answers,
// refusing bytes that do not match the pinned checksum. A transient failure
// of GitHub, which serves the assets, gets another try.
func downloadProgram(ctx context.Context, urls []string, name, want, path string) error {
	var err error
	for _, url := range urls {
		for attempt := 0; attempt < 2; attempt++ {
			if attempt > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(2 * time.Second):
				}
			}
			if err = downloadProgramOnce(ctx, url, name, want, path); err == nil || errors.Is(err, errChecksum) {
				return err
			}
			var status statusError
			if errors.As(err, &status) && status.code < 500 {
				break
			}
		}
	}
	return err
}

// downloadProgramOnce is one attempt of downloadProgram.
func downloadProgramOnce(ctx context.Context, url, name, want, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// The API answers with the bytes of the asset only when asked for them.
	request.Header.Set("Accept", "application/octet-stream")
	client := &http.Client{Timeout: 10 * time.Minute}
	response, err := client.Do(request) //nolint:gosec // The URL is built from constants or a test server.
	if err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return statusError{code: response.StatusCode, url: url}
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".cloudflared-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer func() { _ = os.Remove(temporary) }()
	sum := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, sum), response.Body); err != nil {
		_ = file.Close()
		return fmt.Errorf("download %s: %w", url, err)
	}
	if err := file.Close(); err != nil {
		return err
	}
	if got := hex.EncodeToString(sum.Sum(nil)); got != want {
		return fmt.Errorf("%s: %w: %s, want %s", name, errChecksum, got, want)
	}
	if err := os.Chmod(temporary, 0o700); err != nil { //nolint:gosec // The program must be executable by its owner.
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		// Windows does not replace an existing file with a rename.
		if removeErr := os.Remove(path); removeErr != nil {
			return err
		}
		return os.Rename(temporary, path)
	}
	return nil
}
