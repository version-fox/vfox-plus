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

package gateway

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAPairingLinkIsTakenApartForWhatItMayBeUsedFor(t *testing.T) {
	good := map[string]struct{ base, token string }{
		"https://192.168.1.20:8369/auth/connect/0123456789abcdef0123456789abcdef": {"https://192.168.1.20:8369", "0123456789abcdef0123456789abcdef"},
		"  https://host.example:8369/auth/connect/abcdefgh12345678  \n":           {"https://host.example:8369", "abcdefgh12345678"},
		"https://[fe80::1]:8369/auth/connect/abcdefgh12345678":                    {"https://[fe80::1]:8369", "abcdefgh12345678"},
		"http://127.0.0.1:41233/auth/connect/abcdefgh12345678":                    {"http://127.0.0.1:41233", "abcdefgh12345678"},
		"http://localhost:41233/auth/connect/abcdefgh12345678":                    {"http://localhost:41233", "abcdefgh12345678"},
		"http://[::1]:41233/auth/connect/abcdefgh12345678":                        {"http://[::1]:41233", "abcdefgh12345678"},
		// Whatever follows the token is not the instance's address, and is dropped.
		"https://host:8369/auth/connect/abcdefgh12345678?x=1#fragment": {"https://host:8369", "abcdefgh12345678"},
	}
	for link, want := range good {
		base, token, err := parseLink(link)
		if err != nil || base.String() != want.base || token != want.token {
			t.Errorf("parseLink(%q) = %v, %q, %v; want %s, %q", link, base, token, err, want.base, want.token)
		}
	}
	bad := map[string]string{
		"":                                                      "not a pairing link",
		"hello":                                                 "not a pairing link",
		"https://":                                              "not a pairing link",
		"://x/auth/connect/abcdefgh12345678":                    "not a pairing link",
		"ftp://host/auth/connect/abcdefgh1234":                  "begins with https://",
		"file:///auth/connect/abcdefgh1234":                     "not a pairing link",
		"https://host/auth/connect/":                            "not a pairing link",
		"https://host/auth/connect/short":                       "not a pairing link",
		"https://host/auth/connect/ab/cdefghij":                 "not a pairing link",
		"https://host/auth/connect/..%2f..%2fx":                 "not a pairing link",
		"https://host/auth/connect/abcdefgh!!!":                 "not a pairing link",
		"https://host/elsewhere/abcdefgh12345":                  "not a pairing link",
		"https://host/api/token":                                "not a pairing link",
		"https://u:p@host/auth/connect/abcdefgh12345678":        "not a pairing link",
		"http://192.168.1.2:8369/auth/connect/abcdefgh12345678": "only on the loopback address",
		"http://host.example/auth/connect/abcdefgh12345678":     "only on the loopback address",
	}
	for link, reason := range bad {
		if base, _, err := parseLink(link); err == nil || !strings.Contains(err.Error(), reason) {
			t.Errorf("parseLink(%q) = %v, %v; want an error with %q", link, base, err, reason)
		}
	}
}

func TestOnlyTheLoopbackIsTheLoopback(t *testing.T) {
	for host, want := range map[string]bool{
		"localhost": true, "LOCALHOST": true, "127.0.0.1": true, "127.1.2.3": true, "::1": true, "[::1]": true,
		"192.168.1.2": false, "10.0.0.1": false, "example.com": false, "localhost.example.com": false, "": false, "0.0.0.0": false,
	} {
		if got := isLoopback(host); got != want {
			t.Errorf("isLoopback(%q) = %t, want %t", host, got, want)
		}
	}
}

func TestAMessageBeginsWithACapital(t *testing.T) {
	for in, want := range map[string]string{
		"cannot reach vfox+": "Cannot reach vfox+", "Already one": "Already one", "": "", "é ok": "É ok", "1 is a number": "1 is a number",
	} {
		if got := sentence(in); got != want {
			t.Errorf("sentence(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWhatAnInstanceAnswersToAPairingLinkThatIsNotAPairingIsSaid(t *testing.T) {
	cases := map[string]struct {
		answer http.HandlerFunc
		want   string
		is     error
	}{
		"a refusal": {func(w http.ResponseWriter, r *http.Request) { http.Error(w, "no", http.StatusForbidden) }, "", errRefused},
		"a page that is not a redirect": {func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("hello")) },
			"is not what an instance of vfox+ does", nil},
		"a redirect without a session": {func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/app", http.StatusSeeOther) },
			"gave the gateway no session", nil},
		"a redirect with another cookie": {func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: "other", Value: "x", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			http.Redirect(w, r, "/app", http.StatusSeeOther)
		}, "gave the gateway no session", nil},
		"a redirect with an empty session": {func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{Name: instanceCookie, Value: "", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			http.Redirect(w, r, "/app", http.StatusSeeOther)
		}, "gave the gateway no session", nil},
	}
	for name, c := range cases {
		fake := httptest.NewServer(c.answer)
		up := newUpstream(mustURL(t, fake.URL), nil, nil)
		session, err := up.pairWith(context.Background(), "abcdefgh12345678")
		fake.Close()
		switch {
		case session != "":
			t.Errorf("%s: the gateway took a session: %q", name, session)
		case c.is != nil && !errors.Is(err, c.is):
			t.Errorf("%s: %v, want %v", name, err, c.is)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%s: %v, want it to say %q", name, err, c.want)
		}
	}
}

