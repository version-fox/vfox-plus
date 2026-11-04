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

// Package gnomesession implements `vfxtool gnome-session`: it runs a command inside a GNOME
// Shell session that has no screen, which is how the end-to-end tests of the icon in the tray
// meet the shell that the icon is for, on a machine that has none (a CI runner, a container).
//
// The shell runs headless, with a virtual monitor, as a Wayland display server (mutter does that
// without a GPU), and with the AppIndicator extension enabled, which is what puts the icons of
// applications in its panel: it owns org.kde.StatusNotifierWatcher on the session bus. The command
// is started when the extension does, on the display of the shell, with the session bus of the
// shell. The shell is run in unsafe mode, so that the test can ask it what its panel shows
// (org.gnome.Shell.Eval).
//
// Run it in a container. The shell wants a system bus and no logind (there is none in a
// container, and a shell that finds /run/systemd/seats looks for one), and a user to run as: as
// root the command makes the system bus, takes those directories away, and runs the rest as the
// user of the --user flag, under dbus-run-session, which gives the session its bus.
package gnomesession

import (
	"errors"
	"flag"
	"regexp"
	"strings"
	"time"
)

const usage = "Usage: gnome-session [--user NAME] [--extension ID] <command...>"

// The extension that puts the icons of applications in the panel, as the session of Ubuntu has it
// enabled; other distributions call it appindicatorsupport@rgcjonas.gmail.com.
const defaultExtension = "ubuntu-appindicators@ubuntu.com"

// How long the shell has to be up with the extension in it.
const startTimeout = 90 * time.Second

// options are the parsed arguments.
type options struct {
	user      string
	extension string
	// inSession is set by the command itself, for the part of it that runs inside the session of the user.
	inSession bool
	command   []string
}

func parseArgs(args []string) (options, error) {
	var o options
	flags := flag.NewFlagSet("gnome-session", flag.ContinueOnError)
	flags.StringVar(&o.user, "user", "tester", "the user that the session runs as (the command is started as root, and drops to this user)")
	flags.StringVar(&o.extension, "extension", defaultExtension, "the extension that shows the icons of applications in the panel")
	flags.BoolVar(&o.inSession, "in-session", false, "(used by the command itself) run inside the session of the user")
	if err := flags.Parse(args); err != nil {
		return o, err
	}
	o.command = flags.Args()
	if len(o.command) == 0 {
		return o, errors.New(usage)
	}
	return o, nil
}

// enabledExtensions is the value of the setting that names the extensions that are enabled.
func enabledExtensions(id string) string { return "['" + id + "']" }

var displayName = regexp.MustCompile(`Using Wayland display name '([^']+)'`)

// waylandDisplay is the name of the display that the shell says it serves, wayland-0 when it does not say.
func waylandDisplay(shellLog string) string {
	if match := displayName.FindStringSubmatch(shellLog); match != nil {
		return match[1]
	}
	return "wayland-0"
}

// tail is the last n lines of text.
func tail(text string, n int) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
