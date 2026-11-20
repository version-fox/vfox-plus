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

package remote

// The tests run the client against the real server of mygo/pair, over TLS on the
// loopback interface, with the fixtures of vfox. The server is the one of the
// desktop application; the client never imports it.

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// fakeControl stands in for the desktop application: it makes the pairing links
// for other devices and counts the quits that the server asks for.
type fakeControl struct {
	quits atomic.Int32
}

func (f *fakeControl) AddListener(state.Listener) state.Unsubscribe {
	return func() {}
}

func (f *fakeControl) Status() pair.Status {
	return pair.Status{
		Remote:      true,
		Port:        8369,
		Addresses:   []string{"192.168.1.10"},
		Fingerprint: "AB:CD",
		SelfSigned:  true,
	}
}

func (f *fakeControl) RemoteLink(host string) (string, error) {
	return fmt.Sprintf("https://%s:8369/auth/connect/%s", host, strings.Repeat("a", 32)), nil
}

func (f *fakeControl) Quit() {
	f.quits.Add(1)
}

// testServer is a desktop server on the loopback interface.
type testServer struct {
	server *pair.Server
	base   *url.URL
	// fingerprint and certificate are those of the certificate that the server presents:
	// what the user compares, and what the client pins.
	fingerprint string
	certificate []byte
	control     *fakeControl
}

