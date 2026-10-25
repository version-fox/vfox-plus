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

//go:build windows

package settings

import (
	"os"
	"path/filepath"
)

func settingsPath(environment map[string]string, fallbackHome string) string {
	if appData, ok := environment["APPDATA"]; ok && appData != "" {
		return filepath.Join(appData, "vfox-plus", "settings.json")
	}
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "vfox-plus", "settings.json")
	}
	home, ok := environment["USERPROFILE"]
	if !ok || home == "" {
		home = environment["HOME"]
	}
	if home == "" {
		home = fallbackHome
	}
	return filepath.Join(home, "AppData", "Roaming", "vfox-plus", "settings.json")
}
