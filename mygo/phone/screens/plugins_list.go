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

// The registry of the plugin tab under its header: the search and the filter, and the
// cards of the plugins that match them, or what the registry shows while it loads,
// when it fails, or when no plugin matches.

import (
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// pluginsBody builds the search and the filter of the registry, and below them the
// registry: its loading or failure, or the cards of the plugins that match.
func (r *Root) pluginsBody(k kit.Ctx, st *pluginsState) {
	svc := r.Service()
	inventory := svc.Inventory()
	ui.Column(k.Context).Padding(0, 16, 8, 16).Children(func() {
		kit.SearchBar(k, kit.SearchBarSpec{
			ID:           "pluginSearch",
			Hint:         i18n.Text("searchPlugins"),
			Text:         &st.query,
			ClearTooltip: i18n.Text("clearSearch"),
			OnChange:     func(string) { st.limit = availPageSize },
		})
	})
	ui.Column(k.Context).Padding(0, 16, 8, 16).Children(func() {
		kit.ButtonGroup(k, kit.ButtonGroupSpec{
			ID: "pluginFilter",
			Segments: []kit.Segment{
				{Value: pluginFilterAll, Label: i18n.Text("all")},
				{Value: pluginFilterOfficial, Label: i18n.Text("pluginFilterOfficial")},
				{Value: pluginFilterCommunity, Label: i18n.Text("community")},
			},
			Selected: st.filter,
			OnSelect: func(value string) {
				st.filter = value
				st.limit = availPageSize
			},
			Expand: true,
		})
	})

	inv, loaded := inventory.Value()
	if !loaded {
		availStatus(k, svc, inventory, i18n.Text("loading"))
		return
	}
	section := inv.Registry
	switch {
	case len(section.Value) == 0 && section.Error != "":
		availFailed(k, svc, section.Error)
		return
	case len(section.Value) == 0 && section.Loading:
		kit.LoadingMessage(k, i18n.Text("loading"), i18n.Text("loading"))
		return
	}

	installed := pluginInstalled(inv.Installed.Value)
	query := strings.ToLower(st.query)
	var shown []remote.AvailablePlugin
	for _, plugin := range section.Value {
		if pluginMatches(plugin, st.filter, query) {
			shown = append(shown, plugin)
		}
	}
	if len(shown) == 0 {
		kit.StatusMessage(k, kit.StatusSpec{Icon: "notFound", Title: i18n.Text("noPlugins")})
		return
	}
	// The list shows the first plugins that match, as the versions do, and "show more"
	// adds to them.
	visible := shown
	if len(visible) > st.limit {
		visible = visible[:st.limit]
	}
	ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
		ui.Column(k.Context).Padding(0, 16, 16, 16).Gap(12).Children(func() {
			for _, plugin := range visible {
				r.pluginCard(k, st, plugin, installed[plugin.Name])
			}
			if rest := len(shown) - len(visible); rest > 0 {
				kit.Button(k, kit.ButtonSpec{
					ID:      "showMore",
					Kind:    kit.TonalButton,
					Label:   availCount("showMore", rest),
					OnClick: func() { st.limit += availPageStep },
				})
			}
		})
	})
}

// pluginCard is the card of a plugin of the registry: its avatar, its name with the
// mark of the official plugins, its source, and its homepage. Its action opens the
// plugin when it is installed, and installs it from the registry when it is not.
func (r *Root) pluginCard(k kit.Ctx, st *pluginsState, plugin remote.AvailablePlugin, installed bool) {
	cs := k.Colors()
	name := plugin.Name
	kit.Card(k, kit.CardSpec{
		ID:      name,
		Name:    name,
		Color:   cs.SurfaceContainerLow,
		Radius:  kit.CornerLarge,
		Content: cs.OnSurface,
		Children: func() {
			ui.Column(k.Context).Padding(16).Gap(12).Children(func() {
				ui.Row(k.Context).AlignItems(ui.Center).Gap(12).Children(func() {
					sdkAvatar(k, name, 40)
					ui.Column(k.Context).Grow(1).Shrink(1).MinWidth(0).Gap(2).Children(func() {
						ui.Row(k.Context).Wrap().Gap(8).GapY(4).AlignItems(ui.Center).Children(func() {
							kit.Label(k, name, kit.Type(kit.TitleMedium), cs.OnSurface, kit.Monospace(), kit.Breakable())
							if plugin.Official {
								kit.Tag(k, kit.TagSpec{Label: i18n.Text("pluginOfficial"), Icon: "verified", Emphasized: true})
							}
						})
						if source := pluginShortSource(plugin.Homepage); source != "" {
							kit.Label(k, source, kit.Type(kit.BodyMedium), cs.OnSurfaceVariant, kit.Breakable())
						}
					})
				})
				ui.Row(k.Context).AlignItems(ui.Center).Gap(8).Children(func() {
					if plugin.Homepage != "" {
						pluginHomepage(k, plugin)
					}
					kit.Spacer(k)
					r.pluginAction(k, st, name, installed)
				})
			})
		},
	})
}

// pluginHomepage is the button that opens the homepage of a plugin in the browser. A
// homepage that is not a web address is refused with a message, as the web refuses it.
func pluginHomepage(k kit.Ctx, plugin remote.AvailablePlugin) {
	services := k.Services()
	env := k.Env
	kit.IconButton(k, kit.IconButtonSpec{
		ID:      "homepage",
		Icon:    "openInNew",
		Tooltip: i18n.Text("homepage"),
		OnClick: func() {
			address, ok := pluginWebAddress(plugin.Homepage)
			if !ok {
				env.ShowSnackbar(kit.Snack{Text: i18n.Text("linkRefused")})
				return
			}
			services.OpenURL(address)
		},
	})
}

// pluginAction is the action of a card: the plugin's SDK is opened when the plugin is
// installed, and the plugin is installed from the registry, by its name, when it is not.
func (r *Root) pluginAction(k kit.Ctx, st *pluginsState, name string, installed bool) {
	if installed {
		kit.Button(k, kit.ButtonSpec{
			ID:    "open",
			Kind:  kit.TonalButton,
			Label: i18n.Text("added"),
			OnClick: func() {
				r.App().Push(app.Page{Kind: app.PageSDK, Name: name})
			},
		})
		return
	}
	kit.Button(k, kit.ButtonSpec{
		ID:    "install",
		Kind:  kit.TonalButton,
		Label: i18n.Text("install"),
		Icon:  "install",
		OnClick: func() {
			r.availStart(st.landed, remote.TaskRequest{Operation: remote.RequestAdd, Source: name})
		},
	})
}
