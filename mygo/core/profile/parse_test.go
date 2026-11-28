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
	"reflect"
	"slices"
	"strings"
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
)

// parse reads a profile and returns it for linux/amd64 with its problems.
func parse(t *testing.T, content string) (Profile, Document, []string) {
	t.Helper()
	d, problems, err := ParseDocument([]byte(content))
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	var reported []string
	for _, problem := range problems {
		reported = append(reported, problem.String())
	}
	return d.Resolve(linux()), d, reported
}

// reports tells whether a problem starts with the prefix.
func reports(problems []string, prefix string) bool {
	return slices.ContainsFunc(problems, func(p string) bool { return strings.HasPrefix(p, prefix) })
}

func TestTextsAreStringsOrOneByLanguage(t *testing.T) {
	profile, _, problems := parse(t, `{"guide": "Mirror", "tips": [
		{"en": "A", "zh": "甲"}, {"en": "B", "zh-CN": "乙", "ja": "び"}, {"zh": "丙"}, {}, "", 4, {"en": "C", "colour": "red"}]}`)
	if profile.Guide != (Text{EN: "Mirror"}) {
		t.Errorf("guide = %+v", profile.Guide)
	}
	want := []Text{{EN: "A", ZH: "甲"}, {EN: "B", ZH: "乙"}, {EN: "丙", ZH: "丙"}, {EN: "C"}}
	if !reflect.DeepEqual(profile.Tips, want) {
		t.Errorf("tips = %+v, want %+v", profile.Tips, want)
	}
	for _, prefix := range []string{"warning: tips[2]", "error: tips[3]", "error: tips[4]", "error: tips[5]", "warning: tips[6].colour"} {
		if !reports(problems, prefix) {
			t.Errorf("no problem %q in %v", prefix, problems)
		}
	}
}

func TestABadPartCostsOnlyThatPart(t *testing.T) {
	profile, d, problems := parse(t, `{"plugin": "x", "colour": "blue",
		"shorthands": [{"name": "latest", "meaning": "The newest"}, {"name": "-x", "meaning": "no"}, {"name": "a b", "meaning": "no"},
			{"name": "x@1", "meaning": "no"}, {"name": "1"}],
		"variables": "none",
		"legacyFiles": [".x-version"],
		"legacyStrategy": "yes",
		"facets": [{"id": "a", "values": []}],
		"guide": 3,
		"docs": [{"title": "Docs", "url": "https://example.org/docs"}, {"title": "Plain", "url": "http://example.org"}, {"title": "File", "url": "file:///etc/passwd"}]}`)
	if len(profile.Shorthands) != 1 || profile.Shorthands[0].Name != "latest" {
		t.Errorf("shorthands = %+v", profile.Shorthands)
	}
	if len(profile.Docs) != 1 {
		t.Errorf("docs = %+v", profile.Docs)
	}
	want := SectionShorthands | SectionLegacyFiles | SectionDocs
	if d.Sections() != want {
		t.Errorf("sections = %b, want %b: a bad section is not carried", d.Sections(), want)
	}
	for _, prefix := range []string{"warning: colour", "error: shorthands[1].name", "error: shorthands[4].meaning", "error: variables:",
		"error: legacyStrategy", "error: facets[0].values", "error: guide", "error: docs[1].url", "error: docs[2].url"} {
		if !reports(problems, prefix) {
			t.Errorf("no problem %q in %v", prefix, problems)
		}
	}
}

func TestAnEmptyListIsCarried(t *testing.T) {
	// An empty list hides the section of the files after it; a list of which
	// nothing can be read does not.
	_, d, _ := parse(t, `{"tips": [], "variables": [{"key": "PATH"}]}`)
	if d.Sections() != SectionTips {
		t.Errorf("sections = %b", d.Sections())
	}
}

func TestPartsForSomePlatforms(t *testing.T) {
	_, d, problems := parse(t, `{
		"search": [{"when": {"os": "linux", "arch": ["loong64"]}, "label": "L", "default": "loong", "choices": [{"value": "loong"}]},
			{"label": "L", "default": "open", "choices": [{"value": "open"}, {"value": "tem"}]}],
		"variables": [{"when": {"os": ["windows"]}, "key": "VFOX_X_WINDOWS", "help": "h"}, {"key": "VFOX_X_ALL", "help": "h"}],
		"tips": [{"when": {"os": ["plan9"]}, "en": "A"}, {"when": "linux", "en": "B"}]}`)
	on := func(os, arch string) Profile { return d.Resolve(Platform{OS: os, Arch: arch}) }
	if got := on("linux", "loong64").Search.Default; got != "loong" {
		t.Errorf("loong64 searches %q", got)
	}
	if got := on("linux", "amd64").Search.Default; got != "open" {
		t.Errorf("amd64 searches %q", got)
	}
	if got := on("linux", "amd64").Variables; len(got) != 1 || got[0].Key != "VFOX_X_ALL" {
		t.Errorf("linux variables = %+v", got)
	}
	if got := on("windows", "amd64").Variables; len(got) != 2 {
		t.Errorf("windows variables = %+v", got)
	}
	if got := on("plan9", "amd64").Tips; len(got) != 1 {
		t.Errorf("plan9 tips = %+v", got)
	}
	if !reports(problems, `warning: tips[0].when.os`) || !reports(problems, "error: tips[1].when") {
		t.Errorf("problems = %v", problems)
	}
}

