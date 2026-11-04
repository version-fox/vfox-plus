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

package gatewaycheck

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAFingerprintIsWrittenAsOpenSSLWritesIt(t *testing.T) {
	// The SHA-256 of "abc", a published test vector.
	const want = "BA:78:16:BF:8F:01:CF:EA:41:41:40:DE:5D:AE:22:23:B0:03:61:A3:96:17:7A:9C:B4:10:FF:61:F2:00:15:AD"
	if got := fingerprintOf([]byte("abc")); got != want {
		t.Errorf("fingerprintOf = %s, want %s", got, want)
	}
}

func TestAPinnedBrowserTrustsTheCertificateWithTheFingerprintAndNoOther(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "hello") }))
	server.Config.ErrorLog = log.New(io.Discard, "", 0) // A browser that does not trust the certificate hangs up on it.
	server.StartTLS()
	defer server.Close()
	address := strings.TrimPrefix(server.URL, "https://")
	pinned := fingerprintOf(server.Certificate().Raw)

	browser, err := newPinnedSite(address, pinned)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := browser.get("/")
	if err != nil || answer.status != 200 || answer.body != "hello" {
		t.Fatalf("the certificate with the fingerprint: %+v, %v", answer, err)
	}
	other := "00:" + pinned[3:]
	if _, err := newPinnedSite(address, other); err == nil || !strings.Contains(err.Error(), "not the "+other+" that was trusted") {
		t.Errorf("a certificate with another fingerprint was trusted: %v", err)
	}
	// What is trusted is that certificate, and not any that a server presents: the browser does not take another one
	// for it, though the other is made for the same address.
	second := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "another") }))
	second.TLS = &tls.Config{Certificates: []tls.Certificate{anotherCertificate(t)}, MinVersion: tls.VersionTLS12}
	second.Config.ErrorLog = log.New(io.Discard, "", 0)
	second.StartTLS()
	defer second.Close()
	if _, err := browser.client.Get(second.URL); err == nil {
		t.Error("the browser took the certificate of another server for the one that it trusts")
	}
	if got, err := browser.client.Get(server.URL); err != nil {
		t.Errorf("the certificate that it trusts is not taken any more: %v", err)
	} else {
		_ = got.Body.Close()
	}
}

// anotherCertificate is a certificate that nobody signed but itself, for the loopback address, other than the one that
// the servers of the tests have.
func anotherCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(7), Subject: pkix.Name{CommonName: "another"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}, DNSNames: []string{"example.com"},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

func TestAPinnedBrowserKeepsTheCookiesItIsGiven(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pair" {
			http.SetCookie(w, &http.Cookie{Name: "vfox-plus-gateway", Value: "v", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		cookie, _ := r.Cookie("vfox-plus-gateway")
		if cookie == nil {
			http.Error(w, "no", http.StatusUnauthorized)
		}
	}))
	defer server.Close()
	browser, err := newPinnedSite(strings.TrimPrefix(server.URL, "https://"), fingerprintOf(server.Certificate().Raw))
	if err != nil {
		t.Fatal(err)
	}
	if answer, err := browser.get("/pair"); err != nil || answer.status != http.StatusSeeOther || answer.header.Get("Location") != "/" {
		t.Fatalf("the redirect is shown, not followed: %+v, %v", answer, err)
	}
	if answer, err := browser.get("/"); err != nil || answer.status != 200 {
		t.Errorf("the cookie of the pairing is sent back: %+v, %v", answer, err)
	}
}

func TestAnAnswerIsReadAsAPersonReadsIt(t *testing.T) {
	if got := (reply{body: "Cannot reach vfox&#43; at x"}).text(); got != "Cannot reach vfox+ at x" {
		t.Errorf("text = %q", got)
	}
}

// application is a server that answers like the application does, in plain HTTP.
func application(t *testing.T, handler http.HandlerFunc) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return server
}

