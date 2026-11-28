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

// Package profiletest gives tests the profiles of the repository, as the
// application ships them (see package builtin), without the assets that
// `vfxtool prepare` collects.
package profiletest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/core/profile"
)

// Documents reads the profiles of shared/profile/plugins, by plugin name. The
// directory is looked for from the directory of the package under test.
func Documents(t testing.TB) map[string]profile.Document {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "shared", "profile", "plugins")); err == nil {
			break
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("no shared/profile/plugins above the package")
		}
		directory = parent
	}
	files, err := filepath.Glob(filepath.Join(directory, "shared", "profile", "plugins", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no profiles: %v", err)
	}
	documents := make(map[string]profile.Document, len(files))
	for _, file := range files {
		content, err := os.ReadFile(file) //nolint:gosec // A profile of the repository.
		if err != nil {
			t.Fatal(err)
		}
		d, _, err := profile.ParseDocument(content)
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		documents[strings.TrimSuffix(filepath.Base(file), ".json")] = d
	}
	return documents
}

// Registry knows the profiles of the repository for the platform, as the
// application's own.
func Registry(t testing.TB, platform profile.Platform) *profile.Registry {
	t.Helper()
	return profile.NewRegistry(platform, Documents(t))
}
