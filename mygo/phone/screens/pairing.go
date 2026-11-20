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
	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// pairingScreen is the page of a phone that is not paired, the first screen of the
// app: the lead, the field of the pairing link, the three ways to get a link, and the
// button that connects. A computer that needs attention, and the last one, are
// explained above the field. The pairing is in pairForm.
func (r *Root) pairingScreen(k kit.Ctx) {
	ui.Column(k.Context).Fill().Children(func() {
		kit.PageHeader(k, i18n.Text("phone"), "", nil)
		r.pairForm(k)
	})
}

// pairForm is the pairing: what the computer needs from the phone, the field of the
// link, the ways to get a link, the camera's preview while it is on, the button that
// connects, and how to pair. The page of the phone and the page of a new link both
// show it, each with its own reads.
func (r *Root) pairForm(k kit.Ctx) {
	holder := ui.Column(k.Context).Grow(1).Shrink(1)
	reading := pairReadingOf(holder)
	clipboard := k.Services()
	holder.Children(func() {
		ui.Scroll(k.Context).Grow(1).Shrink(1).Children(func() {
			ui.Column(k.Context).Padding(0, 16, 16, 16).Gap(16).Children(func() {
				r.pairStatusCard(k, reading, clipboard)
				r.pairLastCard(k, reading, clipboard)
				kit.Label(k, i18n.Text("pairLead"), kit.Type(kit.BodyLarge), k.Colors().OnSurfaceVariant)
				r.pairLinkField(k, reading)
				r.pairEntries(k, reading, clipboard)
				r.pairCameraPage(k, reading)
				ui.Column(k.Context).Gap(8).Children(func() {
					r.pairLinkButton(k)
					kit.Button(k, kit.ButtonSpec{
						ID:      "howToPair",
						Kind:    kit.TextButton,
						Label:   i18n.Text("pairHow"),
						Icon:    "info",
						OnClick: r.pairHowDialog,
					})
				})
			})
		})
	})
}

// pairLinkField is the field of the pairing link. It says why the text is not a link,
// or why a photo gave none, under itself.
func (r *Root) pairLinkField(k kit.Ctx, reading *pairReading) {
	_, note := reading.current()
	message := pairLinkError(r.linkText)
	if message == "" && note != "" {
		message = i18n.Text(note)
	}
	kit.TextField(k, kit.TextFieldSpec{
		ID:          "link",
		Label:       i18n.Text("pairLinkLabel"),
		Hint:        i18n.Text("pairLinkHint"),
		LeadingIcon: "link",
		Text:        &r.linkText,
		Error:       message,
		Lines:       1,
		Keyboard:    ui.KeyboardURL,
	})
}

// pairEntries are the three ways to get a pairing link, side by side: paste it from the
// clipboard, read the code with the camera, or read a picture of the code. A read that
// is running disables the two that read.
func (r *Root) pairEntries(k kit.Ctx, reading *pairReading, clipboard ui.Services) {
	busy, _ := reading.current()
	ui.Row(k.Context).Gap(8).Children(func() {
		kit.Button(k, kit.ButtonSpec{
			ID:    "paste",
			Kind:  kit.TonalButton,
			Label: i18n.Text("pairPaste"),
			Icon:  "copy",
			OnClick: func() {
				reading.setNote(r.pairPasteClipboard(clipboard))
			},
		}).Grow(1)
		kit.Button(k, kit.ButtonSpec{
			ID:       "photo",
			Kind:     kit.TonalButton,
			Label:    i18n.Text("pairPhoto"),
			Icon:     "qrCode",
			Disabled: busy,
			OnClick:  func() { r.pairStartCamera(reading) },
		}).Grow(1)
		kit.Button(k, kit.ButtonSpec{
			ID:       "picture",
			Kind:     kit.TonalButton,
			Label:    i18n.Text("pairPicture"),
			Icon:     "folder",
			Disabled: busy,
			OnClick:  func() { r.pairPickPicture(reading) },
		}).Grow(1)
	})
}

