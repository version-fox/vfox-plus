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

// Reading profile files. What a file says that this application cannot read
// costs only the part that says it: a bad item of a list its item, a section
// with nothing good in it that section, which then comes from the next file
// about the plugin. Unknown fields are ignored, so that a newer file stays
// readable. Every problem is reported, which `vfxtool profile lint` shows.

package profile

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// Problem is something in a profile file that the application cannot read, or
// that deserves a look.
type Problem struct {
	// Path is where it is, such as `variables[1].key`.
	Path string
	// Message says what is wrong.
	Message string
	// Warning marks a problem whose part is read anyway.
	Warning bool
}

// String is the problem on one line.
func (p Problem) String() string {
	kind := "error"
	if p.Warning {
		kind = "warning"
	}
	if p.Path == "" {
		return kind + ": " + p.Message
	}
	return kind + ": " + p.Path + ": " + p.Message
}

// ParseDocument reads a profile file. It fails only for content that is not a
// JSON object, or too large to be a profile; what else it cannot read is left
// out and reported.
func ParseDocument(content []byte) (Document, []Problem, error) {
	if len(content) > maxFile {
		return Document{}, nil, fmt.Errorf("a profile takes at most %d bytes", maxFile)
	}
	content = bytes.TrimPrefix(content, []byte{0xef, 0xbb, 0xbf}) // The mark of UTF-8 that some editors write.
	var object map[string]json.RawMessage
	if err := json.Unmarshal(content, &object); err != nil {
		return Document{}, nil, fmt.Errorf("a profile is a JSON object: %w", err)
	}
	if object == nil {
		return Document{}, nil, errors.New("a profile is a JSON object")
	}
	p := &parser{}
	p.warnUnknown("", object, "$schema", "schema", "plugin", "search", "naming", "shorthands", "variables",
		"legacyFiles", "legacyStrategy", "facets", "guide", "tips", "docs")
	var d Document
	if raw, ok := object["schema"]; ok {
		var schema int
		if err := json.Unmarshal(raw, &schema); err != nil {
			p.errorf("schema", "is not a number")
		} else if schema > SchemaVersion {
			p.warnf("schema", "is newer than %d: only what that format knows is read", SchemaVersion)
		}
	}
	if name := p.stringField("", object, "plugin", maxName); name != "" {
		if isPluginName(name) {
			d.Plugin = name
		} else {
			p.errorf("plugin", "is not the name of a plugin")
		}
	}
	if raw, ok := object["search"]; ok {
		if variants, ok := p.search(raw); ok {
			d.search, d.sections = variants, d.sections|SectionSearch
		}
	}
	if raw, ok := object["naming"]; ok {
		if naming, ok := p.naming(raw); ok {
			d.naming, d.sections = naming, d.sections|SectionNaming
		}
	}
	if raw, ok := object["shorthands"]; ok {
		if parts, ok := list(p, "shorthands", raw, 100, p.shorthand); ok {
			d.shorthands, d.sections = parts, d.sections|SectionShorthands
		}
	}
	if raw, ok := object["variables"]; ok {
		if parts, ok := list(p, "variables", raw, 50, p.variable); ok {
			d.variables, d.sections = parts, d.sections|SectionVariables
		}
	}
	if raw, ok := object["legacyFiles"]; ok {
		if parts, ok := list(p, "legacyFiles", raw, 50, p.fileName); ok {
			for _, part := range parts {
				d.legacyFiles = append(d.legacyFiles, part.value)
			}
			d.sections |= SectionLegacyFiles
		}
	}
	if raw, ok := object["legacyStrategy"]; ok {
		if err := json.Unmarshal(raw, &d.legacyStrategy); err != nil {
			p.errorf("legacyStrategy", "is not true or false")
		} else {
			d.sections |= SectionLegacyStrategy
		}
	}
	if raw, ok := object["facets"]; ok {
		if parts, ok := list(p, "facets", raw, 20, p.facet); ok {
			d.facets, d.sections = parts, d.sections|SectionFacets
		}
	}
	if raw, ok := object["guide"]; ok {
		if guide, ok := p.text("guide", raw); ok {
			d.guide, d.sections = guide, d.sections|SectionGuide
		}
	}
	if raw, ok := object["tips"]; ok {
		if parts, ok := list(p, "tips", raw, 50, p.tip); ok {
			d.tips, d.sections = parts, d.sections|SectionTips
		}
	}
	if raw, ok := object["docs"]; ok {
		if parts, ok := list(p, "docs", raw, 50, p.link); ok {
			d.docs, d.sections = parts, d.sections|SectionDocs
		}
	}
	return d, p.problems, nil
}

