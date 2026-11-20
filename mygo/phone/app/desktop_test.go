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

package app

// desktop is a fake computer for the tests of the service: the API of the desktop
// application over TLS, with the answers the tests need, and the controls that
// make a computer misbehave on purpose. The contract of the remote client with the
// real server is tested in the remote package, against the server of mygo/pair.

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/mygo/phone/pairing"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
	"github.com/version-fox/vfox-plus/mygo/phone/store"
)

// sessionCookie is the cookie of the session, as the desktop names it.
const sessionCookie = "vfox-plus-session"

type desktop struct {
	t           *testing.T
	server      *httptest.Server
	base        *url.URL
	fingerprint string
	// certificate is the DER encoding of the certificate that the computer presents,
	// which the phone pins once the user has compared its fingerprint.
	certificate []byte

	mu        sync.Mutex
	tokens    map[string]bool
	sessions  map[string]bool
	nextID    int
	tasks     []remote.Task
	config    map[string]string
	search    map[string]string
	variables map[string]string
	settings  remote.AppSettings
	web       remote.WebState
	requests  map[string]int
	holds     map[string]chan struct{}
	failures  map[string]int
	subs      map[chan string]bool
	streams   int
	opened    int
	cut       chan struct{}
	releases  []func()
}

// newDesktop starts a computer with one failed task, and a configuration of one key.
// It is stopped when the test ends.
func newDesktop(t *testing.T) *desktop {
	t.Helper()
	d := &desktop{
		t:         t,
		tokens:    map[string]bool{},
		sessions:  map[string]bool{},
		nextID:    2,
		config:    map[string]string{"proxy.url": ""},
		search:    map[string]string{"java": "lts"},
		variables: map[string]string{},
		settings:  remote.AppSettings{Language: remote.LanguageSystem, ExtraEnvironment: map[string]string{}},
		web:       remote.WebState{Port: 8369, Addresses: []string{"192.168.1.10"}, Fingerprint: "AB:CD"},
		tasks: []remote.Task{{
			ID: 1, Operation: remote.OperationInstall, Sdk: "nodejs", Version: "22.21.0",
			Status: remote.TaskFailed, CommandLine: "vfox install nodejs@22.21.0",
			Log: []string{"no such version"}, HasFullLog: true,
		}},
		requests: map[string]int{},
		holds:    map[string]chan struct{}{},
		failures: map[string]int{},
		subs:     map[chan string]bool{},
		cut:      make(chan struct{}),
	}
	// The server logs the handshakes that the tests refuse on purpose: that is noise.
	d.server = httptest.NewUnstartedServer(d)
	d.server.Config.ErrorLog = log.New(io.Discard, "", 0)
	d.server.StartTLS()
	base, err := url.Parse(d.server.URL)
	if err != nil {
		t.Fatal(err)
	}
	d.base = base
	d.certificate = d.server.Certificate().Raw
	d.fingerprint = pairing.FormatFingerprint(d.certificate)
	t.Cleanup(d.close)
	return d
}

// pairedRecord is the record that a pairing with d saves: the name of a computer that
// the user did not name is the host of its address.
func (d *desktop) pairedRecord() store.Record {
	record := d.record()
	record.Name = d.base.Hostname()
	return record
}

// record is the record of d as the phone saves it after a pairing: the computer, the
// fingerprint that the user compared, and the certificate that it pins.
func (d *desktop) record() store.Record {
	return store.Record{
		Name:        "desktop",
		Base:        d.base.String(),
		Fingerprint: d.fingerprint,
		Certificate: store.EncodeCertificate(d.certificate),
	}
}

// close releases the requests that wait, and stops the server.
func (d *desktop) close() {
	d.mu.Lock()
	releases := d.releases
	d.releases = nil
	d.mu.Unlock()
	for _, release := range releases {
		release()
	}
	d.server.Close()
}

// issueLink makes a one-time pairing link, as the desktop does for a new device.
func (d *desktop) issueLink() pairing.Link {
	d.t.Helper()
	token := fmt.Sprintf("token-%d", time.Now().UnixNano())
	d.mu.Lock()
	d.tokens[token] = true
	d.mu.Unlock()
	link, err := pairing.ParseLink(d.base.String() + "/auth/connect/" + token)
	if err != nil {
		d.t.Fatal(err)
	}
	return link
}

// issueSession makes a session that the computer accepts, as a pairing does.
func (d *desktop) issueSession(session string) string {
	d.mu.Lock()
	d.sessions[session] = true
	d.mu.Unlock()
	return session
}

