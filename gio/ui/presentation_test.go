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

package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/gio/assets"
	"github.com/version-fox/vfox-plus/gio/core/i18n"
	"github.com/version-fox/vfox-plus/gio/core/vfox"
)

func testLocalizations(t *testing.T, language i18n.SupportedLanguage) *App {
	t.Helper()
	english, err := readCatalogue(i18n.English)
	if err != nil {
		t.Fatal(err)
	}
	messages := english
	if language != i18n.English {
		if messages, err = readCatalogue(language); err != nil {
			t.Fatal(err)
		}
	}
	a := &App{}
	a.l10n.Store(i18n.NewLocalizations(language, messages, english))
	return a
}

func TestTaskTitlesNameWhatVfoxDoes(t *testing.T) {
	a := testLocalizations(t, i18n.English)
	for _, tc := range []struct {
		command vfox.VfoxCommand
		want    string
	}{
		{vfox.VfoxCommand{Operation: vfox.OperationInstall, Sdk: "nodejs", Version: "24"}, "Install nodejs@24"},
		{vfox.VfoxCommand{Operation: vfox.OperationUninstall, Sdk: "nodejs", Version: "24"}, "Uninstall nodejs@24"},
		{vfox.VfoxCommand{Operation: vfox.OperationUse, Sdk: "java", Version: "21"}, "Use java@21"},
		{vfox.VfoxCommand{Operation: vfox.OperationUnuse, Sdk: "java"}, "Stop using java"},
		{vfox.VfoxCommand{Operation: vfox.OperationAddPlugin, Sdk: "go"}, "Add plugin go"},
		{vfox.VfoxCommand{Operation: vfox.OperationAddPlugin}, "Add plugin from source"},
		{vfox.VfoxCommand{Operation: vfox.OperationRemovePlugin, Sdk: "go"}, "Remove plugin go"},
		{vfox.VfoxCommand{Operation: vfox.OperationUpdatePlugin, Sdk: "go"}, "Update plugin go"},
		{vfox.VfoxCommand{Operation: vfox.OperationUpdateAllPlugins}, "Update all plugins"},
	} {
		if got := a.taskTitle(tc.command); got != tc.want {
			t.Errorf("%v: got %q, want %q", tc.command.Operation, got, tc.want)
		}
	}
	chinese := testLocalizations(t, i18n.SimplifiedChinese)
	if got := chinese.taskTitle(vfox.VfoxCommand{Operation: vfox.OperationInstall, Sdk: "nodejs", Version: "24"}); !strings.Contains(got, "nodejs@24") || got == "Install nodejs@24" {
		t.Errorf("got %q", got)
	}
}

func TestErrorsAreDescribedByTheirMessage(t *testing.T) {
	if got := describeError(&vfox.VfoxError{Message: "no such plugin"}); got != "no such plugin" {
		t.Errorf("got %q", got)
	}
	if got := describeError(fmt.Errorf("wrapped: %w", &vfox.VfoxError{Message: "inner"})); got != "inner" {
		t.Errorf("got %q", got)
	}
	if got := describeError(errors.New("plain")); got != "plain" {
		t.Errorf("got %q", got)
	}
}

func TestSourcesOfPluginsShowTheRepository(t *testing.T) {
	for homepage, want := range map[string]string{
		"https://github.com/version-fox/vfox-nodejs":  "version-fox/vfox-nodejs",
		"https://github.com/version-fox/vfox-nodejs/": "version-fox/vfox-nodejs",
		"https://example.com":                         "https://example.com",
		"":                                            "",
		"://bad":                                      "://bad",
	} {
		if got := sourceOf(homepage); got != want {
			t.Errorf("%q: got %q, want %q", homepage, got, want)
		}
	}
}

func TestSmallHelpers(t *testing.T) {
	if itoa(42) != "42" || boolText(true) != "true" || boolText(false) != "false" {
		t.Error("formatting")
	}
	if orQuestionMark(func() (string, bool) { return "1.0", true }) != "1.0" || orQuestionMark(func() (string, bool) { return "", false }) != "?" {
		t.Error("versions")
	}
	if !isDirectory(t.TempDir()) || isDirectory("/does/not/exist") {
		t.Error("directories")
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, nil, 0o600); err != nil || isDirectory(file) {
		t.Error("a file is not a directory")
	}
	for name, valid := range map[string]bool{"nodejs": true, "my-sdk_1.0+x": true, "-bad": false, "has space": false, "": false} {
		if pluginNamePattern.MatchString(name) != valid {
			t.Errorf("%q should be valid: %v", name, valid)
		}
	}
}

func TestTheVersionIsThatOfTheVersionFile(t *testing.T) {
	content, err := os.ReadFile("../VERSION")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(content)); got != Version {
		t.Errorf("VERSION says %q, the application %q", got, Version)
	}
}

// messageKeys are the keys that the code asks for.
func messageKeys(t *testing.T) map[string]bool {
	t.Helper()
	pattern := regexp.MustCompile(`\bT\("([A-Za-z0-9]+)"`)
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	keys := map[string]bool{}
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		content, err := os.ReadFile(file) //nolint:gosec // A source file of this package.
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range pattern.FindAllSubmatch(content, -1) {
			keys[string(match[1])] = true
		}
	}
	return keys
}

func TestEveryMessageTheCodeAsksForIsTranslated(t *testing.T) {
	keys := messageKeys(t)
	if len(keys) < 100 {
		t.Fatalf("only %d keys were found", len(keys))
	}
	for _, language := range []i18n.SupportedLanguage{i18n.English, i18n.SimplifiedChinese} {
		catalogue, err := readCatalogue(language)
		if err != nil {
			t.Fatal(err)
		}
		for key := range keys {
			if _, ok := catalogue[key].(string); !ok {
				t.Errorf("%s has no message %q", language, key)
			}
		}
	}
}

func TestEveryMessageOfTheCatalogueIsUsedOrKnownToBeUnused(t *testing.T) {
	// The catalogues are shared with the other implementations; no
	// implementation shows these messages yet.
	known := map[string]bool{
		"actionShowDetails": true, "errorTitle": true, "settingsSystemColorUnavailable": true, "sdkInfoError": true,
	}
	english, err := readCatalogue(i18n.English)
	if err != nil {
		t.Fatal(err)
	}
	keys := messageKeys(t)
	var unused []string
	for _, key := range i18n.MessageKeys(english) {
		if !keys[key] && !known[key] {
			unused = append(unused, key)
		}
	}
	sort.Strings(unused)
	if len(unused) > 0 {
		t.Errorf("messages that nothing shows: %v", unused)
	}
}

func TestTheAssetsAreEmbedded(t *testing.T) {
	for _, name := range []string{"fonts/MaterialSymbolsRounded.ttf", "icons.json", "i18n/app_en.arb", "i18n/app_zh.arb", "licenses.txt"} {
		content, err := assets.Files.ReadFile(name)
		if err != nil || len(content) == 0 {
			t.Errorf("%s: %v", name, err)
		}
	}
}