type parser struct {
	problems []Problem
}

func (p *parser) errorf(path, format string, arguments ...any) {
	p.problems = append(p.problems, Problem{Path: path, Message: fmt.Sprintf(format, arguments...)})
}

func (p *parser) warnf(path, format string, arguments ...any) {
	p.problems = append(p.problems, Problem{Path: path, Message: fmt.Sprintf(format, arguments...), Warning: true})
}

// field is the path of a field of the object at path.
func field(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}

func (p *parser) warnUnknown(path string, object map[string]json.RawMessage, known ...string) {
	for _, name := range slices.Sorted(maps.Keys(object)) {
		if !slices.Contains(known, name) {
			p.warnf(field(path, name), "is not a field of the format")
		}
	}
}

// object reads raw as an object whose fields are known.
func (p *parser) object(path string, raw json.RawMessage, known ...string) (map[string]json.RawMessage, bool) {
	var object map[string]json.RawMessage
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		p.errorf(path, "is not an object")
		return nil, false
	}
	p.warnUnknown(path, object, known...)
	return object, true
}

// items reads raw as a list of at most limit items.
func (p *parser) items(path string, raw json.RawMessage, limit int) ([]json.RawMessage, bool) {
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		p.errorf(path, "is not a list")
		return nil, false
	}
	if len(items) > limit {
		p.errorf(path, "has more than %d items", limit)
		return nil, false
	}
	return items, true
}

// list reads the items of a section with read, and keeps those it accepts. A
// list that is empty is a section without anything, which hides the section of
// the files after it; a list of which no item is good is a bad section.
func list[T any](p *parser, path string, raw json.RawMessage, limit int,
	read func(path string, raw json.RawMessage) (conditional[T], bool),
) ([]conditional[T], bool) {
	items, ok := p.items(path, raw, limit)
	if !ok {
		return nil, false
	}
	var parts []conditional[T]
	for i, item := range items {
		if part, ok := read(fmt.Sprintf("%s[%d]", path, i), item); ok {
			parts = append(parts, part)
		}
	}
	return parts, len(items) == 0 || len(parts) > 0
}

// stringField is the string of a field, empty when it is missing or bad.
func (p *parser) stringField(path string, object map[string]json.RawMessage, name string, limit int) string {
	raw, ok := object[name]
	if !ok {
		return ""
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		p.errorf(field(path, name), "is not a string")
		return ""
	}
	if len(value) > limit {
		p.errorf(field(path, name), "is longer than %d bytes", limit)
		return ""
	}
	return value
}

func (p *parser) boolField(path string, object map[string]json.RawMessage, name string) bool {
	var value bool
	if raw, ok := object[name]; ok {
		if err := json.Unmarshal(raw, &value); err != nil {
			p.errorf(field(path, name), "is not true or false")
		}
	}
	return value
}

// stringList reads one string or a list of strings.
func (p *parser) stringList(path string, raw json.RawMessage, limit, length int) ([]string, bool) {
	var one string
	if json.Unmarshal(raw, &one) == nil {
		raw, _ = json.Marshal([]string{one})
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		p.errorf(path, "is not a list of strings")
		return nil, false
	}
	if len(values) > limit || slices.ContainsFunc(values, func(s string) bool { return s == "" || len(s) > length }) {
		p.errorf(path, "holds at most %d strings of 1 to %d bytes", limit, length)
		return nil, false
	}
	return values, true
}

var languageTag = regexp.MustCompile(`^[a-z]{2,3}(-[A-Za-z0-9]{2,8})*$`)

