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
	"image/color"
	"strings"

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/shapes"
)

// twoPaneWidth is the width in dp from which the SDKs page shows the list and
// the versions of the selected plugin next to each other.
const twoPaneWidth = 840

// sdksPage lists the installed plugins and shows the versions of the one that is
// selected.
func (a *App) sdksPage() kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		sdks, ok := a.model.Installed.Value()
		if !ok {
			return kit.Pane(a.loadingOrError(a.T("sdksLoadError"), a.model.Installed.Error(), func() { a.model.Installed.Load() }))(c)
		}
		selected := ""
		for _, sdk := range sdks {
			if sdk.Name == a.shell.selectedSdk {
				selected = sdk.Name
			}
		}
		if widthDP(c) >= twoPaneWidth {
			list := a.sdkList(sdks, selected, false)
			detail := a.selectPrompt()
			if selected != "" {
				detail = a.detailView(selected, nil)
			}
			return kit.Row(kit.FlexOpts{Cross: kit.CrossStretch},
				kit.Rigid(kit.Sized(360, 0, kit.Pane(list))),
				kit.Rigid(kit.Space(gapXL, 0)),
				kit.Expanded(1, kit.Pane(kit.SwitchView("detail", kit.FadeThrough, false, selected, detail))),
			)(c)
		}
		// One pane at a time: the list, or the plugin that was chosen from it.
		var content kit.Widget
		if selected == "" {
			content = kit.Pane(a.sdkList(sdks, "", true))
		} else {
			content = kit.Pane(a.detailView(selected, func() { a.showSdk("") }))
		}
		return kit.SwitchView("sdks", kit.SharedAxisX, selected == "", selected, content)(c)
	}
}

func (a *App) selectPrompt() kit.Widget {
	return kit.StatusMessage{Icon: "sdks", Shape: shapes.ShapeGem, Title: a.T("sdksSelectPrompt")}.W()
}

// sdkList is the list of the installed plugins, with a field that searches it.
func (a *App) sdkList(sdks []core.InstalledSdk, selected string, chevrons bool) kit.Widget {
	if len(sdks) == 0 {
		return kit.StatusMessage{
			Icon: "empty", Shape: shapes.ShapeFlower, Title: a.T("sdksEmptyTitle"), Body: a.T("sdksEmptyBody"),
			Actions: []kit.Widget{kit.Button{
				ID: "add-plugin", Kind: kit.FilledButton, Icon: "add", Label: a.T("navAddPlugin"),
				OnClick: func() { a.navigate(destPlugins) },
			}.W()},
		}.W()
	}
	return func(c kit.Ctx) layout.Dimensions {
		cs := colors(c)
		query := strings.ToLower(strings.TrimSpace(a.pages.sdksQuery))
		var visible []core.InstalledSdk
		for _, sdk := range sdks {
			if query == "" || strings.Contains(strings.ToLower(sdk.Name), query) {
				visible = append(visible, sdk)
			}
		}
		header := kit.PageHeader(a.T("navSdks"), "", nil,
			a.refreshButton("refresh", a.model.Installed.IsLoading(), func() { a.model.Installed.Load() }))
		search := kit.Pad(kit.Inset{Left: gapL, Right: gapL, Bottom: gapM}, kit.SearchBar{
			ID: "search", Hint: a.T("sdksSearchHint"), Value: a.pages.sdksQuery, ClearTooltip: a.T("actionClose"),
			OnChange: func(text string) { a.pages.sdksQuery = text },
		}.W())
		if len(visible) == 0 {
			return headerAndBody(kit.Column(kit.FlexOpts{Cross: kit.CrossStretch, Min: true}, kit.Rigid(header), kit.Rigid(search)),
				kit.StatusMessage{Icon: "notFound", Title: a.T("sdksNoMatch", "query", strings.TrimSpace(a.pages.sdksQuery))}.W())(c)
		}
		return kit.List("sdks", kit.ListSource{
			Count: len(visible) + 2,
			Key: func(i int) any {
				if i < 2 {
					return i
				}
				return "sdk:" + visible[i-2].Name
			},
			Item: func(c kit.Ctx, i int) layout.Dimensions {
				switch i {
				case 0:
					return header(c)
				case 1:
					return search(c)
				}
				sdk := visible[i-2]
				subtitle := a.T("sdksNoCurrent")
				if sdk.HasCurrent {
					subtitle = a.T("sdksCurrent", "version", sdk.Current)
				}
				subtitle += "\n" + a.T("sdksVersionCount", "count", len(sdk.Versions))
				item := kit.ListItem{
					ID: "sdk:" + sdk.Name, Title: sdk.Name, Subtitle: subtitle, SubtitleLines: 2,
					Position: kit.PositionOf(i-2, len(visible)), Selected: sdk.Name == selected,
					Leading: func(_ color.NRGBA) kit.Widget { return avatar(sdk.Name, 40) },
					OnClick: func() { a.showSdk(sdk.Name) },
				}
				if chevrons {
					item.Trailing = kit.Icon("chevronRight", 24, cs.OnSurfaceVariant, 0)
				}
				return kit.Pad(kit.Inset{Left: gapL, Right: gapL, Bottom: hairline}, item.W())(c)
			},
		})(c)
	}
}
