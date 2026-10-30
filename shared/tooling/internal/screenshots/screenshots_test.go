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

package screenshots

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunsAreNamedAndAskForOneKindOfMachine(t *testing.T) {
	names := map[string]bool{}
	for _, r := range runs {
		if r.name == "" || r.scenario == nil || names[r.name] {
			t.Errorf("run %+v is unnamed, without a scenario or repeated", r)
		}
		names[r.name] = true
		inventories := 0
		for _, assignment := range r.env {
			if !strings.Contains(assignment, "=") {
				t.Errorf("%s: %q is no NAME=value", r.name, assignment)
			}
			if strings.HasPrefix(assignment, "VFOX_PLUS_FAKE_DELAY=") || strings.HasPrefix(assignment, "VFOX_PLUS_FAKE_INVENTORY=") {
				inventories++
			}
		}
		if inventories > 1 {
			t.Errorf("%s: more than one thing that the fake vfox plays", r.name)
		}
	}
}

func TestRunFailsWhenTheApplicationCannotBeStarted(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	out := filepath.Join(t.TempDir(), "new", "screenshots")
	if err := Run([]string{out, "/nonexistent/application"}); err == nil || !strings.Contains(err.Error(), "start the application") {
		t.Fatalf("got %v, want a failure to start the application", err)
	}
	if info, err := os.Stat(out); err != nil || !info.IsDir() {
		t.Errorf("the directory of the screenshots is not made: %v", err)
	}
}

func TestRunWithoutAnApplication(t *testing.T) {
	if err := Run([]string{t.TempDir()}); err == nil || !strings.Contains(err.Error(), "Usage") {
		t.Fatalf("Run without an application = %v, want the usage", err)
	}
}
