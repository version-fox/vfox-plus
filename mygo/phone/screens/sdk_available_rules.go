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

package screens

// The rules of the installable versions, as pure functions: the checks and the
// matches that the web's sdks.ts makes, which the screen calls each frame.

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/version-fox/vfox-plus/mygo/phone/i18n"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// availPageSize is how many versions the list shows at first, and availPageStep how
// many more each "show more" adds; they are the web's 200 and 400.
const (
	availPageSize = 200
	availPageStep = 400
)

// availChosen holds the filters that are on: for each facet, the values that are
// chosen. A value that is not chosen is not in the map.
type availChosen map[string]map[string]bool

// availFacet is a group of the filters of the versions: the facet, and the values
// that at least one version has, in the order of the profile.
type availFacet struct {
	ID     string
	Label  remote.Text
	Values []string
}

// availChinese reports whether the interface is in Chinese, which picks the text of
// the profiles and of the plugins.
func availChinese() bool {
	return i18n.Current() == i18n.Chinese
}

// availSpace reports whether r is white space as JavaScript's \s is, which the web's
// trim and version name check use. Go's unicode.IsSpace differs: it counts U+0085,
// and it does not count the byte order mark.
func availSpace(r rune) bool {
	switch r {
	case '\t', '\n', '\v', '\f', '\r', ' ', 0x00A0, 0x1680, 0x2028, 0x2029, 0x202F, 0x205F, 0x3000, 0xFEFF:
		return true
	}
	return r >= 0x2000 && r <= 0x200A
}

// availTrim removes the white space at both ends of s, as the web's trim does.
func availTrim(s string) string {
	return strings.TrimFunc(s, availSpace)
}

// availValidSpec reports whether v can name a version to install. It follows the
// message that the web shows for a name it refuses: the name has no spaces and does
// not start with a dash. So v is not empty, it has no white space anywhere, and it
// does not start with a dash or an @. The web's own check (validSpec) looks at the
// start only; the phone follows the message, on purpose. v is the trimmed text.
func availValidSpec(v string) bool {
	if v == "" {
		return false
	}
	first, _ := utf8.DecodeRuneInString(v)
	return first != '-' && first != '@' && !strings.ContainsFunc(v, availSpace)
}

// availShorthand returns the shorthand of a profile that is named name, the name
// that the install by name offers as a choice.
func availShorthand(shorthands []remote.Shorthand, name string) (remote.Shorthand, bool) {
	for _, shorthand := range shorthands {
		if shorthand.Name == name {
			return shorthand, true
		}
	}
	return remote.Shorthand{}, false
}

// availSearchValue returns the value of the search choice that the plugin lists now:
// the current one, or the default when none is current. It is "" for a plugin that
// has no choice of search, or no profile.
func availSearchValue(profile *remote.Profile) string {
	if profile == nil || profile.Search == nil {
		return ""
	}
	if profile.Search.Current != "" {
		return profile.Search.Current
	}
	return profile.Search.Default
}

// availChoiceOf returns the choice of the search whose value is value.
func availChoiceOf(search *remote.ProfileSearch, value string) (remote.Choice, bool) {
	if search == nil {
		return remote.Choice{}, false
	}
	for _, choice := range search.Choices {
		if choice.Value == value {
			return choice, true
		}
	}
	return remote.Choice{}, false
}

// availChoiceMatches reports whether a choice matches the query q, which is trimmed
// and in lower case. The web's chooser matches the value, the aliases, and the names
// and the makers in both languages.
func availChoiceMatches(choice remote.Choice, q string) bool {
	if q == "" {
		return true
	}
	texts := append([]string{choice.Value, choice.Name.EN, choice.Name.ZH, choice.Maker.EN, choice.Maker.ZH}, choice.Aliases...)
	for _, text := range texts {
		if strings.Contains(strings.ToLower(text), q) {
			return true
		}
	}
	return false
}

// availFacets returns the groups of filters that the profile names, in its order,
// each with the values that some of the versions has. A group with none is left out.
func availFacets(profile *remote.Profile, versions []remote.AvailableVersion) []availFacet {
	if profile == nil {
		return nil
	}
	var groups []availFacet
	for _, facet := range profile.Facets {
		has := map[string]bool{}
		for _, version := range versions {
			for _, value := range version.Facets[facet.ID] {
				has[value] = true
			}
		}
		var values []string
		for _, value := range facet.Values {
			if has[value] {
				values = append(values, value)
			}
		}
		if len(values) > 0 {
			groups = append(groups, availFacet{ID: facet.ID, Label: facet.Label, Values: values})
		}
	}
	return groups
}

// availToggle chooses the value of a facet, or drops it when it is chosen already.
func availToggle(chosen availChosen, facet, value string) {
	if chosen[facet][value] {
		delete(chosen[facet], value)
		if len(chosen[facet]) == 0 {
			delete(chosen, facet)
		}
		return
	}
	if chosen[facet] == nil {
		chosen[facet] = map[string]bool{}
	}
	chosen[facet][value] = true
}

// availVersionMatches reports whether a version passes the search and the filters.
// The version or its note must contain q, which is in lower case. For each facet that
// has values chosen, the version must have one of them: the values of one facet are
// alternatives, and the facets are all required.
func availVersionMatches(version remote.AvailableVersion, q string, chosen availChosen) bool {
	for facet, values := range chosen {
		if len(values) == 0 {
			continue
		}
		pass := false
		for _, have := range version.Facets[facet] {
			if values[have] {
				pass = true
				break
			}
		}
		if !pass {
			return false
		}
	}
	if q == "" {
		return true
	}
	return strings.Contains(strings.ToLower(version.Version), q) ||
		strings.Contains(strings.ToLower(version.Note), q)
}

// availRows returns the versions that pass the search and the filters, in their
// order. q is the search in lower case.
func availRows(versions []remote.AvailableVersion, q string, chosen availChosen) []remote.AvailableVersion {
	rows := make([]remote.AvailableVersion, 0, len(versions))
	for _, version := range versions {
		if availVersionMatches(version, q, chosen) {
			rows = append(rows, version)
		}
	}
	return rows
}

// availInstalled returns the versions of the plugin name that are installed, as the
// inventory has them. It is empty before the inventory loads.
func availInstalled(inventory remote.Inventory, name string) map[string]bool {
	installed := map[string]bool{}
	for _, sdk := range inventory.Installed.Value {
		if sdk.Name != name {
			continue
		}
		for _, version := range sdk.Versions {
			installed[version] = true
		}
	}
	return installed
}

// availAdditions is the text of the tools that come with a version, such as npm,
// with their versions where they have one.
func availAdditions(additions []remote.VersionAddition) string {
	parts := make([]string, 0, len(additions))
	for _, addition := range additions {
		if addition.Version != "" {
			parts = append(parts, addition.Name+" "+addition.Version)
		} else {
			parts = append(parts, addition.Name)
		}
	}
	return strings.Join(parts, ", ")
}

// availSourceName is the name of the source that a version comes from, in the
// language of the interface, or "" when the version names none.
func availSourceName(version remote.AvailableVersion) string {
	if version.Parts == nil || version.Parts.Source == nil {
		return ""
	}
	return version.Parts.Source.Name.In(availChinese())
}

// availCount is the text of a count, such as "3 versions": the message with the
// number in its place.
func availCount(key string, n int) string {
	return strings.ReplaceAll(i18n.Text(key), "{n}", strconv.Itoa(n))
}
