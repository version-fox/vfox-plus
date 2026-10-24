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

package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseArgs(t *testing.T) {
	cfg, err := parseArgs([]string{"dist/mygo"})
	if err != nil {
		t.Fatalf("parseArgs: %v", err)
	}
	if cfg.e2e {
		t.Error("e2e = true, want false")
	}
	if cfg.output != "dist/mygo" {
		t.Errorf("output = %q, want %q", cfg.output, "dist/mygo")
	}
	if got := cfg.tags(); got != "mygo_noinspector" {
		t.Errorf("tags = %q, want %q", got, "mygo_noinspector")
	}

	cfg, err = parseArgs([]string{"--e2e", "dist/mygo"})
	if err != nil {
		t.Fatalf("parseArgs --e2e: %v", err)
	}
	if !cfg.e2e {
		t.Error("e2e = false, want true")
	}
	if got := cfg.tags(); got != "mygo_noinspector,e2e" {
		t.Errorf("tags = %q, want %q", got, "mygo_noinspector,e2e")
	}

	cfg, err = parseArgs([]string{"--e2e", "--root", "/tmp/mygo", "dist/mygo"})
	if err != nil {
		t.Fatalf("parseArgs --root: %v", err)
	}
	if cfg.root != "/tmp/mygo" {
		t.Errorf("root = %q, want %q", cfg.root, "/tmp/mygo")
	}

	if _, err := parseArgs(nil); err == nil {
		t.Error("parseArgs(nil) = nil, want usage error")
	}
	if _, err := parseArgs([]string{"--e2e"}); err == nil {
		t.Error("parseArgs(--e2e) = nil, want usage error")
	}
	if _, err := parseArgs([]string{"a", "b"}); err == nil {
		t.Error("parseArgs two outputs = nil, want usage error")
	}
}

func TestResolveOutput(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatal(err)
		}
	}()
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}

	got, err := resolveOutput("dist/mygo")
	if err != nil {
		t.Fatalf("resolveOutput: %v", err)
	}
	want := filepath.Join(tmp, "dist", "mygo")
	if got != want {
		t.Errorf("resolveOutput = %q, want %q", got, want)
	}
	if fi, err := os.Stat(filepath.Join(tmp, "dist")); err != nil || !fi.IsDir() {
		t.Errorf("output parent not created: %v", err)
	}
	if !filepath.IsAbs(got) {
		t.Errorf("resolveOutput = %q, want absolute path", got)
	}

	if _, err := resolveOutput(""); err == nil {
		t.Error("resolveOutput(\"\") = nil, want error")
	}
}

func TestLocateMygoRoot(t *testing.T) {
	tmp := t.TempDir()
	mygo := filepath.Join(tmp, "mygo")
	if err := os.MkdirAll(mygo, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mygo, "go.mod"), []byte("module github.com/version-fox/vfox-plus/mygo\n\ngo 1.27.1\n"), 0o666); err != nil {
		t.Fatal(err)
	}
	// A tooling checkout next to mygo must not be mistaken for mygo.
	tooling := filepath.Join(tmp, "shared", "tooling")
	if err := os.MkdirAll(tooling, 0o777); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tooling, "go.mod"), []byte("module github.com/version-fox/vfox-plus/shared/tooling\n\ngo 1.27.1\n"), 0o666); err != nil {
		t.Fatal(err)
	}

	if got, ok := locateMygoRoot(tooling); !ok || got != mygo {
		t.Errorf("locateMygoRoot(tooling) = %q, %v; want %q, true", got, ok, mygo)
	}
	if got, ok := locateMygoRoot(mygo); !ok || got != mygo {
		t.Errorf("locateMygoRoot(mygo) = %q, %v; want %q, true", got, ok, mygo)
	}
	if _, ok := locateMygoRoot(t.TempDir()); ok {
		t.Error("locateMygoRoot(empty) = true, want false")
	}
}
