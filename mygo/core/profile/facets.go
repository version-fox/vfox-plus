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
	"strings"

	core "github.com/version-fox/vfox-plus/mygo/core"
)

// FacetKind tells where a facet reads its values from.
type FacetKind int

// The kinds of facets.
const (
	// FacetNote: the parts of the note a plugin gives a version, which the
	// plugin writes as `LTS`, `LTS, JavaFX` or `stable`.
	FacetNote FacetKind = iota
	// FacetSuffix: the last part of the version, after the last `-`, such as
	// the architecture of `3.47.4-x64`.
	FacetSuffix
)

// Facet is a filter that suits the versions of a plugin: a group of values of
// which the user chooses some. A version passes when it has one of the chosen
// values of every group that has a choice.
type Facet struct {
	// ID identifies the facet among the facets of the plugin.
	ID string
	// Label names the group.
	Label Text
	// Kind tells where the values are read from.
	Kind FacetKind
	// Values are the values the facet knows, in the order they are offered, as
	// they are written; they are compared ignoring case.
	Values []string
}

// Group is a facet with the values that some of the listed versions have.
type Group struct {
	ID    string
	Label Text
	Chips []Chip
}

// Chip is a value of a group, and how many of the listed versions have it.
type Chip struct {
	Value string
	Count int
}

// Selection is what the user chose: the values of every group.
type Selection map[string]map[string]bool

// Empty reports whether nothing is chosen.
func (s Selection) Empty() bool {
	for _, values := range s {
		if len(values) > 0 {
			return false
		}
	}
	return true
}

// Has reports whether the value of the group is chosen.
func (s Selection) Has(group, value string) bool {
	return s[group][value]
}

// Toggled returns the selection with the value of the group chosen if it was
// not, and not chosen if it was. The receiver is left alone.
func (s Selection) Toggled(group, value string) Selection {
	result := make(Selection, len(s)+1)
	for id, values := range s {
		copied := make(map[string]bool, len(values)+1)
		for chosen := range values {
			copied[chosen] = true
		}
		result[id] = copied
	}
	if result[group] == nil {
		result[group] = map[string]bool{}
	}
	if result[group][value] {
		delete(result[group], value)
		if len(result[group]) == 0 {
			delete(result, group)
		}
	} else {
		result[group][value] = true
	}
	return result
}

// Groups returns the facets of the profile with the values that at least one
// of the versions has, and how many have them. A facet no version has a value
// of is left out.
func (p Profile) Groups(versions []core.AvailableVersion) []Group {
	var groups []Group
	for _, facet := range p.Facets {
		counts := make([]int, len(facet.Values))
		for _, version := range versions {
			for i, value := range facet.Values {
				if facet.has(version, value) {
					counts[i]++
				}
			}
		}
		group := Group{ID: facet.ID, Label: facet.Label}
		for i, value := range facet.Values {
			if counts[i] > 0 {
				group.Chips = append(group.Chips, Chip{Value: value, Count: counts[i]})
			}
		}
		if len(group.Chips) > 0 {
			groups = append(groups, group)
		}
	}
	return groups
}

// Matches reports whether the version passes the selection.
func (p Profile) Matches(selection Selection, version core.AvailableVersion) bool {
	for _, facet := range p.Facets {
		chosen := selection[facet.ID]
		if len(chosen) == 0 {
			continue
		}
		passed := false
		for value := range chosen {
			if facet.has(version, value) {
				passed = true
				break
			}
		}
		if !passed {
			return false
		}
	}
	return true
}

func (f Facet) has(version core.AvailableVersion, value string) bool {
	switch f.Kind {
	case FacetSuffix:
		_, suffix, found := cutLast(version.Version, "-")
		return found && strings.EqualFold(suffix, value)
	default:
		for _, token := range NoteTokens(version.Note) {
			if strings.EqualFold(token, value) {
				return true
			}
		}
		return false
	}
}

// NoteTokens splits the note of a version at the commas and semicolons that are
// not inside parentheses: `LTS, JavaFX` gives LTS and JavaFX, and
// `JDK 26+35; ABI2 (Linux 5.10 UAPI, glibc 2.34); JavaFX` gives three parts.
func NoteTokens(note string) []string {
	var tokens []string
	depth, start := 0, 0
	add := func(end int) {
		if token := strings.TrimSpace(note[start:end]); token != "" {
			tokens = append(tokens, token)
		}
	}
	for i, r := range note {
		switch r {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',', ';':
			if depth == 0 {
				add(i)
				start = i + 1
			}
		}
	}
	add(len(note))
	return tokens
}

// cutLast splits text around the last occurrence of separator.
func cutLast(text, separator string) (before, after string, found bool) {
	i := strings.LastIndex(text, separator)
	if i < 0 {
		return text, "", false
	}
	return text[:i], text[i+len(separator):], true
}
