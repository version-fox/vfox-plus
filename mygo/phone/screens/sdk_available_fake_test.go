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

// The computer that the screen tests read: a TLS server that answers the remote API
// with the data a test sets, and records the calls that change the computer. It is
// shared by the tests of the installable versions and of the plugin tab.

import (
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

// availCall is a call that changes the computer: its method, its path with the query,
// and the body that it sent.
type availCall struct {
	Method string
	Path   string
	Body   string
}

// availComputer is the computer of a test. Its data is what the test sets before the
// window reads it. A profile of nil answers 404, which is a plugin with no profile.
type availComputer struct {
	server *httptest.Server

	mu        sync.Mutex
	inventory remote.Inventory
	profiles  map[string]*remote.Profile
	versions  map[string][]remote.AvailableVersion
	// versionsStatus, when not 0, is the status that the versions answer with.
	versionsStatus int
	// failTasks, when set, makes the computer refuse to start a task.
	failTasks bool
	// hold, when not nil, keeps every read waiting until it is closed, so that a test
	// can show the loading (see holdReads).
	hold  chan struct{}
	calls []availCall
}

// holdReads makes the reads of the computer wait until the returned function is
// called, or the test ends. The window then shows its loading state.
func (f *availComputer) holdReads(t *testing.T) func() {
	t.Helper()
	hold := make(chan struct{})
	f.mu.Lock()
	f.hold = hold
	f.mu.Unlock()
	var once sync.Once
	release := func() { once.Do(func() { close(hold) }) }
	t.Cleanup(release)
	return release
}

// availNewComputer starts the computer, and stops it when the test ends.
func availNewComputer(t *testing.T) *availComputer {
	t.Helper()
	f := &availComputer{profiles: map[string]*remote.Profile{}, versions: map[string][]remote.AvailableVersion{}}
	f.server = newTLSServer(t, http.HandlerFunc(f.serve))
	return f
}

// setCall returns the calls that changed the computer, in order.
func (f *availComputer) recorded() []availCall {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]availCall(nil), f.calls...)
}

// serve answers one request of the remote API.
func (f *availComputer) serve(w http.ResponseWriter, req *http.Request) {
	f.mu.Lock()
	hold := f.hold
	f.mu.Unlock()
	if hold != nil && req.Method == http.MethodGet {
		<-hold
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	body, _ := io.ReadAll(req.Body)
	path := req.URL.Path
	if req.Method != http.MethodGet {
		f.calls = append(f.calls, availCall{Method: req.Method, Path: req.URL.RequestURI(), Body: string(body)})
	}
	switch {
	case req.Method == http.MethodGet && path == "/api/inventory":
		availWriteJSON(w, f.inventory)
	case req.Method == http.MethodGet && strings.HasSuffix(path, "/profile"):
		name := availSDKName(path)
		profile, ok := f.profiles[name]
		if !ok || profile == nil {
			http.Error(w, "no profile", http.StatusNotFound)
			return
		}
		availWriteJSON(w, profile)
	case req.Method == http.MethodGet && strings.HasSuffix(path, "/versions"):
		if f.versionsStatus != 0 {
			http.Error(w, "the plugin failed", f.versionsStatus)
			return
		}
		availWriteJSON(w, f.versions[availSDKName(path)])
	case req.Method == http.MethodPut && strings.HasSuffix(path, "/search"):
		// The computer keeps the choice, and the profile names it from then on.
		var choice struct {
			Value string `json:"value"`
		}
		_ = json.Unmarshal(body, &choice)
		if profile := f.profiles[availSDKName(path)]; profile != nil && profile.Search != nil {
			profile.Search.Current = choice.Value
		}
		w.WriteHeader(http.StatusNoContent)
	case req.Method == http.MethodPost && (path == "/api/tasks" || path == "/api/plugins"):
		if f.failTasks {
			http.Error(w, "the task was refused", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(remote.Task{ID: 7, Operation: remote.OperationInstall, Status: remote.TaskQueued})
	default:
		http.NotFound(w, req)
	}
}

// availSDKName returns the plugin named in a path such as /api/sdks/go/versions.
func availSDKName(path string) string {
	parts := strings.Split(strings.TrimPrefix(path, "/api/sdks/"), "/")
	return parts[0]
}

// availWriteJSON answers with v as JSON.
func availWriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// availRoot returns the window of a phone paired with the computer f: the saved
// computer and its session are in the configuration directory of the test, and the
// window reads them. Nothing runs the event stream, so the reads load on their own.
func availRoot(t *testing.T, f *availComputer) *Root {
	t.Helper()
	i18n.SetLanguage(i18n.English)
	t.Cleanup(func() { i18n.SetLanguage(i18n.English) })
	config := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", config)
	dir := filepath.Join(config, configDirName)
	record := store.Record{
		Name:        "computer",
		Base:        f.server.URL,
		Fingerprint: pairing.FormatFingerprint(f.server.Certificate().Raw),
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
	// The connection is checked by the event stream, which these tests do not run: the
	// window is shown as paired, as it is once the computer answers.
	root.App().SetSession(app.Paired)
	return root
}

// availWaitText draws frames until the frame shows the text want, which must be exact.
// The reads load in the background, so the frames that come before the load show the
// loading.
func availWaitText(t *testing.T, tt *ui.Tester, want string) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		tt.Frame()
		if hasText(tt, want) {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("the frame does not show %q; it shows %q", want, tt.Texts())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// availWaitCalls waits until the computer has recorded a call, and returns the calls.
func availWaitCalls(t *testing.T, tt *ui.Tester, f *availComputer) []availCall {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		tt.Frame()
		if calls := f.recorded(); len(calls) > 0 {
			return calls
		}
		if time.Now().After(deadline) {
			t.Fatalf("the computer recorded %v; want a call", f.recorded())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// availShot saves the frame as a PNG in the directory named by PHONE_SHOT_DIR, so that a
// person can look at the layout. Without that variable it saves nothing.
func availShot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("PHONE_SHOT_DIR")
	if dir == "" {
		return
	}
	// The transitions of the dialogs and the menus end within two seconds.
	settle(tt)
	if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // G703: PHONE_SHOT_DIR names the directory; a developer sets it to keep the pictures.
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(dir, name+".png")) //nolint:gosec // G304, G703: the directory of the pictures, as above.
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	if err := png.Encode(file, tt.Image()); err != nil {
		t.Fatal(err)
	}
}
