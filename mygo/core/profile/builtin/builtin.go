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

// Package builtin holds the profiles that the application ships: those of
// shared/profile/plugins, which `vfxtool prepare` collects into the assets.
package builtin

import (
	"io/fs"
	"path"
	"strings"
	"sync"

	"github.com/version-fox/vfox-plus/mygo/assets"
	"github.com/version-fox/vfox-plus/mygo/core/profile"
)

// Files are the profile files, by plugin name.
var Files = sync.OnceValue(func() map[string][]byte {
	files := make(map[string][]byte)
	// Reading embedded files does not fail: a name that the directory lists
	// is there.
	entries, _ := fs.ReadDir(assets.Files, "profiles")
	for _, entry := range entries {
		name, ok := strings.CutSuffix(entry.Name(), ".json")
		if !ok {
			continue
		}
		if content, err := fs.ReadFile(assets.Files, path.Join("profiles", entry.Name())); err == nil {
			files[name] = content
		}
	}
	return files
})

// Documents are the profiles, by plugin name. The tests of package profile
// check that every file reads without a problem.
var Documents = sync.OnceValue(func() map[string]profile.Document {
	documents := make(map[string]profile.Document)
	for name, content := range Files() {
		if d, _, err := profile.ParseDocument(content); err == nil {
			documents[name] = d
		}
	}
	return documents
})
