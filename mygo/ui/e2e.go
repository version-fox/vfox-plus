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

//go:build e2e

package ui

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"

	"github.com/version-fox/vfox-plus/mygo/core/profile"
)

// startE2E prepares the application for the end-to-end tests, which vfxtool runs
// through the automation plugin of mygo (see mygo/automation_e2e.go, and
// MYGO_AUTOMATION, which names the socket): the offline test plugin gets a
// profile that takes an argument, like vfox-java does. The scenarios themselves
// are in shared/tooling; the application has no code for them.
func (a *App) startE2E() {
	a.closeOnSignal()
	a.followControlFile()
	a.writeLink()
	if a.environment["MYGO_AUTOMATION"] == "" {
		return
	}
	a.model.Profiles.Register(testsdkProfile())
}

// closeWindow closes the window, as its close button does: the memory of the lightweight
// mode is measured after that (`vfxtool measure-lightweight`).
func (a *App) closeWindow() {
	if win, ok := a.window().(*mygo.Window); ok {
		win.Close()
	}
}

// followControlFile does what the file that VFOX_PLUS_E2E_CONTROL names asks, once it appears: the
// tools of the end-to-end tests write a word into it and the application removes it. The words are
// what a person does, where no signal says it (Windows has none): "close" closes the window, as
// its close button does, "show" chooses *Open vfox+* of the icon in the tray, and "quit" chooses
// *Quit vfox+* of it.
func (a *App) followControlFile() {
	path := a.environment["VFOX_PLUS_E2E_CONTROL"]
	if path == "" {
		return
	}
	go func() {
		for range time.Tick(100 * time.Millisecond) {
			request, err := os.ReadFile(path) //nolint:gosec // The file of this run, in the sandbox of the test.
			if err != nil || os.Remove(path) != nil {
				continue
			}
			switch strings.TrimSpace(string(request)) {
			case "close":
				a.closeWindow()
			case "show":
				a.showWindow()
			case "quit":
				a.desktop.quit()
			}
		}
	}()
}

// reportTray says in the log whether the desktop gave the icon in the tray a place on the screen,
// which the tools of the end-to-end tests read: it is the one thing that Windows tells the
// application about its icon. A Linux panel draws the icon itself and tells nothing, so there the
// tools ask the host of the tray.
func (a *App) reportTray(icon trayIcon) {
	native, ok := icon.(*nativeTray)
	if !ok || runtime.GOOS == "linux" {
		return
	}
	go func() {
		for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(200 * time.Millisecond) {
			if bounds := native.icon.Bounds(); bounds.Width > 0 && bounds.Height > 0 {
				log.Printf("e2e: the icon is in the tray at %d,%d, %d by %d", bounds.X, bounds.Y, bounds.Width, bounds.Height)
				return
			}
		}
		log.Print("e2e: the icon has no place in the tray")
	}()
}

// writeLink writes a link that pairs a browser with the web interface to the file that
// VFOX_PLUS_E2E_LINK names, once the web interface is up. The web interface is switched on
// by the settings and has no command line to print the link on, so the tools that measure
// the page of the web interface (`vfxtool measure-lightweight`, `vfxtool webheap`) read it
// from there.
func (a *App) writeLink() {
	path := a.environment["VFOX_PLUS_E2E_LINK"]
	if path == "" {
		return
	}
	go func() {
		for deadline := time.Now().Add(time.Minute); time.Now().Before(deadline); time.Sleep(50 * time.Millisecond) {
			if !a.web.Status().Running {
				continue
			}
			link, err := a.web.LocalLink()
			if err != nil {
				return
			}
			// The file appears whole: a tool that polls for it must not read half a link.
			temporary := filepath.Join(filepath.Dir(path), ".link.tmp")
			if os.WriteFile(temporary, []byte(link+"\n"), 0o600) == nil {
				_ = os.Rename(temporary, path)
			}
			return
		}
	}()
}

// testsdkProfile is the profile of the offline test plugin, which takes an argument
// like vfox-java does: `lts` lists the releases with long-term support only.
func testsdkProfile() profile.Profile {
	return profile.Profile{
		Plugin: "testsdk",
		Search: &profile.Search{
			Label: profile.Text{EN: "Channel"}, Default: "all",
			Choices: []profile.Choice{
				{Value: "all", Name: profile.Text{EN: "Every version"}, Popular: true},
				{Value: "lts", Name: profile.Text{EN: "Long-term support"}, Popular: true},
			},
		},
		Shorthands: []profile.Shorthand{{Name: "1.0.0", Meaning: profile.Text{EN: "The first release"}}},
		Facets:     []profile.Facet{{ID: "support", Label: profile.Text{EN: "Support"}, Kind: profile.FacetNote, Values: []string{"LTS"}}},
	}
}

// allowsSeveralInstances tells whether more than one application may run: the end-to-end
// tests start their own, whatever else runs on the machine of the developer.
func (a *App) allowsSeveralInstances() bool { return a.environment["MYGO_AUTOMATION"] != "" }

// windowSize is the size of the window in dp: VFOX_PLUS_E2E_WINDOW, as WIDTHxHEIGHT,
// opens the window of the compact layout for its screenshots.
func (a *App) windowSize() (width, height int) {
	width, height = defaultWidth, defaultHeight
	if w, h, ok := strings.Cut(a.environment["VFOX_PLUS_E2E_WINDOW"], "x"); ok {
		wn, werr := strconv.Atoi(w)
		hn, herr := strconv.Atoi(h)
		if werr == nil && herr == nil && wn > 0 && hn > 0 {
			return wn, hn
		}
	}
	return width, height
}
