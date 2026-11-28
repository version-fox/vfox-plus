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
	"reflect"
	"slices"
	"testing"
)

// document reads a profile that has no error.
func document(t *testing.T, content string) Document {
	t.Helper()
	d, problems, err := ParseDocument([]byte(content))
	if err != nil || slices.ContainsFunc(problems, func(p Problem) bool { return !p.Warning }) {
		t.Fatalf("%s: %v %v", content, err, problems)
	}
	return d
}

func TestEachSectionComesFromTheFirstSourceThatHasIt(t *testing.T) {
	builtin := map[string]Document{"x": document(t, `{"plugin": "x", "guide": "built in", "tips": ["built in"],
		"variables": [{"key": "VFOX_X_MIRROR", "help": "h", "presets": [{"value": "https://vetted.example.org"}]}]}`)}
	registry := NewRegistry(linux(), builtin)
	registry.SetCatalog(map[string]Document{"x": document(t, `{"plugin": "x", "guide": "catalog", "legacyFiles": [".catalog"]}`)})
	own := document(t, `{"guide": "plugin", "variables": [{"key": "VFOX_X_MIRROR", "help": "h",
		"presets": [{"value": "https://vetted.example.org"}, {"value": "https://new.example.org"}]},
		{"key": "VFOX_X_SWITCH", "help": "h", "kind": "toggle"}]}`)
	resolution := registry.Resolve("x", "", &own)
	profile := resolution.Profile
	if !resolution.Found || !resolution.Curated || profile.Plugin != "x" {
		t.Fatalf("resolution = %+v", resolution)
	}
	if profile.Guide.EN != "plugin" || profile.LegacyFiles[0] != ".catalog" || profile.Tips[0].EN != "built in" || len(profile.Variables) != 2 {
		t.Errorf("profile = %+v", profile)
	}
	// What the application's own profile offers is vetted; the rest is not.
	mirror, _ := profile.Variable("VFOX_X_MIRROR")
	if mirror.NeedsConfirmation("https://vetted.example.org") || !mirror.NeedsConfirmation(" https://new.example.org ") || mirror.NeedsConfirmation("https://typed.example.org") {
		t.Errorf("unvetted = %v", mirror.Unvetted)
	}
	toggle, _ := profile.Variable("VFOX_X_SWITCH")
	if !toggle.NeedsConfirmation("1") {
		t.Errorf("unvetted = %v", toggle.Unvetted)
	}
	registry.SetUser(map[string]Document{"x": document(t, `{"plugin": "x", "guide": "user", "tips": []}`)})
	profile = registry.Resolve("x", "", &own).Profile
	if profile.Guide.EN != "user" || profile.Tips != nil {
		t.Errorf("profile of the user = %+v", profile)
	}
	// The application's own, alone, needs no confirmation.
	registry = NewRegistry(linux(), builtin)
	profile, _ = registry.Lookup("x")
	if mirror, _ := profile.Variable("VFOX_X_MIRROR"); mirror.Unvetted != nil {
		t.Errorf("unvetted = %v", mirror.Unvetted)
	}
}

func TestAPluginIsFoundByItsNameOrTheNameItDeclares(t *testing.T) {
	registry := NewRegistry(linux(), map[string]Document{"nodejs": document(t, `{"plugin": "nodejs", "guide": "node"}`)})
	if got := registry.Resolve("node-lts", "nodejs", nil); !got.Found || !got.Curated || got.Profile.Guide.EN != "node" || got.Profile.Plugin != "nodejs" {
		t.Errorf("alias = %+v", got)
	}
	if got := registry.Resolve("node-lts", "", nil); got.Found {
		t.Errorf("an alias that has not said what it is: %+v", got)
	}
	own := document(t, `{"guide": "own"}`)
	if got := registry.Resolve("other", "other-plugin", &own); !got.Found || got.Curated || got.Profile.Plugin != "other-plugin" || got.Sections != SectionGuide {
		t.Errorf("a plugin that describes itself = %+v", got)
	}
	if got := (*Registry)(nil).Resolve("nodejs", "", nil); got.Found {
		t.Errorf("a nil registry knows nothing: %+v", got)
	}
}

func TestListenersHearOfChanges(t *testing.T) {
	registry := NewRegistry(linux(), nil)
	calls := 0
	stop := registry.OnChange(func() { calls++ })
	registry.SetCatalog(nil)
	registry.SetUser(nil)
	registry.Register(Profile{Plugin: "x"})
	stop()
	registry.SetCatalog(nil)
	if calls != 3 {
		t.Errorf("%d calls", calls)
	}
	if got := registry.Resolve("x", "", nil); !got.Curated || got.Sections&SectionSearch == 0 {
		t.Errorf("a registered profile counts as it is: %+v", got)
	}
}

func TestTheProfilesOfTheUser(t *testing.T) {
	directory := t.TempDir()
	for name, content := range map[string]string{
		"node.json":     `{"plugin": "nodejs", "guide": {"en": "Mine", "zh": "我的"}}`,
		"broken.json":   `{`,
		"nameless.json": `{"guide": "x"}`,
		"notes.txt":     `x`,
	} {
		if err := os.WriteFile(filepath.Join(directory, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	documents, problems := LoadUserProfiles(directory)
	if len(documents) != 1 || documents["nodejs"].Resolve(linux()).Guide.ZH != "我的" {
		t.Errorf("documents = %+v", documents)
	}
	var paths []string
	for _, problem := range problems {
		paths = append(paths, problem.Path)
	}
	if !reflect.DeepEqual(paths, []string{"broken.json", "nameless.json: plugin"}) {
		t.Errorf("problems = %v", problems)
	}
	if documents, _ := LoadUserProfiles(filepath.Join(directory, "missing")); documents != nil {
		t.Errorf("a missing directory = %+v", documents)
	}
	if got := ProfilesDir(filepath.Join(directory, "settings.json")); got != filepath.Join(directory, "profiles") {
		t.Errorf("ProfilesDir = %q", got)
	}
}
