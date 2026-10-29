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
	"errors"
	"reflect"
	"strings"
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
)

func linux() Platform { return Platform{OS: "linux", Arch: "amd64"} }

func lookup(t *testing.T, plugin string) Profile {
	t.Helper()
	profile, ok := NewRegistry(linux()).Lookup(plugin)
	if !ok {
		t.Fatalf("no profile for %s", plugin)
	}
	return profile
}

func TestTextFollowsTheLanguage(t *testing.T) {
	text := Text{EN: "Mirror", ZH: "镜像"}
	for tag, want := range map[string]string{"en": "Mirror", "": "Mirror", "zh": "镜像", "zh-Hans": "镜像", "ZH-cn": "镜像", "de": "Mirror"} {
		if got := text.For(tag); got != want {
			t.Errorf("For(%q) = %q, want %q", tag, got, want)
		}
	}
	if got := (Text{EN: "gh-proxy"}).For("zh"); got != "gh-proxy" {
		t.Errorf("a text without a translation falls back to English, got %q", got)
	}
}

func TestRegistryKnowsThePluginsOfTheOrganisation(t *testing.T) {
	registry := NewRegistry(linux())
	for _, plugin := range []string{"java", "nodejs", "flutter"} {
		if profile, ok := registry.Lookup(plugin); !ok || profile.Plugin != plugin {
			t.Errorf("Lookup(%q) = %v, %v", plugin, profile.Plugin, ok)
		}
	}
	if _, ok := registry.Lookup("maven"); ok {
		t.Error("maven has no profile")
	}
	var missing *Registry
	if _, ok := missing.Lookup("java"); ok {
		t.Error("a nil registry knows nothing")
	}
	registry.Register(Profile{Plugin: "maven", Guide: Text{EN: "x"}})
	if profile, ok := registry.Lookup("maven"); !ok || profile.Guide.EN != "x" {
		t.Error("a registered profile is found")
	}
}

func TestJavaOffersEveryDistributionOfFoojay(t *testing.T) {
	java := lookup(t, "java")
	if java.Search == nil || java.Search.Default != "open" {
		t.Fatalf("search = %+v", java.Search)
	}
	if got := len(java.Search.Choices); got != 32 {
		t.Fatalf("%d choices, want the 31 distributions and all", got)
	}
	seen := map[string]bool{}
	for _, choice := range java.Search.Choices {
		if seen[choice.Value] {
			t.Errorf("%s is listed twice", choice.Value)
		}
		seen[choice.Value] = true
		if choice.Name.EN == "" {
			t.Errorf("%s has no name", choice.Value)
		}
		if choice.Value != "all" && len(choice.Aliases) != 1 {
			t.Errorf("%s should name its long name, has %v", choice.Value, choice.Aliases)
		}
	}
	for _, short := range []string{"open", "tem", "zulu", "librca", "amzn", "ms", "oracle", "sem", "sapmchn", "kona", "albba", "bsg", "graal", "graalce", "nik", "mandrel", "jb", "trava", "adpt", "adptj9", "eliya", "ojdkbld", "opnlgc", "rdht", "semcrt", "gln", "grl8", "grl11", "grl16", "grl17", "grl19", "all"} {
		if !seen[short] {
			t.Errorf("the README of vfox-java lists %s", short)
		}
	}
	if seen["lnxabi1"] || seen["lnxabi2"] || seen["loongnix"] {
		t.Error("the Loongnix JDKs are for loong64 only")
	}
	// The popular distributions come first.
	popular := true
	for _, choice := range java.Search.Choices {
		if choice.Popular && !popular {
			t.Errorf("%s is popular and comes after one that is not", choice.Value)
		}
		popular = popular && choice.Popular
	}
	if !java.Search.Choices[0].Popular || java.Search.Choices[0].Value != "open" {
		t.Errorf("the default distribution comes first: %+v", java.Search.Choices[0])
	}
}

func TestJavaOnLoong64OffersTheLoongnixJdks(t *testing.T) {
	registry := NewRegistry(Platform{OS: "linux", Arch: "loong64"})
	java, _ := registry.Lookup("java")
	var values []string
	for _, choice := range java.Search.Choices {
		values = append(values, choice.Value)
	}
	if want := []string{"loongnix", "lnxabi1", "lnxabi2"}; !reflect.DeepEqual(values, want) || java.Search.Default != "loongnix" {
		t.Errorf("choices = %v, default %q", values, java.Search.Default)
	}
}

