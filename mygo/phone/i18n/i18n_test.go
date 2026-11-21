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
	"maps"
	"reflect"
	"slices"
	"testing"
)

// sortedKeys returns the keys of a catalog in sorted order.
func sortedKeys(catalog map[string]string) []string {
	return slices.Sorted(maps.Keys(catalog))
}

func TestTheTwoLanguagesHaveTheSameKeys(t *testing.T) {
	if en, zh := Keys(English), Keys(Chinese); !reflect.DeepEqual(en, zh) {
		t.Fatalf("the keys differ: English has %d, Chinese has %d", len(en), len(zh))
	}
}

func TestTheWebCatalogsKeepTheWebKeys(t *testing.T) {
	// The web's catalogs have 160 keys each; the phone's own messages are in
	// phone.go and are not among them.
	if len(english) != 160 || len(chinese) != 160 {
		t.Fatalf("the web catalogs have %d and %d keys, want 160 each", len(english), len(chinese))
	}
	if !reflect.DeepEqual(sortedKeys(english), sortedKeys(chinese)) {
		t.Fatal("the web catalogs have different keys")
	}
	for key := range phoneEnglish {
		if _, ok := english[key]; ok {
			t.Errorf("the phone message %q is also a web key", key)
		}
	}
}

func TestThePhoneCatalogsHaveTheSameKeys(t *testing.T) {
	if !reflect.DeepEqual(sortedKeys(phoneEnglish), sortedKeys(phoneChinese)) {
		t.Fatal("the phone catalogs have different keys")
	}
}

func TestNoChineseMessageIsBlank(t *testing.T) {
	for key, text := range chinese {
		if text == "" {
			t.Errorf("the Chinese message %q is blank", key)
		}
	}
}

func TestOnlyTheWebsPrefixIsBlankInEnglish(t *testing.T) {
	// droppedHead is a blank prefix in English, as it is on the web.
	for key, text := range english {
		if text == "" && key != "droppedHead" {
			t.Errorf("the English message %q is blank", key)
		}
	}
}

func TestResolveFollowsThePreference(t *testing.T) {
	tests := []struct {
		name      string
		pref      Preference
		languages []string
		want      Language
	}{
		{"system Chinese", PreferenceSystem, []string{"zh-Hans-CN", "en-GB"}, Chinese},
		{"system Chinese in capitals", PreferenceSystem, []string{"ZH-cn"}, Chinese},
		{"system English", PreferenceSystem, []string{"en-US"}, English},
		{"system without languages", PreferenceSystem, nil, English},
		{"system prefers English first", PreferenceSystem, []string{"en-US", "zh-CN"}, English},
		{"the app chose Chinese", PreferenceChinese, []string{"en-US"}, Chinese},
		{"the app chose English", PreferenceEnglish, []string{"zh-CN"}, English},
		{"an unknown preference is the system's", Preference("fr"), []string{"zh"}, Chinese},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Resolve(tt.pref, tt.languages); got != tt.want {
				t.Errorf("Resolve(%q, %q) = %q, want %q", tt.pref, tt.languages, got, tt.want)
			}
		})
	}
}

func TestTextAnswersInTheCurrentLanguage(t *testing.T) {
	t.Cleanup(func() { SetLanguage(English) })
	SetLanguage(English)
	if got := Text("home"); got != "Home" {
		t.Errorf("Text(home) in English = %q, want Home", got)
	}
	SetLanguage(Chinese)
	if got, want := Text("home"), chinese["home"]; got != want {
		t.Errorf("Text(home) in Chinese = %q, want %q", got, want)
	}
	if Current() != Chinese {
		t.Errorf("Current() = %q, want %q", Current(), Chinese)
	}
}

func TestSetLanguageRefusesALanguageTheAppDoesNotHave(t *testing.T) {
	t.Cleanup(func() { SetLanguage(English) })
	SetLanguage(Chinese)
	SetLanguage(Language("fr"))
	if Current() != English {
		t.Errorf("Current() after an unknown language = %q, want %q", Current(), English)
	}
}

func TestMissingChineseMessageFallsBackToEnglish(t *testing.T) {
	saved := catalogs[Chinese]
	t.Cleanup(func() { catalogs[Chinese] = saved })
	catalogs[Chinese] = map[string]string{}
	if got := TextIn(Chinese, "home"); got != "Home" {
		t.Errorf("TextIn(Chinese, home) without a Chinese message = %q, want the English Home", got)
	}
}

func TestUnknownKeyIsReturnedAsItIs(t *testing.T) {
	if got := TextIn(English, "noSuchKey"); got != "noSuchKey" {
		t.Errorf("TextIn for an unknown key = %q, want the key", got)
	}
}

func TestPhoneMessagesAreInBothLanguages(t *testing.T) {
	for _, key := range []string{"appName", "back", "connect", "pairLead", "pairLinkHint", "pairLinkLabel", "phonePending"} {
		if TextIn(English, key) == key || TextIn(Chinese, key) == key {
			t.Errorf("the phone message %q is missing in a language", key)
		}
	}
}
