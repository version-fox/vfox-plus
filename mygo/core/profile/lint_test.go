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
	"slices"
	"testing"
)

func lintStrings(content string, options LintOptions) []string {
	var result []string
	for _, problem := range Lint([]byte(content), options) {
		result = append(result, problem.String())
	}
	return result
}

func TestLintFindsWhatNoPlatformWouldUse(t *testing.T) {
	got := lintStrings(`{"plugin": "x",
		"search": [{"label": "Choose one", "default": "a", "choices": [{"value": "a"}]},
			{"when": {"os": "linux"}, "label": "L", "default": "b", "choices": [{"value": "b"}]}],
		"variables": [{"when": {"os": "linux"}, "key": "VFOX_X_A", "label": "A", "help": "How it works"}, {"key": "VFOX_X_A", "label": "A", "help": "How it works"},
			{"when": {"os": "windows"}, "key": "VFOX_X_B", "label": "B", "help": {"en": "A b", "zh": "乙"}},
			{"when": {"os": "linux"}, "key": "VFOX_X_B", "label": "B", "help": {"en": "A b", "zh": "乙"}}],
		"guide": "How names work"}`, LintOptions{Name: "y"})
	want := []string{
		`error: plugin: the file of y names "x"`,
		`error: variables: "VFOX_X_A" is listed twice for one platform`,
		`warning: search[0].label: has no Chinese text`,
		`warning: search[1]: no platform uses this variant: an earlier one is for the same platforms`,
		`warning: variables[0].help: has no Chinese text`,
		`warning: variables[1].help: has no Chinese text`,
		`warning: guide: has no Chinese text`,
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
}

func TestLintChecksTheProfileAgainstThePlugin(t *testing.T) {
	plugin := files(map[string]string{
		"metadata.lua":        `PLUGIN.legacyFilenames = { ".x-version", ".x-rc" }`,
		"hooks/available.lua": `local a = os.getenv("VFOX_X_MIRROR")`,
		"bin/install":         `if [ "${VFOX_X_BUILD:-no}" != no ]; then :; fi`,
		"README.md":           "VFOX_X_OTHER",
		"lib/vfox-plus.json":  `{"variables": [{"key": "VFOX_X_OTHER"}]}`,
	})
	got := lintStrings(`{"legacyFiles": [".x-version"], "variables": [{"key": "VFOX_X_MIRROR", "label": "M", "help": "h"}, {"key": "VFOX_X_OTHER", "label": "O", "help": "h"},
		{"key": "VFOX_X_BUILD", "label": "B", "help": "h"}]}`, LintOptions{Plugin: plugin})
	want := []string{
		`error: variables[1].key: no file of the plugin names "VFOX_X_OTHER"`,
		`warning: legacyFiles: metadata.lua names ".x-rc" too`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("problems = %q, want %q", got, want)
	}
	if got := lintStrings(`{}`, LintOptions{Plugin: files(nil)}); !slices.Equal(got, []string{"warning: the file carries no section", "error: the plugin has no Lua file"}) {
		t.Errorf("problems = %q", got)
	}
	if got := lintStrings(`[`, LintOptions{}); len(got) != 1 {
		t.Errorf("problems = %q", got)
	}
}
