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

// The SDK tab: the plugins that are installed, with the version each has in use.
// The page of a plugin is in page_sdk.go and its files.

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// sdksTab is the SDK tab, which lists the plugins that are installed, with a search
// over their names. A tap opens the page of the plugin. The list reads the installed
// plugins of the inventory; the registry is the plugins tab's.
func (r *Root) sdksTab(k kit.Ctx, bar bool) {
	ui.Column(k.Context).Fill().Children(func() {
		if bar {
			kit.PageHeader(k, tabTitle(app.TabSDKs), "", nil, func() {
				kit.IconButton(k, kit.IconButtonSpec{
					ID:      "sdkRefresh",
					Icon:    "refresh",
					Tooltip: i18n.Text("refresh"),
					OnClick: r.service.Refresh,
				})
			})
		}
		r.sdkListBody(k)
	})
}

// sdkListBody is the body of the tab: the state of the inventory while it loads or
// fails, the search and the list of plugins once it has them. The search text and the
// number of rows shown are kept on the tab, so that they survive the frames.
func (r *Root) sdkListBody(k kit.Ctx) {
	column := ui.Column(k.Context).Grow(1).Shrink(1)
	query := ui.Local(column, "sdkQuery", func() string { return "" })
	limit := ui.Local(column, "sdkLimit", func() int { return sdkPageSize })
	column.Children(func() {
		r.sdkInventory(k, func(inv remote.Inventory) {
			installed := inv.Installed.Value
			if len(installed) == 0 {
				r.sdkNoPlugins(k)
				return
			}
			ui.Column(k.Context).Grow(1).Shrink(1).Children(func() {
				ui.Column(k.Context).Padding(12, 16, 4, 16).Children(func() {
					kit.SearchBar(k, kit.SearchBarSpec{
						ID:           "sdkSearch",
						Hint:         i18n.Text("searchSdks"),
						Text:         query,
						ClearTooltip: i18n.Text("sdkClearSearch"),
					})
				})
				matches := sdkMatching(installed, *query)
				if len(matches) == 0 {
					kit.StatusMessage(k, kit.StatusSpec{Icon: "search", Title: i18n.Text("noSdkMatch")})
					return
				}
				r.sdkRows(k, matches, limit)
			})
		})
	})
}

// sdkNoPlugins says that no plugin is installed, and offers the plugins tab, where
// one is added.
func (r *Root) sdkNoPlugins(k kit.Ctx) {
	kit.StatusMessage(k, kit.StatusSpec{
		Icon:  "sdks",
		Title: i18n.Text("noSdks"),
		Actions: []func(){func() {
			kit.Button(k, kit.ButtonSpec{
				ID:      "sdkAddPlugin",
				Kind:    kit.TonalButton,
				Icon:    "add",
				Label:   i18n.Text("add"),
				OnClick: func() { r.app.SelectTab(app.TabPlugins) },
			})
		}},
	})
}

// sdkMatching returns the plugins whose names contain query, without regard to case.
// An empty query matches every plugin, as the web interface's search does.
func sdkMatching(installed []remote.InstalledSdk, query string) []remote.InstalledSdk {
	needle := strings.ToLower(query)
	var matches []remote.InstalledSdk
	for _, sdk := range installed {
		if needle == "" || strings.Contains(strings.ToLower(sdk.Name), needle) {
			matches = append(matches, sdk)
		}
	}
	return matches
}

// sdkRows builds the rows of the plugins that match, the first page of them at once,
// with the button that shows the next page, in a scrolling list.
func (r *Root) sdkRows(k kit.Ctx, plugins []remote.InstalledSdk, limit *int) {
	shown := min(*limit, len(plugins))
	rows := make([]kit.ListItemSpec, 0, shown)
	for _, sdk := range plugins[:shown] {
		rows = append(rows, r.sdkRow(k, sdk))
	}
	ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
		ui.Column(k.Context).Padding(0, 16, 16, 16).Children(func() {
			kit.ListGroup(k, rows...)
			if shown < len(plugins) {
				sdkShowMore(k, "sdkPlugins", len(plugins)-shown, limit)
			}
		})
	})
}

// sdkRow is the row of a plugin: its avatar, its name, how many versions it has, the
// version in use, and a tap that opens its page. Without a version in use the count
// starts with a dash, as on the web.
func (r *Root) sdkRow(k kit.Ctx, sdk remote.InstalledSdk) kit.ListItemSpec {
	name := sdk.Name
	count := strconv.Itoa(len(sdk.Versions)) + " " + i18n.Text("versionsN")
	if !sdk.HasCurrent {
		count = i18n.Text("empty") + " · " + count
	}
	return kit.ListItemSpec{
		ID:   name,
		Name: name,
		Leading: func(ui.Color) {
			sdkAvatar(k, name, 40)
		},
		TitleWidget: func(foreground ui.Color) {
			kit.Label(k, name, kit.Type(kit.BodyLarge), foreground, kit.Monospace(), kit.Breakable())
		},
		Subtitle: count,
		Trailing: func() {
			ui.Row(k.Context).Gap(8).AlignItems(ui.Center).Children(func() {
				if sdk.HasCurrent {
					kit.Tag(k, kit.TagSpec{Label: sdk.Current, Emphasized: true})
				}
				kit.Icon(k, "chevronRight", 24, k.Colors().OnSurfaceVariant, 0)
			})
		},
		OnClick: func() {
			r.app.Push(app.Page{Kind: app.PageSDK, Name: name})
		},
	}
}
