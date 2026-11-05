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

package ui

import (
	"bytes"
	"errors"
	"image/png"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/pair"
)

// lightweightApp is an application with its window open, whose web interface is up and whose
// lightweight mode is on: closing the window leaves it running.
func lightweightApp(t *testing.T) *testApp {
	t.Helper()
	ta := defaultApp(t)
	ta.updateSettings(func(s *settings.AppSettings) { s.WebInterface, s.Lightweight = true, true })
	ta.waitFor("the web interface", func(status pair.Status) bool { return status.Running })
	ta.app.openWindow()
	return ta
}

// labels are the labels of the entries, a separator being an empty one.
func labels(entries []trayEntry) []string {
	var out []string
	for _, entry := range entries {
		out = append(out, entry.label)
	}
	return out
}

// gone waits for the icon to be removed from the tray, which happens once the choice that
// brought the window back has returned.
func gone(t *testing.T, tray *fakeTray) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, _, destroyed := tray.state(); destroyed {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the icon is still in the tray")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestClosingTheWindowInTheLightweightModeLeavesAnIconInTheTray(t *testing.T) {
	ta := lightweightApp(t)
	a := ta.app
	if len(ta.desktop.trays) != 0 {
		t.Fatal("an application with a window has no icon in the tray")
	}

	a.windowsClosed()
	if ta.desktop.quits != 0 || a.window() != nil {
		t.Fatalf("the application runs on without its window: %d quits", ta.desktop.quits)
	}
	if len(ta.desktop.trays) != 1 {
		t.Fatalf("the icon is in the tray: %d", len(ta.desktop.trays))
	}
	tray := ta.desktop.trays[0]
	toolTip, entries, destroyed := tray.state()
	if toolTip != "vfox+ is running in the background" || destroyed {
		t.Errorf("tool tip %q, destroyed %t", toolTip, destroyed)
	}
	if got, want := labels(entries), []string{"Open vfox+", "Open the web interface", "", "Quit vfox+"}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("menu = %q, want %q", got, want)
	}
	if entries[2].click != nil {
		t.Error("the empty entry is a separator, which does nothing")
	}
	// The picture is the one that was drawn when the application was built: a PNG of the
	// size that the tray takes.
	img, err := png.Decode(bytes.NewReader(tray.options.icon))
	if err != nil {
		t.Fatalf("the icon is not a PNG: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 32 || b.Dy() != 32 {
		t.Errorf("the icon is %v, want 32 by 32", b)
	}

	// There is one icon, however often the application is asked to show it.
	a.showTray()
	if len(ta.desktop.trays) != 1 {
		t.Errorf("a second icon: %d", len(ta.desktop.trays))
	}
}

func TestClosingTheWindowQuitsWithoutAnIconWhenTheApplicationHasNothingToKeepRunning(t *testing.T) {
	ta := defaultApp(t)
	ta.app.openWindow()
	ta.app.windowsClosed()
	if ta.desktop.quits != 1 || len(ta.desktop.trays) != 0 {
		t.Errorf("%d quits and %d icons: the application is gone, and it leaves nothing in the tray", ta.desktop.quits, len(ta.desktop.trays))
	}
}

func TestTheMenuOfTheIconBringsTheWindowBackOpensTheWebInterfaceAndQuits(t *testing.T) {
	ta := lightweightApp(t)
	a := ta.app
	a.windowsClosed()
	tray := ta.desktop.trays[0]

	if !tray.choose("Open the web interface") {
		t.Fatal("there is no item to open the web interface")
	}
	if len(ta.desktop.opened) != 1 || !strings.HasPrefix(ta.desktop.opened[0], ta.web.Status().LocalURL+"/auth/connect/") {
		t.Errorf("the browser opens a link that pairs it: %v", ta.desktop.opened)
	}
	if len(ta.desktop.windows) != 1 || ta.desktop.quits != 0 {
		t.Error("opening the web interface leaves the application as it is")
	}

	if !tray.choose("Quit vfox+") || ta.desktop.quits != 1 {
		t.Fatalf("the application quits: %d", ta.desktop.quits)
	}
	// Ending the application takes the icon with it, at once.
	a.removeTray()
	if _, _, destroyed := tray.state(); !destroyed {
		t.Error("the icon is still there when the application is gone")
	}
	a.removeTray() // Nothing is left to remove.

	// Choosing the first item opens the window again and takes the icon away: this is a
	// second icon, as the first is gone.
	a.windowsClosed()
	if a.window() != nil || len(ta.desktop.trays) != 2 {
		t.Fatalf("closed again, with an icon of its own: %d", len(ta.desktop.trays))
	}
	second := ta.desktop.trays[1]
	if !second.choose("Open vfox+") {
		t.Fatal("there is no item to bring the window back")
	}
	if len(ta.desktop.windows) != 2 || a.window() == nil || a.released {
		t.Fatalf("the window is back: %d windows", len(ta.desktop.windows))
	}
	gone(t, second)
}

