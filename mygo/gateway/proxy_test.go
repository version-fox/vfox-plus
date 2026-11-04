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
	"bufio"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTheInterfaceOfTheInstanceIsServedThroughTheGateway(t *testing.T) {
	instance := startInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	direct := instance.direct()

	// The page is the instance's own, so that it matches the routes it calls.
	status, page := b.get("/app")
	_, want := direct.get("/app")
	if status != http.StatusOK || page != want || !strings.Contains(page, "EventSource") {
		t.Errorf("/app through the gateway: %d, %d bytes, the instance's page is %d bytes", status, len(page), len(want))
	}
	response := b.send(http.MethodGet, "/app", nil, nil)
	if response.Header.Get("Cache-Control") != "no-store" || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/html") {
		t.Errorf("the headers of the instance are the page's: %v", response.Header)
	}

	// What it reads is the instance's, too.
	status, inventory := b.get("/api/inventory")
	_, wantInventory := direct.get("/api/inventory")
	if status != http.StatusOK || inventory != wantInventory || !strings.Contains(inventory, "nodejs") {
		t.Errorf("/api/inventory through the gateway: %d %.200s, the instance says %.200s", status, inventory, wantInventory)
	}
	if status, body := b.get("/api/sdks/nodejs/info"); status != http.StatusOK || !strings.Contains(body, "nodejs") {
		t.Errorf("/api/sdks/nodejs/info = %d %.200s", status, body)
	}
	if status, _ := b.get("/api/nothing"); status != http.StatusNotFound {
		t.Errorf("a route that the instance has not answers %d, want its 404", status)
	}
}

func TestWhatTheInstanceSetsInTheCookiesDoesNotReachTheBrowser(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())

	response := b.send(http.MethodGet, "/app", nil, nil)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("/app answers %d", response.StatusCode)
	}
	if cookies := response.Header.Values("Set-Cookie"); len(cookies) != 0 {
		t.Errorf("the instance set cookies for the browser: %v", cookies)
	}
}

func TestWhatBelongsToTheBrowserAndTheGatewayDoesNotReachTheInstance(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	session := g.current().session

	response := b.send(http.MethodPost, "/api/echo?x=1&y=%20z", strings.NewReader("hello"), map[string]string{
		"Cookie":              cookieName + "=" + b.session + "; " + instanceCookie + "=forged; other=1",
		"Origin":              b.base,
		"Referer":             b.base + "/app",
		"Authorization":       "Bearer secret",
		"X-Forwarded-For":     "203.0.113.9",
		"X-Forwarded-Host":    "gateway.example",
		"X-Forwarded-Port":    "443",
		"X-Forwarded-Prefix":  "/gateway",
		"Forwarded":           "for=203.0.113.9",
		"X-Real-Ip":           "203.0.113.9",
		"Sec-Fetch-Site":      "same-origin",
		"Sec-Fetch-Mode":      "cors",
		"Content-Type":        "text/plain",
		"X-Something-Of-Mine": "kept",
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("answers %d", response.StatusCode)
	}
	requests := instance.requests("/api/echo")
	if len(requests) != 1 {
		t.Fatalf("the instance saw %d requests of /api/echo, want 1", len(requests))
	}
	seen := requests[0]
	if seen.Method != http.MethodPost || seen.Body != "hello" {
		t.Errorf("the request changed on the way: %s %q", seen.Method, seen.Body)
	}
	if got := seen.Header.Get("Cookie"); got != instanceCookie+"="+session {
		t.Errorf("Cookie = %q, want only the session of the gateway at the instance", got)
	}
	for _, name := range []string{"Origin", "Referer", "Authorization", "X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Port", "X-Forwarded-Prefix", "Forwarded", "X-Real-Ip", "Sec-Fetch-Site", "Sec-Fetch-Mode"} {
		if value := seen.Header.Get(name); value != "" {
			t.Errorf("%s = %q reached the instance", name, value)
		}
	}
	if seen.Header.Get("X-Something-Of-Mine") != "kept" || seen.Header.Get("Content-Type") != "text/plain" {
		t.Errorf("a header that has nothing to do with the browser's identity was lost: %v", seen.Header)
	}
	if seen.Host != strings.TrimPrefix(instance.server.URL, "http://") {
		t.Errorf("Host = %q, want that of the instance", seen.Host)
	}
}

func TestTheEventsOfTheInstanceReachTheBrowserAsTheyHappen(t *testing.T) {
	instance := startInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())

	response := b.send(http.MethodGet, "/api/events", nil, nil)
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("/api/events answers %d %q", response.StatusCode, response.Header.Get("Content-Type"))
	}
	events := make(chan string, 8)
	go func() {
		scanner := bufio.NewScanner(response.Body)
		for scanner.Scan() {
			if event, found := strings.CutPrefix(scanner.Text(), "event: "); found {
				events <- event
			}
		}
		close(events)
	}()
	wait := func(want string) {
		t.Helper()
		select {
		case got, ok := <-events:
			if !ok || got != want {
				t.Fatalf("got the event %q (open: %t), want %q", got, ok, want)
			}
		case <-time.After(5 * time.Second):
			t.Fatalf("no event %q reached the browser: the gateway keeps the stream in its buffer", want)
		}
	}
	// The first comes with the connection and not with the next heartbeat, which is 25 seconds away.
	wait("hello")
	instance.model.Installed.NotifyListeners()
	wait("inventory")
}

