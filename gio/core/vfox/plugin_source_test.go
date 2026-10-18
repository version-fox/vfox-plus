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

package vfox_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/gio/core/vfox"
)

func TestOnlyLocalLuaFilesAreMoved(t *testing.T) {
	if !vfox.VfoxMovesPluginSource("/home/u/plugin.lua") || !vfox.VfoxMovesPluginSource("/home/u/PLUGIN.LUA") {
		t.Error("local .lua files must be copied")
	}
	if vfox.VfoxMovesPluginSource("/home/u/plugin.zip") || vfox.VfoxMovesPluginSource("https://example.com/plugin.lua") {
		t.Error("other sources must be passed through")
	}
	if !vfox.IsRemotePluginSource("http://127.0.0.1:8080/p.zip") {
		t.Error("expected a URL")
	}
}

func TestTemporaryDirectory(t *testing.T) {
	home := t.TempDir()
	if got := vfox.VfoxTemporaryDirectory(map[string]string{"HOME": home}); got != home+"/.vfox/tmp" {
		t.Errorf("got %q", got)
	}
	if err := os.MkdirAll(home+"/.version-fox", 0o750); err != nil {
		t.Fatal(err)
	}
	if got := vfox.VfoxTemporaryDirectory(map[string]string{"HOME": home}); got != home+"/.version-fox/tmp" {
		t.Errorf("got %q", got)
	}
	if got := vfox.VfoxTemporaryDirectory(map[string]string{}); got == "" || !filepath.IsAbs(got) {
		t.Errorf("got %q", got)
	}
}

func TestUrlsAndArchivesArePassedThrough(t *testing.T) {
	root := t.TempDir()
	for _, source := range []string{"https://example.com/p.lua", filepath.Join(root, "p.zip")} {
		prepared := vfox.PreparePluginSource(source, filepath.Join(root, "tmp"))
		if prepared.Path != source || prepared.IsCopy() {
			t.Errorf("got %#v", prepared)
		}
		prepared.Dispose()
	}
}

func TestLocalLuaFileIsCopied(t *testing.T) {
	root := t.TempDir()
	original := filepath.Join(root, "mine.lua")
	if err := os.WriteFile(original, []byte("PLUGIN = {}"), 0o600); err != nil {
		t.Fatal(err)
	}
	temporary := filepath.Join(root, ".vfox", "tmp")
	prepared := vfox.PreparePluginSource(original, temporary)
	if !prepared.IsCopy() || prepared.Path == original || !strings.HasSuffix(prepared.Path, "/mine.lua") {
		t.Fatalf("got %#v", prepared)
	}
	content, err := os.ReadFile(prepared.Path)
	if err != nil || string(content) != "PLUGIN = {}" {
		t.Fatalf("got %q %v", content, err)
	}
	prepared.Dispose()
	if _, err := os.Stat(prepared.Path); !os.IsNotExist(err) {
		t.Error("the copy must be deleted")
	}
	if _, err := os.Stat(original); err != nil {
		t.Error("the original must survive")
	}
	leftovers, err := os.ReadDir(temporary)
	if err == nil && len(leftovers) != 0 {
		t.Errorf("no directories may be left: %v", leftovers)
	}
}

func TestUncopyableFileIsLeftForVfox(t *testing.T) {
	root := t.TempDir()
	temporary := filepath.Join(root, "tmp")
	source := filepath.Join(root, "missing.lua")
	prepared := vfox.PreparePluginSource(source, temporary)
	if prepared.Path != source || prepared.IsCopy() {
		t.Errorf("got %#v", prepared)
	}
	if _, err := os.Stat(temporary); !os.IsNotExist(err) {
		entries, _ := os.ReadDir(temporary)
		if len(entries) != 0 {
			t.Errorf("got %v", entries)
		}
	}
}
