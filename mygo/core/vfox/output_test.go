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

package vfox_test

import (
	"reflect"
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
)

func TestParseVfoxVersion(t *testing.T) {
	if got := vfox.ParseVfoxVersion("vfox version 1.0.12\n"); got != "1.0.12" {
		t.Errorf("got %q", got)
	}
	if got := vfox.ParseVfoxVersion("vfox version 2.0.0-beta.1"); got != "2.0.0-beta.1" {
		t.Errorf("got %q", got)
	}
	if got := vfox.ParseVfoxVersion("command not found"); got != "" {
		t.Errorf("got %q", got)
	}
}

func TestParseCurrent(t *testing.T) {
	got := vfox.ParseCurrent("go -> vv1.2\n")
	want := []vfox.CurrentVersion{{Name: "go", Version: "v1.2", HasVersion: true}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v", got)
	}
	got = vfox.ParseCurrent("Warning: something\njava -> \x1b[92mv21\x1b[0m\nmaven -> N/A \nbad name -> v1\n")
	if len(got) != 2 || got[0] != (vfox.CurrentVersion{Name: "java", Version: "21", HasVersion: true}) || got[1].Name != "maven" || got[1].HasVersion {
		t.Errorf("got %#v", got)
	}
}

func TestParseInstalledTree(t *testing.T) {
	got := vfox.ParseInstalledTree("All installed sdk versions\n└──maven\n")
	if len(got) != 1 || got[0].Name != "maven" || len(got[0].Versions) != 0 {
		t.Fatalf("got %#v", got)
	}
	got = vfox.ParseInstalledTree("All installed sdk versions\n\n└─┬nodejs\n  └──24.11.1\n\n")
	if len(got) != 1 || !reflect.DeepEqual(got[0].Versions, []string{"24.11.1"}) {
		t.Fatalf("got %#v", got)
	}
	if got := vfox.ParseInstalledTree("  └──24.11.1\n"); len(got) != 0 {
		t.Errorf("versions before any plugin must be ignored, got %#v", got)
	}
}

func TestMergeInstalledSdks(t *testing.T) {
	got := vfox.MergeInstalledSdks(
		[]vfox.SdkVersions{{Name: "java", Versions: []string{"21"}}},
		[]vfox.CurrentVersion{{Name: "java", Version: "21", HasVersion: true}, {Name: "go"}},
	)
	want := []core.InstalledSdk{
		{Name: "java", Versions: []string{"21"}, Current: "21", HasCurrent: true},
		{Name: "go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v", got)
	}
}