// text reads a text: a string, which every language shows, or an object with a
// string by language, such as {"en": "Mirror", "zh": "镜像"}.
func (p *parser) text(path string, raw json.RawMessage) (Text, bool) {
	var plain string
	if json.Unmarshal(raw, &plain) == nil {
		if plain == "" || len(plain) > maxText {
			p.errorf(path, "is a text of 1 to %d bytes", maxText)
			return Text{}, false
		}
		return Text{EN: plain}, true
	}
	object := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &object); err != nil || object == nil {
		p.errorf(path, "is not a text")
		return Text{}, false
	}
	var text Text
	for _, language := range slices.Sorted(maps.Keys(object)) {
		if language == "when" {
			continue
		}
		if !languageTag.MatchString(language) {
			p.warnf(field(path, language), "is not a language")
			continue
		}
		value := p.stringField(path, object, language, maxText)
		switch {
		case language == "en":
			text.EN = value
		case language == "zh" || (language == "zh-CN" || language == "zh-Hans") && text.ZH == "":
			text.ZH = value
		}
	}
	if text.EN == "" {
		if text.ZH == "" {
			p.errorf(path, "has no text in English or Chinese")
			return Text{}, false
		}
		p.warnf(path, "has no English text")
		text.EN = text.ZH
	}
	return text, true
}

// textField reads the text of a field; a missing field gives the fallback.
func (p *parser) textField(path string, object map[string]json.RawMessage, name string, fallback Text) (Text, bool) {
	raw, ok := object[name]
	if !ok {
		if fallback == (Text{}) {
			p.errorf(field(path, name), "is missing")
			return Text{}, false
		}
		return fallback, true
	}
	return p.text(field(path, name), raw)
}

var knownOS = []string{"linux", "darwin", "windows", "freebsd"}

// condition reads the `when` of an object: the platforms the part is for.
func (p *parser) condition(path string, object map[string]json.RawMessage) (*condition, bool) {
	raw, ok := object["when"]
	if !ok {
		return nil, true
	}
	path = field(path, "when")
	fields, ok := p.object(path, raw, "os", "arch")
	if !ok {
		return nil, false
	}
	c := &condition{}
	if raw, ok := fields["os"]; ok {
		if c.os, ok = p.stringList(field(path, "os"), raw, 8, 32); !ok {
			return nil, false
		}
		for _, name := range c.os {
			if !slices.Contains(knownOS, name) {
				p.warnf(field(path, "os"), "%q is not an operating system as GOOS names them (%s)", name, strings.Join(knownOS, ", "))
			}
		}
	}
	if raw, ok := fields["arch"]; ok {
		if c.arch, ok = p.stringList(field(path, "arch"), raw, 16, 32); !ok {
			return nil, false
		}
	}
	return c, true
}

// search reads the search section: one variant, or a list of variants for
// different platforms, of which the first for the platform wins.
func (p *parser) search(raw json.RawMessage) ([]conditional[*Search], bool) {
	if trimmed := bytes.TrimSpace(raw); len(trimmed) > 0 && trimmed[0] == '[' {
		return list(p, "search", raw, 20, p.searchVariant)
	}
	part, ok := p.searchVariant("search", raw)
	if !ok {
		return nil, false
	}
	return []conditional[*Search]{part}, true
}

func (p *parser) searchVariant(path string, raw json.RawMessage) (conditional[*Search], bool) {
	var none conditional[*Search]
	object, ok := p.object(path, raw, "when", "label", "default", "choices")
	if !ok {
		return none, false
	}
	when, ok := p.condition(path, object)
	if !ok {
		return none, false
	}
	label, ok := p.textField(path, object, "label", Text{})
	if !ok {
		return none, false
	}
	items, ok := p.items(field(path, "choices"), object["choices"], 200)
	if !ok {
		return none, false
	}
	search := &Search{Label: label, Default: p.stringField(path, object, "default", maxName)}
	for i, item := range items {
		choice, ok := p.choice(fmt.Sprintf("%s.choices[%d]", path, i), item)
		if !ok {
			continue
		}
		if slices.ContainsFunc(search.Choices, func(c Choice) bool { return c.Value == choice.Value }) {
			p.errorf(fmt.Sprintf("%s.choices[%d].value", path, i), "%q is listed twice", choice.Value)
			continue
		}
		search.Choices = append(search.Choices, choice)
	}
	if !slices.ContainsFunc(search.Choices, func(c Choice) bool { return c.Value == search.Default }) {
		p.errorf(field(path, "default"), "names none of the choices: a search needs the one that vfox lists without an argument")
		return none, false
	}
	return conditional[*Search]{when: when, value: search}, true
}

