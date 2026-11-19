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

//go:build e2e

package pair

import "sync"

// loopbackState is the server of this computer, on the loopback interface, that
// the end-to-end tests pair their browsers with. Only the end-to-end build has
// it: the application shows no web interface on this computer.
type loopbackState struct {
	mu     sync.Mutex
	server *Server
}

// LocalLink makes a pairing link for a browser on this computer, and starts the
// loopback server of the tests on first use. The tools of the tests read the link
// from the file that VFOX_PLUS_E2E_LINK names.
func (h *Host) LocalLink() (string, error) {
	h.loopback.mu.Lock()
	defer h.loopback.mu.Unlock()
	if h.loopback.server == nil {
		server, err := NewServer(Options{Model: h.model, Control: h})
		if err != nil {
			return "", err
		}
		h.loopback.server = server
	}
	_, link, err := h.loopback.server.NewToken()
	return link, err
}

// close ends the loopback server of the tests, if it was started.
func (l *loopbackState) close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.server != nil {
		_ = l.server.Close()
		l.server = nil
	}
}
