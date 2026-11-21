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

package screens

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
)

// uiLanguageInConfigDir points the configuration directory of a test at a directory
// of its own, and restores the language when the test ends.
func uiLanguageInConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	i18n.SetLanguage(i18n.English)
	t.Cleanup(func() {
		settingsSystemLanguages = nil
		i18n.SetLanguage(i18n.English)
	})
}

func TestTheInterfaceLanguageIsSavedInTheConfigurationDirectory(t *testing.T) {
	uiLanguageInConfigDir(t)
	if got := loadUILanguage(); got != i18n.PreferenceSystem {
		t.Fatalf("a phone with nothing saved reads %q, want the system's", got)
	}
	if err := saveUILanguage(i18n.PreferenceChinese); err != nil {
		t.Fatal(err)
	}
	path, err := uiLanguagePath()
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != uiLanguageFile || filepath.Base(filepath.Dir(path)) != configDirName {
		t.Errorf("the preference is saved at %s, want %s under %s", path, uiLanguageFile, configDirName)
	}
	// The path is the test's own configuration directory, from uiLanguagePath.
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `{"language":"zh"}` {
		t.Errorf("the file holds %s, want {\"language\":\"zh\"}", data)
	}
	if got := loadUILanguage(); got != i18n.PreferenceChinese {
		t.Errorf("the saved preference reads back as %q, want zh", got)
	}
	if err := saveUILanguage(i18n.PreferenceEnglish); err != nil {
		t.Fatal(err)
	}
	if got := loadUILanguage(); got != i18n.PreferenceEnglish {
		t.Errorf("a second save reads back as %q, want en", got)
	}
}

func TestADamagedOrUnknownLanguageFileMeansTheSystemsLanguage(t *testing.T) {
	uiLanguageInConfigDir(t)
	path, err := uiLanguagePath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, content := range []string{"not json", `{"language":"fr"}`, `{"language":""}`} {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if got := loadUILanguage(); got != i18n.PreferenceSystem {
			t.Errorf("the file %q reads as %q, want the system's", content, got)
		}
	}
}

func TestTheStartLanguageFollowsThePreferenceAndTheSystem(t *testing.T) {
	uiLanguageInConfigDir(t)
	ApplyUILanguage([]string{"zh-CN", "en-US"})
	if got := i18n.Current(); got != i18n.Chinese {
		t.Errorf("a system that reads zh-CN starts in %s, want zh", got)
	}
	if err := saveUILanguage(i18n.PreferenceEnglish); err != nil {
		t.Fatal(err)
	}
	ApplyUILanguage([]string{"zh-CN"})
	if got := i18n.Current(); got != i18n.English {
		t.Errorf("a saved English preference starts in %s, want en", got)
	}
	applyUILanguage(i18n.PreferenceSystem)
	if got := i18n.Current(); got != i18n.Chinese {
		t.Errorf("the system preference answers %s, want zh", got)
	}
}
