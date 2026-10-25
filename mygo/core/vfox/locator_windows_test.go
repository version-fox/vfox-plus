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

//go:build windows

package vfox_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/core/vfox"
)

func TestLocator(t *testing.T) {
	env := map[string]string{
		"PATH":         `C:\tools;D:\bin`,
		"USERPROFILE":  `C:\Users\me`,
		"LOCALAPPDATA": `C:\Users\me\AppData\Local`,
		"ProgramFiles": `C:\Program Files`,
	}
	locator := vfox.NewVfoxLocator(env, func(string) bool { return false })
	got := locator.SearchDirectories()
	for _, want := range []string{
		`C:\tools`,
		`D:\bin`,
		filepath.Join(`C:\Program Files`, "vfox"),
		filepath.Join(`C:\Users\me\AppData\Local`, "Programs", "vfox"),
	} {
		found := false
		for _, dir := range got {
			if dir == want {
				found = true
			}
		}
		if !found {
			t.Errorf("%q missing in %q", want, got)
		}
	}
	found := map[string]bool{filepath.Join(`C:\Users\me\AppData\Local`, "Programs", "vfox", "vfox.exe"): true}
	locator = vfox.NewVfoxLocator(env, func(path string) bool { return found[path] })
	want := filepath.Join(`C:\Users\me\AppData\Local`, "Programs", "vfox", "vfox.exe")
	if got := locator.Locate(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if vfox.VfoxExecutableName != "vfox.exe" {
		t.Errorf("VfoxExecutableName = %q, want vfox.exe", vfox.VfoxExecutableName)
	}
	_ = reflect.DeepEqual
}
