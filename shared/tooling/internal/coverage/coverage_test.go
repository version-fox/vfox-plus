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

package coverage

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleProfile = `mode: set
example.com/mygo/core/store.go:3.14,5.2 10 8
example.com/mygo/core/other.go:1.10,2.20 10 10
example.com/mygo/state/state.go:1.10,2.20 5 5
example.com/mygo/ui/page.go:1.10,2.20 8 6
example.com/other/util.go:1.10,2.20 4 4
example.com/mygo/ui/zero.go:1.10,2.20 2 0
`

func TestSummarize(t *testing.T) {
	stats, err := summarize(strings.NewReader(sampleProfile))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		area       string
		statements int
		covered    int
	}{
		{"core", 20, 20}, // both files have count > 0
		{"state", 5, 5},
		{"ui", 10, 8}, // the zero-count file adds statements but no cover
		{"other", 4, 4},
	}
	for _, tc := range cases {
		s, ok := stats[tc.area]
		if !ok {
			t.Errorf("missing area %q", tc.area)
			continue
		}
		if s.statements != tc.statements || s.covered != tc.covered {
			t.Errorf("area %q = %+v, want statements=%d covered=%d", tc.area, s, tc.statements, tc.covered)
		}
	}
}

func TestSummarizeBadLine(t *testing.T) {
	_, err := summarize(strings.NewReader("mode: set\nnot enough fields\n"))
	if err == nil {
		t.Error("expected an error for a malformed line")
	}
}

func TestReportSamplePass(t *testing.T) {
	// core+state: 25/25 = 100%, ui: 8/10 = 80% -> both gates pass.
	stats, err := summarize(strings.NewReader(sampleProfile))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if !report(stats, 85, "85", &out) {
		t.Errorf("expected both gates to pass, got:\n%s", out.String())
	}
	for _, want := range []string{
		"core and state: 100.0 % (minimum 85 %)",
		"ui: 80.0 % (minimum 75 %)",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q, got:\n%s", want, out.String())
		}
	}
}

// gateProfile covers core 18/20, state 5/5, ui 6/8.
const gateProfile = `mode: set
example.com/mygo/core/a.go:1.1,2.2 18 12
example.com/mygo/core/b.go:1.1,2.2 2 0
example.com/mygo/state/b.go:1.1,2.2 5 5
example.com/mygo/ui/c.go:1.1,2.2 6 6
example.com/mygo/ui/d.go:1.1,2.2 2 0
`

func TestReportGates(t *testing.T) {
	stats, err := summarize(strings.NewReader(gateProfile))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if !report(stats, 85, "85", &out) {
		t.Errorf("expected pass, got:\n%s", out.String())
	}
	text := out.String()
	for _, want := range []string{
		"core and state: 92.0 % (minimum 85 %)",
		"ui: 75.0 % (minimum 75 %)",
		"core    90.0 % of 20 statements",
		"state  100.0 % of 5 statements",
		"ui      75.0 % of 8 statements",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("output missing %q, got:\n%s", want, text)
		}
	}

	// Raising the minimum must fail the core+state gate.
	var out2 bytes.Buffer
	if report(stats, 95, "95", &out2) {
		t.Error("expected failure with minimum 95")
	}
	if !strings.Contains(out2.String(), "core and state: 92.0 % (minimum 95 %)") {
		t.Errorf("gate line missing, got:\n%s", out2.String())
	}
}

func TestReportUIFail(t *testing.T) {
	low := `mode: set
example.com/mygo/core/a.go:1.1,2.2 20 20
example.com/mygo/state/b.go:1.1,2.2 5 5
example.com/mygo/ui/c.go:1.1,2.2 4 4
example.com/mygo/ui/d.go:1.1,2.2 4 0
`
	stats, err := summarize(strings.NewReader(low))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if report(stats, 85, "85", &out) {
		t.Error("expected failure of the ui gate at 50%")
	}
	if !strings.Contains(out.String(), "ui: 50.0 % (minimum 75 %)") {
		t.Errorf("ui gate line missing, got:\n%s", out.String())
	}
}

func TestRun(t *testing.T) {
	dir := t.TempDir()
	profile := filepath.Join(dir, "cover.out")
	if err := os.WriteFile(profile, []byte(gateProfile), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{profile}); err != nil {
		t.Errorf("Run = %v, want nil", err)
	}
	if err := Run([]string{profile, "99"}); err == nil {
		t.Error("Run with minimum 99 = nil, want an error")
	}
	if err := Run(nil); err == nil {
		t.Error("Run with no args = nil, want an error")
	}
	if err := Run([]string{profile, "bogus"}); err == nil {
		t.Error("Run with a bogus minimum = nil, want an error")
	}
	if err := Run([]string{filepath.Join(dir, "missing")}); err == nil {
		t.Error("Run with a missing profile = nil, want an error")
	}
}
