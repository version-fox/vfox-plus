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

package pair_test

import (
	"net"
	"runtime"
	"strconv"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
)

// The port of a server that was closed is free at once, and not when the goroutine that serves gets to run: the
// host stops the server for other devices and starts it again on the same port whenever the certificate or the
// port changes, and a port that is still taken for a moment says "address already in use" to a server that has
// done nothing wrong. http.Server.Close closes the listeners that Serve has registered, and a Close that comes
// before the goroutine that NewServer made has started Serve finds none.
func TestAClosedServerGivesItsPortBackAtOnce(t *testing.T) {
	// One processor, so that the goroutine that serves runs only when the test gives way, and the test
	// does not depend on how fast the machine schedules it.
	previous := runtime.GOMAXPROCS(1)
	defer runtime.GOMAXPROCS(previous)

	model := testModel(t, testsupport.FixtureRunner())
	port := testsupport.FreePort(t)
	for attempt := range 200 {
		server, err := pair.NewServer(pair.Options{Model: model, Port: port})
		if err != nil {
			t.Fatalf("attempt %d: start the server: %v", attempt, err)
		}
		if err := server.Close(); err != nil {
			t.Fatalf("attempt %d: close the server: %v", attempt, err)
		}
		listener, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			t.Fatalf("attempt %d: the port of the server that was closed is still taken: %v", attempt, err)
		}
		_ = listener.Close()
	}
}

// A server that is closed twice, as the host does when it is closed with a server that it already stopped, is
// no failure, and its listener is not closed under another that took its port.
func TestAServerCanBeClosedTwice(t *testing.T) {
	server, err := pair.NewServer(pair.Options{Model: testModel(t, testsupport.FixtureRunner())})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Close(); err != nil {
		t.Fatalf("the first Close failed: %v", err)
	}
	if err := server.Close(); err != nil {
		t.Errorf("the second Close failed: %v", err)
	}
}
