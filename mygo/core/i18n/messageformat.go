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

package i18n

import (
	"fmt"
	"strconv"
	"strings"
)

// The subset of ICU MessageFormat that the catalogues in `shared/i18n` use:
// placeholders (`{name}`) and plurals with exact matches and plural
// categories (`{count, plural, =0{…} =1{…} other{…}}`), which may contain
// placeholders themselves. Apostrophes are literal, as in Flutter's
// `gen-l10n` without `use-escaping`. See `docs/i18n.md`.

// MessageArguments are the arguments of a message, by placeholder name.
type MessageArguments map[string]interface{}

// MessagePart is a part of a parsed message.
type MessagePart interface {
	messagePart()
}

// Text is literal text.
type Text struct{ Content string }

func (Text) messagePart() {}

// Argument is a placeholder: `{name}`.
type Argument struct{ Name string }

func (Argument) messagePart() {}

// Plural is a plural: `{name, plural, =0{…} other{…}}`.
type Plural struct {
	Name  string
	Cases map[string][]MessagePart
}

func (Plural) messagePart() {}

// MessageFormatError reports that a message could not be parsed.
type MessageFormatError struct {
	Message string
	Source  string
}

func (e *MessageFormatError) Error() string {
	return fmt.Sprintf("%s in %q", e.Message, e.Source)
}

// ParseMessage parses source into parts.
func ParseMessage(source string) ([]MessagePart, error) {
	parser := parser{source: source}
	parts, err := parser.parseParts()
	if err != nil {
		return nil, err
	}
	if !parser.atEnd() {
		return nil, &MessageFormatError{Message: `Unbalanced "}"`, Source: source}
	}
	return parts, nil
}

// PluralCategory is the CLDR plural category of value in language: English
// has `one` and `other`, Chinese only `other`.
func PluralCategory(value float64, language string) string {
	if language == "en" && value == 1 {
		return "one"
	}
	return "other"
}

// FormatMessage formats parsed parts with args, choosing plural cases with
// the plural rules of language.
func FormatMessage(parts []MessagePart, args MessageArguments, language string) string {
	var out strings.Builder
	for _, part := range parts {
		switch part := part.(type) {
		case Text:
			out.WriteString(part.Content)
		case Argument:
			if value, ok := args[part.Name]; ok {
				out.WriteString(formatValue(value))
			} else {
				out.WriteString("{" + part.Name + "}")
			}
		case Plural:
			number := numberOf(args[part.Name])
			var chosen []MessagePart
			matched := false
			if isInteger(number) && number >= -1e15 && number <= 1e15 {
				chosen, matched = part.Cases["="+strconv.Itoa(int(number))]
			}
			if !matched {
				chosen, matched = part.Cases[PluralCategory(number, language)]
			}
			if !matched {
				chosen, _ = part.Cases["other"]
			}
			out.WriteString(FormatMessage(chosen, args, language))
		}
	}
	return out.String()
}

func isInteger(value float64) bool { return value == float64(int(value)) }

func formatValue(value interface{}) string {
	switch value := value.(type) {
	case float64:
		if isInteger(value) {
			return strconv.Itoa(int(value))
		}
		return strconv.FormatFloat(value, 'f', -1, 64)
	case float32:
		return formatValue(float64(value))
	case int:
		return strconv.Itoa(value)
	case int64:
		return strconv.FormatInt(value, 10)
	case string:
		return value
	default:
		return fmt.Sprintf("%v", value)
	}
}

func numberOf(value interface{}) float64 {
	switch value := value.(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int64:
		return float64(value)
	case string:
		if number, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil {
			return number
		}
	}
	return 0
}

type parser struct {
	source   string
	position int
}

func (p *parser) atEnd() bool { return p.position >= len(p.source) }

