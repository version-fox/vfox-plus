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

// The options tab of an SDK's page, for a plugin with a profile: the variables the
// plugin reads, the version files it may read, its aliases, and its notes and its
// documents.

import (
	"strconv"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// The keys of the vfox configuration that the version files use, and their values.
const (
	sdkLegacyEnableKey   = "legacyVersionFile.enable"
	sdkLegacyStrategyKey = "legacyVersionFile.strategy"
)

// sdkOptionsTab is the options tab. It shows what the profile holds, in the order
// of the web interface: the variables, the version files, the aliases, the notes and
// the documents. It waits for the profile, and asks again when the computer failed.
func (r *Root) sdkOptionsTab(k kit.Ctx, name string, profile app.Resource[*remote.Profile]) {
	value, loaded := profile.Value()
	switch {
	case !loaded && profile.Err() != nil:
		r.sdkFailed(k, sdkErrorText(profile.Err()))
		return
	case !loaded:
		sdkLoading(k)
		return
	case value == nil:
		return
	}
	inv, _ := r.service.Inventory().Value()
	ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
		content := ui.Column(k.Context).Padding(0, 16, 24, 16).Gap(8)
		variableLimit := ui.Local(content, "sdkVariablesLimit", func() int { return sdkPageSize })
		aliasLimit := ui.Local(content, "sdkAliasesLimit", func() int { return sdkPageSize })
		content.Children(func() {
			if len(value.Variables) > 0 {
				r.sdkVariables(k, name, value.Variables, variableLimit)
			}
			if len(value.LegacyFiles) > 0 {
				r.sdkLegacy(k, value, inv)
			}
			r.sdkAliases(k, name, aliasLimit)
			if len(value.Tips) > 0 {
				r.sdkTips(k, value.Tips)
			}
			if len(value.Docs) > 0 {
				r.sdkDocs(k, value.Docs)
			}
		})
	})
}

// sdkVariables is the list of the plugin's variables, a page at a time, each with
// what it is set to now.
func (r *Root) sdkVariables(k kit.Ctx, name string, variables []remote.Variable, limit *int) {
	shown := min(*limit, len(variables))
	chinese := sdkChinese()
	rows := make([]kit.ListItemSpec, 0, shown)
	for _, variable := range variables[:shown] {
		rows = append(rows, r.sdkVariableRow(k, name, variable, chinese))
	}
	kit.GroupHeader(k, i18n.Text("pluginVariables"), nil)
	kit.ListGroup(k, rows...)
	if shown < len(variables) {
		sdkShowMore(k, "sdkVariables", len(variables)-shown, limit)
	}
}

// sdkVariableRow is the row of a variable: its label, a tag when it is advanced, its
// key, what it is set to, and the button that changes it. A tap on the row changes it
// too.
func (r *Root) sdkVariableRow(k kit.Ctx, name string, variable remote.Variable, chinese bool) kit.ListItemSpec {
	title := variable.Label.In(chinese)
	return kit.ListItemSpec{
		ID:      variable.Key,
		Name:    title,
		OnClick: func() { r.sdkOpenVariable(name, variable) },
		TitleWidget: func(foreground ui.Color) {
			ui.Row(k.Context).Gap(8).AlignItems(ui.Center).Wrap().Children(func() {
				kit.Label(k, title, kit.Type(kit.BodyLarge), foreground, kit.Breakable())
				if variable.Advanced {
					kit.Tag(k, kit.TagSpec{Label: i18n.Text("advanced")})
				}
			})
		},
		SubtitleWidget: func(foreground ui.Color) {
			ui.Column(k.Context).Gap(2).AlignItems(ui.Start).Children(func() {
				kit.Label(k, variable.Key, kit.Type(kit.BodyMedium), foreground, kit.Monospace(), kit.Breakable())
				kit.Label(k, sdkVariableValue(variable, chinese), kit.Type(kit.BodyMedium), foreground, kit.Breakable())
			})
		},
		Trailing: func() {
			kit.Button(k, kit.ButtonSpec{
				ID:    "sdkChange",
				Kind:  kit.TonalButton,
				Label: i18n.Text("change"),
				OnClick: func() {
					r.sdkOpenVariable(name, variable)
				},
			})
		},
	}
}

// sdkVariableValue says what a variable is set to: the name of the preset that holds
// its value, and the value; "Default" when it is not set.
func sdkVariableValue(variable remote.Variable, chinese bool) string {
	if variable.Value == "" {
		return i18n.Text("unset")
	}
	for _, preset := range variable.Presets {
		if preset.Value == variable.Value {
			return preset.Name.In(chinese) + " · " + variable.Value
		}
	}
	return variable.Value
}

