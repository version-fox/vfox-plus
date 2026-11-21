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
	"net/url"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
)

// The deep link that opens the pairing page with a link:
// vfoxplus://pair?link=<the pairing link, percent-encoded>.
const (
	pairScheme = "vfoxplus"
	pairHost   = "pair"
)

// pairingDeepLink returns the pairing link that a deep link carries, and whether raw
// is a deep link of the pairing page at all. A deep link without a link still opens
// the page; the link is then empty.
func pairingDeepLink(raw string) (string, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || !strings.EqualFold(u.Scheme, pairScheme) || u.Host != pairHost {
		return "", false
	}
	// Query decodes the percent-encoding of the link, which the deep link carries.
	return strings.TrimSpace(u.Query().Get("link")), true
}

// OpenURL handles a deep link that the system passes to the app. A link of the
// pairing page opens the pairing page, fills the field with the pairing link, starts
// the probe of its computer and asks the user to confirm the fingerprint. It never
// pairs by itself: the confirmation is the user's. The app's callbacks run on the
// main thread, so OpenURL may change the window.
func (r *Root) OpenURL(raw string) {
	link, ok := pairingDeepLink(raw)
	if !ok {
		return
	}
	if r.app.Screen() == app.ScreenTabs {
		r.app.Push(app.Page{Kind: app.PagePair})
	}
	r.pairConnectText(link)
}

// pairConnectText puts text in the field of the link. When text is a pairing link, it
// starts the probe of its computer and opens the dialog that asks the user to confirm
// the fingerprint. A text that is not a link stays in the field, where the field says
// why it is not one. It runs on the main thread.
func (r *Root) pairConnectText(text string) {
	r.linkText = text
	link, err := pairing.ParseLink(text)
	if err != nil {
		return
	}
	// The probe starts with this read. The dialog reads the same link from the field,
	// so the probe it shows is this one.
	r.service.Probe(link)
	r.pairOpenConfirm()
}

// pairLinkError returns the message for the text of the field when the text is not a
// pairing link, and "" when it is one or when it is empty. The message is the one of
// the error's kind, as the rest of the app says it.
func pairLinkError(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}
	if _, err := pairing.ParseLink(text); err != nil {
		return noticeText(app.KindOf(err))
	}
	return ""
}

// pairFieldHoldsLink reports whether the field holds a pairing link, which is the
// condition of the Connect button.
func pairFieldHoldsLink(text string) bool {
	_, err := pairing.ParseLink(strings.TrimSpace(text))
	return err == nil
}

// pairPasteClipboard puts the text of the clipboard in the field of the link. It
// reports the key of the message that says why nothing was pasted, or "" when the
// text was pasted. The clipboard is read on the main thread.
func (r *Root) pairPasteClipboard(clipboard ui.Services) string {
	text := strings.TrimSpace(clipboard.ReadClipboard())
	if text == "" {
		return "pairClipboardEmpty"
	}
	r.linkText = text
	return ""
}
