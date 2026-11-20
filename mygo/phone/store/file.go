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

package store

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

// NewFileServerStore keeps the server record as JSON in the file at path, with
// mode 0600. The directory is created when it is missing.
func NewFileServerStore(path string) ServerStore {
	return &fileServerStore{file: file{path: path}}
}

// NewFileSessionStore keeps the session as JSON in the file at path, with mode
// 0600. The directory is created when it is missing. Give it a path of its own,
// apart from the server record.
func NewFileSessionStore(path string) SessionStore {
	return &fileSessionStore{file: file{path: path}}
}

// file reads and writes one file atomically, so that a crash leaves either the
// old content or the new one.
type file struct {
	path string
	mu   sync.Mutex
}

// read returns the content of the file. It is fs.ErrNotExist when there is no
// file.
func (f *file) read() ([]byte, error) {
	return os.ReadFile(f.path) //nolint:gosec // The path is chosen by the app, not by a user of the file.
}

// write replaces the content of the file. The temporary file is created with
// mode 0600 and renamed over the file, so the final file has that mode too.
func (f *file) write(content []byte) error {
	dir := filepath.Dir(f.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, filepath.Base(f.path)+".tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temp.Name()) }()
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), f.path)
}

// remove deletes the file; a missing file is not an error.
func (f *file) remove() error {
	err := os.Remove(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

// isBlank reports whether content holds nothing but white space.
func isBlank(content []byte) bool {
	return len(bytes.TrimSpace(content)) == 0
}

// fileServerStore is the ServerStore of a file.
type fileServerStore struct {
	file file
}

// Load implements ServerStore.
func (s *fileServerStore) Load() (Record, error) {
	s.file.mu.Lock()
	defer s.file.mu.Unlock()
	content, err := s.file.read()
	if errors.Is(err, fs.ErrNotExist) {
		return Record{}, ErrNoRecord
	}
	if err != nil {
		return Record{}, fmt.Errorf("read the saved server: %w", err)
	}
	if isBlank(content) {
		return Record{}, ErrNoRecord
	}
	var record Record
	if err := json.Unmarshal(content, &record); err != nil {
		return Record{}, fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	if record.Base == "" {
		return Record{}, fmt.Errorf("%w: the record has no address", ErrCorrupt)
	}
	if _, err := record.PinnedCertificate(); err != nil {
		return Record{}, err
	}
	return record, nil
}

// Save implements ServerStore.
func (s *fileServerStore) Save(record Record) error {
	if record.Base == "" {
		return errors.New("a server record needs the address of the server")
	}
	if _, err := record.PinnedCertificate(); err != nil {
		return err
	}
	content, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	s.file.mu.Lock()
	defer s.file.mu.Unlock()
	return s.file.write(append(content, '\n'))
}

// Forget implements ServerStore.
func (s *fileServerStore) Forget() error {
	s.file.mu.Lock()
	defer s.file.mu.Unlock()
	return s.file.remove()
}

// sessionFile is the JSON form of a session.
type sessionFile struct {
	Session string `json:"session"`
}

// fileSessionStore is the SessionStore of a file.
type fileSessionStore struct {
	file file
}

// Load implements SessionStore.
func (s *fileSessionStore) Load() (string, error) {
	s.file.mu.Lock()
	defer s.file.mu.Unlock()
	content, err := s.file.read()
	if errors.Is(err, fs.ErrNotExist) {
		return "", ErrNoSession
	}
	if err != nil {
		return "", fmt.Errorf("read the saved session: %w", err)
	}
	if isBlank(content) {
		return "", ErrNoSession
	}
	var saved sessionFile
	if err := json.Unmarshal(content, &saved); err != nil {
		return "", fmt.Errorf("%w: %w", ErrCorrupt, err)
	}
	if saved.Session == "" {
		return "", fmt.Errorf("%w: the session is empty", ErrCorrupt)
	}
	return saved.Session, nil
}

// Save implements SessionStore.
func (s *fileSessionStore) Save(session string) error {
	if session == "" {
		return errors.New("an empty session cannot be saved")
	}
	content, err := json.Marshal(sessionFile{Session: session})
	if err != nil {
		return err
	}
	s.file.mu.Lock()
	defer s.file.mu.Unlock()
	return s.file.write(content)
}

// Forget implements SessionStore.
func (s *fileSessionStore) Forget() error {
	s.file.mu.Lock()
	defer s.file.mu.Unlock()
	return s.file.remove()
}