func TestAChangeThatTheBrowserMakesIsAChangeOfTheInstance(t *testing.T) {
	instance := startInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	direct := instance.direct()

	// The page of the gateway sends the origin of the gateway. The instance would refuse it, as it names
	// another host than the instance's: that is what the cross-origin protection is for, and the gateway
	// is not another site.
	refused := direct.send(http.MethodDelete, "/api/tasks/finished", nil, map[string]string{"Origin": b.base, "Sec-Fetch-Site": "cross-site"})
	if refused.StatusCode != http.StatusForbidden {
		t.Fatalf("the instance answers a request from another origin with %d, want 403", refused.StatusCode)
	}
	response := b.send(http.MethodDelete, "/api/tasks/finished", nil, map[string]string{"Origin": b.base, "Sec-Fetch-Site": "same-origin"})
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || !strings.Contains(string(body), `"cleared":true`) {
		t.Errorf("through the gateway: %d %s", response.StatusCode, body)
	}
}

func TestARequestFromAnotherSiteIsRefusedBeforeItReachesTheInstance(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())

	for name, headers := range map[string]map[string]string{
		"another origin":                             {"Origin": "https://evil.example"},
		"a request of another site":                  {"Sec-Fetch-Site": "cross-site"},
		"a request of the same site, another origin": {"Sec-Fetch-Site": "same-site"},
	} {
		response := b.send(http.MethodPost, "/api/echo", strings.NewReader("x"), headers)
		if response.StatusCode != http.StatusForbidden {
			t.Errorf("%s: answers %d, want 403", name, response.StatusCode)
		}
	}
	if requests := instance.requests("/api/echo"); len(requests) != 0 {
		t.Errorf("the instance got %d requests that the gateway should have refused", len(requests))
	}
	// A request that no browser sent, as curl's is, has the session as its credential.
	if response := b.send(http.MethodPost, "/api/echo", strings.NewReader("x"), nil); response.StatusCode != http.StatusOK {
		t.Errorf("a request without origin answers %d, want 200", response.StatusCode)
	}
}

func TestAnUploadIsForwardedWhole(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())

	payload := strings.Repeat("0123456789abcdef", 16<<10) // 256 KiB
	response := b.send(http.MethodPost, "/api/echo", strings.NewReader(payload), map[string]string{"Origin": b.base})
	var answer struct{ Body string }
	if err := json.NewDecoder(response.Body).Decode(&answer); err != nil || answer.Body != payload {
		t.Errorf("the upload arrived as %d bytes (%v), want %d", len(answer.Body), err, len(payload))
	}
}

func TestTheGatewayLearnsThatTheInstanceHasEndedItsSession(t *testing.T) {
	instance := startFakeInstance(t)
	dir := t.TempDir()
	g, server := serveGateway(t, Options{Dir: dir})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	if status, _ := b.get("/api/echo"); status != http.StatusOK {
		t.Fatalf("before: %d", status)
	}

	instance.endSessions()
	// The web interface goes to the page of the gateway on a 401, and the instance's answer is that.
	if status, _ := b.get("/api/echo"); status != http.StatusUnauthorized {
		t.Fatalf("the answer of the instance is passed on: %d, want 401", status)
	}
	if status := g.Status(); status.Connected || status.Address == "" {
		t.Errorf("status = %+v, want the instance kept and the session gone", status)
	}
	if remembered := loadState(dir); remembered.Session != "" || remembered.Upstream != instance.server.URL {
		t.Errorf("remembered = %+v, want the address and no session", remembered)
	}
	if response := b.send(http.MethodGet, "/app", nil, nil); response.StatusCode != http.StatusFound || response.Header.Get("Location") != "/" {
		t.Errorf("/app = %d to %q, want the page of the gateway", response.StatusCode, response.Header.Get("Location"))
	}
	status, body := b.get("/api/echo")
	if status != http.StatusUnauthorized || !strings.Contains(body, "not paired with a vfox+") {
		t.Errorf("the API of a gateway that is not paired answers %d %s, want 401 that says so", status, body)
	}
	if requests := instance.requests("/api/echo"); len(requests) != 2 {
		t.Errorf("the instance was asked %d times, want 2: nothing is forwarded once the session is known to be gone", len(requests))
	}
	_, page := b.get("/")
	if !strings.Contains(page, "has ended") || !strings.Contains(page, instance.server.URL) {
		t.Errorf("the page does not say that the session ended: %.400s", page)
	}

	// Pairing it again with a new link is all it takes.
	pairGateway(t, b, instance.link())
	if status, _ := b.get("/api/echo"); status != http.StatusOK {
		t.Errorf("after pairing again: %d", status)
	}
}

