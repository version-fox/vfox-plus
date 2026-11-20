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

package screens

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// homeCommand is the line that puts vfox in the shell. It is a command, not a text of
// the interface, so it is not translated.
const homeCommand = `eval "$(vfox activate bash)"`

// homeCopiedFor is how long the copy button shows that the command was copied.
const homeCopiedFor = 1200 * time.Millisecond

// homeState is what the home tab keeps between frames: when the command was copied.
type homeState struct {
	copied time.Time
}

// homeTab is the home tab. It shows the version of vfox with a refresh, the SDKs that
// are in use, the health of vfox and of the SDKs, and the line that puts vfox in the
// shell. Each part reads its resource without waiting; the reads load what they need.
func (r *Root) homeTab(k kit.Ctx, bar bool) {
	inventory := r.service.Inventory()
	column := ui.Column(k.Context).Fill()
	state := ui.Local(column, "home", func() homeState { return homeState{} })
	column.Children(func() {
		r.homeHeader(k, inventory, bar)
		ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
			ui.Column(k.Context).Padding(8, 16, 24, 16).Gap(16).Children(func() {
				r.homeActive(k, inventory)
				r.homeHealth(k)
				r.homeShellCard(k, state)
			})
		})
	})
}

// homeHeader is the title of the home tab: the name of the app, and the version of
// vfox under it, with the button that refreshes everything. Where the system draws
// the bar, the title is its own; the version and the refresh stay in the page.
func (r *Root) homeHeader(k kit.Ctx, inventory app.Resource[remote.Inventory], bar bool) {
	subtitle := homeVersionLine(inventory)
	refresh := func() {
		kit.IconButton(k, kit.IconButtonSpec{
			ID:      "refresh",
			Icon:    "refresh",
			Tooltip: i18n.Text("refresh"),
			OnClick: r.service.Refresh,
		})
	}
	if bar {
		kit.PageHeader(k, i18n.Text("appName"), subtitle, nil, refresh)
		return
	}
	ui.Row(k.Context).Padding(16, 16, 8, 16).Gap(8).Children(func() {
		kit.Label(k, subtitle, kit.Type(kit.TitleMedium), k.Colors().OnSurfaceVariant).Grow(1).Shrink(1)
		refresh()
	})
}

// homeVersionLine returns the subtitle of the home tab: the version of vfox, or the
// message that vfox is missing. A failed inventory leaves the subtitle empty: the
// sections below say why nothing is listed.
func homeVersionLine(inventory app.Resource[remote.Inventory]) string {
	value, loaded := inventory.Value()
	switch {
	case loaded && value.Vfox.Version != "":
		return strings.ReplaceAll(i18n.Text("homeVersion"), "{version}", value.Vfox.Version)
	case loaded:
		return i18n.Text("vfoxMissing")
	case inventory.Err() != nil:
		return ""
	}
	return i18n.Text("loading")
}

// homeSection is the heading of a part of the home tab.
func homeSection(k kit.Ctx, title string) {
	kit.Label(k, title, kit.Type(kit.TitleMedium), k.Colors().OnSurface)
}

// homeActive lists the SDKs in use: each one's current version, which opens the SDK.
// The list is the installed SDKs whose version is set, from the inventory.
func (r *Root) homeActive(k kit.Ctx, inventory app.Resource[remote.Inventory]) {
	cs := k.Colors()
	homeSection(k, i18n.Text("active"))
	value, loaded := inventory.Value()
	switch {
	case !loaded && inventory.Err() != nil:
		kit.Label(k, noticeText(app.KindOf(inventory.Err())), kit.Type(kit.BodyMedium), cs.Error)
		return
	case !loaded:
		kit.Label(k, i18n.Text("loading"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
		return
	case value.Installed.Error != "":
		kit.Label(k, value.Installed.Error, kit.Type(kit.BodyMedium), cs.Error)
		return
	}
	var items []kit.ListItemSpec
	for _, sdk := range value.Installed.Value {
		if !sdk.HasCurrent {
			continue
		}
		name, current := sdk.Name, sdk.Current
		items = append(items, kit.ListItemSpec{
			ID:       name,
			Name:     name,
			Leading:  func(ui.Color) { sdkAvatar(k, name, 40) },
			Title:    name,
			Subtitle: current,
			Trailing: func() { kit.Icon(k, "chevronRight", 24, cs.OnSurfaceVariant, 0) },
			OnClick:  func() { r.app.Push(app.Page{Kind: app.PageSDK, Name: name}) },
		})
	}
	if len(items) == 0 {
		kit.Label(k, i18n.Text("activeEmpty"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
		return
	}
	kit.ListGroup(k, items...)
}

// homeShellCard is the card with the line that puts vfox in the shell, and the button
// that copies it. The button shows a check mark for a moment after the copy.
func (r *Root) homeShellCard(k kit.Ctx, state *homeState) {
	cs := k.Colors()
	services := k.Services()
	copied := !state.copied.IsZero() && time.Since(state.copied) < homeCopiedFor
	icon, tooltip := "copy", i18n.Text("copyCommand")
	if copied {
		icon = "check"
	}
	kit.Card(k, kit.CardSpec{
		ID:      "shellCommand",
		Color:   cs.SurfaceContainerHigh,
		Radius:  16,
		Content: cs.OnSurface,
		Name:    i18n.Text("homeShellHint"),
		Children: func() {
			ui.Column(k.Context).Padding(16).Gap(8).Children(func() {
				kit.Label(k, i18n.Text("homeShellHint"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
				ui.Row(k.Context).Gap(8).Children(func() {
					kit.Label(k, homeCommand, kit.Type(kit.BodyLarge), cs.OnSurface, kit.Monospace(), kit.Breakable(), kit.Selectable()).Grow(1).Shrink(1)
					kit.IconButton(k, kit.IconButtonSpec{
						ID:      "copyCommand",
						Icon:    icon,
						Tooltip: tooltip,
						OnClick: func() {
							services.WriteClipboard(homeCommand)
							state.copied = time.Now()
							// The window is drawn again when the mark goes back to a copy.
							time.AfterFunc(homeCopiedFor, services.Invalidate)
						},
					})
				})
			})
		},
	})
}
