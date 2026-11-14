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

package release

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeCheckout writes the files that carry the version, as the checkout has them
// before a release: version is the version they say now.
func writeCheckout(t *testing.T, root, version string) {
	t.Helper()
	files := map[string]string{
		VersionFile:  version + "\n",
		AppFile:      "package ui\n\nconst Version = \"" + version + "\"\n",
		MetainfoFile: "<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<component>\n  <releases>\n    <release version=\"" + version + "\" date=\"2026-10-17\"/>\n  </releases>\n</component>\n",
	}
	for file, text := range files {
		writeFile(t, root, file, text)
	}
}

// writeFile writes text to file, under root, creating its directories.
func writeFile(t *testing.T, root, file, text string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readFile returns the content of file, under root.
func readFile(t *testing.T, root, file string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSetVersionWritesTheThreeFiles(t *testing.T) {
	root := t.TempDir()
	writeCheckout(t, root, "0.1.0")

	if err := SetVersion(root, "1.0.0", "2026-11-15"); err != nil {
		t.Fatalf("SetVersion: %v", err)
	}
	if got := readFile(t, root, VersionFile); got != "1.0.0\n" {
		t.Errorf("VERSION = %q, want 1.0.0 with a newline", got)
	}
	if app := readFile(t, root, AppFile); !strings.Contains(app, `const Version = "1.0.0"`) || strings.Contains(app, "0.1.0") {
		t.Errorf("app.go does not carry only the new version:\n%s", app)
	}
	metainfo := readFile(t, root, MetainfoFile)
	entry := `    <release version="1.0.0" date="2026-11-15"/>`
	if !strings.Contains(metainfo, entry+"\n") {
		t.Errorf("metainfo has no entry %q:\n%s", entry, metainfo)
	}
	if !strings.Contains(metainfo, `<release version="0.1.0"`) {
		t.Errorf("the entry of the old version was removed:\n%s", metainfo)
	}
}

func TestSetVersionIsRepeatable(t *testing.T) {
	root := t.TempDir()
	writeCheckout(t, root, "0.1.0")
	for i := 0; i < 2; i++ {
		if err := SetVersion(root, "1.0.0", "2026-11-15"); err != nil {
			t.Fatalf("SetVersion run %d: %v", i+1, err)
		}
	}
	if n := strings.Count(readFile(t, root, MetainfoFile), `<release version="1.0.0"`); n != 1 {
		t.Errorf("the entry of 1.0.0 appears %d times, want once", n)
	}
}

func TestSetVersionRefusesWhatItCannotWrite(t *testing.T) {
	cases := []struct {
		name    string
		version string
		date    string
		prepare func(t *testing.T, root string)
		wantErr string
	}{
		{name: "a version that is not major.minor.patch", version: "1.0", date: "2026-11-15",
			wantErr: "Not a version"},
		{name: "a date that is not YYYY-MM-DD", version: "1.0.0", date: "15-11-2026",
			wantErr: "must be YYYY-MM-DD"},
		{name: "app.go without a Version constant", version: "1.0.0", date: "2026-11-15",
			prepare: func(t *testing.T, root string) {
				writeFile(t, root, AppFile, "package ui\n")
			},
			wantErr: "must have one Version constant, found 0"},
		{name: "a metainfo without a releases list", version: "1.0.0", date: "2026-11-15",
			prepare: func(t *testing.T, root string) {
				writeFile(t, root, MetainfoFile, "<component/>\n")
			},
			wantErr: "has no <releases> list"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			writeCheckout(t, root, "0.1.0")
			if c.prepare != nil {
				c.prepare(t, root)
			}
			err := SetVersion(root, c.version, c.date)
			if err == nil || !strings.Contains(err.Error(), c.wantErr) {
				t.Fatalf("SetVersion error = %v, want one containing %q", err, c.wantErr)
			}
		})
	}
}
