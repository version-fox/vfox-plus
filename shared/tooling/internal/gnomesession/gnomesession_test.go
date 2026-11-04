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

package gnomesession

import (
	"slices"
	"strings"
	"testing"
)

func TestTheArgumentsAreAUserAnExtensionAndACommand(t *testing.T) {
	o, err := parseArgs([]string{"vfxtool", "e2e-run", "tray-gnome", "app"})
	if err != nil {
		t.Fatal(err)
	}
	if o.user != "tester" || o.extension != defaultExtension || o.inSession || !slices.Equal(o.command, []string{"vfxtool", "e2e-run", "tray-gnome", "app"}) {
		t.Errorf("defaults = %+v", o)
	}
	o, err = parseArgs([]string{"--user", "other", "--extension", "appindicatorsupport@rgcjonas.gmail.com", "--in-session", "--", "echo", "--user"})
	if err != nil {
		t.Fatal(err)
	}
	if o.user != "other" || o.extension != "appindicatorsupport@rgcjonas.gmail.com" || !o.inSession || !slices.Equal(o.command, []string{"echo", "--user"}) {
		t.Errorf("options = %+v: what follows the command is the command's", o)
	}
	for _, bad := range [][]string{nil, {"--user", "x"}, {"--nonsense", "echo"}} {
		if _, err := parseArgs(bad); err == nil {
			t.Errorf("parseArgs(%q) succeeded", bad)
		}
	}
	if _, err := parseArgs(nil); err == nil || !strings.Contains(err.Error(), "Usage: gnome-session") {
		t.Errorf("without a command: %v", err)
	}
}

func TestTheExtensionIsEnabledByTheSettingOfTheShell(t *testing.T) {
	if got := enabledExtensions("ubuntu-appindicators@ubuntu.com"); got != "['ubuntu-appindicators@ubuntu.com']" {
		t.Errorf("setting = %q", got)
	}
}

func TestTheDisplayOfTheShellIsTheOneThatItSays(t *testing.T) {
	log := "libmutter-Message: 07:10:49.797: Using public X11 display :0\nlibmutter-Message: 07:10:49.797: Using Wayland display name 'wayland-3'\n"
	if got := waylandDisplay(log); got != "wayland-3" {
		t.Errorf("display = %q, want wayland-3", got)
	}
	if got := waylandDisplay("nothing of the kind"); got != "wayland-0" {
		t.Errorf("display = %q, want the default wayland-0", got)
	}
}

func TestTheEndOfALogIsItsLastLines(t *testing.T) {
	text := "a\nb\nc\nd\n"
	if got := tail(text, 2); got != "c\nd" {
		t.Errorf("tail = %q", got)
	}
	if got := tail(text, 10); got != "a\nb\nc\nd" {
		t.Errorf("tail of a short log = %q", got)
	}
	if got := tail("", 3); got != "" {
		t.Errorf("tail of nothing = %q", got)
	}
}
