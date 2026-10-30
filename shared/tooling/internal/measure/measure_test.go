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

package measure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
)

func TestResolveFakeVfox(t *testing.T) {
	fake := filepath.Join(t.TempDir(), "fake")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := appsession.FindRepoRoot()
	if err != nil {
		t.Fatal(err)
	}

	cases := map[string]struct {
		fixtures, fake string
		wantFake       string
		wantFixtures   string
	}{
		"nothing given, the repository":           {"", "", "", appsession.FixturesDir(root)},
		"fixtures of another version":             {"/x/vfox-1.0.12", "", "", "/x/vfox-1.0.12"},
		"fixtures given as their directory":       {"/x/fixtures", "", "", "/x/fixtures/vfox-1.0.13"},
		"a fake vfox, fixtures of the repository": {"", fake, fake, appsession.FixturesDir(root)},
		"a fake vfox and fixtures":                {"/x/fixtures", fake, fake, "/x/fixtures/vfox-1.0.13"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			gotFake, gotFixtures, err := resolveFakeVfox(c.fixtures, c.fake)
			if err != nil || gotFake != c.wantFake || gotFixtures != c.wantFixtures {
				t.Fatalf("resolveFakeVfox(%q, %q) = %q, %q, %v; want %q, %q", c.fixtures, c.fake, gotFake, gotFixtures, err, c.wantFake, c.wantFixtures)
			}
		})
	}

	if _, _, err := resolveFakeVfox("", filepath.Join(t.TempDir(), "missing")); err == nil || !strings.Contains(err.Error(), "fake vfox not found") {
		t.Errorf("a fake vfox that is not there: %v", err)
	}

	// Away from a repository, the fixtures lie next to the fake vfox, or must be named.
	t.Chdir(t.TempDir())
	if _, err := appsession.FindRepoRoot(); err == nil {
		return
	}
	if gotFake, gotFixtures, err := resolveFakeVfox("", fake); err != nil || gotFake != fake || gotFixtures != filepath.Join(filepath.Dir(fake), "vfox-1.0.13") {
		t.Errorf("a fake vfox away from the repository = %q, %q, %v", gotFake, gotFixtures, err)
	}
	if _, _, err := resolveFakeVfox("", ""); err == nil || !strings.Contains(err.Error(), "pass --fake-vfox or --fixtures-root") {
		t.Errorf("nothing away from the repository: %v", err)
	}
}

func TestRunValidatesItsArgumentsAndPreparesTheOutputBeforeStarting(t *testing.T) {
	if err := Run(nil); err == nil || !strings.Contains(err.Error(), "Usage: measure") {
		t.Errorf("without an application: %v", err)
	}
	if err := Run([]string{"--no-such-flag", "application"}); err == nil {
		t.Error("an unknown flag was accepted")
	}
	if err := Run([]string{"--fake-vfox", filepath.Join(t.TempDir(), "missing"), "application"}); err == nil || !strings.Contains(err.Error(), "fake vfox not found") {
		t.Errorf("a fake vfox that is not there: %v", err)
	}

	// The file with the numbers starts empty, however old it is, so that a run that
	// fails leaves nothing of an earlier one; it is named by a flag or by the
	// environment.
	t.Setenv("TMPDIR", t.TempDir())
	fake := filepath.Join(t.TempDir(), "fake")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	fromFlag, fromEnvironment := filepath.Join(dir, "a", "flag.txt"), filepath.Join(dir, "environment.txt")
	if err := os.WriteFile(fromEnvironment, []byte("memory (home): stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VFOX_PLUS_MEASURE_OUTPUT", fromEnvironment)
	for _, args := range [][]string{
		{"--output", fromFlag, "--fake-vfox", fake, "/nonexistent/application"},
		{"--fake-vfox", fake, "/nonexistent/application"},
	} {
		if err := Run(args); err == nil || !strings.Contains(err.Error(), "start the application") {
			t.Fatalf("Run(%q) = %v, want a failure to start the application", args, err)
		}
	}
	for _, path := range []string{fromFlag, fromEnvironment} {
		if data, err := os.ReadFile(path); err != nil || len(data) != 0 {
			t.Errorf("%s = %q, %v; want it made, and empty", path, data, err)
		}
	}
}

func TestMemoryOfThisProcess(t *testing.T) {
	if _, err := os.Stat("/proc/self/smaps_rollup"); err != nil {
		t.Skip("no /proc/<pid>/smaps_rollup on this platform")
	}
	text := MemoryOf(os.Getpid())
	if !strings.HasPrefix(text, "RSS ") || !strings.Contains(text, " MB, PSS ") || !strings.HasSuffix(text, " processes") {
		t.Fatalf("MemoryOf = %q", text)
	}
}

func TestFixturesDirFromRoot(t *testing.T) {
	cases := map[string]string{
		"shared/fixtures":             "shared/fixtures/vfox-1.0.13",
		"shared/fixtures/vfox-1.0.13": "shared/fixtures/vfox-1.0.13",
		"/x/vfox-1.0.12":              "/x/vfox-1.0.12",
		"/x/fixtures":                 "/x/fixtures/vfox-1.0.13",
	}
	for root, want := range cases {
		if got := fixturesDirFromRoot(root); got != want {
			t.Errorf("fixturesDirFromRoot(%q) = %q, want %q", root, got, want)
		}
	}
}
