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

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
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
func (a *App) pluginsPage(k kit.Ctx) {
	kit.Pane(k, func() {
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

		header := func() {
			kit.PageHeader(k, a.T("pluginsTitle"), a.T("pluginsSubtitle"), nil,
				func() {
					if a.model.Tasks.ActiveTask(vfox.OperationUpdateAllPlugins, "", "") != nil {
						kit.LoadingIndicator(k, kit.LoadingSpec{ID: "updating-all", Size: 40})
						return
					}
					kit.IconButton(k, kit.IconButtonSpec{
						ID: "update-all", Icon: "update", Tooltip: a.T("pluginsUpdateAll"), Disabled: len(installed) == 0,
						OnClick: func() { a.model.Run(a.model.Cli.UpdateAllPlugins()) },
					})
				},
				func() {
					a.refreshButton(k, "refresh", a.model.Registry.IsLoading(), func() { a.model.Registry.Load() })
				},
				func() {
					kit.Button(k, kit.ButtonSpec{
						ID: "custom", Kind: kit.TonalButton, Icon: "add", Label: a.T("pluginsAddCustom"), OnClick: a.askCustomPlugin,
					})
				},
			)
		}
		selected := a.pages.pluginsFilter
		if selected == "" {
			selected = filterAll
		}
		toolbar := func() {
			mui.Row(k.Context).Wrap().Gap(gapM).GapY(gapM).Margin(0, registryInset, gapL, registryInset).Children(func() {
				kit.SearchBar(k, kit.SearchBarSpec{
					ID: "search", Hint: a.T("pluginsSearchHint"), Text: &a.pages.pluginsQuery, ClearTooltip: a.T("actionClose"),
				}).Width(420).Shrink(1)
				kit.ButtonGroup(k, kit.ButtonGroupSpec{
					ID: "filter", Selected: selected, OnSelect: func(value string) { a.pages.pluginsFilter = value },
					Segments: []kit.Segment{
						{Value: filterAll, Label: a.T("pluginsFilterAll")},
						{Value: filterOfficial, Label: a.T("pluginsFilterOfficial"), Icon: "verified"},
						{Value: filterCommunity, Label: a.T("pluginsFilterCommunity")},
					},
				})
			})
		}
		top := func() {
			header()
			toolbar()
		}
		switch {
		case !loaded:
			headerAndBody(k, top, func() {
				a.loadingOrError(k, a.T("pluginsLoadError"), a.model.Registry.Error(), func() { a.model.Registry.Load() })
			})
			return
		case len(visible) == 0:
			headerAndBody(k, top, func() {
				kit.StatusMessage(k, kit.StatusSpec{Icon: "notFound", Title: a.T("pluginsNoMatch")})
			})
			return
		}

		// The grid: as many columns as fit, each at most 340 dp wide.
		available := a.pageWidth(k) - 2*registryInset
		columns := max(1, int(math.Ceil(float64(available)/float64(cardMaxWidth+cardGap))))
		rows := (len(visible) + columns - 1) / columns
		list := &a.pages.pluginsList
		list.Key = func(i int) any {
			if i < 2 {
				return i
			}
			return "row:" + visible[(i-2)*columns].Name
		}
		mui.List(k.Context, list, rows+2, func(i int) {
			switch i {
			case 0:
				header()
				return
			case 1:
				toolbar()
				return
			}
			mui.Row(k.Context).Gap(cardGap).AlignItems(mui.Stretch).Margin(0, registryInset, cardGap, registryInset).Children(func() {
				for column := range columns {
					index := (i-2)*columns + column
					if index >= len(visible) {
						mui.Box(k.Context).Grow(1).Basis(0)
						continue
					}
					plugin := visible[index]
					task := a.model.Tasks.ActiveTask(vfox.OperationAddPlugin, plugin.Name, "")
					a.pluginCard(k, plugin, installed[plugin.Name], task).Grow(1).Basis(0).Height(cardHeight)
				}
			})
		}).Grow(1)
	}).Grow(1)
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
func (a *App) pluginCard(k kit.Ctx, plugin core.AvailablePlugin, installed bool, task *state.VfoxTask) mui.Element {
	cs := k.Colors()
	background := cs.SurfaceContainer
	if installed {
		background = cs.SurfaceContainerHighest
	}
	return kit.Card(k, kit.CardSpec{
		ID: "plugin:" + plugin.Name, Color: background, Radius: kit.CornerLargeInc,
		Children: func() {
			mui.Column(k.Context).Grow(1).Padding(gapL).Children(func() {
				mui.Row(k.Context).Gap(gapM).Children(func() {
					avatar(k, plugin.Name, 44)
					mui.Column(k.Context).Grow(1).Shrink(1).AlignItems(mui.Start).Children(func() {
						mui.Row(k.Context).Gap(gapXS).Children(func() {
							kit.Line(k, plugin.Name, kit.Emph(kit.TitleMedium), cs.OnSurface).Shrink(1)
							if plugin.Official {
								kit.Tip(k, kit.Icon(k, "verified", 18, cs.Primary, 1), a.T("pluginsOfficial"))
							}
						})
						kit.Line(k, sourceOf(plugin.Homepage), kit.Type(kit.BodySmall), cs.OnSurfaceVariant)
					})
				})
				mui.Spacer(k.Context)
				mui.Row(k.Context).Children(func() {
					kit.IconButton(k, kit.IconButtonSpec{
						ID: "homepage", Icon: "openInNew", Tooltip: a.T("actionOpenHomepage"), Disabled: plugin.Homepage == "",
						OnClick: func() { a.openURL(plugin.Homepage) },
					})
					mui.Spacer(k.Context)
					switch {
					case task != nil:
						kit.LoadingIndicator(k, kit.LoadingSpec{ID: "adding", Size: 40})
					case installed:
						kit.Button(k, kit.ButtonSpec{
							ID: "added", Kind: kit.TextButton, Icon: "check", Label: a.T("pluginsAdded"),
							OnClick: func() { a.showSdk(plugin.Name) },
						})
					default:
						kit.Button(k, kit.ButtonSpec{
							ID: "add", Kind: kit.TonalButton, Icon: "add", Label: a.T("pluginsAdd"),
							OnClick: func() { a.run(a.model.Cli.AddPlugin(plugin.Name, "", "")) },
						})
					}
				})
			})
		},
	})
}
