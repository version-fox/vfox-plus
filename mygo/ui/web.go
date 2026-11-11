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
	"context"
	"net"
	"strconv"
	"strings"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// Web is the web interface of the application, which the settings of the window
// switch on and off: the browsers of this computer, and the phones and the other
// computers of the network if the user allows them, connect to it. The Host of
// package pair is the one.
type Web interface {
	state.Listenable
	// Start serves the web interface as the settings say, and follows them.
	Start()
	// Close stops serving it, and ends the connections.
	Close()
	// Status describes how it is served.
	Status() pair.Status
	// LocalLink makes a pairing link for a browser on this computer.
	LocalLink() (string, error)
	// RemoteLink makes a pairing link for a device on the network, with host as
	// the address of this computer, or the likeliest one when it is empty.
	RemoteLink(host string) (string, error)
	// Publish starts the public tunnel, and returns the address it serves.
	Publish() (string, error)
	// Unpublish ends the public tunnel.
	Unpublish()
}

// pairChoice is an address that a pairing link may name: what the dialog offers
// and the pairing row lists, with the value that goes into the link.
type pairChoice struct {
	value string
	label string
}

// pairingState is the link that the dialog shows, and the code of it.
type pairingState struct {
	link   string
	host   string
	matrix [][]bool
	err    error
}

// webSection holds what the web interface asks of the user: whether it is served, whether
// the application stays when the window is closed, and whether other devices connect.
func (a *App) webSection(k kit.Ctx) {
	current := a.model.Settings.Settings()
	status := a.web.Status()
	items := []kit.ListItemSpec{a.webInterfaceItem(k, current, status)}
	if status.Running {
		items = append(items, a.webOpenItem(k))
	}
	items = append(items, a.lightweightItem(k, current), a.remoteItem(k, current, status))
	if current.WebInterface && current.WebRemote {
		items = append(items, a.portItem(k, current), a.publicAddressItem(k, current), a.publishItem(k, status))
		if status.Remote {
			items = append(items, a.pairItem(k, status), a.certificateItem(k, current, status))
		}
	}
	if current.Lightweight && status.Running {
		items = append(items, a.quitItem(k))
	}
	kit.GroupHeader(k, a.T("settingsWeb"), nil)
	kit.ListGroup(k, items...)
}

// switchItem is an item of a list with a switch at its end. A switch that cannot be
// flipped now is left out of the interaction.
func (a *App) switchItem(k kit.Ctx, id, title, subtitle, iconName string, on, enabled bool, change func(on bool)) kit.ListItemSpec {
	item := kit.ListItemSpec{
		ID: id, Title: title, Subtitle: subtitle, Leading: icon(k, iconName), Wrap: a.wideSettingsWrap(k),
		Trailing: func() {
			kit.Switch(k, kit.SwitchSpec{ID: id + "-switch", On: on, Label: title, Disabled: !enabled, OnChange: change})
		},
	}
	if enabled {
		item.OnClick = func() { change(!on) }
	}
	return item
}

func (a *App) webInterfaceItem(k kit.Ctx, current settings.AppSettings, status pair.Status) kit.ListItemSpec {
	subtitle := ""
	switch {
	case status.Error != "":
		subtitle = a.T("webCannotServe", "reason", status.Error)
	case status.Running:
		subtitle = a.T("webServing", "address", status.LocalURL)
	}
	return a.switchItem(k, "web-interface", a.T("settingsWebInterface"), subtitle, "webInterface", current.WebInterface, true,
		func(on bool) { a.updateSettings(func(s *settings.AppSettings) { s.WebInterface = on }) })
}

// webOpenItem opens the web interface in the browser of this computer, through a
// link that pairs it.
func (a *App) webOpenItem(k kit.Ctx) kit.ListItemSpec {
	return kit.ListItemSpec{
		ID: "web-open", Title: a.T("webOpen"), Leading: icon(k, "openInNew"), Wrap: a.wideSettingsWrap(k),
		Trailing: func() {
			mui.Row(k.Context).Gap(gapXS).Children(func() {
				kit.IconButton(k, kit.IconButtonSpec{ID: "web-copy", Icon: "copy", Tooltip: a.T("webCopyLink"), OnClick: a.copyWebLink})
				kit.Button(k, kit.ButtonSpec{ID: "web-open-button", Kind: kit.TonalButton, Label: a.T("webOpenAction"), OnClick: a.openWeb})
			})
		},
	}
}

