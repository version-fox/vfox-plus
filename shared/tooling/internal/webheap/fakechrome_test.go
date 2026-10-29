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

package webheap

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
)

// The tests run this test binary as the browser: with fakeChromeEnv set, it
// does what Chrome does for vfxtool webheap (prints the DevTools address, lists
// one page, and answers the commands over a WebSocket) instead of running the
// tests. The value of the variable is the way it misbehaves:
//
//	ok             the way it should work
//	exit           exits before it listens
//	late-page      lists no page for the first few requests
//	navigate-error cannot open the app
//	redirect       sends the app somewhere else
//	never-ready    never shows an SDK list
//	no-element     shows the SDK list, but none of the elements that a scene clicks
//	fragmented     sends the screenshot in fragments, with a ping inside
//
// fakeChromeLogEnv names the file where every command is logged, one line each.
const (
	fakeChromeEnv    = "VFOX_WEBHEAP_FAKE_CHROME"
	fakeChromeLogEnv = "VFOX_WEBHEAP_FAKE_LOG"
)

// fakeScreenshot is the picture that the fake browser takes: larger than 64 KB
// in the message, so that the client reads a 64-bit length.
var fakeScreenshot = bytes.Repeat([]byte{0x89, 'P', 'N', 'G'}, 75000)

func runFakeChrome(mode string) {
	if mode == "exit" {
		fmt.Fprintln(os.Stderr, "fake chrome: cannot open a display")
		os.Exit(3)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	page := &fakePage{mode: mode, heaps: []float64{1000, 3000, 2000, 5000, 4000, 6000}}
	page.log("start", strings.Join(os.Args[1:], " "))
	page.log("pid", fmt.Sprint(os.Getpid()))
	mux := http.NewServeMux()
	var lists int
	mux.HandleFunc("/json/list", func(w http.ResponseWriter, _ *http.Request) {
		lists++
		targets := []map[string]string{
			{"type": "service_worker", "url": "https://example.test/", "webSocketDebuggerUrl": "ws://127.0.0.1:1/never"},
			{"type": "page", "url": "about:blank", "webSocketDebuggerUrl": fmt.Sprintf("ws://127.0.0.1:%d/devtools/page/1", port)},
		}
		if mode == "late-page" && lists <= 3 {
			targets = nil
		}
		_ = json.NewEncoder(w).Encode(targets)
	})
	mux.HandleFunc("/devtools/page/1", page.serve)
	go func() { _ = http.Serve(listener, mux) }()
	fmt.Fprintf(os.Stderr, "\nDevTools listening on ws://127.0.0.1:%d/devtools/browser/fake\n", port)
	select {} // Until the tool kills this process.
}

// fakePage is the one page of the fake browser.
type fakePage struct {
	mode string

	mu    sync.Mutex
	url   string
	heaps []float64
}

func (p *fakePage) log(kind, detail string) {
	path := os.Getenv(fakeChromeLogEnv)
	if path == "" {
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = file.Close() }()
	_, _ = fmt.Fprintf(file, "%s\t%s\n", kind, strings.ReplaceAll(detail, "\n", " "))
}

func (p *fakePage) serve(w http.ResponseWriter, r *http.Request) {
	socket, err := acceptSocket(w, r)
	if err != nil {
		return
	}
	defer func() { _ = socket.conn.Close() }()
	for {
		_, opcode, payload, err := socket.read()
		if err != nil || opcode == opClose {
			return
		}
		if opcode != opText {
			continue
		}
		var request struct {
			ID     int             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(payload, &request) != nil {
			continue
		}
		p.handle(socket, request.ID, request.Method, request.Params)
	}
}

func (p *fakePage) reply(s *serverSocket, id int, result any) {
	raw, _ := json.Marshal(map[string]any{"id": id, "result": result})
	_ = s.send(true, opText, raw)
}

func (p *fakePage) fail(s *serverSocket, id int, message string) {
	raw, _ := json.Marshal(map[string]any{"id": id, "error": map[string]any{"code": -32601, "message": message}})
	_ = s.send(true, opText, raw)
}

func (p *fakePage) event(s *serverSocket, method string) {
	raw, _ := json.Marshal(map[string]any{"method": method, "params": map[string]any{"timestamp": 1}})
	_ = s.send(true, opText, raw)
}

func (p *fakePage) handle(s *serverSocket, id int, method string, params json.RawMessage) {
	switch method {
	case "Page.enable", "Runtime.enable":
		p.log(method, "")
		p.reply(s, id, map[string]any{})
	case "Emulation.setEmulatedMedia":
		p.log(method, string(params))
		p.reply(s, id, map[string]any{})
	case "Page.navigate":
		var navigation struct {
			URL string `json:"url"`
		}
		_ = json.Unmarshal(params, &navigation)
		p.log(method, navigation.URL)
		if p.mode == "navigate-error" && strings.HasSuffix(navigation.URL, "/app") {
			p.reply(s, id, map[string]any{"frameId": "1", "errorText": "net::ERR_CONNECTION_REFUSED"})
			return
		}
		p.mu.Lock()
		p.url = navigation.URL
		p.mu.Unlock()
		p.reply(s, id, map[string]any{"frameId": "1"})
		p.event(s, "Page.loadEventFired")
	case "Runtime.evaluate":
		var evaluation struct {
			Expression string `json:"expression"`
		}
		_ = json.Unmarshal(params, &evaluation)
		p.log(method, evaluation.Expression)
		p.evaluate(s, id, evaluation.Expression)
	case "Runtime.getHeapUsage":
		p.log(method, "")
		p.mu.Lock()
		used := p.heaps[0]
		p.heaps = append(p.heaps[1:], p.heaps[0])
		p.mu.Unlock()
		p.reply(s, id, map[string]any{"usedSize": used, "totalSize": used * 2})
	case "Browser.getVersion":
		p.log(method, "")
		p.reply(s, id, map[string]any{"product": "FakeChrome/1.2.3"})
	case "Page.captureScreenshot":
		p.log(method, string(params))
		raw, _ := json.Marshal(map[string]any{"id": id, "result": map[string]any{"data": base64.StdEncoding.EncodeToString(fakeScreenshot)}})
		if p.mode != "fragmented" {
			_ = s.send(true, opText, raw)
			return
		}
		third := len(raw) / 3
		_ = s.send(false, opText, raw[:third])
		_ = s.send(true, opPing, []byte("keep-alive"))
		_ = s.send(false, opContinuation, raw[third:2*third])
		_ = s.send(true, opContinuation, raw[2*third:])
	default:
		p.log(method, "unknown")
		p.fail(s, id, "'"+method+"' wasn't found")
	}
}

func (p *fakePage) evaluate(s *serverSocket, id int, expression string) {
	value := func(v any) {
		p.reply(s, id, map[string]any{"result": map[string]any{"value": v}})
	}
	switch {
	case expression == "location.href":
		p.mu.Lock()
		href := p.url
		p.mu.Unlock()
		if p.mode == "redirect" && strings.HasSuffix(href, "/app") {
			href = "http://127.0.0.1:1/"
		}
		value(href)
	case strings.Contains(expression, "#sdkList"):
		value(p.mode != "never-ready")
	case strings.Contains(expression, `querySelectorAll("*")`):
		value(321)
	case strings.HasPrefix(expression, "localStorage.setItem"):
		p.reply(s, id, map[string]any{"result": map[string]any{"type": "undefined"}})
		p.event(s, "Page.loadEventFired") // The reload.
	case strings.Contains(expression, "element.click()") || strings.HasPrefix(expression, "!!document.querySelector(") ||
		strings.Contains(expression, "textContent"):
		value(p.mode != "no-element") // A step of a scene: the element is there, or is not.
	case strings.Contains(expression, "throw"):
		p.reply(s, id, map[string]any{"result": map[string]any{"type": "object"}, "exceptionDetails": map[string]any{"text": "Uncaught"}})
	default:
		p.reply(s, id, map[string]any{"result": map[string]any{"type": "undefined"}})
	}
}