func TestSearchArgumentsAndChoices(t *testing.T) {
	java := lookup(t, "java")
	for value, want := range map[string][]string{"": nil, "open": nil, "tem": {"tem"}, "all": {"all"}} {
		if got := java.Search.Arguments(value); !reflect.DeepEqual(got, want) {
			t.Errorf("Arguments(%q) = %v, want %v", value, got, want)
		}
	}
	if choice, ok := java.Search.Choice(""); !ok || choice.Value != "open" {
		t.Errorf("the empty value is the default, got %+v", choice)
	}
	if _, ok := java.Search.Choice("nope"); ok {
		t.Error("an unknown value is no choice")
	}
	var none *Search
	if none.Arguments("tem") != nil {
		t.Error("a nil search passes nothing")
	}
	if _, ok := none.Choice("tem"); ok {
		t.Error("a nil search has no choices")
	}
	if lookup(t, "nodejs").Search != nil || lookup(t, "flutter").Search != nil {
		t.Error("nodejs and flutter ignore search arguments")
	}
}

func TestChoiceMatchesItsNames(t *testing.T) {
	java := lookup(t, "java")
	temurin, _ := java.Search.Choice("tem")
	for query, want := range map[string]bool{"": true, "  ": true, "tem": true, "TEMURIN": true, "adoptium": true, "zulu": false} {
		if got := temurin.Matches(query); got != want {
			t.Errorf("Matches(%q) = %v, want %v", query, got, want)
		}
	}
	bisheng, _ := java.Search.Choice("bsg")
	if !bisheng.Matches("毕昇") || !bisheng.Matches("华为") {
		t.Error("the Chinese names are matched as well")
	}
}

func TestVariablesValidateTheirValues(t *testing.T) {
	nodejs := lookup(t, "nodejs")
	mirror, _ := nodejs.Variable("VFOX_NODEJS_MIRROR")
	for value, ok := range map[string]bool{
		"": true, "  ": true, "https://npmmirror.com/mirrors/node": true, " http://127.0.0.1:8080/dist ": true,
		"npmmirror.com/mirrors/node": false, "ftp://example.org/node": false, "https://": false, "https://a b": false, "/usr/local/node": false,
	} {
		err := mirror.Validate(value)
		if (err == nil) != ok {
			t.Errorf("Validate(%q) = %v, want ok %v", value, err, ok)
		}
		var invalid *ValueError
		if err != nil && (!errors.As(err, &invalid) || invalid.Reason != ReasonAddress || !strings.Contains(err.Error(), "VFOX_NODEJS_MIRROR")) {
			t.Errorf("Validate(%q) = %v", value, err)
		}
	}
	libc, _ := nodejs.Variable("VFOX_NODEJS_LIBC")
	if libc.Validate("musl") != nil || libc.Validate("") != nil {
		t.Error("musl and the empty value are fine")
	}
	var invalid *ValueError
	if err := libc.Validate("gnu"); !errors.As(err, &invalid) || invalid.Reason != ReasonChoice {
		t.Errorf("Validate(gnu) = %v", err)
	}
	if _, ok := nodejs.Variable("HOME"); ok {
		t.Error("HOME is no variable of the plugin")
	}
	if preset, ok := mirror.PresetFor(" https://npmmirror.com/mirrors/node "); !ok || preset.Name.EN != "npmmirror" {
		t.Errorf("PresetFor = %+v, %v", preset, ok)
	}
	if _, ok := mirror.PresetFor("https://example.org"); ok {
		t.Error("an address of the user is no preset")
	}
}