func TestABrowserIsPairedWithTheApplicationByItsLink(t *testing.T) {
	server := application(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/auth/connect/good":
			http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "s3", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
			http.Redirect(w, r, "/app", http.StatusSeeOther)
		case "/auth/connect/page":
			_, _ = io.WriteString(w, "a page")
		case "/auth/connect/nocookie":
			http.Redirect(w, r, "/app", http.StatusSeeOther)
		case "/api/web":
			if r.Header.Get("Cookie") != sessionCookie+"=s3" {
				http.Error(w, "Not paired", http.StatusUnauthorized)
				return
			}
			_, _ = io.WriteString(w, `{"remote":true,"port":8369,"listening":true,"addresses":["10.1.2.3"],"fingerprint":"AA:BB","selfSigned":true}`)
		}
	})
	app, err := pairPlain(server.URL + "/auth/connect/good\n")
	if err != nil {
		t.Fatal(err)
	}
	status, err := app.webStatus()
	if err != nil || !status.Listening || status.Fingerprint != "AA:BB" || !status.SelfSigned || len(status.Addresses) != 1 || status.Addresses[0] != "10.1.2.3" {
		t.Errorf("the session is sent by hand, as the cookie is Secure: %+v, %v", status, err)
	}
	for path, want := range map[string]string{"/auth/connect/page": "want 303", "/auth/connect/nocookie": "no session"} {
		if _, err := pairPlain(server.URL + path); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want an error that says %q", path, err, want)
		}
	}
	if _, err := pairPlain("http://127.0.0.1:1/auth/connect/x"); err == nil {
		t.Error("a link of nothing paired a browser")
	}
	if _, err := pairPlain("://"); err == nil {
		t.Error("a link that is not a link paired a browser")
	}
	unpaired := &site{base: server.URL, client: app.client}
	if _, err := unpaired.webStatus(); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("the status for a browser that is not paired: %v", err)
	}
}

func TestTheFormsAndTheRequestsOfThePageAreSentFromItsOrigin(t *testing.T) {
	type seen struct {
		method, path, contentType, origin, cookie, body string
	}
	got := make(chan seen, 4)
	server := application(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got <- seen{r.Method, r.URL.Path, r.Header.Get("Content-Type"), r.Header.Get("Origin"), r.Header.Get("Cookie"), string(body)}
	})
	browser := &site{base: server.URL, client: &http.Client{CheckRedirect: noRedirect}, cookie: "a=b"}

	if _, err := browser.postForm("/form", "link", "https://x/auth/connect/y?z=1", "fingerprint", "AA", "dangling"); err != nil {
		t.Fatal(err)
	}
	form := <-got
	values, _ := url.ParseQuery(form.body)
	if form.method != "POST" || form.contentType != "application/x-www-form-urlencoded" || form.origin != server.URL || form.cookie != "a=b" ||
		values.Get("link") != "https://x/auth/connect/y?z=1" || values.Get("fingerprint") != "AA" || len(values) != 2 {
		t.Errorf("form = %+v", form)
	}
	if _, err := browser.postJSON("/json", `{"a":1}`); err != nil {
		t.Fatal(err)
	}
	if json := <-got; json.contentType != "application/json" || json.origin != server.URL || json.body != `{"a":1}` {
		t.Errorf("json = %+v", json)
	}
	if _, err := browser.postJSONFrom("https://evil.example", "/json", `{}`); err != nil {
		t.Fatal(err)
	}
	if other := <-got; other.origin != "https://evil.example" {
		t.Errorf("origin = %q", other.origin)
	}
}

func TestTheLinkForOtherDevicesIsAskedForAnAddressAndMustBeHTTPS(t *testing.T) {
	answers := map[string]struct {
		status int
		body   string
	}{
		"good":   {200, `{"url":"https://10.1.2.3:8369/auth/connect/abc","svg":"<svg/>"}`},
		"plain":  {200, `{"url":"http://10.1.2.3:8369/auth/connect/abc"}`},
		"empty":  {200, `{}`},
		"broken": {200, `not json`},
		"refuse": {409, "Other devices are not allowed to connect."},
	}
	var asked string
	server := application(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		asked = string(body)
		host := ""
		for name := range answers {
			if strings.Contains(asked, `"`+name+`"`) {
				host = name
			}
		}
		w.WriteHeader(answers[host].status)
		_, _ = io.WriteString(w, answers[host].body)
	})
	app := &site{base: server.URL, client: server.Client()}
	if link, err := app.remoteLink("good"); err != nil || link != "https://10.1.2.3:8369/auth/connect/abc" || asked != `{"host":"good"}` {
		t.Errorf("remoteLink = %q, %v (asked %s)", link, err, asked)
	}
	for _, host := range []string{"plain", "empty", "broken", "refuse"} {
		if link, err := app.remoteLink(host); err == nil {
			t.Errorf("%s: a link was taken: %q", host, link)
		}
	}
	if _, err := (&site{base: "http://127.0.0.1:1", client: server.Client()}).remoteLink("x"); err == nil {
		t.Error("a link was made by nothing")
	}
}

