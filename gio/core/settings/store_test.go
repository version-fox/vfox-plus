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

package settings_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox-plus/gio/core/settings"
)

func TestSettingsPathFollowsTheXDGDirectories(t *testing.T) {
	for name, tc := range map[string]struct {
		environment map[string]string
		want        string
	}{
		"xdg":      {map[string]string{"XDG_CONFIG_HOME": "/cfg", "HOME": "/h"}, "/cfg/vfox-plus/settings.json"},
		"relative": {map[string]string{"XDG_CONFIG_HOME": "cfg", "HOME": "/h"}, "/h/.config/vfox-plus/settings.json"},
		"home":     {map[string]string{"HOME": "/home/u"}, "/home/u/.config/vfox-plus/settings.json"},
		"fallback": {map[string]string{}, "/fallback/.config/vfox-plus/settings.json"},
	} {
		if got := settings.SettingsPath(tc.environment, "/fallback"); got != tc.want {
			t.Errorf("%s: got %q, want %q", name, got, tc.want)
		}
	}
}

func TestFileStoreSavesAtomicallyAndLoadsWhatItSaved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config", "vfox-plus", "settings.json")
	store := settings.JSONFileSettingsStore{Path: path}
	if loaded := store.Load(); loaded.Theme != settings.ThemeSystem {
		t.Fatalf("a missing file gives the defaults, got %#v", loaded)
	}
	changed := settings.DefaultSettings()
	changed.Theme = settings.ThemeDark
	changed.RailExpanded = true
	changed.ExtraEnvironment = map[string]string{"A": "1"}
	changed.Unknown = map[string]interface{}{"future": true}
	if err := store.Save(changed); err != nil {
		t.Fatal(err)
	}
	loaded := store.Load()
	if loaded.Theme != settings.ThemeDark || !loaded.RailExpanded || loaded.ExtraEnvironment["A"] != "1" || loaded.Unknown["future"] != true {
		t.Errorf("got %#v", loaded)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Errorf("no temporary file is left behind: %v %v", entries, err)
	}
}

func TestFileStoreFailsWhereItCannotWrite(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	// A directory cannot be made inside a file.
	store := settings.JSONFileSettingsStore{Path: filepath.Join(blocker, "settings.json")}
	if err := store.Save(settings.DefaultSettings()); err == nil {
		t.Error("expected an error")
	}
}

func TestMemoryStoreKeepsACopy(t *testing.T) {
	initial := settings.DefaultSettings()
	store := settings.NewMemorySettingsStore(initial)
	initial.ExtraEnvironment["A"] = "1"
	if len(store.Load().ExtraEnvironment) != 0 {
		t.Error("the store holds a copy")
	}
}
