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

//go:build linux

package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/assets"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

func TestTheIconFontsGoToADirectoryOfTheirOwnWhenTheCacheCannotBeWritten(t *testing.T) {
	// A directory cannot be made below a file, whoever runs the test.
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("XDG_CACHE_HOME", filepath.Join(blocker, "cache"))
	read := func(name string) []byte {
		content, err := assets.Files.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		return content
	}
	loadIcons(read("fonts/MaterialSymbolsRounded.ttf"), read("fonts/MaterialSymbolsRoundedFilled.ttf"), read("icons.json"))
	cache := os.Getenv("XDG_CACHE_HOME")
	t.Cleanup(func() { _ = os.RemoveAll(cache) }) //nolint:gosec // The directory that the code under test made.
	if strings.HasPrefix(cache, blocker) {
		t.Fatalf("the cache directory is still below the file: %s", cache)
	}
	fonts, err := filepath.Glob(filepath.Join(cache, "mygo", "fonts", "*.font"))
	if err != nil || len(fonts) != 2 {
		t.Errorf("the two icon fonts are in the new directory: %v (%v)", fonts, err)
	}
	if !kit.HasIcon("home") {
		t.Error("the interface knows its icons")
	}
}
