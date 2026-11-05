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

package main

import (
	"path/filepath"
	"testing"
)

func TestTheFakeVfoxIsTheProgramNamedVfox(t *testing.T) {
	for program, want := range map[string]bool{
		"vfox":                               true,
		"/tmp/sandbox/vfox":                  true,
		"vfox.exe":                           true,
		filepath.Join("sandbox", "vfox.exe"): true,  // The separator of the system, as the sandbox of a test is made.
		"/tmp/sandbox/VFOX.EXE":              false, // Names are case sensitive; only the extension is not.
		"vfxtool":                            false,
		"vfxtool.exe":                        false,
		"/tmp/vfox/vfxtool":                  false,
		"vfox.sh":                            false,
		"vfox-plus":                          false,
		"":                                   false,
	} {
		if got := isFakeVfox(program); got != want {
			t.Errorf("isFakeVfox(%q) = %t, want %t", program, got, want)
		}
	}
}