func TestSearchesNeedTheirDefault(t *testing.T) {
	_, d, problems := parse(t, `{"search": {"label": "D", "default": "x", "choices": [{"value": "a"}, {"value": "a"}, {"value": "--all"}, {"value": "b c"}]}}`)
	if d.Sections() != 0 {
		t.Errorf("a search without its default is not read: %b", d.Sections())
	}
	for _, prefix := range []string{"error: search.choices[1].value", "error: search.choices[2].value", "error: search.choices[3].value", "error: search.default"} {
		if !reports(problems, prefix) {
			t.Errorf("no problem %q in %v", prefix, problems)
		}
	}
}

func TestVariablesOfEveryKind(t *testing.T) {
	profile, _, problems := parse(t, `{"variables": [
		{"key": "VFOX_X_MIRROR", "help": "h", "presets": [{"value": "https://mirror.example.org"}, {"value": "http://mirror.example.org"}]},
		{"key": "VFOX_X_CHOICE", "help": "h", "kind": "choice", "presets": [{"value": "a", "name": "A"}]},
		{"key": "VFOX_X_NOTHING", "help": "h", "kind": "choice"},
		{"key": "VFOX_X_SWITCH", "help": "h", "kind": "toggle", "presets": [{"value": "ignored"}]},
		{"key": "VFOX_X_YESNO", "help": "h", "kind": "toggle", "on": "yes", "off": "no"},
		{"key": "VFOX_X_SAME", "help": "h", "kind": "toggle", "on": "1", "off": "1"},
		{"key": "VFOX_X_OPTIONS", "help": "h", "kind": "text", "presets": [{"value": "--enable-x --with-y"}]},
		{"key": "VFOX_X_NEW", "help": "h", "kind": "slider"},
		{"key": "not a name", "help": "h"}]}`)
	keys := map[string]Variable{}
	for _, v := range profile.Variables {
		keys[v.Key] = v
	}
	if len(profile.Variables) != 5 {
		t.Fatalf("variables = %+v", profile.Variables)
	}
	mirror := keys["VFOX_X_MIRROR"]
	if mirror.Kind != VariableURL || len(mirror.Presets) != 1 || mirror.Presets[0].Name.EN != "https://mirror.example.org" || mirror.Label.EN != "X Mirror" {
		t.Errorf("mirror = %+v", mirror)
	}
	if v := keys["VFOX_X_SWITCH"]; v.Kind != VariableToggle || v.On != "1" || v.Off != "" || v.Presets != nil {
		t.Errorf("switch = %+v", v)
	}
	for value, ok := range map[string]bool{"": true, "yes": true, "no": true, "1": false} {
		if (keys["VFOX_X_YESNO"].Validate(value) == nil) != ok {
			t.Errorf("toggle takes %q: %v", value, !ok)
		}
	}
	options := keys["VFOX_X_OPTIONS"]
	for value, ok := range map[string]bool{"--prefix=/opt a": true, "a\nb": false, strings.Repeat("x", 3000): false} {
		if (options.Validate(value) == nil) != ok {
			t.Errorf("text takes %q: %v", value, !ok)
		}
	}
	for _, prefix := range []string{"error: variables[0].presets[1].value", "error: variables[2].presets", "error: variables[5].on",
		"error: variables[7].kind", "error: variables[8].key"} {
		if !reports(problems, prefix) {
			t.Errorf("no problem %q in %v", prefix, problems)
		}
	}
}