func (p *parser) choice(path string, raw json.RawMessage) (Choice, bool) {
	object, ok := p.object(path, raw, "value", "name", "maker", "aliases", "popular", "everything")
	if !ok {
		return Choice{}, false
	}
	choice := Choice{
		Value:      p.stringField(path, object, "value", maxName),
		Popular:    p.boolField(path, object, "popular"),
		Everything: p.boolField(path, object, "everything"),
	}
	if !isArgument(choice.Value) {
		p.errorf(field(path, "value"), "is not an argument of vfox search: no leading - and no spaces")
		return Choice{}, false
	}
	if choice.Name, ok = p.textField(path, object, "name", Text{EN: choice.Value}); !ok {
		return Choice{}, false
	}
	if raw, ok := object["maker"]; ok {
		choice.Maker, _ = p.text(field(path, "maker"), raw)
	}
	if raw, ok := object["aliases"]; ok {
		choice.Aliases, _ = p.stringList(field(path, "aliases"), raw, 20, maxName)
	}
	return choice, true
}

// naming reads the naming section; the string "plain" says that the names have
// no structure.
func (p *parser) naming(raw json.RawMessage) (*Naming, bool) {
	var word string
	if json.Unmarshal(raw, &word) == nil {
		if word == "plain" {
			return nil, true
		}
		p.errorf("naming", `is "plain" or an object with rules, not %q`, word)
		return nil, false
	}
	object, ok := p.object("naming", raw, "sources", "defaultSource", "rules")
	if !ok {
		return nil, false
	}
	naming := &Naming{DefaultSource: p.stringField("naming", object, "defaultSource", maxName)}
	if raw, ok := object["sources"]; ok {
		sources, ok := p.items("naming.sources", raw, 200)
		if !ok {
			return nil, false
		}
		for i, item := range sources {
			source, ok := p.nameSource(fmt.Sprintf("naming.sources[%d]", i), item)
			if !ok {
				return nil, false
			}
			naming.Sources = append(naming.Sources, source)
		}
	}
	rules, ok := p.items("naming.rules", object["rules"], 20)
	if !ok {
		return nil, false
	}
	// The order of the rules matters: one bad rule would let another read the
	// names it was written for.
	for i, item := range rules {
		rule, ok := p.nameRule(fmt.Sprintf("naming.rules[%d]", i), item, naming.Sources)
		if !ok {
			return nil, false
		}
		naming.Rules = append(naming.Rules, rule)
	}
	if len(naming.Rules) == 0 {
		p.errorf("naming.rules", "is empty")
		return nil, false
	}
	return naming, true
}

func (p *parser) nameSource(path string, raw json.RawMessage) (NameSource, bool) {
	var value string
	if json.Unmarshal(raw, &value) == nil {
		raw, _ = json.Marshal(map[string]string{"value": value})
	}
	object, ok := p.object(path, raw, "value", "name")
	if !ok {
		return NameSource{}, false
	}
	source := NameSource{Value: p.stringField(path, object, "value", maxName)}
	if source.Value == "" {
		p.errorf(field(path, "value"), "is missing")
		return NameSource{}, false
	}
	source.Name, ok = p.textField(path, object, "name", Text{EN: source.Value})
	return source, ok
}

func (p *parser) nameRule(path string, raw json.RawMessage, sources []NameSource) (NameRule, bool) {
	object, ok := p.object(path, raw, "pattern", "flavors", "labels")
	if !ok {
		return NameRule{}, false
	}
	pattern := p.stringField(path, object, "pattern", maxPattern)
	if strings.Contains(pattern, sourcesPlaceholder) && len(sources) == 0 {
		p.errorf(field(path, "pattern"), "names {sources} without any source")
		return NameRule{}, false
	}
	expr, ok := p.compile(field(path, "pattern"), expandSources(pattern, sources))
	if !ok {
		return NameRule{}, false
	}
	rule := NameRule{Pattern: expr}
	if raw, ok := object["flavors"]; ok {
		if rule.Flavors, ok = p.stringList(field(path, "flavors"), raw, 10, maxName); !ok {
			return NameRule{}, false
		}
	}
	if raw, ok := object["labels"]; ok {
		var labels map[string]string
		if err := json.Unmarshal(raw, &labels); err != nil || len(labels) > 100 {
			p.errorf(field(path, "labels"), "is not an object of at most 100 labels")
			return NameRule{}, false
		}
		rule.Labels = make(map[string]string, len(labels))
		for text, label := range labels {
			rule.Labels[strings.ToLower(text)] = label
		}
	}
	return rule, true
}

