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

package validatemsix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckSelfSignInputs(t *testing.T) {
	dir := t.TempDir()
	err := checkSelfSignInputs(dir)
	if err == nil || !strings.Contains(err.Error(), "missing self-sign input: stage-x64/vfox-plus.exe") {
		t.Fatalf("checkSelfSignInputs on an empty directory = %v", err)
	}
	for _, rel := range selfSignInputs {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := checkSelfSignInputs(dir); err != nil {
		t.Fatalf("checkSelfSignInputs with every input: %v", err)
	}
}
