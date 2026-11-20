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

package remote

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Events kinds of the server.
const (
	EventHello     = "hello"
	EventTask      = "task"
	EventInventory = "inventory"
	EventVfox      = "vfox"
	EventWeb       = "web"
)

// Events streams the server-sent events of the server (GET /api/events) and calls
// onEvent with the kind of each one, on the goroutine of the call. The stream
// opens with EventHello. The events carry no data that the phone needs: it
// refetches what the kind names. Heartbeat comments are ignored.
//
// Events returns ctx.Err() when ctx ends, nil when the server closes the stream,
// and another error when the connection breaks or the server refuses the stream
// (an *APIError; ErrUnpaired when the session is no longer accepted). The caller
// reconnects after a pause, such as three seconds, unless ctx has ended.
func (c *Client) Events(ctx context.Context, onEvent func(kind string)) error {
	if onEvent == nil {
		onEvent = func(string) {}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint("/api/events", nil), nil)
	if err != nil {
		return fmt.Errorf("remote: make the request: %w", err)
	}
	request.Header.Set("Accept", "text/event-stream")
	c.authorize(request)
	response, err := c.streamer.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return requestError(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		answer, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		return newAPIError(response.StatusCode, answer)
	}
	return readEvents(ctx, response.Body, onEvent)
}

// readEvents parses the server-sent events of body. An event is a block of
// lines ended by an empty line; it is delivered when it has a kind and data, as
// the format prescribes. Lines that start with a colon are comments.
func readEvents(ctx context.Context, body io.Reader, onEvent func(kind string)) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 4<<10), 64<<10)
	kind := ""
	hasData := false
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if kind != "" && hasData {
				onEvent(kind)
			}
			kind, hasData = "", false
		case strings.HasPrefix(line, ":"):
			// A comment: the heartbeat of the server.
		case strings.HasPrefix(line, "event:"):
			kind = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			hasData = true
		}
	}
	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("remote: the event stream broke: %w", err)
	}
	return ctx.Err()
}
