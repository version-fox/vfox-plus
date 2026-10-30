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

package prepare

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha512"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// fakeIcons is the icons.json of a package that has what the tests ask of it: icons
// and aliases, and an icon with a box of its own, one that flips and one alias
// that flips.
const fakeIcons = `{
  "prefix": "material-symbols",
  "icons": {
    "home-rounded": {"body": "<path fill=\"currentColor\" d=\"M1 1h2z\"/>"},
    "home-outline-rounded": {"body": "<path fill=\"currentColor\" d=\"M2 2h2z\"/>"},
    "folder-open-rounded": {"body": "<path d=\"F\"/>"},
    "folder-open-outline-rounded-real": {"body": "<path d=\"O\"/>", "left": 1, "top": 2, "width": 20, "height": 21},
    "inventory-2-rounded": {"body": "<path d=\"I\"/>"},
    "inventory-2-outline-rounded": {"body": "<path d=\"J\"/>"},
    "turned-rounded": {"body": "<path d=\"T\"/>", "hFlip": true},
    "turned-outline-rounded": {"body": "<path d=\"U\"/>"}
  },
  "aliases": {
    "folder-open-outline-rounded": {"parent": "folder-open-outline-rounded-real"},
    "twice-rounded": {"parent": "home-rounded"},
    "flipped-rounded": {"parent": "home-rounded", "hFlip": true}
  },
  "width": 24,
  "height": 24
}`

// packageOf is the tarball of an npm package that holds icons.json.
func packageOf(t *testing.T, icons string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	zipped := gzip.NewWriter(&buffer)
	archive := tar.NewWriter(zipped)
	for name, content := range map[string]string{"package/package.json": `{"name":"@iconify-json/material-symbols"}`, "package/icons.json": icons} {
		if err := archive.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(content))}); err != nil {
			t.Fatal(err)
		}
		if _, err := archive.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := zipped.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func sourceOf(data []byte) iconSource {
	sum := sha512.Sum512(data)
	return iconSource{Package: "@iconify-json/material-symbols", Version: "1.2.97", SHA512: hex.EncodeToString(sum[:])}
}

// registry serves what answer says at the address of the package of the source,
// and counts the downloads (the requests that get the file).
func registry(t *testing.T, source iconSource, answer func(requests int32, w http.ResponseWriter)) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/" + source.Package + "/-/" + source.tarballName()
		if r.URL.Path != want {
			http.Error(w, "not found: "+r.URL.Path, http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			answer(0, w) // The downloader asks for the head of a file first.
			return
		}
		answer(requests.Add(1), w)
	}))
	t.Cleanup(server.Close)
	return server, &requests
}

