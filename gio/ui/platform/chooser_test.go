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

package platform

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileURIsBecomePaths(t *testing.T) {
	for uri, want := range map[string]string{
		"file:///home/user/My%20Plugin.lua": "/home/user/My Plugin.lua",
		"file:///tmp/a.zip":                 "/tmp/a.zip",
		"/already/a/path":                   "/already/a/path",
		"https://example.com/x":             "https://example.com/x",
		"%zz":                               "%zz",
	} {
		if got := pathOfURI(uri); got != want {
			t.Errorf("%q: got %q, want %q", uri, got, want)
		}
	}
}

// program writes an executable shell script called name into a new directory,
// and returns that directory.
func program(t *testing.T, name, script string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("#!/bin/sh\n"+script+"\n"), 0o700); err != nil { //nolint:gosec // A script of the test.
		t.Fatal(err)
	}
	return dir
}

func TestChoosingWithZenityReturnsThePathAndNothingOnCancel(t *testing.T) {
	t.Setenv("PATH", program(t, "zenity", `printf '%s\n' "$*" > "$ZENITY_ARGS"; echo /home/user/plugin.lua`))
	arguments := filepath.Join(t.TempDir(), "arguments")
	t.Setenv("ZENITY_ARGS", arguments)
	path, err := Choose(context.Background(), nil, ChooseOptions{
		Title: "Add", Initial: "/home/user", FilterLabel: "vfox plugins", Extensions: []string{"zip", "lua"},
	})
	if err != nil || path != "/home/user/plugin.lua" {
		t.Fatalf("got %q %v", path, err)
	}
	content, _ := os.ReadFile(arguments) //nolint:gosec // A file of the test.
	for _, want := range []string{"--file-selection", "--title=Add", "--file-filter=vfox plugins | *.zip *.lua", "--filename=/home/user/"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("arguments %q lack %q", content, want)
		}
	}
	directory, err := Choose(context.Background(), nil, ChooseOptions{Title: "Pick", Directory: true})
	if err != nil || directory != "/home/user/plugin.lua" {
		t.Fatalf("got %q %v", directory, err)
	}
	t.Setenv("PATH", program(t, "zenity", "exit 1"))
	if path, err := Choose(context.Background(), nil, ChooseOptions{Title: "Add"}); err != nil || path != "" {
		t.Errorf("a cancelled chooser gives nothing: %q %v", path, err)
	}
}

func TestChoosingWithKDialogWhenThereIsNoZenity(t *testing.T) {
	t.Setenv("PATH", program(t, "kdialog", `echo "$*"`))
	file, err := Choose(context.Background(), nil, ChooseOptions{Title: "Add", Initial: "/h", FilterLabel: "plugins", Extensions: []string{"lua"}})
	if err != nil || file != "--title Add --getopenfilename /h *.lua|plugins" {
		t.Errorf("got %q %v", file, err)
	}
	directory, err := Choose(context.Background(), nil, ChooseOptions{Title: "Pick", Directory: true})
	if err != nil || directory != "--title Pick --getexistingdirectory ." {
		t.Errorf("got %q %v", directory, err)
	}
}

func TestChoosingFailsWhereThereIsNoChooser(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := Choose(context.Background(), nil, ChooseOptions{Title: "Add"}); !errors.Is(err, ErrUnavailable) {
		t.Errorf("got %v", err)
	}
	// Opening a link without a browser is not an error that anyone can act on.
	OpenURL("https://vfox.dev/")
}
