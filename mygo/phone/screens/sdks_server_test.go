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

// The fixtures of the tests of the SDK screens: a computer that answers the remote
// API over TLS with the types of package remote, and a window that is paired with
// it. Nothing here reads the user's files or the network beyond the local server.

import (
	"context"
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/phone/store"
)

// sdkCall is one request that the fake computer got: its method, its path with its
// query, and its body.
type sdkCall struct {
	Method string
	Path   string
	Body   string
}

// sdkFailure is an answer that the fake computer gives instead of its data: the
// status, and the text of the answer.
type sdkFailure struct {
	Status  int
	Message string
}

// sdkFake is the computer of the tests. Its data are the fields below, which a test
// sets before it makes the window. Failures and a gate, which holds the answer of
// the inventory until the test opens it, make the loading and failure states.
type sdkFake struct {
	server *httptest.Server

	// Inventory is the answer of GET /api/inventory.
	Inventory remote.Inventory
	// Profiles are the answers of GET /api/sdks/<name>/profile; a name without one
	// gets 404, as a plugin with no profile does.
	Profiles map[string]remote.Profile
	// Aliases are the answers of GET /api/sdks/<name>/aliases.
	Aliases map[string][]remote.VersionAlias
	// Infos are the answers of GET /api/sdks/<name>/info.
	Infos map[string]remote.PluginInfo
	// Fail answers the request "METHOD /path" with a failure instead of its answer.
	Fail map[string]sdkFailure

	mu    sync.Mutex
	gate  chan struct{}
	calls []sdkCall
}

// newSDKFake starts a computer on a local TLS server, which the test closes.
func newSDKFake(t *testing.T) *sdkFake {
	t.Helper()
	fake := &sdkFake{
		Profiles: map[string]remote.Profile{},
		Aliases:  map[string][]remote.VersionAlias{},
		Infos:    map[string]remote.PluginInfo{},
		Fail:     map[string]sdkFailure{},
	}
	fake.server = newTLSServer(t, http.HandlerFunc(fake.serve))
	return fake
}

// Hold makes GET /api/inventory wait until the returned function is called, so
// that a test can see the loading state. The function may be called once.
func (f *sdkFake) Hold() func() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gate = make(chan struct{})
	gate := f.gate
	return func() { close(gate) }
}

// Requests returns the requests of method whose path starts with prefix, in the
// order they came.
func (f *sdkFake) Requests(method, prefix string) []sdkCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []sdkCall
	for _, call := range f.calls {
		if call.Method == method && strings.HasPrefix(call.Path, prefix) {
			out = append(out, call)
		}
	}
	return out
}

// serve answers the requests of the remote API that the screens make.
func (f *sdkFake) serve(w http.ResponseWriter, req *http.Request) {
	body, _ := io.ReadAll(req.Body)
	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path += "?" + req.URL.RawQuery
	}
	f.mu.Lock()
	f.calls = append(f.calls, sdkCall{Method: req.Method, Path: path, Body: string(body)})
	gate := f.gate
	failure, failing := f.Fail[req.Method+" "+req.URL.Path]
	f.mu.Unlock()

	if failing {
		http.Error(w, failure.Message, failure.Status)
		return
	}
	if req.Method == http.MethodGet && req.URL.Path == "/api/inventory" && gate != nil {
		<-gate
	}
	switch {
	case req.Method == http.MethodGet && req.URL.Path == "/api/inventory":
		f.reply(w, f.Inventory)
	case req.Method == http.MethodGet && req.URL.Path == "/api/tasks":
		f.reply(w, []remote.Task{})
	case req.Method == http.MethodGet && req.URL.Path == "/api/settings":
		f.reply(w, remote.AppSettings{Language: remote.LanguageSystem})
	case req.Method == http.MethodGet && req.URL.Path == "/api/events":
		// The event stream stays open until the test ends, as the desktop's does.
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		<-req.Context().Done()
	case req.Method == http.MethodPost && req.URL.Path == "/api/tasks":
		var task remote.TaskRequest
		_ = json.Unmarshal(body, &task)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(remote.Task{ID: 1, Operation: task.Operation, Sdk: task.Sdk, Status: remote.TaskQueued})
	case strings.HasPrefix(req.URL.Path, "/api/sdks/"):
		f.serveSdk(w, req)
	case req.URL.Path == "/api/config" || req.URL.Path == "/api/config/":
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, req)
	}
}

// serveSdk answers the requests about one plugin, /api/sdks/<name>/<what>.
func (f *sdkFake) serveSdk(w http.ResponseWriter, req *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(req.URL.Path, "/api/sdks/"), "/", 2)
	name, what := parts[0], ""
	if len(parts) == 2 {
		what = parts[1]
	}
	switch {
	case req.Method == http.MethodGet && what == "profile":
		profile, ok := f.Profiles[name]
		if !ok {
			http.NotFound(w, req)
			return
		}
		f.reply(w, profile)
	case req.Method == http.MethodGet && what == "aliases":
		f.reply(w, f.Aliases[name])
	case req.Method == http.MethodGet && what == "info":
		info, ok := f.Infos[name]
		if !ok {
			info = remote.PluginInfo{Name: name}
		}
		f.reply(w, info)
	case req.Method == http.MethodGet && what == "versions":
		f.reply(w, []remote.AvailableVersion{})
	case req.Method == http.MethodPut && (what == "variables" || what == "search"):
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, req)
	}
}

