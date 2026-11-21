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
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// phoneCard is the card of this phone in the settings: the computer it is paired with,
// the button that pairs a new link, and the button that disconnects. The settings page
// calls it. Pairing a new link keeps the connection until the new pairing succeeds.
func (r *Root) phoneCard(k kit.Ctx) {
	cs := k.Colors()
	address := r.service.Status().Server.Base
	if address == "" {
		address = i18n.Text("empty")
	}
	kit.Card(k, kit.CardSpec{
		ID:      "phoneCard",
		Color:   cs.SurfaceContainerHigh,
		Radius:  16,
		Content: cs.OnSurface,
		Name:    i18n.Text("phone"),
		Children: func() {
			ui.Column(k.Context).Padding(16).Gap(12).Children(func() {
				kit.Label(k, i18n.Text("phone"), kit.Type(kit.TitleMedium), cs.OnSurface)
				kit.Label(k, i18n.Text("phoneLead"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
				kit.Label(k, address, kit.Type(kit.TitleSmall), cs.OnSurface, kit.Monospace(), kit.Breakable(), kit.Selectable())
				ui.Column(k.Context).Gap(8).Children(func() {
					kit.Button(k, kit.ButtonSpec{
						ID:    "pairNew",
						Kind:  kit.TonalButton,
						Label: i18n.Text("phonePair"),
						Icon:  "link",
						OnClick: func() {
							r.app.Push(app.Page{Kind: app.PagePair})
						},
					})
					kit.Button(k, kit.ButtonSpec{
						ID:      "disconnect",
						Kind:    kit.OutlinedButton,
						Label:   i18n.Text("phoneOut"),
						Icon:    "quit",
						OnClick: r.phoneDisconnectDialog,
					})
				})
			})
		},
	})
}

// phoneDisconnectDialog asks the user to confirm the disconnection. Once confirmed, the
// phone forgets the computer and its session, and the pairing page shows.
func (r *Root) phoneDisconnectDialog() {
	r.env.ShowDialog("phoneDisconnect", func(c kit.Ctx, close func()) kit.DialogSpec {
		return kit.DialogSpec{
			Icon:      "quit",
			IconError: true,
			Title:     i18n.Text("phoneOutTitle"),
			Content: func() {
				kit.Label(c, i18n.Text("phoneOutBody"), kit.Type(kit.BodyMedium), c.Colors().OnSurfaceVariant)
			},
			Actions: []func(){
				func() {
					kit.Button(c, kit.ButtonSpec{ID: "cancelDisconnect", Kind: kit.TextButton, Label: i18n.Text("cancel"), OnClick: close})
				},
				func() {
					kit.Button(c, kit.ButtonSpec{
						ID:     "confirmDisconnect",
						Kind:   kit.FilledButton,
						Danger: true,
						Label:  i18n.Text("phoneOut"),
						OnClick: func() {
							close()
							r.Do(func(context.Context) error { return r.service.Disconnect() }, "")
						},
					})
				},
			},
		}
	})
}
