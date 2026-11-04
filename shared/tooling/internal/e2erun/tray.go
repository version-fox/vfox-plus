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
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
)

// The tray scenario is the application in the lightweight mode on a desktop with a tray: it
// closes its window, an icon stands for it in the tray, the icon brings the window back and
// ends the application. What the desktop has to do with the icon differs, and so does the
// scenario: on Linux there is a stand-in for the host of the tray (package trayhost), which a CI
// runner needs because it has no panel; on Windows the desktop is the runner's own.

// trayToolTip is what the icon says it is.
const trayToolTip = "vfox+ is running in the background"

// trayMenu is the menu of the icon, in English, which the scenario chooses by position and
// checks by label. The empty one is the separator.
var trayMenu = []string{"Open vfox+", "Open the web interface", "", "Quit vfox+"}

// The positions of the entries that the scenario chooses.
const (
	menuOpen = 0
	menuWeb  = 1
	menuQuit = 3
)

// How long what the scenario waits for may take: the window to close or to open, the icon to come
// or to go, the application to end.
const trayWait = 30 * time.Second

// trayConfig is how the application starts for the scenario: with the web interface and the
// lightweight mode on, which is what keeps it running without its window.
func trayConfig(command []string) (appsession.Config, error) {
	root, err := appsession.FindRepoRoot()
	if err != nil {
		return appsession.Config{}, err
	}
	return appsession.Config{
		Name: "tray", Command: command, Fixtures: appsession.FixturesDir(root),
		Settings: map[string]any{"webInterface": true, "lightweight": true},
	}, nil
}

// closeTheWindow closes the window as the close button does and waits for it to be gone: the
// application has to run on.
func closeTheWindow(s *appsession.Session) {
	s.Check(s.Tell("close") == nil, "the application is asked to close its window")
	s.WaitFor("the window to close", trayWait, func() bool { return !s.HasWindow() })
	s.Check(s.Alive(), "the application runs on without its window")
}

// waitForTheWindow waits for the window to open, with the page of a new start in it.
func waitForTheWindow(s *appsession.Session) {
	s.WaitFor("the window to open", trayWait, s.HasWindow)
	s.WaitForText("Versions in use")
}

// The end-to-end build writes these to its log: the desktop gave the icon a place on the screen (the
// notification area of the taskbar), or it did not within the time that it has. Windows tells the application that
// much of its icon, and it is what a person sees of it. The words are the application's (reportTray of
// mygo/ui/e2e.go); a test reads them there.
const (
	iconPlaced  = "e2e: the icon is in the tray at"
	iconNoPlace = "e2e: the icon has no place in the tray"
)

// placedCount is how many times the log says that the desktop gave the icon a place.
func placedCount(log string) int { return strings.Count(log, iconPlaced) }

// gaveNoPlace says whether the log says that the desktop gave the icon none.
func gaveNoPlace(log string) bool { return strings.Contains(log, iconNoPlace) }
