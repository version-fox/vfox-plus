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

package buildmsix

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMapMsixArch(t *testing.T) {
	for goarch, want := range map[string]string{"amd64": "x64", "x64": "x64", "arm64": "arm64"} {
		got, err := MapMsixArch(goarch)
		if err != nil || got != want {
			t.Errorf("MapMsixArch(%q) = %q, %v, want %q", goarch, got, err, want)
		}
	}
	if _, err := MapMsixArch("riscv64"); err == nil {
		t.Error("expected an error for riscv64")
	}
}

func TestToQuadVersion(t *testing.T) {
	for in, want := range map[string]string{
		"0.1.0":   "0.1.0.0",
		"1.2":     "1.2.0.0",
		"3":       "3.0.0.0",
		"1.2.3.4": "1.2.3.4",
	} {
		got, err := ToQuadVersion(in)
		if err != nil || got != want {
			t.Errorf("ToQuadVersion(%q) = %q, %v, want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "1.2.3.4.5", "a.b.c", "1..3", "70000.0.0.0"} {
		if _, err := ToQuadVersion(bad); err == nil {
			t.Errorf("ToQuadVersion(%q) = nil error", bad)
		}
	}
}

func testMsixConfig() MsixConfig {
	exe := append([]byte{'M', 'Z'}, bytes.Repeat([]byte{0x90}, 100000)...)
	return MsixConfig{
		Identity:             "dev.vfox.VfoxPlus",
		Publisher:            defaultPublisher,
		Version:              "0.1.0.0",
		Architecture:         "x64",
		DisplayName:          "vfox+",
		PublisherDisplayName: "Han Li and contributors",
		Description:          "Install and switch SDK versions with vfox",
		Executable:           "vfox-plus.exe",
		ExecutableData:       exe,
		MTime:                time.Unix(1234567890, 0).UTC(),
	}
}

func readZip(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string][]byte{}
	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(rc); err != nil {
			t.Fatal(err)
		}
		rc.Close()
		out[f.Name] = buf.Bytes()
		if want := methodOf(f.Name); f.Method != want {
			t.Errorf("%s uses method %d, want %d", f.Name, f.Method, want)
		}
	}
	return out
}

func TestWriteMsix(t *testing.T) {
	msix, err := WriteMsix(testMsixConfig())
	if err != nil {
		t.Fatal(err)
	}
	files := readZip(t, msix)
	for _, name := range []string{
		"AppxManifest.xml",
		"[Content_Types].xml",
		"AppxBlockMap.xml",
		"vfox-plus.exe",
		"Assets/StoreLogo.png",
		"Assets/Square44x44Logo.png",
		"Assets/Square150x150Logo.png",
		"Assets/Wide310x150Logo.png",
	} {
		if _, ok := files[name]; !ok {
			t.Errorf("%s missing in .msix", name)
		}
	}
	manifest := string(files["AppxManifest.xml"])
	for _, want := range []string{
		`Name="dev.vfox.VfoxPlus"`,
		`Publisher="CN=vfox-plus, OID.2.25.311729368913984317654407730594956997722=1"`,
		`Version="0.1.0.0"`,
		`ProcessorArchitecture="x64"`,
		`Executable="vfox-plus.exe"`,
		`TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.22000.0"`,
		`runFullTrust`,
		`Language="zh-cn"`,
	} {
		if !strings.Contains(manifest, want) {
			t.Errorf("manifest missing %q:\n%s", want, manifest)
		}
	}
	blockmap := string(files["AppxBlockMap.xml"])
	if !strings.Contains(blockmap, `Name="vfox-plus.exe"`) || !strings.Contains(blockmap, `Name="Assets\Square44x44Logo.png"`) {
		t.Errorf("blockmap names wrong:\n%s", blockmap)
	}
	// PNG magic.
	for _, name := range []string{"Assets/StoreLogo.png", "Assets/Square44x44Logo.png"} {
		if len(files[name]) < 8 || string(files[name][:8]) != "\x89PNG\r\n\x1a\n" {
			t.Errorf("%s is not a PNG", name)
		}
	}
}

func TestWriteMsixRejectsNonWindows(t *testing.T) {
	cfg := testMsixConfig()
	cfg.ExecutableData = []byte("#!/bin/sh\necho hi\n")
	if _, err := WriteMsix(cfg); err == nil {
		t.Error("expected an error for a non-MZ executable")
	}
}