func TestEnsureTarballDownloadsChecksAndKeeps(t *testing.T) {
	data := packageOf(t, fakeIcons)
	source := sourceOf(data)
	server, requests := registry(t, source, func(_ int32, w http.ResponseWriter) { _, _ = w.Write(data) })
	dir := filepath.Join(t.TempDir(), "cache", "icons")
	file, err := ensureTarball(context.Background(), source, dir, server.URL, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if file != filepath.Join(dir, "material-symbols-1.2.97.tgz") {
		t.Errorf("the tarball is %s", file)
	}
	if got, _ := os.ReadFile(file); !bytes.Equal(got, data) {
		t.Error("the tarball is not what the registry sent")
	}
	// A tarball in the cache is used as it is.
	if _, err := ensureTarball(context.Background(), source, dir, server.URL, io.Discard); err != nil || requests.Load() != 1 {
		t.Errorf("the second call asked the registry again: %d requests, %v", requests.Load(), err)
	}
	// One that does not match the pin is downloaded again.
	if err := os.WriteFile(file, []byte("damaged"), 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureTarball(context.Background(), source, dir, server.URL, io.Discard); err != nil || requests.Load() != 2 {
		t.Errorf("a damaged tarball was not replaced: %d requests, %v", requests.Load(), err)
	}
	if got, _ := os.ReadFile(file); !bytes.Equal(got, data) {
		t.Error("the damaged tarball is still there")
	}
	// A registry with a trailing slash, such as a mirror, serves the same path.
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if _, err := ensureTarball(context.Background(), source, dir, server.URL+"/", io.Discard); err != nil {
		t.Errorf("a registry with a slash: %v", err)
	}
}

func TestEnsureTarballRefusesWhatIsNotPinned(t *testing.T) {
	good := packageOf(t, fakeIcons)
	source := sourceOf(good)
	server, requests := registry(t, source, func(_ int32, w http.ResponseWriter) { _, _ = w.Write(packageOf(t, `{"icons": {}}`)) })
	dir := t.TempDir()
	_, err := ensureTarball(context.Background(), source, dir, server.URL, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "material-symbols-1.2.97.tgz") {
		t.Fatalf("got %v, want a failure that names the package", err)
	}
	if requests.Load() != 1 {
		t.Errorf("a file of the wrong hash was fetched %d times; the same file would come again", requests.Load())
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("left in the cache: %v", entries)
	}
}

func TestEnsureTarballTriesAgainAfterAServerError(t *testing.T) {
	retryPause = func(int) time.Duration { return 0 }
	t.Cleanup(func() {
		retryPause = func(attempt int) time.Duration { return time.Duration(attempt) * 2 * time.Second }
	})
	data := packageOf(t, fakeIcons)
	source := sourceOf(data)
	server, requests := registry(t, source, func(n int32, w http.ResponseWriter) {
		if n < downloadAttempts {
			http.Error(w, "busy", http.StatusBadGateway)
			return
		}
		_, _ = w.Write(data)
	})
	if _, err := ensureTarball(context.Background(), source, t.TempDir(), server.URL, io.Discard); err != nil {
		t.Fatalf("the last attempt works: %v", err)
	}
	if requests.Load() != downloadAttempts {
		t.Errorf("%d requests", requests.Load())
	}
	// A registry that never answers well gives up, and says how to get round it.
	always, _ := registry(t, source, func(_ int32, w http.ResponseWriter) { http.Error(w, "gone", http.StatusInternalServerError) })
	_, err := ensureTarball(context.Background(), source, t.TempDir(), always.URL, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "VFOX_PLUS_NPM_REGISTRY") || !strings.Contains(err.Error(), "VFOX_PLUS_ICONS") {
		t.Errorf("got %v, want the two ways round it", err)
	}
}

func TestIconCacheDir(t *testing.T) {
	env := func(values map[string]string) func(string) string {
		return func(name string) string { return values[name] }
	}
	userCache := func() (string, error) { return "/home/u/.cache", nil }
	if dir, err := iconCacheDir(env(map[string]string{"VFOX_PLUS_ICONS": "/seeded"}), userCache); err != nil || dir != "/seeded" {
		t.Errorf("a directory that was named: %q, %v", dir, err)
	}
	if dir, err := iconCacheDir(env(nil), userCache); err != nil || dir != filepath.Join("/home/u/.cache", "vfox-plus", "icons") {
		t.Errorf("the cache of the user: %q, %v", dir, err)
	}
	if _, err := iconCacheDir(env(nil), func() (string, error) { return "", errors.New("no $HOME") }); err == nil || !strings.Contains(err.Error(), "VFOX_PLUS_ICONS") {
		t.Errorf("got %v, want the way round it", err)
	}
}

func TestReadIconSet(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "material-symbols-1.2.97.tgz")
	if err := os.WriteFile(file, packageOf(t, fakeIcons), 0o666); err != nil {
		t.Fatal(err)
	}
	set, err := readIconSet(file)
	if err != nil || len(set.Icons) != 8 || len(set.Aliases) != 3 || set.Width != 24 {
		t.Fatalf("%+v, %v", set, err)
	}
	if err := os.WriteFile(file, []byte("not a tarball"), 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := readIconSet(file); err == nil {
		t.Error("a file that is no tarball")
	}
	empty := filepath.Join(dir, "empty.tgz")
	var buffer bytes.Buffer
	zipped := gzip.NewWriter(&buffer)
	_ = tar.NewWriter(zipped).Close()
	_ = zipped.Close()
	if err := os.WriteFile(empty, buffer.Bytes(), 0o666); err != nil {
		t.Fatal(err)
	}
	if _, err := readIconSet(empty); err == nil || !strings.Contains(err.Error(), "package/icons.json") {
		t.Errorf("got %v, want the file that is missing", err)
	}
}

func TestIconNamesAndFiles(t *testing.T) {
	if outlined, filled := iconNames("folder_open"); outlined != "folder-open-outline-rounded" || filled != "folder-open-rounded" {
		t.Errorf("%s, %s", outlined, filled)
	}
	set, err := readIconSet(writeTarball(t, fakeIcons))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := writeIcons(set, []string{"home", "folder_open", "inventory_2"}, dir); err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		t.Helper()
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}
	if got := read("home.svg"); got != `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="24" height="24"><path fill="currentColor" d="M2 2h2z"/></svg>`+"\n" {
		t.Errorf("home.svg: %q", got)
	}
	if got := read("home-fill.svg"); !strings.Contains(got, `d="M1 1h2z"`) {
		t.Errorf("home-fill.svg is the filled one: %q", got)
	}
	// An outlined icon that is an alias of an icon with a box of its own.
	if got := read("folder_open.svg"); !strings.Contains(got, `viewBox="1 2 20 21" width="20" height="21"`) || !strings.Contains(got, `d="O"`) {
		t.Errorf("folder_open.svg: %q", got)
	}
	if got := read("inventory_2-fill.svg"); !strings.Contains(got, `d="I"`) {
		t.Errorf("inventory_2-fill.svg: %q", got)
	}
}

func TestWriteIconsNamesEveryIconThatIsMissing(t *testing.T) {
	set, err := readIconSet(writeTarball(t, fakeIcons))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	err = writeIcons(set, []string{"home", "nowhere", "turned"}, dir)
	if err == nil {
		t.Fatal("icons that the package lacks")
	}
	for _, want := range []string{"nowhere: the package has no icon nowhere-outline-rounded", "nowhere: the package has no icon nowhere-rounded", "turned: turned-rounded is transformed"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the error does not say %q: %v", want, err)
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Errorf("files were written although icons are missing: %v", entries)
	}
	// An alias that transforms its parent is refused, a plain alias is followed.
	if _, err := set.find("flipped-rounded"); err == nil || !strings.Contains(err.Error(), "transforms") {
		t.Errorf("got %v", err)
	}
	if icon, err := set.find("twice-rounded"); err != nil || !strings.Contains(icon.Body, "M1 1h2z") {
		t.Errorf("%v, %v", icon, err)
	}
}

// writeTarball writes the package of the tests to a file.
func writeTarball(t *testing.T, icons string) string {
	t.Helper()
	file := filepath.Join(t.TempDir(), "package.tgz")
	if err := os.WriteFile(file, packageOf(t, icons), 0o666); err != nil {
		t.Fatal(err)
	}
	return file
}

func TestReadManifest(t *testing.T) {
	dir := t.TempDir()
	write := func(content string) string {
		file := filepath.Join(dir, "icons.json")
		if err := os.WriteFile(file, []byte(content), 0o666); err != nil {
			t.Fatal(err)
		}
		return file
	}
	hash := strings.Repeat("ab", sha512.Size)
	good := `{"source": {"package": "@a/b", "version": "1.0.0", "sha512": "` + hash + `"}, "icons": {"home": "home", "back": "arrow_back", "also": "home"}}`
	manifest, err := readManifest(write(good))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(manifest.symbols(), " "); got != "arrow_back home" {
		t.Errorf("the symbols, each once: %q", got)
	}
	for name, content := range map[string]string{
		"no source":   `{"icons": {"home": "home"}}`,
		"no hash":     `{"source": {"package": "@a/b", "version": "1.0.0"}, "icons": {"home": "home"}}`,
		"short hash":  `{"source": {"package": "@a/b", "version": "1.0.0", "sha512": "abcd"}, "icons": {"home": "home"}}`,
		"no icons":    `{"source": {"package": "@a/b", "version": "1.0.0", "sha512": "` + hash + `"}, "icons": {}}`,
		"not a table": `{`,
	} {
		if _, err := readManifest(write(content)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	if _, err := readManifest(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("a file that is not there")
	}
}

// The manifest of the repository is one that prepare can use, and every symbol of
// it has the shape of a symbol of Google's.
func TestTheManifestOfTheRepository(t *testing.T) {
	manifest, err := readManifest(filepath.Join("..", "..", "..", "..", "shared", "icons.json"))
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Source.Package != "@iconify-json/material-symbols" || manifest.Source.tarballName() != "material-symbols-"+manifest.Source.Version+".tgz" {
		t.Errorf("the source: %+v", manifest.Source)
	}
	for meaning, symbol := range manifest.Icons {
		if symbol == "" || strings.ToLower(symbol) != symbol || strings.ContainsAny(symbol, " -") {
			t.Errorf("%s: %q is not a symbol of Material Symbols (lower case, with underscores)", meaning, symbol)
		}
	}
}
