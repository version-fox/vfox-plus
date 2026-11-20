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
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
)

// uiLanguageFile is the file of the interface language, which the phone keeps in
// its configuration directory, next to the saved computer. It is a preference of
// this phone only: the language of the computer is set on the computer.
const uiLanguageFile = "ui-language.json"

// settingsUILanguage is what the file holds: the preference, as the web names its
// page language: "system", "en" or "zh".
type settingsUILanguage struct {
	Language string `json:"language"`
}

// settingsSystemLanguages are the languages that the system reads, in its order. The
// preference "system" follows them; ApplyUILanguage sets them once, at the start.
var settingsSystemLanguages []string

// ApplyUILanguage sets the language of the interface at the start: the saved
// preference of the phone, and the languages that the system reads when the
// preference follows the system. The app calls it once, before the window is shown.
func ApplyUILanguage(systemLanguages []string) {
	settingsSystemLanguages = slices.Clone(systemLanguages)
	applyUILanguage(loadUILanguage())
}

// applyUILanguage makes the interface answer in the language of a preference.
func applyUILanguage(preference i18n.Preference) {
	i18n.SetLanguage(i18n.Resolve(preference, settingsSystemLanguages))
}

// uiLanguagePath is the file of the interface language, in the configuration
// directory of the app.
func uiLanguagePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, configDirName, uiLanguageFile), nil
}

// loadUILanguage returns the saved preference of the interface language. A missing,
// damaged or unknown file means the system's language, which is what a new phone
// has.
func loadUILanguage() i18n.Preference {
	path, err := uiLanguagePath()
	if err != nil {
		return i18n.PreferenceSystem
	}
	// The path comes from uiLanguagePath, which joins the configuration directory and the
	// file's name; no text that the user typed goes into it. Clean only normalises its spelling.
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return i18n.PreferenceSystem
	}
	var saved settingsUILanguage
	if json.Unmarshal(data, &saved) != nil {
		return i18n.PreferenceSystem
	}
	switch preference := i18n.Preference(saved.Language); preference {
	case i18n.PreferenceEnglish, i18n.PreferenceChinese:
		return preference
	}
	return i18n.PreferenceSystem
}

// saveUILanguage keeps the preference of the interface language. The file is
// written whole, through a file of its own that replaces it, so that a failure
// leaves the old one.
func saveUILanguage(preference i18n.Preference) error {
	path, err := uiLanguagePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(settingsUILanguage{Language: string(preference)})
	if err != nil {
		return err
	}
	temp := path + ".tmp"
	if err := os.WriteFile(temp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temp, path)
}
