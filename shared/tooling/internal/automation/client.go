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

// Package automation is a client of the automation plugin of mygo
// (github.com/egoist/mygo/plugins/automation, docs/plugins/automation.md in that
// module): it drives the built application from outside, as an assistive
// technology does. The application listens on a Unix domain socket when it is
// built with the tag e2e and MYGO_AUTOMATION names the socket; the protocol is
// one JSON object per line, answered in order.
//
// The end-to-end scenarios, the memory measurement and the screenshots drive
// the application through it; no scenario code is in the application.
package automation

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
)

// Error is a failure that the application reports, with the code of the
// protocol (no_window, no_element, unsupported, timeout, failed, ...).
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// IsCode reports whether err is a failure of the application with the code.
func IsCode(err error, code string) bool {
	var failure *Error
	return errors.As(err, &failure) && failure.Code == code
}

// maxAnswer is the longest answer a line may hold: a capture of a window is a
// PNG image in base64.
const maxAnswer = 64 << 20

// Client is a connection to the application. A client is for one goroutine at a
// time: the application answers the requests of a connection in order.
type Client struct {
	mu     sync.Mutex
	conn   net.Conn
	reader *bufio.Reader
	next   int
}

// Dial connects to the socket at path. The application makes the socket when it
// starts, so Dial waits for it until the context ends.
func Dial(ctx context.Context, path string) (*Client, error) {
	var dialer net.Dialer
	for {
		conn, err := dialer.DialContext(ctx, "unix", path)
		if err == nil {
			return &Client{conn: conn, reader: bufio.NewReaderSize(conn, 1<<20)}, nil
		}
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("connect to the application at %s: %w (the last attempt: %v)", path, ctx.Err(), err)
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// Close closes the connection.
func (c *Client) Close() error { return c.conn.Close() }

// Call sends a request and decodes the result of its answer into result, if it
// is not nil. A failure that the application reports is an *Error.
func (c *Client) Call(method string, params, result any) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.next++
	id := c.next
	line, err := json.Marshal(struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params,omitempty"`
	}{id, method, params})
	if err != nil {
		return err
	}
	if _, err := c.conn.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	answer, err := c.readLine()
	if err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	var response struct {
		ID     int             `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *Error          `json:"error"`
	}
	if err := json.Unmarshal(answer, &response); err != nil {
		return fmt.Errorf("%s: the answer is no JSON: %w", method, err)
	}
	if response.Error != nil {
		return fmt.Errorf("%s: %w", method, response.Error)
	}
	if response.ID != id {
		return fmt.Errorf("%s: the answer is for request %d, not %d", method, response.ID, id)
	}
	if result == nil {
		return nil
	}
	if err := json.Unmarshal(response.Result, result); err != nil {
		return fmt.Errorf("%s: %w", method, err)
	}
	return nil
}

// readLine reads one answer, however long.
func (c *Client) readLine() ([]byte, error) {
	var line []byte
	for {
		part, more, err := c.reader.ReadLine()
		if err != nil {
			return nil, err
		}
		line = append(line, part...)
		if len(line) > maxAnswer {
			return nil, fmt.Errorf("an answer is longer than %d bytes", maxAnswer)
		}
		if !more {
			return line, nil
		}
	}
}

// Rect is a position and a size, in DIPs.
type Rect struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Window is an open window of the application.
type Window struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	// Kind is "content" for native UI and "page" for a web page.
	Kind          string `json:"kind"`
	URL           string `json:"url"`
	Bounds        Rect   `json:"bounds"`
	ContentBounds Rect   `json:"contentBounds"`
	Focused       bool   `json:"focused"`
	Visible       bool   `json:"visible"`
}

// Windows lists the open windows.
func (c *Client) Windows() ([]Window, error) {
	var result struct {
		Windows []Window `json:"windows"`
	}
	if err := c.Call("windows", nil, &result); err != nil {
		return nil, err
	}
	return result.Windows, nil
}

// WaitForWindow waits until the application has a window, and returns the first.
func (c *Client) WaitForWindow(ctx context.Context) (Window, error) {
	for {
		windows, err := c.Windows()
		if err == nil && len(windows) > 0 {
			return windows[0], nil
		}
		if err != nil && !IsCode(err, "no_window") {
			return Window{}, err
		}
		select {
		case <-ctx.Done():
			return Window{}, fmt.Errorf("the application opened no window: %w", ctx.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}
}

// Idle waits until the window has drawn the frames it asked for. A window that
// never settles, as one with an animation that keeps going, fails with the code
// timeout after the timeout.
func (c *Client) Idle(timeout time.Duration) error {
	params := map[string]any{}
	if timeout > 0 {
		params["timeout"] = timeout.Milliseconds()
	}
	return c.Call("idle", params, nil)
}

// Action is what Act does to an element, as assistive technology does.
type Action string

// The actions of the protocol that the scenarios use.
const (
	Press          Action = "press"
	Focus          Action = "focus"
	SetValue       Action = "setValue"
	InsertText     Action = "insertText"
	ScrollIntoView Action = "scrollIntoView"
)

// Act performs an action on the element with the ID. text is for SetValue and
// InsertText.
func (c *Client) Act(id string, action Action, text string) error {
	params := map[string]any{"id": id, "action": action}
	if text != "" || action == SetValue {
		params["text"] = text
	}
	return c.Call("act", params, nil)
}

// Event is a pointer, wheel, key or text event of Input.
type Event struct {
	Type string  `json:"type"`
	X    float64 `json:"x,omitempty"`
	Y    float64 `json:"y,omitempty"`
	Text string  `json:"text,omitempty"`
	// Key is a key with its modifiers as menu accelerators write them, such as
	// Enter or CmdOrCtrl+A.
	Key string `json:"key,omitempty"`
}

// Input sends events to the content of the window, in order.
func (c *Client) Input(events ...Event) error {
	return c.Call("input", map[string]any{"events": events}, nil)
}

// Capture returns the content of the window as a PNG image.
func (c *Client) Capture() (png []byte, width, height int, err error) {
	var result struct {
		PNG    string `json:"png"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}
	if err := c.Call("capture", nil, &result); err != nil {
		return nil, 0, 0, err
	}
	data, err := base64.StdEncoding.DecodeString(result.PNG)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("capture: %w", err)
	}
	return data, result.Width, result.Height, nil
}

