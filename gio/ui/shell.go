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
	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/core/settings"
	"github.com/version-fox/vfox-plus/gio/state"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
)

// compactWidth is the width in dp below which the navigation moves to the
// bottom (docs/ui-spec.md#window-and-navigation).
const compactWidth = 600

// The gaps between elements, on a grid of 4 dp.
const (
	hairline float32 = 2
	gapXS    float32 = 4
	gapS     float32 = 8
	gapM     float32 = 12
	gapL     float32 = 16
	gapXL    float32 = 24
	gapXXL   float32 = 32
)

// destination is a top-level page of the navigation.
type destination int

const (
	destHome destination = iota
	destSdks
	destPlugins
	destTasks
	destSettings
)

// shellState is where the user is.
type shellState struct {
	destination destination
	// selectedSdk is the plugin whose versions the SDKs page shows.
	selectedSdk string
}

// navigate opens a page.
func (a *App) navigate(d destination) {
	if a.shell.destination != d {
		a.shell.destination = d
		a.env.Services.Invalidate()
	}
}

// showSdk opens the SDKs page with the plugin selected, or without any.
func (a *App) showSdk(name string) {
	a.shell.destination = destSdks
	a.shell.selectedSdk = name
	a.env.Services.Invalidate()
}

// pageKey tells pages apart for the transition between them.
type pageKey struct {
	destination destination
	unavailable bool
}

// layoutShell lays out the navigation and the page.
func (a *App) layoutShell(c kit.Ctx) layout.Dimensions {
	current := a.shell.destination
	unavailable := a.model.Vfox.Availability() != state.VfoxReady && current != destSettings
	var page kit.Widget
	if unavailable {
		page = a.unavailablePage()
	} else {
		switch current {
		case destHome:
			page = a.homePage()
		case destSdks:
			page = a.sdksPage()
		case destPlugins:
			page = a.pluginsPage()
		case destTasks:
			page = a.tasksPage()
		default:
			page = a.settingsPage()
		}
	}
	switcher := kit.SwitchView("page", kit.FadeThrough, false, pageKey{current, unavailable}, page)
	destinations := []kit.NavDestination{
		{Icon: "home", Label: a.T("navHome")},
		{Icon: "sdks", Label: a.T("navSdks")},
		{Icon: "plugins", Label: a.T("navPlugins")},
		{Icon: "tasks", Label: a.T("navTasks"), Badge: a.model.Tasks.ActiveCount()},
		{Icon: "settings", Label: a.T("navSettings")},
	}
	onSelect := func(index int) { a.navigate(destination(index)) }
	if c.Constraints.Max.X < c.DP(compactWidth) {
		bar := kit.NavBar{ID: "bar", Destinations: destinations, Selected: int(current), OnSelect: onSelect}
		return kit.Column(kit.FlexOpts{Cross: kit.CrossStretch},
			kit.Expanded(1, switcher), kit.Rigid(bar.W()))(c)
	}
	rail := kit.NavRail{
		ID: "rail", Destinations: destinations, Selected: int(current), OnSelect: onSelect,
		Expanded: a.model.Settings.Settings().RailExpanded,
		OnExpandedChange: func(expanded bool) {
			a.model.Settings.Update(func(s settings.AppSettings) settings.AppSettings {
				s.RailExpanded = expanded
				return s
			})
		},
		ExpandTooltip: a.T("navExpand"), CollapseTooltip: a.T("navCollapse"),
	}
	if a.model.Vfox.Availability() == state.VfoxReady {
		rail.FABIcon, rail.FABLabel = "add", a.T("navAddPlugin")
		rail.OnFAB = func() { a.navigate(destPlugins) }
	}
	return kit.Row(kit.FlexOpts{},
		kit.Rigid(rail.W()),
		kit.Expanded(1, kit.Pad(kit.Inset{Top: gapL, Right: gapL, Bottom: gapL}, switcher)),
	)(c)
}

// scrollPage lays the items out one under the other in a scrolling list.
func scrollPage(id string, items ...kit.Widget) kit.Widget {
	return kit.List(id, kit.ListSource{
		Count: len(items),
		Item:  func(c kit.Ctx, i int) layout.Dimensions { return items[i](c) },
	})
}

// headerAndBody puts a header above a body that takes the rest of the space,
// for pages that show a message instead of their content.
func headerAndBody(header, body kit.Widget) kit.Widget {
	return kit.Column(kit.FlexOpts{Cross: kit.CrossStretch}, kit.Rigid(header), kit.Expanded(1, body))
}

// colors is the palette in use.
func colors(c kit.Ctx) kit.Palette { return c.Env.Theme.Colors }

// widthDP is the width of the space in dp.
func widthDP(c kit.Ctx) float32 {
	return float32(c.Constraints.Max.X) / float32(c.DP(1))
}