func TestActivatingTheIconBringsTheWindowBack(t *testing.T) {
	ta := lightweightApp(t)
	a := ta.app
	a.windowsClosed()
	tray := ta.desktop.trays[0]

	tray.options.activate()
	if len(ta.desktop.windows) != 2 || a.window() == nil {
		t.Fatalf("a double click brings the window back: %d windows", len(ta.desktop.windows))
	}
	gone(t, tray)

	// With the window open, the application has no icon, and a window that is open is
	// brought to the front.
	a.showWindow()
	if len(ta.desktop.windows) != 2 || ta.desktop.windows[1].shown.Load() != 1 {
		t.Errorf("one window, in front: %+v", ta.desktop.windows)
	}
	if len(ta.desktop.trays) != 1 {
		t.Errorf("no new icon: %d", len(ta.desktop.trays))
	}
}

func TestTheMenuOfTheIconFollowsTheLanguageThatTheSettingsChooseWhileTheWindowIsClosed(t *testing.T) {
	ta := lightweightApp(t)
	a := ta.app
	a.windowsClosed()
	tray := ta.desktop.trays[0]

	// The web interface changes the language: there is no window to draw it, and the menu
	// follows it all the same.
	ta.updateSettings(func(s *settings.AppSettings) { s.Language = settings.LanguageSimplifiedChinese })
	waitForTray(t, func() bool { toolTip, _, _ := tray.state(); return toolTip == "vfox+ 正在后台运行" })
	_, entries, _ := tray.state()
	if got, want := labels(entries), []string{"打开 vfox+", "打开网页界面", "", "退出 vfox+"}; strings.Join(got, "|") != strings.Join(want, "|") {
		t.Errorf("menu = %q, want %q", got, want)
	}
	updates := tray.updates

	// A setting that is not the language leaves the menu alone.
	ta.updateSettings(func(s *settings.AppSettings) { s.Theme = settings.ThemeDark })
	time.Sleep(50 * time.Millisecond)
	if tray.updates != updates {
		t.Errorf("the menu was written again for the theme: %d times", tray.updates-updates)
	}

	// And the icon that is made later speaks the language too.
	a.showWindow()
	gone(t, tray)
	a.windowsClosed()
	if toolTip, _, _ := ta.desktop.trays[1].state(); toolTip != "vfox+ 正在后台运行" {
		t.Errorf("tool tip %q", toolTip)
	}
}

func waitForTray(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the icon to change")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestADesktopWithoutATrayStillRunsTheApplicationWithoutItsWindow(t *testing.T) {
	ta := lightweightApp(t)
	a := ta.app
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	ta.desktop.trayError = errors.New("tray icons need libayatana-appindicator3")

	a.windowsClosed()
	if ta.desktop.quits != 0 || a.window() != nil || !a.released {
		t.Fatalf("the application runs on: %d quits", ta.desktop.quits)
	}
	if len(ta.desktop.trays) != 0 || a.tray != nil {
		t.Error("there is no tray to put an icon in")
	}
	if !strings.Contains(logged.String(), "no icon in the tray: tray icons need libayatana-appindicator3") {
		t.Errorf("the log says why: %q", logged.String())
	}

	// Starting the application again brings the window back, as it always did.
	a.showWindow()
	if len(ta.desktop.windows) != 2 || a.window() == nil {
		t.Fatalf("the window is back: %d windows", len(ta.desktop.windows))
	}
	// And the tray that is there the next time gets its icon.
	ta.desktop.trayError = nil
	a.windowsClosed()
	if len(ta.desktop.trays) != 1 {
		t.Errorf("the icon is in the tray: %d", len(ta.desktop.trays))
	}
}

