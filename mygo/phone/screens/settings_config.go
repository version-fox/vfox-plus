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

import (
	"context"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// settingsConfigSection is vfox's configuration: each key with its value, and a menu
// of the key that changes its value or resets it. The keys are the ones the computer
// has; a key that is not set shows an empty value.
func (r *Root) settingsConfigSection(k kit.Ctx) {
	cs := k.Colors()
	section := ui.Column(k.Context).Gap(2)
	res := r.service.Inventory()
	inventory, loaded := res.Value()
	section.Children(func() {
		kit.GroupHeader(k, i18n.Text("config"), nil)
		config := inventory.Config
		switch {
		case !loaded && res.Phase() == app.Failed:
			kit.Label(k, noticeText(app.KindOf(res.Err())), kit.Type(kit.BodyMedium), cs.Error, kit.Breakable())
		case !loaded || (config.Loading && len(config.Value) == 0 && config.Error == ""):
			kit.Label(k, i18n.Text("loading"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
		case config.Error != "" && len(config.Value) == 0:
			kit.Label(k, config.Error, kit.Type(kit.BodyMedium), cs.Error, kit.Breakable())
		default:
			items := make([]kit.ListItemSpec, 0, len(config.Value))
			for _, entry := range config.Value {
				items = append(items, r.settingsConfigItem(k, entry))
			}
			kit.ListGroup(k, items...)
		}
	})
}

// settingsConfigItem is one key of the configuration.
func (r *Root) settingsConfigItem(k kit.Ctx, entry remote.ConfigEntry) kit.ListItemSpec {
	cs := k.Colors()
	key, value := entry.Key, entry.Value
	subtitle := value
	if value == "" {
		subtitle = i18n.Text("empty")
	}
	return kit.ListItemSpec{
		ID: key, Name: key,
		TitleWidget: func(ui.Color) {
			kit.Label(k, key, kit.Type(kit.BodyLarge), cs.OnSurface, kit.Monospace(), kit.Breakable())
		},
		Subtitle: subtitle, SubtitleLines: 2,
		Trailing: func() {
			kit.IconButton(k, kit.IconButtonSpec{
				ID: "more", Icon: "more", Tooltip: i18n.Text("actions"),
				Menu: []kit.MenuItem{
					{Label: i18n.Text("edit"), Icon: "vfoxConfig", OnClick: func() {
						r.settingsEditConfig(k, key, value)
					}},
					{Label: i18n.Text("reset"), Icon: "legacyFiles", Danger: true, OnClick: func() {
						r.settingsResetConfig(key)
					}},
				},
			})
		},
	}
}

// settingsEditConfig opens the dialog that changes the value of a key. The field starts
// with the value the key has; an empty value is a value that is saved as it is.
func (r *Root) settingsEditConfig(k kit.Ctx, key, value string) {
	draft := &settingsDraft{text: value, last: value, set: true}
	k.Env.ShowDialog("set-config-edit", func(c kit.Ctx, closeDialog func()) kit.DialogSpec {
		return kit.DialogSpec{
			Icon: "vfoxConfig", Title: key,
			Content: func() {
				kit.Label(c, i18n.Text("setConfigEdit"), kit.Type(kit.BodyMedium), c.Colors().OnSurfaceVariant, kit.Breakable())
				kit.TextField(c, kit.TextFieldSpec{
					ID: "config-value", Label: key, Text: &draft.text, Monospace: true, MaxLines: 4,
					OnSubmit: func(string) {
						closeDialog()
						r.settingsPutConfig(key, draft.text)
					},
				})
			},
			Actions: []func(){
				func() {
					kit.Button(c, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: i18n.Text("cancel"), OnClick: closeDialog})
				},
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID: "save", Kind: kit.FilledButton, Label: i18n.Text("save"),
						OnClick: func() {
							closeDialog()
							r.settingsPutConfig(key, draft.text)
						},
					})
				},
			},
		}
	})
}

// settingsPutConfig sets a key of the configuration.
func (r *Root) settingsPutConfig(key, value string) {
	r.Do(func(ctx context.Context) error {
		return r.service.PutConfig(ctx, key, value)
	}, "saved")
}

// settingsResetConfig takes a key back to its default.
func (r *Root) settingsResetConfig(key string) {
	r.Do(func(ctx context.Context) error {
		return r.service.DeleteConfig(ctx, key)
	}, "")
}
