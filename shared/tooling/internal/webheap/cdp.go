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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// cdpError is the error object of a failed Chrome DevTools Protocol command.
type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *cdpError) Error() string { return fmt.Sprintf("%s (%d)", e.Message, e.Code) }

// cdpMessage is what Chrome sends: the response to a command (with its id) or
// an event (with a method).
type cdpMessage struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Result json.RawMessage `json:"result"`
	Error  *cdpError       `json:"error"`
}

// cdpClient talks to one page target: it matches responses to the commands
// that wait for them and lets a caller wait for the next event of a kind.
// Other events are dropped, which is all that Page and Runtime need here.
type cdpClient struct {
	ws *wsConn

	mu      sync.Mutex
	nextID  int
	pending map[int]chan cdpMessage
	waiting map[string][]chan struct{}
	failure error
	done    chan struct{} // closed once the connection has failed
}

func newCDPClient(ws *wsConn) *cdpClient {
	client := &cdpClient{
		ws:      ws,
		pending: map[int]chan cdpMessage{},
		waiting: map[string][]chan struct{}{},
		done:    make(chan struct{}),
	}
	go client.read()
	return client
}

// read dispatches messages until the connection fails.
func (c *cdpClient) read() {
	for {
		raw, err := c.ws.readMessage()
		if err != nil {
			c.fail(err)
			return
		}
		var message cdpMessage
		if json.Unmarshal(raw, &message) != nil {
			continue
		}
		c.mu.Lock()
		switch {
		case message.ID != 0:
			reply := c.pending[message.ID]
			delete(c.pending, message.ID)
			c.mu.Unlock()
			if reply != nil {
				reply <- message
			}
		case message.Method != "":
			waiters := c.waiting[message.Method]
			delete(c.waiting, message.Method)
			c.mu.Unlock()
			for _, waiter := range waiters {
				close(waiter)
			}
		default:
			c.mu.Unlock()
		}
	}
}

// fail records why the connection ended and wakes everyone who waits.
func (c *cdpClient) fail(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.failure != nil {
		return
	}
	c.failure = err
	close(c.done)
}

func (c *cdpClient) failed() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.failure
}

func (c *cdpClient) forget(id int) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

// call runs a command and decodes its result into result (when it is not nil).
func (c *cdpClient) call(ctx context.Context, method string, params, result any) error {
	c.mu.Lock()
	if c.failure != nil {
		err := c.failure
		c.mu.Unlock()
		return err
	}
	c.nextID++
	id := c.nextID
	reply := make(chan cdpMessage, 1)
	c.pending[id] = reply
	c.mu.Unlock()

	if params == nil {
		params = struct{}{}
	}
	raw, err := json.Marshal(struct {
		ID     int    `json:"id"`
		Method string `json:"method"`
		Params any    `json:"params"`
	}{id, method, params})
	if err != nil {
		c.forget(id)
		return err
	}
	if err := c.ws.writeFrame(opText, raw); err != nil {
		c.forget(id)
		return err
	}
	select {
	case message := <-reply:
		if message.Error != nil {
			return fmt.Errorf("%s: %w", method, message.Error)
		}
		if result == nil || len(message.Result) == 0 {
			return nil
		}
		return json.Unmarshal(message.Result, result)
	case <-c.done:
		return c.failed()
	case <-ctx.Done():
		c.forget(id)
		return ctx.Err()
	}
}

// expect registers for the next event of the kind before the command that
// causes it is sent, so that the event cannot be missed. The channel closes
// when the event arrives.
func (c *cdpClient) expect(method string) <-chan struct{} {
	ready := make(chan struct{})
	c.mu.Lock()
	c.waiting[method] = append(c.waiting[method], ready)
	c.mu.Unlock()
	return ready
}

// await waits for an event that expect registered.
func (c *cdpClient) await(ctx context.Context, ready <-chan struct{}) error {
	select {
	case <-ready:
		return nil
	case <-c.done:
		return c.failed()
	case <-ctx.Done():
		return ctx.Err()
	}
}

// evaluate runs a JavaScript expression in the page and decodes its value into
// result (when it is not nil and the expression has a value).
func (c *cdpClient) evaluate(ctx context.Context, expression string, result any) error {
	var response struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
		ExceptionDetails json.RawMessage `json:"exceptionDetails"`
	}
	err := c.call(ctx, "Runtime.evaluate", map[string]any{"expression": expression, "returnByValue": true}, &response)
	if err != nil {
		return err
	}
	if len(response.ExceptionDetails) > 0 {
		return errors.New("the page raised an exception")
	}
	if result == nil || len(response.Result.Value) == 0 {
		return nil
	}
	return json.Unmarshal(response.Result.Value, result)
}

// close ends the connection.
func (c *cdpClient) close() {
	c.ws.close()
	c.fail(errors.New("closed"))
}