// openWeb opens the web interface in the browser, paired.
func (a *App) openWeb() {
	link, err := a.web.LocalLink()
	if err != nil {
		a.notify(a.T("webLinkFailed", "reason", describeError(err)))
		return
	}
	a.openURL(link)
}

// copyWebLink puts a link that pairs a browser of this computer on the clipboard.
func (a *App) copyWebLink() {
	link, err := a.web.LocalLink()
	if err != nil {
		a.notify(a.T("webLinkFailed", "reason", describeError(err)))
		return
	}
	a.copy(link)
	a.notify(a.T("actionCopied"))
}

func (a *App) lightweightItem(k kit.Ctx, current settings.AppSettings) kit.ListItemSpec {
	subtitle := a.T("settingsLightweightHelp")
	if !current.WebInterface {
		subtitle += " " + a.T("settingsNeedsWebInterface")
	}
	return a.switchItem(k, "lightweight", a.T("settingsLightweight"), subtitle, "lightweight", current.Lightweight, current.WebInterface,
		func(on bool) { a.updateSettings(func(s *settings.AppSettings) { s.Lightweight = on }) })
}

func (a *App) remoteItem(k kit.Ctx, current settings.AppSettings, status pair.Status) kit.ListItemSpec {
	subtitle := ""
	switch {
	case !current.WebInterface:
		subtitle = a.T("settingsNeedsWebInterface")
	case status.RemoteError != "":
		subtitle = a.T("webRemoteFailed", "reason", status.RemoteError)
	case status.Remote:
		subtitle = a.T("webRemoteListening", "port", itoa(status.Port))
	}
	return a.switchItem(k, "web-remote", a.T("settingsRemote"), subtitle, "otherDevices", current.WebRemote, current.WebInterface, a.allowOtherDevices)
}

// allowOtherDevices lets other devices connect, after the user has confirmed, or stops them.
func (a *App) allowOtherDevices(allow bool) {
	if !allow {
		a.updateSettings(func(s *settings.AppSettings) { s.WebRemote = false })
		return
	}
	a.confirm("web-remote", confirmation{
		Title: a.T("webRemoteConfirmTitle"), Body: a.T("webRemoteConfirmBody"), ConfirmLabel: a.T("webRemoteConfirmAction"), Icon: "otherDevices",
		OnConfirm: func() {
			a.pages.pairing = nil
			a.updateSettings(func(s *settings.AppSettings) { s.WebRemote = true })
		},
	})
}

// portItem is the port that other devices connect to.
func (a *App) portItem(k kit.Ctx, current settings.AppSettings) kit.ListItemSpec {
	// The field shows the port of the settings, also when a page of the web interface changed it.
	if effective := current.EffectiveWebPort(); !a.pages.portReady || effective != a.pages.portSeen {
		a.pages.portReady, a.pages.portSeen, a.pages.portText = true, effective, itoa(effective)
	}
	save := func(text string) {
		port, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || !settings.ValidWebPort(port) {
			a.pages.portText = itoa(a.model.Settings.Settings().EffectiveWebPort())
			a.notify(a.T("webPortInvalid", "min", itoa(settings.MinWebPort), "max", itoa(settings.MaxWebPort)))
			return
		}
		a.updateSettings(func(s *settings.AppSettings) { s.WebPort = port })
	}
	return kit.ListItemSpec{
		ID: "web-port", Title: a.T("settingsWebPort"), Subtitle: a.T("settingsWebPortHelp"), Leading: icon(k, "proxy"), Wrap: a.wideSettingsWrap(k),
		Trailing: func() {
			kit.TextField(k, kit.TextFieldSpec{
				ID: "web-port.field", Hint: itoa(settings.DefaultWebPort), Text: &a.pages.portText, Dense: true,
				OnSubmit: save, OnBlur: save,
			}).Width(120)
		},
	}
}

