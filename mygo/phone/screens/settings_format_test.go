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

package screens

import (
	"maps"
	"testing"
)

func TestTheExtraEnvironmentIsReadLineByLine(t *testing.T) {
	text := "A=1\n" +
		"B=two=2\n" +
		" =skipped\n" +
		"=also skipped\n" +
		"NO_EQUALS\n" +
		"  C  = spaced value \n" +
		"A=3\n" +
		"EMPTY=\n"
	want := map[string]string{
		"A":     "3",
		"B":     "two=2",
		"C":     " spaced value ",
		"EMPTY": "",
	}
	if got := settingsParseEnvironment(text); !maps.Equal(got, want) {
		t.Errorf("settingsParseEnvironment = %q, want %q", got, want)
	}
	if got := settingsParseEnvironment(""); len(got) != 0 {
		t.Errorf("an empty field reads %q, want no variables", got)
	}
}

func TestTheExtraEnvironmentIsWrittenInKeyOrder(t *testing.T) {
	env := map[string]string{"ZED": "last", "ALPHA": "first=1", "MID": ""}
	want := "ALPHA=first=1\nMID=\nZED=last"
	if got := settingsFormatEnvironment(env); got != want {
		t.Errorf("settingsFormatEnvironment = %q, want %q", got, want)
	}
	if got := settingsFormatEnvironment(nil); got != "" {
		t.Errorf("settingsFormatEnvironment(nil) = %q, want an empty field", got)
	}
	if back := settingsParseEnvironment(settingsFormatEnvironment(env)); !maps.Equal(back, env) {
		t.Errorf("the written environment reads back as %q, want %q", back, env)
	}
}

func TestThePortIsAWholeNumberInTheRange(t *testing.T) {
	cases := []struct {
		text string
		want int
		ok   bool
	}{
		{"1024", 1024, true},
		{"65535", 65535, true},
		{" 8080 ", 8080, true},
		{"1023", 0, false},
		{"65536", 0, false},
		{"0", 0, false},
		{"-1", 0, false},
		{"80a", 0, false},
		{"", 0, false},
		{"1.5", 0, false},
	}
	for _, tc := range cases {
		got, ok := settingsParsePort(tc.text)
		if got != tc.want || ok != tc.ok {
			t.Errorf("settingsParsePort(%q) = (%d, %t), want (%d, %t)", tc.text, got, ok, tc.want, tc.ok)
		}
	}
}
