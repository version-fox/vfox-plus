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
	"net/url"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// TestThePairingLinkOfADeepLinkIsTheDecodedLink checks the deep link: the link is the
// percent-decoded value of its link parameter, and ParseLink reads it as the pairing
// link it is.
func TestThePairingLinkOfADeepLinkIsTheDecodedLink(t *testing.T) {
	link := "https://192.168.1.10:8369/auth/connect/Ab-9_z"
	raw := "vfoxplus://pair?link=" + url.QueryEscape(link)
	got, ok := pairingDeepLink(raw)
	if !ok {
		t.Fatalf("pairingDeepLink(%q) says it is not a deep link of the pairing page", raw)
	}
	if got != link {
		t.Fatalf("the link is %q, want %q", got, link)
	}
	parsed, err := pairing.ParseLink(got)
	if err != nil {
		t.Fatalf("ParseLink(%q): %v", got, err)
	}
	if parsed.Base.String() != "https://192.168.1.10:8369" || parsed.Token != "Ab-9_z" {
		t.Errorf("the link names %v and token %q", parsed.Base, parsed.Token)
	}
}

// TestDeepLinksOfOtherPagesAndSchemesAreNotPairing checks that only vfoxplus://pair is
// the pairing page's deep link.
func TestDeepLinksOfOtherPagesAndSchemesAreNotPairing(t *testing.T) {
	for _, raw := range []string{
		"",
		"https://192.168.1.10:8369/auth/connect/abc",
		"vfoxplus://sdk/go",
		"otherapp://pair?link=x",
		"not a url at all: %",
	} {
		if link, ok := pairingDeepLink(raw); ok {
			t.Errorf("pairingDeepLink(%q) = %q, true; want not a deep link", raw, link)
		}
	}
}

// TestADeepLinkWithoutALinkStillOpensThePage checks that the page opens, with an empty
// field, when the deep link carries no link.
func TestADeepLinkWithoutALinkStillOpensThePage(t *testing.T) {
	link, ok := pairingDeepLink("vfoxplus://pair")
	if !ok || link != "" {
		t.Errorf("pairingDeepLink without a link = %q, %v; want \"\", true", link, ok)
	}
}

// TestADeepLinkWithAnInvalidLinkIsSaidByTheField checks that the link of a deep link is
// validated like a pasted one: the message is the one of a text that is not a link.
func TestADeepLinkWithAnInvalidLinkIsSaidByTheField(t *testing.T) {
	link, ok := pairingDeepLink("vfoxplus://pair?link=" + url.QueryEscape("http://192.168.1.10/auth/connect/abc"))
	if !ok {
		t.Fatal("the deep link is not recognised")
	}
	if got := pairLinkError(link); got != noticeText(app.KindNotLink) {
		t.Errorf("pairLinkError = %q, want the message of a text that is not a link", got)
	}
	if pairFieldHoldsLink(link) {
		t.Error("the Connect button would be enabled for an http link")
	}
}

// TestPairLinkErrorIsEmptyForAValidLinkAndForNothing checks the validation of the field.
func TestPairLinkErrorIsEmptyForAValidLinkAndForNothing(t *testing.T) {
	if got := pairLinkError("   "); got != "" {
		t.Errorf("pairLinkError of blank text = %q, want none", got)
	}
	valid := "https://192.168.1.10:8369/auth/connect/abc123"
	if got := pairLinkError(valid); got != "" {
		t.Errorf("pairLinkError of a valid link = %q, want none", got)
	}
	if !pairFieldHoldsLink(" " + valid + " ") {
		t.Error("a valid link with spaces around it does not enable Connect")
	}
	if got := pairLinkError("hello"); got != noticeText(app.KindNotLink) {
		t.Errorf("pairLinkError of text = %q, want the message of a text that is not a link", got)
	}
}

// TestTheConfirmationFlowRecordsItsOutcome checks the states of a pairing: a call that
// paired, one that failed with the kind of its error, and one that the app cancelled.
func TestTheConfirmationFlowRecordsItsOutcome(t *testing.T) {
	var flow pairFlow
	flow.begin()
	if busy, paired, failure := flow.state(); !busy || paired || failure != app.KindNone {
		t.Fatalf("after begin: busy %v, paired %v, failure %v", busy, paired, failure)
	}
	flow.end(nil)
	if busy, paired, _ := flow.state(); busy || !paired {
		t.Errorf("after a pairing: busy %v, paired %v; want idle and paired", busy, paired)
	}

	var refused pairFlow
	refused.begin()
	refused.end(remote.ErrCertificateChanged)
	if _, paired, failure := refused.state(); paired || failure != app.KindCertificateChanged {
		t.Errorf("after a changed certificate: paired %v, failure %v", paired, failure)
	}

	var cancelled pairFlow
	cancelled.begin()
	cancelled.end(context.Canceled)
	if _, _, failure := cancelled.state(); failure != app.KindCanceled {
		t.Errorf("after a cancelled call: failure %v, want %v", failure, app.KindCanceled)
	}
}