// compile reads a regular expression of Go (RE2 syntax), which matches in
// linear time.
func (p *parser) compile(path, pattern string) (*regexp.Regexp, bool) {
	if pattern == "" {
		p.errorf(path, "is missing")
		return nil, false
	}
	expr, err := regexp.Compile(pattern)
	if err != nil {
		p.errorf(path, "is not a regular expression: %v", err)
		return nil, false
	}
	return expr, true
}

func (p *parser) shorthand(path string, raw json.RawMessage) (conditional[Shorthand], bool) {
	var none conditional[Shorthand]
	object, ok := p.object(path, raw, "when", "name", "meaning")
	if !ok {
		return none, false
	}
	when, ok := p.condition(path, object)
	if !ok {
		return none, false
	}
	shorthand := Shorthand{Name: p.stringField(path, object, "name", maxName)}
	if !isArgument(shorthand.Name) || strings.Contains(shorthand.Name, "@") {
		p.errorf(field(path, "name"), "is not a version name: no leading -, no spaces and no @")
		return none, false
	}
	if shorthand.Meaning, ok = p.textField(path, object, "meaning", Text{}); !ok {
		return none, false
	}
	return conditional[Shorthand]{when: when, value: shorthand}, true
}

func (p *parser) variable(path string, raw json.RawMessage) (conditional[Variable], bool) {
	var none conditional[Variable]
	object, ok := p.object(path, raw, "when", "key", "label", "help", "kind", "presets", "on", "off", "example", "advanced")
	if !ok {
		return none, false
	}
	when, ok := p.condition(path, object)
	if !ok {
		return none, false
	}
	variable := Variable{
		Key:      p.stringField(path, object, "key", maxKey),
		Example:  p.stringField(path, object, "example", maxValue),
		Advanced: p.boolField(path, object, "advanced"),
	}
	if problem := variableKeyProblem(variable.Key); problem != "" {
		p.errorf(field(path, "key"), "%q %s", variable.Key, problem)
		return none, false
	}
	if _, ok := object["label"]; !ok {
		p.warnf(field(path, "label"), "is missing")
	}
	if variable.Label, ok = p.textField(path, object, "label", Text{EN: humanizeEnvKey(variable.Key)}); !ok {
		return none, false
	}
	if raw, ok := object["help"]; ok {
		variable.Help, _ = p.text(field(path, "help"), raw)
	} else {
		p.warnf(field(path, "help"), "is missing")
	}
	switch kind := p.stringField(path, object, "kind", maxName); kind {
	case "", "url":
		variable.Kind = VariableURL
	case "choice":
		variable.Kind = VariableChoice
	case "toggle":
		variable.Kind = VariableToggle
		variable.On, variable.Off = p.stringField(path, object, "on", maxValue), p.stringField(path, object, "off", maxValue)
		if variable.On == "" {
			variable.On = "1"
		}
		if variable.On == variable.Off || !isPlainValue(variable.On) || !isPlainValue(variable.Off) {
			p.errorf(field(path, "on"), "and off are two different values of one line")
			return none, false
		}
	case "text":
		variable.Kind = VariableText
	default:
		p.errorf(field(path, "kind"), "%q is not url, choice, toggle or text", kind)
		return none, false
	}
	if raw, ok := object["presets"]; ok && variable.Kind != VariableToggle {
		items, ok := p.items(field(path, "presets"), raw, 100)
		if !ok {
			return none, false
		}
		for i, item := range items {
			if preset, ok := p.preset(fmt.Sprintf("%s.presets[%d]", path, i), item, variable.Kind); ok {
				variable.Presets = append(variable.Presets, preset)
			}
		}
	}
	if variable.Kind == VariableChoice && len(variable.Presets) == 0 {
		p.errorf(field(path, "presets"), "a choice needs the values it takes")
		return none, false
	}
	return conditional[Variable]{when: when, value: variable}, true
}

