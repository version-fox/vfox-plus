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

package screens

import (
	"testing"

	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// TestTheSourceOfAPluginIsItsRepository covers the short source of the web's
// plugins.ts: the owner and the repository of a GitHub homepage, and the address
// without its scheme for any other.
func TestTheSourceOfAPluginIsItsRepository(t *testing.T) {
	cases := map[string]string{
		"https://github.com/version-fox/vfox-java":     "version-fox/vfox-java",
		"https://github.com/version-fox/vfox-java.git": "version-fox/vfox-java",
		"https://GitHub.com/A/B/":                      "A/B",
		"https://github.com/owner/repo/tree/main?x=1":  "owner/repo",
		"https://example.com/rust/":                    "example.com/rust",
		"http://example.com":                           "example.com",
		"":                                             "",
	}
	for homepage, want := range cases {
		if got := pluginShortSource(homepage); got != want {
			t.Errorf("pluginShortSource(%q) = %q, want %q", homepage, got, want)
		}
	}
}

// TestOnlyAWebAddressIsOpened covers the web's webAddress, with the cases of its test:
// an absolute http or https address is opened, and any other is refused.
func TestOnlyAWebAddressIsOpened(t *testing.T) {
	accepted := map[string]string{
		"https://github.com/version-fox/vfox-dart": "https://github.com/version-fox/vfox-dart",
		" http://127.0.0.1:8080/app ":              "http://127.0.0.1:8080/app",
		"HTTPS://example.com/docs":                 "https://example.com/docs",
	}
	for raw, want := range accepted {
		got, ok := pluginWebAddress(raw)
		if !ok || got != want {
			t.Errorf("pluginWebAddress(%q) = %q, %v; want %q", raw, got, ok, want)
		}
	}
	refused := []string{
		"javascript:alert(1)",
		"JavaScript:alert(1)",
		" javascript:alert(1)",
		"data:text/html,<script>alert(1)</script>",
		"file:///C:/Windows/System32/calc.exe",
		"ms-msdt:/id PCWDiagnostic",
		"ftp://example.com/installer.exe",
		"mailto:someone",
		"https://",
		"github.com/version-fox/vfox-dart",
		"",
	}
	for _, raw := range refused {
		if got, ok := pluginWebAddress(raw); ok {
			t.Errorf("pluginWebAddress(%q) = %q, want it refused", raw, got)
		}
	}
}

// TestAPluginIsKeptByItsFilterAndItsSearch covers the registry's filters, which are
// all, official and community, and the search, which matches the name or the homepage.
func TestAPluginIsKeptByItsFilterAndItsSearch(t *testing.T) {
	official := remote.AvailablePlugin{Name: "go", Homepage: "https://github.com/version-fox/vfox-go", Official: true}
	community := remote.AvailablePlugin{Name: "rust", Homepage: "https://example.com/rust", Official: false}

	cases := []struct {
		plugin remote.AvailablePlugin
		filter string
		q      string
		want   bool
	}{
		{official, pluginFilterAll, "", true},
		{official, pluginFilterOfficial, "", true},
		{official, pluginFilterCommunity, "", false},
		{community, pluginFilterOfficial, "", false},
		{community, pluginFilterCommunity, "", true},
		{community, pluginFilterAll, "rust", true},
		{community, pluginFilterAll, "example", true},
		{community, pluginFilterAll, "go", false},
		{official, pluginFilterAll, "version-fox", true},
	}
	for _, c := range cases {
		if got := pluginMatches(c.plugin, c.filter, c.q); got != c.want {
			t.Errorf("pluginMatches(%s, %q, %q) = %v, want %v", c.plugin.Name, c.filter, c.q, got, c.want)
		}
	}
}

// TestTheInstalledPluginsAreNamed covers the marks that the cards of the registry show.
func TestTheInstalledPluginsAreNamed(t *testing.T) {
	installed := pluginInstalled([]remote.InstalledSdk{{Name: "go"}, {Name: "java"}})
	if !installed["go"] || !installed["java"] || installed["rust"] {
		t.Errorf("pluginInstalled = %v", installed)
	}
}
