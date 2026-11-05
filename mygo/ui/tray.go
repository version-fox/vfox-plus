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
	"errors"
	"log"

	"github.com/egoist/mygo"

	"github.com/version-fox/vfox-plus/mygo/assets"
)

// The tray is where the application goes when its window is closed in the lightweight
// mode: an icon in the notification area, which brings the window back and ends the
// application, as the window did. The icon is there while the application has no window.
// It is made when the window is closed and removed when a window opens again, so that an
// application with a window has no icon in the tray and holds nothing for one.
//
// Not every desktop has a tray, and on Linux the icon needs a library that the desktop may
// not have installed (libayatana-appindicator3). Without it the application runs as it did:
// starting it again brings the window back.

// trayEntry is an item of the menu of the tray icon: a label, and what choosing it does.
// An entry that does nothing is a separator.
type trayEntry struct {
	label string
	click func()
}

// trayOptions say what the icon of the tray looks like and does.
type trayOptions struct {
	// icon is the picture of the icon, a PNG image.
	icon []byte
	// toolTip is the text that shows where the pointer rests on the icon.
	toolTip string
	// entries are the items of the menu that a click on the icon shows.
	entries []trayEntry
	// activate is what the icon does when the desktop activates it, which is a click on
	// Windows, and a double click on the desktops of Linux that show the menu for a single
	// one.
	activate func()
}

// trayIcon is the icon in the tray as the application uses it.
type trayIcon interface {
	// Update sets the text and the menu again, as when the language changed.
	Update(toolTip string, entries []trayEntry)
	// Destroy removes the icon.
	Destroy()
}

// nativeTray is the icon in the tray that mygo draws.
type nativeTray struct{ icon *mygo.Tray }

// errNoTrayHost says that nothing on the desktop shows tray icons.
var errNoTrayHost = errors.New("nothing on the desktop shows tray icons")

// newNativeTray puts an icon in the tray of the desktop. It fails where the desktop has none,
// or has no library to draw the icon with.
func newNativeTray(options trayOptions) (trayIcon, error) {
	if !trayHostAvailable() {
		return nil, errNoTrayHost
	}
	icon, err := mygo.NewTray(mygo.TrayOptions{
		// The name keeps the place that the user gave the icon on Windows, and on macOS.
		ID:      ApplicationID,
		Icon:    options.icon,
		ToolTip: options.toolTip,
		Menu:    trayMenu(options.entries),
	})
	if err != nil {
		return nil, err
	}
	icon.OnClick(options.activate)
	return &nativeTray{icon: icon}, nil
}

func (t *nativeTray) Update(toolTip string, entries []trayEntry) {
	t.icon.SetToolTip(toolTip)
	t.icon.SetMenu(trayMenu(entries))
}

func (t *nativeTray) Destroy() { t.icon.Destroy() }

// trayMenu makes the menu of the icon of the entries.
func trayMenu(entries []trayEntry) *mygo.Menu {
	items := make([]*mygo.MenuItem, 0, len(entries))
	for _, entry := range entries {
		if entry.click == nil {
			items = append(items, mygo.Separator())
			continue
		}
		click := entry.click
		items = append(items, &mygo.MenuItem{Label: entry.label, Click: func(*mygo.MenuItem, *mygo.Window) { click() }})
	}
	return mygo.NewMenu(items)
}

// trayEntries are the items of the menu, in the language of the user interface.
func (a *App) trayEntries() []trayEntry {
	return []trayEntry{
		{label: a.T("trayOpen"), click: a.showWindow},
		{label: a.T("trayOpenWeb"), click: a.openWebFromTray},
		{},
		{label: a.T("webQuit"), click: a.desktop.quit},
	}
}

// openWebFromTray opens the web interface in the browser, paired. There is no window to tell
// the user what went wrong, so the log does.
func (a *App) openWebFromTray() {
	link, err := a.web.LocalLink()
	if err != nil {
		log.Printf("cannot open the web interface: %v", err)
		return
	}
	a.openURL(link)
}

// showTray puts the icon of the application in the tray, if the desktop has one, and the
// application has none there yet.
func (a *App) showTray() {
	a.trayMu.Lock()
	shown := a.tray != nil
	a.trayMu.Unlock()
	if shown {
		return
	}
	picture, err := assets.Files.ReadFile("tray.png")
	if err != nil {
		log.Printf("no picture for the icon in the tray: %v", err)
		return
	}
	// Nothing has drawn a frame since the window closed, which is when the language is
	// loaded again: the settings may have changed it from the web interface.
	a.syncLanguage()
	tray, err := a.desktop.newTray(trayOptions{
		icon: picture, toolTip: a.T("trayToolTip"), entries: a.trayEntries(), activate: a.showWindow,
	})
	if err != nil {
		log.Printf("no icon in the tray: %v", err)
		return
	}
	a.trayMu.Lock()
	a.tray = tray
	a.trayMu.Unlock()
	a.reportTray(tray)
}

// refreshTray writes the menu of the icon again when the language of the settings has
// changed, which the web interface can do while the application has no window to show it.
func (a *App) refreshTray() {
	a.trayMu.Lock()
	tray := a.tray
	a.trayMu.Unlock()
	if tray == nil {
		return
	}
	before := a.l10n.Load().Language
	a.syncLanguage()
	if a.l10n.Load().Language == before {
		return
	}
	tray.Update(a.T("trayToolTip"), a.trayEntries())
}

// takeTray lets go of the icon of the application, and returns it: nil when it has none.
func (a *App) takeTray() trayIcon {
	a.trayMu.Lock()
	defer a.trayMu.Unlock()
	tray := a.tray
	a.tray = nil
	return tray
}

// hideTray removes the icon from the tray when a window opens. The window is often opened by
// choosing an item of the menu of the icon, which the libraries of the desktop do not expect
// to see destroyed under their hands, so the icon is removed once that has returned.
func (a *App) hideTray() {
	if tray := a.takeTray(); tray != nil {
		go tray.Destroy()
	}
}

// removeTray removes the icon from the tray at once, as the application ends: Windows keeps
// showing the icon of a program that is gone until the pointer passes over it.
func (a *App) removeTray() {
	if tray := a.takeTray(); tray != nil {
		tray.Destroy()
	}
}
