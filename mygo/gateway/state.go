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

package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// stateFileName is the file of the directory that holds what the gateway remembers of the instance.
const stateFileName = "gateway.json"

// saved is what the gateway remembers of the instance it is paired with, so that a gateway that is
// started again is paired still, for as long as the session at the instance lasts.
type saved struct {
	// Upstream is the address of the instance, without a path: https://192.168.1.20:8369.
	Upstream string `json:"upstream"`
	// Fingerprint is the SHA-256 fingerprint of the certificate of the instance that its owner compared
	// and the gateway trusts, in the form that the settings of the application show. It is empty when the
	// certificate verifies, with the roots of the system, and for the plain HTTP of a loopback address.
	Fingerprint string `json:"fingerprint,omitempty"`
	// Session is the session cookie that the instance gave the gateway: a secret, and why the file is
	// private. It is empty after the instance has ended the session.
	Session   string    `json:"session,omitempty"`
	RenewedAt time.Time `json:"renewedAt,omitzero"`
}

// loadState reads what the gateway remembers. A file that is missing or cannot be read is a gateway
// that is paired with nothing, as it is the first time.
func loadState(dir string) saved {
	content, err := os.ReadFile(filepath.Join(dir, stateFileName)) //nolint:gosec // The file of the gateway, in its directory.
	if err != nil {
		return saved{}
	}
	var state saved
	if err := json.Unmarshal(content, &state); err != nil {
		return saved{}
	}
	return state
}

// saveState writes what the gateway remembers, whole or not at all.
func saveState(dir string, state saved) error {
	content, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(dir, stateFileName+".*")
	if err != nil {
		return err
	}
	name := file.Name()
	if _, err := file.Write(append(content, '\n')); err != nil {
		_ = file.Close()
		_ = os.Remove(name)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, filepath.Join(dir, stateFileName)); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
