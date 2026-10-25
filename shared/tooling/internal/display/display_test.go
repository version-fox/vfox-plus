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

package display

import "testing"

func TestParseBusAddress(t *testing.T) {
	// Lines as `dbus-launch --sh-syntax` prints them: one statement per line.
	const output = "DBUS_SESSION_BUS_ADDRESS='unix:path=/run/bus,guid=1';\n" +
		"export DBUS_SESSION_BUS_ADDRESS;\n" +
		"DBUS_SESSION_BUS_PID=123;\n" +
		"export DBUS_SESSION_BUS_PID;\n"
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"quoted", output, "unix:path=/run/bus,guid=1"},
		{"unquoted", "DBUS_SESSION_BUS_ADDRESS=unix:abstract=/tmp/bus;\n", "unix:abstract=/tmp/bus"},
		{"ignores other variables", "DBUS_SESSION_BUS_PID=123;\n", ""},
		{"ignores garbage", "not an assignment\n", ""},
		{"empty", "", ""},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseBusAddress([]byte(tt.input)); got != tt.want {
				t.Errorf("parseBusAddress(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDisplayEnviron(t *testing.T) {
	d := &Display{Name: ":42", BusAddress: "unix:path=/run/bus"}
	got := d.Environ()
	want := []string{"DISPLAY=:42", "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/bus"}
	if len(got) != len(want) {
		t.Fatalf("Environ() = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Environ()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
