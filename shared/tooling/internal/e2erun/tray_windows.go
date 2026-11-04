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

//go:build windows

package e2erun

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
)

// runTray runs the application on the desktop of the runner, which has a taskbar and a notification
// area of its own, so there is nothing to stand in for.
func runTray(command []string) error {
	config, err := trayConfig(command)
	if err != nil {
		return err
	}
	return appsession.RunScenario(config, "tray", windowsTrayScenario)
}

// windowsTrayScenario is the application on Windows: the icon comes with the closing of the
// window, and goes when the window is back; the window comes back and the application ends as the
// menu of the icon asks. The tools do what the menu does through the control file of the
// end-to-end build, since a click on an icon of the taskbar is not for a program to make.
func windowsTrayScenario(s *appsession.Session) {
	s.WaitForText("Versions in use")
	s.Step("home; the shell of the machine: " + taskbarState())
	s.Check(placed(s) == 0, "an application with a window has no icon in the tray")

	s.Step("close the window")
	closeTheWindow(s)
	waitForIcon(s, 1)

	s.Step("choose Open vfox+")
	s.Check(s.Tell("show") == nil, "Open vfox+ is chosen")
	waitForTheWindow(s)

	s.Step("close the window again")
	closeTheWindow(s)
	waitForIcon(s, 2)

	s.Step("choose Quit vfox+")
	s.Check(s.Tell("quit") == nil, "Quit vfox+ is chosen")
	s.Check(s.WaitExit(trayWait), "the application ends")
	s.Check(s.ExitError() == nil, "it ends with the code 0")
}

// placed is how many times the desktop has given the icon a place.
func placed(s *appsession.Session) int { return placedCount(s.LogText()) }

// waitForIcon waits for the desktop to have given the icon its place the given number of times,
// and fails when it says that there is none.
func waitForIcon(s *appsession.Session, times int) {
	s.WaitFor("the desktop to give the icon a place in the tray", trayWait, func() bool {
		if gaveNoPlace(s.LogText()) {
			s.Fail("the desktop gave the icon no place in the tray (%s)\n%s", taskbarState(), s.LogText())
		}
		return placed(s) >= times
	})
}

// taskbarState says whether the shell of Windows, explorer.exe, runs on the machine: it draws the
// taskbar and its notification area, which give the icon its place, and a machine that has none
// has no place to give. It is there for the one who reads the log of a failure, and of a run.
func taskbarState() string {
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq explorer.exe", "/NH").CombinedOutput() //nolint:gosec // The tool of the system, with fixed arguments.
	text := strings.Join(strings.Fields(string(out)), " ")
	switch {
	case err != nil:
		return fmt.Sprintf("tasklist failed: %v: %s", err, text)
	case strings.Contains(strings.ToLower(text), "explorer.exe") && !strings.Contains(strings.ToLower(text), "no tasks"):
		return "explorer.exe runs: " + text
	default:
		return "explorer.exe does not run: " + text
	}
}
