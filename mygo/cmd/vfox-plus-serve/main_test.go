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

package main

import (
	"os/exec"
	"strings"
	"testing"
)

// The headless binary exists to stay small: it must never link the mygo
// user interface framework (nor the ui package built on it), which only the
// desktop application needs.
func TestHeadlessBinaryLinksNoInterfaceFramework(t *testing.T) {
	out, err := exec.Command("go", "list", "-deps", ".").Output()
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, dep := range strings.Fields(string(out)) {
		if strings.Contains(dep, "github.com/egoist/mygo") || strings.HasSuffix(dep, "/mygo/ui") || strings.Contains(dep, "/mygo/ui/") {
			t.Fatalf("headless binary links the interface: %s", dep)
		}
		if dep == "github.com/version-fox/vfox-plus/mygo/ui" {
			t.Fatalf("headless binary links the desktop interface: %s", dep)
		}
	}
}

func TestHeadlessUsage(t *testing.T) {
	if got := run(nil); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
	if got := run([]string{"bogus"}); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
	if got := run([]string{"help"}); got != 0 {
		t.Fatalf("got %d, want 0", got)
	}
}