func TestParseSearch(t *testing.T) {
	got := vfox.ParseSearch("Available versions:\n" +
		" - 24.22.0 (LTS) [npm 11.19.1,corepack 0.34.0] (installed)\n" +
		" - 1.0 (release candidate)\n" +
		" - 0.1.14 [npm ]\n" +
		" - 2.0\n" +
		" - 3.0 ()\n")
	want := []core.AvailableVersion{
		{Version: "24.22.0", Note: "LTS", HasNote: true, Additions: []core.VersionAddition{{Name: "npm", Version: "11.19.1"}, {Name: "corepack", Version: "0.34.0"}}, Installed: true},
		{Version: "1.0", Note: "release candidate", HasNote: true},
		{Version: "0.1.14", Additions: []core.VersionAddition{{Name: "npm"}}},
		{Version: "2.0"},
		{Version: "3.0"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v", got)
	}
	if additions := []string{got[0].Additions[0].String(), got[0].Additions[1].String(), got[2].Additions[0].String()}; !reflect.DeepEqual(additions, []string{"npm 11.19.1", "corepack 0.34.0", "npm"}) {
		t.Errorf("got %q", additions)
	}
	if got := vfox.ParseSearch(" - 1.0 preview\n"); len(got) != 1 || got[0].Note != "preview" {
		t.Errorf("got %#v", got)
	}
	if got := vfox.ParseSearch("Please select a version\n -\n - \n"); len(got) != 0 {
		t.Errorf("got %#v", got)
	}
}

func TestParsePluginInfo(t *testing.T) {
	info, ok := vfox.ParsePluginInfo("a\x1f1.0\x1fhttps://x\x1f/p\x1fone\x1ftwo")
	if !ok || info.Description != "one\x1ftwo" || info.Homepage != "https://x" {
		t.Errorf("got %#v %v", info, ok)
	}
	if _, ok := vfox.ParsePluginInfo("nodejs not supported"); ok {
		t.Error("incomplete output must be rejected")
	}
}

func TestParseInstallPath(t *testing.T) {
	if path, ok := vfox.ParseInstallPath("notfound\n"); ok || path != "" {
		t.Errorf("got %q %v", path, ok)
	}
	if path, ok := vfox.ParseInstallPath("/opt/sdk/v-1\n"); !ok || path != "/opt/sdk/v-1" {
		t.Errorf("got %q %v", path, ok)
	}
}

func TestParseConfigList(t *testing.T) {
	got := vfox.ParseConfigList("proxy.url = \nfoo.bar = a = b\nnot a setting\n = x\n")
	want := []core.ConfigEntry{{Key: "proxy.url"}, {Key: "foo.bar", Value: "a = b"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v", got)
	}
	got = vfox.ParseConfigList("registry.address =\n")
	if !reflect.DeepEqual(got, []core.ConfigEntry{{Key: "registry.address"}}) {
		t.Errorf("got %#v", got)
	}
}

func TestParseDownloadProgress(t *testing.T) {
	if got := vfox.ParseDownloadProgress("Downloading...  42% [===>  ] (1 MB/s) [1s:2s]"); got != 0.42 {
		t.Errorf("got %v", got)
	}
	if got := vfox.ParseDownloadProgress("Downloading... 100% [======]"); got != 1 {
		t.Errorf("got %v", got)
	}
	if got := vfox.ParseDownloadProgress("Downloading... 250% [======]"); got != 1 {
		t.Errorf("got %v", got)
	}
	if got := vfox.ParseDownloadProgress("Preinstalling maven@3.9.16..."); got >= 0 {
		t.Errorf("got %v", got)
	}
}

func TestSummarizeError(t *testing.T) {
	if got := vfox.SummarizeError("Removing...\n", "Use the -y flag\n"); got != "Use the -y flag" {
		t.Errorf("got %q", got)
	}
	got := vfox.SummarizeError(
		"Preinstalling x@1...\nfailed to download x@1 file, err:EOF\nfailed to install x\n",
		"\rDownloading...  3% [>   ]\r      \r")
	if got != "failed to download x@1 file, err:EOF" {
		t.Errorf("got %q", got)
	}
	// vfox 1.0.13 prints the error a command returns to standard error.
	got = vfox.SummarizeError(
		"Preinstalling x@1...\nfailed to download x@1 file, err:EOF\n",
		"\rDownloading...  3% [>   ]\r      \rfailed to install x\n")
	if got != "failed to download x@1 file, err:EOF" {
		t.Errorf("got %q", got)
	}
	if got := vfox.SummarizeError("", "  \n"); got != "" {
		t.Errorf("got %q", got)
	}
	if got := vfox.SummarizeError("", "failed to install x\n"); got != "failed to install x" {
		t.Errorf("got %q", got)
	}
}

func TestStripAnsi(t *testing.T) {
	if got := core.StripANSI("\x1b[1;32mok\x1b[0m \x1b]8;;https://x\x07link\x1b]8;;\x07 \x1bMx"); got != "ok link x" {
		t.Errorf("got %q", got)
	}
}

func TestSplitTerminalLines(t *testing.T) {
	if got := core.SplitTerminalLines("a\r\nb\rc\nd"); !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Errorf("got %q", got)
	}
}

func TestLineSplitter(t *testing.T) {
	splitter := core.TerminalLineSplitter{}
	if got := splitter.Push([]byte("one\r\x1b[32mtw")); !reflect.DeepEqual(got, []string{"one"}) {
		t.Fatalf("got %q", got)
	}
	// A multi-byte character split between two chunks stays intact.
	euro := []byte("€")
	if got := splitter.Push(append([]byte("o "), euro[:1]...)); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
	{
		chunk := append(euro[1:], '\n', ' ', '\n')
		if got := splitter.Push(chunk); !reflect.DeepEqual(got, []string{"two €"}) {
			t.Fatalf("got %q", got)
		}
	}
	if got := splitter.Push([]byte("last")); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
	if got := splitter.Close(); !reflect.DeepEqual(got, []string{"last"}) {
		t.Fatalf("got %q", got)
	}
	if got := splitter.Close(); len(got) != 0 {
		t.Fatalf("got %q", got)
	}
}

func TestLineSplitterReplacesMalformedBytes(t *testing.T) {
	splitter := core.TerminalLineSplitter{}
	if got := splitter.Push([]byte{'a', 0xff, 'b', '\n'}); !reflect.DeepEqual(got, []string{"a�b"}) {
		t.Errorf("got %q", got)
	}
}

func TestVersionAdditionString(t *testing.T) {
	if got := (core.VersionAddition{Name: "npm", Version: "11.19.1"}).String(); got != "npm 11.19.1" {
		t.Errorf("got %q", got)
	}
	if got := (core.VersionAddition{Name: "npm"}).String(); got != "npm" {
		t.Errorf("got %q", got)
	}
}
