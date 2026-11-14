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
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/trayhost"
)

// runTray runs the application on a desktop that has a tray: the stand-in for the host of the tray
// icons (package trayhost) owns the name of the watcher on the session bus before the application
// starts, as the panel does on a real desktop.
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
	// And where the desktop has no tray: the host leaves the bus, as there is none on a desktop
	// without one.
	if err := host.Close(); err != nil {
		return err
	}
	return appsession.RunScenario(config, "no-tray", noTrayScenario)
}

// trayScenario is the application against the stand-in host: the icon comes with the closing
// of the window, its picture and its menu are right, it brings the window back and goes with it,
// the web interface opens in the browser, and the menu ends the application.
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
	// The library of the version of Ubuntu that the application is built for does not take
	// Activate: the panel shows the menu, which is how the user gets to the window. An older
	// library does, and then a double click opens the window: both are right.
	if err := first.Activate(); err != nil {
		s.Check(errors.Is(err, trayhost.ErrNoActivation), "Activate is taken, or refused as an unknown method")
		s.Check(!s.HasWindow(), "an activation that is refused opens nothing")
	} else {
		waitForTheWindow(s)
		s.Step("activated: the window is back")
		waitPassive(s, first)
		closeTheWindow(s)
		first = nextIcon(s, host)
		checkIcon(s, first)
	}

	s.Check(!s.HasWindow(), "the window is closed")
	s.Step("choose Open vfox+")
	s.Check(first.Choose(menuOpen) == nil, "Open vfox+ is chosen in the menu")
	waitForTheWindow(s)
	waitPassive(s, first)
	s.Check(host.Registered() == 0, "the icon goes when the window comes back")

	s.Step("close the window again")
	closeTheWindow(s)
	second := nextIcon(s, host)
	secondProperties := checkIcon(s, second)
	s.Check(secondProperties.ID != properties.ID, "the icon that comes back is a new one, not the one that went")

	s.Step("choose Open the web interface")
	s.Check(second.Choose(menuWeb) == nil, "Open the web interface is chosen in the menu")
	link := waitForTheBrowser(s)
	s.Check(strings.HasPrefix(link, "http://127.0.0.1:") && strings.Contains(link, "/auth/connect/"), "the browser is told to open a link that pairs it: "+link)
	checkLink(s, link)
	s.Check(!s.HasWindow() && s.Alive(), "opening the web interface leaves the application as it is")

	s.Step("choose Quit vfox+")
	s.Check(second.Choose(menuQuit) == nil, "Quit vfox+ is chosen in the menu")
	s.Check(s.WaitExit(trayWait), "the application ends")
	s.Check(s.ExitError() == nil, fmt.Sprintf("it ends with the code 0 (%v)", s.ExitError()))
	ctx, cancel := context.WithTimeout(context.Background(), trayWait)
	defer cancel()
	s.Check(second.WaitGone(ctx) == nil, "the application leaves the bus")
	_, err := os.Stat(secondProperties.IconThemePath)
	s.Check(os.IsNotExist(err), "the picture of the icon is removed with it: "+secondProperties.IconThemePath)
}

// noTrayScenario is the application on a desktop that has no tray: libayatana-appindicator3 would
// make an icon of the old kind that nothing shows, and 7 MB of memory with it, so the application
// makes none, says why, and goes on: the window comes back as it is asked to.
func noTrayScenario(s *appsession.Session) {
	s.WaitForText("Versions in use")
	s.Step("close the window, with no host of the tray")
	closeTheWindow(s)
	s.WaitFor("the application to say that it makes no icon", trayWait, func() bool {
		return strings.Contains(s.LogText(), "no icon in the tray: nothing on the desktop shows tray icons")
	})
	s.Check(!strings.Contains(s.LogText(), "GtkStatusIcon"), "no icon of the old kind is made")
	s.Step("bring the window back, with no icon to bring it")
	s.Check(s.Tell("show") == nil, "the window is asked to come back")
	waitForTheWindow(s)
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

// checkIcon checks what the panel reads of the icon: what it says it is, the picture and the menu.
func checkIcon(s *appsession.Session, item *trayhost.Item) trayhost.Properties {
	properties, err := item.Properties()
	if err != nil {
		s.Fail("%v", err)
	}
	s.Check(properties.Title == trayToolTip, fmt.Sprintf("the icon says what it is: %q", properties.Title))
	s.Check(properties.Status == "Active" && properties.Category == "ApplicationStatus", fmt.Sprintf("the icon is active: %+v", properties))
	s.Check(properties.Menu != "", "the icon has a menu")

	// The picture is the one that was drawn when the application was built, a PNG that the panel
	// finds by the name of the icon in the directory that the icon names. The name has the extension
	// of the file where the application gives the icon by its path (mygo 0.9 does, so that the icon
	// theme of KDE does not take a name for one of its own), and not where it gives a name.
	file := filepath.Join(properties.IconThemePath, properties.IconName)
	if !strings.HasSuffix(file, ".png") {
		file += ".png"
	}
	data, err := os.ReadFile(file) //nolint:gosec // The picture that the application made for its icon.
	s.Check(err == nil, fmt.Sprintf("the panel can read the picture of the icon at %s (%v)", file, err))
	picture, err := png.Decode(bytes.NewReader(data))
	s.Check(err == nil, fmt.Sprintf("the picture is a PNG (%v)", err))
	bounds := picture.Bounds()
	s.Check(bounds.Dx() == 32 && bounds.Dy() == 32, fmt.Sprintf("the picture is 32 by 32 pixels, not %v", bounds))
	_, _, _, middle := picture.At(bounds.Dx()/2, bounds.Dy()/2).RGBA()
	_, _, _, corner := picture.At(0, 0).RGBA()
	s.Check(middle == 0xFFFF && corner == 0, "the picture is the fox, with no plate")

	entries, err := item.Entries()
	if err != nil {
		s.Fail("%v", err)
	}
	var labels []string
	for _, entry := range entries {
		labels = append(labels, entry.Label)
	}
	s.Check(slices.Equal(labels, trayMenu), fmt.Sprintf("the menu is %q, not %q", trayMenu, labels))
	s.Check(len(entries) == len(trayMenu) && entries[2].Separator, "the third entry is the separator")
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

// waitForTheBrowser waits for the application to ask the desktop to open a link, and returns it.
func waitForTheBrowser(s *appsession.Session) string {
	record := filepath.Join(s.Home, "opened.txt")
	var link string
	s.WaitFor("the desktop to open the link of the web interface", trayWait, func() bool {
		data, err := os.ReadFile(record) //nolint:gosec // The file of the test browser.
		link = strings.TrimSpace(string(data))
		return err == nil && link != ""
	})
	return link
}

// checkLink uses the link as the browser does: it pairs, with a session and a redirect to the page.
func checkLink(s *appsession.Session, link string) {
	client := &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Get(link) //nolint:noctx // A request of a test, with a timeout.
	if err != nil {
		s.Fail("open the link of the web interface: %v", err)
	}
	defer func() { _ = response.Body.Close() }()
	s.Check(response.StatusCode == http.StatusSeeOther && response.Header.Get("Location") == "/app",
		fmt.Sprintf("the link pairs the browser and sends it to the page: %s %s", response.Status, response.Header.Get("Location")))
	var session bool
	for _, cookie := range response.Cookies() {
		session = session || (cookie.Name == "vfox-plus-session" && cookie.Value != "")
	}
	s.Check(session, "the link gives the browser a session")
}