func TestEveryPluginSaysWhatItDoes(t *testing.T) {
	want := map[string]struct {
		variables []string
		legacy    []string
	}{
		"java":    {[]string{"VFOX_JAVA_GITHUB_MIRROR"}, []string{".sdkmanrc"}},
		"nodejs":  {[]string{"VFOX_NODEJS_MIRROR", "VFOX_NODEJS_LIBC"}, []string{".node-version", ".nvmrc"}},
		"flutter": {[]string{"FLUTTER_STORAGE_BASE_URL", "VFOX_FLUTTER_GITHUB_MIRROR"}, nil},
	}
	for plugin, expected := range want {
		profile := lookup(t, plugin)
		var keys []string
		for _, variable := range profile.Variables {
			keys = append(keys, variable.Key)
			if variable.Label.EN == "" || variable.Label.ZH == "" || variable.Help.EN == "" || variable.Help.ZH == "" {
				t.Errorf("%s of %s lacks a label or a help in one language", variable.Key, plugin)
			}
			if variable.Kind == VariableURL {
				if variable.Example == "" || len(variable.Presets) == 0 {
					t.Errorf("%s of %s needs an example and presets", variable.Key, plugin)
				}
				for _, preset := range variable.Presets {
					if err := variable.Validate(preset.Value); err != nil {
						t.Errorf("the preset %q of %s: %v", preset.Value, variable.Key, err)
					}
				}
			}
		}
		if !reflect.DeepEqual(keys, expected.variables) {
			t.Errorf("%s variables = %v, want %v", plugin, keys, expected.variables)
		}
		if !reflect.DeepEqual(profile.LegacyFiles, expected.legacy) {
			t.Errorf("%s legacy files = %v, want %v", plugin, profile.LegacyFiles, expected.legacy)
		}
		if profile.Guide.EN == "" || profile.Guide.ZH == "" || len(profile.Tips) == 0 || len(profile.Shorthands) == 0 || len(profile.Docs) == 0 {
			t.Errorf("%s needs a guide, tips, shorthands and documents", plugin)
		}
		for _, tip := range profile.Tips {
			if tip.EN == "" || tip.ZH == "" {
				t.Errorf("%s has a tip without both languages: %+v", plugin, tip)
			}
		}
		for _, shorthand := range profile.Shorthands {
			if shorthand.Name == "" || strings.ContainsAny(shorthand.Name, " @") || shorthand.Meaning.EN == "" || shorthand.Meaning.ZH == "" {
				t.Errorf("%s has a bad shorthand %+v", plugin, shorthand)
			}
		}
		for _, link := range profile.Docs {
			if !strings.HasPrefix(link.URL.EN, "https://github.com/version-fox/vfox-"+plugin+"/blob/main/docs/") ||
				!strings.Contains(link.URL.ZH, "/docs/zh-CN/") || link.Title.EN == "" {
				t.Errorf("%s has a bad link %+v", plugin, link)
			}
		}
	}
	if lookup(t, "nodejs").LegacyStrategy != true || lookup(t, "java").LegacyStrategy {
		t.Error("only vfox-nodejs reads the strategy of the legacy version files")
	}
}

func TestEveryDistributionIsDescribedInBothLanguages(t *testing.T) {
	for _, choice := range javaDistributions {
		if choice.Name.EN == "" || choice.Maker.EN == "" {
			t.Errorf("%s lacks a name or a maker", choice.Value)
		}
	}
	// Sentences are translated; names of products are not.
	for _, choice := range append([]Choice{javaEverything}, javaLoongnix...) {
		if choice.Maker.ZH == "" {
			t.Errorf("%s needs a translation of what it is", choice.Value)
		}
	}
	if javaEverything.Name.ZH == "" || javaLoongnix[0].Name.ZH == "" {
		t.Error("the choices that stand for several distributions need translated names")
	}
}

func TestNoteTokensSplitOutsideParentheses(t *testing.T) {
	for note, want := range map[string][]string{
		"":            nil,
		"LTS":         {"LTS"},
		"LTS, JavaFX": {"LTS", "JavaFX"},
		"JDK 26+35; ABI2 (Linux 5.10 UAPI, glibc 2.34); JavaFX": {"JDK 26+35", "ABI2 (Linux 5.10 UAPI, glibc 2.34)", "JavaFX"},
		" stable ,": {"stable"},
		"a)b, c":    {"a)b", "c"},
	} {
		if got := NoteTokens(note); !reflect.DeepEqual(got, want) {
			t.Errorf("NoteTokens(%q) = %q, want %q", note, got, want)
		}
	}
}