// revoke makes the computer refuse every session the phone has.
func (d *desktop) revoke() {
	d.mu.Lock()
	d.sessions = map[string]bool{}
	d.mu.Unlock()
}

// hold makes the requests with method and path wait until the returned function
// releases them. A request is counted as soon as it arrives.
func (d *desktop) hold(method, path string) func() {
	ch := make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(ch) }) }
	d.mu.Lock()
	d.holds[method+" "+path] = ch
	d.releases = append(d.releases, release)
	d.mu.Unlock()
	return release
}

// fail makes the requests with method and path answer status; status zero makes
// them answer normally again.
func (d *desktop) fail(method, path string, status int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if status == 0 {
		delete(d.failures, method+" "+path)
		return
	}
	d.failures[method+" "+path] = status
}

// count returns how many GET requests with path arrived.
func (d *desktop) count(path string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.requests["GET "+path]
}

// openStreams returns how many event streams are open now.
func (d *desktop) openStreams() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.streams
}

// openedStreams returns how many event streams were ever opened.
func (d *desktop) openedStreams() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.opened
}

// emit sends an event of kind to the open event streams.
func (d *desktop) emit(kind string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for ch := range d.subs {
		select {
		case ch <- kind:
		default:
		}
	}
}

// breakStreams ends the open event streams, as a network break does.
func (d *desktop) breakStreams() {
	d.mu.Lock()
	defer d.mu.Unlock()
	close(d.cut)
	d.cut = make(chan struct{})
}

// ServeHTTP answers the API.
func (d *desktop) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.Method + " " + r.URL.Path
	d.mu.Lock()
	d.requests[key]++
	hold := d.holds[key]
	status, failing := d.failures[key]
	d.mu.Unlock()
	if strings.HasPrefix(r.URL.Path, "/auth/connect/") {
		d.pair(w, strings.TrimPrefix(r.URL.Path, "/auth/connect/"))
		return
	}
	if !d.authorized(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "pair again"})
		return
	}
	if hold != nil {
		select {
		case <-hold:
		case <-r.Context().Done():
			return
		}
	}
	if failing {
		writeJSON(w, status, map[string]string{"error": "the test asked for this failure"})
		return
	}
	d.route(w, r)
}

// pair exchanges a one-time token for a session, as the desktop does.
func (d *desktop) pair(w http.ResponseWriter, token string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if !d.tokens[token] {
		http.Error(w, "the pairing link is invalid, used or expired", http.StatusForbidden)
		return
	}
	delete(d.tokens, token)
	session := fmt.Sprintf("session-%d", len(d.sessions)+1)
	d.sessions[session] = true
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: session, Path: "/",
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	w.WriteHeader(http.StatusNoContent)
}

// authorized reports whether the request carries a session the computer accepts.
func (d *desktop) authorized(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookie)
	if err != nil {
		return false
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sessions[cookie.Value]
}