func TestTheWebInterfaceThatCannotBeOpenedFromTheTrayIsLogged(t *testing.T) {
	ta := lightweightApp(t)
	a := ta.app
	a.windowsClosed()
	tray := ta.desktop.trays[0]
	ta.updateSettings(func(s *settings.AppSettings) { s.WebInterface = false })
	ta.waitFor("the web interface to stop", func(status pair.Status) bool { return !status.Running })
	var logged bytes.Buffer
	log.SetOutput(&logged)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })

	if !tray.choose("Open the web interface") {
		t.Fatal("the item is in the menu")
	}
	if len(ta.desktop.opened) != 0 {
		t.Errorf("there is no link to open: %v", ta.desktop.opened)
	}
	if !strings.Contains(logged.String(), "cannot open the web interface") {
		t.Errorf("there is no window to say so, the log does: %q", logged.String())
	}
}

func TestTheMenuOfTheIconCallsWhatItsEntriesDo(t *testing.T) {
	var called []string
	menu := trayMenu([]trayEntry{
		{label: "A", click: func() { called = append(called, "A") }},
		{},
		{label: "B", click: func() { called = append(called, "B") }},
	})
	items := menu.Items()
	if len(items) != 3 || items[0].Label != "A" || items[2].Label != "B" {
		t.Fatalf("items = %+v", items)
	}
	if items[1].Type != mygo.MenuItemSeparator {
		t.Errorf("the empty entry is a %v, want a separator", items[1].Type)
	}
	for _, item := range []*mygo.MenuItem{items[0], items[2], items[0]} {
		item.Click(item, nil)
	}
	if strings.Join(called, "") != "ABA" {
		t.Errorf("called %v", called)
	}
}

// shortenPageOutDelay lets a test wait for the paging out, which waits for the window to stay closed.
func shortenPageOutDelay(t *testing.T, delay time.Duration) {
	t.Helper()
	previous := pageOutDelay
	pageOutDelay = delay
	t.Cleanup(func() { pageOutDelay = previous })
}

func TestTheFilesAreTakenBackOnlyWhenTheWindowStaysClosed(t *testing.T) {
	shortenPageOutDelay(t, 150*time.Millisecond)
	ta := lightweightApp(t)
	a := ta.app

	// The window closes and stays closed: the application tells the system that it does not want
	// the pages of its files, once.
	a.windowsClosed()
	if got := ta.desktop.pagedOut.Load(); got != 0 {
		t.Fatalf("paged out at once, %d times: the window and the icon are torn down in that time", got)
	}
	deadline := time.Now().Add(5 * time.Second)
	for ta.desktop.pagedOut.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if got := ta.desktop.pagedOut.Load(); got != 1 {
		t.Fatalf("paged out %d times after the window stayed closed, want 1", got)
	}

	// The window opens and closes again before the time is up: the files are not paged out under
	// a window that runs them, and the second closing waits its own time.
	a.showWindow()
	a.windowsClosed()
	a.showWindow()
	time.Sleep(4 * pageOutDelay)
	if got := ta.desktop.pagedOut.Load(); got != 1 {
		t.Errorf("paged out %d times, want no more while a window is open", got)
	}
	if a.trim != nil {
		t.Error("the timer of a window that is open is still set")
	}
}

func TestClosingTheWindowAgainStartsTheWaitAgain(t *testing.T) {
	shortenPageOutDelay(t, 300*time.Millisecond)
	ta := lightweightApp(t)
	a := ta.app
	a.windowsClosed()
	time.Sleep(200 * time.Millisecond)
	// A second release before the time is up (the window opened and closed): one paging out, a
	// full wait after the last.
	a.releaseWindow()
	time.Sleep(200 * time.Millisecond)
	if got := ta.desktop.pagedOut.Load(); got != 0 {
		t.Fatalf("paged out %d times before the wait after the last closing was over", got)
	}
	deadline := time.Now().Add(5 * time.Second)
	for ta.desktop.pagedOut.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	time.Sleep(400 * time.Millisecond)
	if got := ta.desktop.pagedOut.Load(); got != 1 {
		t.Errorf("paged out %d times, want 1", got)
	}
}
