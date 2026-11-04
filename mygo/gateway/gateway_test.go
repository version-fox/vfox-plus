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
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAGatewayNeedsADirectoryToRememberIn(t *testing.T) {
	if _, err := New(Options{}); err == nil {
		t.Fatal("a gateway without a directory was made")
	}
}

func TestABrowserThatIsNotPairedSeesTheIntroAndNothingOfTheInstance(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	pairGateway(t, pairedBrowser(t, g, server), instance.link())

	stranger := unpaired(t, server)
	status, body := stranger.get("/")
	if status != http.StatusOK || !strings.Contains(body, "This is the gateway of vfox+") || !strings.Contains(body, "这是 vfox+ 的网关") {
		t.Fatalf("the intro is in both languages, got %d: %.200s", status, body)
	}
	if strings.Contains(body, instance.server.Listener.Addr().String()) {
		t.Errorf("the intro names the instance: %s", body)
	}
	if strings.Contains(body, "/auth/connect/") {
		t.Errorf("the intro tells where the links are: %s", body)
	}
	if response := stranger.send(http.MethodGet, "/app", nil, nil); response.StatusCode != http.StatusFound || response.Header.Get("Location") != "/" {
		t.Errorf("/app sends a browser that is not paired to the intro, got %d to %q", response.StatusCode, response.Header.Get("Location"))
	}
	for _, path := range []string{"/api/inventory", "/api/events", "/api/token", "/api/settings"} {
		if status, _ := stranger.get(path); status != http.StatusUnauthorized {
			t.Errorf("%s answers a browser that is not paired with %d, want 401", path, status)
		}
	}
	for _, path := range []string{"/gateway/connect", "/gateway/trust", "/gateway/invite", "/gateway/forget"} {
		if response, _ := stranger.post(path, url.Values{"link": {instance.link()}}); response.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s answers a browser that is not paired with %d, want 401", path, response.StatusCode)
		}
	}
	if requests := instance.requests("/api/inventory"); len(requests) != 0 {
		t.Errorf("a request of a stranger reached the instance: %+v", requests)
	}
	if status, _ := stranger.get("/nothing-here"); status != http.StatusNotFound {
		t.Errorf("a path of nobody answers %d, want 404", status)
	}
}

func TestThePairingLinkOfABrowserWorksOnceAndExpires(t *testing.T) {
	moment := newClock()
	g, server := serveGateway(t, Options{clock: moment.now})
	first := unpaired(t, server)
	token := g.NewToken()

	response := first.send(http.MethodGet, pairingPath+token, nil, nil)
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/" {
		t.Fatalf("a gateway that is paired with nothing takes the browser to its page, got %d to %q", response.StatusCode, response.Header.Get("Location"))
	}
	var cookie *http.Cookie
	for _, c := range response.Cookies() {
		if c.Name == cookieName {
			cookie = c
		}
	}
	if cookie == nil || !cookie.Secure || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/" || len(cookie.Value) < 20 {
		t.Fatalf("the cookie of a paired browser is private to it: %+v", cookie)
	}
	if cookie.Name == instanceCookie {
		t.Error("the gateway uses the cookie of the instances, which a host shares between its ports")
	}
	if status, _ := unpaired(t, server).get(pairingPath + token); status != http.StatusForbidden {
		t.Errorf("a link used twice answers %d, want 403", status)
	}
	// A link works for five minutes, to the second.
	last, expired := g.NewToken(), g.NewToken()
	moment.advance(DefaultTokenTTL - time.Second)
	if status, _ := unpaired(t, server).get(pairingPath + last); status != http.StatusSeeOther {
		t.Errorf("a link a second before it expires answers %d, want 303", status)
	}
	moment.advance(2 * time.Second)
	if status, _ := unpaired(t, server).get(pairingPath + expired); status != http.StatusForbidden {
		t.Errorf("an expired link answers %d, want 403", status)
	}
	for _, link := range []string{pairingPath, pairingPath + "a/b", pairingPath + "nothing"} {
		if status, _ := unpaired(t, server).get(link); status != http.StatusForbidden {
			t.Errorf("%q answers %d, want 403", link, status)
		}
	}
}

func TestABrowserThatPairsAfterTheGatewayIsPairedGoesStraightToTheInterface(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	pairGateway(t, pairedBrowser(t, g, server), instance.link())

	response := unpaired(t, server).send(http.MethodGet, pairingPath+g.NewToken(), nil, nil)
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/app" {
		t.Errorf("got %d to %q, want 303 to /app", response.StatusCode, response.Header.Get("Location"))
	}
}