// route answers an authorized request. Paths are matched by their shape: a task's
// ID and an SDK's name are written as :id and :name.
func (d *desktop) route(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	shape := make([]string, len(parts))
	for i, part := range parts {
		shape[i] = part
		if i > 0 && parts[i-1] == "tasks" && part != "finished" {
			shape[i] = ":id"
		}
		if i > 0 && parts[i-1] == "sdks" {
			shape[i] = ":name"
		}
	}
	var id int
	if len(parts) > 2 && parts[1] == "tasks" && shape[2] == ":id" {
		// The error is ignored on purpose: a part that is not a number leaves id at zero.
		_, _ = fmt.Sscan(parts[2], &id)
	}
	name := ""
	if len(parts) > 2 && parts[1] == "sdks" {
		name = parts[2]
	}
	switch r.Method + " " + strings.Join(shape, "/") {
	case "GET api/events":
		d.events(w, r)
	case "GET api/settings":
		d.mu.Lock()
		defer d.mu.Unlock()
		writeJSON(w, http.StatusOK, d.settings)
	case "PUT api/settings":
		d.putSettings(w, r)
	case "GET api/inventory":
		writeJSON(w, http.StatusOK, d.inventory())
	case "GET api/tasks":
		d.mu.Lock()
		defer d.mu.Unlock()
		writeJSON(w, http.StatusOK, d.tasks)
	case "POST api/tasks":
		d.startTask(w, r)
	case "POST api/plugins":
		d.startTask(w, r)
	case "DELETE api/tasks/finished":
		d.clearFinished(w)
	case "GET api/tasks/:id":
		d.getTask(w, id)
	case "GET api/tasks/:id/log":
		d.getLog(w, id)
	case "POST api/tasks/:id/cancel":
		d.setStatus(w, id, remote.TaskCancelled, remote.TaskQueued, remote.TaskRunning)
	case "POST api/tasks/:id/retry":
		d.setStatus(w, id, remote.TaskQueued, remote.TaskFailed, remote.TaskCancelled)
	case "PUT api/config":
		d.putConfig(w, r)
	case "DELETE api/config":
		d.mu.Lock()
		delete(d.config, r.URL.Query().Get("key"))
		d.mu.Unlock()
		w.WriteHeader(http.StatusNoContent)
	case "GET api/doctor":
		writeJSON(w, http.StatusOK, remote.DoctorReport{OK: true, Findings: []remote.DoctorFinding{}})
	case "GET api/web":
		d.mu.Lock()
		defer d.mu.Unlock()
		writeJSON(w, http.StatusOK, d.web)
	case "PUT api/web":
		d.putWeb(w, r)
	case "POST api/web/pairing":
		token := fmt.Sprintf("web-%d", time.Now().UnixNano())
		d.mu.Lock()
		d.tokens[token] = true
		d.mu.Unlock()
		writeJSON(w, http.StatusOK, remote.PairingCode{URL: d.base.String() + "/auth/connect/" + token, SVG: "<svg/>"})
	case "POST api/quit":
		w.WriteHeader(http.StatusAccepted)
	case "GET api/sdks/:name/info":
		writeJSON(w, http.StatusOK, remote.PluginInfo{Name: name, Version: "1.0.0", Homepage: "https://example.test/" + name})
	case "GET api/sdks/:name/versions":
		d.versions(w, r, name)
	case "GET api/sdks/:name/profile":
		d.profile(w, name)
	case "GET api/sdks/:name/aliases":
		writeJSON(w, http.StatusOK, []remote.VersionAlias{{Sdk: name, Alias: "lts", Target: name + "@22.21.0", Display: "lts"}})
	case "PUT api/sdks/:name/search":
		d.putSearch(w, r, name)
	case "PUT api/sdks/:name/variables":
		d.putVariable(w, r, name)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such route " + r.URL.Path})
	}
}

// events streams the events of the computer: a hello, then the events that emit
// sends, until the client leaves or the stream is broken.
func (d *desktop) events(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	ch := make(chan string, 16)
	d.mu.Lock()
	d.subs[ch] = true
	d.streams++
	d.opened++
	cut := d.cut
	d.mu.Unlock()
	defer func() {
		d.mu.Lock()
		delete(d.subs, ch)
		d.streams--
		d.mu.Unlock()
	}()
	_, _ = fmt.Fprint(w, "event: hello\ndata: {}\n\n")
	flusher.Flush()
	for {
		select {
		case kind := <-ch:
			_, _ = fmt.Fprintf(w, "event: %s\ndata: {}\n\n", kind)
			flusher.Flush()
		case <-cut:
			return
		case <-r.Context().Done():
			return
		}
	}
}

