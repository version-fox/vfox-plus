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

package i18n_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	mygoi18n "github.com/egoist/mygo/i18n"

	"github.com/version-fox/vfox-plus/mygo/core/i18n"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
)

func catalogues(t *testing.T) (i18n.ArbCatalogue, i18n.ArbCatalogue) {
	t.Helper()
	directory := filepath.Join(testsupport.SharedDir(), "i18n")
	english, err := i18n.LoadCatalogue(directory, i18n.English)
	if err != nil {
		t.Fatal(err)
	}
	chinese, err := i18n.LoadCatalogue(directory, i18n.SimplifiedChinese)
	if err != nil {
		t.Fatal(err)
	}
	return english, chinese
}

// placeholders are the names of the arguments of a message, nested ones too.
func placeholders(t *testing.T, source string) map[string]bool {
	t.Helper()
	message, err := mygoi18n.ParseMessage(source)
	if err != nil {
		t.Fatalf("%q: %v", source, err)
	}
	names := make(map[string]bool)
	for _, argument := range message.Args() {
		names[argument.Name] = true
	}
	return names
}

func TestCatalogues(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(testsupport.SharedDir(), "i18n"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	if len(names) != 2 || names[0] != "app_en.arb" || names[1] != "app_zh.arb" {
		t.Fatalf("got %q", names)
	}
	english, chinese := catalogues(t)
	if english["@@locale"] != "en" || chinese["@@locale"] != "zh" {
		t.Error("wrong template language")
	}
	englishKeys := sortedKeys(i18n.MessageKeys(english))
	chineseKeys := sortedKeys(i18n.MessageKeys(chinese))
	if !reflectEqual(englishKeys, chineseKeys) {
		t.Error("the catalogues must have the same messages")
	}
	for _, key := range englishKeys {
		meta, _ := english["@"+key].(map[string]interface{})
		if meta == nil || meta["description"] == nil {
			t.Errorf("%s is not described", key)
		}
		template, translated := placeholders(t, english[key].(string)), placeholders(t, chinese[key].(string))
		for name := range template {
			if !translated[name] {
				t.Errorf("%s lacks {%s} in app_zh.arb", key, name)
			}
		}
		declared := map[string]bool{}
		if placeholdersMeta, ok := meta["placeholders"].(map[string]interface{}); ok {
			for name := range placeholdersMeta {
				declared[name] = true
			}
		}
		if !reflectEqual(sortedMapKeys(template), sortedMapKeys(declared)) {
			t.Errorf("%s: placeholders %v, declared %v", key, template, declared)
		}
	}
}

func sortedKeys(keys []string) []string {
	sorted := append([]string(nil), keys...)
	sort.Strings(sorted)
	return sorted
}

func sortedMapKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func reflectEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestLocalizations(t *testing.T) {
	english, chinese := catalogues(t)
	en := i18n.NewLocalizations(i18n.English, english, english)
	zh := i18n.NewLocalizations(i18n.SimplifiedChinese, chinese, english)
	if got := en.T("sdksVersionCount", i18n.MessageArguments{"count": 2}); got != "2 versions installed" {
		t.Errorf("got %q", got)
	}
	if got := en.T("sdksVersionCount", i18n.MessageArguments{"count": 1}); got != "1 version installed" {
		t.Errorf("got %q", got)
	}
	if got := zh.T("sdksVersionCount", i18n.MessageArguments{"count": 2}); got != "已安装 2 个版本" {
		t.Errorf("got %q", got)
	}
	if got := zh.T("taskInstall", i18n.MessageArguments{"subject": "maven@3.9.16"}); got != "安装 maven@3.9.16" {
		t.Errorf("got %q", got)
	}
	fallback := i18n.NewLocalizations(i18n.SimplifiedChinese, i18n.ArbCatalogue{"navHome": "首页"}, english)
	if got := fallback.T("navSdks", nil); got != "SDKs" {
		t.Errorf("got %q", got)
	}
	if got := en.T("noSuchKey", nil); got != "noSuchKey" {
		t.Errorf("got %q", got)
	}
	if en.LanguageTag() != "en" || zh.LanguageTag() != "zh-Hans" {
		t.Error("wrong language tags")
	}
}

func TestResolveLanguage(t *testing.T) {
	if got := i18n.ResolveLanguage(settings.LanguageSystem, []string{"zh-TW"}); got != i18n.SimplifiedChinese {
		t.Errorf("got %q", got)
	}
	if got := i18n.ResolveLanguage(settings.LanguageSystem, []string{"zh_CN.UTF-8"}); got != i18n.SimplifiedChinese {
		t.Errorf("got %q", got)
	}
	if got := i18n.ResolveLanguage(settings.LanguageSystem, []string{"de-DE", "en-US"}); got != i18n.English {
		t.Errorf("got %q", got)
	}
	if got := i18n.ResolveLanguage(settings.LanguageSystem, []string{"fr"}); got != i18n.English {
		t.Errorf("got %q", got)
	}
	if got := i18n.ResolveLanguage(settings.LanguageEnglish, []string{"zh-CN"}); got != i18n.English {
		t.Errorf("got %q", got)
	}
	if got := i18n.ResolveLanguage(settings.LanguageSimplifiedChinese, []string{"en"}); got != i18n.SimplifiedChinese {
		t.Errorf("got %q", got)
	}
}

func TestSystemLocales(t *testing.T) {
	got := i18n.SystemLocales(map[string]string{"LANGUAGE": "zh_CN:en", "LC_ALL": "", "LC_MESSAGES": "C.UTF-8", "LANG": "de_DE.UTF-8"})
	want := []string{"zh_CN", "en", "de_DE.UTF-8"}
	if !reflectEqual(got, want) {
		t.Errorf("got %q", got)
	}
	if got := i18n.SystemLocales(map[string]string{"LANG": "C"}); len(got) != 0 {
		t.Errorf("got %q", got)
	}
	if got := i18n.SystemLocales(map[string]string{"LANG": "POSIX"}); len(got) != 0 {
		t.Errorf("got %q", got)
	}
}

func TestLocalizationsFormatPluralsOfTheirLanguage(t *testing.T) {
	catalogue := i18n.ArbCatalogue{
		"count":  "{count, plural, =0{none} one{# version} other{# versions}}",
		"broken": "{count, plural, one{x}}",
	}
	en := i18n.NewLocalizations(i18n.English, catalogue, catalogue)
	zh := i18n.NewLocalizations(i18n.SimplifiedChinese, catalogue, catalogue)
	for count, want := range map[int]string{0: "none", 1: "1 version", 2: "2 versions", 1234: "1,234 versions"} {
		if got := en.T("count", i18n.Arguments("count", count)); got != want {
			t.Errorf("English, %d: got %q, want %q", count, got, want)
		}
	}
	// Chinese has no `one` category: 1 is `other`.
	if got := zh.T("count", i18n.Arguments("count", 1)); got != "1 versions" {
		t.Errorf("Chinese, 1: got %q", got)
	}
	// A message that does not parse shows its key, which the tests of the
	// catalogues catch.
	if got := en.T("broken", i18n.Arguments("count", 1)); got != "broken" {
		t.Errorf("got %q", got)
	}
}
