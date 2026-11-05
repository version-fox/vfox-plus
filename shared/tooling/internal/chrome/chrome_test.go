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

package chrome

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestFind(t *testing.T) {
	t.Setenv("CHROME_BIN", "/from/env")
	if got, err := Find("/named"); err != nil || got != "/named" {
		t.Errorf("a named browser: %q, %v", got, err)
	}
	if got, err := Find(""); err != nil || got != "/from/env" {
		t.Errorf("$CHROME_BIN: %q, %v", got, err)
	}
	t.Setenv("CHROME_BIN", "")
	bin := t.TempDir()
	t.Setenv("PATH", bin)
	if _, err := Find(""); err == nil && runtime.GOOS != "darwin" {
		t.Error("found a Chrome in an empty PATH")
	}
	if err := os.WriteFile(filepath.Join(bin, "chromium"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" {
		if got, err := Find(""); err != nil || got != filepath.Join(bin, "chromium") {
			t.Errorf("chromium in PATH: %q, %v", got, err)
		}
	}
}
