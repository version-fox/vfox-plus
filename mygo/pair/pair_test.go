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

package pair_test

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
)

// testServer is a server on loopback with the application state of the captured
// output of vfox.
func testServer(t *testing.T) *pair.Server {
	t.Helper()
	return webServer(t, testModel(t, testsupport.FixtureRunner()))
}

// noRedirect is a client that shows the answer of the server and does not follow
// where it sends the browser: pairing answers with the cookie and no content.
var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// request sends a GET without following redirects.
func request(t *testing.T, url, cookie string) *http.Response {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if cookie != "" {
		request.AddCookie(&http.Cookie{Name: "vfox-plus-session", Value: cookie, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	response, err := noRedirect.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

func get(t *testing.T, url, cookie string) (int, string, []*http.Cookie) {
	t.Helper()
	response := request(t, url, cookie)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return response.StatusCode, string(body), response.Cookies()
}

func sessionOf(cookies []*http.Cookie) string {
	for _, cookie := range cookies {
		if cookie.Name == "vfox-plus-session" {
			return cookie.Value
		}
	}
	return ""
}

func TestPairLinkWorksOnceAndIssuesASession(t *testing.T) {
	server := testServer(t)
	_, url, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, server.BaseURL()+"/auth/connect/") {
		t.Fatalf("got %q", url)
	}
	response := request(t, url, "")
	if response.StatusCode != http.StatusNoContent || response.Header.Get("Location") != "" {
		t.Fatalf("pairing answers 204 No Content with no redirect, got %d to %q", response.StatusCode, response.Header.Get("Location"))
	}
	session := sessionOf(response.Cookies())
	if session == "" {
		t.Fatal("pairing must issue a session")
	}
	// The link is single-use.
	if status, body, _ := get(t, url, ""); status != http.StatusForbidden || strings.Contains(body, "vfox-plus pair") {
		t.Errorf("a used link must be rejected, and not by sending the user to a command, got %d: %s", status, body)
	}
	// The session unlocks the API.
	if status, body, _ := get(t, server.BaseURL()+"/api/inventory", session); status != http.StatusOK || !strings.Contains(body, "24.11.1") {
		t.Errorf("got %d: %q", status, body)
	}
	// Without the session the API refuses.
	if status, _, _ := get(t, server.BaseURL()+"/api/inventory", ""); status != http.StatusUnauthorized {
		t.Errorf("got %d", status)
	}
	// A link that was never made does not pair.
	if status, _, cookies := get(t, server.BaseURL()+"/auth/connect/0123", ""); status != http.StatusForbidden || sessionOf(cookies) != "" {
		t.Errorf("got %d", status)
	}
}

func TestPairLinkExpires(t *testing.T) {
	server, err := pair.NewServer(pair.Options{
		TokenTTL: 10 * time.Millisecond,
		Model:    testModel(t, testsupport.FixtureRunner()),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	_, url, err := server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if status, _, _ := get(t, url, ""); status != http.StatusForbidden {
		t.Errorf("an expired link must be rejected, got %d", status)
	}
}

func TestServerRefusesToServeUnsafely(t *testing.T) {
	model := testModel(t, testsupport.FixtureRunner())
	for name, options := range map[string]pair.Options{
		"without the application state":              {},
		"on an address of the network":               {Model: model, Bind: "0.0.0.0"},
		"on an address of the network, in the clear": {Model: model, Bind: "0.0.0.0", AllowNonLoopback: true},
		"with a certificate and no key":              {Model: model, TLSCertFile: "cert.pem"},
		"with a key and no certificate":              {Model: model, TLSKeyFile: "key.pem"},
		"with a certificate that is not there":       {Model: model, TLSCertFile: "cert.pem", TLSKeyFile: "key.pem"},
		"on a name instead of an address":            {Model: model, Bind: "localhost"},
	} {
		server, err := pair.NewServer(options)
		if err == nil {
			_ = server.Close()
			t.Errorf("a server %s must be refused", name)
		}
	}
	server := testServer(t)
	if addr := server.Addr().String(); !strings.HasPrefix(addr, "127.0.0.1:") {
		t.Errorf("a server listens on the loopback interface, got %q", addr)
	}
	if !strings.HasPrefix(server.BaseURL(), "http://127.0.0.1:") {
		t.Errorf("got %q", server.BaseURL())
	}
}

func TestSessionExpires(t *testing.T) {
	server, err := pair.NewServer(pair.Options{
		SessionTTL: 20 * time.Millisecond,
		Model:      testModel(t, testsupport.FixtureRunner()),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.Close() }()
	session := webSession(t, server)
	if status, _, _ := get(t, server.BaseURL()+"/api/token", session); status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	time.Sleep(50 * time.Millisecond)
	if status, _, _ := get(t, server.BaseURL()+"/api/token", session); status != http.StatusUnauthorized {
		t.Errorf("an expired session must be rejected, got %d", status)
	}
}

func TestPairedBrowserMintsLinks(t *testing.T) {
	server := testServer(t)
	session := webSession(t, server)
	// Minting needs the session.
	if status, _, _ := get(t, server.BaseURL()+"/api/token", ""); status != http.StatusUnauthorized {
		t.Fatalf("got %d", status)
	}
	status, body, _ := get(t, server.BaseURL()+"/api/token", session)
	if status != http.StatusOK {
		t.Fatalf("got %d", status)
	}
	var minted struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal([]byte(body), &minted); err != nil || minted.URL == "" {
		t.Fatalf("got %q, %v", body, err)
	}
	if !strings.HasPrefix(minted.URL, server.BaseURL()+"/auth/connect/") {
		t.Errorf("got %q", minted.URL)
	}
	// The minted link pairs once.
	if status, _, _ := get(t, minted.URL, ""); status != http.StatusNoContent {
		t.Errorf("got %d", status)
	}
	if status, _, _ := get(t, minted.URL, ""); status != http.StatusForbidden {
		t.Errorf("got %d", status)
	}
}

// The server serves no page: the interface is the gateway's, and a browser that asks this server for a page
// gets a not found, with or without a session.
func TestTheServerServesNoPage(t *testing.T) {
	server := testServer(t)
	session := webSession(t, server)
	for _, path := range []string{"/", "/app", "/manifest.webmanifest", "/nothing"} {
		for _, cookie := range []string{"", session} {
			if status, _, _ := get(t, server.BaseURL()+path, cookie); status != http.StatusNotFound {
				t.Errorf("%s answers %d, want 404", path, status)
			}
		}
	}
}
