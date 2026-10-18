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
	"math"
	"net/url"
	"strings"

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/core/vfox"
	"github.com/version-fox/vfox-plus/gio/state"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
)

// The filters of the plugin registry.
const (
	filterAll       = "all"
	filterOfficial  = "official"
	filterCommunity = "community"
)

// The cards of the registry are at most 340 dp wide and 152 dp high.
const (
	cardMaxWidth  = 340
	cardHeight    = 152
	cardGap       = gapM
	registryInset = gapXL
)

// pluginsPage lists the plugins of the registry, to add them.
func (a *App) pluginsPage() kit.Widget {
	return kit.Pane(func(c kit.Ctx) layout.Dimensions {
		if !a.pages.pluginsRequested {
			a.pages.pluginsRequested = true
			a.model.Registry.EnsureLoaded()
		}
		plugins, loaded := a.model.Registry.Value()
		installedSdks, _ := a.model.Installed.Value()
		installed := make(map[string]bool, len(installedSdks))
		for _, sdk := range installedSdks {
			installed[sdk.Name] = true
		}
		query := strings.ToLower(strings.TrimSpace(a.pages.pluginsQuery))
		var visible []core.AvailablePlugin
		for _, plugin := range plugins {
			matches := a.pages.pluginsFilter == filterAll || a.pages.pluginsFilter == "" ||
				a.pages.pluginsFilter == filterOfficial && plugin.Official ||
				a.pages.pluginsFilter == filterCommunity && !plugin.Official
			if matches && (query == "" || strings.Contains(strings.ToLower(plugin.Name), query) || strings.Contains(plugin.Homepage, query)) {
				visible = append(visible, plugin)
			}
		}

		var updateAll kit.Widget
		if a.model.Tasks.ActiveTask(vfox.OperationUpdateAllPlugins, "", "") != nil {
			updateAll = kit.LoadingIndicator{ID: "updating-all", Size: 40}.W()
		} else {
			updateAll = kit.IconButton{
				ID: "update-all", Icon: "update", Tooltip: a.T("pluginsUpdateAll"), Disabled: len(installed) == 0,
				OnClick: func() { a.model.Run(a.model.Cli.UpdateAllPlugins()) },
			}.W()
		}
		header := kit.PageHeader(a.T("pluginsTitle"), a.T("pluginsSubtitle"), nil,
			updateAll,
			a.refreshButton("refresh", a.model.Registry.IsLoading(), func() { a.model.Registry.Load() }),
			kit.Button{ID: "custom", Kind: kit.TonalButton, Icon: "add", Label: a.T("pluginsAddCustom"), OnClick: a.askCustomPlugin}.W(),
		)
		selected := a.pages.pluginsFilter
		if selected == "" {
			selected = filterAll
		}
		toolbar := kit.Pad(kit.Inset{Left: registryInset, Right: registryInset, Bottom: gapL},
			kit.Wrap{Spacing: gapM, RunSpacing: gapM, CrossCenter: true}.Layout(
				kit.MaxWidth(420, kit.SearchBar{
					ID: "search", Hint: a.T("pluginsSearchHint"), Value: a.pages.pluginsQuery, ClearTooltip: a.T("actionClose"),
					OnChange: func(text string) { a.pages.pluginsQuery = text },
				}.W()),
				kit.ButtonGroup{
					ID: "filter", Selected: selected, OnSelect: func(value string) { a.pages.pluginsFilter = value },
					Segments: []kit.Segment{
						{Value: filterAll, Label: a.T("pluginsFilterAll")},
						{Value: filterOfficial, Label: a.T("pluginsFilterOfficial"), Icon: "verified"},
						{Value: filterCommunity, Label: a.T("pluginsFilterCommunity")},
					},
				}.W(),
			))
		top := kit.Column(kit.FlexOpts{Cross: kit.CrossStretch, Min: true}, kit.Rigid(header), kit.Rigid(toolbar))

		switch {
		case !loaded:
			return headerAndBody(top, a.loadingOrError(a.T("pluginsLoadError"), a.model.Registry.Error(), func() { a.model.Registry.Load() }))(c)
		case len(visible) == 0:
			return headerAndBody(top, kit.StatusMessage{Icon: "notFound", Title: a.T("pluginsNoMatch")}.W())(c)
		}

		// The grid: as many columns as fit, each at most 340 dp wide.
		inset := c.DP(registryInset)
		gap := c.DP(cardGap)
		available := c.Constraints.Max.X - 2*inset
		columns := max(1, int(math.Ceil(float64(available)/float64(c.DP(cardMaxWidth)+gap))))
		rows := (len(visible) + columns - 1) / columns
		return kit.List("plugins", kit.ListSource{
			Count: rows + 2,
			Key: func(i int) any {
				if i < 2 {
					return i
				}
				return "row:" + visible[(i-2)*columns].Name
			},
			Item: func(c kit.Ctx, i int) layout.Dimensions {
				switch i {
				case 0:
					return header(c)
				case 1:
					return toolbar(c)
				}
				cells := make([]kit.FlexChild, columns)
				for column := range cells {
					index := (i-2)*columns + column
					if index >= len(visible) {
						cells[column] = kit.Expanded(1, kit.Space(0, 0))
						continue
					}
					plugin := visible[index]
					task := a.model.Tasks.ActiveTask(vfox.OperationAddPlugin, plugin.Name, "")
					cells[column] = kit.Expanded(1, kit.Sized(0, cardHeight, a.pluginCard(plugin, installed[plugin.Name], task)))
				}
				return kit.Pad(kit.Inset{Left: registryInset, Right: registryInset, Bottom: cardGap},
					kit.Row(kit.FlexOpts{Gap: cardGap, Cross: kit.CrossStretch}, cells...))(c)
			},
		})(c)
	})
}

