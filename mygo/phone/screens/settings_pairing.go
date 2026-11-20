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
	"sync"

	"github.com/egoist/mygo/ui"
	"github.com/skip2/go-qrcode"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// The size of the pairing dialog and of its code, in dp.
const (
	settingsPairingWidth = 440
	settingsPairingSize  = 240
)

// The colours of a code. A scanner reads the contrast and not the colours, and a
// code in the colours of the theme would be inverted in the dark theme, which some
// scanners do not read; so a code is dark on light whatever the theme is.
var (
	settingsQRDark  = ui.RGB(0, 0, 0)
	settingsQRLight = ui.RGB(255, 255, 255)
)

// settingsPairing is the state of the pairing dialog: the address the link names,
// and the link and its code once the computer has made them. The computer answers
// on a goroutine of its own, so the state is guarded by mu. gen counts the requests,
// so that an answer to an address that was replaced is dropped.
type settingsPairing struct {
	mu      sync.Mutex
	gen     int
	host    string
	code    remote.PairingCode
	matrix  [][]bool
	err     error
	pending bool
}

// settingsOpenPairing opens the dialog with a pairing link for the likeliest address
// of the computer, and asks the computer for it.
func (r *Root) settingsOpenPairing(k kit.Ctx) {
	web, _ := r.service.Web().Value()
	st := &settingsPairing{}
	k.Env.ShowDialog("set-pairing", func(c kit.Ctx, closeDialog func()) kit.DialogSpec {
		return r.settingsPairingDialog(c, st, closeDialog)
	})
	r.settingsRequestPairing(st, settingsInitialHost(web))
}

// settingsRequestPairing asks the computer for a pairing link for host, in the
// background. The answer is shown by the dialog; a failure is also a notice, which
// the window shows as a snack bar.
func (r *Root) settingsRequestPairing(st *settingsPairing, host string) {
	st.mu.Lock()
	st.gen++
	gen := st.gen
	st.host, st.pending, st.err = host, true, nil
	st.mu.Unlock()
	r.Do(func(ctx context.Context) error {
		code, err := r.service.WebPairing(ctx, host)
		var matrix [][]bool
		if err == nil {
			matrix, err = settingsQRMatrix(code.URL)
		}
		st.mu.Lock()
		if st.gen == gen {
			st.code, st.matrix, st.err, st.pending = code, matrix, err, false
		}
		st.mu.Unlock()
		r.invalidate()
		return err
	}, "")
}

// settingsPairingDialog is the dialog with the code: the code itself, how long it
// works, the addresses to choose from when the computer has more than one, and the
// link and the certificate, which stay folded away.
func (r *Root) settingsPairingDialog(c kit.Ctx, st *settingsPairing, closeDialog func()) kit.DialogSpec {
	cs := c.Colors()
	web, _ := r.service.Web().Value()
	st.mu.Lock()
	host, code, matrix, err, pending := st.host, st.code, st.matrix, st.err, st.pending
	st.mu.Unlock()
	hosts := settingsHostChoices(web)
	return kit.DialogSpec{
		Icon: "qrCode", Title: i18n.Text("webPairBtn"), MaxWidth: settingsPairingWidth,
		Content: func() {
			ui.Column(c.Context).Gap(12).FillWidth().AlignItems(ui.Center).Children(func() {
				switch {
				case err != nil:
					kit.Label(c, i18n.Text("failed"), kit.Type(kit.BodyMedium), cs.Error, kit.Centered())
				case matrix == nil || pending:
					kit.LoadingIndicator(c, kit.LoadingSpec{ID: "pairing", Size: 48, Label: i18n.Text("loading")})
				default:
					settingsQRCode(c, matrix, settingsPairingSize)
					kit.Label(c, i18n.Text("pairScan"), kit.Type(kit.BodyMedium), cs.OnSurfaceVariant, kit.Centered(), kit.Breakable())
					kit.Label(c, i18n.Text("setPairOnce"), kit.Type(kit.BodySmall), cs.OnSurfaceVariant, kit.Centered(), kit.Breakable())
				}
				if len(hosts) > 1 {
					kit.ButtonGroup(c, kit.ButtonGroupSpec{
						ID: "pair-hosts", Selected: host, Segments: hosts,
						OnSelect: func(address string) { r.settingsRequestPairing(st, address) },
						MaxWidth: kit.DialogContentWidth(c, settingsPairingWidth), Collapse: true,
					})
				}
				if code.URL != "" {
					settingsPairingDetails(c, code.URL, web.Fingerprint)
				}
			})
		},
		Actions: []func(){
			func() {
				kit.Button(c, kit.ButtonSpec{
					ID: "copy-link", Kind: kit.TextButton, Label: i18n.Text("webCopyLink"), Disabled: code.URL == "",
					OnClick: func() {
						settingsCopy(code.URL)
						c.Env.ShowSnackbar(kit.Snack{Text: i18n.Text("copied")})
					},
				})
			},
			func() {
				kit.Button(c, kit.ButtonSpec{ID: "close", Kind: kit.TextButton, Label: i18n.Text("close"), OnClick: closeDialog})
			},
		},
	}
}