// pairStatusCard says what the computer needs from the phone, when it needs something:
// the phone must pair again, because the computer does not accept it, or its
// certificate is not the one that was paired. Nothing is trusted automatically: the
// pairing below is the way to trust a computer again.
func (r *Root) pairStatusCard(k kit.Ctx, reading *pairReading, clipboard ui.Services) {
	cs := k.Colors()
	// Pair again takes the link on the clipboard, which the computer shows on its
	// pairing page: the link is then confirmed like any other.
	pairAgain := func() {
		if key := r.pairPasteClipboard(clipboard); key != "" {
			reading.setNote(key)
		}
	}
	switch r.service.Status().Session {
	case app.NeedsPairing:
		pairCard(k, "pairNeeds", "warning", i18n.Text("pairNeedsTitle"), noticeText(app.KindUnpaired),
			cs.SurfaceContainerHigh, cs.OnSurface, cs.OnSurfaceVariant, pairAgain)
	case app.CertificateChanged:
		pairCard(k, "pairCert", "certificate", i18n.Text("pairCertTitle"), noticeText(app.KindCertificateChanged),
			cs.ErrorContainer, cs.OnErrorContainer, cs.OnErrorContainer, pairAgain)
	}
}

// pairLastCard is the last computer, when the phone has one saved and must pair again:
// its address, and the button that connects with the link on the clipboard. The link
// is the one-time code of the computer, which is never saved, so connecting needs a
// new link; the confirmation of its fingerprint still comes first.
func (r *Root) pairLastCard(k kit.Ctx, reading *pairReading, clipboard ui.Services) {
	server := r.service.Status().Server
	if server.Base == "" {
		return
	}
	switch r.service.Status().Session {
	case app.NeedsPairing, app.CertificateChanged:
	default:
		return
	}
	cs := k.Colors()
	kit.Card(k, kit.CardSpec{
		ID:      "lastComputer",
		Color:   cs.SurfaceContainerHigh,
		Radius:  16,
		Content: cs.OnSurface,
		Name:    i18n.Text("pairLastComputer"),
		Children: func() {
			ui.Column(k.Context).Padding(16).Gap(8).Children(func() {
				kit.Label(k, i18n.Text("pairLastComputer"), kit.Type(kit.TitleMedium), cs.OnSurface)
				kit.Label(k, server.Base, kit.Type(kit.BodyMedium), cs.OnSurfaceVariant, kit.Monospace(), kit.Breakable())
				kit.Button(k, kit.ButtonSpec{
					ID:    "reconnect",
					Kind:  kit.TonalButton,
					Label: i18n.Text("connect"),
					Icon:  "link",
					OnClick: func() {
						if key := r.pairPasteClipboard(clipboard); key != "" {
							reading.setNote(key)
							return
						}
						r.pairConnectText(r.linkText)
					},
				})
			})
		},
	})
}

// pairLinkButton is the button that connects with the link in the field. It stays
// disabled until the field holds a pairing link.
func (r *Root) pairLinkButton(k kit.Ctx) {
	kit.Button(k, kit.ButtonSpec{
		ID:       "connect",
		Kind:     kit.FilledButton,
		Label:    i18n.Text("connect"),
		Icon:     "link",
		Disabled: !pairFieldHoldsLink(r.linkText),
		OnClick:  func() { r.pairConnectText(r.linkText) },
	})
}

// pairHowDialog explains how to get a pairing link from the computer, and what to
// compare before trusting it.
func (r *Root) pairHowDialog() {
	r.env.ShowDialog("pairHow", func(c kit.Ctx, close func()) kit.DialogSpec {
		return kit.DialogSpec{
			Icon:  "info",
			Title: i18n.Text("pairHowTitle"),
			Content: func() {
				kit.Label(c, i18n.Text("pairHowBody"), kit.Type(kit.BodyMedium), c.Colors().OnSurfaceVariant)
			},
			Actions: []func(){pairCloseAction(c, close)},
		}
	})
}

// pairCard is a card that says one message, with an icon and a title, in the colours it
// is given, and the button that acts on it: pair again, when action is not nil.
func pairCard(k kit.Ctx, id, icon, title, body string, background, foreground, text ui.Color, action func()) {
	kit.Card(k, kit.CardSpec{
		ID:      id,
		Color:   background,
		Radius:  16,
		Content: foreground,
		Name:    title,
		Children: func() {
			ui.Column(k.Context).Padding(16).Gap(8).Children(func() {
				ui.Row(k.Context).Gap(12).Children(func() {
					kit.Icon(k, icon, 24, foreground, 0)
					kit.Label(k, title, kit.Type(kit.TitleMedium), foreground)
				})
				kit.Label(k, body, kit.Type(kit.BodyMedium), text)
				if action != nil {
					kit.Button(k, kit.ButtonSpec{
						ID:      "pairAgain",
						Kind:    kit.TonalButton,
						Label:   i18n.Text("pairAgain"),
						Icon:    "copy",
						OnClick: action,
					})
				}
			})
		},
	})
}
