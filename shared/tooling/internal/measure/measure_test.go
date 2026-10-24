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

import "testing"

func TestParseMeasurePoint(t *testing.T) {
	cases := []struct {
		name  string
		line  string
		point string
		ok    bool
	}{
		{"home", "e2e: MEASURE home", "home", true},
		{"nodejs", "e2e: MEASURE nodejs", "nodejs", true},
		{"with prefix log", "2026-01-01 app e2e: MEASURE home", "home", true},
		{"empty point", "e2e: MEASURE ", "", true},
		{"no marker", "e2e: scenario measure passed", "", false},
		{"lowercase", "e2e: measure home", "", false},
		{"missing trailing space", "e2e: MEASURE", "", false},
		{"last marker wins", "e2e: MEASURE a MEASURE b", "b", true},
		{"point with spaces", "e2e: MEASURE my point", "my point", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			point, ok := ParseMeasurePoint(c.line)
			if ok != c.ok {
				t.Fatalf("ParseMeasurePoint(%q) ok=%v, want %v", c.line, ok, c.ok)
			}
			if ok && point != c.point {
				t.Fatalf("ParseMeasurePoint(%q)=%q, want %q", c.line, point, c.point)
			}
			if IsMeasureLine(c.line) != c.ok {
				t.Fatalf("IsMeasureLine(%q)=%v, want %v", c.line, !c.ok, c.ok)
			}
		})
	}
}
