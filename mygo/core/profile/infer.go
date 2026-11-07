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
	"regexp"
	"sort"
	"strings"

	core "github.com/version-fox/vfox-plus/mygo/core"
)

// Generic adaptation for plugins without a built-in profile.
//
// The built-in profiles of java, nodejs, flutter and gradle stay curated,
// but any other plugin — including future third-party ones — works without
// a code change:
//
//   - facets are inferred from the versions a search lists (notes become a
//     note filter, shared suffixes after the last `-` a variant filter),
//   - environment variables are inferred by scanning the Lua sources of the
//     installed plugin for os.getenv("VFOX_*"), and legacy version files by
//     reading PLUGIN.legacyFilenames from its metadata.lua,
//   - an optional JSON profile (see external.go) overlays these inferences
//     with curated texts, and the built-in registry overlays that.
//
// This mirrors what https://mise-versions.jdx.dev does for mise: the tool
// list needs no per-tool UI code, because backends are declarative data.

// InferFacets builds filter facets from listed versions, without any
// plugin-specific knowledge. Notes that plugins attach to versions
// (for example `LTS`) become a note facet; shared suffixes after the last
// `-` (for example `x64`) become a variant facet. It returns nil when the
// versions carry nothing worth filtering by.
func InferFacets(versions []core.AvailableVersion) []Facet {
	noteSeen := make(map[string]string)
	noteOrder := []string{}
	for _, version := range versions {
		if !version.HasNote {
			continue
		}
		for _, token := range NoteTokens(version.Note) {
			key := strings.ToLower(token)
			if _, ok := noteSeen[key]; !ok {
				noteSeen[key] = token
				noteOrder = append(noteOrder, token)
			}
		}
	}
	suffixSeen := make(map[string]string)
	suffixCounts := make(map[string]int)
	for _, version := range versions {
		if _, suffix, found := cutLast(version.Version, "-"); found {
			if !isVariantSuffix(suffix) {
				continue
			}
			key := strings.ToLower(suffix)
			if _, ok := suffixSeen[key]; !ok {
				suffixSeen[key] = suffix
			}
			suffixCounts[key]++
		}
	}
	var facets []Facet
	if len(noteOrder) >= 2 && len(noteOrder) <= 20 {
		sort.Strings(noteOrder)
		facets = append(facets, Facet{
			ID:     "note",
			Label:  Text{EN: "Note", ZH: "备注"},
			Kind:   FacetNote,
			Values: append([]string(nil), noteOrder...),
		})
	} else if len(noteOrder) == 1 {
		// A single note is still worth offering: it answers "which of these
		// are LTS?".
		facets = append(facets, Facet{
			ID:     "note",
			Label:  Text{EN: "Note", ZH: "备注"},
			Kind:   FacetNote,
			Values: append([]string(nil), noteOrder...),
		})
	}
	if len(suffixSeen) >= 2 {
		var values []string
		for _, display := range suffixSeen {
			values = append(values, display)
		}
		if len(values) >= 2 && len(values) <= 10 {
			sort.Strings(values)
			facets = append(facets, Facet{
				ID:     "variant",
				Label:  Text{EN: "Variant", ZH: "变体"},
				Kind:   FacetSuffix,
				Values: values,
			})
		}
	}
	return facets
}

func isVariantSuffix(suffix string) bool {
	if suffix == "" || len(suffix) > 16 {
		return false
	}
	letter := false
	for _, r := range suffix {
		if r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' {
			letter = true
		} else if !(r >= '0' && r <= '9' || r == '+' || r == '.') {
			return false
		}
	}
	// A purely numeric tail such as `-1` of `9.0.0-rc-1` is a version
	// number, not a variant.
	return letter
}

// InferredProfile builds a minimal profile for a plugin without a built-in
// or external one, from what vfox already reports: its description and
// homepage become the guide and the documentation link, its listed versions
// the facets. Variables and legacy files stay empty here; they come from
// ScanPluginDir when the plugin is installed locally.
func InferredProfile(plugin string, info core.PluginInfo, versions []core.AvailableVersion) Profile {
	profile := Profile{Plugin: plugin, Naming: NamingPlain}
	if info.Description != "" {
		profile.Guide = Text{EN: info.Description}
	} else {
		profile.Guide = Text{
			EN: "Install a version from the list, or by the name the plugin resolves.",
			ZH: "从列表安装一个版本，或按插件能解析的名称安装。",
		}
	}
	if info.Homepage != "" {
		profile.Docs = []Link{{Title: Text{EN: "Homepage", ZH: "主页"}, URL: Text{EN: info.Homepage}}}
	}
	profile.Facets = InferFacets(versions)
	return profile
}