// Node is an element of the window as assistive technology reads it.
type Node struct {
	// ID changes with the frames; Key is the name that the application gave the
	// element (a string, a boolean or a number as itself).
	ID          string   `json:"id"`
	Parent      string   `json:"parent"`
	Role        string   `json:"role"`
	Key         any      `json:"key"`
	Label       string   `json:"label"`
	Value       string   `json:"value"`
	Placeholder string   `json:"placeholder"`
	Description string   `json:"description"`
	Bounds      Rect     `json:"bounds"`
	States      []string `json:"states"`
	Actions     []string `json:"actions"`
	// Min, Max and Now are those of a progress bar, a slider or a stepper.
	Min *float64 `json:"min"`
	Max *float64 `json:"max"`
	Now *float64 `json:"now"`
}

// KeyText is the key of the node as text, "" when it has none.
func (n Node) KeyText() string {
	switch key := n.Key.(type) {
	case nil:
		return ""
	case string:
		return key
	case float64:
		return strconv.FormatFloat(key, 'f', -1, 64)
	default:
		return fmt.Sprint(key)
	}
}

// Has reports whether the node has the state, such as "disabled" or "selected".
func (n Node) Has(state string) bool {
	for _, have := range n.States {
		if have == state {
			return true
		}
	}
	return false
}

// CanDo reports whether the node takes the action.
func (n Node) CanDo(action Action) bool {
	for _, can := range n.Actions {
		if can == string(action) {
			return true
		}
	}
	return false
}

// Tree is the elements of the last frame of a window, in tree order: a parent
// before its children.
type Tree struct {
	Window int    `json:"window"`
	Focus  string `json:"focus"`
	Nodes  []Node `json:"nodes"`
}

// Tree reads the elements of the window.
func (c *Client) Tree() (Tree, error) {
	var tree Tree
	err := c.Call("tree", nil, &tree)
	return tree, err
}