func TestEventsAreWaitedForByTheirKind(t *testing.T) {
	release := make(chan struct{})
	server := application(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/events" && r.Header.Get("Cookie") == "" {
			http.Error(w, "no", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
		flusher := w.(http.Flusher)
		_, _ = io.WriteString(w, "event: hello\ndata: {}\n\n: a comment\n\nevent: inventory\ndata: {}\n\n")
		flusher.Flush()
		select {
		case <-release:
			_, _ = io.WriteString(w, "event: task\ndata: {}\n\n")
			flusher.Flush()
		case <-r.Context().Done():
		}
		<-r.Context().Done() // The events never end by themselves.
	})
	// A request of this browser may take 200 ms; the events are open for longer than that.
	browser := &site{base: server.URL, client: &http.Client{CheckRedirect: noRedirect, Timeout: 200 * time.Millisecond}, cookie: "a=b"}
	events, err := browser.openEvents()
	if err != nil {
		t.Fatal(err)
	}
	defer events.close()

	if err := events.wait("hello", 5*time.Second); err != nil {
		t.Errorf("hello: %v", err)
	}
	// Events of other kinds go by on the way to the one that is waited for, and the limit of time of a request
	// does not end the events: they end when the page does.
	time.Sleep(500 * time.Millisecond)
	close(release)
	if err := events.wait("task", 5*time.Second); err != nil {
		t.Errorf("task: %v", err)
	}
	if err := events.wait("never", 50*time.Millisecond); err == nil || !strings.Contains(err.Error(), `no "never" event`) {
		t.Errorf("an event that does not come: %v", err)
	}

	if _, err := (&site{base: server.URL, client: browser.client}).openEvents(); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("events for a browser that is not paired: %v", err)
	}
}

func TestEventsThatEndAreSaidToHaveEnded(t *testing.T) {
	server := application(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "event: hello\ndata: {}\n\n")
	})
	events, err := (&site{base: server.URL, client: &http.Client{}}).openEvents()
	if err != nil {
		t.Fatal(err)
	}
	defer events.close()
	if err := events.wait("task", 5*time.Second); err == nil || !strings.Contains(err.Error(), "ended") {
		t.Errorf("wait on a stream that ended: %v", err)
	}
}

func TestAnAnswerThatIsNotEventsIsNotTakenForEvents(t *testing.T) {
	server := application(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = io.WriteString(w, "<html>")
	})
	if _, err := (&site{base: server.URL, client: &http.Client{}}).openEvents(); err == nil || !strings.Contains(err.Error(), "not events") {
		t.Errorf("openEvents = %v", err)
	}
	if _, err := (&site{base: "http://127.0.0.1:1", client: &http.Client{}}).openEvents(); err == nil {
		t.Error("events were opened at nothing")
	}
}

func TestAFileIsWaitedForAndReadWhole(t *testing.T) {
	path := filepath.Join(t.TempDir(), "link.txt")
	go func() {
		time.Sleep(150 * time.Millisecond)
		_ = os.WriteFile(path, []byte("\n"), 0o600) // An empty file is not a link yet.
		time.Sleep(100 * time.Millisecond)
		_ = os.WriteFile(path, []byte("http://127.0.0.1:1/auth/connect/x\n"), 0o600)
	}()
	got, err := waitForFile(path, 10*time.Second)
	if err != nil || got != "http://127.0.0.1:1/auth/connect/x" {
		t.Errorf("waitForFile = %q, %v", got, err)
	}
	if _, err := waitForFile(filepath.Join(t.TempDir(), "never"), 100*time.Millisecond); err == nil || !strings.Contains(err.Error(), "did not appear") {
		t.Errorf("a file that never comes: %v", err)
	}
}

func TestATaskIsReadFromWhatTheApplicationAnswers(t *testing.T) {
	if id, err := taskOf(`{"id":7,"operation":"install","status":"queued","log":[]}`); err != nil || id != 7 {
		t.Errorf("taskOf = %d, %v", id, err)
	}
	for _, body := range []string{"", "{}", `{"id":0}`, "[1]", "nonsense"} {
		if id, err := taskOf(body); err == nil {
			t.Errorf("taskOf(%q) = %d, want an error", body, id)
		}
	}
	list := `[{"id":1,"status":"failed"},{"id":2,"status":"running"},{"id":3,"status":"succeeded"}]`
	for id, want := range map[int]string{1: "failed", 2: "running", 3: "succeeded"} {
		if got, ok := statusOf(list, id); !ok || got != want {
			t.Errorf("statusOf(%d) = %q, %t; want %q", id, got, ok, want)
		}
	}
	if _, ok := statusOf(list, 9); ok {
		t.Error("a task that is not in the list has a status")
	}
	if _, ok := statusOf("nonsense", 1); ok {
		t.Error("a list that is not one has a status")
	}
}
