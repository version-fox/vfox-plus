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

// The other ways to add a version of an SDK, which the web's "More ways to add" menu
// offers: installing by a name that the person types, detecting an installation and
// linking it, and linking an SDK that is on the computer already.

import (
	"strings"
	"sync/atomic"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// availMoreButton is the button of the other ways to add a version: its menu installs
// by name, detects and links, or links an SDK that is on the computer.
func (r *Root) availMoreButton(k kit.Ctx, name string, profile *remote.Profile, landed *atomic.Bool) {
	env := k.Env
	menu := []kit.MenuItem{{
		Label: i18n.Text("installByName"),
		Icon:  "terminal",
		OnClick: func() {
			r.availOpenInstallName(env, name, profile, landed, "")
		},
	}}
	// A plugin without the Detect hook cannot find what is installed already.
	if profile.CanDetect() {
		menu = append(menu, kit.MenuItem{
			Label: i18n.Text("linkDetect"),
			Icon:  "link",
			OnClick: func() {
				r.availStart(landed, remote.TaskRequest{Operation: remote.RequestLinkDetect, Sdk: name, Yes: true})
			},
		})
	}
	menu = append(menu, kit.MenuItem{
		Label: i18n.Text("linkManual"),
		Icon:  "folder",
		OnClick: func() {
			r.availOpenLink(env, name, landed)
		},
	})
	kit.IconButton(k, kit.IconButtonSpec{ID: "moreWays", Icon: "add", Tooltip: i18n.Text("moreWays"), Menu: menu})
}

// availNameForm is what the install-by-name dialog keeps while it is open: the name
// that is typed, the message of the check that refused it, and whether to use the
// version after it is installed.
type availNameForm struct {
	text string
	err  string
	use  bool
}

// availOpenInstallName opens the dialog that installs a version of the plugin name by
// the name that the person types. The plugin's profile suggests names, and its guide
// explains them. The name is checked as the web checks it, before it is sent.
func (r *Root) availOpenInstallName(env *kit.Env, name string, profile *remote.Profile, landed *atomic.Bool, initial string) {
	form := &availNameForm{text: initial}
	env.ShowDialog("availInstallName", func(c kit.Ctx, close func()) kit.DialogSpec {
		submit := func() {
			version := availTrim(form.text)
			switch {
			case version == "":
				form.err = i18n.Text("nameRequired")
			case !availValidSpec(version):
				form.err = i18n.Text("nameInvalid")
			default:
				close()
				r.availStart(landed, remote.TaskRequest{Operation: remote.RequestInstall, Sdk: name, Version: version, Use: form.use})
			}
		}
		guide := ""
		if profile != nil {
			guide = profile.Guide.In(availChinese())
		}
		return kit.DialogSpec{
			Icon:  "terminal",
			Title: strings.ReplaceAll(i18n.Text("installNameTitle"), "{name}", name),
			Content: func() {
				ui.Column(c.Context).Gap(12).Children(func() {
					kit.TextField(c, kit.TextFieldSpec{
						ID:        "name",
						Label:     i18n.Text("versionName"),
						Text:      &form.text,
						Monospace: true,
						Helper:    guide,
						Error:     form.err,
						AutoFocus: true,
						OnChange:  func(string) { form.err = "" },
						OnSubmit:  func(string) { submit() },
					})
					if profile != nil && len(profile.Shorthands) > 0 {
						availShorthands(c, profile.Shorthands, form)
					}
					availSwitch(c, "use", i18n.Text("installUseAfter"), form.use, func(on bool) { form.use = on })
				})
			},
			Actions: []func(){
				func() {
					kit.Button(c, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: i18n.Text("cancel"), OnClick: close})
				},
				func() {
					kit.Button(c, kit.ButtonSpec{ID: "install", Kind: kit.FilledButton, Label: i18n.Text("install"), Icon: "install", OnClick: submit})
				},
			},
		}
	})
}

