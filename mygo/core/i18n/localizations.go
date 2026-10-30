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

	mygoi18n "github.com/egoist/mygo/i18n"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
)

// Translations read from the ARB catalogues in `shared/i18n`.
//
// The catalogues are bundled as they are; nothing is converted. English is
// the template and the fallback. Their messages are ICU MessageFormat, which
// the i18n package of mygo (it needs nothing but golang.org/x/text/language)
// parses and formats with the plural rules of the Unicode CLDR.

// MessageArguments are the arguments of a message, by placeholder name.
type MessageArguments = map[string]any

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
// GLib reads them from environment. Windows sets none of the variables: the user
// interface asks mygo for the display languages once it runs (ui.App.Run).
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
		switch mygoi18n.Base(locale) {
		case "zh":
			return SimplifiedChinese
		case "en":
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
	cache    map[string]*mygoi18n.Message
}

// NewLocalizations creates the messages of language.
func NewLocalizations(language SupportedLanguage, messages, fallback ArbCatalogue) *Localizations {
	return &Localizations{Language: language, messages: messages, fallback: fallback, cache: make(map[string]*mygoi18n.Message)}
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
	message := l.message(key)
	if message == nil {
		return key
	}
	return message.Format(l.LanguageTag(), args)
}

// message parses the message of key once: the one of the language, else the
// English one. It is nil when no catalogue has the key or the message does not
// parse.
func (l *Localizations) message(key string) *mygoi18n.Message {
	l.mu.Lock()
	defer l.mu.Unlock()
	if message, done := l.cache[key]; done {
		return message
	}
	source, _ := l.messages[key].(string)
	if source == "" {
		source, _ = l.fallback[key].(string)
	}
	var message *mygoi18n.Message
	if source != "" {
		if parsed, err := mygoi18n.ParseMessage(source); err == nil {
			message = parsed
		}
	}
	l.cache[key] = message
	return message
}