// reply writes value as the JSON answer of a successful request.
func (f *sdkFake) reply(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

// sdkRoot returns the window of a phone that is paired with the fake computer, in
// English. The saved computer and its session are written to a configuration
// directory of the test, which the window reads as the app does.
func sdkRoot(t *testing.T, fake *sdkFake) *Root {
	t.Helper()
	i18n.SetLanguage(i18n.English)
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	dir := filepath.Join(config, configDirName)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	record := store.Record{
		Name:        "fake",
		Base:        fake.server.URL,
		Fingerprint: pairing.FormatFingerprint(fake.server.Certificate().Raw),
		Certificate: store.EncodeCertificate(fake.server.Certificate().Raw),
	}
	if err := store.NewFileServerStore(filepath.Join(dir, serverFile)).Save(record); err != nil {
		t.Fatal(err)
	}
	if err := store.NewFileSessionStore(filepath.Join(dir, sessionFile)).Save("test-session"); err != nil {
		t.Fatal(err)
	}
	root, err := New()
	if err != nil {
		t.Fatal(err)
	}
	// The app checks the saved session in Service.Run, which keeps the event
	// stream of the computer open. The tests run it the same way, and wait until
	// the computer has accepted the phone, so that the window shows the tabs.
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go root.Service().Run(ctx)
	deadline := time.Now().Add(5 * time.Second)
	for root.Service().Status().Session != app.Paired {
		if time.Now().After(deadline) {
			t.Fatalf("the computer did not accept the phone: %v", root.Service().Status().Cause)
		}
		time.Sleep(2 * time.Millisecond)
	}
	return root
}

// sdkWindow is a window of the tests, with its frames.
func sdkWindow(root *Root) *ui.Tester {
	return ui.NewTester(root.View, 480, 900)
}

// sdkSettle draws the frame once the transitions of the window are over.
func sdkSettle(tt *ui.Tester) {
	tt.Advance(2 * time.Second)
	tt.Frame()
}

// sdkHasText reports whether the frame shows a text that is exactly s.
func sdkHasText(tt *ui.Tester, s string) bool {
	for _, text := range tt.Texts() {
		if text == s {
			return true
		}
	}
	return false
}

// sdkWait draws frames until the frame meets cond. The loads of the screens run in
// the background, so a test waits for them this way; it fails after a few seconds.
func sdkWait(t *testing.T, tt *ui.Tester, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		tt.Frame()
		if cond() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s; the frame shows %q", what, tt.Texts())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// sdkWaitText waits until the frame shows the text s exactly.
func sdkWaitText(t *testing.T, tt *ui.Tester, s string) {
	t.Helper()
	sdkWait(t, tt, "the frame does not show "+s, func() bool { return sdkHasText(tt, s) })
}

// sdkClickScrolled clicks the element that shows text. It scrolls the window down
// until the element is in it first, since a long list is taller than the window.
func sdkClickScrolled(t *testing.T, tt *ui.Tester, text string) {
	t.Helper()
	for range 80 {
		if box, ok := tt.Find(text); ok && box.Y >= 0 && box.Y+box.H <= 900 {
			if err := tt.Click(text); err != nil {
				t.Fatalf("Click(%q): %v", text, err)
			}
			return
		}
		tt.Scroll(240, 450, 0, 300)
		tt.Frame()
	}
	t.Fatalf("%q does not come into the window; the frame shows %q", text, tt.Texts())
}

// sdkClickSwitch clicks the switch at the end of the row whose title shows text. The
// switch is labelled with the same text, and a click on the text would reach the
// title of the row, so the switch is told apart by its place: the row's padding, and
// half of the width of the switch, from the right edge of the window.
func sdkClickSwitch(t *testing.T, tt *ui.Tester, text string) {
	t.Helper()
	box, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no element shows %q; the frame shows %q", text, tt.Texts())
	}
	const fromEdge = 16 + 16 + 26
	tt.ClickAt(480-fromEdge, box.Y+box.H/2)
}

// sdkBesideGap is how far past the right edge of the anchor the click of
// sdkClickBeside lands, in points.
const sdkBesideGap = 28

// sdkClickBeside clicks the place just right of the element that shows anchor, where
// the button next to it is, in a row that ends with the anchor's button. A dialog
// titles itself with the text of its confirm button, and a click on the text would
// reach the title, so the confirm button is told apart by its place.
func sdkClickBeside(t *testing.T, tt *ui.Tester, anchor string) {
	t.Helper()
	box, ok := tt.Find(anchor)
	if !ok {
		t.Fatalf("no element shows %q; the frame shows %q", anchor, tt.Texts())
	}
	tt.ClickAt(box.X+box.W+sdkBesideGap, box.Y+box.H/2)
}

// sdkShot saves the frame as a PNG in the directory named by PHONE_SDK_SHOTS, for
// the check of the layout by eye. Without the variable, nothing is saved.
func sdkShot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("PHONE_SDK_SHOTS")
	if dir == "" {
		return
	}
	file, err := os.Create(filepath.Join(dir, name+".png")) //nolint:gosec // G304, G703: PHONE_SDK_SHOTS names the directory; a developer sets it to keep the pictures.
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
