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
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/godbus/dbus/v5"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/trayhost"
)

// The tray-gnome scenario is the application on the GNOME Shell of a session (vfxtool
// gnome-session makes one where there is no screen): the host of the tray icons is the AppIndicator
// extension of the shell, which owns the name of the watcher, and the icon is what the shell shows
// in its panel. The scenario asks the shell what its panel has, and what menu the shell has made
// of the menu of the application, and has the extension do what it does when the user chooses an
// entry: send the event of a click to the application.

// runTrayGnome runs the scenario against the shell that the session of the process has.
func runTrayGnome(command []string) error {
	config, err := trayConfig(command)
	if err != nil {
		return err
	}
	config.Setup = browser
	host, err := trayhost.Attach()
	if err != nil {
		return fmt.Errorf("the session has no GNOME Shell with the AppIndicator extension (vfxtool gnome-session runs this scenario in one): %w", err)
	}
	defer func() { _ = host.Close() }()
	shell, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer func() { _ = shell.Close() }()
	return appsession.RunScenario(config, "tray-gnome", func(s *appsession.Session) { gnomeScenario(s, host, shellOf{shell}) })
}

// shellOf asks the shell about itself, which it answers in unsafe mode: org.gnome.Shell.Eval runs a
// script in the shell, with its modules (Main among them) in scope, and answers with the value of its
// last expression written as JSON.
type shellOf struct{ conn *dbus.Conn }

func (sh shellOf) eval(script string) (string, error) {
	var ok bool
	var result string
	call := sh.conn.Object("org.gnome.Shell", "/org/gnome/Shell").Call("org.gnome.Shell.Eval", 0, script)
	if err := call.Store(&ok, &result); err != nil {
		return "", fmt.Errorf("ask the shell: %w", err)
	}
	if !ok {
		return "", fmt.Errorf("the shell could not run the script: %s", result)
	}
	return result, nil
}

// text is the value of a script that gives a string.
func (sh shellOf) text(script string) (string, error) {
	result, err := sh.eval(script)
	if err != nil {
		return "", err
	}
	var value string
	if err := json.Unmarshal([]byte(result), &value); err != nil {
		return "", fmt.Errorf("the shell answered %q to %q, which is not a string", result, script)
	}
	return value, nil
}

// flag is the value of a script that gives a boolean.
func (sh shellOf) flag(script string) (bool, error) {
	result, err := sh.eval(script)
	return result == "true", err
}

// panelKeyOf finds the indicator that the shell made for the icon in the status area of its panel:
// the extension names it after the item.
func panelKeyOf(s *appsession.Session, sh shellOf, item *trayhost.Item) string {
	var key string
	s.WaitFor("the shell to put the icon in its panel", trayWait, func() bool {
		keys, err := sh.text("Object.keys(Main.panel.statusArea).filter(k => k.startsWith('appindicator-')).join('|')")
		if err != nil {
			s.Fail("%v", err)
		}
		for _, candidate := range strings.Split(keys, "|") {
			if candidate != "" && strings.HasSuffix(candidate, string(item.Path)) {
				key = candidate
			}
		}
		return key != ""
	})
	return key
}

// indicator is a script that gives the indicator of the panel with the key.
func indicator(key string) string { return "Main.panel.statusArea['" + key + "']" }

