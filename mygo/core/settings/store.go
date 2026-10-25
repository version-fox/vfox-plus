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

package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// SettingsStore loads and saves AppSettings.
type SettingsStore interface {
	// Load loads the settings, or returns the defaults if none are stored.
	Load() AppSettings
	// Save persists settings.
	Save(settings AppSettings) error
}

// SettingsPath returns the settings file for environment.
//
// On Unix it is `$XDG_CONFIG_HOME/vfox-plus/settings.json`, where
// `XDG_CONFIG_HOME` defaults to `~/.config` as defined by the XDG Base
// Directory Specification. On Windows it is `%APPDATA%\vfox-plus\settings.json`
// (see settings_windows.go).
func SettingsPath(environment map[string]string, fallbackHome string) string {
	return settingsPath(environment, fallbackHome)
}

// JSONFileSettingsStore stores settings as JSON in a file, by default the
// one of SettingsPath.
//
// Corrupted files are ignored rather than fatal: the application then starts
// with the defaults.
type JSONFileSettingsStore struct {
	// Path is the settings file.
	Path string
}

// Load implements SettingsStore.
func (s JSONFileSettingsStore) Load() AppSettings {
	content, err := os.ReadFile(s.Path) //nolint:gosec // The path is the settings file.
	if err != nil {
		return DefaultSettings()
	}
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return DefaultSettings()
	}
	return SettingsFromJSON(data)
}

// Save implements SettingsStore.
func (s JSONFileSettingsStore) Save(settings AppSettings) error {
	// A temporary file is written first and renamed, so that a crash never
	// leaves a truncated settings file behind.
	content, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	content = append(content, '\n')
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o750); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(s.Path), ".settings-*.json")
	if err != nil {
		return err
	}
	name := temporary.Name()
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		_ = os.Remove(name)
		return err
	}
	if err := temporary.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, s.Path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}

// MemorySettingsStore keeps settings in memory. Used by tests and as a
// fallback. It is safe for use from several goroutines.
type MemorySettingsStore struct {
	mu        sync.Mutex
	settings  AppSettings
	saveCount int
}

// NewMemorySettingsStore creates a store holding settings.
func NewMemorySettingsStore(settings AppSettings) *MemorySettingsStore {
	return &MemorySettingsStore{settings: settings.Clone()}
}

// Load implements SettingsStore.
func (s *MemorySettingsStore) Load() AppSettings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.settings.Clone()
}

// Save implements SettingsStore.
func (s *MemorySettingsStore) Save(settings AppSettings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saveCount++
	s.settings = settings.Clone()
	return nil
}

// SaveCount returns how often Save was called.
func (s *MemorySettingsStore) SaveCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveCount
}
