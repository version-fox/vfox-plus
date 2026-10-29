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

package pair

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/mygo/state"
)

//go:embed webapp.html
var webappHTML []byte

// The page around its two placeholders: the favicon link in the head, and the
// icon in the brand mark of the navigation rail. The page is written in
// pieces, so that serving it copies nothing.
var (
	webappHead, webappRest, _   = bytes.Cut(webappHTML, []byte(faviconPlaceholder))
	webappMiddle, webappTail, _ = bytes.Cut(webappRest, []byte(logoPlaceholder))
)

// Web UI routes served by Server when it owns an AppModel:
//
//	/app                  the single-page interface (paired browsers only)
//	/api/inventory        installed SDKs, registry plugins and vfox config
//	/api/sdks/<name>/info    plugin metadata
//	/api/sdks/<name>/versions  installable versions
//	/api/tasks            list tasks (GET) and start one (POST)
//	/api/tasks/finished   forget finished tasks (DELETE)
//	/api/tasks/<id>       one task with its log tail (GET)
//	/api/tasks/<id>/log   the retained full log of a failed task (GET)
//	/api/tasks/<id>/cancel   cancel a task (POST)
//	/api/tasks/<id>/retry    run a task again (POST)
//	/api/plugins          add a plugin from a URL or an upload (POST)
//	/api/settings         the GUI settings (GET and PUT)
//	/api/events           server-sent events: task, inventory and vfox
//
// Every route except the intro and the pairing links needs a paired session.
// State-changing routes additionally need a same-origin Origin or Referer
// header when the browser sends one, so that a foreign page cannot drive a
// paired browser with cross-site requests. Requests without either header,
// like curl, are allowed through: the session cookie is the credential.
func (s *Server) serveApp(writer http.ResponseWriter, request *http.Request) {
	if !s.hasSession(request) {
		http.Redirect(writer, request, "/", http.StatusFound)
		return
	}
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	_, _ = writer.Write(webappHead)
	_, _ = io.WriteString(writer, faviconLink)
	_, _ = writer.Write(webappMiddle)
	_, _ = io.WriteString(writer, faviconURI)
	_, _ = writer.Write(webappTail)
}

func (s *Server) requireModel(writer http.ResponseWriter) *state.AppModel {
	if s.model == nil {
		http.Error(writer, "This server runs without an application model.", http.StatusServiceUnavailable)
		return nil
	}
	return s.model
}

// requireSession rejects unpaired browsers.
func (s *Server) requireSession(writer http.ResponseWriter, request *http.Request) bool {
	if !s.hasSession(request) {
		http.Error(writer, "Not paired. Open a pairing link first.", http.StatusUnauthorized)
		return false
	}
	return true
}

// requireSameOrigin rejects cross-site requests that carry an Origin or a
// Referer header from elsewhere. Same-origin fetches from the web interface
// always carry the server itself.
func (s *Server) requireSameOrigin(writer http.ResponseWriter, request *http.Request) bool {
	origin := request.Header.Get("Origin")
	if origin == "" {
		referer := request.Header.Get("Referer")
		if referer == "" {
			return true
		}
		if sameHost(referer, request.Host) {
			return true
		}
	} else if sameHost(origin, request.Host) {
		return true
	}
	http.Error(writer, "Cross-site requests are not allowed.", http.StatusForbidden)
	return false
}

// sameHost reports whether rawURL names host.
func sameHost(rawURL, host string) bool {
	withoutScheme := rawURL
	if i := strings.Index(withoutScheme, "://"); i >= 0 {
		withoutScheme = withoutScheme[i+3:]
	}
	if i := strings.IndexByte(withoutScheme, '/'); i >= 0 {
		withoutScheme = withoutScheme[:i]
	}
	return strings.EqualFold(withoutScheme, host)
}

// writeJSON answers with value.
func writeJSON(writer http.ResponseWriter, value interface{}) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(writer).Encode(value)
}

// writeVfoxError answers with the message of a vfox failure.
func writeVfoxError(writer http.ResponseWriter, err error) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(http.StatusBadGateway)
	_ = json.NewEncoder(writer).Encode(map[string]string{"error": err.Error()})
}

// serveEvents streams task, inventory and vfox changes to the web interface.
// Each message is `event: <kind>\ndata: {}\n\n`; the payload is empty and
// the browser refetches what changed. A heartbeat comment keeps proxies
// from closing idle connections.
func (s *Server) serveEvents(writer http.ResponseWriter, request *http.Request) {
	if !s.requireSession(writer, request) {
		return
	}
	model := s.requireModel(writer)
	if model == nil {
		return
	}
	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "Streaming is not supported.", http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	writer.Header().Set("Cache-Control", "no-store")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")

	notify := make(chan string, 8)
	emit := func(kind string) {
		select {
		case notify <- kind:
		default:
		}
	}
	var stops []state.Unsubscribe
	defer func() {
		for _, stop := range stops {
			stop()
		}
	}()
	stops = append(stops,
		model.Tasks.AddListener(func() { emit("task") }),
		model.Installed.AddListener(func() { emit("inventory") }),
		model.Registry.AddListener(func() { emit("inventory") }),
		model.VfoxConfig.AddListener(func() { emit("inventory") }),
		model.Vfox.AddListener(func() { emit("vfox") }),
	)
	var taskStops []state.Unsubscribe
	cleanupTasks := func() {
		for _, stop := range taskStops {
			stop()
		}
		taskStops = nil
	}
	defer cleanupTasks()
	watchTasks := func() {
		cleanupTasks()
		for _, task := range model.Tasks.Tasks() {
			task := task
			taskStops = append(taskStops, task.AddListener(func() { emit("task") }))
		}
	}
	// Rebuild the per-task listeners whenever anything changed, so that new
	// tasks are watched and finished ones are released.
	watchTasks()

	send := func(kind string) bool {
		_, err := fmt.Fprintf(writer, "event: %s\ndata: {}\n\n", kind)
		if err != nil {
			return false
		}
		flusher.Flush()
		return true
	}
	if !send("hello") {
		return
	}
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	pending := map[string]bool{}
	flush := func() bool {
		for kind := range pending {
			delete(pending, kind)
			if kind == "task" {
				watchTasks()
			}
			if !send(kind) {
				return false
			}
		}
		return true
	}
	for {
		select {
		case <-request.Context().Done():
			return
		case kind := <-notify:
			pending[kind] = true
			// Drain bursts, like download progress lines, into one update.
			for {
				select {
				case kind := <-notify:
					pending[kind] = true
				default:
					goto drained
				}
			}
		drained:
			if !flush() {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprintf(writer, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
