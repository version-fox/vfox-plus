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

package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sourceDirectory is shared/profile of the repository.
var sourceDirectory = filepath.Join("..", "..", "..", "shared", "profile")

// sourceFiles are the profile files of the repository, by plugin name.
func sourceFiles(t testing.TB) map[string][]byte {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(sourceDirectory, "plugins", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no profiles in %s: %v", sourceDirectory, err)
	}
	contents := make(map[string][]byte, len(files))
	for _, file := range files {
		content, err := os.ReadFile(file) //nolint:gosec // A profile of the repository.
		if err != nil {
			t.Fatal(err)
		}
		contents[strings.TrimSuffix(filepath.Base(file), ".json")] = content
	}
	return contents
}

// sourceRegistry knows the profiles of the repository as the application's own.
func sourceRegistry(t testing.TB, platform Platform) *Registry {
	t.Helper()
	documents := make(map[string]Document)
	for name, content := range sourceFiles(t) {
		d, _, err := ParseDocument(content)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		documents[name] = d
	}
	return NewRegistry(platform, documents)
}

// Every profile of the repository reads without a problem, and not even a
// warning: both languages, one definition of each key.
func TestTheProfilesOfTheRepositoryAreClean(t *testing.T) {
	for name, content := range sourceFiles(t) {
		for _, problem := range Lint(content, LintOptions{Name: name}) {
			t.Errorf("%s: %s", name, problem)
		}
	}
}

// The index of the online directory lists every profile with its checksum; run
// `vfxtool profile index` after changing a profile.
func TestTheIndexListsTheProfilesOfTheRepository(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(sourceDirectory, "index.json")) //nolint:gosec // The index of the repository.
	if err != nil {
		t.Fatal(err)
	}
	index, err := ParseIndex(content)
	if err != nil {
		t.Fatal(err)
	}
	files := sourceFiles(t)
	for name, file := range files {
		if index.Plugins[name] != SHA256(file) {
			t.Errorf("index.json is stale for %s: run vfxtool profile index", name)
		}
	}
	for name := range index.Plugins {
		if _, ok := files[name]; !ok {
			t.Errorf("index.json lists %s, which has no file: run vfxtool profile index", name)
		}
	}
}
