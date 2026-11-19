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

//go:build linux

package e2erun

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/trayhost"
)

// trayItemID is the ID of the icon: the ID of the application, by which a desktop keeps the
// settings it has for the icon, and which therefore stays the same from one appearance to the next.
const trayItemID = "dev.vfox.VfoxPlus"

// runTray runs the application on a desktop that has a tray: the stand-in for the host of the tray
// icons (package trayhost) owns the name of the watcher on the session bus before the application
// starts, as the panel does on a real desktop. Then the host is gone, and comes up after the window
// has closed: the icon waits for it.
func runTray(command []string) error {
	config, err := trayConfig(command)
	if err != nil {
		return err
	}
	config.Setup = browser
	host, err := trayhost.Start()
	if err != nil {
		return fmt.Errorf("start the stand-in for the host of the tray icons (it needs the session bus of a desktop session; dbus-run-session gives one): %w", err)
	}
	defer func() { _ = host.Close() }()
	if err := appsession.RunScenario(config, "tray", func(s *appsession.Session) { trayScenario(s, host) }); err != nil {
		return err
	}
	// The host leaves the bus, as there is none on a desktop that has no panel yet.
	if err := host.Close(); err != nil {
		return err
	}
	return appsession.RunScenario(config, "tray-late", lateHostScenario)
}

// trayScenario is the application against the stand-in host: the icon comes with the closing of the
// window, its ID, its picture and its menu are right, it brings the window back and goes with it, and
// the host of the tray icons starts again, as a desktop does when its shell restarts: the icon takes
// its place with the new host. The menu has no item for a web interface, and it ends the application.
func trayScenario(s *appsession.Session, host *trayhost.Host) {
	s.WaitForText("Versions in use")
	s.Step("home")
	s.Check(host.Registered() == 0, "an application with a window has no icon in the tray")

	s.Step("close the window")
	closeTheWindow(s)
	first := nextIcon(s, host)
	properties := checkIcon(s, first)
	s.Step("the icon is in the tray: " + properties.ID)

	s.Check(host.Registered() == 1, "one icon is in the tray")
	// The item is a menu, so a click on the icon shows the menu, which is how the user gets to the
	// window. A desktop that activates the icon all the same brings the window back.
	s.Check(first.Activate() == nil, "Activate brings the window back")
	waitForTheWindow(s)
	s.Step("activated: the window is back")
	waitPassive(s, first)
	closeTheWindow(s)
	first = nextIcon(s, host)
	checkIcon(s, first)

	s.Check(!s.HasWindow(), "the window is closed")
	s.Step("choose Open vfox+")
	s.Check(first.Choose(menuOpen) == nil, "Open vfox+ is chosen in the menu")
	waitForTheWindow(s)
	waitPassive(s, first)
	waitGone(s, first)
	s.Check(host.Registered() == 0, "the icon goes when the window comes back")

	s.Step("close the window again")
	closeTheWindow(s)
	second := nextIcon(s, host)
	secondProperties := checkIcon(s, second)
	s.Check(secondProperties.ID == properties.ID, "the icon keeps its ID, by which a desktop keeps its settings for it")
	s.Check(second.Service != first.Service, "the icon that comes back is an item of its own, under a name of its own: "+second.Service)

	s.Step("the host of the tray icons starts again, as a desktop does when its shell restarts")
	restarted := restartHost(s, host)
	defer func() { _ = restarted.Close() }()
	third := nextIcon(s, restarted)
	thirdProperties := checkIcon(s, third)
	s.Check(thirdProperties.ID == properties.ID, "the icon that the new host shows keeps its ID")
	s.Check(restarted.Registered() == 1, "the new host has the icon, which the application registered again")

	s.Step("the application opens no browser of this computer")
	expectNoBrowser(s)
	s.Check(!s.HasWindow() && s.Alive(), "the application is as it was")

	s.Step("choose Quit vfox+")
	s.Check(third.Choose(menuQuit) == nil, "Quit vfox+ is chosen in the menu")
	s.Check(s.WaitExit(trayWait), "the application ends")
	s.Check(s.ExitError() == nil, fmt.Sprintf("it ends with the code 0 (%v)", s.ExitError()))
	waitGone(s, third)
}

