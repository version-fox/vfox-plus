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

package capture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEscapeControls(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"empty", "", ""},
		{"plain", "vfox version 1.0.13\n", "vfox version 1.0.13\n"},
		{"backslash", `a\b`, `a\\b`},
		{"backslash run", `\\\`, `\\\\\\`},
		{"tab and newline preserved", "a\tb\n", "a\tb\n"},
		{"carriage return", "a\rb", `a\x0db`},
		{"nul", "a\x00b", `a\x00b`},
		{"bell through backspace", "\a\b", `\x07\x08`},
		{"vertical tab and form feed", "\v\f", `\x0b\x0c`},
		{"escape", "\x1b[0m", `\x1b[0m`},
		{"unit separator", "\x1f", `\x1f`},
		{"del", "\x7f", `\x7f`},
		{"printable boundary", " ~", " ~"},
		{"high bytes pass through", "\x80\xff", "\x80\xff"},
		{
			"mixed",
			"path\\to\x00dir\twith\nlines\r\x1bend\x7f\x80",
			`path\\to\x00dir	with` + "\n" + `lines\x0d\x1bend\x7f` + "\x80",
		},
		{"lowercase hex", "a\x0db\x00c", `a\x0db\x00c`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(EscapeControls([]byte(tt.input))); got != tt.want {
				t.Errorf("EscapeControls(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestEscapeFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.stdout")
	if err := os.WriteFile(path, []byte("a\\b\x00c\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := escapeFiles(path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := `a\\b\x00c\x0d` + "\n"; string(got) != want {
		t.Errorf("escapeFiles wrote %q, want %q", got, want)
	}
}
