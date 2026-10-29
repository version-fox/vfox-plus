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
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseArgs(t *testing.T) {
	cfg, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("parseArgs(nil): %v", err)
	}
	if cfg.assets != "" || cfg.root != "" {
		t.Errorf("defaults = %+v, want empty", cfg)
	}

	cfg, err = parseArgs([]string{"--assets", "/a", "--root", "/r"})
	if err != nil {
		t.Fatalf("parseArgs flags: %v", err)
	}
	if cfg.assets != "/a" || cfg.root != "/r" {
		t.Errorf("parsed = %+v, want assets=/a root=/r", cfg)
	}

	if _, err := parseArgs([]string{"extra"}); err == nil {
		t.Error("parseArgs(positional) = nil, want usage error")
	}
}

func TestResolveAssetsDir(t *testing.T) {
	if got := resolveAssetsDir("/flag", "/root"); got != "/flag" {
		t.Errorf("flag = %q, want /flag", got)
	}
	want := filepath.Join("/root", "assets")
	if got := resolveAssetsDir("", "/root"); got != want {
		t.Errorf("default = %q, want %q", got, want)
	}
}

func TestCopyFiles(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	for name, content := range map[string]string{"add.svg": "<svg/>", "add-fill.svg": "<svg fill/>", "LICENSE": "license"} {
		if err := os.WriteFile(filepath.Join(src, name), []byte(content), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	if err := copyFiles(src, "*.svg", dst); err != nil {
		t.Fatalf("copyFiles: %v", err)
	}
	entries, err := os.ReadDir(dst)
	if err != nil || len(entries) != 2 {
		t.Fatalf("copied %v (%v), want the two SVGs and not the LICENSE", entries, err)
	}
	if data, _ := os.ReadFile(filepath.Join(dst, "add-fill.svg")); string(data) != "<svg fill/>" {
		t.Errorf("add-fill.svg = %q", data)
	}
	if err := copyFiles(src, "*.arb", dst); err == nil {
		t.Error("copyFiles of nothing = nil, want an error: an empty directory would fail the embedding later")
	}
}

func TestWriteLicenses(t *testing.T) {
	dir := t.TempDir()
	notice := filepath.Join(dir, "NOTICE.txt")
	license := filepath.Join(dir, "LICENSE")
	output := filepath.Join(dir, "assets", "licenses.txt")
	if err := os.WriteFile(notice, []byte("mygo notices\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(license, []byte("Apache License text"), 0o666); err != nil {
		t.Fatal(err)
	}
	source := iconSource{Package: "@iconify-json/material-symbols", Version: "1.2.97"}
	if err := writeLicenses(notice, license, source, output); err != nil {
		t.Fatalf("writeLicenses: %v", err)
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.HasPrefix(text, "mygo notices\n") {
		t.Errorf("licenses do not start with NOTICE.txt:\n%s", text)
	}
	for _, want := range []string{"Material Symbols Rounded (the icons)", "@iconify-json/material-symbols 1.2.97", "https://github.com/google/material-design-icons"} {
		if !strings.Contains(text, want) {
			t.Errorf("licenses miss %q:\n%s", want, text)
		}
	}
	if !strings.HasSuffix(text, "Apache License text\n") {
		t.Errorf("licenses do not end with the license text and a newline: %q", text)
	}
}

// Run collects the assets of a mygo module from a repository of the shape of this
// one, with the icons from a registry that serves the tarball of the manifest.
func TestRunCollectsTheAssets(t *testing.T) {
	data := packageOf(t, fakeIcons)
	source := sourceOf(data)
	server, requests := registry(t, source, func(_ int32, w http.ResponseWriter) { _, _ = w.Write(data) })

	repo := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		file := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	write("mygo/go.mod", "module github.com/version-fox/vfox-plus/mygo\n\ngo 1.27\n")
	write("mygo/NOTICE.txt", "notices\n")
	write("LICENSE", "Apache License text\n")
	write("shared/i18n/app_en.arb", "{}")
	write("shared/icons.json", `{"source": {"package": "`+source.Package+`", "version": "`+source.Version+`", "sha512": "`+source.SHA512+`"},
		"icons": {"home": "home", "folder": "folder_open"}}`)
	cache := filepath.Join(t.TempDir(), "cache")
	t.Setenv("VFOX_PLUS_ICONS", cache)
	t.Setenv("VFOX_PLUS_NPM_REGISTRY", server.URL)
	t.Setenv("MYGO_SKIP_LICENSES", "1")

	// --fetch only fills the cache.
	if err := Run([]string{"--fetch", "--root", filepath.Join(repo, "mygo")}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cache, "material-symbols-1.2.97.tgz")); err != nil {
		t.Errorf("--fetch left nothing in the cache: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repo, "mygo", "assets")); err == nil {
		t.Error("--fetch made the assets")
	}

	// Without the network, the rest follows from the cache.
	server.Close()
	if err := Run([]string{"--root", filepath.Join(repo, "mygo")}); err != nil {
		t.Fatal(err)
	}
	if requests.Load() != 1 {
		t.Errorf("%d requests", requests.Load())
	}
	for _, name := range []string{"icons/home.svg", "icons/home-fill.svg", "icons/folder_open.svg", "icons/folder_open-fill.svg", "icons.json", "i18n/app_en.arb", "licenses.txt"} {
		if _, err := os.Stat(filepath.Join(repo, "mygo", "assets", filepath.FromSlash(name))); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
	licenses, _ := os.ReadFile(filepath.Join(repo, "mygo", "assets", "licenses.txt"))
	if !strings.Contains(string(licenses), "Apache License text") || !strings.HasPrefix(string(licenses), "notices\n") {
		t.Errorf("licenses: %q", licenses)
	}
}
