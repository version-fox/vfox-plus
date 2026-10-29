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
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testExeData() []byte {
	return append([]byte{'M', 'Z'}, bytes.Repeat([]byte{0x90}, 100000)...)
}

func testValidateConfig(arch string) MsixConfig {
	return MsixConfig{
		Identity:             "dev.vfox.VfoxPlus",
		Publisher:            defaultPublisher,
		Version:              "0.1.0.0",
		Architecture:         arch,
		DisplayName:          "vfox+",
		PublisherDisplayName: "Han Li and contributors",
		Description:          "Install and switch SDK versions with vfox",
		Executable:           "vfox-plus.exe",
		ExecutableData:       testExeData(),
		MTime:                time.Unix(1234567890, 0).UTC(),
	}
}

func writeValidateFixtures(t *testing.T) string {
	t.Helper()
	x64, err := WriteMsix(testValidateConfig("x64"))
	if err != nil {
		t.Fatal(err)
	}
	arm64, err := WriteMsix(testValidateConfig("arm64"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "vfox-plus_0.1.0_x64.msix"), x64, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "vfox-plus_0.1.0_arm64.msix"), arm64, 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, name, err := WriteBundle(BundleConfig{
		Package:   "vfox-plus",
		Version:   "0.1.0",
		Identity:  "dev.vfox.VfoxPlus",
		Publisher: defaultPublisher,
		Packages: []BundlePackage{
			{Architecture: "x64", Filename: "vfox-plus_0.1.0_x64.msix", Data: x64},
			{Architecture: "arm64", Filename: "vfox-plus_0.1.0_arm64.msix", Data: arm64},
		},
		MTime: time.Unix(1234567890, 0).UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if name != "vfox-plus_0.1.0.msixbundle" {
		t.Fatalf("bundle name = %q", name)
	}
	if err := os.WriteFile(filepath.Join(dir, name), bundle, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestValidateDirOK(t *testing.T) {
	if err := ValidateDir(writeValidateFixtures(t)); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDirEmpty(t *testing.T) {
	if err := ValidateDir(t.TempDir()); err == nil {
		t.Error("expected an error for a directory without packages")
	}
}

func TestValidateMsixMissingBlockMap(t *testing.T) {
	manifest := RenderAppxManifest(testValidateConfig("x64"))
	entries := []zipEntry{
		{name: "AppxManifest.xml", data: []byte(manifest)},
		{name: "[Content_Types].xml", data: []byte(RenderContentTypes()), deflated: true},
	}
	data, err := writeZip(entries, time.Unix(1234567890, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateMsix(data, "broken.msix"); err == nil {
		t.Error("expected an error for a package without a block map")
	} else if !strings.Contains(err.Error(), "AppxBlockMap.xml") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestValidateBundleBadOffset(t *testing.T) {
	x64, err := WriteMsix(testValidateConfig("x64"))
	if err != nil {
		t.Fatal(err)
	}
	arm64, err := WriteMsix(testValidateConfig("arm64"))
	if err != nil {
		t.Fatal(err)
	}
	sorted := []BundlePackage{
		{Architecture: "arm64", Filename: "vfox-plus_0.1.0_arm64.msix", Data: arm64},
		{Architecture: "x64", Filename: "vfox-plus_0.1.0_x64.msix", Data: x64},
	}
	// A manifest that lies about the offsets, inside an otherwise valid ZIP,
	// so that only the offset check can fail.
	lying := RenderBundleManifest("dev.vfox.VfoxPlus", defaultPublisher, "0.1.0.0", []BundleEntry{
		{Architecture: "arm64", Filename: "vfox-plus_0.1.0_arm64.msix", Size: int64(len(arm64))},
		{Architecture: "x64", Filename: "vfox-plus_0.1.0_x64.msix", Size: int64(len(x64))},
	})
	data, err := assembleBundle(sorted, lying, RenderBundleContentTypes(sorted), RenderBlockMap(nil), time.Unix(1234567890, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBundle(data, "lying.msixbundle"); err == nil {
		t.Error("expected an error for a wrong bundle offset")
	} else if !strings.Contains(err.Error(), "offset") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestValidateMsixVersionMismatch(t *testing.T) {
	msix, err := WriteMsix(testValidateConfig("x64"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateMsix(msix, "vfox-plus_9.9.9_x64.msix"); err == nil {
		t.Error("expected an error for a version mismatch")
	} else if !strings.Contains(err.Error(), "0.1.0.0") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestValidateMsixRejectsTopLevelDescription(t *testing.T) {
	manifest := RenderAppxManifest(testValidateConfig("x64"))
	if err := checkManifestOrder([]byte(manifest), "good.msix"); err != nil {
		t.Fatal(err)
	}
	// Reintroduce the old bug: Description as a direct child of Package,
	// which Windows rejects with 0x80080204.
	line := "    <Description>Install and switch SDK versions with vfox</Description>\n"
	if !strings.Contains(manifest, line) {
		t.Fatalf("fixture manifest changed:\n%s", manifest)
	}
	bad := strings.Replace(manifest, line, "", 1)
	bad = strings.Replace(bad, "  </Properties>\n", "  </Properties>\n  <Description>Install and switch SDK versions with vfox</Description>\n", 1)
	if err := checkManifestOrder([]byte(bad), "bad.msix"); err == nil {
		t.Error("expected an error for a top-level Description")
	} else if !strings.Contains(err.Error(), "unexpected Package element") {
		t.Errorf("wrong error: %v", err)
	}
}

func TestRenderAppxManifestVisualElements(t *testing.T) {
	manifest := RenderAppxManifest(testValidateConfig("x64"))
	// Wide310x150Logo is not a VisualElements attribute: it belongs on the
	// uap:DefaultTile child, and Windows rejects it elsewhere (0xC00CE015).
	for _, line := range strings.Split(manifest, "\n") {
		if strings.Contains(line, "<uap:VisualElements") && strings.Contains(line, "Wide310x150Logo") {
			t.Errorf("Wide310x150Logo on VisualElements:\n%s", line)
		}
	}
	if !strings.Contains(manifest, `<uap:DefaultTile Wide310x150Logo="Assets\Wide310x150Logo.png"/>`) {
		t.Errorf("DefaultTile with the wide logo missing:\n%s", manifest)
	}
}

func TestWriteBundleBlockMapListsOnlyManifest(t *testing.T) {
	x64, err := WriteMsix(testValidateConfig("x64"))
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := WriteBundle(BundleConfig{
		Package:   "vfox-plus",
		Version:   "0.1.0",
		Identity:  "dev.vfox.VfoxPlus",
		Publisher: defaultPublisher,
		Packages: []BundlePackage{
			{Architecture: "x64", Filename: "vfox-plus_0.1.0_x64.msix", Data: x64},
		},
		MTime: time.Unix(1234567890, 0).UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	files, err := checkZipLayout(bundle, "test.msixbundle", false)
	if err != nil {
		t.Fatal(err)
	}
	var doc blockMapDoc
	if err := xml.Unmarshal(files["AppxBlockMap.xml"], &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Files) != 1 || doc.Files[0].Name != `AppxMetadata\AppxBundleManifest.xml` {
		names := make([]string, 0, len(doc.Files))
		for _, f := range doc.Files {
			names = append(names, f.Name)
		}
		t.Fatalf("bundle block map lists %q, want only the manifest", names)
	}
}

func TestValidateBundleRejectsPayloadBlockMapEntries(t *testing.T) {
	x64, err := WriteMsix(testValidateConfig("x64"))
	if err != nil {
		t.Fatal(err)
	}
	mtime := time.Unix(1234567890, 0).UTC()
	sorted := []BundlePackage{{Architecture: "x64", Filename: "vfox-plus_0.1.0_x64.msix", Data: x64}}
	contentTypes := RenderBundleContentTypes(sorted)
	probe, err := assembleBundle(sorted,
		RenderBundleManifest("dev.vfox.VfoxPlus", defaultPublisher, "0.1.0.0",
			[]BundleEntry{{Architecture: "x64", Filename: sorted[0].Filename}}),
		contentTypes, RenderBlockMap(nil), mtime)
	if err != nil {
		t.Fatal(err)
	}
	probeHeaders, err := parseLocalHeaders(probe)
	if err != nil {
		t.Fatal(err)
	}
	manifest := RenderBundleManifest("dev.vfox.VfoxPlus", defaultPublisher, "0.1.0.0", []BundleEntry{{
		Architecture: "x64",
		Filename:     sorted[0].Filename,
		Offset:       probeHeaders[sorted[0].Filename].dataOffset,
		Size:         int64(len(x64)),
	}})
	manifestInfo := probeHeaders["AppxMetadata/AppxBundleManifest.xml"]
	payloadInfo := probeHeaders[sorted[0].Filename]
	blockMap := RenderBlockMap([]BlockMapFile{{
		Name:    "AppxMetadata/AppxBundleManifest.xml",
		Size:    int64(len(manifest)),
		LfhSize: manifestInfo.headerSize,
		Blocks:  blockHashes([]byte(manifest)),
	}, {
		Name:    sorted[0].Filename,
		Size:    int64(len(x64)),
		LfhSize: payloadInfo.headerSize,
		Blocks:  blockHashes(x64),
	}})
	data, err := assembleBundle(sorted, manifest, contentTypes, blockMap, mtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBundle(data, "bad.msixbundle"); err == nil {
		t.Fatal("expected an error for payload entries in the bundle block map")
	} else if !strings.Contains(err.Error(), "must list only") {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestRenderBundleContentTypesDeclaresPayloadsAsAppx(t *testing.T) {
	got := RenderBundleContentTypes([]BundlePackage{
		{Architecture: "x64", Filename: "vfox-plus_0.1.0_x64.msix"},
		{Architecture: "arm64", Filename: "vfox-plus_0.1.0_arm64.msix"},
	})
	if !strings.Contains(got, `<Default Extension="msix" ContentType="application/vnd.ms-appx"/>`) {
		t.Errorf("payloads are not declared as MSIX packages:\n%s", got)
	}
	if strings.Contains(got, "vnd.ms-appx.msix") {
		t.Errorf("invented payload content type:\n%s", got)
	}
	if !strings.Contains(got, `<Default Extension="xml" ContentType="application/vnd.ms-appx.bundlemanifest+xml"/>`) {
		t.Errorf("bundle manifest content type missing:\n%s", got)
	}
}

func TestValidateBundleRejectsWrongPayloadContentType(t *testing.T) {
	x64, err := WriteMsix(testValidateConfig("x64"))
	if err != nil {
		t.Fatal(err)
	}
	mtime := time.Unix(1234567890, 0).UTC()
	sorted := []BundlePackage{{Architecture: "x64", Filename: "vfox-plus_0.1.0_x64.msix", Data: x64}}
	probe, err := assembleBundle(sorted,
		RenderBundleManifest("dev.vfox.VfoxPlus", defaultPublisher, "0.1.0.0",
			[]BundleEntry{{Architecture: "x64", Filename: sorted[0].Filename}}),
		RenderBundleContentTypes(sorted), RenderBlockMap(nil), mtime)
	if err != nil {
		t.Fatal(err)
	}
	headers, err := parseLocalHeaders(probe)
	if err != nil {
		t.Fatal(err)
	}
	manifest := RenderBundleManifest("dev.vfox.VfoxPlus", defaultPublisher, "0.1.0.0", []BundleEntry{{
		Architecture: "x64",
		Filename:     sorted[0].Filename,
		Offset:       headers[sorted[0].Filename].dataOffset,
		Size:         int64(len(x64)),
	}})
	info := headers["AppxMetadata/AppxBundleManifest.xml"]
	blockMap := RenderBlockMap([]BlockMapFile{{
		Name:    "AppxMetadata/AppxBundleManifest.xml",
		Size:    int64(len(manifest)),
		LfhSize: info.headerSize,
		Blocks:  blockHashes([]byte(manifest)),
	}})
	// Declare the payload with the invented content type instead of the
	// Default rule MakeAppx writes.
	contentTypes := `<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="xml" ContentType="application/vnd.ms-appx.bundlemanifest+xml"/>
  <Override PartName="/` + sorted[0].Filename + `" ContentType="application/vnd.ms-appx.msix"/>
  <Override PartName="/AppxBlockMap.xml" ContentType="application/vnd.ms-appx.blockmap+xml"/>
</Types>
`
	data, err := assembleBundle(sorted, manifest, contentTypes, blockMap, mtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateBundle(data, "bad.msixbundle"); err == nil {
		t.Fatal("expected an error for a payload with a wrong content type")
	} else if !strings.Contains(err.Error(), "want \"application/vnd.ms-appx\"") {
		t.Fatalf("wrong error: %v", err)
	}
}

func TestWriteMsixBlockMapExcludesContentTypes(t *testing.T) {
	msix, err := WriteMsix(testValidateConfig("x64"))
	if err != nil {
		t.Fatal(err)
	}
	files, err := checkZipLayout(msix, "test.msix", true)
	if err != nil {
		t.Fatal(err)
	}
	var doc blockMapDoc
	if err := xml.Unmarshal(files["AppxBlockMap.xml"], &doc); err != nil {
		t.Fatal(err)
	}
	listed := map[string]bool{}
	for _, entry := range doc.Files {
		listed[strings.ReplaceAll(entry.Name, "\\", "/")] = true
	}
	// [Content_Types].xml must not be in the block map: Windows rejects that
	// with APPX_E_BLOCK_MAP_INVALID.
	if listed["[Content_Types].xml"] {
		t.Error("the package block map lists [Content_Types].xml")
	}
	for name := range files {
		if name == "AppxBlockMap.xml" || name == "[Content_Types].xml" {
			continue
		}
		if !listed[name] {
			t.Errorf("the package block map misses %s", name)
		}
	}
}

func TestValidateMsixRejectsContentTypesInBlockMap(t *testing.T) {
	manifest := RenderAppxManifest(testValidateConfig("x64"))
	contentTypes := RenderContentTypes()
	mtime := time.Unix(1234567890, 0).UTC()
	entries := []zipEntry{
		{name: "AppxManifest.xml", data: []byte(manifest)},
		{name: "[Content_Types].xml", data: []byte(contentTypes)},
		{name: "vfox-plus.exe", data: testExeData()},
	}
	probe, err := writeZip(append(append([]zipEntry(nil), entries...),
		zipEntry{name: "AppxBlockMap.xml", data: []byte(RenderBlockMap(nil))}), mtime)
	if err != nil {
		t.Fatal(err)
	}
	headers, err := parseLocalHeaders(probe)
	if err != nil {
		t.Fatal(err)
	}
	var blockFiles []BlockMapFile
	for _, name := range []string{"AppxManifest.xml", "[Content_Types].xml", "vfox-plus.exe"} {
		data := map[string][]byte{
			"AppxManifest.xml":    []byte(manifest),
			"[Content_Types].xml": []byte(contentTypes),
			"vfox-plus.exe":       testExeData(),
		}[name]
		blockFiles = append(blockFiles, BlockMapFile{
			Name: name, Size: int64(len(data)), LfhSize: headers[name].headerSize, Blocks: blockHashes(data),
		})
	}
	data, err := writeZip(append(append([]zipEntry(nil), entries...),
		zipEntry{name: "AppxBlockMap.xml", data: []byte(RenderBlockMap(blockFiles))}), mtime)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateMsix(data, "bad.msix"); err == nil {
		t.Fatal("expected an error for [Content_Types].xml in the block map")
	} else if !strings.Contains(err.Error(), "[Content_Types].xml") {
		t.Fatalf("wrong error: %v", err)
	}
}

// The content types are the part that osslsigncode rewrites when it signs, and
// it only handles a deflated one: a package that stores them is refused here, as
// Windows refuses what osslsigncode makes of it.
func TestValidateMsixRefusesStoredContentTypes(t *testing.T) {
	manifest := RenderAppxManifest(testValidateConfig("x64"))
	entries := []zipEntry{
		{name: "AppxBlockMap.xml", data: []byte(RenderBlockMap(nil))},
		{name: "AppxManifest.xml", data: []byte(manifest)},
		{name: "[Content_Types].xml", data: []byte(RenderContentTypes())},
	}
	data, err := writeZip(entries, time.Unix(1234567890, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateMsix(data, "stored.msix"); err == nil || !strings.Contains(err.Error(), "[Content_Types].xml") || !strings.Contains(err.Error(), "Deflate") {
		t.Errorf("got %v", err)
	}
}