func sampleFlutter() []core.AvailableVersion {
	return []core.AvailableVersion{
		{Version: "3.47.7-x64", Note: "stable", HasNote: true},
		{Version: "3.47.7-arm64", Note: "stable", HasNote: true},
		{Version: "3.50.0-0.1.pre-x64", Note: "beta", HasNote: true},
		{Version: "2.13.0-0.1.pre", Note: "dev", HasNote: true},
		{Version: "1.17.0", Note: "stable", HasNote: true},
		{Version: "3.41.10-ohos-1.0.0", Note: "OpenHarmony", HasNote: true},
	}
}

func TestFlutterGroupsAreBuiltFromTheVersionsThatAreListed(t *testing.T) {
	flutter := lookup(t, "flutter")
	groups := flutter.Groups(sampleFlutter())
	if len(groups) != 2 || groups[0].ID != "channel" || groups[1].ID != "architecture" {
		t.Fatalf("groups = %+v", groups)
	}
	if got, want := groups[0].Chips, []Chip{{"stable", 3}, {"beta", 1}, {"dev", 1}, {"OpenHarmony", 1}}; !reflect.DeepEqual(got, want) {
		t.Errorf("channels = %+v, want %+v", got, want)
	}
	if got, want := groups[1].Chips, []Chip{{"x64", 2}, {"arm64", 1}}; !reflect.DeepEqual(got, want) {
		t.Errorf("architectures = %+v, want %+v", got, want)
	}
	// A value no version has is not offered.
	only := flutter.Groups([]core.AvailableVersion{{Version: "3.47.7-x64", Note: "stable", HasNote: true}})
	if len(only) != 2 || len(only[0].Chips) != 1 || len(only[1].Chips) != 1 {
		t.Errorf("groups of one version = %+v", only)
	}
	if got := flutter.Groups(nil); got != nil {
		t.Errorf("no versions, no groups: %+v", got)
	}
}

func TestSelectionsFilterTheVersions(t *testing.T) {
	flutter := lookup(t, "flutter")
	versions := sampleFlutter()
	pass := func(selection Selection) []string {
		var names []string
		for _, version := range versions {
			if flutter.Matches(selection, version) {
				names = append(names, version.Version)
			}
		}
		return names
	}
	if got := pass(nil); len(got) != 6 {
		t.Errorf("no selection passes everything, got %v", got)
	}
	selection := Selection{}.Toggled("channel", "stable")
	if got := pass(selection); !reflect.DeepEqual(got, []string{"3.47.7-x64", "3.47.7-arm64", "1.17.0"}) {
		t.Errorf("stable = %v", got)
	}
	// Values of one group are alternatives; groups narrow each other.
	selection = selection.Toggled("channel", "beta")
	if got := pass(selection); len(got) != 4 {
		t.Errorf("stable or beta = %v", got)
	}
	selection = selection.Toggled("architecture", "x64")
	if got := pass(selection); !reflect.DeepEqual(got, []string{"3.47.7-x64", "3.50.0-0.1.pre-x64"}) {
		t.Errorf("stable or beta, x64 = %v", got)
	}
	if selection.Empty() || !selection.Has("architecture", "x64") || selection.Has("architecture", "arm64") {
		t.Errorf("selection = %v", selection)
	}
}

func TestToggledLeavesTheReceiverAlone(t *testing.T) {
	base := Selection{}.Toggled("channel", "stable")
	next := base.Toggled("channel", "beta")
	if base.Has("channel", "beta") || !next.Has("channel", "beta") || !next.Has("channel", "stable") {
		t.Errorf("base = %v, next = %v", base, next)
	}
	cleared := base.Toggled("channel", "stable")
	if !cleared.Empty() || len(cleared) != 0 {
		t.Errorf("toggling the only value off empties the selection: %v", cleared)
	}
	if !(Selection{"channel": {}}).Empty() || !Selection(nil).Empty() {
		t.Error("a selection without values is empty")
	}
}