// gnomeScenario is the application against the shell.
func gnomeScenario(s *appsession.Session, host *trayhost.Host, sh shellOf) {
	s.WaitForText("Versions in use")
	s.Step("home")
	s.Check(host.Registered() == 0, "an application with a window has no icon in the tray")

	s.Step("close the window")
	closeTheWindow(s)
	first := nextIcon(s, host)
	properties := checkIcon(s, first)
	key := panelKeyOf(s, sh, first)
	s.Step("the icon is in the panel of the shell: " + key)
	poll(s, "the shell to show the icon", trayWait, func() bool { return visible(s, sh, key) })
	// The extension fills the menu once it has read it from the application, after the icon is in the panel.
	var labels string
	poll(s, fmt.Sprintf("the shell to make the menu %q of the menu of the application", trayMenu), trayWait, func() bool {
		labels = menuOf(s, sh, key)
		return labels == strings.Join(trayMenu, "|")
	})

	s.Step("the shell chooses Open vfox+")
	click(s, sh, key, menuOpen)
	waitForTheWindow(s)
	s.WaitFor("the shell to take the icon out of its panel", trayWait, func() bool { return !visible(s, sh, key) })
	s.Check(host.Registered() == 0, "the icon goes when the window comes back")

	s.Step("close the window again")
	closeTheWindow(s)
	second := nextIcon(s, host)
	secondProperties := checkIcon(s, second)
	secondKey := panelKeyOf(s, sh, second)
	s.Check(secondKey != key && secondProperties.ID != properties.ID, "the icon that comes back is a new one, with a place of its own in the panel")
	poll(s, "the shell to show the new icon", trayWait, func() bool { return visible(s, sh, secondKey) })
	poll(s, "the shell to make the menu of the new icon", trayWait, func() bool { return menuOf(s, sh, secondKey) == strings.Join(trayMenu, "|") })

	s.Step("the shell chooses Open the web interface")
	click(s, sh, secondKey, menuWeb)
	link := waitForTheBrowser(s)
	s.Check(strings.HasPrefix(link, "http://127.0.0.1:") && strings.Contains(link, "/auth/connect/"), "the browser is told to open a link that pairs it: "+link)
	checkLink(s, link)
	s.Check(!s.HasWindow() && s.Alive(), "opening the web interface leaves the application as it is")

	s.Step("the shell chooses Quit vfox+")
	click(s, sh, secondKey, menuQuit)
	s.Check(s.WaitExit(trayWait), "the application ends")
	s.Check(s.ExitError() == nil, fmt.Sprintf("it ends with the code 0 (%v)", s.ExitError()))
	// The application is gone: what is waited for is the shell, which learns it from the bus.
	poll(s, "the shell to take the icon of the application that ended out of its panel", trayWait, func() bool { return !visible(s, sh, secondKey) })
	alive, err := sh.eval("1 + 1")
	s.Check(err == nil && alive == "2", fmt.Sprintf("the shell is still up after all this (%q, %v)", alive, err))
}

// poll waits for a condition, as WaitFor does, but does not mind that the application has ended: for what is
// waited for after it has.
func poll(s *appsession.Session, description string, timeout time.Duration, condition func() bool) {
	for deadline := time.Now().Add(timeout); !condition(); time.Sleep(100 * time.Millisecond) {
		if time.Now().After(deadline) {
			s.Fail("Timed out waiting for %s", description)
		}
	}
}

// menuOf is the menu that the shell has made for the indicator, as the labels of its items joined by |, a separator
// having none.
func menuOf(s *appsession.Session, sh shellOf, key string) string {
	labels, err := sh.text("(() => { const i = " + indicator(key) + "; return i === undefined ? '' : i.menu._getMenuItems().map(m => m.label ? m.label.text : '').join('|'); })()")
	if err != nil {
		s.Fail("%v", err)
	}
	return labels
}

// visible says whether the panel of the shell shows the indicator: it does not when the shell has
// taken it away, or hidden it, as it does when the application says that its icon is passive.
func visible(s *appsession.Session, sh shellOf, key string) bool {
	shown, err := sh.flag("(() => { const i = " + indicator(key) + "; return i !== undefined && i.visible; })()")
	if err != nil {
		s.Fail("%v", err)
	}
	return shown
}

// click has the extension of the shell do what it does when the user chooses an entry of the menu
// of the icon: send the event "clicked" for the entry to the application. It runs the code of the
// extension that does it (the client of the menu that the shell made), and waits for it.
func click(s *appsession.Session, sh shellOf, key string, entry int) {
	script := fmt.Sprintf("globalThis.__vfoxPlusClick = 'pending'; "+
		"import('gi://GLib').then(m => { const i = %s.menu._getMenuItems()[%d]; "+
		"return i._dbusItem.handleEvent('clicked', m.default.Variant.new('i', 0), 0); })"+
		".then(() => { globalThis.__vfoxPlusClick = 'sent'; })"+
		".catch(e => { globalThis.__vfoxPlusClick = 'failed: ' + e; }); 'started'", indicator(key), entry)
	if _, err := sh.eval(script); err != nil {
		s.Fail("%v", err)
	}
	s.WaitFor("the extension to send the click", 10*time.Second, func() bool {
		state, err := sh.text("globalThis.__vfoxPlusClick")
		if err != nil {
			s.Fail("%v", err)
		}
		if strings.HasPrefix(state, "failed") {
			s.Fail("the extension could not send the click: %s", state)
		}
		return state == "sent"
	})
}
