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

//go:build !linux

package ui

import (
	"github.com/egoist/mygo"
)

// nativeTray is the icon in the tray that mygo draws: in the notification area of Windows, and in
// the menu bar of macOS.
type nativeTray struct{ icon *mygo.Tray }

// newNativeTray puts an icon in the tray of the desktop. The ID keeps the place that the user gave
// the icon, as the desktop keeps it for an icon of the application by its ID.
func newNativeTray(options trayOptions) (trayIcon, error) {
	icon, err := mygo.NewTray(mygo.TrayOptions{
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

// place is where the icon is on the screen, once the desktop has it there (see placedTray).
func (t *nativeTray) place() (x, y, width, height int, ok bool) {
	bounds := t.icon.Bounds()
	return bounds.X, bounds.Y, bounds.Width, bounds.Height, bounds.Width > 0 && bounds.Height > 0
}

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