func TestJavaAndNodeFiltersReadTheNotes(t *testing.T) {
	versions := []core.AvailableVersion{
		{Version: "25.0.4+7-zulu-fx", Note: "LTS, JavaFX", HasNote: true},
		{Version: "25.0.4+7-zulu", Note: "LTS", HasNote: true},
		{Version: "26+35-zulu-fx", Note: "JavaFX", HasNote: true},
		{Version: "26+35-zulu"},
	}
	java := lookup(t, "java")
	groups := java.Groups(versions)
	if len(groups) != 2 || groups[0].Chips[0] != (Chip{"LTS", 2}) || groups[1].Chips[0] != (Chip{"JavaFX", 2}) {
		t.Fatalf("groups = %+v", groups)
	}
	selection := Selection{}.Toggled("support", "LTS").Toggled("flavor", "JavaFX")
	var kept []string
	for _, version := range versions {
		if java.Matches(selection, version) {
			kept = append(kept, version.Version)
		}
	}
	if !reflect.DeepEqual(kept, []string{"25.0.4+7-zulu-fx"}) {
		t.Errorf("LTS with JavaFX = %v", kept)
	}
	node := lookup(t, "nodejs")
	if groups := node.Groups([]core.AvailableVersion{{Version: "24.22.0", Note: "LTS", HasNote: true}, {Version: "26.12.0"}}); len(groups) != 1 || groups[0].Chips[0].Count != 1 {
		t.Errorf("node groups = %+v", groups)
	}
}

func TestJavaNamesAreTakenApart(t *testing.T) {
	java := lookup(t, "java")
	cases := []struct {
		name, version, source string
		flavors               []string
	}{
		{"21.0.12.1+1-tem", "21.0.12.1+1", "tem", nil},
		{"25.0.4+9-librca-fx", "25.0.4+9", "librca", []string{"JavaFX"}},
		{"25.0.2+10", "25.0.2+10", "open", nil},
		{"28-ea+19", "28-ea+19", "open", nil},
		{"17.0.12-graal", "17.0.12", "graal", nil},
		{"25.4.4.1.1-graal", "25.4.4.1.1", "graal", nil},
		{"22.3.3-grl17", "22.3.3", "grl17", nil},
		{"11.0.10-adptj9", "11.0.10", "adptj9", nil},
		{"23.1.12.1-mandrel", "23.1.12.1", "mandrel", nil},
		{"8.0.482+8-zulu", "8.0.482+8", "zulu", nil},
		{"21.0.12+8-lnxabi2-fx", "21.0.12+8", "lnxabi2", []string{"JavaFX"}},
	}
	for _, c := range cases {
		parts, ok := java.Describe(c.name)
		if !ok || parts.Version != c.version || parts.SourceValue != c.source || !reflect.DeepEqual(parts.Flavors, c.flavors) {
			t.Errorf("Describe(%q) = %+v, %v; want %q, %q, %v", c.name, parts, ok, c.version, c.source, c.flavors)
		}
	}
	if parts, _ := java.Describe("21.0.12.1+1-tem"); parts.Source.EN != "Temurin" {
		t.Errorf("source = %+v", parts.Source)
	}
}

func TestFlutterNamesAreTakenApart(t *testing.T) {
	flutter := lookup(t, "flutter")
	cases := []struct {
		name, version string
		flavors       []string
	}{
		{"3.47.7-x64", "3.47.7", []string{"x64"}},
		{"3.50.0-0.1.pre-arm64", "3.50.0-0.1.pre", []string{"arm64"}},
		{"3.47.4", "3.47.4", nil},
		{"1.17.0-3.2.pre", "1.17.0-3.2.pre", nil},
		{"3.41.10-ohos-1.0.0", "3.41.10-ohos-1.0.0", []string{"OpenHarmony"}},
		{"3.41.9+ohos-1.0.2", "3.41.9+ohos-1.0.2", []string{"OpenHarmony"}},
	}
	for _, c := range cases {
		parts, ok := flutter.Describe(c.name)
		if !ok || parts.Version != c.version || !reflect.DeepEqual(parts.Flavors, c.flavors) || parts.SourceValue != "" {
			t.Errorf("Describe(%q) = %+v, %v", c.name, parts, ok)
		}
	}
	if _, ok := lookup(t, "nodejs").Describe("24.22.0"); ok {
		t.Error("the names of nodejs have no structure to read")
	}
}