// settingsPairingDetails is the link and the certificate of the pairing, folded away
// under a button: the code is what a phone needs.
func settingsPairingDetails(c kit.Ctx, link, fingerprint string) {
	cs := c.Colors()
	column := ui.Column(c.Context).Gap(8).FillWidth()
	open := ui.Local(column, "details", func() bool { return false })
	column.Children(func() {
		kit.Button(c, kit.ButtonSpec{
			ID: "details", Kind: kit.TextButton, Label: i18n.Text("pairDetails"), Icon: "expandMore",
			OnClick: func() { *open = !*open },
		})
		kit.Collapsible(c, "details-body", *open, func() {
			ui.Column(c.Context).Gap(4).FillWidth().Children(func() {
				kit.Label(c, link, kit.Type(kit.BodySmall), cs.OnSurface, kit.Monospace(), kit.Breakable(), kit.Selectable())
				kit.Label(c, i18n.Text("webFingerprint"), kit.Type(kit.BodySmall), cs.OnSurfaceVariant, kit.Breakable())
				kit.Label(c, fingerprint, kit.Type(kit.BodySmall), cs.OnSurface, kit.Monospace(), kit.Breakable(), kit.Selectable())
			})
		})
	})
}

// settingsHostChoices are the addresses that a pairing link can name: the public
// address first, then the others, each once. Each is labelled without its scheme,
// as the web labels them.
func settingsHostChoices(web remote.WebState) []kit.Segment {
	seen := map[string]bool{}
	var choices []kit.Segment
	add := func(address string) {
		if address == "" || seen[address] {
			return
		}
		seen[address] = true
		label := strings.TrimPrefix(strings.TrimPrefix(address, "https://"), "http://")
		choices = append(choices, kit.Segment{Value: address, Label: label})
	}
	add(web.PublicAddress)
	for _, address := range web.Addresses {
		add(address)
	}
	return choices
}

// settingsInitialHost is the address that a pairing link names first: the public
// address when there is one, else the first address of the computer, else none, which
// lets the computer choose.
func settingsInitialHost(web remote.WebState) string {
	switch {
	case web.PublicAddress != "":
		return web.PublicAddress
	case len(web.Addresses) > 0:
		return web.Addresses[0]
	}
	return ""
}

// settingsQRMatrix encodes text as a QR code with the medium error correction, and
// returns its modules row by row, a true one being dark. The matrix has the quiet zone
// around the code, which a scanner needs.
func settingsQRMatrix(text string) ([][]bool, error) {
	code, err := qrcode.New(text, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	return code.Bitmap(), nil
}

// settingsQRCode draws a matrix of settingsQRMatrix, as a square of size dp. Each run
// of dark modules is one rectangle.
func settingsQRCode(c kit.Ctx, matrix [][]bool, size float32) {
	el := ui.Column(c.Context).Size(size, size).Shrink(0)
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, settingsQRLight, 12)
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
				p.Fill(ui.Rect{X: r.X + float32(x)*module, Y: r.Y + float32(y)*module, W: float32(end-x) * module, H: module}, settingsQRDark, 0)
				x = end
			}
		}
	})
}
