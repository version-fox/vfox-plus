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

// The installed tab of an SDK's page: one row for each version that is installed,
// with the version in use, where the version comes from, and its aliases.

import (
	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// sdkInstalledTab is the installed tab: the versions of the plugin, a page at a time,
// or what to do when there is none, which is to browse the versions that can be
// installed. The versions are the plugin's in the inventory; where each comes from
// and its aliases are read as well, and are added to the rows when they arrive.
func (r *Root) sdkInstalledTab(k kit.Ctx, name string, tab *string) {
	column := ui.Column(k.Context).Grow(1).Shrink(1)
	limit := ui.Local(column, "sdkVersionsLimit", func() int { return sdkPageSize })
	column.Children(func() {
		r.sdkInventory(k, func(inv remote.Inventory) {
			sdk, _ := sdkFindInstalled(inv.Installed.Value, name)
			if len(sdk.Versions) == 0 {
				r.sdkNoVersions(k, tab)
				return
			}
			r.sdkVersions(k, name, sdk, limit)
		})
	})
}

// sdkNoVersions says that no version of the plugin is installed, and switches to the
// available tab, where a version is chosen.
func (r *Root) sdkNoVersions(k kit.Ctx, tab *string) {
	kit.StatusMessage(k, kit.StatusSpec{
		Icon:  "install",
		Title: i18n.Text("noVersions"),
		Actions: []func(){func() {
			kit.Button(k, kit.ButtonSpec{
				ID:    "sdkBrowse",
				Kind:  kit.TonalButton,
				Icon:  "versions",
				Label: i18n.Text("browse"),
				OnClick: func() {
					*tab = sdkAvailable
				},
			})
		}},
	})
}

// sdkVersions is the list of the installed versions of the plugin name, the first
// page of them, with the button that shows more.
func (r *Root) sdkVersions(k kit.Ctx, name string, sdk remote.InstalledSdk, limit *int) {
	aliases, _ := r.service.Aliases(name).Value()
	profile, _ := r.service.Profile(name).Value()
	chinese := sdkChinese()
	shown := min(*limit, len(sdk.Versions))
	rows := make([]kit.ListItemSpec, 0, shown)
	for _, version := range sdk.Versions[:shown] {
		var parts string
		if profile != nil {
			parts = sdkPartsLine(profile.Installed[version], chinese)
		}
		detail := sdkJoin(" · ", parts, sdkAliasesOf(aliases, name, version))
		current := sdk.HasCurrent && sdk.Current == version
		rows = append(rows, r.sdkVersionRow(k, name, version, current, detail))
	}
	ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
		ui.Column(k.Context).Padding(0, 16, 16, 16).Children(func() {
			kit.ListGroup(k, rows...)
			if shown < len(sdk.Versions) {
				sdkShowMore(k, "sdkVersions", len(sdk.Versions)-shown, limit)
			}
		})
	})
}

// sdkVersionRow is the row of one installed version: the version, a tag when it is
// the one in use, and a second line with its source, its flavours and its aliases.
// Its actions are a button that sets the version as the one in use (not offered for
// the version in use already), and a menu with reinstalling, naming an alias and
// uninstalling.
func (r *Root) sdkVersionRow(k kit.Ctx, name, version string, current bool, detail string) kit.ListItemSpec {
	return kit.ListItemSpec{
		ID:       version,
		Name:     version,
		Subtitle: detail,
		TitleWidget: func(foreground ui.Color) {
			ui.Row(k.Context).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				kit.Label(k, version, kit.Type(kit.BodyLarge), foreground, kit.Monospace(), kit.Breakable())
				if current {
					kit.Tag(k, kit.TagSpec{Label: i18n.Text("current"), Emphasized: true})
				}
			})
		},
		Trailing: func() {
			ui.Row(k.Context).Gap(4).AlignItems(ui.Center).Children(func() {
				if !current {
					kit.Button(k, kit.ButtonSpec{
						ID:    "sdkUse",
						Kind:  kit.TonalButton,
						Icon:  "globalScope",
						Label: i18n.Text("use"),
						OnClick: func() {
							r.sdkRun(remote.TaskRequest{Operation: remote.RequestUse, Sdk: name, Version: version, Scope: "global"})
						},
					})
				}
				kit.IconButton(k, kit.IconButtonSpec{
					ID:      "sdkVersionMenu",
					Icon:    "more",
					Tooltip: i18n.Text("actions"),
					Menu: []kit.MenuItem{
						{
							Label: i18n.Text("versionReinstall"),
							Icon:  "refresh",
							OnClick: func() {
								r.sdkRun(remote.TaskRequest{Operation: remote.RequestInstall, Sdk: name, Version: version, Reinstall: true})
							},
						},
						{
							Label:   i18n.Text("aliasSet"),
							Icon:    "alias",
							OnClick: func() { r.sdkOpenAlias(name, version) },
						},
						{
							Label:   i18n.Text("uninstall"),
							Icon:    "uninstall",
							Danger:  true,
							OnClick: func() { r.sdkConfirmUninstall(name, version) },
						},
					},
				})
			})
		},
	}
}
