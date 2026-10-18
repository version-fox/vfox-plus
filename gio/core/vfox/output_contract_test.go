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

package vfox_test

// Runs the parsers against the shared vfox output fixtures.
//
// The expected results live in shared/fixtures/vfox-*/cases.json and are
// shared by every implementation of the GUI; see shared/fixtures/README.md.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	core "github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/core/vfox"
	"github.com/version-fox/vfox-plus/gio/internal/testsupport"
)

type fixtureCase struct {
	Name    string
	Command []string
	Expect  map[string]interface{}
}

func loadCases(t *testing.T) (string, []fixtureCase) {
	t.Helper()
	pattern := filepath.Join(testsupport.SharedDir(), "fixtures", "vfox-*", "cases.json")
	files, err := filepath.Glob(pattern)
	if err != nil || len(files) == 0 {
		t.Fatalf("no fixture manifests match %s", pattern)
	}
	sort.Strings(files)
	var cases []fixtureCase
	version := ""
	for _, file := range files {
		content, err := os.ReadFile(file) //nolint:gosec // The path is a fixture.
		if err != nil {
			t.Fatal(err)
		}
		var manifest struct {
			Vfox  string `json:"vfox"`
			Cases []struct {
				Name    string                 `json:"name"`
				Command []string               `json:"command"`
				Expect  map[string]interface{} `json:"expect"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(content, &manifest); err != nil {
			t.Fatal(err)
		}
		if version == "" {
			version = manifest.Vfox
		}
		directory := filepath.Dir(file)
		for _, c := range manifest.Cases {
			for check, expected := range c.Expect {
				cases = append(cases, fixtureCase{
					Name:    directory + "|" + c.Name + "|" + check,
					Command: c.Command,
					Expect: map[string]interface{}{
						"directory": directory,
						"case":      c.Name,
						"check":     check,
						"expected":  expected,
					},
				})
			}
		}
	}
	if len(cases) == 0 {
		t.Fatal("no fixture cases found")
	}
	return version, cases
}

func indexed(items []interface{}) map[string]interface{} {
	entries := make(map[string]interface{}, len(items))
	for i, item := range items {
		entries[fmt.Sprint(i)] = item
	}
	return map[string]interface{}{"count": len(items), "items": entries}
}

func toJSON(t *testing.T, value interface{}) interface{} {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var decoded interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	return decoded
}

func TestParsersMatchTheFixtures(t *testing.T) {
	_, cases := loadCases(t)
	for _, c := range cases {
		directory := c.Expect["directory"].(string)
		name := c.Expect["case"].(string)
		check := c.Expect["check"].(string)
		expected := c.Expect["expected"]
		stdout := testsupport.ReadCapture(filepath.Join(directory, name+".stdout"))
		stderr := testsupport.ReadCapture(filepath.Join(directory, name+".stderr"))
		var actual interface{}
		switch check {
		case "version":
			actual = vfox.ParseVfoxVersion(stdout)
		case "current":
			actual = currentToJSON(vfox.ParseCurrent(stdout))
		case "list":
			actual = treeToJSON(vfox.ParseInstalledTree(stdout))
		case "search":
			actual = indexed(searchToJSON(vfox.ParseSearch(stdout)))
		case "available":
			actual = indexed(availableToJSON(vfox.ParseAvailablePlugins(stdout)))
		case "pluginInfo":
			info, ok := vfox.ParsePluginInfo(stdout)
			if !ok {
				actual = nil
			} else {
				actual = pluginInfoToJSON(info)
			}
		case "installPath":
			path, ok := vfox.ParseInstallPath(stdout)
			if !ok {
				actual = nil
			} else {
				actual = path
			}
		case "config":
			actual = configToJSON(vfox.ParseConfigList(stdout))
		case "finalProgress":
			var last interface{}
			for _, line := range core.SplitTerminalLines(core.StripANSI(stderr)) {
				if progress := vfox.ParseDownloadProgress(line); progress >= 0 {
					last = progress
				}
			}
			actual = last
		case "error":
			actual = vfox.SummarizeError(stdout, stderr)
		default:
			t.Fatalf("unknown parser %q in cases.json", check)
		}
		actualJSON := toJSON(t, selectItems(check, actual, expected))
		if !reflect.DeepEqual(actualJSON, expected) {
			t.Errorf("%s %s: got %#v, want %#v", name, check, actualJSON, expected)
		}
	}
}

func selectItems(check string, actual interface{}, expected interface{}) interface{} {
	if check != "search" && check != "available" {
		return actual
	}
	actualMap, _ := actual.(map[string]interface{})
	expectedMap, _ := expected.(map[string]interface{})
	expectedItems, _ := expectedMap["items"].(map[string]interface{})
	items := map[string]interface{}{}
	if actualMap != nil {
		if actualItems, ok := actualMap["items"].(map[string]interface{}); ok {
			for index := range expectedItems {
				items[index] = actualItems[index]
			}
		}
		return map[string]interface{}{"count": actualMap["count"], "items": items}
	}
	return actual
}

func currentToJSON(pairs []vfox.CurrentVersion) interface{} {
	result := make(map[string]interface{}, len(pairs))
	for _, pair := range pairs {
		if pair.HasVersion {
			result[pair.Name] = pair.Version
		} else {
			result[pair.Name] = nil
		}
	}
	return result
}

func treeToJSON(tree []vfox.SdkVersions) interface{} {
	result := make(map[string]interface{}, len(tree))
	for _, item := range tree {
		versions := make([]interface{}, 0, len(item.Versions))
		for _, v := range item.Versions {
			versions = append(versions, v)
		}
		result[item.Name] = versions
	}
	return result
}

func pluginInfoToJSON(info core.PluginInfo) interface{} {
	return map[string]interface{}{
		"name": info.Name, "version": info.Version, "homepage": info.Homepage,
		"installPath": info.InstallPath, "description": info.Description,
	}
}

func searchToJSON(versions []core.AvailableVersion) []interface{} {
	result := make([]interface{}, 0, len(versions))
	for _, version := range versions {
		additions := make([]interface{}, 0, len(version.Additions))
		for _, addition := range version.Additions {
			additions = append(additions, map[string]interface{}{"name": addition.Name, "version": addition.Version})
		}
		var note interface{}
		if version.HasNote {
			note = version.Note
		}
		result = append(result, map[string]interface{}{
			"version": version.Version, "note": note, "additions": additions, "installed": version.Installed,
		})
	}
	return result
}

func availableToJSON(plugins []core.AvailablePlugin) []interface{} {
	result := make([]interface{}, 0, len(plugins))
	for _, plugin := range plugins {
		result = append(result, map[string]interface{}{
			"name": plugin.Name, "official": plugin.Official, "homepage": plugin.Homepage,
		})
	}
	return result
}

func configToJSON(entries []core.ConfigEntry) []interface{} {
	result := make([]interface{}, 0, len(entries))
	for _, entry := range entries {
		result = append(result, map[string]interface{}{"key": entry.Key, "value": entry.Value})
	}
	return result
}