func TestWriteBundle(t *testing.T) {
	x64, err := WriteMsix(testMsixConfig())
	if err != nil {
		t.Fatal(err)
	}
	armCfg := testMsixConfig()
	armCfg.Architecture = "arm64"
	arm64, err := WriteMsix(armCfg)
	if err != nil {
		t.Fatal(err)
	}
	mtime := time.Unix(1234567890, 0).UTC()
	bundle, name, err := WriteBundle(BundleConfig{
		Package:   "vfox-plus",
		Version:   "0.1.0",
		Identity:  "dev.vfox.VfoxPlus",
		Publisher: defaultPublisher,
		Packages: []BundlePackage{
			{Architecture: "x64", Filename: "vfox-plus_0.1.0_x64.msix", Data: x64},
			{Architecture: "arm64", Filename: "vfox-plus_0.1.0_arm64.msix", Data: arm64},
		},
		MTime: mtime,
	})
	if err != nil {
		t.Fatal(err)
	}
	if name != "vfox-plus_0.1.0.msixbundle" {
		t.Errorf("bundle name = %q", name)
	}
	files := readZip(t, bundle)
	for _, want := range []string{
		"vfox-plus_0.1.0_x64.msix",
		"vfox-plus_0.1.0_arm64.msix",
		"AppxMetadata/AppxBundleManifest.xml",
		"[Content_Types].xml",
		"AppxBlockMap.xml",
	} {
		if _, ok := files[want]; !ok {
			t.Errorf("%s missing in bundle", want)
		}
	}
	manifest := string(files["AppxMetadata/AppxBundleManifest.xml"])
	for _, want := range []string{
		`Architecture="x64" FileName="vfox-plus_0.1.0_x64.msix"`,
		`Architecture="arm64" FileName="vfox-plus_0.1.0_arm64.msix"`,
		`Version="0.1.0.0"`,
	} {
		if !strings.Contains(manifest, want) {
			t.Errorf("bundle manifest missing %q:\n%s", want, manifest)
		}
	}
	// Embedded packages round-trip.
	for _, want := range []string{"vfox-plus_0.1.0_x64.msix", "vfox-plus_0.1.0_arm64.msix"} {
		inner := readZip(t, files[want])
		if _, ok := inner["AppxManifest.xml"]; !ok {
			t.Errorf("%s has no manifest", want)
		}
	}
}

func TestReproducible(t *testing.T) {
	build := func() []byte {
		msix, err := WriteMsix(testMsixConfig())
		if err != nil {
			t.Fatal(err)
		}
		bundle, _, err := WriteBundle(BundleConfig{
			Package:   "vfox-plus",
			Version:   "0.1.0",
			Identity:  "dev.vfox.VfoxPlus",
			Publisher: defaultPublisher,
			Packages:  []BundlePackage{{Architecture: "x64", Filename: "a.msix", Data: msix}},
			MTime:     time.Unix(1234567890, 0).UTC(),
		})
		if err != nil {
			t.Fatal(err)
		}
		return bundle
	}
	first, second := build(), build()
	if !bytes.Equal(first, second) {
		t.Fatal("two builds produced different bundles")
	}
}