func TestAnInstanceThatIsDownIsSaidSoInTheFormTheInterfaceReads(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	instance.server.Close()

	response := b.send(http.MethodGet, "/api/echo", nil, nil)
	var failure struct{ Error string }
	if err := json.NewDecoder(response.Body).Decode(&failure); err != nil || response.StatusCode != http.StatusBadGateway ||
		!strings.HasPrefix(failure.Error, "Cannot reach vfox+ at ") {
		t.Errorf("the API answers %d %+v (%v), want 502 with the error in the form the interface reads", response.StatusCode, failure, err)
	}
	if !strings.HasPrefix(response.Header.Get("Content-Type"), "application/json") {
		t.Errorf("Content-Type = %q", response.Header.Get("Content-Type"))
	}
	status, page := b.get("/app")
	if status != http.StatusBadGateway || !strings.Contains(page, "cannot be reached") {
		t.Errorf("the page answers %d: %.300s", status, page)
	}
	// An instance that is down for a moment has not ended the session.
	if !g.Status().Connected {
		t.Error("the gateway forgot its session because the instance could not be reached")
	}
}

func TestTheManifestIsServedToABrowserThatIsNotPaired(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	stranger := unpaired(t, server)
	if status, _ := stranger.get("/manifest.webmanifest"); status != http.StatusNotFound {
		t.Errorf("a gateway that is paired with nothing has no manifest, got %d", status)
	}
	pairGateway(t, pairedBrowser(t, g, server), instance.link())

	// A browser may fetch it without the cookie, to offer the interface as an application.
	response := stranger.send(http.MethodGet, "/manifest.webmanifest", nil, nil)
	body, _ := io.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Type") != "application/manifest+json" || !strings.Contains(string(body), `"vfox+"`) {
		t.Errorf("the manifest: %d %q %s", response.StatusCode, response.Header.Get("Content-Type"), body)
	}
}

// The manifest is forwarded for a browser that has no cookie to send, and for a gateway that holds no session.
// Then there is no session of the gateway to put in the place of the cookies of the browser, and the cookies of
// the browser are not sent to the instance as they are.
func TestTheCookiesOfTheBrowserDoNotReachTheInstanceWhenTheGatewayHasNoSession(t *testing.T) {
	instance := startFakeInstance(t)
	g, server := serveGateway(t, Options{})
	b := pairedBrowser(t, g, server)
	pairGateway(t, b, instance.link())
	instance.endSessions()
	b.get("/api/echo") // The gateway learns that the session ended.
	if g.Status().Connected {
		t.Fatal("the gateway still holds a session")
	}

	response := b.send(http.MethodGet, "/manifest.webmanifest", nil, map[string]string{
		"Cookie": cookieName + "=" + b.session + "; " + instanceCookie + "=forged",
	})
	if response.StatusCode != http.StatusOK {
		t.Fatalf("the manifest answers %d", response.StatusCode)
	}
	requests := instance.requests("/manifest.webmanifest")
	if len(requests) != 1 {
		t.Fatalf("the instance saw %d requests of the manifest, want 1", len(requests))
	}
	if cookie := requests[0].Header.Get("Cookie"); cookie != "" {
		t.Errorf("the cookies of the browser reached the instance: %q", cookie)
	}
}

func TestAFileThatHoldsTheSessionIsNotLeftWorldReadable(t *testing.T) {
	instance := startFakeInstance(t)
	dir := t.TempDir()
	g, server := serveGateway(t, Options{Dir: dir})
	pairGateway(t, pairedBrowser(t, g, server), instance.link())
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		info, _ := os.Stat(filepath.Join(dir, entry.Name()))
		if info.Mode().Perm()&0o077 != 0 {
			t.Errorf("%s is readable by others: %v", entry.Name(), info.Mode().Perm())
		}
	}
}
