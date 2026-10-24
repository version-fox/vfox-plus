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

package runtask

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidTask(t *testing.T) {
	for _, task := range []string{
		"deps",
		"check",
		"build",
		"e2e",
		"measure",
		"screenshots",
		"package",
		"install-test",
		"shell",
	} {
		if !ValidTask(task) {
			t.Errorf("ValidTask(%q) = false, want true", task)
		}
	}
	for _, task := range []string{"", "test", "Check", "install_test", "all"} {
		if ValidTask(task) {
			t.Errorf("ValidTask(%q) = true, want false", task)
		}
	}
}

func TestLoadEnv(t *testing.T) {
	e := loadEnv()
	if e.src == "" || e.out == "" || e.work == "" {
		t.Fatalf("loadEnv has empty fields: %+v", e)
	}
	t.Setenv("SRC", "/r")
	t.Setenv("OUT", "/o")
	t.Setenv("WORK", "/w")
	e = loadEnv()
	if e.src != "/r" || e.out != "/o" || e.work != "/w" {
		t.Fatalf("loadEnv ignores the environment: %+v", e)
	}
	_ = os.Unsetenv("SRC")
}

func TestFindWallpaper(t *testing.T) {
	dir := t.TempDir()
	if _, err := findWallpaper(dir); err == nil {
		t.Errorf("findWallpaper(%q) = nil error, want the missing-file error", dir)
	} else if got := err.Error(); !strings.Contains(got, "warty-final-ubuntu.png") || !strings.Contains(got, "ubuntu-wallpapers") {
		t.Errorf("findWallpaper error = %q, want the file name and the package", got)
	}
	name := filepath.Join(dir, "warty-final-ubuntu.png")
	if err := os.WriteFile(name, []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := findWallpaper(dir); err != nil || got != name {
		t.Errorf("findWallpaper(%q) = %q, %v, want %q, nil", dir, got, err, name)
	}
}

// testerHead must end with setting HOME, and must only use runuser when the
// process does not already run as tester: only root may use runuser, so a
// nested runuser (tester through tester) fails.
func TestTesterHead(t *testing.T) {
	head := testerHead()
	if len(head) < 2 || head[len(head)-2] != "env" || head[len(head)-1] != "HOME=/home/tester" {
		t.Fatalf("testerHead() = %q, want [... env HOME=/home/tester]", head)
	}
	if alreadyTester() && head[0] != "env" {
		t.Errorf("already tester, but testerHead() starts with %q, want direct env", head[0])
	}
	if !alreadyTester() && head[0] != "runuser" {
		t.Errorf("not tester, but testerHead() starts with %q, want runuser", head[0])
	}
}

func TestDependsComplete(t *testing.T) {
	// The installation test checks these against ldd on Ubuntu 26.04; the
	// constant must stay a comma-separated list with the fonts alternative.
	for _, dep := range []string{"libc6", "libgtk-3-0t64", "fonts-ubuntu | fonts-noto-core"} {
		found := false
		for _, part := range splitList(Depends) {
			if part == dep {
				found = true
			}
		}
		if !found {
			t.Errorf("Depends %q misses %q", Depends, dep)
		}
	}
	if Recommends == "" {
		t.Error("Recommends must not be empty")
	}
}

func splitList(s string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			part := s[start:i]
			for len(part) > 0 && (part[0] == ' ' || part[0] == '\t') {
				part = part[1:]
			}
			for len(part) > 0 && (part[len(part)-1] == ' ' || part[len(part)-1] == '\t') {
				part = part[:len(part)-1]
			}
			parts = append(parts, part)
			start = i + 1
		}
	}
	return parts
}
