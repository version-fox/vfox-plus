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
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// settingsAppSection holds the settings of the application that the computer keeps:
// vfox's executable, the language of the application and the extra environment, and
// the language of this phone's screens, which the phone keeps itself. A field stays
// disabled until the computer has answered, so that a save cannot write the empty
// value that is shown before the answer.
func (r *Root) settingsAppSection(k kit.Ctx) {
	section := ui.Column(k.Context).Gap(8)
	exe := ui.Local(section, "exe", func() settingsDraft { return settingsDraft{} })
	env := ui.Local(section, "env", func() settingsDraft { return settingsDraft{} })
	screenLanguage := ui.Local(section, "screenLanguage", loadUILanguage)
	res := r.service.Settings()
	settings, loaded := res.Value()
	language := remote.LanguageSystem
	if loaded {
		exe.follow(settings.VfoxExecutable)
		env.follow(settingsFormatEnvironment(settings.ExtraEnvironment))
		if settings.Language != "" {
			language = settings.Language
		}
	}
	section.Children(func() {
		kit.GroupHeader(k, i18n.Text("app"), nil)
		settingsCard(k, func() {
			kit.TextField(k, kit.TextFieldSpec{
				ID: "exe-field", Label: i18n.Text("vfoxExecutable"), Text: &exe.text, Monospace: true, Disabled: !loaded,
				OnSubmit: func(text string) { r.settingsPutExe(strings.TrimSpace(text)) },
			})
			settingsActions(k, func() {
				kit.Button(k, kit.ButtonSpec{
					ID: "exe-save", Kind: kit.TonalButton, Label: i18n.Text("save"), Disabled: !loaded,
					OnClick: func() { r.settingsPutExe(strings.TrimSpace(exe.text)) },
				})
			})
		})
		settingsCard(k, func() {
			settingsHeader(k, "language", i18n.Text("applicationLanguage"), "")
			group := kit.ButtonGroupSpec{
				ID: "language-group", Expand: true, Selected: language,
				Segments: []kit.Segment{
					{Value: remote.LanguageSystem, Label: i18n.Text("langSystem")},
					{Value: remote.LanguageEnglish, Label: i18n.Text("setLangEnglish")},
					{Value: remote.LanguageSimplifiedChinese, Label: i18n.Text("setLangChinese")},
				},
			}
			// The choice is what the computer keeps: it changes when the answer of the computer does.
			if loaded {
				group.OnSelect = func(value string) { r.settingsPutLanguage(value) }
			}
			ui.Column(k.Context).FillWidth().Children(func() {
				kit.ButtonGroup(k, group)
			})
		})
		settingsCard(k, func() {
			settingsHeader(k, "globalScope", i18n.Text("setUILanguage"), i18n.Text("setUILanguageHelp"))
			ui.Column(k.Context).FillWidth().Children(func() {
				kit.ButtonGroup(k, kit.ButtonGroupSpec{
					ID: "screen-language-group", Expand: true, Selected: string(*screenLanguage),
					Segments: []kit.Segment{
						{Value: string(i18n.PreferenceSystem), Label: i18n.Text("langSystem")},
						{Value: string(i18n.PreferenceEnglish), Label: i18n.Text("setLangEnglish")},
						{Value: string(i18n.PreferenceChinese), Label: i18n.Text("setLangChinese")},
					},
					OnSelect: func(value string) {
						r.settingsSetScreenLanguage(k, screenLanguage, i18n.Preference(value))
					},
				})
			})
		})
		settingsCard(k, func() {
			kit.TextField(k, kit.TextFieldSpec{
				ID: "env-field", Label: i18n.Text("extraEnv"), Helper: i18n.Text("setEnvHelp"), Text: &env.text,
				Monospace: true, Lines: 3, MaxLines: 8, Disabled: !loaded,
			})
			settingsActions(k, func() {
				kit.Button(k, kit.ButtonSpec{
					ID: "env-save", Kind: kit.TonalButton, Label: i18n.Text("save"), Disabled: !loaded,
					OnClick: func() { r.settingsPutEnv(env.text) },
				})
			})
		})
	})
}

// settingsCard is a surface of the settings that holds a field, or a choice, and its
// buttons.
func settingsCard(k kit.Ctx, build func()) ui.Element {
	return ui.Column(k.Context).Background(k.Colors().SurfaceContainer).Radius(kit.CornerLarge).
		Padding(16).Gap(12).FillWidth().Children(build)
}

// settingsHeader is the title of a card that holds a choice, with its icon and, under
// it, what the choice changes.
func settingsHeader(k kit.Ctx, icon, title, help string) {
	cs := k.Colors()
	ui.Row(k.Context).Gap(12).AlignItems(ui.Center).Children(func() {
		kit.Icon(k, icon, 24, cs.OnSurfaceVariant, 0)
		kit.Label(k, title, kit.Type(kit.TitleMedium), cs.OnSurface, kit.Breakable()).Grow(1).Shrink(1)
	})
	if help != "" {
		kit.Label(k, help, kit.Type(kit.BodyMedium), cs.OnSurfaceVariant, kit.Breakable())
	}
}

// settingsActions is the row of buttons at the end of a card.
func settingsActions(k kit.Ctx, build func()) {
	ui.Row(k.Context).Justify(ui.End).Wrap().Gap(8).Children(build)
}

// settingsLeading is the icon at the start of a row of the settings.
func settingsLeading(k kit.Ctx, name string) func(foreground ui.Color) {
	return func(foreground ui.Color) {
		kit.Icon(k, name, 24, foreground, 0)
	}
}

// settingsSetScreenLanguage changes the language of the screens at once, and keeps
// the choice on the phone, so that the next start has it too.
func (r *Root) settingsSetScreenLanguage(k kit.Ctx, preference *i18n.Preference, value i18n.Preference) {
	*preference = value
	applyUILanguage(value)
	if err := saveUILanguage(value); err != nil {
		k.Env.ShowSnackbar(kit.Snack{Text: i18n.Text("setUIPrefFailed")})
	}
}

// settingsPutExe saves vfox's executable. An empty value is the default one.
func (r *Root) settingsPutExe(value string) {
	r.Do(func(ctx context.Context) error {
		_, err := r.service.PutSettings(ctx, remote.SettingsPatch{VfoxExecutable: &value})
		return err
	}, "saved")
}

// settingsPutLanguage saves the language of the application: system, english or
// simplifiedChinese.
func (r *Root) settingsPutLanguage(language string) {
	r.Do(func(ctx context.Context) error {
		_, err := r.service.PutSettings(ctx, remote.SettingsPatch{Language: &language})
		return err
	}, "saved")
}

// settingsPutEnv saves the extra environment that the text holds. It replaces all
// the variables the computer has; the field says so.
func (r *Root) settingsPutEnv(text string) {
	env := settingsParseEnvironment(text)
	r.Do(func(ctx context.Context) error {
		_, err := r.service.PutSettings(ctx, remote.SettingsPatch{ExtraEnvironment: &env})
		return err
	}, "saved")
}