// lateHostScenario is the application on a desktop whose host of the tray icons comes up after the
// window has closed, as it does when the session starts before the shell is ready: the icon waits for
// the host, and the host takes it when it comes up. The application runs on in the meantime.
func lateHostScenario(s *appsession.Session) {
	s.WaitForText("Versions in use")
	s.Step("close the window, with no host of the tray icons on the bus")
	closeTheWindow(s)
	s.WaitFor("the application to say that its icon waits for a host", trayWait, func() bool {
		return strings.Contains(s.LogText(), "the icon waits for a host of the tray icons")
	})
	s.Check(s.Alive(), "the application runs on while its icon waits for a host")

	s.Step("the host of the tray icons comes up")
	host := startHost(s)
	defer func() { _ = host.Close() }()
	item := nextIcon(s, host)
	properties := checkIcon(s, item)
	s.Step("the icon is in the tray, taken by the host that came up: " + properties.ID)
	s.Check(host.Registered() == 1, "one icon is in the tray")

	s.Step("choose Open vfox+")
	s.Check(item.Choose(menuOpen) == nil, "Open vfox+ is chosen in the menu")
	waitForTheWindow(s)
	waitPassive(s, item)
	waitGone(s, item)

	s.Step("close the window again, and quit from the menu")
	closeTheWindow(s)
	second := nextIcon(s, host)
	checkIcon(s, second)
	s.Check(second.Choose(menuQuit) == nil, "Quit vfox+ is chosen in the menu")
	s.Check(s.WaitExit(trayWait), "the application ends")
	s.Check(s.ExitError() == nil, fmt.Sprintf("it ends with the code 0 (%v)", s.ExitError()))
	waitGone(s, second)
}