func TestRunBuildsMsixAndBundle(t *testing.T) {
	dir := t.TempDir()
	stageX64 := filepath.Join(dir, "stage-x64")
	stageArm := filepath.Join(dir, "stage-arm64")
	for _, stage := range []string{stageX64, stageArm} {
		if err := os.MkdirAll(stage, 0o755); err != nil {
			t.Fatal(err)
		}
		exe := append([]byte{'M', 'Z'}, bytes.Repeat([]byte{0x90}, 1000)...)
		if err := os.WriteFile(filepath.Join(stage, "vfox-plus.exe"), exe, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out := filepath.Join(dir, "out")
	t.Setenv("SOURCE_DATE_EPOCH", "1234567890")
	if err := Run([]string{
		"--package", "vfox-plus",
		"--version", "0.1.0",
		"--stage-x64", stageX64,
		"--stage-arm64", stageArm,
		"--executable", "vfox-plus.exe",
		out,
	}); err != nil {
		t.Fatal(err)
	}
	for _, pattern := range []string{"*_x64.msix", "*_arm64.msix", "*.msixbundle"} {
		matches, err := filepath.Glob(filepath.Join(out, pattern))
		if err != nil || len(matches) != 1 {
			t.Fatalf("expected one %s, got %v, %v", pattern, matches, err)
		}
	}
}

// A stored entry that osslsigncode rewrites comes out as a corrupt entry (the
// data deflated, the method still Store), which Windows refuses; the content
// types are the one part that it rewrites, so that part is a deflated entry
// and the rest stays stored.
func TestContentTypesAreDeflatedAndReadBack(t *testing.T) {
	msix, err := WriteMsix(testMsixConfig())
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(msix), int64(len(msix)))
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for _, file := range reader.File {
		if file.Name != "[Content_Types].xml" {
			if file.Method != zip.Store || file.CompressedSize64 != file.UncompressedSize64 {
				t.Errorf("%s: method %d, %d of %d bytes", file.Name, file.Method, file.CompressedSize64, file.UncompressedSize64)
			}
			continue
		}
		seen = true
		if file.Method != zip.Deflate {
			t.Errorf("the content types use method %d", file.Method)
		}
		// Reading checks the CRC: the stream inflates to what was written.
		content, err := readZipFile(reader, file.Name)
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != RenderContentTypes() {
			t.Errorf("the content types read back as %q", content)
		}
		if file.CompressedSize64 <= file.UncompressedSize64 {
			t.Errorf("a stored block adds its header, but %d bytes became %d", file.UncompressedSize64, file.CompressedSize64)
		}
	}
	if !seen {
		t.Fatal("no content types")
	}
}

func TestStoredDeflateInflatesToTheInput(t *testing.T) {
	for _, size := range []int{0, 1, 100, 0xFFFE, 0xFFFF, 0xFFFF + 1, 3*0xFFFF + 7} {
		data := bytes.Repeat([]byte("vfox"), size/4+1)[:size]
		stream := storedDeflate(data)
		// Four bytes of header for each stored block, and no compression.
		blocks := max(1, (size+0xFFFE)/0xFFFF)
		if len(stream) != size+5*blocks {
			t.Errorf("%d bytes: a stream of %d bytes, want %d", size, len(stream), size+5*blocks)
		}
		inflated, err := io.ReadAll(flate.NewReader(bytes.NewReader(stream)))
		if err != nil || !bytes.Equal(inflated, data) {
			t.Errorf("%d bytes: %d bytes come back, %v", size, len(inflated), err)
		}
	}
}

// The bundle of signed packages: the manifest holds the offsets and the sizes of
// the packages that are in it now, and its block map the manifest that is in it.
func TestRebundleAfterTheSignaturesGrewThePackages(t *testing.T) {
	directory := t.TempDir()
	var arches []string
	for _, arch := range []string{"arm64", "x64"} {
		cfg := testMsixConfig()
		cfg.Architecture = arch
		msix, err := WriteMsix(cfg)
		if err != nil {
			t.Fatal(err)
		}
		// A signature makes the package larger, by a different amount for each.
		padded := append(msix, bytes.Repeat([]byte{byte(len(arch))}, 1000+100*len(arch))...)
		name := filepath.Join(directory, "vfox-plus_0.1.0_"+arch+".msix")
		if err := os.WriteFile(name, padded, 0o644); err != nil {
			t.Fatal(err)
		}
		arches = append(arches, arch)
	}
	path, err := Rebundle(RebundleConfig{Dir: directory, Package: "vfox-plus", Version: "0.1.0", Publisher: "CN=vfox-plus", Architectures: arches})
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(path) != "vfox-plus_0.1.0.msixbundle" {
		t.Errorf("wrote %s", path)
	}
	bundle, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]*zip.File{}
	for _, file := range reader.File {
		files[file.Name] = file
	}
	manifest, err := readZipFile(reader, "AppxMetadata/AppxBundleManifest.xml")
	if err != nil {
		t.Fatal(err)
	}
	// Every package of the manifest is where it says, as large as it says, and
	// is the file of the directory.
	listed := regexp.MustCompile(`FileName="([^"]+)" Offset="(\d+)" Size="(\d+)"`).FindAllStringSubmatch(string(manifest), -1)
	if len(listed) != 2 {
		t.Fatalf("the manifest lists %d packages: %s", len(listed), manifest)
	}
	for _, match := range listed {
		offset, _ := strconv.ParseInt(match[2], 10, 64)
		size, _ := strconv.ParseInt(match[3], 10, 64)
		file, ok := files[match[1]]
		if !ok {
			t.Fatalf("%s is not in the bundle", match[1])
		}
		dataOffset, err := file.DataOffset()
		if err != nil || dataOffset != offset || int64(file.UncompressedSize64) != size {
			t.Errorf("%s: the manifest says %d+%d, the bundle has %d+%d (%v)", match[1], offset, size, dataOffset, file.UncompressedSize64, err)
		}
		want, err := os.ReadFile(filepath.Join(directory, match[1]))
		if err != nil || !bytes.Equal(bundle[offset:offset+size], want) {
			t.Errorf("%s: the bundle holds another file than the directory (%v)", match[1], err)
		}
	}
	if _, err := checkZipLayout(bundle, "rebundled", false); err != nil {
		t.Errorf("the layout of the bundle: %v", err)
	}
}
