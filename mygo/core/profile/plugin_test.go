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
	"strings"
	"testing"
	"testing/fstest"
)

func files(contents map[string]string) fstest.MapFS {
	result := fstest.MapFS{}
	for name, content := range contents {
		result[name] = &fstest.MapFile{Data: []byte(content)}
	}
	return result
}

func TestReadPlugin(t *testing.T) {
	plugin := ReadPlugin(files(map[string]string{
		"metadata.lua":        "PLUGIN.legacyFilenames = { \".x-version\" }",
		"hooks/available.lua": `local m = os.getenv("VFOX_X_MIRROR")`,
		"hooks/detect.lua":    `function PLUGIN:Detect(ctx) end`,
		"lib/util.lua":        `os.getenv('VFOX_X_SWITCH')`,
		PluginProfile:         `{"plugin": "someone-else", "guide": "Own", "tips": [{"en": 3}]}`,
		"README.md":           "VFOX_X_NOT_LUA",
	}))
	if !reflect.DeepEqual(plugin.LegacyFiles, []string{".x-version"}) || !reflect.DeepEqual(plugin.EnvKeys, []string{"VFOX_X_MIRROR", "VFOX_X_SWITCH"}) {
		t.Errorf("legacy = %v, keys = %v", plugin.LegacyFiles, plugin.EnvKeys)
	}
	if plugin.Detect != CapabilityPresent || !plugin.Detect.Offered() {
		t.Errorf("detect = %v", plugin.Detect)
	}
	if plugin.Profile == nil || plugin.Profile.Plugin != "" || plugin.Profile.Resolve(linux()).Guide.EN != "Own" || len(plugin.Problems) == 0 {
		t.Errorf("profile = %+v, problems = %v", plugin.Profile, plugin.Problems)
	}
}

func TestWhatThePluginLacks(t *testing.T) {
	for name, c := range map[string]struct {
		files map[string]string
		want  Capability
	}{
		"hooks without detect":  {map[string]string{"hooks/available.lua": ""}, CapabilityAbsent},
		"one file with detect":  {map[string]string{"main.lua": "function PLUGIN:Detect(ctx) end"}, CapabilityPresent},
		"one file with its own": {map[string]string{"main.lua": "PLUGIN.Detect = function(ctx) end"}, CapabilityPresent},
		"one file without":      {map[string]string{"main.lua": "function PLUGIN:Available(ctx) end"}, CapabilityAbsent},
		"nothing":               {nil, CapabilityUnknown},
	} {
		plugin := ReadPlugin(files(c.files))
		if plugin.Detect != c.want || plugin.Profile != nil || plugin.Problems != nil {
			t.Errorf("%s: %+v", name, plugin)
		}
	}
	if ReadPlugin(os.DirFS(filepath.Join(t.TempDir(), "missing"))).Detect != CapabilityUnknown {
		t.Error("a directory that is missing tells nothing")
	}
}

func TestAProfileThatIsTooLargeIsLeftOut(t *testing.T) {
	plugin := ReadPlugin(files(map[string]string{PluginProfile: `{"guide": "` + strings.Repeat("x", maxFile) + `"}`}))
	if plugin.Profile != nil || len(plugin.Problems) != 1 {
		t.Errorf("plugin = %+v", plugin)
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
	// never escape the plugin directory: no separators, no ".." anywhere.
	for _, name := range []string{"", ".", "..", "a/b", `a\b`, "a/../b", "a..b"} {
		if got := PluginDir(environment, name); got != "" {
			t.Errorf("PluginDir(%q) = %q, want empty", name, got)
		}
	}
}
