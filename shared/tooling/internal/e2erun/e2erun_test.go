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

package e2erun

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStreamAppReapsPromptly(t *testing.T) {
	// /bin/true exits instantly: the run must fail fast on the missing
	// "passed" line instead of polling a zombie until the deadline.
	// (With kill -0 polling an unreaped exit looks alive.)
	if _, err := os.Stat("/bin/true"); err != nil {
		t.Skip("no /bin/true on this platform")
	}
	dir := t.TempDir()
	start := time.Now()
	err := streamApp("fake", []string{"/bin/true"}, os.Environ(), dir, filepath.Join(dir, "app.log"), 60*time.Second)
	elapsed := time.Since(start)
	if err == nil || !strings.Contains(err.Error(), "did not pass") {
		t.Fatalf("streamApp(/bin/true) = %v, want a did-not-pass failure", err)
	}
	if elapsed >= 30*time.Second {
		t.Fatalf("streamApp took %v; the finished process was polled instead of reaped", elapsed)
	}
}

func TestIsValidScenario(t *testing.T) {
	valid := []string{"fake", "real", "measure"}
	for _, s := range valid {
		if !IsValidScenario(s) {
			t.Fatalf("IsValidScenario(%q)=false, want true", s)
		}
		if err := ValidateScenario(s); err != nil {
			t.Fatalf("ValidateScenario(%q) returned %v, want nil", s, err)
		}
	}
	invalid := []string{"", "Fake", "REAL", "screenshots", "unknown", " fake", "fake ", "measure "}
	for _, s := range invalid {
		if IsValidScenario(s) {
			t.Fatalf("IsValidScenario(%q)=true, want false", s)
		}
		if err := ValidateScenario(s); err == nil {
			t.Fatalf("ValidateScenario(%q)=nil, want error", s)
		}
	}
}
