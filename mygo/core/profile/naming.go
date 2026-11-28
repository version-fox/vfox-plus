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
	"strings"
)

// Naming is how the version names of a plugin are put together: rules, the
// first of which that matches a name reads it.
type Naming struct {
	// Sources are the distributions that the group `source` of a rule names.
	Sources []NameSource
	// DefaultSource is the distribution of a name whose rule found none, empty
	// when there is none.
	DefaultSource string
	// Rules are tried in order.
	Rules []NameRule
}

// NameSource is a distribution that version names name.
type NameSource struct {
	// Value is the text in the name, such as `tem`.
	Value string
	// Name is how the distribution is called.
	Name Text
}

// NameRule reads the names that its pattern matches. The group `version` of
// the pattern is the version, the whole name when there is no such group; the
// group `source` is the distribution; every other named group that matches
// text adds a flavor, such as JavaFX or an architecture.
type NameRule struct {
	Pattern *regexp.Regexp
	// Flavors are added to every name that the rule reads.
	Flavors []string
	// Labels name captured flavors, by their text in lower case. A captured
	// text without a label is shown as it is.
	Labels map[string]string
}

// Parts is a version name taken apart.
type Parts struct {
	// Version is the name without the distribution and the flavors.
	Version string
	// Source names the distribution the version is of, when the plugin has
	// distributions.
	Source Text
	// SourceValue is the short name of the distribution, such as `tem`.
	SourceValue string
	// Flavors say what else the name tells: `JavaFX`, `x64`, `OpenHarmony`.
	Flavors []string
}

// The names of the groups of a NameRule that are not flavors.
const (
	groupVersion = "version"
	groupSource  = "source"
)

// Describe takes a version name apart the way the plugin puts it together. It
// reports false for a plugin whose names have no structure that is read.
func (p Profile) Describe(name string) (Parts, bool) {
	if p.Naming == nil || len(p.Naming.Rules) == 0 {
		return Parts{}, false
	}
	return p.Naming.describe(name), true
}

func (n *Naming) describe(name string) Parts {
	parts := Parts{Version: name}
	for _, rule := range n.Rules {
		if rule.Pattern == nil {
			continue
		}
		match := rule.Pattern.FindStringSubmatchIndex(name)
		if match == nil {
			continue
		}
		parts.Flavors = slices.Clone(rule.Flavors)
		for i, group := range rule.Pattern.SubexpNames() {
			if group == "" || match[2*i] < 0 {
				continue
			}
			value := name[match[2*i]:match[2*i+1]]
			switch {
			case group == groupVersion:
				parts.Version = value
			case group == groupSource:
				parts.SourceValue = value
			case value != "":
				if label, ok := rule.Labels[strings.ToLower(value)]; ok {
					value = label
				}
				parts.Flavors = append(parts.Flavors, value)
			}
		}
		if parts.SourceValue == "" {
			parts.SourceValue = n.DefaultSource
		}
		if parts.SourceValue != "" {
			parts.Source = n.sourceName(parts.SourceValue)
		}
		break
	}
	return parts
}

// sourceName is how the distribution with the value is called: the value itself
// when no source has it.
func (n *Naming) sourceName(value string) Text {
	for _, source := range n.Sources {
		if source.Value == value {
			return source.Name
		}
	}
	return Text{EN: value}
}

// sourcesPlaceholder stands in a pattern for the values of the sources.
const sourcesPlaceholder = "{sources}"

// expandSources replaces `{sources}` in the pattern with the alternation of
// the values of the sources, the longest first.
func expandSources(pattern string, sources []NameSource) string {
	if !strings.Contains(pattern, sourcesPlaceholder) {
		return pattern
	}
	values := make([]string, 0, len(sources))
	for _, source := range sources {
		values = append(values, source.Value)
	}
	slices.SortStableFunc(values, func(a, b string) int { return len(b) - len(a) })
	for i, value := range values {
		values[i] = regexp.QuoteMeta(value)
	}
	return strings.ReplaceAll(pattern, sourcesPlaceholder, "(?:"+strings.Join(values, "|")+")")
}