func TestTheSessionOfABrowserEndsAfterItsLifetime(t *testing.T) {
	moment := newClock()
	g, server := serveGateway(t, Options{clock: moment.now})
	b := pairedBrowser(t, g, server)
	if _, body := b.get("/"); !strings.Contains(body, "not paired with a vfox+ yet") {
		t.Fatalf("the page of a paired browser, got %.200s", body)
	}
	moment.advance(DefaultSessionTTL - time.Minute)
	if _, body := b.get("/"); !strings.Contains(body, "not paired with a vfox+ yet") {
		t.Errorf("a browser is paired for a day: a minute before the end it is not, got %.200s", body)
	}
	moment.advance(2 * time.Minute)
	if _, body := b.get("/"); !strings.Contains(body, "This is the gateway of vfox+") {
		t.Errorf("the browser whose session ended is paired still: %.200s", body)
	}
}

func TestThePageOfAPairedBrowserSaysToWhichInstanceTheGatewayIsPaired(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)

	status, body := b.get("/")
	if status != http.StatusOK || !strings.Contains(body, "not paired with a vfox+ yet") || !strings.Contains(body, `name="link"`) {
		t.Fatalf("before the pairing: %d %.300s", status, body)
	}
	if strings.Contains(body, `href="/app"`) {
		t.Error("the page offers an interface that is not there")
	}

	pairGateway(t, b, instance.link())
	_, body = b.get("/")
	if !strings.Contains(body, instance.server.URL) || !strings.Contains(body, `href="/app"`) || !strings.Contains(body, "Forget this vfox+") {
		t.Errorf("after the pairing the page says which instance, and offers it: %.600s", body)
	}
	if status := g.Status(); !status.Connected || status.Address != strings.TrimPrefix(instance.server.URL, "http://") || status.Fingerprint != "" {
		t.Errorf("status = %+v", status)
	}
}

func TestTheGatewayRemembersTheInstanceInAPrivateFile(t *testing.T) {
	instance := startFakeInstance(t)
	dir := t.TempDir()
	g, server := serveGateway(t, Options{Dir: dir})
	pairGateway(t, pairedBrowser(t, g, server), instance.link())

	path := filepath.Join(dir, stateFileName)
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("the file holds the session at the instance and is %v, want 0600", info.Mode().Perm())
	}
	var remembered saved
	content, _ := os.ReadFile(path) //nolint:gosec // The file of this test.
	if err := json.Unmarshal(content, &remembered); err != nil {
		t.Fatalf("%v: %s", err, content)
	}
	if remembered.Upstream != instance.server.URL || remembered.Session == "" || remembered.RenewedAt.IsZero() {
		t.Errorf("remembered = %+v", remembered)
	}
	leftovers, _ := filepath.Glob(filepath.Join(dir, stateFileName+".*"))
	if len(leftovers) != 0 {
		t.Errorf("writing the file left temporary files: %v", leftovers)
	}

	// A gateway that is started again is paired still, with the same session.
	again, err := New(Options{Dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	if status := again.Status(); !status.Connected || status.Address != g.Status().Address {
		t.Errorf("a gateway that was started again: %+v, want it paired with %s", status, g.Status().Address)
	}
}

func TestAFileThatCannotBeReadIsAGatewayThatIsPairedWithNothing(t *testing.T) {
	for name, content := range map[string]string{
		"not JSON":                       "{",
		"an address that is not one":     `{"upstream": "ftp://x", "session": "s"}`,
		"plain HTTP to another computer": `{"upstream": "http://192.168.1.2:8369", "session": "s"}`,
		"no host":                        `{"upstream": "https://", "session": "s"}`,
		"a certificate that is not one":  `{"upstream": "https://host:8369", "session": "s", "certificate": "AAEC"}`,
	} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, stateFileName), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		g, err := New(Options{Dir: dir})
		if err != nil {
			t.Fatal(err)
		}
		if status := g.Status(); status != (Status{}) {
			t.Errorf("%s: status = %+v, want a gateway that is paired with nothing", name, status)
		}
	}
}

