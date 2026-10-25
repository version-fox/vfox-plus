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

//go:build !windows

package vfox_test

import (
	"reflect"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/core/vfox"
)

func TestLocator(t *testing.T) {
	locator := vfox.NewVfoxLocator(map[string]string{"PATH": "/opt/bin::/usr/bin", "HOME": "/home/me"}, func(string) bool { return false })
	want := []string{"/opt/bin", "/usr/bin", "/usr/local/bin", "/home/me/.local/bin", "/home/me/bin", "/home/me/.linuxbrew/bin", "/home/linuxbrew/.linuxbrew/bin", "/snap/bin"}
	if got := locator.SearchDirectories(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %q", got)
	}
	found := map[string]bool{"/home/me/.local/bin/vfox": true, "/snap/bin/vfox": true}
	locator = vfox.NewVfoxLocator(map[string]string{"HOME": "/home/me"}, func(path string) bool { return found[path] })
	if got := locator.Locate(); got != "/home/me/.local/bin/vfox" {
		t.Errorf("got %q", got)
	}
	if got := vfox.NewVfoxLocator(map[string]string{}, func(string) bool { return false }).Locate(); got != "" {
		t.Errorf("got %q", got)
	}
}
