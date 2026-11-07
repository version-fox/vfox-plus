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

package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseProfileJSON(t *testing.T) {
	content := []byte(`{"plugin":"gradle","naming":"plain",` +
		`"variables":[{"key":"VFOX_GRADLE_MIRRORS","label":{"en":"Download mirror"},"help":{"en":"Mirror"},"kind":"url","example":"https://","presets":[]}],` +
		`"facets":[],"guide":{"en":"Guide"},"tips":[],"docs":[]}`)
	profile, err := ParseProfileJSON(content)
	if err != nil || profile.Plugin != "gradle" || len(profile.Variables) != 1 {
		t.Fatalf("got %+v, %v", profile, err)
	}
	if _, err := ParseProfileJSON([]byte(`{"naming":"plain"}`)); err == nil {
		t.Error("a profile without a plugin must fail")
	}
}

func TestLoadExternalProfilesOverlaysBuiltins(t *testing.T) {
	directory := t.TempDir()
	content := []byte(`{"plugin":"nodejs","guide":{"en":"External","zh":"外部"}}`)
	if err := os.WriteFile(filepath.Join(directory, "nodejs.json"), content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "broken.json"), []byte(`{`), 0o600); err != nil {
		t.Fatal(err)
	}
	registry := NewRegistry(Platform{OS: "linux", Arch: "amd64"})
	if loaded := LoadExternalProfiles(registry, directory); loaded != 1 {
		t.Fatalf("loaded %d", loaded)
	}
	profile, ok := registry.Lookup("nodejs")
	if !ok || profile.Guide.EN != "External" {
		t.Errorf("got %+v, %v", profile, ok)
	}
}

func TestProfileURLSupportsGitHubOnly(t *testing.T) {
	got := ProfileURL("https://github.com/version-fox/vfox-gradle")
	want := "https://raw.githubusercontent.com/version-fox/vfox-gradle/main/vfox-plus.json"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := ProfileURL("https://example.com/x"); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestMirrorURLPrefixes(t *testing.T) {
	raw := "https://raw.githubusercontent.com/version-fox/vfox-gradle/main/vfox-plus.json"
	if got := MirrorURL("https://gh-proxy.org/", raw); got != "https://gh-proxy.org/"+raw {
		t.Errorf("got %q", got)
	}
	if got := MirrorURL("https://gh-proxy.org", raw); got != "https://gh-proxy.org/"+raw {
		t.Errorf("missing slash: %q", got)
	}
	if got := MirrorURL("", raw); got != raw {
		t.Errorf("got %q", got)
	}
}