// startServer starts a server with the fixtures of vfox and a certificate of its
// own. The runner answers the calls that the contract tests make beyond the
// fixtures, so that every kind of answer has content: a config change; an install
// of a version that does not exist, which fails and so keeps its full log; two
// installed Java versions, which the profile of Java takes apart; a finding of
// the doctor; and an alias of nodejs.
func startServer(t *testing.T) *testServer {
	t.Helper()
	certFile, keyFile, fingerprint, err := pair.EnsureServerCert(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	pairOfFiles, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	installed := testsupport.FixtureOutput("list", 0)
	installed.Stdout += "\n├─┬java\n│ ├──17.0.9-tem\n│ └──21.0.5-tem\n"
	doctor := testsupport.FixtureOutput("doctor-json", 0)
	doctor.Stdout = `{"ok": false, "findings": [{"id": "sdk.missing", "severity": "warning", "sdk": "nodejs", ` +
		`"version": "22.21.0", "message": "a version is missing", "hints": ["install it"], ` +
		`"fix": ["vfox install nodejs@22.21.0"], "ignored": false}]}`
	runner := testsupport.FixtureRunner().
		On("list", installed).
		On("doctor --json", doctor).
		On("alias list nodejs", core.ProcessOutput{Stdout: "nodejs@lts -> nodejs@22.21.0\n"}).
		On("config proxy.url http://proxy.example:8080", core.ProcessOutput{}).
		On("config --unset proxy.url", core.ProcessOutput{}).
		On("install --yes nodejs@99.0.0", core.ProcessOutput{ExitCode: 1, Stderr: "no such version"})
	model := newModel(t, runner, t.TempDir())
	control := &fakeControl{}
	server, err := pair.NewServer(pair.Options{
		Model:       model,
		TLSCertFile: certFile,
		TLSKeyFile:  keyFile,
		Control:     control,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = server.Close() })
	base, err := url.Parse(server.BaseURL())
	if err != nil {
		t.Fatal(err)
	}
	return &testServer{server: server, base: base, fingerprint: fingerprint, certificate: pairOfFiles.Certificate[0], control: control}
}

// newModel returns the application state over runner, once the installed SDKs
// have loaded.
func newModel(t *testing.T, runner *testsupport.FakeProcessRunner, logDir string) *state.AppModel {
	t.Helper()
	model := state.NewAppModel(state.AppModelOptions{
		Runner:        runner,
		SettingsStore: settings.NewMemorySettingsStore(settings.DefaultSettings()),
		Environment:   map[string]string{"HOME": "/home/u"},
		Locator: vfox.NewVfoxLocator(map[string]string{},
			func(path string) bool { return path == "/usr/bin/vfox" }),
		TaskLogDir: logDir,
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

// pairRaw issues a pairing link and follows it, and returns the session that it
// gives. It reads the session cookie whatever the status of the answer, so the
// tests of the other calls do not depend on how the answer to a link is written.
// TestPairingIssuesASession tests Pair itself.
func (s *testServer) pairRaw(t *testing.T) string {
	t.Helper()
	_, link, err := s.server.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	return s.follow(t, link)
}

// follow requests a pairing link without the client and returns the session
// cookie of the answer.
func (s *testServer) follow(t *testing.T, link string) string {
	t.Helper()
	// The certificate of the test server is the only root, as it is for the client of the
	// phone, and it is verified for a name that it has: no connection of the tests skips
	// the verification either.
	root, err := x509.ParseCertificate(s.certificate)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(root)
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{RootCAs: roots, ServerName: nameOf(root), MinVersion: tls.VersionTLS12},
		},
		CheckRedirect: noRedirects,
	}
	defer client.CloseIdleConnections()
	response, err := client.Get(link)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	_, _ = io.Copy(io.Discard, response.Body)
	for _, cookie := range response.Cookies() {
		if cookie.Name == sessionCookie {
			return cookie.Value
		}
	}
	t.Fatal("the pairing link issued no session")
	return ""
}

// client returns a client of the server with session, pinned to its
// certificate, and the recorder of its answers.
func (s *testServer) client(t *testing.T, session string) (*Client, *recorder) {
	t.Helper()
	client, err := New(s.base, s.certificate, session)
	if err != nil {
		t.Fatal(err)
	}
	rec := &recorder{next: client.client.Transport}
	client.client.Transport = rec
	return client, rec
}

// paired returns a client that is paired with the server.
func (s *testServer) paired(t *testing.T) (*Client, *recorder) {
	t.Helper()
	return s.client(t, s.pairRaw(t))
}

// recorder keeps the body of the last successful answer, so that a test can
// decode it strictly.
type recorder struct {
	next http.RoundTripper

	mu     sync.Mutex
	answer []byte
}

// RoundTrip implements http.RoundTripper.
func (r *recorder) RoundTrip(request *http.Request) (*http.Response, error) {
	response, err := r.next.RoundTrip(request)
	if err != nil || response.StatusCode < 200 || response.StatusCode > 299 {
		return response, err
	}
	body, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil {
		return nil, err
	}
	r.mu.Lock()
	r.answer = body
	r.mu.Unlock()
	response.Body = io.NopCloser(bytes.NewReader(body))
	return response, nil
}

// lastAnswer returns the body of the last successful answer.
func (r *recorder) lastAnswer(t *testing.T) []byte {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.answer == nil {
		t.Fatal("no successful answer was recorded")
	}
	return r.answer
}

// strictly decodes the last answer of the server as T, and checks that T is the
// answer: no key of the answer is unknown to the decoder, and T has exactly the
// keys of the answer at every level. Go matches JSON keys without regard to
// case, so the key check compares them as they are written.
func strictly[T any](t *testing.T, rec *recorder) T {
	t.Helper()
	raw := rec.lastAnswer(t)
	var typed T
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&typed); err != nil {
		t.Fatalf("the answer does not decode strictly as %T: %v\nanswer: %s", typed, err, raw)
	}
	encoded, err := json.Marshal(typed)
	if err != nil {
		t.Fatal(err)
	}
	var want, got any
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if where, ok := sameKeys(want, got, "$"); !ok {
		t.Fatalf("%T does not have the keys of the answer, first difference at %s\nanswer: %s\nre-encoded: %s",
			typed, where, raw, encoded)
	}
	return typed
}

// sameKeys reports whether the JSON values want and got have the same keys at
// every level and the same kinds of value. It returns the place of the first
// difference.
func sameKeys(want, got any, where string) (string, bool) {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok || len(g) != len(w) {
			return where, false
		}
		for key, value := range w {
			inner, ok := g[key]
			if !ok {
				return where + "." + key, false
			}
			if at, ok := sameKeys(value, inner, where+"."+key); !ok {
				return at, false
			}
		}
		return "", true
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return where, false
		}
		for i := range w {
			if at, ok := sameKeys(w[i], g[i], fmt.Sprintf("%s[%d]", where, i)); !ok {
				return at, false
			}
		}
		return "", true
	case nil:
		return where, got == nil
	default:
		return where, fmt.Sprintf("%T", want) == fmt.Sprintf("%T", got)
	}
}

// waitForStatus polls task until it has the status, or fails the test.
func waitForStatus(t *testing.T, client *Client, id int, status string) Task {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		task, err := client.Task(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		if task.Status == status {
			return task
		}
		if time.Now().After(deadline) {
			t.Fatalf("task %d is %q, want %q", id, task.Status, status)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