func (p *parser) preset(path string, raw json.RawMessage, kind VariableKind) (Preset, bool) {
	object, ok := p.object(path, raw, "value", "name")
	if !ok {
		return Preset{}, false
	}
	preset := Preset{Value: strings.TrimSpace(p.stringField(path, object, "value", maxValue))}
	switch {
	case preset.Value == "" || !isPlainValue(preset.Value):
		p.errorf(field(path, "value"), "is a value of one line")
		return Preset{}, false
	case kind == VariableURL && !isSecureAddress(preset.Value):
		p.errorf(field(path, "value"), "is not an https:// address")
		return Preset{}, false
	}
	preset.Name, ok = p.textField(path, object, "name", Text{EN: preset.Value})
	return preset, ok
}

func (p *parser) fileName(path string, raw json.RawMessage) (conditional[string], bool) {
	var name string
	if err := json.Unmarshal(raw, &name); err != nil || name == "" || len(name) > 256 || strings.ContainsFunc(name, unicode.IsControl) {
		p.errorf(path, "is not the name of a file")
		return conditional[string]{}, false
	}
	return conditional[string]{value: name}, true
}

func (p *parser) facet(path string, raw json.RawMessage) (conditional[Facet], bool) {
	var none conditional[Facet]
	object, ok := p.object(path, raw, "when", "id", "label", "kind", "values", "pattern")
	if !ok {
		return none, false
	}
	when, ok := p.condition(path, object)
	if !ok {
		return none, false
	}
	facet := Facet{ID: p.stringField(path, object, "id", 64)}
	if facet.ID == "" {
		p.errorf(field(path, "id"), "is missing")
		return none, false
	}
	if facet.Label, ok = p.textField(path, object, "label", Text{EN: facet.ID}); !ok {
		return none, false
	}
	switch kind := p.stringField(path, object, "kind", maxName); kind {
	case "", "note":
		facet.Kind = FacetNote
	case "suffix":
		facet.Kind = FacetSuffix
	case "pattern":
		facet.Kind = FacetPattern
		if facet.Pattern, ok = p.compile(field(path, "pattern"), p.stringField(path, object, "pattern", maxPattern)); !ok {
			return none, false
		}
	default:
		p.errorf(field(path, "kind"), "%q is not note, suffix or pattern", kind)
		return none, false
	}
	values, ok := object["values"]
	if ok {
		facet.Values, ok = p.stringList(field(path, "values"), values, 200, maxName)
	}
	if !ok || len(facet.Values) == 0 {
		p.errorf(field(path, "values"), "lists the values the facet knows")
		return none, false
	}
	return conditional[Facet]{when: when, value: facet}, true
}

func (p *parser) tip(path string, raw json.RawMessage) (conditional[Text], bool) {
	var when *condition
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) == nil && object != nil {
		var ok bool
		if when, ok = p.condition(path, object); !ok {
			return conditional[Text]{}, false
		}
	}
	text, ok := p.text(path, raw)
	return conditional[Text]{when: when, value: text}, ok
}

func (p *parser) link(path string, raw json.RawMessage) (conditional[Link], bool) {
	var none conditional[Link]
	object, ok := p.object(path, raw, "when", "title", "url")
	if !ok {
		return none, false
	}
	when, ok := p.condition(path, object)
	if !ok {
		return none, false
	}
	var link Link
	if link.Title, ok = p.textField(path, object, "title", Text{}); !ok {
		return none, false
	}
	if link.URL, ok = p.textField(path, object, "url", Text{}); !ok {
		return none, false
	}
	if !isSecureAddress(link.URL.EN) || link.URL.ZH != "" && !isSecureAddress(link.URL.ZH) {
		p.errorf(field(path, "url"), "is not an https:// address")
		return none, false
	}
	return conditional[Link]{when: when, value: link}, true
}
