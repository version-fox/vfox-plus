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
	"reflect"
	"testing"
)

// A plugin names its homepage in its own metadata, so the system must only ever
// be handed http and https addresses.
func TestOpenWebAddressHandsOnlyWebAddressesToTheSystem(t *testing.T) {
	var handed []string
	defer func(original func(string) error) { shellOpen = original }(shellOpen)
	shellOpen = func(address string) error {
		handed = append(handed, address)
		return nil
	}

	refused := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///C:/Windows/System32/calc.exe",
		"ms-msdt:/id PCWDiagnostic",
		"mailto:someone",
		// These have a host, so only the scheme check refuses them.
		"ftp://example.com/installer.exe",
		"smb://fileserver/share/tool.exe",
		"ms-windows-store://pdp/?ProductId=9WZDNCRFJ3PS",
		"intent://example.com/#Intent;scheme=https;end",
		"vfox-plus://pair/abc",
		"https://",
		"github.com/version-fox/vfox-dart",
		"",
		"   ",
	}
	for _, address := range refused {
		if err := openWebAddress(address); err == nil {
			t.Errorf("openWebAddress(%q) opened the address, want a refusal", address)
		}
	}
	if len(handed) != 0 {
		t.Fatalf("the system was handed %q, want nothing", handed)
	}

	accepted := []struct{ address, want string }{
		{"https://github.com/version-fox/vfox-dart", "https://github.com/version-fox/vfox-dart"},
		{" http://127.0.0.1:8080/auth/connect/abc ", "http://127.0.0.1:8080/auth/connect/abc"},
		{"HTTPS://example.com/docs", "HTTPS://example.com/docs"},
	}
	var want []string
	for _, item := range accepted {
		if err := openWebAddress(item.address); err != nil {
			t.Errorf("openWebAddress(%q) = %v, want it opened", item.address, err)
		}
		want = append(want, item.want)
	}
	if !reflect.DeepEqual(handed, want) {
		t.Fatalf("the system was handed %q, want %q", handed, want)
	}
}
