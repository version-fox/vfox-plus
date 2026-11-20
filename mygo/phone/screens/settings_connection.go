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
	"strconv"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// settingsConnectionSection is how other devices reach the computer: whether they
// may connect, on which port, the pairing code for a phone, the lightweight mode,
// and quitting the computer's app. It is the web's card of the same name.
//
// Turning the access off disconnects every device, this phone included, and the
// phone must pair again to come back. So both ways of the switch ask first.
func (r *Root) settingsConnectionSection(k kit.Ctx) {
	section := ui.Column(k.Context).Gap(2)
	port := ui.Local(section, "port", func() settingsDraft { return settingsDraft{} })
	res := r.service.Web()
	web, loaded := res.Value()
	if loaded {
		port.follow(strconv.Itoa(web.Port))
	}
	section.Children(func() {
		kit.GroupHeader(k, i18n.Text("webH"), nil)
		switch {
		case !loaded && res.Phase() == app.Failed:
			kit.Label(k, noticeText(app.KindOf(res.Err())), kit.Type(kit.BodyMedium), k.Colors().Error, kit.Breakable())
		case !loaded:
			kit.Label(k, i18n.Text("loading"), kit.Type(kit.BodyMedium), k.Colors().OnSurfaceVariant)
		default:
			items := []kit.ListItemSpec{r.settingsRemoteItem(k, web)}
			if web.Remote {
				items = append(items, r.settingsPortItem(k, web, port))
				switch {
				case web.Error != "":
					items = append(items, settingsErrorItem(k, web.Error))
				case web.Listening:
					items = append(items, r.settingsPairItem(k))
				}
			}
			items = append(items, r.settingsLightItem(k, web), r.settingsQuitItem(k))
			kit.ListGroup(k, items...)
		}
	})
}

// settingsRemoteItem is the switch that lets other devices connect. The whole row
// flips it, as the switch does.
func (r *Root) settingsRemoteItem(k kit.Ctx, web remote.WebState) kit.ListItemSpec {
	return kit.ListItemSpec{
		ID: "remote", Leading: settingsLeading(k, "otherDevices"), Title: i18n.Text("webRemote"),
		OnClick: func() { r.settingsAskRemote(k, !web.Remote) },
		Trailing: func() {
			kit.Switch(k, kit.SwitchSpec{
				ID: "remote-switch", On: web.Remote, Label: i18n.Text("webRemote"),
				OnChange: func(on bool) { r.settingsAskRemote(k, on) },
			})
		},
	}
}

// settingsAskRemote asks before other devices are let in, or kept out. Keeping them
// out disconnects this phone too, and it must pair again to come back; the dialog says
// so, and the switch changes only when the computer answers.
func (r *Root) settingsAskRemote(k kit.Ctx, on bool) {
	if on {
		settingsConfirm(k, "set-remote-on", "otherDevices",
			i18n.Text("webRemoteTitle"), i18n.Text("webRemoteBody"), i18n.Text("webRemoteOk"), false,
			func() { r.settingsPutWeb(remote.WebPatch{Remote: settingsBool(true)}) })
		return
	}
	settingsConfirm(k, "set-remote-off", "otherDevices",
		i18n.Text("setRemoteOffTitle"), i18n.Text("setRemoteOffBody"), i18n.Text("setRemoteOffOk"), true,
		func() { r.settingsPutWeb(remote.WebPatch{Remote: settingsBool(false)}) })
}

// settingsPortItem is the port of the connection. The field is checked as the user
// types, and the port is saved when the user presses Enter or leaves the field.
func (r *Root) settingsPortItem(k kit.Ctx, web remote.WebState, port *settingsDraft) kit.ListItemSpec {
	errorText := ""
	if _, ok := settingsParsePort(port.text); !ok {
		errorText = i18n.Text("setPortRange")
	}
	commit := func(text string) {
		value, ok := settingsParsePort(text)
		if ok && value != web.Port {
			r.settingsPutWeb(remote.WebPatch{Port: &value})
		}
	}
	return kit.ListItemSpec{
		ID: "port", Leading: settingsLeading(k, "options"), Title: i18n.Text("webPort"),
		Subtitle: i18n.Text("webPortHelp"), Wrap: true,
		Trailing: func() {
			kit.TextField(k, kit.TextFieldSpec{
				ID: "port-field", Label: i18n.Text("setPortHint"), Text: &port.text, Error: errorText,
				OnSubmit: commit, OnBlur: commit,
				// A number, as the web interface's field of the port is: the keyboard of digits.
				Keyboard: ui.KeyboardNumber,
			})
		},
	}
}

