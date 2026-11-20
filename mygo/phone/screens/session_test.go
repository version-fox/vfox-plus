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
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
)

// pairWrongFingerprint is a fingerprint that no certificate has: the saved computer of the
// test presents another certificate, which the phone refuses.
const pairWrongFingerprint = "00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:00"

// TestAConnectingPhoneShowsItsProgress checks the strip above the tabs while the saved
// computer is checked: the tabs are shown, with the progress.
func TestAConnectingPhoneShowsItsProgress(t *testing.T) {
	fd := newPairDesktop(t)
	root := pairSavedRoot(t, fd.server.URL, fd.fingerprint, true)
	tt := ui.NewTester(root.View, 480, 900)
	tt.Frame()
	if root.Service().Status().Session != app.Connecting {
		t.Fatalf("the session is %v, want connecting", root.Service().Status().Session)
	}
	pairWaitFor(t, tt, i18n.TextIn(i18n.English, "sessionConnecting"))
	if !hasText(tt, "Settings") {
		t.Errorf("a connecting phone shows %q; want the tabs", tt.Texts())
	}
	pairSavePicture(t, tt, "session-connecting")
}

// TestAnUnreachableComputerShowsRetry checks that a computer that does not answer keeps
// the tabs, says so, and offers the retry.
func TestAnUnreachableComputerShowsRetry(t *testing.T) {
	fd := newPairDesktop(t)
	root := pairSavedRoot(t, fd.server.URL, fd.fingerprint, true)
	fd.server.Close()
	pairStartService(t, root)
	tt := ui.NewTester(root.View, 480, 900)
	pairWaitUntil(t, tt, "the computer to be unreachable", func() bool {
		return root.Service().Status().Session == app.Unreachable
	})
	pairWaitFor(t, tt, "Retry")
	if !hasText(tt, noticeText(app.KindUnreachable)) {
		t.Errorf("the strip does not say why; the frame shows %q", tt.Texts())
	}
	if err := tt.Click("Retry"); err != nil {
		t.Fatalf("Click(Retry): %v", err)
	}
	tt.Frame()
	pairSavePicture(t, tt, "session-unreachable")
}

// TestANeedsPairingComputerShowsTheExplanation checks the pairing page of a phone whose
// computer no longer accepts it: the explanation, and the computer it had.
func TestANeedsPairingComputerShowsTheExplanation(t *testing.T) {
	fd := newPairDesktop(t)
	root := pairSavedRoot(t, fd.server.URL, fd.fingerprint, false)
	tt := ui.NewTester(root.View, 480, 900)
	tt.Frame()
	if root.Service().Status().Session != app.NeedsPairing {
		t.Fatalf("the session is %v, want needs pairing", root.Service().Status().Session)
	}
	for _, text := range []string{"This phone must pair again", noticeText(app.KindUnpaired), "Last computer", fd.server.URL, "Pair again"} {
		if !hasText(tt, text) {
			t.Errorf("the pairing page has no %q; the frame shows %q", text, tt.Texts())
		}
	}
	if hasText(tt, "Settings") {
		t.Error("a phone that must pair again shows the tabs")
	}
	// Pair again takes the link of the clipboard, as the computer shows it.
	link := fd.server.URL + "/auth/connect/" + pairTestToken
	tt.SetClipboard(link)
	if err := tt.Click("Pair again"); err != nil {
		t.Fatalf("Click(Pair again): %v", err)
	}
	tt.Frame()
	if root.linkText != link {
		t.Errorf("Pair again left the field with %q, want the clipboard's link %q", root.linkText, link)
	}
	pairSavePicture(t, tt, "session-needs-pairing")
}

// TestACertificateThatChangedIsWarnedAndNotTrusted checks a computer whose certificate is
// not the one saved: the phone refuses it, the pairing page warns, and nothing is trusted.
func TestACertificateThatChangedIsWarnedAndNotTrusted(t *testing.T) {
	fd := newPairDesktop(t)
	root := pairSavedRoot(t, fd.server.URL, pairWrongFingerprint, true)
	pairStartService(t, root)
	tt := ui.NewTester(root.View, 480, 900)
	pairWaitUntil(t, tt, "the refusal of the certificate", func() bool {
		return root.Service().Status().Session == app.CertificateChanged
	})
	pairWaitFor(t, tt, "The computer's certificate changed")
	if !hasText(tt, "Pair again") {
		t.Errorf("the warning offers no way to pair again; the frame shows %q", tt.Texts())
	}
	if !hasText(tt, noticeText(app.KindCertificateChanged)) {
		t.Errorf("the warning does not say why; the frame shows %q", tt.Texts())
	}
	fd.mu.Lock()
	connects := fd.connects
	fd.mu.Unlock()
	if connects != 0 {
		t.Errorf("the phone exchanged a token with the changed computer %d times", connects)
	}
	pairSavePicture(t, tt, "session-cert-changed")
}

// phoneCardView draws the card of this phone alone, with what the card shows above it.
func phoneCardView(root *Root) func(c *ui.Context) {
	return func(c *ui.Context) {
		root.applyTheme(c)
		k := root.env.Begin(c)
		root.phoneCard(k)
		root.env.LayoutDialogs(k)
		root.env.LayoutSnackbars(k)
	}
}

// TestThePhoneCardShowsTheComputerAndPairsOrDisconnects checks the card of this phone:
// the address of the computer, the page of a new link that keeps the session, and the
// disconnection, which waits for the confirmation and then forgets the computer.
func TestThePhoneCardShowsTheComputerAndPairsOrDisconnects(t *testing.T) {
	fd := newPairDesktop(t)
	root := pairSavedRoot(t, fd.server.URL, fd.fingerprint, true)
	pairStartService(t, root)
	paired := ui.NewTester(root.View, 480, 900)
	pairWaitUntil(t, paired, "the connection", func() bool {
		return root.Service().Status().Session == app.Paired
	})
	tt := ui.NewTester(phoneCardView(root), 480, 900)
	for _, text := range []string{"This phone", fd.server.URL, "Pair a new link", "Disconnect"} {
		if !hasText(tt, text) {
			t.Errorf("the card has no %q; the frame shows %q", text, tt.Texts())
		}
	}
	if err := tt.Click("Pair a new link"); err != nil {
		t.Fatalf("Click(Pair a new link): %v", err)
	}
	tt.Frame()
	if pages := root.App().Pages(); len(pages) != 1 || pages[0].Kind != app.PagePair {
		t.Errorf("pairing a new link opened %v; want the pairing page", pages)
	}
	if root.Service().Status().Session != app.Paired {
		t.Errorf("pairing a new link changed the session to %v", root.Service().Status().Session)
	}
	root.App().Pop()

	if err := tt.Click("Disconnect"); err != nil {
		t.Fatalf("Click(Disconnect): %v", err)
	}
	pairWaitFor(t, tt, i18n.TextIn(i18n.English, "phoneOutTitle"))
	if root.Service().Status().Session != app.Paired {
		t.Errorf("the phone disconnected before the confirmation")
	}
	pairSavePicture(t, tt, "phone-card")
	if err := tt.Click("Disconnect"); err != nil {
		t.Fatalf("Click(Disconnect) to confirm: %v", err)
	}
	pairWaitUntil(t, tt, "the disconnection", func() bool { return root.Service().Status().Session == app.Unpaired })
}
