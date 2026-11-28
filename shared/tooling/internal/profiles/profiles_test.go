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

package profiles

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// repository makes a repository with a mygo module and the profiles, and
// returns the module.
func repository(t *testing.T, profiles map[string]string) string {
	t.Helper()
	repo := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		file := filepath.Join(repo, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(file), 0o777); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(file, []byte(content), 0o666); err != nil {
			t.Fatal(err)
		}
	}
	write("mygo/go.mod", "module github.com/version-fox/vfox-plus/mygo\n\ngo 1.27\n")
	for name, content := range profiles {
		write("shared/profile/plugins/"+name+".json", content)
	}
	return filepath.Join(repo, "mygo")
}

func TestIndexListsEveryProfileWithItsChecksum(t *testing.T) {
	root := repository(t, map[string]string{"b": "{}\n", "a": "{\"schema\": 2}\n"})
	if err := Run([]string{"index", "--root", root}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "..", "shared", "profile", "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"schema\": 2,\n  \"plugins\": {\n    \"a\": \"674b14804543b6902bbd1d96a55f4326f94c747e8acbb6dff9e937154619d39f\",\n    \"b\": \"ca3d163bab055381827226140568f3bef7eaac187cebd76878e0b63e9e442356\"\n  }\n}\n"
	if string(content) != want {
		t.Errorf("index = %s, want %s", content, want)
	}
}

func TestLintRunsInTheModuleWithAbsolutePaths(t *testing.T) {
	root := repository(t, map[string]string{"java": "{}"})
	got, err := lintArguments(root, "", false, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"run", "./core/profile/cmd/profilelint", "--directory", filepath.Join(root, "..", "shared", "profile", "plugins", "java.json")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("arguments = %v, want %v", got, want)
	}
	plugin := t.TempDir()
	got, err = lintArguments(root, plugin, true, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"run", "./core/profile/cmd/profilelint", "--strict", "--plugin", plugin}; !reflect.DeepEqual(got, want) {
		t.Errorf("arguments = %v, want %v", got, want)
	}
}

func TestProfileNeedsASubcommand(t *testing.T) {
	for _, args := range [][]string{nil, {"unknown"}, {"index", "extra"}} {
		if err := Run(args); err == nil {
			t.Errorf("Run(%q) succeeded", args)
		}
	}
}
