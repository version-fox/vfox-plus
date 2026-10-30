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

package automation

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// handler answers a request of a fake application: its result, or its failure.
type handler func(method string, params map[string]any) (result any, failure *Error)

// socketPath is a path for a socket that is short enough for the system.
func socketPath(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "au")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return filepath.Join(dir, "s")
}

// listen serves the protocol on the socket, as the application does: a line is a
// request, and its answer is a line.
func listen(t *testing.T, path string, answer handler) {
	t.Helper()
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	go func() {
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
					if err := json.Unmarshal(line, &request); err != nil {
						return
					}
					result, failure := answer(request.Method, request.Params)
					response := map[string]any{"id": json.RawMessage(request.ID)}
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
	}()
}

// dial connects to a fake application.
func dial(t *testing.T, answer handler) *Client {
	t.Helper()
	path := socketPath(t)
	listen(t, path, answer)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := Dial(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	return client
}

func TestCallDecodesTheResultAndTheFailure(t *testing.T) {
	client := dial(t, func(method string, params map[string]any) (any, *Error) {
		switch method {
		case "echo":
			return params, nil
		case "refuse":
			return nil, &Error{Code: "no_element", Message: "no element 7"}
		}
		return nil, &Error{Code: "unknown_method", Message: method}
	})
	var echoed map[string]any
	if err := client.Call("echo", map[string]any{"a": 1}, &echoed); err != nil || echoed["a"] != float64(1) {
		t.Fatalf("echo = %v, %v", echoed, err)
	}
	err := client.Call("refuse", nil, nil)
	if !IsCode(err, "no_element") || !strings.Contains(err.Error(), "no element 7") {
		t.Fatalf("refuse = %v, want the failure with its code and message", err)
	}
	if IsCode(err, "timeout") || IsCode(nil, "no_element") {
		t.Fatal("IsCode answers yes to another code or to no failure")
	}
	// A failure does not end the connection.
	if err := client.Call("echo", nil, nil); err != nil {
		t.Fatalf("after a failure: %v", err)
	}
}

func TestCallRefusesAnAnswerToAnotherRequest(t *testing.T) {
	path := socketPath(t)
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = bufio.NewReader(conn).ReadBytes('\n')
		_, _ = conn.Write([]byte("{\"id\":99,\"result\":{}}\n"))
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := Dial(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Call("windows", nil, nil); err == nil || !strings.Contains(err.Error(), "request 99") {
		t.Fatalf("got %v, want a refusal of the answer for request 99", err)
	}
}

func TestCallReportsAClosedConnection(t *testing.T) {
	path := socketPath(t)
	listener, err := net.Listen("unix", path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = listener.Close() }()
	go func() {
		if conn, err := listener.Accept(); err == nil {
			_ = conn.Close()
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := Dial(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Call("tree", nil, nil); err == nil || !strings.HasPrefix(err.Error(), "tree: ") {
		t.Fatalf("got %v, want the method with the failure", err)
	}
}

func TestDialWaitsForTheSocketAndGivesUp(t *testing.T) {
	path := socketPath(t)
	go func() {
		time.Sleep(200 * time.Millisecond)
		listen(t, path, func(string, map[string]any) (any, *Error) { return map[string]any{}, nil })
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	client, err := Dial(ctx, path)
	if err != nil {
		t.Fatalf("Dial did not wait for the application to listen: %v", err)
	}
	_ = client.Close()

	ctx, cancel = context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	missing := filepath.Join(filepath.Dir(path), "missing")
	if _, err := Dial(ctx, missing); err == nil || !strings.Contains(err.Error(), missing) {
		t.Fatalf("got %v, want a failure that names the socket", err)
	}
}

func TestWaitForWindowWaitsForTheFirst(t *testing.T) {
	calls := 0
	client := dial(t, func(method string, _ map[string]any) (any, *Error) {
		calls++
		switch {
		case calls == 1:
			return nil, &Error{Code: "no_window", Message: "the app has no window"}
		case calls == 2:
			return map[string]any{"windows": []any{}}, nil
		}
		return map[string]any{"windows": []Window{{ID: 3, Title: "vfox+", Kind: "content", Focused: true, Visible: true}}}, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	window, err := client.WaitForWindow(ctx)
	if err != nil || window.ID != 3 || window.Title != "vfox+" || calls != 3 {
		t.Fatalf("window = %+v, %v after %d calls", window, err, calls)
	}

	// Another failure is not waited out.
	failing := dial(t, func(string, map[string]any) (any, *Error) { return nil, &Error{Code: "failed", Message: "boom"} })
	if _, err := failing.WaitForWindow(ctx); !IsCode(err, "failed") {
		t.Fatalf("got %v, want the failure", err)
	}
	// And the wait ends with the context.
	empty := dial(t, func(string, map[string]any) (any, *Error) { return map[string]any{"windows": []any{}}, nil })
	short, stop := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer stop()
	if _, err := empty.WaitForWindow(short); err == nil || !strings.Contains(err.Error(), "no window") {
		t.Fatalf("got %v, want a failure that says no window opened", err)
	}
}

func TestRequestsCarryWhatTheProtocolAsks(t *testing.T) {
	var methods []string
	var seen []map[string]any
	client := dial(t, func(method string, params map[string]any) (any, *Error) {
		methods = append(methods, method)
		seen = append(seen, params)
		return map[string]any{}, nil
	})
	if err := client.Act("12", Press, ""); err != nil {
		t.Fatal(err)
	}
	if err := client.Act("12", SetValue, ""); err != nil {
		t.Fatal(err)
	}
	if err := client.Act("12", InsertText, "x"); err != nil {
		t.Fatal(err)
	}
	if err := client.Input(Event{Type: "key", Key: "Enter"}, Event{Type: "click", X: 3, Y: 4}); err != nil {
		t.Fatal(err)
	}
	if err := client.Idle(1500 * time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if err := client.Idle(0); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(methods, " "); got != "act act act input idle idle" {
		t.Fatalf("methods = %q", got)
	}
	if seen[0]["id"] != "12" || seen[0]["action"] != "press" || seen[0]["text"] != nil {
		t.Errorf("a press carries no text: %v", seen[0])
	}
	if text, present := seen[1]["text"]; !present || text != "" {
		t.Errorf("setValue empties the field with an empty text, and says so: %v", seen[1])
	}
	if seen[2]["text"] != "x" {
		t.Errorf("insertText carries its text: %v", seen[2])
	}
	events, _ := seen[3]["events"].([]any)
	if len(events) != 2 || events[0].(map[string]any)["key"] != "Enter" || events[1].(map[string]any)["x"] != float64(3) {
		t.Errorf("input events = %v", seen[3]["events"])
	}
	if seen[4]["timeout"] != float64(1500) || seen[5]["timeout"] != nil {
		t.Errorf("idle timeouts = %v, %v", seen[4]["timeout"], seen[5]["timeout"])
	}
}

func TestIdleTimeoutIsAFailureWithItsCode(t *testing.T) {
	client := dial(t, func(string, map[string]any) (any, *Error) {
		return nil, &Error{Code: "timeout", Message: "window 1 still draws after 1.5s"}
	})
	if err := client.Idle(time.Second); !IsCode(err, "timeout") {
		t.Fatalf("got %v, want the code timeout", err)
	}
}

func TestTreeAndWindowsAreDecoded(t *testing.T) {
	client := dial(t, func(method string, _ map[string]any) (any, *Error) {
		switch method {
		case "windows":
			return map[string]any{"windows": []map[string]any{{
				"id": 1, "title": "vfox+", "kind": "content", "focused": true, "visible": true,
				"bounds": map[string]any{"x": 26, "y": 23, "width": 1280, "height": 820},
			}}}, nil
		case "tree":
			return map[string]any{"window": 1, "focus": "9", "nodes": []map[string]any{
				{"id": "9", "role": "button", "key": 4, "label": "Settings", "states": []string{"focusable"}, "actions": []string{"press", "focus"}},
				{"id": "10", "parent": "9", "role": "progress", "key": "progress", "label": "Install x", "min": 0, "max": 1, "now": 0.25},
			}}, nil
		}
		return nil, &Error{Code: "unknown_method", Message: method}
	})
	windows, err := client.Windows()
	if err != nil || len(windows) != 1 || windows[0].Bounds.Width != 1280 || !windows[0].Focused {
		t.Fatalf("windows = %+v, %v", windows, err)
	}
	tree, err := client.Tree()
	if err != nil || tree.Window != 1 || tree.Focus != "9" || len(tree.Nodes) != 2 {
		t.Fatalf("tree = %+v, %v", tree, err)
	}
	button, bar := tree.Nodes[0], tree.Nodes[1]
	if button.KeyText() != "4" || !button.CanDo(Press) || button.CanDo(SetValue) || !button.Has("focusable") || button.Has("disabled") {
		t.Errorf("button = %+v", button)
	}
	if bar.Parent != "9" || bar.KeyText() != "progress" || bar.Now == nil || *bar.Now != 0.25 {
		t.Errorf("bar = %+v", bar)
	}
}

func TestCaptureDecodesThePNG(t *testing.T) {
	image := []byte("\x89PNG\r\n\x1a\n not really an image, but long enough to need base64 padding")
	client := dial(t, func(string, map[string]any) (any, *Error) {
		return map[string]any{"png": base64.StdEncoding.EncodeToString(image), "width": 2560, "height": 1640}, nil
	})
	got, width, height, err := client.Capture()
	if err != nil || string(got) != string(image) || width != 2560 || height != 1640 {
		t.Fatalf("capture = %q, %d x %d, %v", got, width, height, err)
	}
	bad := dial(t, func(string, map[string]any) (any, *Error) { return map[string]any{"png": "***"}, nil })
	if _, _, _, err := bad.Capture(); err == nil {
		t.Fatal("a capture that is no base64 was accepted")
	}
}

func TestALongAnswerIsRead(t *testing.T) {
	// A capture of a window is longer than what a line scanner takes by default.
	big := strings.Repeat("a", 5<<20)
	client := dial(t, func(string, map[string]any) (any, *Error) { return map[string]any{"value": big}, nil })
	var result struct {
		Value string `json:"value"`
	}
	if err := client.Call("eval", nil, &result); err != nil || len(result.Value) != len(big) {
		t.Fatalf("read %d bytes, want %d: %v", len(result.Value), len(big), err)
	}
}