// publicAddressItem is the address at which something else publishes the
// interface: pairing links and the code a phone scans carry it instead of the
// address of this computer on the network.
func (a *App) publicAddressItem(k kit.Ctx, current settings.AppSettings) kit.ListItemSpec {
	if address := current.EffectiveWebAddress(); !a.pages.publicAddressReady || address != a.pages.publicAddressSeen {
		a.pages.publicAddressReady, a.pages.publicAddressSeen, a.pages.publicAddressText = true, address, address
	}
	save := func(text string) {
		address := strings.TrimRight(strings.TrimSpace(text), "/")
		if address != "" && !settings.ValidWebAddress(address) {
			a.pages.publicAddressText = a.model.Settings.Settings().EffectiveWebAddress()
			a.notify(a.T("settingsWebPublicAddressInvalid"))
			return
		}
		a.pages.publicAddressSeen = address
		a.updateSettings(func(s *settings.AppSettings) { s.WebPublicAddress = address })
	}
	return kit.ListItemSpec{
		ID: "web-public-address", Title: a.T("settingsWebPublicAddress"), Subtitle: a.T("settingsWebPublicAddressHelp"),
		Leading: icon(k, "proxy"), Wrap: a.wideSettingsWrap(k),
		Trailing: func() {
			kit.TextField(k, kit.TextFieldSpec{
				ID: "web-public-address.field", Hint: "https://vfox.example.com", Text: &a.pages.publicAddressText,
				Dense: true, OnSubmit: save, OnBlur: save,
			}).Width(260)
		},
	}
}

// publishItem publishes the interface for other devices through a quick tunnel
// of Cloudflare: a random public address, no account, while the switch is on.
func (a *App) publishItem(k kit.Ctx, status pair.Status) kit.ListItemSpec {
	on := status.TunnelURL != ""
	subtitle := a.T("settingsPublishHelp")
	switch {
	case on:
		subtitle = status.TunnelURL
	case a.pages.publishing:
		subtitle = a.T("settingsPublishBusy")
	}
	return a.switchItem(k, "web-publish", a.T("settingsPublish"), subtitle, "otherDevices", on, status.Remote && !a.pages.publishing, func(want bool) {
		if !want {
			a.web.Unpublish()
			return
		}
		body := a.T("settingsPublishConfirmBody")
		if pair.CloudflaredPath() == "" {
			body = a.T("settingsPublishDownloadBody")
		}
		a.confirm("web-publish", confirmation{
			Title: a.T("settingsPublishConfirmTitle"), Body: body,
			ConfirmLabel: a.T("settingsPublishConfirmAction"), Icon: "otherDevices",
			OnConfirm: func() { a.publish() },
		})
	})
}

// publish downloads the tunnel program when it is not there yet, then starts
// the tunnel off the main thread, and tells what became of it.
func (a *App) publish() {
	a.pages.publishing = true
	a.invalidate()
	go func() {
		if pair.CloudflaredPath() == "" {
			a.post(func() { a.notify(a.T("settingsPublishDownloading")) })
			if _, err := pair.DownloadCloudflared(context.Background()); err != nil {
				a.post(func() {
					a.pages.publishing = false
					a.notify(a.T("settingsPublishDownloadFailed", "reason", describeError(err)))
					a.invalidate()
				})
				return
			}
		}
		_, err := a.web.Publish()
		a.post(func() {
			a.pages.publishing = false
			if err != nil {
				a.notify(a.T("settingsPublishFailed", "reason", describeError(err)))
			}
			a.invalidate()
		})
	}()
}

// reachable lists the addresses that other devices connect to.
func reachable(status pair.Status) []string {
	addresses := make([]string, 0, len(status.Addresses))
	for _, address := range status.Addresses {
		addresses = append(addresses, "https://"+net.JoinHostPort(address, itoa(status.Port)))
	}
	return addresses
}

// pairItem shows the code that a phone scans.
func (a *App) pairItem(k kit.Ctx, status pair.Status) kit.ListItemSpec {
	addresses := a.pairAddresses(status)
	subtitle := a.T("webPairNone")
	if len(addresses) > 0 {
		subtitle = a.T("webReachable", "addresses", strings.Join(addresses, " · "))
	}
	return kit.ListItemSpec{
		ID: "web-pair", Title: a.T("webPair"), Subtitle: subtitle, Leading: icon(k, "qrCode"), Wrap: a.wideSettingsWrap(k),
		Trailing: func() {
			kit.Button(k, kit.ButtonSpec{
				ID: "web-pair-button", Kind: kit.TonalButton, Label: a.T("webPairShow"), Disabled: len(addresses) == 0,
				OnClick: func() { a.showPairing("") },
			})
		},
	}
}

// pairPublic is the address at which something else publishes the interface —
// the tunnel of the application, or the proxy or gateway of the settings — or
// empty when a pairing link names this computer instead.
func (a *App) pairPublic(status pair.Status) string {
	if status.TunnelURL != "" {
		return status.TunnelURL
	}
	return a.model.Settings.Settings().EffectiveWebAddress()
}