// sourceOf is the repository of a plugin as owner/name, from its homepage.
func sourceOf(homepage string) string {
	parsed, err := url.Parse(homepage)
	if err != nil {
		return homepage
	}
	var segments []string
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment != "" {
			segments = append(segments, segment)
		}
	}
	if len(segments) == 0 {
		return homepage
	}
	return strings.Join(segments, "/")
}

// pluginCard is a plugin of the registry, with the button that adds it.
func (a *App) pluginCard(plugin core.AvailablePlugin, installed bool, task *state.VfoxTask) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		c = c.Scope("plugin:" + plugin.Name)
		cs := colors(c)
		var action kit.Widget
		switch {
		case task != nil:
			action = kit.LoadingIndicator{ID: "adding", Size: 40}.W()
		case installed:
			action = kit.Button{
				ID: "added", Kind: kit.TextButton, Icon: "check", Label: a.T("pluginsAdded"),
				OnClick: func() { a.showSdk(plugin.Name) },
			}.W()
		default:
			action = kit.Button{
				ID: "add", Kind: kit.TonalButton, Icon: "add", Label: a.T("pluginsAdd"),
				OnClick: func() { a.run(a.model.Cli.AddPlugin(plugin.Name, "", "")) },
			}.W()
		}
		title := []kit.FlexChild{kit.Flexible(1, kit.Line(plugin.Name, kit.Emph(kit.TitleMedium), cs.OnSurface))}
		if plugin.Official {
			title = append(title, kit.Rigid(kit.Space(gapXS, 0)),
				kit.Rigid(kit.Tip("official", a.T("pluginsOfficial"), kit.Icon("verified", 18, cs.Primary, 1))))
		}
		background := cs.SurfaceContainer
		if installed {
			background = cs.SurfaceContainerHighest
		}
		heading := kit.Row(kit.FlexOpts{Gap: gapM, Cross: kit.CrossCenter},
			kit.Rigid(avatar(plugin.Name, 44)),
			kit.Expanded(1, kit.Column(kit.FlexOpts{Cross: kit.CrossStart, Min: true},
				kit.Rigid(kit.Row(kit.FlexOpts{Cross: kit.CrossCenter, Min: true}, title...)),
				kit.Rigid(kit.Line(sourceOf(plugin.Homepage), kit.Type(kit.BodySmall), cs.OnSurfaceVariant)),
			)),
		)
		footer := kit.Row(kit.FlexOpts{Cross: kit.CrossCenter},
			kit.Rigid(kit.IconButton{
				ID: "homepage", Icon: "openInNew", Tooltip: a.T("actionOpenHomepage"), Disabled: plugin.Homepage == "",
				OnClick: func() { a.openURL(plugin.Homepage) },
			}.W()),
			kit.Expanded(1, kit.Space(0, 0)),
			kit.Rigid(action),
		)
		return kit.Card{
			Color: background, Radius: kit.CornerLargeInc,
			Child: kit.Pad(kit.All(gapL), kit.Column(kit.FlexOpts{Cross: kit.CrossStretch},
				kit.Rigid(heading), kit.Expanded(1, kit.Space(0, 0)), kit.Rigid(footer))),
		}.W()(c)
	}
}
