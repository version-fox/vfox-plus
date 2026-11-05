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
	"fmt"
	"path"
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

// The end-to-end build writes these to its log, for the icon that it makes each time the window closes: the
// desktop gave the icon a place on the screen (the notification area of the taskbar), or it did not within the
// time that it has. Windows tells the application that much of its icon, and it is what a person sees of it. The
// words are the application's (reportTray of mygo/ui/e2e.go); a test reads them there.
const (
	iconPlaced  = "e2e: the icon is in the tray at"
	iconNoPlace = "e2e: the icon has no place in the tray"
)

// iconOutcome is what the desktop answered for an icon that the application made.
type iconOutcome struct {
	// Placed is true when the icon has a place on the screen, which the line says.
	Placed bool
	// Line is what the application wrote.
	Line string
}

// iconOutcomes reads the log of the application for what the desktop answered for each icon that it made, in the
// order in which it answered.
func iconOutcomes(log string) []iconOutcome {
	var outcomes []iconOutcome
	for _, line := range strings.Split(log, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.Contains(line, iconPlaced):
			outcomes = append(outcomes, iconOutcome{Placed: true, Line: line})
		case strings.Contains(line, iconNoPlace):
			outcomes = append(outcomes, iconOutcome{Line: line})
		}
	}
	return outcomes
}

// shellRecord is what the shell of Windows 11 keeps of an icon that an application adds to the notification area:
// it records each under HKEY_CURRENT_USER\Control Panel\NotifyIconSettings, which is where the page of the
// settings that lists the icons of the taskbar reads them from. It is made when the icon is added, and says that
// the shell took it, where the icon has no place that the application can ask for: Windows 11 puts a new icon in
// the overflow of the taskbar, which the notification area of Windows Server, which shows all of its icons, does
// not.
type shellRecord struct {
	Key        string
	Executable string
	ToolTip    string
	// Promoted is 1 when the user has the icon on the taskbar itself and not in the overflow, 0 when not, and -1
	// when the record does not say.
	Promoted int
}

// baseName is the name of the file of a path of either system.
func baseName(name string) string { return path.Base(strings.ReplaceAll(name, "\\", "/")) }

// recordOf finds the record of the icon of the executable among the records. The shell keeps the path of the
// program that made the icon, written as the system writes it (the scenario was given it with the slashes of the
// shell that started it), so the files are compared by name.
func recordOf(records []shellRecord, executable string) (shellRecord, bool) {
	name := baseName(executable)
	for _, record := range records {
		if record.Executable != "" && strings.EqualFold(baseName(record.Executable), name) {
			return record, true
		}
	}
	return shellRecord{}, false
}

func describeRecord(record shellRecord) string {
	return fmt.Sprintf("%s: %s, tool tip %q, promoted %d", record.Key, record.Executable, record.ToolTip, record.Promoted)
}

// describeRecords lists the records, or says why there are none to list, for the one who reads the log of a
// failure.
func describeRecords(records []shellRecord, err error) string {
	switch {
	case err != nil:
		return "the records of the shell cannot be read: " + err.Error()
	case len(records) == 0:
		return "the shell has no records of icons"
	}
	lines := make([]string, 0, len(records))
	for _, record := range records {
		lines = append(lines, "  "+describeRecord(record))
	}
	return fmt.Sprintf("the shell has %d records of icons:\n%s", len(records), strings.Join(lines, "\n"))
}