// pairAddresses are the addresses that the pairing row shows: the public one
// first, then the ones of this computer on the networks it is on.
func (a *App) pairAddresses(status pair.Status) []string {
	addresses := reachable(status)
	if public := a.pairPublic(status); public != "" {
		addresses = append([]string{public}, addresses...)
	}
	return addresses
}

// pairChoices are the addresses that the pairing dialog offers: what goes into
// the link, and the short label of it.
func (a *App) pairChoices(status pair.Status) []pairChoice {
	choices := make([]pairChoice, 0, len(status.Addresses)+1)
	if public := a.pairPublic(status); public != "" {
		choices = append(choices, pairChoice{public, strings.TrimPrefix(strings.TrimPrefix(public, "https://"), "http://")})
	}
	for _, address := range status.Addresses {
		choices = append(choices, pairChoice{address, address})
	}
	return choices
}

// certificateItem says which certificate other devices get, and lets the user bring their own.
func (a *App) certificateItem(k kit.Ctx, current settings.AppSettings, status pair.Status) kit.ListItemSpec {
	cs := k.Colors()
	kind := a.T("webCertificateOwn", "file", current.WebCertificate)
	if status.SelfSigned {
		kind = a.T("webCertificateMade")
	}
	custom := current.WebCertificate != "" || current.WebKey != ""
	return kit.ListItemSpec{
		ID: "web-certificate", Title: a.T("webCertificate"), Leading: icon(k, "certificate"), Wrap: a.wideSettingsWrap(k),
		// The fingerprint is to be compared with the one that the other device shows, so it is
		// text that can be selected, in lines that fit the item: a line of colons does not break.
		SubtitleWidget: func(mui.Color) {
			mui.Column(k.Context).Children(func() {
				kit.Label(k, kind, kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
				kit.Label(k, a.T("webFingerprint"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
				for _, line := range fingerprintLines(status.Fingerprint, 16) {
					kit.Label(k, line, kit.Type(kit.BodySmall), cs.OnSurfaceVariant, kit.Selectable(), kit.Monospace())
				}
			})
		},
		Trailing: func() {
			mui.Row(k.Context).Gap(gapXS).Children(func() {
				if custom {
					kit.IconButton(k, kit.IconButtonSpec{ID: "web-certificate-reset", Icon: "close", Tooltip: a.T("webCertificateReset"), OnClick: func() {
						a.updateSettings(func(s *settings.AppSettings) { s.WebCertificate, s.WebKey = "", "" })
					}})
				}
				kit.Button(k, kit.ButtonSpec{ID: "web-certificate-button", Kind: kit.TonalButton, Label: a.T("actionBrowse"), OnClick: a.chooseCertificate})
			})
		},
	}
}

// fingerprintLines breaks a fingerprint, the bytes of it in hexadecimal and separated by
// colons, into lines of perLine bytes.
func fingerprintLines(fingerprint string, perLine int) []string {
	bytes := strings.Split(fingerprint, ":")
	var lines []string
	for len(bytes) > perLine {
		lines = append(lines, strings.Join(bytes[:perLine], ":"))
		bytes = bytes[perLine:]
	}
	return append(lines, strings.Join(bytes, ":"))
}

// chooseCertificate asks for the certificate and then for its key.
func (a *App) chooseCertificate() {
	a.choose(chooseOptions{Title: a.T("webCertificateFile"), Initial: a.home(), FilterLabel: a.T("webCertificateFilter"), Extensions: []string{"pem", "crt", "cer"}}, func(certificate string) {
		a.choose(chooseOptions{Title: a.T("webKeyFile"), Initial: a.home(), FilterLabel: a.T("webCertificateFilter"), Extensions: []string{"pem", "key"}}, func(key string) {
			a.updateSettings(func(s *settings.AppSettings) { s.WebCertificate, s.WebKey = certificate, key })
		})
	})
}

// quitItem ends the application, which closing the window no longer does.
// The confirm dialog explains what quitting ends, so the row needs no help.
func (a *App) quitItem(k kit.Ctx) kit.ListItemSpec {
	return kit.ListItemSpec{
		ID: "web-quit", Title: a.T("webQuit"), Leading: icon(k, "quit"), Wrap: a.wideSettingsWrap(k),
		Trailing: func() {
			kit.Button(k, kit.ButtonSpec{ID: "web-quit-button", Kind: kit.TonalButton, Label: a.T("webQuitAction"), OnClick: func() {
				a.confirm("web-quit", confirmation{
					Title: a.T("webQuitConfirmTitle"), Body: a.T("webQuitConfirmBody"), ConfirmLabel: a.T("webQuitAction"), Icon: "quit", Destructive: true,
					OnConfirm: a.desktop.quit,
				})
			}})
		},
	}
}

// showPairing opens the dialog with the code that a phone scans, for host, the address of
// this computer, or the likeliest one when it is empty.
func (a *App) showPairing(host string) {
	a.makePairing(host)
	a.env.ShowDialog("web-pair", func(k kit.Ctx, closeDialog func()) kit.DialogSpec {
		cs := k.Colors()
		pairing := a.pages.pairing
		return kit.DialogSpec{
			Icon: "qrCode", Title: a.T("webPair"), MaxWidth: 440,
			Content: func() {
				mui.Column(k.Context).Gap(gapM).AlignItems(mui.Center).Children(func() {
					a.pairingContent(k, pairing)
				})
			},
			Actions: []func(){
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "copy-link", Kind: kit.TextButton, Label: a.T("webCopyLink"), Disabled: pairing == nil || pairing.err != nil, OnClick: func() {
						a.copy(pairing.link)
						a.notify(a.T("actionCopied"))
					}})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "new-code", Kind: kit.TextButton, Label: a.T("webPairNew"), OnClick: func() { a.makePairing(pairing.host) }})
				},
				func() {
					kit.Button(k, kit.ButtonSpec{ID: "close", Kind: kit.TextButton, Label: a.T("actionClose"), OnClick: func() {
						a.pages.pairing = nil
						closeDialog()
					}, Content: cs.Primary})
				},
			},
		}
	})
}