// availShorthands builds the names that the plugin suggests: a chip for each, which
// puts it in the field, and the meaning of the name that the field holds, when the
// plugin says what it means.
func availShorthands(c kit.Ctx, shorthands []remote.Shorthand, form *availNameForm) {
	text := availTrim(form.text)
	ui.Column(c.Context).Gap(8).Children(func() {
		kit.Label(c, i18n.Text("namesToTry"), kit.Type(kit.LabelLarge), c.Colors().OnSurfaceVariant)
		ui.Row(c.Context).Wrap().Gap(8).GapY(8).Children(func() {
			for _, shorthand := range shorthands {
				kit.FilterChip(c, kit.ChipSpec{
					ID:       shorthand.Name,
					Label:    shorthand.Name,
					Selected: text == shorthand.Name,
					OnClick: func() {
						form.text = shorthand.Name
						form.err = ""
					},
				})
			}
		})
		if found, ok := availShorthand(shorthands, text); ok {
			kit.Label(c, found.Meaning.In(availChinese()), kit.Type(kit.BodyLarge), c.Colors().OnSurface, kit.Lines(3))
		}
	})
}

// availSwitch is a row of a dialog: the label of a switch, and the switch that it names.
func availSwitch(c kit.Ctx, id, label string, on bool, change func(bool)) {
	ui.Row(c.Context).AlignItems(ui.Center).Gap(16).Children(func() {
		kit.Label(c, label, kit.Type(kit.BodyLarge), c.Colors().OnSurface, kit.Lines(2)).Grow(1).Shrink(1)
		kit.Switch(c, kit.SwitchSpec{ID: id, On: on, OnChange: change, Label: label})
	})
}

// availLinkForm is what the link dialog keeps while it is open.
type availLinkForm struct {
	version string
	folder  string
	force   bool
}

// availOpenLink opens the dialog that links an SDK which is on the computer already: the
// name of its version, the folder of the SDK, which is on the computer and not on this
// phone, and whether a version that is linked already is pointed at the folder again.
func (r *Root) availOpenLink(env *kit.Env, name string, landed *atomic.Bool) {
	form := &availLinkForm{}
	env.ShowDialog("availLink", func(c kit.Ctx, close func()) kit.DialogSpec {
		ready := func() bool {
			return availTrim(form.version) != "" && availTrim(form.folder) != ""
		}
		submit := func() {
			if !ready() {
				return
			}
			close()
			r.availStart(landed, remote.TaskRequest{
				Operation: remote.RequestLink,
				Sdk:       name,
				Version:   availTrim(form.version),
				Path:      availTrim(form.folder),
				Force:     form.force,
			})
		}
		return kit.DialogSpec{
			Icon:  "link",
			Title: i18n.Text("linkPathTitle"),
			Content: func() {
				ui.Column(c.Context).Gap(12).Children(func() {
					kit.TextField(c, kit.TextFieldSpec{
						ID:        "linkVersion",
						Label:     i18n.Text("linkVersionField"),
						Text:      &form.version,
						Monospace: true,
						AutoFocus: true,
						OnSubmit:  func(string) { submit() },
					})
					kit.TextField(c, kit.TextFieldSpec{
						ID:        "linkFolder",
						Label:     i18n.Text("linkPathField"),
						Text:      &form.folder,
						Monospace: true,
						Helper:    i18n.Text("linkPathHelp"),
						OnSubmit:  func(string) { submit() },
					})
					kit.Label(c, i18n.Text("availPathOnComputer"), kit.Type(kit.BodyMedium), c.Colors().OnSurfaceVariant, kit.Lines(2))
					availSwitch(c, "linkForce", i18n.Text("linkForce"), form.force, func(on bool) { form.force = on })
				})
			},
			Actions: []func(){
				func() {
					kit.Button(c, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: i18n.Text("cancel"), OnClick: close})
				},
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID: "link", Kind: kit.FilledButton, Label: i18n.Text("linkConfirm"), Icon: "link",
						Disabled: !ready(), OnClick: submit,
					})
				},
			},
		}
	})
}
