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
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/gio/core/settings"
)

func decode(t *testing.T, text string) interface{} {
	t.Helper()
	var data interface{}
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatal(err)
	}
	return data
}

func TestSettingsRoundTrip(t *testing.T) {
	original := settings.AppSettings{
		Theme: settings.ThemeDark, UseSystemColor: false, SeedColor: 0xFF123456,
		Language:         settings.LanguageSimplifiedChinese,
		VfoxExecutable:   "/opt/vfox",
		ExtraEnvironment: map[string]string{"A": "1"},
		RailExpanded:     true,
		Unknown:          map[string]interface{}{"future": true},
	}
	raw, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	if got := settings.SettingsFromJSON(decode(t, string(raw))); !got.Equal(original) {
		t.Errorf("got %#v", got)
	}
}

func TestSettingsWriteTheKeysOfTheSpecificationInOrder(t *testing.T) {
	raw, err := json.Marshal(settings.DefaultSettings())
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	keys := []string{`"theme"`, `"useSystemColor"`, `"seedColor"`, `"language"`, `"extraEnvironment"`, `"railExpanded"`}
	previous := -1
	for _, key := range keys {
		index := strings.Index(text, key)
		if index < 0 || index < previous {
			t.Fatalf("keys out of order in %s", text)
		}
		previous = index
	}
	if strings.Contains(text, "vfoxExecutable") {
		t.Errorf("absent executable must be omitted: %s", text)
	}
}

func TestSettingsIgnoreMalformedValuesAndKeepUnknownKeys(t *testing.T) {
	parsed := settings.SettingsFromJSON(decode(t, `{"theme":"purple","useSystemColor":"yes",`+
		`"seedColor":"#fff","language":42,"vfoxExecutable":"",`+
		`"extraEnvironment":{"A":"1","B":2},"railExpanded":1,"future":true}`))
	if parsed.Theme != settings.ThemeSystem || !parsed.UseSystemColor ||
		parsed.SeedColor != settings.DefaultSeedColor || parsed.Language != settings.LanguageSystem ||
		parsed.VfoxExecutable != "" || !reflect.DeepEqual(parsed.ExtraEnvironment, map[string]string{"A": "1"}) ||
		parsed.RailExpanded {
		t.Errorf("got %#v", parsed)
	}
	raw, err := json.Marshal(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"future":true`) {
		t.Errorf("unknown keys must survive: %s", raw)
	}
	if got := settings.SettingsFromJSON(decode(t, `[1,2]`)); !got.Equal(settings.DefaultSettings()) {
		t.Errorf("got %#v", got)
	}
	if got := settings.SettingsFromJSON(decode(t, `{"extraEnvironment":["x"]}`)); len(got.ExtraEnvironment) != 0 {
		t.Errorf("got %#v", got)
	}
	if got := settings.SettingsFromJSON(decode(t, `{"seedColor":true}`)); got.SeedColor != settings.DefaultSeedColor {
		t.Errorf("a boolean is not a colour: %#v", got)
	}
}

func TestSettingsCompareByValue(t *testing.T) {
	base := settings.DefaultSettings()
	other := base
	other.ExtraEnvironment = map[string]string{}
	if !base.Equal(other) {
		t.Error("empty and missing environments are equal")
	}
	other.RailExpanded = true
	if base.Equal(other) {
		t.Error("expected a difference")
	}
}

func TestLanguageCodes(t *testing.T) {
	if settings.LanguageSystem.LanguageCode() != "" || settings.LanguageEnglish.LanguageCode() != "en" ||
		settings.LanguageSimplifiedChinese.LanguageCode() != "zh" {
		t.Error("wrong language codes")
	}
}

func TestEnvironmentLines(t *testing.T) {
	parsed := settings.ParseEnvironmentLines("# mirrors\nA=1\n\n B = x=y \nnot valid\n1X=2\n")
	if !reflect.DeepEqual(parsed.Variables, map[string]string{"A": "1", "B": "x=y"}) ||
		!reflect.DeepEqual(parsed.InvalidLines, []int{5, 6}) {
		t.Errorf("got %#v", parsed)
	}
	if got := settings.FormatEnvironmentLines(map[string]string{"B": "2", "A": "1"}); got != "A=1\nB=2" {
		t.Errorf("got %q", got)
	}
}

func TestSettingsPath(t *testing.T) {
	if got := settings.SettingsPath(map[string]string{"XDG_CONFIG_HOME": "/x", "HOME": "/h"}, "/fallback"); got != "/x/vfox-plus/settings.json" {
		t.Errorf("got %q", got)
	}
	if got := settings.SettingsPath(map[string]string{"XDG_CONFIG_HOME": "relative", "HOME": "/h"}, "/fallback"); got != "/h/.config/vfox-plus/settings.json" {
		t.Errorf("got %q", got)
	}
	if got := settings.SettingsPath(map[string]string{}, "/fallback"); got != "/fallback/.config/vfox-plus/settings.json" {
		t.Errorf("got %q", got)
	}
}

func TestJsonStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "settings.json")
	store := settings.JSONFileSettingsStore{Path: path}
	if got := store.Load(); !got.Equal(settings.DefaultSettings()) {
		t.Errorf("got %#v", got)
	}
	want := settings.DefaultSettings()
	want.Theme = settings.ThemeLight
	want.ExtraEnvironment = map[string]string{"K": "é"}
	if err := store.Save(want); err != nil {
		t.Fatal(err)
	}
	if got := store.Load(); !got.Equal(want) {
		t.Errorf("got %#v", got)
	}
	content, err := os.ReadFile(path) //nolint:gosec // The path is a file of the test.
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(content), "}\n") || !strings.Contains(string(content), `"K": "é"`) {
		t.Errorf("got %q", content)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil || len(entries) != 1 {
		t.Errorf("temporary files must not be left behind: %v %v", entries, err)
	}
}

func TestJsonStoreCorruptedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := (settings.JSONFileSettingsStore{Path: path}.Load()); !got.Equal(settings.DefaultSettings()) {
		t.Errorf("got %#v", got)
	}
}
