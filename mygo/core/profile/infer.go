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
	"regexp"
	"slices"
	"sort"
	"strings"

	core "github.com/version-fox/vfox-plus/mygo/core"
)

// What can be told about a plugin that no profile of the application describes,
// from what vfox reports and from its files:
//
//   - facets from the versions a search lists (notes become a note filter,
//     shared suffixes after the last `-` a variant filter),
//   - environment variables from the os.getenv calls of its Lua sources, and
//     legacy version files from PLUGIN.legacyFilenames of its metadata.lua
//     (see ReadPlugin),
//   - the guide and the documentation from its description and homepage.

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
	for _, version := range versions {
		// The tail after the last "-", if any: a shared tail such as x64 is
		// a variant worth filtering by. (Written out instead of cutLast so
		// that the shared helper keeps its separator parameter.)
		if i := strings.LastIndex(version.Version, "-"); i >= 0 {
			suffix := version.Version[i+1:]
			if !isVariantSuffix(suffix) {
				continue
			}
			key := strings.ToLower(suffix)
			if _, ok := suffixSeen[key]; !ok {
				suffixSeen[key] = suffix
			}
		}
	}
	var facets []Facet
	// A single note is still worth offering: it answers "which of these are
	// LTS?".
	if len(noteOrder) >= 1 && len(noteOrder) <= 20 {
		sort.Strings(noteOrder)
		facets = append(facets, Facet{
			ID:     "note",
			Label:  Text{EN: "Note", ZH: "备注"},
			Kind:   FacetNote,
			Values: slices.Clone(noteOrder),
		})
	}
	if len(suffixSeen) >= 2 && len(suffixSeen) <= 10 {
		var values []string
		for _, display := range suffixSeen {
			values = append(values, display)
		}
		sort.Strings(values)
		facets = append(facets, Facet{
			ID:     "variant",
			Label:  Text{EN: "Variant", ZH: "变体"},
			Kind:   FacetSuffix,
			Values: values,
		})
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
		} else if (r < '0' || r > '9') && r != '+' && r != '.' {
			return false
		}
	}
	// A purely numeric tail such as `-1` of `9.0.0-rc-1` is a version
	// number, not a variant.
	return letter
}

// EffectiveFacets returns the facets to filter versions by: those of the
// profile when it has any, those inferred from the versions otherwise.
func EffectiveFacets(profile Profile, versions []core.AvailableVersion) []Facet {
	if len(profile.Facets) > 0 {
		return profile.Facets
	}
	return InferFacets(versions)
}

// InferredProfile builds a minimal profile for a plugin that no profile
// describes, from what vfox already reports: its description and homepage
// become the guide and the documentation link, its listed versions the facets.
// Variables and legacy files come from its files (see ReadPlugin).
func InferredProfile(plugin string, info core.PluginInfo, versions []core.AvailableVersion) Profile {
	profile := Profile{Plugin: plugin}
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
// sources. Only names that look like plugin settings are kept (see
// IsPluginEnvKey).
func ScanEnvKeys(sources map[string]string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, content := range sources {
		for _, match := range getenvPattern.FindAllStringSubmatch(content, -1) {
			key := match[1]
			if seen[key] || !IsPluginEnvKey(key) {
				continue
			}
			seen[key] = true
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

// IsPluginEnvKey reports whether key looks like a setting a vfox plugin reads:
// VFOX_*, FLUTTER_*, or a name that says a mirror, a base address, a store, a
// proxy or GitHub, and that is not reserved (see IsReservedVariable): PATH and
// HOME are not settings of a plugin.
func IsPluginEnvKey(key string) bool {
	if IsReservedVariable(key) {
		return false
	}
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

// GenericVariable returns an editable variable for a plugin setting that no
// profile describes: an address when the name says one, any line of text
// otherwise. It lets SetPluginVariable accept mirrors of third-party plugins
// that no profile names.
func GenericVariable(key string) (Variable, bool) {
	if !IsPluginEnvKey(key) {
		return Variable{}, false
	}
	variable := Variable{
		Key:   key,
		Label: Text{EN: humanizeEnvKey(key)},
		Help: Text{
			EN: "Read by the plugin from the environment of vfox. Leave it empty to use the default.",
			ZH: "插件从 vfox 进程的环境里读取。留空则使用默认值。",
		},
		Kind: VariableText,
	}
	if looksLikeURLSetting(key) {
		variable.Kind, variable.Example = VariableURL, "https://"
	}
	return variable, true
}

// GenericVariableForPlugin is GenericVariable for the plugin: the key must
// name the plugin (VFOX_GRADLE_MIRRORS for gradle) so that the mirror of one
// plugin is not set on another. An empty plugin accepts any plugin setting.
func GenericVariableForPlugin(plugin, key string) (Variable, bool) {
	variable, ok := GenericVariable(key)
	if !ok {
		return Variable{}, false
	}
	plugin = strings.ToUpper(strings.TrimSpace(plugin))
	if plugin == "" || strings.Contains(strings.ToUpper(key), plugin) {
		return variable, true
	}
	return Variable{}, false
}

// InferredVariables turns scanned environment keys into editable variables (see
// GenericVariable).
func InferredVariables(keys []string) []Variable {
	var result []Variable
	for _, key := range keys {
		if variable, ok := GenericVariable(key); ok {
			result = append(result, variable)
		}
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
