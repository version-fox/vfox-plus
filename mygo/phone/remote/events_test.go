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
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestEventsDeliverTheHelloOfTheServerAndTheTasks(t *testing.T) {
	s := startServer(t)
	client, _ := s.paired(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	kinds := make(chan string, 64)
	done := make(chan error, 1)
	go func() { done <- client.Events(ctx, func(kind string) { kinds <- kind }) }()

	waitFor := func(want string) {
		t.Helper()
		deadline := time.After(5 * time.Second)
		for {
			select {
			case kind := <-kinds:
				if kind == want {
					return
				}
				if want == EventHello {
					t.Fatalf("the first event is %q, want hello", kind)
				}
			case err := <-done:
				t.Fatalf("Events ended before %q: %v", want, err)
			case <-deadline:
				t.Fatalf("no %q event", want)
			}
		}
	}
	waitFor(EventHello)

	// Starting a task tells the stream about it.
	if _, err := client.StartTask(ctx, TaskRequest{Operation: RequestInstall, Sdk: "nodejs", Version: "22.21.0"}); err != nil {
		t.Fatal(err)
	}
	waitFor(EventTask)

	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("Events after cancel = %v, want context.Canceled", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Events did not end after its context was cancelled")
	}
}

func TestEventsRefuseAnUnpairedStream(t *testing.T) {
	s := startServer(t)
	client, _ := s.client(t, "")
	err := client.Events(t.Context(), func(string) {})
	if !errors.Is(err, ErrUnpaired) {
		t.Fatalf("Events without a session = %v, want ErrUnpaired", err)
	}
}

func TestEventsIgnoreHeartbeatsAndEndWithTheStream(t *testing.T) {
	// The stream of a server on the loopback interface: heartbeat comments, events
	// with and without data, and CRLF line endings.
	const stream = ": ping\n\n" +
		"event: task\ndata: {}\n\n" +
		": ping\n\n" +
		"event: vfox\ndata: {}\n\n" +
		"event: nodata\n\n" +
		"event: web\r\ndata: {}\r\n\r\n"
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		cookie, err := request.Cookie("vfox-plus-session")
		if err != nil || cookie.Value != "pretend" {
			http.Error(writer, "Not paired.", http.StatusUnauthorized)
			return
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(writer, stream)
	}))
	defer server.Close()
	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client, err := New(base, "", "pretend")
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	if err := client.Events(t.Context(), func(kind string) { got = append(got, kind) }); err != nil {
		t.Fatalf("Events of a stream that ends: %v", err)
	}
	if want := []string{EventTask, EventVfox, EventWeb}; !reflect.DeepEqual(got, want) {
		t.Fatalf("the events are %v, want %v", got, want)
	}
}
