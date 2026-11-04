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

	"golang.org/x/sys/windows/registry"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
)

// runTray runs the application on the desktop of the runner, which has a taskbar and a notification
// area of its own, so there is nothing to stand in for.
func runTray(command []string) error {
	config, err := trayConfig(command)
	if err != nil {
		return err
	}
	return appsession.RunScenario(config, "tray", func(s *appsession.Session) { windowsTrayScenario(s, command[0]) })
}

// windowsTrayScenario is the application on Windows: the icon comes with the closing of the
// window, and goes when the window is back; the window comes back and the application ends as the
// menu of the icon asks. The tools do what the menu does through the control file of the
// end-to-end build, since a click on an icon of the taskbar is not for a program to make.
func windowsTrayScenario(s *appsession.Session, executable string) {
	s.WaitForText("Versions in use")
	s.Step("home; " + windowsVersion() + "; the shell of the machine: " + taskbarState())
	s.Check(len(iconOutcomes(s.LogText())) == 0, "an application with a window has no icon in the tray")

	s.Step("close the window")
	closeTheWindow(s)
	waitForIcon(s, executable, 1)

	s.Step("choose Open vfox+")
	s.Check(s.Tell("show") == nil, "Open vfox+ is chosen")
	waitForTheWindow(s)

	s.Step("close the window again")
	closeTheWindow(s)
	waitForIcon(s, executable, 2)

	s.Step("choose Quit vfox+")
	s.Check(s.Tell("quit") == nil, "Quit vfox+ is chosen")
	s.Check(s.WaitExit(trayWait), "the application ends")
	s.Check(s.ExitError() == nil, "it ends with the code 0")
}

// waitForIcon waits for the desktop to answer for the icon that the application makes for the times-th
// time: the application asks the desktop where the icon is for fifteen seconds, and says what the answer
// was. The answer is a place on the screen, which a taskbar that shows the icon gives it. Windows 11 puts a
// new icon in the overflow of the taskbar, where it has no place to tell, and then the shell has to have a
// record of the icon (see shellRecord): that the shell took the icon is what there is to see of it.
func waitForIcon(s *appsession.Session, executable string, times int) {
	s.WaitFor("the desktop to answer for the icon in the tray", trayWait, func() bool {
		return len(iconOutcomes(s.LogText())) >= times
	})
	outcome := iconOutcomes(s.LogText())[times-1]
	if outcome.Placed {
		s.Step("the desktop gave the icon a place: " + outcome.Line)
		return
	}
	records, err := shellRecords()
	record, found := recordOf(records, executable)
	if !found {
		s.Fail("the desktop gave the icon no place in the tray, and the shell has no record of the icon (%s; %s)\n%s",
			windowsVersion(), taskbarState(), describeRecords(records, err))
	}
	s.Step("the desktop gave the icon no place, and the shell has a record of it: " + describeRecord(record))
}

// shellRecords reads the records that the shell keeps of the icons of the notification area.
func shellRecords() ([]shellRecord, error) {
	root, err := registry.OpenKey(registry.CURRENT_USER, `Control Panel\NotifyIconSettings`, registry.READ)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()
	names, err := root.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}
	records := make([]shellRecord, 0, len(names))
	for _, name := range names {
		key, err := registry.OpenKey(root, name, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		record := shellRecord{Key: name, Promoted: -1}
		record.Executable, _, _ = key.GetStringValue("ExecutablePath")
		record.ToolTip, _, _ = key.GetStringValue("InitialTooltip")
		if promoted, _, err := key.GetIntegerValue("IsPromoted"); err == nil {
			record.Promoted = int(promoted)
		}
		_ = key.Close()
		records = append(records, record)
	}
	return records, nil
}

// windowsVersion says which Windows the machine has, as the registry has it: the name of the product is
// "Windows 10" on Windows 11 too, and the build tells them apart (22000 and later is Windows 11), with the
// setting that has the taskbar show every icon, which Windows Server has by default.
func windowsVersion() string {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Windows NT\CurrentVersion`, registry.QUERY_VALUE)
	if err != nil {
		return "Windows of an unknown version (" + err.Error() + ")"
	}
	defer func() { _ = key.Close() }()
	product, _, _ := key.GetStringValue("ProductName")
	display, _, _ := key.GetStringValue("DisplayVersion")
	build, _, _ := key.GetStringValue("CurrentBuild")
	revision, _, _ := key.GetIntegerValue("UBR")
	return fmt.Sprintf("%s %s, build %s.%d, all icons shown: %s", product, display, build, revision, showsAllIcons())
}

// showsAllIcons says what the setting of the taskbar to show every icon of the notification area is: 0 is on
// (EnableAutoTray), and Windows 11 has no page of the settings for it.
func showsAllIcons() string {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Explorer`, registry.QUERY_VALUE)
	if err != nil {
		return "unknown"
	}
	defer func() { _ = key.Close() }()
	value, _, err := key.GetIntegerValue("EnableAutoTray")
	if err != nil {
		return "not set"
	}
	if value == 0 {
		return "yes (EnableAutoTray 0)"
	}
	return fmt.Sprintf("no (EnableAutoTray %d)", value)
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