// parseParts parses parts until the end or an unmatched `}`, which is not
// consumed.
func (p *parser) parseParts() ([]MessagePart, error) {
	var parts []MessagePart
	var text strings.Builder
	for !p.atEnd() {
		char := p.source[p.position]
		if char == '}' {
			break
		}
		if char == '{' {
			if text.Len() > 0 {
				parts = append(parts, Text{text.String()})
				text.Reset()
			}
			part, err := p.parsePlaceholder()
			if err != nil {
				return nil, err
			}
			parts = append(parts, part)
		} else {
			text.WriteByte(char)
			p.position++
		}
	}
	if text.Len() > 0 {
		parts = append(parts, Text{text.String()})
	}
	return parts, nil
}

func (p *parser) parsePlaceholder() (MessagePart, error) {
	if err := p.expect('{'); err != nil {
		return nil, err
	}
	name, err := p.readIdentifier("Expected a name")
	if err != nil {
		return nil, err
	}
	p.skipSpaces()
	if p.peek() == '}' {
		p.position++
		return Argument{name}, nil
	}
	if err := p.expect(','); err != nil {
		return nil, err
	}
	kind, err := p.readIdentifier("Expected a name")
	if err != nil {
		return nil, err
	}
	if kind != "plural" {
		return nil, &MessageFormatError{Message: fmt.Sprintf("Unsupported argument type %q", kind), Source: p.source}
	}
	p.skipSpaces()
	if err := p.expect(','); err != nil {
		return nil, err
	}
	cases := make(map[string][]MessagePart)
	for {
		p.skipSpaces()
		if p.peek() == '}' {
			p.position++
			break
		}
		selector, err := p.readSelector()
		if err != nil {
			return nil, err
		}
		p.skipSpaces()
		if err := p.expect('{'); err != nil {
			return nil, err
		}
		body, err := p.parseParts()
		if err != nil {
			return nil, err
		}
		if err := p.expect('}'); err != nil {
			return nil, err
		}
		cases[selector] = body
	}
	if _, ok := cases["other"]; !ok {
		return nil, &MessageFormatError{Message: `A plural needs an "other" case`, Source: p.source}
	}
	return Plural{Name: name, Cases: cases}, nil
}

func (p *parser) readIdentifier(what string) (string, error) {
	p.skipSpaces()
	start := p.position
	for p.position < len(p.source) {
		char := p.source[p.position]
		if char == '_' || char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || p.position > start && char >= '0' && char <= '9' {
			p.position++
		} else {
			break
		}
	}
	if p.position == start {
		return "", &MessageFormatError{Message: what, Source: p.source}
	}
	return p.source[start:p.position], nil
}

func (p *parser) readSelector() (string, error) {
	p.skipSpaces()
	start := p.position
	if p.position < len(p.source) && p.source[p.position] == '=' {
		p.position++
		digits := p.position
		for p.position < len(p.source) && p.source[p.position] >= '0' && p.source[p.position] <= '9' {
			p.position++
		}
		if p.position == digits {
			return "", &MessageFormatError{Message: "Expected a plural case", Source: p.source}
		}
		return p.source[start:p.position], nil
	}
	letters := p.position
	for p.position < len(p.source) && p.source[p.position] >= 'a' && p.source[p.position] <= 'z' {
		p.position++
	}
	if p.position == letters {
		return "", &MessageFormatError{Message: "Expected a plural case", Source: p.source}
	}
	return p.source[start:p.position], nil
}

func (p *parser) skipSpaces() {
	for !p.atEnd() {
		char := p.source[p.position]
		if char == ' ' || char == '\t' || char == '\n' || char == '\r' {
			p.position++
		} else {
			return
		}
	}
}

func (p *parser) peek() byte {
	if p.atEnd() {
		return 0
	}
	return p.source[p.position]
}

func (p *parser) expect(char byte) error {
	if p.peek() != char {
		return &MessageFormatError{Message: fmt.Sprintf("Expected %q at %d", string(char), p.position), Source: p.source}
	}
	p.position++
	return nil
}
