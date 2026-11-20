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

// Package store keeps what the phone keeps between runs: the saved server, and
// the session with it. The two are kept apart. The server record is not secret
// (a name, an address and a fingerprint); the session is a credential, and it
// is never written into the record.
//
// The file implementations in this package serve Linux, the tests and the
// Android build. On Android they keep their files in the private directory of
// the app, which no other app can read, and the session file has the mode 0600
// as on Linux. The secret storage of mygo (the Android keystore) is not used
// yet; an implementation on it must keep the same contract: Load reports
// ErrNoRecord or ErrNoSession when nothing is saved, Forget is idempotent, and
// a damaged value is ErrCorrupt.
package store

import "errors"

var (
	// ErrNoRecord reports that no server is saved.
	ErrNoRecord = errors.New("no server is saved")
	// ErrNoSession reports that no session is saved.
	ErrNoSession = errors.New("no session is saved")
	// ErrCorrupt reports saved data that cannot be read. A caller treats it as
	// nothing saved, and the next Save replaces it.
	ErrCorrupt = errors.New("the saved data is damaged")
)

// Record is a saved server: the name the user knows it by, its address and the
// fingerprint of its certificate that the user confirmed. It holds no secret.
type Record struct {
	// Name is the name that the user gave the server, or its host.
	Name string `json:"name"`
	// Base is the address of the server, https://host:port.
	Base string `json:"base"`
	// Fingerprint is the fingerprint of the server's certificate that the user
	// confirmed, in the form pairing.FormatFingerprint gives.
	Fingerprint string `json:"fingerprint"`
}

// ServerStore keeps the one saved server. The phone keeps one server only, which
// is how the Kotlin app of the desktop works too.
type ServerStore interface {
	// Load returns the saved record. It is ErrNoRecord when none is saved, and
	// it wraps ErrCorrupt when the saved data is damaged.
	Load() (Record, error)
	// Save replaces the saved record.
	Save(record Record) error
	// Forget removes the saved record. Forgetting nothing is not an error.
	Forget() error
}

// SessionStore keeps the session of the saved server. The session is a secret:
// implementations keep it apart from the record and private to the app.
type SessionStore interface {
	// Load returns the saved session. It is ErrNoSession when none is saved, and
	// it wraps ErrCorrupt when the saved data is damaged.
	Load() (string, error)
	// Save replaces the saved session. An empty session is an error.
	Save(session string) error
	// Forget removes the saved session. Forgetting nothing is not an error.
	Forget() error
}