func TestProfilesMayNotSetWhatRunsPrograms(t *testing.T) {
	for _, key := range []string{"PATH", "path", "Home", "VFOX_HOME", "__VFOX_SHELL", "LD_PRELOAD", "ld_library_path", "DYLD_INSERT_LIBRARIES",
		"NODE_OPTIONS", "JAVA_TOOL_OPTIONS", "_JAVA_OPTIONS", "BASH_ENV", "HTTPS_PROXY", "SSL_CERT_FILE", "GITHUB_TOKEN", "MAKEFLAGS", "CC",
		"GIT_SSH_COMMAND", "npm_config_registry", "BASH_FUNC_x"} {
		_, d, problems := parse(t, `{"variables": [{"key": "`+key+`", "help": "h", "kind": "text"}]}`)
		if d.Sections() != 0 || !reports(problems, "error: variables[0].key") {
			t.Errorf("a profile may not set %s: %v", key, problems)
		}
		if _, ok := GenericVariable(key); ok {
			t.Errorf("%s is no generic setting", key)
		}
	}
	for _, key := range []string{"VFOX_NODEJS_MIRROR", "FLUTTER_STORAGE_BASE_URL", "USE_PREBUILT_OTP", "Conda_Forge", "PHP_WITH_PEAR"} {
		if IsReservedVariable(key) {
			t.Errorf("%s is a setting of a plugin", key)
		}
	}
}

func TestNamingRules(t *testing.T) {
	profile, _, problems := parse(t, `{"naming": {"sources": ["tem", {"value": "temfx", "name": "Temurin FX"}], "defaultSource": "tem",
		"rules": [{"pattern": "^nightly$", "flavors": ["Nightly"]},
			{"pattern": "^(?<version>[0-9.]+)(?:-(?<source>{sources}))?(?:-(?<arch>x64|X64))?$", "labels": {"x64": "Intel"}}]}}`)
	if len(problems) != 0 {
		t.Fatalf("problems = %v", problems)
	}
	cases := map[string]Parts{
		"nightly":        {Version: "nightly", SourceValue: "tem", Source: Text{EN: "tem"}, Flavors: []string{"Nightly"}},
		"21.0-temfx":     {Version: "21.0", SourceValue: "temfx", Source: Text{EN: "Temurin FX"}},
		"21-X64":         {Version: "21", SourceValue: "tem", Source: Text{EN: "tem"}, Flavors: []string{"Intel"}},
		"21-zulu":        {Version: "21-zulu"},
		"21.0.8-tem-x64": {Version: "21.0.8", SourceValue: "tem", Source: Text{EN: "tem"}, Flavors: []string{"Intel"}},
	}
	for name, want := range cases {
		if got, ok := profile.Describe(name); !ok || !reflect.DeepEqual(got, want) {
			t.Errorf("Describe(%q) = %+v, %v; want %+v", name, got, ok, want)
		}
	}
	if profile, _, _ := parse(t, `{"naming": "plain"}`); profile.Naming != nil {
		t.Error("plain names have no structure")
	}
	for _, bad := range []string{`"java"`, `{"rules": []}`, `{"rules": [{"pattern": "("}]}`, `{"rules": [{"pattern": "{sources}"}]}`,
		`{"rules": [{"pattern": "a"}, {"pattern": "[}"}]}`} {
		_, d, problems := parse(t, `{"naming": `+bad+`}`)
		if d.Sections() != 0 || len(problems) == 0 {
			t.Errorf("naming %s is read: %v", bad, problems)
		}
	}
}

func TestPatternFacets(t *testing.T) {
	profile, _, problems := parse(t, `{"facets": [{"id": "pre", "label": "Preview", "kind": "pattern", "pattern": "-(rc|preview)", "values": ["rc", "preview"]},
		{"id": "line", "kind": "pattern", "pattern": "^(?<value>[0-9]+)\\.", "values": ["8", "9"]},
		{"id": "bad", "kind": "pattern", "pattern": "(", "values": ["x"]}]}`)
	if len(profile.Facets) != 2 || !reports(problems, "error: facets[2].pattern") {
		t.Fatalf("facets = %+v, problems = %v", profile.Facets, problems)
	}
	versions := []core.AvailableVersion{{Version: "9.0.100-preview.1"}, {Version: "8.0.100-rc.1"}, {Version: "9.0.100"}}
	groups := profile.Groups(versions)
	want := []Group{
		{ID: "pre", Label: Text{EN: "Preview"}, Chips: []Chip{{"rc", 1}, {"preview", 1}}},
		{ID: "line", Label: Text{EN: "line"}, Chips: []Chip{{"8", 1}, {"9", 2}}},
	}
	if !reflect.DeepEqual(groups, want) {
		t.Errorf("groups = %+v", groups)
	}
}

func TestWhatIsNoProfileAtAll(t *testing.T) {
	for _, content := range []string{``, `[]`, `null`, `"x"`, `{`, `{"tips": ["` + strings.Repeat("x", maxFile) + `"]}`} {
		if _, _, err := ParseDocument([]byte(content)); err == nil {
			t.Errorf("%.40q is no profile", content)
		}
	}
	d, problems, err := ParseDocument([]byte("\xef\xbb\xbf" + `{"schema": 3, "plugin": "../x", "unknown": 1}`))
	if err != nil || d.Plugin != "" || len(problems) != 3 {
		t.Errorf("ParseDocument = %+v, %v, %v", d, problems, err)
	}
}
