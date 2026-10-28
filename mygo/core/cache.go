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

package core

import (
	"os"
	"path/filepath"
)

// UserCacheSubdir returns a directory under the operating system's user
// cache, creating it with mode 0700. Ephemeral files that must not roam with
// backups — spilled task logs, for example — live here:
// $XDG_CACHE_HOME on Linux, ~/Library/Caches on macOS, %LOCALAPPDATA% on
// Windows.
func UserCacheSubdir(elem ...string) (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(append([]string{base}, elem...)...)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	return dir, nil
}
