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

// tsFake is a computer for the rendering tests of the tasks and the settings: the
// remote API over TLS, with the state that a test sets. Each change that a screen
// makes to the computer is recorded, so that a test can check what was asked.
// Screenshots are written only when VFOX_PHONE_SHOTS names a directory.

import (
	"context"
	"encoding/json"
	"image/png"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/phone/store"
)

// tsSession is the session of the fake computer, which the phone sends as its cookie.
const tsSession = "test-session"

type tsFake struct {
	server      *httptest.Server
	fingerprint string

	mu       sync.Mutex
	tasks    []remote.Task
	logs     map[int]string
	settings remote.AppSettings
	web      remote.WebState
	config   []remote.ConfigEntry
	calls    []string

	// gates holds the paths whose answers wait for their test to release them (see tsHold).
	gates map[string]chan struct{}
}

// tsNewFake starts a computer, and stops it when the test ends.
func tsNewFake(t *testing.T) *tsFake {
	t.Helper()
	f := &tsFake{logs: map[int]string{}}
	f.server = newTLSServer(t, http.HandlerFunc(f.serve))
	base, err := url.Parse(f.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	probe, err := remote.Probe(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	f.fingerprint = probe.Fingerprint
	return f
}

// tsRoot returns the window of a phone that is paired with the computer f. The
// configuration directory is one of the test.
func tsRoot(t *testing.T, f *tsFake) *Root {
	t.Helper()
	i18n.SetLanguage(i18n.English)
	t.Cleanup(func() {
		settingsSystemLanguages = nil
		i18n.SetLanguage(i18n.English)
	})
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	dir, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	dir = filepath.Join(dir, configDirName)
	servers := store.NewFileServerStore(filepath.Join(dir, serverFile))
	sessions := store.NewFileSessionStore(filepath.Join(dir, sessionFile))
	if err := servers.Save(store.Record{Name: "desktop", Base: f.server.URL, Fingerprint: f.fingerprint}); err != nil {
		t.Fatal(err)
	}
	if err := sessions.Save(tsSession); err != nil {
		t.Fatal(err)
	}
	root, err := New()
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// tsSetTasks replaces the tasks of the computer.
func (f *tsFake) tsSetTasks(tasks ...remote.Task) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tasks = tasks
}

// tsSetStatus changes the status of a task that the computer has.
func (f *tsFake) tsSetStatus(id int, status string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.tasks {
		if f.tasks[i].ID == id {
			f.tasks[i].Status = status
		}
	}
}

// tsRecorded returns the changes that the screens asked for, in order.
func (f *tsFake) tsRecorded() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// tsHasCall reports whether a change was asked for, by its method and path.
func (f *tsFake) tsHasCall(call string) bool {
	for _, got := range f.tsRecorded() {
		if got == call {
			return true
		}
	}
	return false
}

// tsWaitCall draws the frames until the computer has a change, and fails at the
// deadline.
func tsWaitCall(t *testing.T, tt *ui.Tester, f *tsFake, call string) {
	t.Helper()
	tsWaitUntil(t, tt, "the computer to receive "+call, func() bool { return f.tsHasCall(call) })
}

// tsHold makes the computer keep its answer to one path until the returned function is
// called, or the test ends. A test uses it to have one read answered after the others,
// which is an order that a busy computer may give.
func (f *tsFake) tsHold(t *testing.T, path string) func() {
	t.Helper()
	gate := make(chan struct{})
	f.mu.Lock()
	if f.gates == nil {
		f.gates = map[string]chan struct{}{}
	}
	f.gates[path] = gate
	f.mu.Unlock()
	var once sync.Once
	release := func() { once.Do(func() { close(gate) }) }
	t.Cleanup(release)
	return release
}

func (f *tsFake) serve(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("vfox-plus-session"); err != nil || cookie.Value != tsSession {
		http.Error(w, "not paired", http.StatusUnauthorized)
		return
	}
	// A held path waits for its test outside the lock, so that the other requests are
	// answered meanwhile.
	f.mu.Lock()
	gate := f.gates[r.URL.Path]
	f.mu.Unlock()
	if gate != nil {
		<-gate
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	path := r.URL.Path
	body, _ := io.ReadAll(r.Body)
	switch {
	case r.Method == http.MethodGet && path == "/api/inventory":
		tsWriteJSON(w, f.inventory())
	case r.Method == http.MethodGet && path == "/api/tasks":
		tsWriteJSON(w, f.tasks)
	case r.Method == http.MethodDelete && path == "/api/tasks/finished":
		f.record(r, "")
		kept := f.tasks[:0]
		for _, task := range f.tasks {
			if task.Status == remote.TaskQueued || task.Status == remote.TaskRunning {
				kept = append(kept, task)
			}
		}
		f.tasks = kept
		w.WriteHeader(http.StatusNoContent)
	case strings.HasPrefix(path, "/api/tasks/"):
		f.task(w, r, strings.TrimPrefix(path, "/api/tasks/"))
	case path == "/api/settings" && r.Method == http.MethodGet:
		tsWriteJSON(w, f.settings)
	case path == "/api/settings" && r.Method == http.MethodPut:
		f.record(r, string(body))
		var patch map[string]json.RawMessage
		_ = json.Unmarshal(body, &patch)
		if raw, ok := patch["vfoxExecutable"]; ok {
			_ = json.Unmarshal(raw, &f.settings.VfoxExecutable)
		}
		if raw, ok := patch["language"]; ok {
			_ = json.Unmarshal(raw, &f.settings.Language)
		}
		if raw, ok := patch["extraEnvironment"]; ok {
			f.settings.ExtraEnvironment = nil
			_ = json.Unmarshal(raw, &f.settings.ExtraEnvironment)
		}
		tsWriteJSON(w, f.settings)
	case path == "/api/web" && r.Method == http.MethodGet:
		tsWriteJSON(w, f.web)
	case path == "/api/web" && r.Method == http.MethodPut:
		f.record(r, string(body))
		var patch remote.WebPatch
		_ = json.Unmarshal(body, &patch)
		if patch.Remote != nil {
			f.web.Remote = *patch.Remote
		}
		if patch.Port != nil {
			f.web.Port = *patch.Port
		}
		if patch.Lightweight != nil {
			f.web.Lightweight = *patch.Lightweight
		}
		tsWriteJSON(w, f.web)
	case path == "/api/web/pairing" && r.Method == http.MethodPost:
		f.record(r, string(body))
		var request struct {
			Host string `json:"host"`
		}
		_ = json.Unmarshal(body, &request)
		host := request.Host
		if host == "" {
			host = f.web.PublicAddress
		}
		tsWriteJSON(w, remote.PairingCode{URL: host + "/auth/connect/0123456789abcdef"})
	case path == "/api/config" && r.Method == http.MethodPut:
		f.record(r, string(body))
		var entry remote.ConfigEntry
		_ = json.Unmarshal(body, &entry)
		f.setConfig(entry.Key, entry.Value)
		w.WriteHeader(http.StatusNoContent)
	case path == "/api/config" && r.Method == http.MethodDelete:
		key := r.URL.Query().Get("key")
		f.record(r, key)
		f.setConfig(key, "")
		w.WriteHeader(http.StatusNoContent)
	case path == "/api/quit" && r.Method == http.MethodPost:
		f.record(r, "")
		w.WriteHeader(http.StatusAccepted)
	default:
		http.NotFound(w, r)
	}
}

// task answers the calls on one task: its detail, its log, and its cancel and retry.
func (f *tsFake) task(w http.ResponseWriter, r *http.Request, rest string) {
	idText, action, _ := strings.Cut(rest, "/")
	id, err := strconv.Atoi(idText)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	index := -1
	for i := range f.tasks {
		if f.tasks[i].ID == id {
			index = i
		}
	}
	switch {
	case action == "" && r.Method == http.MethodGet && index >= 0:
		tsWriteJSON(w, f.tasks[index])
	case action == "log" && r.Method == http.MethodGet:
		text, ok := f.logs[id]
		if !ok {
			http.Error(w, "no complete log", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = io.WriteString(w, text)
	case (action == "cancel" || action == "retry") && r.Method == http.MethodPost && index >= 0:
		f.record(r, "")
		if action == "cancel" {
			f.tasks[index].Status = remote.TaskCancelled
		} else {
			f.tasks[index].Status = remote.TaskQueued
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

// record keeps a change of the computer, as "METHOD path" and its body.
func (f *tsFake) record(r *http.Request, body string) {
	call := r.Method + " " + r.URL.Path
	if body != "" {
		call += " " + body
	}
	f.calls = append(f.calls, call)
}

// setConfig sets a key of the configuration, which the inventory lists.
func (f *tsFake) setConfig(key, value string) {
	for i := range f.config {
		if f.config[i].Key == key {
			f.config[i].Value = value
			return
		}
	}
	f.config = append(f.config, remote.ConfigEntry{Key: key, Value: value})
}

// inventory is the inventory of the computer: vfox, which runs, and the configuration.
func (f *tsFake) inventory() remote.Inventory {
	return remote.Inventory{
		Vfox:      remote.VfoxStatus{Availability: remote.AvailabilityReady, Version: "0.9.0"},
		Installed: remote.Section[[]remote.InstalledSdk]{Value: []remote.InstalledSdk{}},
		Registry:  remote.Section[[]remote.AvailablePlugin]{Value: []remote.AvailablePlugin{}},
		Config:    remote.Section[[]remote.ConfigEntry]{Value: f.config},
		Complete:  true,
	}
}

// tsWriteJSON answers v as JSON, with the status 200.
func tsWriteJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(v)
}

// tsSettle draws the frames once the transitions of the window are over.
func tsSettle(tt *ui.Tester) {
	tt.Advance(2 * time.Second)
	tt.Frame()
}

// tsHas reports whether the frame shows a text that is exactly s.
func tsHas(tt *ui.Tester, s string) bool {
	for _, text := range tt.Texts() {
		if text == s {
			return true
		}
	}
	return false
}

// tsWaitText draws the frames until the frame shows the text s, and fails at the deadline.
func tsWaitText(t *testing.T, tt *ui.Tester, s string) {
	t.Helper()
	tsWaitUntil(t, tt, "the text "+strconv.Quote(s), func() bool { return tsHas(tt, s) })
}

// tsWaitUntil draws the frames until ok reports true, and fails at the deadline.
func tsWaitUntil(t *testing.T, tt *ui.Tester, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		// The transitions of the window end with the time that the frames advance by.
		tt.Advance(100 * time.Millisecond)
		tt.Frame()
		if ok() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("waiting for %s; the frame shows %q", what, tt.Texts())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// tsShot saves the frame as a PNG, in the directory that VFOX_PHONE_SHOTS names. The
// transitions of the window are over in it: a page slides in, and a dialog scales in.
func tsShot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("VFOX_PHONE_SHOTS")
	if dir == "" {
		return
	}
	tsSettle(tt)
	if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // G703: VFOX_PHONE_SHOTS names the directory; a developer sets it to keep the pictures.
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

// tsTask is a task of the fake computer, with the time of its creation.
func tsTask(id int, operation, sdk, version, status string) remote.Task {
	return remote.Task{
		ID: id, Operation: operation, Sdk: sdk, Version: version, Status: status,
		CommandLine: "vfox " + operation + " " + sdk + "@" + version,
		CreatedAt:   "2026-11-20T10:00:00Z",
	}
}
