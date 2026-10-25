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

//go:build !windows

package settings

import (
	"path/filepath"
)

func settingsPath(environment map[string]string, fallbackHome string) string {
	if xdg, ok := environment["XDG_CONFIG_HOME"]; ok && len(xdg) > 0 && xdg[0] == '/' {
		return filepath.Join(xdg, "vfox-plus", "settings.json")
	}
	home, ok := environment["HOME"]
	if !ok || home == "" {
		home = fallbackHome
	}
	return filepath.Join(home, ".config", "vfox-plus", "settings.json")
}
