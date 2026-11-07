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
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
)

func TestInferFacetsFromNotesAndSuffixes(t *testing.T) {
	versions := []core.AvailableVersion{
		{Version: "24.22.0", Note: "LTS", HasNote: true},
		{Version: "26.12.0"},
		{Version: "3.47.7-x64", Note: "stable", HasNote: true},
		{Version: "3.47.7-arm64", Note: "stable", HasNote: true},
	}
	facets := InferFacets(versions)
	if len(facets) != 2 {
		t.Fatalf("facets = %+v", facets)
	}
	if facets[0].ID != "note" || facets[1].ID != "variant" {
		t.Fatalf("facets = %+v", facets)
	}
	if !reflect.DeepEqual(facets[0].Values, []string{"LTS", "stable"}) {
		t.Errorf("notes = %v", facets[0].Values)
	}
	if !reflect.DeepEqual(facets[1].Values, []string{"arm64", "x64"}) {
		t.Errorf("variants = %v", facets[1].Values)
	}
	if got := InferFacets(nil); got != nil {
		t.Errorf("no versions, no facets: %+v", got)
	}
	if got := InferFacets([]core.AvailableVersion{{Version: "8.7"}, {Version: "9.0.0"}}); got != nil {
		t.Errorf("gradle versions carry nothing to filter by: %+v", got)
	}
}

func TestParseLegacyFilenames(t *testing.T) {
	metadata := "PLUGIN.legacyFilenames = {\n    \".node-version\",\n    \".nvmrc\",\n}"
	if got := ParseLegacyFilenames(metadata); !reflect.DeepEqual(got, []string{".node-version", ".nvmrc"}) {
		t.Errorf("got %q", got)
	}
	if got := ParseLegacyFilenames("PLUGIN = {}"); got != nil {
		t.Errorf("got %q", got)
	}
}

func TestScanEnvKeysKeepsPluginSettings(t *testing.T) {
	sources := map[string]string{
		"a.lua": `local m = os.getenv("VFOX_GRADLE_MIRRORS") .. os.getenv("PATH")`,
		"b.lua": `os.getenv('FLUTTER_STORAGE_BASE_URL')`,
	}
	if got := ScanEnvKeys(sources); !reflect.DeepEqual(got, []string{"FLUTTER_STORAGE_BASE_URL", "VFOX_GRADLE_MIRRORS"}) {
		t.Errorf("got %q", got)
	}
}

func TestInferredVariablesAreEditable(t *testing.T) {
	variables := InferredVariables([]string{"VFOX_GRADLE_MIRRORS"})
	if len(variables) != 1 || variables[0].Kind != VariableURL {
		t.Fatalf("got %+v", variables)
	}
	if err := variables[0].Validate("https://repo.huaweicloud.com/gradle"); err != nil {
		t.Errorf("mirror rejected: %v", err)
	}
}

func TestPluginHomeRespectsVfoxHome(t *testing.T) {
	home := PluginHome(map[string]string{"VFOX_HOME": "/data/vfox"})
	if home != "/data/vfox" {
		t.Errorf("got %q", home)
	}
	directory := t.TempDir()
	legacy := filepath.Join(directory, ".version-fox")
	if err := os.MkdirAll(legacy, 0o750); err != nil {
		t.Fatal(err)
	}
	if got := PluginHome(map[string]string{"HOME": directory}); got != legacy {
		t.Errorf("legacy home = %q, want %q", got, legacy)
	}
}

func TestPluginDirKeepsTheNameInsideThePluginRoot(t *testing.T) {
	environment := map[string]string{"VFOX_HOME": "/data/vfox"}
	if got := PluginDir(environment, "java"); got != filepath.Join("/data/vfox", "plugin", "java") {
		t.Errorf("got %q", got)
	}
	// A name from the user (an alias they typed, a segment of a URL) must
	// never escape the plugin directory: no separators, no dot-only names.
	for _, name := range []string{"", ".", "..", "a/b", `a\b`, "a/../b"} {
		if got := PluginDir(environment, name); got != "" {
			t.Errorf("PluginDir(%q) = %q, want empty", name, got)
		}
	}
	// Dots inside a name are fine: they stay a single directory.
	if got := PluginDir(environment, "a..b"); got == "" {
		t.Error("a dotted name is a single directory")
	}
}

func TestInferredProfileHasGuideAndDocs(t *testing.T) {
	info := core.PluginInfo{Description: "Gradle runtime.", Homepage: "https://github.com/version-fox/vfox-gradle"}
	versions := []core.AvailableVersion{{Version: "8.7"}, {Version: "9.0.0"}}
	profile := InferredProfile("gradle-test", info, versions)
	if profile.Plugin != "gradle-test" || profile.Guide.EN == "" || len(profile.Docs) != 1 {
		t.Errorf("got %+v", profile)
	}
}
