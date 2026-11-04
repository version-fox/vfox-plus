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
	"context"
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
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
	s.Step("home; " + windowsVersion() + "; " + desktopState(executable))
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
//
// A desktop with no taskbar on it has no notification area to put the icon in, whatever the application
// does, and the scenario says so and goes on with what it can check: the window comes back and the
// application ends as the menu of the icon asks, which the control file stands in for.
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
	if found {
		s.Step("the desktop gave the icon no place, and the shell has a record of it: " + describeRecord(record))
		return
	}
	if taskbarWindow() == 0 {
		s.Step("NOTE: the desktop of the test has no taskbar, so there is no notification area to put the icon in and " +
			"the icon is not checked (" + desktopState(executable) + ")")
		return
	}
	s.Fail("the desktop gave the icon no place in the tray, and the shell has no record of the icon\n%s; %s\n%s\n%s",
		windowsVersion(), desktopState(executable), describeRecords(records, err), probeNotifyIcon())
}

// taskbarWindow is the window of the taskbar, on the desktop that this process has and that the application, which
// it starts, shares: there is none on a desktop that the shell does not draw, as that of a service.
func taskbarWindow() windows.HWND {
	class, err := windows.UTF16PtrFromString("Shell_TrayWnd")
	if err != nil {
		return 0
	}
	found, _, _ := windows.NewLazySystemDLL("user32.dll").NewProc("FindWindowW").Call(uintptr(unsafe.Pointer(class)), 0)
	return windows.HWND(found)
}

// ownerOf is the process that made a window and the session that the process is in.
func ownerOf(window windows.HWND) (process, session uint32) {
	if window == 0 {
		return 0, 0
	}
	if _, err := windows.GetWindowThreadProcessId(window, &process); err != nil {
		return 0, 0
	}
	_ = windows.ProcessIdToSessionId(process, &session)
	return process, session
}

// desktopState says where the test runs, and where the taskbar is that the icon is for: the icon reaches the
// taskbar only if the application, which is started by the test and has its user, session and desktop, is on the
// desktop that the shell draws. It is there for the one who reads the log of a failure, and of a run.
func desktopState(executable string) (state string) {
	// What is said here is for reading, and a failure of saying it is no failure of the scenario.
	defer func() {
		if recovered := recover(); recovered != nil {
			state = fmt.Sprintf("the state of the desktop could not be read: %v", recovered)
		}
	}()
	var session uint32
	_ = windows.ProcessIdToSessionId(windows.GetCurrentProcessId(), &session)
	who := "an unknown user"
	if current, err := user.Current(); err == nil {
		who = current.Username
	}
	describe := func(window windows.HWND) string {
		if window == 0 {
			return "none"
		}
		process, owner := ownerOf(window)
		return fmt.Sprintf("0x%x of the process %d in the session %d", uintptr(window), process, owner)
	}
	return fmt.Sprintf("the test runs as %s in the session %d; on its desktop the shell window is %s and the taskbar window is %s; %s; %s",
		who, session, describe(windows.GetShellWindow()), describe(taskbarWindow()),
		processState("explorer.exe"), processState(filepath.Base(executable)))
}

// processState says what the system lists of the processes of a program: the session and the user.
func processState(program string) string {
	// /V asks each window of the system whether it responds, which can take a while on a busy machine.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "tasklist", "/V", "/FO", "CSV", "/NH", "/FI", "IMAGENAME eq "+program).CombinedOutput() //nolint:gosec // The tool of the system, with the name of a program of the scenario.
	text := strings.Join(strings.Fields(string(out)), " ")
	switch {
	case err != nil:
		return fmt.Sprintf("%s: tasklist failed: %v: %s", program, err, text)
	case strings.Contains(strings.ToLower(text), "no tasks"):
		return program + " does not run"
	default:
		return program + " runs: " + text
	}
}

// probeNotifyIcon adds an icon to the notification area as any program does, with Windows Forms, and says
// whether the shell recorded it: what the machine does with an icon that is not the application's. A failure
// of the scenario reports it, to tell an application that is not let in from a machine that lets nobody in.
func probeNotifyIcon() string {
	const script = `$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$icon = New-Object System.Windows.Forms.NotifyIcon
$icon.Icon = [System.Drawing.SystemIcons]::Information
$icon.Text = 'vfox-plus probe'
$icon.Visible = $true
Start-Sleep -Seconds 4
$recorded = @(Get-ChildItem 'HKCU:\Control Panel\NotifyIconSettings' -ErrorAction SilentlyContinue |
  ForEach-Object { (Get-ItemProperty $_.PSPath).ExecutablePath } | Where-Object { $_ -like '*powershell*' })
$icon.Visible = $false
$icon.Dispose()
'records of the shell for powershell.exe: ' + $recorded.Count`
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	out, err := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput() //nolint:gosec // The tool of the system, with a script of the scenario.
	text := strings.Join(strings.Fields(string(out)), " ")
	if err != nil {
		return fmt.Sprintf("a plain notification icon, as a probe: failed: %v: %s", err, text)
	}
	return "a plain notification icon, as a probe, with " + text
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