func TestAPairingLinkIsNotFollowedWhereTheInstanceSendsIt(t *testing.T) {
	// The answer of the instance is the redirect, and the cookie it sets is what is wanted; following it
	// would go where the instance says, which is of no use and, for an address that is not an instance, a way to
	// make the gateway ask for pages.
	visited := make(chan string, 4)
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { visited <- r.URL.Path }))
	defer target.Close()
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{Name: instanceCookie, Value: "session", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		http.Redirect(w, r, target.URL+"/elsewhere", http.StatusSeeOther)
	}))
	defer fake.Close()
	session, err := newUpstream(mustURL(t, fake.URL), nil, nil).pairWith(context.Background(), "abcdefgh12345678")
	if err != nil || session != "session" {
		t.Fatalf("pairWith = %q, %v", session, err)
	}
	select {
	case path := <-visited:
		t.Errorf("the gateway followed the redirect to %s", path)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestAnInstanceThatDoesNotAnswerInTimeIsGivenUpOn(t *testing.T) {
	held := make(chan struct{})
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-held }))
	defer fake.Close()
	defer close(held)
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := newUpstream(mustURL(t, fake.URL), nil, nil).pairWith(ctx, "abcdefgh12345678"); err == nil || !strings.Contains(err.Error(), "cannot reach vfox+") {
		t.Errorf("pairWith = %v, want the instance to be unreachable", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Error("the gateway waited for an instance that does not answer")
	}
}

func TestTheTokensThatExpiredAreSweptWhenANewOneIsMade(t *testing.T) {
	moment := newClock()
	b := newBrowsers(30*time.Second, time.Hour, moment.now)
	for range 5 {
		b.newToken()
	}
	if len(b.tokens) != 5 {
		t.Fatalf("%d tokens, want 5", len(b.tokens))
	}
	moment.advance(time.Minute)
	fresh := b.newToken()
	if len(b.tokens) != 1 || !b.takeToken(fresh) {
		t.Errorf("%d tokens are held after the sweep, want only the new one", len(b.tokens))
	}
	if b.takeToken(fresh) {
		t.Error("a token was taken twice")
	}
	if b.takeToken("") || b.takeToken("never-made") {
		t.Error("a token that was never made was taken")
	}
}

func TestATokenIsLongAndDifferentEachTime(t *testing.T) {
	b := newBrowsers(time.Minute, time.Minute, nil)
	seen := map[string]bool{}
	for range 200 {
		token := b.newToken()
		if len(token) < 20 || seen[token] || b.newSession() == token {
			t.Fatalf("token %q is short or was made before", token)
		}
		seen[token] = true
	}
}

func TestWhatTheGatewayRemembersSurvivesAndABadDirectoryIsNoFailureOfTheGateway(t *testing.T) {
	dir := t.TempDir()
	want := saved{Upstream: "https://host:8369", Certificate: []byte{0x30, 0x82, 0x01}, Session: "secret", RenewedAt: time.Unix(1_700_000_000, 0).UTC()}
	if err := saveState(dir, want); err != nil {
		t.Fatal(err)
	}
	if got := loadState(dir); !reflect.DeepEqual(got, want) {
		t.Errorf("loadState = %+v, want %+v", got, want)
	}
	if got := loadState(filepath.Join(dir, "none")); !reflect.DeepEqual(got, saved{}) {
		t.Errorf("a directory that is not there: %+v", got)
	}

	// A directory that cannot be written is a gateway that does not remember, and works.
	file := filepath.Join(dir, "a-file")
	if err := os.WriteFile(file, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := saveState(filepath.Join(file, "inside"), want); err == nil {
		t.Error("a state was saved in a place that is a file")
	}
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{Dir: filepath.Join(file, "inside")})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	if status, _ := b.get("/api/echo"); status != http.StatusOK {
		t.Errorf("a gateway that cannot remember does not serve: %d", status)
	}
}
