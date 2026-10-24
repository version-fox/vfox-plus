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

import "testing"

func TestParseShotName(t *testing.T) {
	cases := []struct {
		name string
		line string
		shot string
		ok   bool
	}{
		{"simple", "e2e: SHOT home", "home", true},
		{"with dash", "e2e: SHOT settings-page", "settings-page", true},
		{"other e2e line", "e2e: scenario screenshots passed", "", false},
		{"missing prefix", "SHOT home", "", false},
		{"contains but no prefix", "xx e2e: SHOT home", "", false},
		{"empty name", "e2e: SHOT ", "", false},
		{"only spaces", "e2e: SHOT    ", "", false},
		{"slash rejected", "e2e: SHOT a/b", "", false},
		{"absolute rejected", "e2e: SHOT /tmp/x", "", false},
		{"dotdot rejected", "e2e: SHOT ..", "", false},
		{"dotdot path rejected", "e2e: SHOT ../escape", "", false},
		{"embedded dotdot rejected", "e2e: SHOT a..b", "", false},
		{"trailing spaces trimmed", "e2e: SHOT home  ", "home", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			shot, ok := ParseShotName(c.line)
			if ok != c.ok {
				t.Fatalf("ParseShotName(%q) ok=%v, want %v", c.line, ok, c.ok)
			}
			if ok && shot != c.shot {
				t.Fatalf("ParseShotName(%q)=%q, want %q", c.line, shot, c.shot)
			}
		})
	}
}