// sdkLegacy is the switch that lets vfox read the version files of other tools, and,
// when the plugin says which version a file names, the choice of that version. The
// two settings are keys of the vfox configuration.
func (r *Root) sdkLegacy(k kit.Ctx, profile *remote.Profile, inv remote.Inventory) {
	enabled := sdkConfigValue(inv, sdkLegacyEnableKey) == "true"
	kit.GroupHeader(k, i18n.Text("legacyTitle"), nil)
	items := []kit.ListItemSpec{{
		ID:       "sdkLegacyEnable",
		Name:     i18n.Text("legacyEnable"),
		Title:    i18n.Text("legacyEnable"),
		Subtitle: sdkJoin(" · ", profile.LegacyFiles...),
		Trailing: func() {
			kit.Switch(k, kit.SwitchSpec{
				ID:    "sdkLegacySwitch",
				On:    enabled,
				Label: i18n.Text("legacyEnable"),
				OnChange: func(on bool) {
					r.sdkPutConfig(sdkLegacyEnableKey, strconv.FormatBool(on))
				},
			})
		},
	}}
	if profile.LegacyStrategy {
		items = append(items, kit.ListItemSpec{
			ID:    "sdkLegacyStrategy",
			Name:  i18n.Text("legacyStrategy"),
			Title: i18n.Text("legacyStrategy"),
			Trailing: func() {
				kit.ButtonGroup(k, kit.ButtonGroupSpec{
					ID: "sdkStrategy",
					Segments: []kit.Segment{
						{Value: "specified", Label: i18n.Text("strategySpecified")},
						{Value: "installed", Label: i18n.Text("strategyInstalled")},
						{Value: "available", Label: i18n.Text("strategyAvailable")},
					},
					Selected: sdkConfigValue(inv, sdkLegacyStrategyKey),
					OnSelect: func(value string) {
						r.sdkPutConfig(sdkLegacyStrategyKey, value)
					},
				})
			},
		})
	}
	kit.ListGroup(k, items...)
}

// sdkAliases is the list of the aliases of the plugin, each with the button that
// removes it, and the row that names a new one. The aliases come from the computer
// when they are read; the list says so while they are not there yet.
func (r *Root) sdkAliases(k kit.Ctx, name string, limit *int) {
	res := r.service.Aliases(name)
	aliases, loaded := res.Value()
	kit.GroupHeader(k, i18n.Text("aliasTitle"), nil)
	var rows []kit.ListItemSpec
	shown := 0
	switch {
	case !loaded && res.Err() != nil:
		rows = append(rows, kit.ListItemSpec{
			ID:       "sdkAliasesFailed",
			Name:     i18n.Text("failed"),
			Title:    i18n.Text("failed"),
			Subtitle: sdkErrorText(res.Err()),
			Trailing: func() {
				kit.Button(k, kit.ButtonSpec{
					ID:      "sdkAliasesRetry",
					Kind:    kit.TonalButton,
					Label:   i18n.Text("retry"),
					OnClick: r.service.Refresh,
				})
			},
		})
	case !loaded:
		rows = append(rows, kit.ListItemSpec{ID: "sdkAliasesLoading", Name: i18n.Text("loading"), Title: i18n.Text("loading")})
	default:
		shown = min(*limit, len(aliases))
		for _, alias := range aliases[:shown] {
			rows = append(rows, r.sdkAliasRow(k, name, alias))
		}
	}
	rows = append(rows, kit.ListItemSpec{
		ID:      "sdkAliasAdd",
		Name:    i18n.Text("aliasSet"),
		Title:   i18n.Text("aliasSet"),
		Leading: func(foreground ui.Color) { kit.Icon(k, "add", 24, foreground, 0) },
		OnClick: func() { r.sdkOpenAlias(name, "") },
	})
	kit.ListGroup(k, rows...)
	if loaded && shown < len(aliases) {
		sdkShowMore(k, "sdkAliases", len(aliases)-shown, limit)
	}
}

// sdkAliasRow is the row of an alias: the alias, the version it names, and the button
// that removes the alias.
func (r *Root) sdkAliasRow(k kit.Ctx, name string, alias remote.VersionAlias) kit.ListItemSpec {
	return kit.ListItemSpec{
		ID:   alias.Alias,
		Name: alias.Alias,
		TitleWidget: func(foreground ui.Color) {
			kit.Label(k, alias.Alias, kit.Type(kit.BodyLarge), foreground, kit.Monospace(), kit.Breakable())
		},
		Subtitle: alias.Target,
		Trailing: func() {
			kit.Button(k, kit.ButtonSpec{
				ID:    "sdkAliasUnset",
				Kind:  kit.TextButton,
				Label: i18n.Text("aliasUnset"),
				OnClick: func() {
					r.sdkRun(remote.TaskRequest{Operation: remote.RequestAliasUnset, Sdk: name, Alias: alias.Alias})
				},
			})
		},
	}
}

// sdkTips is the list of the notes that the plugin gives.
func (r *Root) sdkTips(k kit.Ctx, tips []remote.Text) {
	kit.GroupHeader(k, i18n.Text("tips"), nil)
	rows := make([]kit.ListItemSpec, 0, len(tips))
	for i, tip := range tips {
		text := sdkText(tip)
		rows = append(rows, kit.ListItemSpec{ID: i, Name: text, Title: text})
	}
	kit.ListGroup(k, rows...)
}

// sdkDocs is the list of the documents of the plugin. A tap opens one in the browser.
func (r *Root) sdkDocs(k kit.Ctx, docs []remote.Link) {
	services := k.Services()
	kit.GroupHeader(k, i18n.Text("docs"), nil)
	rows := make([]kit.ListItemSpec, 0, len(docs))
	for i, doc := range docs {
		title := sdkText(doc.Title)
		address := sdkText(doc.URL)
		rows = append(rows, kit.ListItemSpec{
			ID:    i,
			Name:  title,
			Title: title,
			Trailing: func() {
				kit.Icon(k, "openInNew", 24, k.Colors().OnSurfaceVariant, 0)
			},
			OnClick: func() { r.sdkOpenLink(services, address) },
		})
	}
	kit.ListGroup(k, rows...)
}
