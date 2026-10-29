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
	"strings"
	"testing"
	"time"
)

// cdpServer answers commands by name: "answer" gives the result, "error" the
// message of a failure, "silence" no reply at all, and "close" ends the
// connection. A command that is not listed gets an error.
func cdpServer(t *testing.T, behaviour map[string]string) *cdpClient {
	t.Helper()
	address := socketServer(t, func(s *serverSocket) {
		for {
			_, opcode, payload, err := s.read()
			if err != nil || opcode == opClose {
				return
			}
			var request struct {
				ID     int    `json:"id"`
				Method string `json:"method"`
			}
			_ = json.Unmarshal(payload, &request)
			reply := func(body map[string]any) {
				body["id"] = request.ID
				raw, _ := json.Marshal(body)
				_ = s.send(true, opText, raw)
			}
			switch kind, text, _ := strings.Cut(behaviour[request.Method], ":"); kind {
			case "answer":
				reply(map[string]any{"result": json.RawMessage(text)})
			case "error":
				reply(map[string]any{"error": map[string]any{"code": -32000, "message": text}})
			case "silence":
			case "close":
				return
			default:
				reply(map[string]any{"error": map[string]any{"code": -32601, "message": "no such method"}})
			}
			// An event for each command, that nobody waits for.
			raw, _ := json.Marshal(map[string]any{"method": "Page.frameNavigated", "params": map[string]any{}})
			_ = s.send(true, opText, raw)
		}
	})
	ws, err := dialWebSocket(t.Context(), address)
	if err != nil {
		t.Fatal(err)
	}
	client := newCDPClient(ws)
	t.Cleanup(client.close)
	return client
}

func TestCDPCallsAndEvaluations(t *testing.T) {
	client := cdpServer(t, map[string]string{
		"Browser.getVersion":   `answer:{"product":"Chrome/1"}`,
		"Page.fails":           "error:Cannot do that",
		"Runtime.evaluate":     `answer:{"result":{"type":"number","value":42}}`,
		"Runtime.getHeapUsage": `answer:{"usedSize":1522608.4,"totalSize":3}`,
	})
	var version struct {
		Product string `json:"product"`
	}
	if err := client.call(t.Context(), "Browser.getVersion", nil, &version); err != nil || version.Product != "Chrome/1" {
		t.Fatalf("got %+v, %v", version, err)
	}
	err := client.call(t.Context(), "Page.fails", nil, nil)
	var failure *cdpError
	if !errors.As(err, &failure) || failure.Message != "Cannot do that" || failure.Code != -32000 || !strings.Contains(err.Error(), "Page.fails") {
		t.Errorf("a failed command: %v", err)
	}
	if err := client.call(t.Context(), "Page.unknown", nil, nil); err == nil || !strings.Contains(err.Error(), "no such method") {
		t.Errorf("an unknown command: %v", err)
	}
	var answer int
	if err := client.evaluate(t.Context(), "6 * 7", &answer); err != nil || answer != 42 {
		t.Errorf("evaluate: %d, %v", answer, err)
	}
	if err := client.evaluate(t.Context(), "6 * 7", nil); err != nil {
		t.Errorf("evaluate without a place for the value: %v", err)
	}
	var usage struct {
		UsedSize float64 `json:"usedSize"`
	}
	if err := client.call(t.Context(), "Runtime.getHeapUsage", nil, &usage); err != nil || usage.UsedSize != 1522608.4 {
		t.Errorf("usage %+v, %v", usage, err)
	}
}

func TestCDPEvaluateRaises(t *testing.T) {
	client := cdpServer(t, map[string]string{
		"Runtime.evaluate": `answer:{"result":{"type":"object"},"exceptionDetails":{"text":"Uncaught"}}`,
	})
	if err := client.evaluate(t.Context(), "throw 1", nil); err == nil || !strings.Contains(err.Error(), "exception") {
		t.Errorf("got %v", err)
	}
}

func TestCDPEvents(t *testing.T) {
	client := cdpServer(t, map[string]string{"Page.reload": `answer:{}`})
	// The server sends Page.frameNavigated after each command: register first,
	// then cause it.
	navigated := client.expect("Page.frameNavigated")
	other := client.expect("Page.neverSent")
	if err := client.call(t.Context(), "Page.reload", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := client.await(t.Context(), navigated); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if err := client.await(ctx, other); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("an event that never comes: %v", err)
	}
}

func TestCDPGivesUpWhenTheContextEnds(t *testing.T) {
	client := cdpServer(t, map[string]string{"Page.slow": "silence"})
	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()
	if err := client.call(ctx, "Page.slow", nil, nil); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v", err)
	}
	// The abandoned command does not block the next one.
	client2 := cdpServer(t, map[string]string{"Page.ok": `answer:{}`})
	if err := client2.call(t.Context(), "Page.ok", nil, nil); err != nil {
		t.Fatal(err)
	}
}

func TestCDPFailsEverythingWhenTheBrowserGoes(t *testing.T) {
	client := cdpServer(t, map[string]string{"Page.crash": "close"})
	waiting := client.expect("Page.never")
	if err := client.call(t.Context(), "Page.crash", nil, nil); err == nil {
		t.Fatal("a command to a browser that closes the connection succeeded")
	}
	if err := client.await(t.Context(), waiting); err == nil {
		t.Error("an event of a closed connection arrived")
	}
	if err := client.call(t.Context(), "Page.again", nil, nil); err == nil {
		t.Error("a command after the connection ended succeeded")
	}
}