// startHost starts the stand-in host. The name of the watcher may still belong to a host that has just
// left the bus, which the bus takes off it a moment later, so the start is tried until it works.
func startHost(s *appsession.Session) *trayhost.Host {
	deadline := time.Now().Add(trayWait)
	for {
		host, err := trayhost.Start()
		if err == nil {
			return host
		}
		if time.Now().After(deadline) {
			s.Fail("start the stand-in for the host of the tray icons: %v", err)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// restartHost takes the stand-in host off the bus, and starts one again.
func restartHost(s *appsession.Session, host *trayhost.Host) *trayhost.Host {
	if err := host.Close(); err != nil {
		s.Fail("take the stand-in for the host of the tray icons off the bus: %v", err)
	}
	return startHost(s)
}

// nextIcon waits for the application to register its icon with the host.
func nextIcon(s *appsession.Session, host *trayhost.Host) *trayhost.Item {
	ctx, cancel := context.WithTimeout(context.Background(), trayWait)
	defer cancel()
	item, err := host.NextItem(ctx)
	if err != nil {
		s.Fail("%v\nThe application wrote:\n%s", err, s.LogText())
	}
	return item
}

// waitPassive waits for the icon to be taken away.
func waitPassive(s *appsession.Session, item *trayhost.Item) {
	ctx, cancel := context.WithTimeout(context.Background(), trayWait)
	defer cancel()
	if err := item.WaitPassive(ctx); err != nil {
		s.Fail("%v", err)
	}
}

// waitGone waits for the application to take the item off the bus, which it does with the icon.
func waitGone(s *appsession.Session, item *trayhost.Item) {
	ctx, cancel := context.WithTimeout(context.Background(), trayWait)
	defer cancel()
	if err := item.WaitGone(ctx); err != nil {
		s.Fail("%v", err)
	}
}

// checkIcon checks what the panel reads of the icon: what it says it is, its picture, and its menu.
func checkIcon(s *appsession.Session, item *trayhost.Item) trayhost.Properties {
	properties, err := item.Properties()
	if err != nil {
		s.Fail("%v", err)
	}
	s.Check(properties.ID == trayItemID, fmt.Sprintf("the icon has the ID of the application: %q", properties.ID))
	s.Check(properties.Title == trayToolTip, fmt.Sprintf("the icon says what it is: %q", properties.Title))
	s.Check(properties.Status == "Active" && properties.Category == "ApplicationStatus", fmt.Sprintf("the icon is active: %+v", properties))
	s.Check(properties.Menu != "", "the icon has a menu")

	// The picture is in the item itself: the panel draws the pixels that the item has, and finds no
	// file or name of the desktop's themes for them.
	s.Check(properties.IconName == "" && len(properties.Pixmaps) == 1,
		fmt.Sprintf("the picture is in the item, and no name of a theme: %q, %d pictures", properties.IconName, len(properties.Pixmaps)))
	if len(properties.Pixmaps) == 1 {
		picture := properties.Pixmaps[0]
		s.Check(picture.Width == 32 && picture.Height == 32 && len(picture.Pixels) == 4*32*32,
			fmt.Sprintf("the picture is 32 by 32 pixels, not %d by %d with %d bytes", picture.Width, picture.Height, len(picture.Pixels)))
		if len(picture.Pixels) == 4*32*32 {
			// Each pixel is alpha, then red, green and blue: the fox is opaque at the middle, and the corner is clear.
			middle := 4 * (16*32 + 16)
			s.Check(picture.Pixels[middle] == 0xFF && picture.Pixels[0] == 0, "the picture is the fox, with no plate")
		}
	}

	entries, err := item.Entries()
	if err != nil {
		s.Fail("%v", err)
	}
	var labels []string
	for _, entry := range entries {
		labels = append(labels, entry.Label)
	}
	s.Check(slices.Equal(labels, trayMenu), fmt.Sprintf("the menu is %q, not %q", trayMenu, labels))
	s.Check(len(entries) == len(trayMenu) && entries[1].Separator, "the second entry is the separator")
	return properties
}

// browser is the browser of the user: the application that the desktop starts for the links of
// the web. It writes the link that it is given to a file, so that the scenario sees what the
// application asked the desktop to open, and nothing starts a browser on a runner that has one.
func browser(home string) ([]string, error) {
	record := filepath.Join(home, "opened.txt")
	script := filepath.Join(home, "browser.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$1\" >> '"+record+"'\n"), 0o755); err != nil { //nolint:gosec // The script of the test must run.
		return nil, err
	}
	data := filepath.Join(home, ".local", "share")
	for _, dir := range []string{filepath.Join(data, "applications"), filepath.Join(home, ".config")} {
		if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // The home directory of the test.
			return nil, err
		}
	}
	entry := "[Desktop Entry]\nType=Application\nName=Test browser\nExec=" + script + " %u\nMimeType=x-scheme-handler/http;x-scheme-handler/https;\nNoDisplay=true\n"
	if err := os.WriteFile(filepath.Join(data, "applications", "vfox-test-browser.desktop"), []byte(entry), 0o644); err != nil { //nolint:gosec // As above.
		return nil, err
	}
	defaults := "[Default Applications]\nx-scheme-handler/http=vfox-test-browser.desktop\nx-scheme-handler/https=vfox-test-browser.desktop\n"
	if err := os.WriteFile(filepath.Join(home, ".config", "mimeapps.list"), []byte(defaults), 0o644); err != nil { //nolint:gosec // As above.
		return nil, err
	}
	return []string{"XDG_DATA_HOME=" + data}, nil
}

// expectNoBrowser lets a moment pass, and checks that the desktop was asked to open no link: the test browser writes
// each link that it is given to a file, and no link is written when the application opens none.
func expectNoBrowser(s *appsession.Session) {
	time.Sleep(time.Second)
	_, err := os.Stat(filepath.Join(s.Home, "opened.txt"))
	s.Check(os.IsNotExist(err), "the desktop is asked to open no link")
}
