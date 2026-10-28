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
	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// compactWidth is the width in dp below which the navigation moves to the bottom
// (docs/en/dev/ui-spec.md#window-and-navigation).
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
		a.invalidate()
	}
}

// showSdk opens the SDKs page with the plugin selected, or without any.
func (a *App) showSdk(name string) {
	a.shell.destination = destSdks
	a.shell.selectedSdk = name
	a.invalidate()
}

// pageKey tells pages apart for the transition between them.
type pageKey struct {
	destination destination
	unavailable bool
}

// railWidth is the width in dp of the navigation rail once it has settled.
func (a *App) railWidth() float32 {
	if a.env.Compact {
		return 0
	}
	if a.model.Settings.Settings().RailExpanded {
		return 240
	}
	return 96
}

// pageWidth is the width in dp that a page is built in: the window without the
// navigation rail and the space around the page.
func (a *App) pageWidth(k kit.Ctx) float32 {
	width, _ := k.Size()
	if a.env.Compact {
		return width
	}
	return width - a.railWidth() - gapL
}

// layoutShell builds the navigation and the page.
func (a *App) layoutShell(k kit.Ctx) {
	current := a.shell.destination
	// The model starts without probing vfox; the first page that needs it
	// triggers the probe here, off the drawing goroutine. Settings opens
	// without it and only probes for its version and configuration section.
	if current != destSettings && a.model.Vfox.Availability() == state.VfoxUnknown {
		go a.model.Vfox.EnsureChecked()
	}
	unavailable := a.model.Vfox.Availability() != state.VfoxReady && current != destSettings
	page := func() {
		if unavailable {
			a.unavailablePage(k)
			return
		}
		switch current {
		case destHome:
			a.homePage(k)
		case destSdks:
			a.sdksPage(k)
		case destPlugins:
			a.pluginsPage(k)
		case destTasks:
			a.tasksPage(k)
		default:
			a.settingsPage(k)
		}
	}
	destinations := []kit.NavDestination{
		{Icon: "home", Label: a.T("navHome")},
		{Icon: "sdks", Label: a.T("navSdks")},
		{Icon: "plugins", Label: a.T("navPlugins")},
		{Icon: "tasks", Label: a.T("navTasks"), Badge: a.model.Tasks.ActiveCount()},
		{Icon: "settings", Label: a.T("navSettings")},
	}
	onSelect := func(index int) { a.navigate(destination(index)) }
	if a.env.Compact {
		mui.Column(k.Context).Fill().Draw(a.noteRenderer).Children(func() {
			kit.SwitchView(k, kit.FadeThrough, 0, pageKey{current, unavailable}, page).Grow(1)
			kit.NavBar(k, kit.NavBarSpec{ID: "bar", Destinations: destinations, Selected: int(current), OnSelect: onSelect})
		})
		return
	}
	rail := kit.NavRailSpec{
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
	mui.Row(k.Context).Fill().AlignItems(mui.Stretch).Draw(a.noteRenderer).Children(func() {
		kit.NavRail(k, rail)
		kit.SwitchView(k, kit.FadeThrough, 0, pageKey{current, unavailable}, page).Grow(1).Shrink(1).
			Margin(gapL, gapL, gapL, 0)
	})
}

// noteRenderer records whether the frame is drawn by a GPU, for the transitions
// that the next builds start (kit.Env.Software). The painter tells only while
// it paints, so the first build assumes a GPU.
func (a *App) noteRenderer(p *mui.Painter, _ mui.Rect) { a.env.Software = !p.Accelerated() }

// headerAndBody puts a header above a body that takes the rest of the space, for
// pages that show a message instead of their content.
func headerAndBody(k kit.Ctx, header, body func()) {
	header()
	mui.Column(k.Context).Grow(1).Shrink(1).Children(body)
}
