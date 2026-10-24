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

package sandbox

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestShortHash(t *testing.T) {
	// sha256("abc") = ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad.
	if got := ShortHash([]byte("abc")); got != "ba7816bf8f01" {
		t.Fatalf("ShortHash = %q, want ba7816bf8f01", got)
	}
}

func TestContentTag(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(dir, "b")
	if err := os.WriteFile(a, []byte("foo"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("bar"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ContentTag([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("foobar"))
	want := hex.EncodeToString(sum[:])[:12]
	if got != want {
		t.Fatalf("ContentTag = %q, want %q", got, want)
	}
	// The tag changes when any input changes, so that an image is rebuilt
	// exactly when its inputs change.
	changed, err := ContentTag([]string{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if changed == got {
		t.Fatal("ContentTag ignores the order of the inputs")
	}
	if _, err := ContentTag([]string{filepath.Join(dir, "missing")}); err == nil {
		t.Fatal("expected an error for a missing input")
	}
}

func TestBaseImageName(t *testing.T) {
	got := BaseImageName("1.0.13", "abcdef123456")
	want := "vfox-plus-sandbox-base:abcdef123456-vfox-1.0.13"
	if got != want {
		t.Fatalf("BaseImageName = %q, want %q", got, want)
	}
}

func TestImplImageName(t *testing.T) {
	dockerfile := []byte("FROM base\n")
	versions := []byte("VFOX_VERSION=1.0.13\n")
	got := ImplImageName("vfox-plus-sandbox-mygo", "parent:tag", dockerfile, versions)
	sum := sha256.Sum256([]byte("FROM base\nVFOX_VERSION=1.0.13\nparent:tag\n"))
	want := "vfox-plus-sandbox-mygo:" + hex.EncodeToString(sum[:])[:12]
	if got != want {
		t.Fatalf("ImplImageName = %q, want %q", got, want)
	}
	// The parent image is part of the tag, with the trailing newline of echo.
	other := ImplImageName("vfox-plus-sandbox-mygo", "other:tag", dockerfile, versions)
	if other == got {
		t.Fatal("ImplImageName ignores the parent image")
	}
}

func TestReadVersions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "versions.env")
	content := "# a comment\n\nVFOX_VERSION=1.0.13\nVFOX_SHA256=abc123\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	versions, err := readVersions(path)
	if err != nil {
		t.Fatal(err)
	}
	if versions["VFOX_VERSION"] != "1.0.13" || versions["VFOX_SHA256"] != "abc123" {
		t.Fatalf("unexpected versions: %v", versions)
	}
}