// makePairing makes a link for a phone, and the code of it.
func (a *App) makePairing(host string) {
	link, err := a.web.RemoteLink(host)
	pairing := &pairingState{link: link, host: host, err: err}
	if err == nil {
		pairing.matrix, pairing.err = pair.QRMatrix(link)
	}
	a.pages.pairing = pairing
}

// pairingContent is what the pairing dialog shows: the code, how to use it, the addresses
// to choose from when the computer has more than one, and the link.
func (a *App) pairingContent(k kit.Ctx, pairing *pairingState) {
	cs := k.Colors()
	if pairing == nil || pairing.err != nil {
		reason := a.T("webPairNone")
		if pairing != nil {
			reason = a.T("webLinkFailed", "reason", describeError(pairing.err))
		}
		kit.Label(k, reason, kit.Type(kit.BodyMedium), cs.Error)
		return
	}
	qrCode(k, pairing.matrix, 240)
	kit.Label(k, a.T("webPairHelp", "minutes", itoa(int(pair.DefaultTokenTTL.Minutes()))), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant)
	if choices := a.pairChoices(a.web.Status()); len(choices) > 1 {
		segments := make([]kit.Segment, 0, len(choices))
		selected := pairing.host
		if selected == "" {
			selected = choices[0].value
		}
		for _, choice := range choices {
			segments = append(segments, kit.Segment{Value: choice.value, Label: choice.label})
		}
		kit.ButtonGroup(k, kit.ButtonGroupSpec{
			ID: "pair-hosts", Selected: selected, Segments: segments,
			OnSelect: func(address string) { a.makePairing(address) },
		})
	}
	kit.Label(k, pairing.link, kit.Type(kit.BodySmall), cs.OnSurfaceVariant, kit.Selectable(), kit.Monospace())
}

// qrCode draws the modules of a QR code, black on white whatever the theme is: a scanner
// reads the contrast and not the colours. The matrix has the quiet zone around the code.
func qrCode(k kit.Ctx, matrix [][]bool, size float32) mui.Element {
	el := mui.Column(k.Context).Size(size, size).Shrink(0)
	black, white := mui.RGB(0, 0, 0), mui.RGB(255, 255, 255)
	el.Draw(func(p *mui.Painter, r mui.Rect) {
		p.Fill(r, white, 12)
		module := min(r.W, r.H) / float32(len(matrix))
		for y, row := range matrix {
			for x := 0; x < len(row); x++ {
				if !row[x] {
					continue
				}
				end := x
				for end < len(row) && row[end] {
					end++
				}
				p.Fill(mui.Rect{X: r.X + float32(x)*module, Y: r.Y + float32(y)*module, W: float32(end-x) * module, H: module}, black, 0)
				x = end
			}
		}
	})
	return el
}
