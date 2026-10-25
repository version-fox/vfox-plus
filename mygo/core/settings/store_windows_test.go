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

package settings_test

import (
	"path/filepath"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
)

func TestSettingsPathWindows(t *testing.T) {
	got := settings.SettingsPath(map[string]string{"APPDATA": `C:\Users\me\AppData\Roaming`}, `C:\fallback`)
	want := filepath.Join(`C:\Users\me\AppData\Roaming`, "vfox-plus", "settings.json")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
