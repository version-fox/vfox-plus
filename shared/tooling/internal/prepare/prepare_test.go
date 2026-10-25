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

func fakeLookup(env map[string]string) func(string) string {
	return func(k string) string { return env[k] }
}

func TestParseArgs(t *testing.T) {
	cfg, err := parseArgs(nil)
	if err != nil {
		t.Fatalf("parseArgs(nil): %v", err)
	}
	if cfg.symbols != "" || cfg.assets != "" || cfg.root != "" {
		t.Errorf("defaults = %+v, want empty", cfg)
	}

	cfg, err = parseArgs([]string{"--symbols", "/s", "--assets", "/a", "--root", "/r"})
	if err != nil {
		t.Fatalf("parseArgs flags: %v", err)
	}
	if cfg.symbols != "/s" || cfg.assets != "/a" || cfg.root != "/r" {
		t.Errorf("parsed = %+v, want symbols=/s assets=/a root=/r", cfg)
	}

	if _, err := parseArgs([]string{"extra"}); err == nil {
		t.Error("parseArgs(positional) = nil, want usage error")
	}
}

func TestResolveSymbolsDir(t *testing.T) {
	// --symbols wins over everything.
	got := resolveSymbolsDir("/flag", fakeLookup(map[string]string{
		"VFOX_PLUS_SYMBOLS": "/env",
		"HOME":              "/home/u",
	}))
	if got != "/flag" {
		t.Errorf("flag override = %q, want /flag", got)
	}

	// $VFOX_PLUS_SYMBOLS wins over the cache default.
	got = resolveSymbolsDir("", fakeLookup(map[string]string{
		"VFOX_PLUS_SYMBOLS": "/env",
		"HOME":              "/home/u",
	}))
	if got != "/env" {
		t.Errorf("env override = %q, want /env", got)
	}

	// Otherwise the XDG cache directory.
	got = resolveSymbolsDir("", fakeLookup(map[string]string{
		"XDG_CACHE_HOME": "/xdg",
		"HOME":           "/home/u",
	}))
	want := filepath.Join("/xdg", "vfox-plus", "material-symbols")
	if got != want {
		t.Errorf("xdg default = %q, want %q", got, want)
	}

	// Otherwise $HOME/.cache.
	got = resolveSymbolsDir("", fakeLookup(map[string]string{"HOME": "/home/u"}))
	want = filepath.Join("/home/u", ".cache", "vfox-plus", "material-symbols")
	if got != want {
		t.Errorf("home default = %q, want %q", got, want)
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

func TestHasValidSymbols(t *testing.T) {
	dir := t.TempDir()
	if hasValidSymbols(dir) {
		t.Error("empty dir valid, want false")
	}
	if err := os.WriteFile(filepath.Join(dir, "LICENSE"), []byte("x"), 0o666); err != nil {
		t.Fatal(err)
	}
	if hasValidSymbols(dir) {
		t.Error("LICENSE only valid, want false")
	}
	if err := os.WriteFile(filepath.Join(dir, "MaterialSymbolsRounded[FILL,GRAD,opsz,wght].ttf"), []byte("x"), 0o666); err != nil {
		t.Fatal(err)
	}
	if hasValidSymbols(dir) {
		t.Error("LICENSE+ttf valid, want false (codepoints missing)")
	}
	if err := os.WriteFile(filepath.Join(dir, "MaterialSymbolsRounded[FILL,GRAD,opsz,wght].codepoints"), []byte("x"), 0o666); err != nil {
		t.Fatal(err)
	}
	if !hasValidSymbols(dir) {
		t.Error("full dir invalid, want true")
	}
}

func TestWriteLicenses(t *testing.T) {
	dir := t.TempDir()
	notice := filepath.Join(dir, "NOTICE.txt")
	symbols := t.TempDir()
	output := filepath.Join(dir, "assets", "licenses.txt")
	if err := os.WriteFile(notice, []byte("mygo notices\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(symbols, "LICENSE"), []byte("symbols license\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := writeLicenses(notice, filepath.Join(symbols, "LICENSE"), output); err != nil {
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
	if !strings.Contains(text, "Material Symbols Rounded (the icon fonts)") {
		t.Errorf("licenses miss the symbols heading:\n%s", text)
	}
	if !strings.HasSuffix(text, "symbols license\n\n") {
		t.Errorf("licenses do not end with symbols LICENSE plus newline: %q", text)
	}
}
