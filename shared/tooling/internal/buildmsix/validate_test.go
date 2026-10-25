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
		{name: "AppxManifest.xml", data: []byte(manifest), mode: 0o644},
		{name: "[Content_Types].xml", data: []byte(RenderContentTypes()), mode: 0o644},
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