// inventory returns the inventory as the desktop answers it.
func (d *desktop) inventory() remote.Inventory {
	d.mu.Lock()
	defer d.mu.Unlock()
	keys := make([]string, 0, len(d.config))
	for key := range d.config {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	config := make([]remote.ConfigEntry, 0, len(keys))
	for _, key := range keys {
		config = append(config, remote.ConfigEntry{Key: key, Value: d.config[key]})
	}
	return remote.Inventory{
		Vfox:      remote.VfoxStatus{Availability: remote.AvailabilityReady, Version: "0.9.0"},
		Installed: remote.Section[[]remote.InstalledSdk]{Value: []remote.InstalledSdk{{Name: "nodejs", Versions: []string{"22.21.0"}, Current: "22.21.0", HasCurrent: true}}},
		Registry:  remote.Section[[]remote.AvailablePlugin]{Value: []remote.AvailablePlugin{{Name: "nodejs", Homepage: "https://nodejs.org", Official: true}}},
		Config:    remote.Section[[]remote.ConfigEntry]{Value: config},
		Complete:  true,
	}
}

// startTask queues the task that the request names, and answers 201 with it.
func (d *desktop) startTask(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Operation string `json:"operation"`
		Sdk       string `json:"sdk"`
		Version   string `json:"version"`
		Source    string `json:"source"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	task := remote.Task{
		ID: d.nextID, Operation: request.Operation, Sdk: request.Sdk, Version: request.Version,
		Status: remote.TaskQueued, CommandLine: "vfox " + request.Operation, Log: []string{},
	}
	d.nextID++
	d.tasks = append(d.tasks, task)
	writeJSON(w, http.StatusCreated, task)
}

// clearFinished forgets the tasks that are not queued or running.
func (d *desktop) clearFinished(w http.ResponseWriter) {
	d.mu.Lock()
	defer d.mu.Unlock()
	kept := d.tasks[:0]
	for _, task := range d.tasks {
		if task.Status == remote.TaskQueued || task.Status == remote.TaskRunning {
			kept = append(kept, task)
		}
	}
	d.tasks = kept
	w.WriteHeader(http.StatusNoContent)
}

// getTask answers the task with id, or 404.
func (d *desktop) getTask(w http.ResponseWriter, id int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, task := range d.tasks {
		if task.ID == id {
			writeJSON(w, http.StatusOK, task)
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such task"})
}

// getLog answers the complete output of a task that kept one, or 404.
func (d *desktop) getLog(w http.ResponseWriter, id int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, task := range d.tasks {
		if task.ID == id && task.HasFullLog {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = io.WriteString(w, fmt.Sprintf("the complete output of task %d\n", id))
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "no complete output"})
}

// setStatus moves the task with id to status, when its status is one of from.
func (d *desktop) setStatus(w http.ResponseWriter, id int, status string, from ...string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for i := range d.tasks {
		if d.tasks[i].ID != id {
			continue
		}
		for _, allowed := range from {
			if d.tasks[i].Status == allowed {
				d.tasks[i].Status = status
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": "the task is in another state"})
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "no such task"})
}

// putConfig sets a key of the configuration.
func (d *desktop) putConfig(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	d.mu.Lock()
	d.config[request.Key] = request.Value
	d.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// putSettings applies the settings that the patch sets, and answers all of them.
func (d *desktop) putSettings(w http.ResponseWriter, r *http.Request) {
	var patch remote.SettingsPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if patch.VfoxExecutable != nil {
		d.settings.VfoxExecutable = *patch.VfoxExecutable
	}
	if patch.Language != nil {
		d.settings.Language = *patch.Language
	}
	if patch.UpdateMirror != nil {
		d.settings.UpdateMirror = *patch.UpdateMirror
	}
	if patch.ExtraEnvironment != nil {
		d.settings.ExtraEnvironment = *patch.ExtraEnvironment
	}
	writeJSON(w, http.StatusOK, d.settings)
}

// putWeb applies the web settings that the patch sets, and answers the state.
func (d *desktop) putWeb(w http.ResponseWriter, r *http.Request) {
	var patch remote.WebPatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if patch.Port != nil {
		d.web.Port = *patch.Port
	}
	if patch.Remote != nil {
		d.web.Remote = *patch.Remote
	}
	writeJSON(w, http.StatusOK, d.web)
}

// versions answers the versions of a plugin; the note carries the plugin's search,
// so that a change of it shows in the answer.
func (d *desktop) versions(w http.ResponseWriter, r *http.Request, name string) {
	d.mu.Lock()
	search := d.search[name]
	d.mu.Unlock()
	writeJSON(w, http.StatusOK, []remote.AvailableVersion{{
		Version: "22.21.0", Note: "search " + search, HasNote: true,
		Additions: []remote.VersionAddition{}, Installed: true,
	}, {
		Version: "20.19.0", Note: "search " + r.URL.Query().Get("search"),
		Additions: []remote.VersionAddition{},
	}})
}

// profile answers the profile of a plugin; the plugin "none" has no profile.
func (d *desktop) profile(w http.ResponseWriter, name string) {
	if name == "none" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "the plugin has no profile"})
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	writeJSON(w, http.StatusOK, remote.Profile{
		Plugin: name, Naming: remote.NamingPlain,
		Search: &remote.ProfileSearch{Default: "lts", Current: d.search[name], Choices: []remote.Choice{{Value: "lts"}}},
		Variables: []remote.Variable{{
			Key: "VFOX_MIRROR", Kind: remote.VariableURL, Value: d.variables[name+" VFOX_MIRROR"],
		}},
	})
}

// putSearch sets the search of a plugin.
func (d *desktop) putSearch(w http.ResponseWriter, r *http.Request, name string) {
	var request struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	d.mu.Lock()
	d.search[name] = request.Value
	d.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// putVariable sets a variable of a plugin.
func (d *desktop) putVariable(w http.ResponseWriter, r *http.Request, name string) {
	var request struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	d.mu.Lock()
	d.variables[name+" "+request.Key] = request.Value
	d.mu.Unlock()
	w.WriteHeader(http.StatusNoContent)
}

// writeJSON answers v as JSON with status.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
