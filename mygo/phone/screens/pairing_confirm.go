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
	"fmt"
	"sync"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// pairConfirmID is the id of the dialog that asks the user to trust the fingerprint of
// the computer. It is open once at a time.
const pairConfirmID = "pairConfirm"

// pairFlow is one pairing, from the tap on Trust and pair to its outcome. The dialog
// keeps it while it is open: the goroutine of the call sets it under mu, and the dialog
// takes it on the main thread, at its next build.
type pairFlow struct {
	mu sync.Mutex
	// busy is the call in progress.
	busy bool
	// paired is set when the call paired the phone; the dialog then closes.
	paired bool
	// failure is the kind of the error that the call ended with, KindNone for none.
	failure app.Kind
}

// begin marks the call as running.
func (f *pairFlow) begin() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.busy, f.failure = true, app.KindNone
}

// end records the error that the call ended with, nil for a call that paired the phone.
func (f *pairFlow) end(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.busy = false
	if err == nil {
		f.paired = true
		return
	}
	f.failure = app.KindOf(err)
}

// state returns what the call is doing, and how it ended.
func (f *pairFlow) state() (bool, bool, app.Kind) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.busy, f.paired, f.failure
}

// pairOpenConfirm opens the dialog that asks the user to trust the fingerprint of the
// computer that the field names. The dialog probes the computer and shows its
// fingerprint; it pairs only when the user taps Trust and pair. The pairing page, a
// deep link and a picture all open it, and none of them pairs by itself.
func (r *Root) pairOpenConfirm() {
	flow := &pairFlow{}
	r.env.ShowDialog(pairConfirmID, func(c kit.Ctx, close func()) kit.DialogSpec {
		return r.pairConfirmSpec(c, close, flow)
	})
}

// pairConfirmSpec builds the dialog for the link in the field, in the state that the
// probe and the call are in. Each frame reads them again.
func (r *Root) pairConfirmSpec(c kit.Ctx, close func(), flow *pairFlow) kit.DialogSpec {
	spec := kit.DialogSpec{Icon: "certificate", Title: i18n.Text("pairConfirmTitle")}
	busy, paired, failure := flow.state()
	switch {
	case paired:
		// The phone is paired: the home tab shows the computer, above any page.
		r.app.SelectTab(app.TabHome)
		close()
		return spec
	case failure == app.KindCanceled:
		// The app went to the background during the call. The call is not an error.
		close()
		return spec
	case busy:
		spec.Content = func() { pairBusyContent(c) }
		return spec
	case failure != app.KindNone:
		spec.Icon, spec.IconError = "error", true
		spec.Content = func() { pairTextContent(c, noticeText(failure), c.Colors().Error) }
		spec.Actions = []func(){pairCloseAction(c, close)}
		if failure == app.KindCertificateChanged {
			clipboard := c.Services()
			spec.Actions = append(spec.Actions, func() {
				kit.Button(c, kit.ButtonSpec{
					ID:    "pairAgain",
					Kind:  kit.FilledButton,
					Label: i18n.Text("pairAgain"),
					Icon:  "copy",
					OnClick: func() {
						if key := r.pairPasteClipboard(clipboard); key != "" {
							c.Env.ShowSnackbar(kit.Snack{Text: i18n.Text(key)})
							return
						}
						close()
					},
				})
			})
		}
		return spec
	}

	link, err := pairing.ParseLink(r.linkText)
	if err != nil {
		spec.Icon, spec.IconError = "error", true
		spec.Content = func() { pairTextContent(c, noticeText(app.KindOf(err)), c.Colors().Error) }
		spec.Actions = []func(){pairCloseAction(c, close)}
		return spec
	}
	// The probe of the link: the first read starts it, and the reads of the same link
	// share it.
	probe := r.service.Probe(link)
	switch probe.Phase() {
	case app.Ready:
	case app.Failed:
		spec.Icon, spec.IconError = "error", true
		spec.Content = func() { pairTextContent(c, noticeText(app.KindOf(probe.Err())), c.Colors().Error) }
		spec.Actions = []func(){pairCloseAction(c, close)}
		return spec
	default:
		spec.Content = func() { pairBusyContent(c, i18n.Text("pairProbing")) }
		spec.Actions = []func(){pairCloseAction(c, close)}
		return spec
	}

	candidate, ok := probe.Value()
	if !ok {
		return spec
	}
	known := r.service.Status().Server.Fingerprint
	spec.Content = func() {
		ui.Column(c.Context).Gap(12).Children(func() {
			kit.Label(c, i18n.Text("pairConfirmBody"), kit.Type(kit.BodyMedium), c.Colors().OnSurfaceVariant)
			kit.Label(c, candidate.Fingerprint, kit.Type(kit.TitleSmall), c.Colors().OnSurface, kit.Monospace(), kit.Breakable(), kit.Selectable())
			if known != "" && known != candidate.Fingerprint {
				kit.Label(c, i18n.Text("pairCertDiffers"), kit.Type(kit.BodyMedium), c.Colors().Error)
			}
		})
	}
	spec.Actions = []func(){
		func() {
			kit.Button(c, kit.ButtonSpec{ID: "pairCancel", Kind: kit.TextButton, Label: i18n.Text("cancel"), OnClick: close})
		},
		func() {
			kit.Button(c, kit.ButtonSpec{
				ID:      "trustAndPair",
				Kind:    kit.FilledButton,
				Label:   i18n.Text("trustAndPair"),
				Icon:    "verified",
				OnClick: func() { r.pairTrust(candidate, flow) },
			})
		},
	}
	return spec
}

// pairTrust pairs the phone with the computer of candidate, in the background, and
// records how the call ended in flow: the dialog shows it, and closes once the phone is
// paired. The fingerprint that the user compared is the one the pairing pins.
func (r *Root) pairTrust(candidate app.Candidate, flow *pairFlow) {
	flow.begin()
	r.Do(func(ctx context.Context) error {
		err := r.service.Pair(ctx, candidate)
		if err == nil {
			fmt.Printf("vfoxplus: paired host=%s fingerprint=%s\n", candidate.Link.Base.Hostname(), candidate.Fingerprint)
		}
		flow.end(err)
		r.invalidate()
		return nil
	}, "")
}

// pairBusyContent is the content of the dialog while it waits: a progress indicator,
// and the text beside it (the pairing's by default).
func pairBusyContent(c kit.Ctx, text ...string) {
	label := i18n.Text("pairPairing")
	if len(text) > 0 {
		label = text[0]
	}
	ui.Column(c.Context).Gap(12).Children(func() {
		kit.LoadingIndicator(c, kit.LoadingSpec{ID: "pairBusy", Size: 40, Label: label})
		kit.Label(c, label, kit.Type(kit.BodyMedium), c.Colors().OnSurfaceVariant, kit.Lines(2))
	})
}

// pairTextContent is the content of the dialog that says one message, in colour.
func pairTextContent(c kit.Ctx, text string, colour ui.Color) {
	kit.Label(c, text, kit.Type(kit.BodyMedium), colour)
}

// pairCloseAction is the button that closes a dialog.
func pairCloseAction(c kit.Ctx, close func()) func() {
	return func() {
		kit.Button(c, kit.ButtonSpec{ID: "pairClose", Kind: kit.TextButton, Label: i18n.Text("close"), OnClick: close})
	}
}
