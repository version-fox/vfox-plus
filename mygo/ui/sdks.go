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
	"strings"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// twoPaneWidth is the width in dp from which the SDKs page shows the list and the
// versions of the selected plugin next to each other.
const twoPaneWidth = 840

// listPaneWidth is the width in dp of the list of plugins next to the details.
const listPaneWidth = 360

// sdksPage lists the installed plugins and shows the versions of the one that is
// selected.
func (a *App) sdksPage(k kit.Ctx) {
	sdks, ok := a.model.Installed.Value()
	if !ok {
		kit.Pane(k, func() {
			a.loadingOrError(k, a.T("sdksLoadError"), a.model.Installed.Error(), func() { a.model.Installed.Load() })
		}).Grow(1)
		return
	}
	selected := ""
	for _, sdk := range sdks {
		if sdk.Name == a.shell.selectedSdk {
			selected = sdk.Name
		}
	}
	width := a.pageWidth(k)
	if width >= twoPaneWidth {
		a.pages.detailWidth = width - listPaneWidth - gapXL
		mui.Row(k.Context).Grow(1).AlignItems(mui.Stretch).Gap(gapXL).Children(func() {
			kit.Pane(k, func() { a.sdkList(k, sdks, selected, false) }).Width(listPaneWidth).Shrink(0)
			kit.Pane(k, func() {
				kit.SwitchView(k, kit.FadeThrough, 0, selected, func() {
					if selected == "" {
						a.selectPrompt(k)
					} else {
						a.detailView(k, selected, nil)
					}
				}).Grow(1)
			}).Grow(1).Shrink(1)
		})
		return
	}
	// One pane at a time: the list, or the plugin that was chosen from it.
	a.pages.detailWidth = width
	side := float32(-1)
	if selected != "" {
		side = 1
	}
	kit.SwitchView(k, kit.SharedAxisX, side, selected, func() {
		kit.Pane(k, func() {
			if selected == "" {
				a.sdkList(k, sdks, "", true)
			} else {
				a.detailView(k, selected, func() { a.showSdk("") })
			}
		}).Grow(1)
	}).Grow(1)
}

func (a *App) selectPrompt(k kit.Ctx) {
	kit.StatusMessage(k, kit.StatusSpec{Icon: "sdks", Shape: shapes.ShapeGem, Title: a.T("sdksSelectPrompt")})
}

// sdkList is the list of the installed plugins, with a field that searches it.
func (a *App) sdkList(k kit.Ctx, sdks []core.InstalledSdk, selected string, chevrons bool) {
	if len(sdks) == 0 {
		kit.StatusMessage(k, kit.StatusSpec{
			Icon: "empty", Shape: shapes.ShapeFlower, Title: a.T("sdksEmptyTitle"), Body: a.T("sdksEmptyBody"),
			Actions: []func(){func() {
				kit.Button(k, kit.ButtonSpec{
					ID: "add-plugin", Kind: kit.FilledButton, Icon: "add", Label: a.T("navAddPlugin"),
					OnClick: func() { a.navigate(destPlugins) },
				})
			}},
		})
		return
	}
	cs := k.Colors()
	query := strings.ToLower(strings.TrimSpace(a.pages.sdksQuery))
	var visible []core.InstalledSdk
	for _, sdk := range sdks {
		if query == "" || strings.Contains(strings.ToLower(sdk.Name), query) {
			visible = append(visible, sdk)
		}
	}
	header := func() {
		kit.PageHeader(k, a.T("navSdks"), "", nil, func() {
			a.refreshButton(k, "refresh", a.model.Installed.IsLoading(), func() { a.model.Installed.Load() })
		})
	}
	search := func() mui.Element {
		return kit.SearchBar(k, kit.SearchBarSpec{
			ID: "search", Hint: a.T("sdksSearchHint"), Text: &a.pages.sdksQuery, ClearTooltip: a.T("actionClose"),
		}).Margin(0, gapL, gapM, gapL)
	}
	if len(visible) == 0 {
		headerAndBody(k, func() {
			header()
			search()
		}, func() {
			kit.StatusMessage(k, kit.StatusSpec{Icon: "notFound", Title: a.T("sdksNoMatch", "query", strings.TrimSpace(a.pages.sdksQuery))})
		})
		return
	}
	list := &a.pages.sdksList
	list.Key = func(i int) any {
		if i < 2 {
			return i
		}
		return "sdk:" + visible[i-2].Name
	}
	mui.List(k.Context, list, len(visible)+2, func(i int) {
		switch i {
		case 0:
			header()
			return
		case 1:
			search()
			return
		}
		sdk := visible[i-2]
		subtitle := a.T("sdksNoCurrent")
		if sdk.HasCurrent {
			subtitle = a.T("sdksCurrent", "version", sdk.Current)
		}
		subtitle += "\n" + a.T("sdksVersionCount", "count", len(sdk.Versions))
		item := kit.ListItemSpec{
			ID: "sdk:" + sdk.Name, Title: sdk.Name, Subtitle: subtitle, SubtitleLines: 2,
			Position: kit.PositionOf(i-2, len(visible)), Selected: sdk.Name == selected,
			Leading: func(mui.Color) { avatar(k, sdk.Name, 40) },
			OnClick: func() { a.showSdk(sdk.Name) },
		}
		if chevrons {
			item.Trailing = func() { kit.Icon(k, "chevronRight", 24, cs.OnSurfaceVariant, 0) }
		}
		kit.ListItem(k, item).Margin(0, gapL, hairline, gapL)
	}).Grow(1)
}
