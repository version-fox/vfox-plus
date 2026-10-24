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

package installpkgs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePrintURIs(t *testing.T) {
	output := "'http://mirror/ubuntu/a.deb' a.deb 100 SHA256:aaa\n" +
		"not a uris line\n" +
		"'http://mirror/ubuntu/b.deb' b.deb 200 SHA256:bbb b\n"
	entries := parsePrintURIs(output)
	if len(entries) != 2 {
		t.Fatalf("entries = %v", entries)
	}
	if entries[0].size != 100 || entries[0].fields[1] != "a.deb" || entries[0].fields[3] != "http://mirror/ubuntu/a.deb" {
		t.Fatalf("entries[0] = %+v", entries[0])
	}
	if entries[1].fields[2] != "SHA256:bbb" {
		t.Fatalf("entries[1] = %+v", entries[1])
	}
}

func TestExpandMirrors(t *testing.T) {
	entries := []entry{
		{size: 10, fields: []string{"10", "a.deb", "SHA256:aaa", "mirror+file:/etc/apt/apt-mirrors.txt/ubuntu/a.deb"}},
		{size: 20, fields: []string{"20", "b.deb", "SHA256:bbb", "http://other/b.deb"}},
	}
	expanded := expand(entries, []string{"http://one/", "http://two"})
	// The largest comes first.
	if expanded[0].fields[1] != "b.deb" {
		t.Fatalf("order = %v", expanded)
	}
	if expanded[1].fields[1] != "a.deb" {
		t.Fatalf("order = %v", expanded)
	}
	got := expanded[1].fields[3:]
	want := []string{"http://one/ubuntu/a.deb", "http://two/ubuntu/a.deb"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("addresses = %v", got)
	}
	// Without mirrors the address of apt stays as it is.
	kept := expand([]entry{{size: 1, fields: []string{"1", "a.deb", "H", "mirror+file:/etc/apt/apt-mirrors.txt/x"}}}, nil)
	if kept[0].fields[3] != "mirror+file:/etc/apt/apt-mirrors.txt/x" {
		t.Fatalf("addresses = %v", kept[0].fields)
	}
}

func TestMissing(t *testing.T) {
	directory := t.TempDir()
	old := archivesDir
	archivesDir = directory
	t.Cleanup(func() { archivesDir = old })
	if err := os.WriteFile(filepath.Join(directory, "a.deb"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	list := filepath.Join(t.TempDir(), "list")
	content := "100 a.deb SHA256:aaa http://m/a.deb\n200 b.deb SHA256:bbb http://m/b.deb\n"
	if err := os.WriteFile(list, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	missing, err := missing(list)
	if err != nil {
		t.Fatal(err)
	}
	if missing != 1 {
		t.Fatalf("missing = %d, want 1", missing)
	}
}

func TestRunUsage(t *testing.T) {
	if err := Run(nil); err == nil || !strings.Contains(err.Error(), "Usage: installpkgs") {
		t.Fatalf("err = %v", err)
	}
}
