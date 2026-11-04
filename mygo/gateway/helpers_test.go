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
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// TestMain keeps the log of the gateway, which says what it does on a console, out of the output of the tests.
func TestMain(m *testing.M) {
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

// clock is a time that moves when a test says so: the links and the sessions of the browsers go by it, so that
// no test waits for five minutes or a day, or hopes that a request is quicker than a tenth of a second.
type clock struct {
	mu sync.Mutex
	at time.Time
}

func newClock() *clock { return &clock{at: time.Unix(1_700_000_000, 0)} }

func (c *clock) now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.at
}

func (c *clock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = c.at.Add(d)
}

// testModel is the application state of the captured output of vfox: what an instance serves.
func testModel(t *testing.T) *state.AppModel {
	t.Helper()
	model := state.NewAppModel(state.AppModelOptions{
		Runner:        testsupport.FixtureRunner(),
		SettingsStore: settings.NewMemorySettingsStore(settings.DefaultSettings()),
		Environment:   map[string]string{"HOME": "/home/u"},
		Locator: vfox.NewVfoxLocator(map[string]string{},
			func(path string) bool { return path == "/usr/bin/vfox" }),
	})
	t.Cleanup(model.Dispose)
	model.Start()
	model.Vfox.EnsureChecked()
	deadline := time.Now().Add(5 * time.Second)
	for !model.Installed.HasValue() {
		if time.Now().After(deadline) {
			t.Fatal("timed out waiting for the installed SDKs")
		}
		time.Sleep(2 * time.Millisecond)
	}
	return model
}

// instance is a real server of the application, the way the application serves its web interface: on the
// loopback address in plain HTTP, or in HTTPS with the certificate that the application makes for itself.
type instance struct {
	t      *testing.T
	server *pair.Server
	model  *state.AppModel
	// fingerprint is that of its certificate, empty for plain HTTP.
	fingerprint string
	certFile    string
}

// startInstance serves the application state on a loopback address, in plain HTTP.
func startInstance(t *testing.T) *instance { return startInstanceFor(t, 0) }

// startInstanceFor is startInstance with sessions that last as long as sessionTTL (0: as long as the server).
func startInstanceFor(t *testing.T, sessionTTL time.Duration) *instance {
	t.Helper()
	model := testModel(t)
	server, err := pair.NewServer(pair.Options{Model: model, SessionTTL: sessionTTL})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return &instance{t: t, server: server, model: model}
}