// EffectiveFacets returns the facets to filter versions by: the ones of the
// profile when it has any, the inferred ones otherwise. It lets interfaces
// offer filters for plugins without a built-in profile.
func EffectiveFacets(profile Profile, hasProfile bool, versions []core.AvailableVersion) []Facet {
	if hasProfile && len(profile.Facets) > 0 {
		return profile.Facets
	}
	return InferFacets(versions)
}

// getenvPattern finds os.getenv("NAME") in Lua sources.
var getenvPattern = regexp.MustCompile(`os\.getenv\(\s*["']([A-Za-z_][A-Za-z0-9_]*)["']`)

// legacyPattern reads PLUGIN.legacyFilenames = { ".nvmrc", ... } out of
// metadata.lua without running Lua.
var legacyPattern = regexp.MustCompile(`(?s)legacyFilenames\s*=\s*\{(.*?)\}`)

var quotedPattern = regexp.MustCompile(`["']([^"']+)["']`)

// ParseLegacyFilenames extracts the file names of PLUGIN.legacyFilenames
// from the text of a metadata.lua.
func ParseLegacyFilenames(metadata string) []string {
	match := legacyPattern.FindStringSubmatch(metadata)
	if match == nil {
		return nil
	}
	var result []string
	seen := make(map[string]bool)
	for _, name := range quotedPattern.FindAllStringSubmatch(match[1], -1) {
		value := strings.TrimSpace(name[1])
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

// ScanEnvKeys extracts the environment variables a plugin reads from its Lua
// sources. Only names that look like plugin settings are kept: VFOX_*,
// FLUTTER_*, or names containing MIRROR, MIRRORS, URL, BASE_URL or PROXY.
// Generic names such as PATH and HOME are left out: they are not settings of
// the plugin.
func ScanEnvKeys(sources map[string]string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, content := range sources {
		for _, match := range getenvPattern.FindAllStringSubmatch(content, -1) {
			key := match[1]
			if seen[key] || !isPluginEnvKey(key) {
				continue
			}
			seen[key] = true
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

func isPluginEnvKey(key string) bool {
	upper := strings.ToUpper(key)
	if strings.HasPrefix(upper, "VFOX_") || strings.HasPrefix(upper, "FLUTTER_") {
		return true
	}
	for _, fragment := range []string{"MIRROR", "MIRRORS", "BASE_URL", "STORAGE", "PROXY", "GITHUB"} {
		if strings.Contains(upper, fragment) {
			return true
		}
	}
	return false
}

// IsPluginEnvKey reports whether key looks like a setting a vfox plugin
// reads, such as VFOX_GRADLE_MIRRORS. It is the same check ScanEnvKeys uses
// to keep plugin settings and leave out PATH and HOME.
func IsPluginEnvKey(key string) bool { return isPluginEnvKey(key) }

// GenericVariable returns an editable variable for a plugin setting without
// a curated profile: a URL setting when the name suggests one, a free-form
// choice otherwise. It lets SetPluginVariable accept mirrors of third-party
// plugins that no profile names yet.
func GenericVariable(key string) (Variable, bool) {
	if !isPluginEnvKey(key) {
		return Variable{}, false
	}
	if looksLikeURLSetting(key) {
		return Variable{
			Key:   key,
			Label: Text{EN: humanizeEnvKey(key)},
			Help: Text{
				EN: "Read by the plugin from the environment of vfox. Leave it empty to use the default.",
				ZH: "插件从 vfox 进程的环境里读取。留空则使用默认值。",
			},
			Kind:    VariableURL,
			Example: "https://",
		}, true
	}
	return Variable{
		Key:   key,
		Label: Text{EN: humanizeEnvKey(key)},
		Help: Text{
			EN: "Read by the plugin from the environment of vfox. Leave it empty to use the default.",
			ZH: "插件从 vfox 进程的环境里读取。留空则使用默认值。",
		},
		Kind: VariableChoice,
	}, true
}

// GenericVariableForPlugin is GenericVariable for the plugin: the key must
// name the plugin (VFOX_GRADLE_MIRRORS for gradle) so that the mirror of one
// plugin is not set on another. An empty plugin accepts any plugin setting,
// for plugins without a profile yet.
func GenericVariableForPlugin(plugin, key string) (Variable, bool) {
	variable, ok := GenericVariable(key)
	if !ok {
		return Variable{}, false
	}
	plugin = strings.ToUpper(strings.TrimSpace(plugin))
	if plugin == "" {
		return variable, true
	}
	if strings.Contains(strings.ToUpper(key), plugin) {
		return variable, true
	}
	return Variable{}, false
}

// InferredVariables turns scanned environment keys into editable variables.
// Names with MIRROR, URL, BASE_URL, STORAGE or PROXY take addresses;
// anything else takes one of no preset values, so the interface offers a
// plain text field. Labels and help are generic: an external profile file
// (see external.go) replaces them with curated texts without a code change.
func InferredVariables(keys []string) []Variable {
	var result []Variable
	for _, key := range keys {
		label := Text{EN: humanizeEnvKey(key)}
		help := Text{
			EN: "Read by the plugin from the environment of vfox. Leave it empty to use the default.",
			ZH: "插件从 vfox 进程的环境里读取。留空则使用默认值。",
		}
		kind := VariableURL
		if !looksLikeURLSetting(key) {
			// A free-form value: no presets, so the interface offers text.
			kind = VariableChoice
		}
		variable := Variable{Key: key, Label: label, Help: help, Kind: kind}
		if kind == VariableURL {
			variable.Example = "https://"
		}
		result = append(result, variable)
	}
	return result
}

func looksLikeURLSetting(key string) bool {
	upper := strings.ToUpper(key)
	for _, fragment := range []string{"MIRROR", "MIRRORS", "URL", "BASE_URL", "STORAGE", "PROXY"} {
		if strings.Contains(upper, fragment) {
			return true
		}
	}
	return false
}

func humanizeEnvKey(key string) string {
	lower := strings.ToLower(key)
	lower = strings.TrimPrefix(lower, "vfox_")
	lower = strings.TrimPrefix(lower, "flutter_")
	words := strings.Split(lower, "_")
	for i, word := range words {
		if word != "" {
			words[i] = strings.ToUpper(word[:1]) + word[1:]
		}
	}
	return strings.Join(words, " ")
}

// PluginHome returns the directory that holds the plugins of vfox: VFOX_HOME
// when it is set, the legacy ~/.version-fox when it exists, ~/.vfox
// otherwise (under %USERPROFILE% on Windows).
func PluginHome(environment map[string]string) string {
	if home := strings.TrimSpace(environment["VFOX_HOME"]); home != "" {
		return home
	}
	home := pluginUserHome(environment)
	if home == "" {
		return ""
	}
	legacy := filepath.Join(home, ".version-fox")
	if info, err := os.Stat(legacy); err == nil && info.IsDir() {
		return legacy
	}
	return filepath.Join(home, ".vfox")
}

func pluginUserHome(environment map[string]string) string {
	if home := strings.TrimSpace(environment["HOME"]); home != "" {
		return home
	}
	return strings.TrimSpace(environment["USERPROFILE"])
}

// PluginDir returns the directory of the installed plugin name.
func PluginDir(environment map[string]string, plugin string) string {
	home := PluginHome(environment)
	if home == "" || plugin == "" {
		return ""
	}
	return filepath.Join(home, "plugin", plugin)
}

// ScanPluginDir reads the installed plugin for generic adaptation: its
// legacy version files from metadata.lua and the environment variables its
// Lua sources read. Missing files give empty results, never an error.
func ScanPluginDir(directory string) (legacy []string, envKeys []string) {
	if directory == "" {
		return nil, nil
	}
	if content, err := os.ReadFile(filepath.Join(directory, "metadata.lua")); err == nil { //nolint:gosec // The path is the installed plugin.
		legacy = ParseLegacyFilenames(string(content))
	}
	sources := make(map[string]string)
	_ = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(strings.ToLower(info.Name()), ".lua") {
			return nil
		}
		if content, err := os.ReadFile(path); err == nil { //nolint:gosec // The path is the installed plugin.
			sources[path] = string(content)
		}
		return nil
	})
	envKeys = ScanEnvKeys(sources)
	return legacy, envKeys
}
