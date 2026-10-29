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
	icons := t.TempDir()
	output := filepath.Join(dir, "assets", "licenses.txt")
	if err := os.WriteFile(notice, []byte("mygo notices\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(icons, "LICENSE"), []byte("icons license\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := writeLicenses(notice, filepath.Join(icons, "LICENSE"), output); err != nil {
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
	if !strings.Contains(text, "Material Symbols Rounded (the icons)") {
		t.Errorf("licenses miss the icons heading:\n%s", text)
	}
	if !strings.HasSuffix(text, "icons license\n") {
		t.Errorf("licenses do not end with the license of the icons: %q", text)
	}
}
