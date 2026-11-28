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

import "slices"

// SchemaVersion is the format of the profile files that this application
// writes and reads. A later format that keeps the meaning of these sections is
// read as far as this one knows it.
const SchemaVersion = 2

// Sections is a set of the parts that a profile file carries.
type Sections uint16

// The sections, in the order of the format.
const (
	SectionSearch Sections = 1 << iota
	SectionNaming
	SectionShorthands
	SectionVariables
	SectionLegacyFiles
	SectionLegacyStrategy
	SectionFacets
	SectionGuide
	SectionTips
	SectionDocs
)

// Has reports whether every section of other is in the set.
func (s Sections) Has(other Sections) bool { return s&other == other }

// condition is the `when` of a part of a profile: the platforms it is for. An
// empty list stands for every one.
type condition struct {
	os, arch []string
}

func (c *condition) matches(platform Platform) bool {
	if c == nil {
		return true
	}
	return (len(c.os) == 0 || slices.Contains(c.os, platform.OS)) && (len(c.arch) == 0 || slices.Contains(c.arch, platform.Arch))
}

// conditional is a part of a profile that may be for some platforms only.
type conditional[T any] struct {
	when  *condition
	value T
}

// selected are the values of the parts that are for the platform, in order,
// nil when there are none. Of the parts with the same key the first counts; a
// key that is empty is never the same as another.
func selected[T any](parts []conditional[T], platform Platform, key func(T) string) []T {
	var values []T
	var seen map[string]bool
	for _, part := range parts {
		if !part.when.matches(platform) {
			continue
		}
		if id := key(part.value); id != "" {
			if seen[id] {
				continue
			}
			if seen == nil {
				seen = make(map[string]bool)
			}
			seen[id] = true
		}
		values = append(values, part.value)
	}
	return values
}

// Document is a profile file: the sections that it carries, for every
// platform. Resolve picks the profile of one platform.
type Document struct {
	// Plugin is the name of the plugin that the file describes, empty for the
	// file that a plugin ships about itself.
	Plugin string

	sections Sections

	// search lists the variants of the search: the first for the platform wins.
	search         []conditional[*Search]
	naming         *Naming
	shorthands     []conditional[Shorthand]
	variables      []conditional[Variable]
	legacyFiles    []string
	legacyStrategy bool
	facets         []conditional[Facet]
	guide          Text
	tips           []conditional[Text]
	docs           []conditional[Link]
}

// Sections are the sections that the document carries.
func (d Document) Sections() Sections { return d.sections }

// Merge puts the documents about one plugin together, the first having the
// highest priority: each section comes whole from the first document that
// carries it.
func Merge(documents ...Document) Document {
	var merged Document
	for _, d := range documents {
		if merged.Plugin == "" {
			merged.Plugin = d.Plugin
		}
		missing := d.sections &^ merged.sections
		if missing&SectionSearch != 0 {
			merged.search = d.search
		}
		if missing&SectionNaming != 0 {
			merged.naming = d.naming
		}
		if missing&SectionShorthands != 0 {
			merged.shorthands = d.shorthands
		}
		if missing&SectionVariables != 0 {
			merged.variables = d.variables
		}
		if missing&SectionLegacyFiles != 0 {
			merged.legacyFiles = d.legacyFiles
		}
		if missing&SectionLegacyStrategy != 0 {
			merged.legacyStrategy = d.legacyStrategy
		}
		if missing&SectionFacets != 0 {
			merged.facets = d.facets
		}
		if missing&SectionGuide != 0 {
			merged.guide = d.guide
		}
		if missing&SectionTips != 0 {
			merged.tips = d.tips
		}
		if missing&SectionDocs != 0 {
			merged.docs = d.docs
		}
		merged.sections |= missing
	}
	return merged
}

// Resolve is the profile of the platform that vfox runs on. The slices of the
// profile belong to the document: do not change them.
func (d Document) Resolve(platform Platform) Profile {
	profile := Profile{
		Plugin:         d.Plugin,
		Naming:         d.naming,
		Shorthands:     selected(d.shorthands, platform, func(s Shorthand) string { return s.Name }),
		Variables:      selected(d.variables, platform, func(v Variable) string { return v.Key }),
		LegacyFiles:    d.legacyFiles,
		LegacyStrategy: d.legacyStrategy,
		Facets:         selected(d.facets, platform, func(f Facet) string { return f.ID }),
		Guide:          d.guide,
		Tips:           selected(d.tips, platform, func(Text) string { return "" }),
		Docs:           selected(d.docs, platform, func(Link) string { return "" }),
	}
	for _, variant := range d.search {
		if variant.when.matches(platform) {
			profile.Search = variant.value
			break
		}
	}
	return profile
}
