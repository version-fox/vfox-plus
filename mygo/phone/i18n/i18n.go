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

// Package i18n holds the messages of the phone app, in English and in
// Simplified Chinese. The catalogs of the web interface (en.go and zh.go) keep
// its keys, so that the two interfaces say the same things under the same
// names; phone.go adds the messages that only the phone shows. Every message of
// the app is read through Text, and nowhere else.
package i18n

import (
	"maps"
	"slices"
	"sort"
	"strings"
	"sync/atomic"
)

// Language is a language the interface is written in.
type Language string

// The languages of the interface.
const (
	English Language = "en"
	Chinese Language = "zh"
)

// Preference is the language the user chose: the system's, or one of the
// languages. Its values are those of the web interface's page language.
type Preference string

// The preferences of the language.
const (
	PreferenceSystem  Preference = "system"
	PreferenceEnglish Preference = "en"
	PreferenceChinese Preference = "zh"
)

// catalogs holds the messages of each language: the web's and the phone's.
var catalogs = map[Language]map[string]string{
	English: merge(english, phoneEnglish),
	Chinese: merge(chinese, phoneChinese),
}

// current is the language that Text uses, a Language; English until SetLanguage.
var current atomic.Value

// merge returns the messages of two catalogs together; the second one's keys
// are not in the first.
func merge(base, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for key, text := range base {
		out[key] = text
	}
	for key, text := range extra {
		out[key] = text
	}
	return out
}

// Resolve returns the language for a preference. The system's preference
// follows the first of the languages the system reads, as BCP 47 tags, and is
// Chinese when that tag starts with zh, English otherwise.
func Resolve(preference Preference, systemLanguages []string) Language {
	switch preference {
	case PreferenceEnglish:
		return English
	case PreferenceChinese:
		return Chinese
	}
	if len(systemLanguages) > 0 && strings.HasPrefix(strings.ToLower(systemLanguages[0]), "zh") {
		return Chinese
	}
	return English
}

// SetLanguage makes Text answer in lang. A language that the app does not have
// makes it answer in English.
func SetLanguage(lang Language) {
	if _, ok := catalogs[lang]; !ok {
		lang = English
	}
	current.Store(lang)
}

// Current returns the language that Text answers in.
func Current() Language {
	if lang, ok := current.Load().(Language); ok {
		return lang
	}
	return English
}

// Text returns the message for key in the current language.
func Text(key string) string {
	return TextIn(Current(), key)
}

// TextIn returns the message for key in lang. Where lang has no message for the
// key, it returns the English one, and for a key that no catalog has, the key
// itself, so that a missing message shows as its name.
func TextIn(lang Language, key string) string {
	if text, ok := catalogs[lang][key]; ok {
		return text
	}
	if text, ok := catalogs[English][key]; ok {
		return text
	}
	return key
}

// registered holds the keys that Register added, for the tests to read.
var registered = map[string]bool{}

// Register adds messages of the screens: for each key, its English and its
// Simplified Chinese text. The screens' files i18n/phone_<area>.go call it from
// their init functions, which run before the app or any test starts, so Register
// is not safe for use after that. It panics when a key has a message already, in
// the web's catalogs or the phone's, and when a text is blank: a message that is
// missing is a mistake of the code, and it is found at start.
func Register(messages map[string][2]string) {
	for _, key := range slices.Sorted(maps.Keys(messages)) {
		texts := messages[key]
		if _, ok := catalogs[English][key]; ok {
			panic("i18n: the message " + key + " is registered already")
		}
		if texts[0] == "" || texts[1] == "" {
			panic("i18n: the message " + key + " has a blank text")
		}
		catalogs[English][key] = texts[0]
		catalogs[Chinese][key] = texts[1]
		registered[key] = true
	}
}

// Keys returns the keys of the messages of lang, in sorted order.
func Keys(lang Language) []string {
	keys := make([]string, 0, len(catalogs[lang]))
	for key := range catalogs[lang] {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