// settingsErrorItem says why other devices cannot connect, in the colour of errors.
func settingsErrorItem(k kit.Ctx, message string) kit.ListItemSpec {
	text := i18n.Text("webError") + " " + message
	return kit.ListItemSpec{
		ID: "error", Name: text, Leading: settingsLeading(k, "error"),
		TitleWidget: func(ui.Color) {
			kit.Label(k, text, kit.Type(kit.BodyLarge), k.Colors().Error, kit.Breakable())
		},
	}
}

// settingsPairItem opens the pairing code, which another phone reads to pair.
func (r *Root) settingsPairItem(k kit.Ctx) kit.ListItemSpec {
	return kit.ListItemSpec{
		ID: "pair", Leading: settingsLeading(k, "registry"), Title: i18n.Text("webPairBtn"),
		Trailing: func() {
			kit.Button(k, kit.ButtonSpec{
				ID: "pair-button", Kind: kit.TonalButton, Label: i18n.Text("pairShowCode"), Icon: "qrCode",
				OnClick: func() { r.settingsOpenPairing(k) },
			})
		},
	}
}

// settingsLightItem is the lightweight mode, which keeps the computer's app running
// after its window closes.
func (r *Root) settingsLightItem(k kit.Ctx, web remote.WebState) kit.ListItemSpec {
	return kit.ListItemSpec{
		ID: "light", Leading: settingsLeading(k, "lightweight"), Title: i18n.Text("webLight"),
		Subtitle: i18n.Text("webLightHelp"),
		Trailing: func() {
			kit.Switch(k, kit.SwitchSpec{
				ID: "light-switch", On: web.Lightweight, Label: i18n.Text("webLight"),
				OnChange: func(on bool) { r.settingsPutWeb(remote.WebPatch{Lightweight: settingsBool(on)}) },
			})
		},
	}
}

// settingsQuitItem quits the computer's app, after a confirmation.
func (r *Root) settingsQuitItem(k kit.Ctx) kit.ListItemSpec {
	return kit.ListItemSpec{
		ID: "quit", Leading: settingsLeading(k, "quit"), Title: i18n.Text("webQuit"),
		Trailing: func() {
			kit.Button(k, kit.ButtonSpec{
				ID: "quit-button", Kind: kit.FilledButton, Danger: true, Label: i18n.Text("webQuitBtn"),
				OnClick: func() {
					settingsConfirm(k, "set-quit", "quit",
						i18n.Text("webQuitTitle"), i18n.Text("webQuitBody"), i18n.Text("webQuitBtn"), true,
						func() { r.settingsQuit() })
				},
			})
		},
	}
}

// settingsConfirm asks the user to confirm an action, in a dialog with a cancel button
// and the action's own button. The action runs only for the second; a dangerous one is
// drawn in the colour of errors.
func settingsConfirm(k kit.Ctx, id, icon, title, body, okLabel string, danger bool, ok func()) {
	k.Env.ShowDialog(id, func(c kit.Ctx, closeDialog func()) kit.DialogSpec {
		return kit.DialogSpec{
			Icon: icon, IconError: danger, Title: title,
			Content: func() {
				kit.Label(c, body, kit.Type(kit.BodyMedium), c.Colors().OnSurfaceVariant, kit.Breakable(), kit.Centered())
			},
			Actions: []func(){
				func() {
					kit.Button(c, kit.ButtonSpec{ID: "cancel", Kind: kit.TextButton, Label: i18n.Text("cancel"), OnClick: closeDialog})
				},
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID: "confirm", Kind: kit.FilledButton, Danger: danger, Label: okLabel,
						OnClick: func() {
							closeDialog()
							ok()
						},
					})
				},
			},
		}
	})
}

// settingsPutWeb changes how other devices connect. The answer of the computer is the
// state of the connection from then on, so the switches show it.
func (r *Root) settingsPutWeb(patch remote.WebPatch) {
	r.Do(func(ctx context.Context) error {
		_, err := r.service.PutWeb(ctx, patch)
		return err
	}, "")
}

// settingsQuit asks the computer to quit its app.
func (r *Root) settingsQuit() {
	r.Do(func(ctx context.Context) error {
		return r.service.Quit(ctx)
	}, "webQuitting")
}

// settingsBool is a pointer to v, for the fields of a patch that are set only when
// they are given.
func settingsBool(v bool) *bool {
	return &v
}