func TestTheGatewayForgetsItsInstance(t *testing.T) {
	instance := startFakeInstance(t)
	dir := t.TempDir()
	g, server := serveGateway(t, Options{Dir: dir})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())

	response, _ := b.post("/gateway/forget", url.Values{})
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/" {
		t.Fatalf("got %d to %q", response.StatusCode, response.Header.Get("Location"))
	}
	if status := g.Status(); status != (Status{}) {
		t.Errorf("status = %+v, want nothing", status)
	}
	if remembered := loadState(dir); remembered.Upstream != "" || remembered.Session != "" {
		t.Errorf("the session is remembered still: %+v", remembered)
	}
	if response := b.send(http.MethodGet, "/app", nil, nil); response.StatusCode != http.StatusFound {
		t.Errorf("/app answers %d, want a redirect to the page", response.StatusCode)
	}
}

func TestABrowserThatIsPairedPairsAnotherWithALink(t *testing.T) {
	g, server := serveGateway(t, Options{})
	first := pairedBrowser(t, g, server)

	response, body := first.post("/gateway/invite", url.Values{})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("got %d: %s", response.StatusCode, body)
	}
	start := strings.Index(body, server.URL+pairingPath)
	if start < 0 {
		t.Fatalf("the page has no link: %s", body)
	}
	link := body[start:]
	link = link[:strings.IndexAny(link, "<\" ")]
	second := unpaired(t, server)
	if status, _ := second.get(strings.TrimPrefix(link, server.URL)); status != http.StatusSeeOther {
		t.Errorf("the link of the page answers %d, want 303", status)
	}
	if status, _ := second.get(strings.TrimPrefix(link, server.URL)); status != http.StatusForbidden {
		t.Errorf("it works once: the second time answers %d, want 403", status)
	}

	// The same as the servers of the application give a paired device, as JSON.
	status, body := first.get("/api/token")
	var minted struct{ URL string }
	if status != http.StatusOK || json.Unmarshal([]byte(body), &minted) != nil || !strings.HasPrefix(minted.URL, server.URL+pairingPath) {
		t.Errorf("/api/token = %d %s", status, body)
	}
	if response := first.send(http.MethodPost, "/api/token", nil, nil); response.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST /api/token answers %d, want 405", response.StatusCode)
	}
}

func TestTheFormsOfTheGatewayRefuseTheRequestsOfOtherSites(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	form := url.Values{"link": {instance.link()}}

	for name, headers := range map[string]map[string]string{
		"another origin":            {"Origin": "https://evil.example"},
		"a request of another site": {"Sec-Fetch-Site": "cross-site"},
	} {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
		response := b.send(http.MethodPost, "/gateway/connect", strings.NewReader(form.Encode()), headers)
		if response.StatusCode != http.StatusForbidden {
			t.Errorf("%s: answers %d, want 403", name, response.StatusCode)
		}
	}
	if g.Status().Connected {
		t.Error("a request of another site paired the gateway")
	}
	if requests := instance.requests(pairingPath + "token00000001"); len(requests) != 0 {
		t.Errorf("the instance was asked: %+v", requests)
	}
	// A form that is too big is refused before it is read.
	big := strings.NewReader("link=" + strings.Repeat("a", 2*maxForm))
	response := b.send(http.MethodPost, "/gateway/connect", big, map[string]string{"Content-Type": "application/x-www-form-urlencoded", "Origin": b.base})
	if response.StatusCode != http.StatusBadRequest {
		t.Errorf("a form of %d bytes answers %d, want 400", 2*maxForm, response.StatusCode)
	}
}

func TestOnlyThePostOfAFormChangesWhatTheGatewayIsPairedWith(t *testing.T) {
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	for _, path := range []string{"/gateway/connect", "/gateway/trust", "/gateway/invite", "/gateway/forget"} {
		if status, _ := b.get(path); status != http.StatusMethodNotAllowed {
			t.Errorf("GET %s answers %d, want 405", path, status)
		}
	}
}

func TestThePagesOfTheGatewayAreNotKeptNorFramed(t *testing.T) {
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	for _, browser := range []*browser{unpaired(t, server), b} {
		response := browser.send(http.MethodGet, "/", nil, nil)
		header := response.Header
		if header.Get("Cache-Control") != "no-store" || header.Get("X-Content-Type-Options") != "nosniff" ||
			!strings.Contains(header.Get("Content-Security-Policy"), "frame-ancestors 'none'") ||
			!strings.Contains(header.Get("Content-Security-Policy"), "default-src 'none'") ||
			!strings.HasPrefix(header.Get("Content-Type"), "text/html") {
			t.Errorf("headers = %v", header)
		}
	}
}
