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

package vfox

import (
	"os"
	"path/filepath"
	"strings"
)

// IsRemotePluginSource reports whether source is a URL rather than a local
// path.
func IsRemotePluginSource(source string) bool {
	return strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://")
}

// VfoxMovesPluginSource reports whether vfox would move source instead of
// reading it.
//
// vfox (still in 1.0.13) renames a local `.lua` plugin into its temporary
// directory, which deletes the user's file and fails when the file is on
// another filesystem; local `.zip` archives and URLs are only read. See
// docs/en/dev/vfox-cli-contract.md.
func VfoxMovesPluginSource(source string) bool {
	return !IsRemotePluginSource(source) && strings.HasSuffix(strings.ToLower(source), ".lua")
}

// VfoxTemporaryDirectory is the temporary directory of vfox when it runs with
// environment: `~/.version-fox/tmp` if the legacy `~/.version-fox` exists,
// `~/.vfox/tmp` otherwise (on Windows under %USERPROFILE%, see
// userHomeDir). Without a home, where vfox does not run at all, the system
// temporary directory.
func VfoxTemporaryDirectory(environment map[string]string) string {
	if home := userHomeDir(environment); home != "" {
		legacy := filepath.Join(home, ".version-fox")
		if info, err := os.Stat(legacy); err == nil && info.IsDir() {
			return filepath.Join(legacy, "tmp")
		}
		return filepath.Join(home, ".vfox", "tmp")
	}
	return os.TempDir()
}

// PreparedPluginSource is a plugin source that is safe to hand to vfox.
type PreparedPluginSource struct {
	// Path is the path or URL to pass to `vfox add --source`.
	Path string
	// CopyDirectory is the directory of the private copy of the original
	// source, or empty when Path is the original source.
	CopyDirectory string
}

// IsCopy reports whether Path is a private copy of the original source.
func (p PreparedPluginSource) IsCopy() bool { return p.CopyDirectory != "" }

// Dispose deletes the private copy, if one was made.
func (p PreparedPluginSource) Dispose() {
	if p.CopyDirectory != "" {
		_ = os.RemoveAll(p.CopyDirectory)
	}
}

// PreparePluginSource prepares source for `vfox add --source`.
//
// A local `.lua` file is copied into a new directory under temporaryRoot, so
// that vfox moves the copy and the user's file stays where it is. Pass the
// VfoxTemporaryDirectory: vfox moves the file with a rename into a new
// directory there, which only works on the same filesystem. Other sources
// are returned as they are. If the copy cannot be made, the original path is
// returned and vfox reports the problem; it fails to create its own
// directory in temporaryRoot before moving anything.
func PreparePluginSource(source, temporaryRoot string) PreparedPluginSource {
	original := PreparedPluginSource{Path: source}
	if !VfoxMovesPluginSource(source) {
		return original
	}
	var directory string
	defer func() {
		if directory != "" {
			// Only reached when the copy below failed.
			_ = os.RemoveAll(directory)
		}
	}()
	if err := os.MkdirAll(temporaryRoot, 0o750); err != nil {
		return original
	}
	var err error
	directory, err = os.MkdirTemp(temporaryRoot, "vfox-plus-plugin-")
	if err != nil {
		directory = ""
		return original
	}
	name := filepath.Base(source)
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = "plugin.lua"
	}
	copy := filepath.Join(directory, name)
	content, err := os.ReadFile(source) //nolint:gosec // The user chose the plugin file.
	if err != nil {
		return original
	}
	err = os.WriteFile(copy, content, 0o600) //nolint:gosec // The copy is in a new directory of the temporary directory of vfox.
	if err != nil {
		return original
	}
	prepared := PreparedPluginSource{Path: copy, CopyDirectory: directory}
	directory = ""
	return prepared
}
