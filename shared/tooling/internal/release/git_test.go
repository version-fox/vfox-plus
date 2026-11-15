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

package release

import (
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

func TestGitReadsTagsAndFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	git("init", "-q")
	git("-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false",
		"commit", "-q", "--allow-empty", "-m", "init")
	git("tag", "v0.1.0")
	git("tag", "vendor")
	writeFile(t, dir, VersionFile, "0.1.0\n")

	repo := Git{Dir: dir}
	if ok, err := repo.TagExists("v0.1.0"); err != nil || !ok {
		t.Errorf("TagExists(v0.1.0) = %t, %v; want true", ok, err)
	}
	if ok, err := repo.TagExists("v0.2.0"); err != nil || ok {
		t.Errorf("TagExists(v0.2.0) = %t, %v; want false", ok, err)
	}
	tags, err := repo.Tags()
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(tags)
	if !slices.Equal(tags, []string{"v0.1.0", "vendor"}) {
		t.Errorf("Tags = %v, want [v0.1.0 vendor]", tags)
	}
	text, err := repo.ReadFile(filepath.ToSlash(VersionFile))
	if err != nil || text != "0.1.0\n" {
		t.Errorf("ReadFile = %q, %v; want 0.1.0 with a newline", text, err)
	}
}
