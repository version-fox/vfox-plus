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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/version-fox/vfox-plus/gio/core/settings"
)

// Translations read from the ARB catalogues in `shared/i18n`.
//
// The catalogues are bundled as they are; nothing is converted. English is
// the template and the fallback.

// ArbCatalogue is an ARB catalogue: messages and their `@`-prefixed metadata.
type ArbCatalogue map[string]interface{}

// SupportedLanguage is a language code with a catalogue.
type SupportedLanguage string

// The languages with a catalogue.
const (
	English            SupportedLanguage = "en"
	SimplifiedChinese  SupportedLanguage = "zh"
	SupportedLanguages                   = "en,zh"
)

// SystemLocales returns the locales of the desktop, most preferred first, as
// GLib reads them from environment.
func SystemLocales(environment map[string]string) []string {
	var locales []string
	add := func(value string) {
		for _, locale := range strings.Split(value, ":") {
			if locale == "" || locale == "C" || locale == "POSIX" || strings.HasPrefix(locale, "C.") {
				continue
			}
			locales = append(locales, locale)
		}
	}
	add(environment["LANGUAGE"])
	for _, key := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		add(environment[key])
	}
	return locales
}

// ResolveLanguage picks the catalogue for preference. With System, the
// desktop locales are matched against the catalogues; every Chinese locale
// gets Simplified Chinese and English is the fallback.
func ResolveLanguage(preference settings.LanguagePreference, locales []string) SupportedLanguage {
	switch preference.LanguageCode() {
	case "zh":
		return SimplifiedChinese
	case "en":
		return English
	}
	for _, locale := range locales {
		language := strings.ToLower(locale)
		if index := strings.IndexAny(language, "-_.@"); index >= 0 {
			language = language[:index]
		}
		if language == "zh" {
			return SimplifiedChinese
		}
		if language == "en" {
			return English
		}
	}
	return English
}

// LoadCatalogue reads the catalogue `app_<language>.arb` from directory.
func LoadCatalogue(directory string, language SupportedLanguage) (ArbCatalogue, error) {
	content, err := os.ReadFile(filepath.Join(directory, "app_"+string(language)+".arb")) //nolint:gosec // The catalogue of a language, in the directory given.
	if err != nil {
		return nil, err
	}
	return ParseCatalogue(content)
}

// ParseCatalogue parses the content of an ARB file.
func ParseCatalogue(content []byte) (ArbCatalogue, error) {
	var catalogue ArbCatalogue
	if err := json.Unmarshal(content, &catalogue); err != nil {
		return nil, err
	}
	return catalogue, nil
}

// Arguments builds the arguments of a message from alternating names and
// values: Arguments("count", 2, "name", "node").
func Arguments(pairs ...any) MessageArguments {
	args := make(MessageArguments, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		if name, ok := pairs[i].(string); ok {
			args[name] = pairs[i+1]
		}
	}
	return args
}

// MessageKeys returns the message keys of a catalogue, without metadata.
func MessageKeys(catalogue ArbCatalogue) []string {
	var keys []string
	for key := range catalogue {
		if !strings.HasPrefix(key, "@") {
			keys = append(keys, key)
		}
	}
	return keys
}

// Localizations are the messages of one language.
//
// It is safe for use from several goroutines.
type Localizations struct {
	// Language is the language code of the messages.
	Language SupportedLanguage
	messages ArbCatalogue
	fallback ArbCatalogue
	mu       sync.Mutex
	cache    map[string][]MessagePart
}

// NewLocalizations creates the messages of language.
func NewLocalizations(language SupportedLanguage, messages, fallback ArbCatalogue) *Localizations {
	return &Localizations{Language: language, messages: messages, fallback: fallback, cache: make(map[string][]MessagePart)}
}

// LanguageTag is the BCP 47 tag of the language.
func (l *Localizations) LanguageTag() string {
	if l.Language == SimplifiedChinese {
		return "zh-Hans"
	}
	return "en"
}

// T formats the message key with args.
//
// A key that the language lacks falls back to English; a key that no
// catalogue has is returned as it is, which the tests catch.
func (l *Localizations) T(key string, args MessageArguments) string {
	l.mu.Lock()
	parts, ok := l.cache[key]
	if !ok {
		source, _ := l.messages[key].(string)
		if source == "" {
			source, _ = l.fallback[key].(string)
		}
		if source == "" {
			parts = []MessagePart{Text{Content: key}}
		} else if parsed, err := ParseMessage(source); err == nil {
			parts = parsed
		} else {
			parts = []MessagePart{Text{Content: key}}
		}
		l.cache[key] = parts
	}
	l.mu.Unlock()
	return FormatMessage(parts, args, string(l.Language))
}
