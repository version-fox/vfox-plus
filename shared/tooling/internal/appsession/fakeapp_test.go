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

package appsession

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/automation"
)

// act is an action that the fake application received.
type act struct {
	ID     string
	Action string
	Text   string
}

// fakeApp is an application that speaks the protocol of the automation plugin, with
// a tree that the test sets and an action that may change it: what the window shows
// after a press, as the application would.
type fakeApp struct {
	mu     sync.Mutex
	nodes  []automation.Node
	acts   []act
	inputs []automation.Event
	idle   *automation.Error
	// onAct runs for each action and may return a failure for it.
	onAct func(app *fakeApp, a act) *automation.Error
	// treeFailure makes every tree request fail.
	treeFailure *automation.Error
}

// set replaces the elements of the window.
func (f *fakeApp) set(nodes ...automation.Node) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nodes = nodes
}

// actions returns the actions received so far.
func (f *fakeApp) actions() []act {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]act(nil), f.acts...)
}

// events returns the input events received so far.
func (f *fakeApp) events() []automation.Event {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]automation.Event(nil), f.inputs...)
}

func (f *fakeApp) handle(method string, params map[string]any) (any, *automation.Error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch method {
	case "windows":
		return map[string]any{"windows": []map[string]any{{"id": 1, "title": "vfox+", "kind": "content"}}}, nil
	case "tree":
		if f.treeFailure != nil {
			return nil, f.treeFailure
		}
		return map[string]any{"window": 1, "nodes": f.nodes}, nil
	case "idle":
		if f.idle != nil {
			return nil, f.idle
		}
		return map[string]any{}, nil
	case "act":
		a := act{ID: params["id"].(string), Action: params["action"].(string)}
		a.Text, _ = params["text"].(string)
		f.acts = append(f.acts, a)
		if f.onAct != nil {
			f.mu.Unlock()
			failure := f.onAct(f, a)
			f.mu.Lock()
			if failure != nil {
				return nil, failure
			}
		}
		return map[string]any{}, nil
	case "input":
		events, _ := json.Marshal(params["events"])
		var parsed []automation.Event
		_ = json.Unmarshal(events, &parsed)
		f.inputs = append(f.inputs, parsed...)
		return map[string]any{}, nil
	}
	return nil, &automation.Error{Code: "unknown_method", Message: method}
}

// serve answers the connections of the listener, as the application does: a line is a
// request, and its answer is a line.
func (f *fakeApp) serve(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go func() {
			defer func() { _ = conn.Close() }()
			reader := bufio.NewReaderSize(conn, 1<<20)
			for {
				line, err := reader.ReadBytes('\n')
				if err != nil {
					return
				}
				var request struct {
					ID     json.RawMessage `json:"id"`
					Method string          `json:"method"`
					Params map[string]any  `json:"params"`
				}
				if json.Unmarshal(line, &request) != nil {
					return
				}
				result, failure := f.handle(request.Method, request.Params)
				response := map[string]any{"id": request.ID}
				if failure != nil {
					response["error"] = failure
				} else {
					response["result"] = result
				}
				data, _ := json.Marshal(response)
				_, _ = conn.Write(append(data, '\n'))
			}
		}()
	}
}

// newSession connects a session to a fake application that shows the nodes.
func newSession(t *testing.T, nodes ...automation.Node) (*Session, *fakeApp) {
	t.Helper()
	app := &fakeApp{nodes: nodes}
	dir, err := os.MkdirTemp("", "as")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "s")
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go app.serve(listener)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := automation.Dial(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return &Session{Sandbox: dir, Home: filepath.Join(dir, "home"), Vfox: "/bin/true", Log: filepath.Join(dir, "app.log"), client: client, exited: make(chan struct{})}, app
}

// node is a shorthand for the elements of a page.
func node(id, parent, role, label string, actions ...string) automation.Node {
	return automation.Node{ID: id, Parent: parent, Role: role, Label: label, Actions: actions}
}

func keyed(n automation.Node, key any) automation.Node {
	n.Key = key
	return n
}
