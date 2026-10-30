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
	"strings"
	"testing"
)

func TestRunValidatesItsArguments(t *testing.T) {
	cases := map[string]struct {
		args []string
		want string
	}{
		"nothing":                    {nil, "Usage: e2e-run"},
		"no application":             {[]string{"fake"}, "Usage: e2e-run"},
		"a scenario that is not one": {[]string{"screenshots", "application"}, "Unknown scenario: screenshots"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if err := Run(c.args); err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("Run(%q) = %v, want %q", c.args, err, c.want)
			}
		})
	}
}

func TestRunRealNeedsTheRealVfox(t *testing.T) {
	t.Setenv("VFOX_PLUS_E2E_VFOX", "")
	if err := Run([]string{"real", "/nonexistent/application"}); err == nil || !strings.Contains(err.Error(), "VFOX_PLUS_E2E_VFOX") {
		t.Fatalf("got %v, want the demand for the real vfox", err)
	}
}

// An application that cannot be started fails every scenario in the same way, after the
// scenario has set up what it needs (the test server of the real one).
func TestRunFailsWhenTheApplicationCannotBeStarted(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	t.Setenv("VFOX_PLUS_E2E_VFOX", "/bin/true")
	for _, scenario := range []string{"fake", "real", "measure"} {
		t.Run(scenario, func(t *testing.T) {
			if err := Run([]string{scenario, "/nonexistent/application"}); err == nil || !strings.Contains(err.Error(), "start the application") {
				t.Fatalf("Run(%s) = %v, want a failure to start the application", scenario, err)
			}
		})
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