// startSecureInstance serves it in HTTPS, with a self-signed certificate made in dir, on port (0: any).
func startSecureInstance(t *testing.T, dir string, port int) *instance {
	t.Helper()
	certFile, keyFile, fingerprint, err := pair.EnsureServerCert(dir, []string{"127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	model := testModel(t)
	server, err := pair.NewServer(pair.Options{
		Bind: "127.0.0.1", AllowNonLoopback: true, Port: port,
		Model: model, TLSCertFile: certFile, TLSKeyFile: keyFile, Fingerprint: fingerprint,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	return &instance{t: t, server: server, model: model, fingerprint: fingerprint, certFile: certFile}
}

// link makes a pairing link, as the settings of the application do.
func (i *instance) link() string {
	i.t.Helper()
	_, link, err := i.server.NewToken()
	if err != nil {
		i.t.Fatal(err)
	}
	return link
}

// fakeInstance is a server that answers like an instance and notes what it was asked, for what the real
// one does not show: the headers that arrive.
type fakeInstance struct {
	t      *testing.T
	server *httptest.Server

	mu       sync.Mutex
	tokens   map[string]bool
	sessions map[string]bool
	seen     []seenRequest
	counter  int
	// held, when it is set, is what /api/token waits for before it answers.
	held chan struct{}
}

// seenRequest is a request that the fake instance got.
type seenRequest struct {
	Method string
	Path   string
	Host   string
	Header http.Header
	Body   string
}

func startFakeInstance(t *testing.T) *fakeInstance {
	t.Helper()
	f := &fakeInstance{t: t, tokens: map[string]bool{}, sessions: map[string]bool{}}
	f.server = httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeInstance) next(prefix string) string {
	f.counter++
	return fmt.Sprintf("%s%08d", prefix, f.counter)
}

// link makes a pairing link of the fake instance.
func (f *fakeInstance) link() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	token := f.next("token")
	f.tokens[token] = true
	return f.server.URL + pairingPath + token
}

// holdTokens makes /api/token wait until the function that it returns is called.
func (f *fakeInstance) holdTokens() (release func()) {
	f.mu.Lock()
	defer f.mu.Unlock()
	held := make(chan struct{})
	f.held = held
	var once sync.Once
	return func() { once.Do(func() { close(held) }) }
}

// endSessions makes the instance forget every session, as a day or a restart does.
func (f *fakeInstance) endSessions() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sessions = map[string]bool{}
}

// requests are what it was asked at a path.
func (f *fakeInstance) requests(path string) []seenRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	var found []seenRequest
	for _, request := range f.seen {
		if request.Path == path {
			found = append(found, request)
		}
	}
	return found
}

func (f *fakeInstance) serve(writer http.ResponseWriter, request *http.Request) {
	body, _ := io.ReadAll(io.LimitReader(request.Body, 1<<20))
	f.mu.Lock()
	f.seen = append(f.seen, seenRequest{Method: request.Method, Path: request.URL.Path, Host: request.Host, Header: request.Header.Clone(), Body: string(body)})
	cookie, _ := request.Cookie(instanceCookie)
	paired := cookie != nil && f.sessions[cookie.Value]
	f.mu.Unlock()

	switch {
	case strings.HasPrefix(request.URL.Path, pairingPath):
		token := strings.TrimPrefix(request.URL.Path, pairingPath)
		f.mu.Lock()
		ok := f.tokens[token]
		delete(f.tokens, token)
		session := ""
		if ok {
			session = f.next("session")
			f.sessions[session] = true
		}
		f.mu.Unlock()
		if !ok {
			http.Error(writer, "invalid", http.StatusForbidden)
			return
		}
		http.SetCookie(writer, &http.Cookie{Name: instanceCookie, Value: session, Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		http.Redirect(writer, request, "/app", http.StatusSeeOther)
	case request.URL.Path == "/manifest.webmanifest":
		writer.Header().Set("Content-Type", "application/manifest+json")
		_, _ = io.WriteString(writer, `{"name":"vfox+"}`)
	case !paired:
		http.Error(writer, "Not paired. Open a pairing link first.", http.StatusUnauthorized)
	case request.URL.Path == "/api/token":
		f.mu.Lock()
		held := f.held
		f.mu.Unlock()
		if held != nil {
			<-held
		}
		f.mu.Lock()
		token := f.next("token")
		f.tokens[token] = true
		f.mu.Unlock()
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]string{"url": "http://" + request.Host + pairingPath + token})
	case request.URL.Path == "/app":
		// The instance sets a cookie of its own with the page, which the browser must not get.
		http.SetCookie(writer, &http.Cookie{Name: "leak", Value: "1", Path: "/", Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		writer.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(writer, "<html><body>the page of the instance</body></html>")
	case request.URL.Path == "/api/echo":
		writer.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(writer).Encode(map[string]string{"method": request.Method, "body": string(body)})
	default:
		http.NotFound(writer, request)
	}
}

// browser is a person's browser at the gateway: it sends the cookie that pairing gave it, and follows no
// redirect, so that the test sees where it is sent.
type browser struct {
	t       *testing.T
	base    string
	session string
	// cookie is the name of the cookie that carries the session: the gateway's when empty.
	cookie string
}

var noRedirect = &http.Client{
	Timeout:       30 * time.Second,
	CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
}

// serveGateway serves the handler of a gateway in plain HTTP, which the cookie is sent over by hand: it is
// Secure, and clients do not send that over HTTP.
func serveGateway(t *testing.T, options Options) (*Gateway, *httptest.Server) {
	t.Helper()
	if options.Dir == "" {
		options.Dir = t.TempDir()
	}
	g, err := New(options)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(g.Handler())
	t.Cleanup(server.Close)
	return g, server
}

// unpaired is a browser that is not paired with the gateway.
func unpaired(t *testing.T, server *httptest.Server) *browser {
	return &browser{t: t, base: server.URL}
}

// pairedBrowser is a browser that was paired with the link that the gateway prints.
func pairedBrowser(t *testing.T, g *Gateway, server *httptest.Server) *browser {
	t.Helper()
	b := unpaired(t, server)
	response := b.send(http.MethodGet, pairingPath+g.NewToken(), nil, nil)
	if response.StatusCode != http.StatusSeeOther {
		t.Fatalf("pairing a browser answered %d, want 303", response.StatusCode)
	}
	for _, cookie := range response.Cookies() {
		if cookie.Name == cookieName {
			b.session = cookie.Value
		}
	}
	if b.session == "" {
		t.Fatal("pairing a browser gave it no session")
	}
	return b
}

// send sends a request and returns the answer with its body still to read.
func (b *browser) send(method, path string, body io.Reader, headers map[string]string) *http.Response {
	b.t.Helper()
	request, err := http.NewRequest(method, b.base+path, body)
	if err != nil {
		b.t.Fatal(err)
	}
	if b.session != "" {
		name := b.cookie
		if name == "" {
			name = cookieName
		}
		request.Header.Add("Cookie", name+"="+b.session)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := noRedirect.Do(request)
	if err != nil {
		b.t.Fatal(err)
	}
	b.t.Cleanup(func() { _ = response.Body.Close() })
	return response
}

// get asks for a path and returns the status and the body.
func (b *browser) get(path string) (int, string) {
	b.t.Helper()
	response := b.send(http.MethodGet, path, nil, nil)
	body, err := io.ReadAll(response.Body)
	if err != nil {
		b.t.Fatal(err)
	}
	return response.StatusCode, string(body)
}

// post sends a form as the page of the gateway does, from its own origin.
func (b *browser) post(path string, form url.Values) (*http.Response, string) {
	b.t.Helper()
	response := b.send(http.MethodPost, path, strings.NewReader(form.Encode()), map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
		"Origin":       b.base,
	})
	body, err := io.ReadAll(response.Body)
	if err != nil {
		b.t.Fatal(err)
	}
	return response, string(body)
}

// connect pairs the gateway with an instance by its link, as the person does on the page.
func (b *browser) connect(link string) (*http.Response, string) {
	b.t.Helper()
	return b.post("/gateway/connect", url.Values{"link": {link}})
}

// pairGateway pairs the gateway with the instance and fails the test unless that worked.
func pairGateway(t *testing.T, b *browser, link string) {
	t.Helper()
	response, body := b.connect(link)
	if response.StatusCode != http.StatusSeeOther || response.Header.Get("Location") != "/app" {
		t.Fatalf("pairing the gateway answered %d (%s): %s", response.StatusCode, response.Header.Get("Location"), body)
	}
}

// direct is a browser that is paired with the instance itself, without the gateway: what the gateway shows
// is compared with what the instance shows it.
func (i *instance) direct() *browser {
	i.t.Helper()
	b := &browser{t: i.t, base: i.server.BaseURL()}
	response, err := noRedirect.Get(i.link())
	if err != nil {
		i.t.Fatal(err)
	}
	_ = response.Body.Close()
	for _, cookie := range response.Cookies() {
		if cookie.Name == instanceCookie {
			b.session = cookie.Value
			b.cookie = instanceCookie
		}
	}
	if b.session == "" {
		i.t.Fatal("the instance gave no session")
	}
	return b
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
