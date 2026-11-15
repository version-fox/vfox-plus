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
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// signedFixture returns the package of cfg with the part that signing adds to it.
func signedFixture(t *testing.T, cfg MsixConfig) []byte {
	t.Helper()
	msix, err := WriteMsix(cfg)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(msix), int64(len(msix)))
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, file := range reader.File {
		entry, err := writer.Create(file.Name)
		if err != nil {
			t.Fatal(err)
		}
		source, err := file.Open()
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.Copy(entry, source); err != nil {
			t.Fatal(err)
		}
		if err := source.Close(); err != nil {
			t.Fatal(err)
		}
	}
	signature, err := writer.Create(signatureEntry)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signature.Write([]byte("signature")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

// writeRebundleDir writes the x64 and arm64 packages into a new directory, signed
// or not, with the publisher of a certificate.
func writeRebundleDir(t *testing.T, signed bool) string {
	t.Helper()
	directory := t.TempDir()
	for _, arch := range []string{"x64", "arm64"} {
		cfg := testMsixConfig()
		cfg.Architecture = arch
		cfg.Publisher = "CN=Test Publisher"
		data, err := WriteMsix(cfg)
		if signed {
			data = signedFixture(t, cfg)
		}
		if err != nil {
			t.Fatal(err)
		}
		name := filepath.Join(directory, "vfox-plus_0.1.0_"+arch+".msix")
		if err := os.WriteFile(name, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

func TestRunRebundleRefusesUnsignedPackages(t *testing.T) {
	directory := writeRebundleDir(t, false)
	err := RunRebundle([]string{"--package", "vfox-plus", "--version", "0.1.0", "--publisher", "CN=Test Publisher", directory})
	if err == nil || !strings.Contains(err.Error(), "is not signed") {
		t.Fatalf("RunRebundle = %v, want the unsigned packages refused", err)
	}
}

func TestRunRebundleBuildsTheBundleOfSignedPackages(t *testing.T) {
	directory := writeRebundleDir(t, true)
	args := []string{"--package", "vfox-plus", "--version", "0.1.0", "--publisher", "CN=Test Publisher", directory}
	if err := RunRebundle(args); err != nil {
		t.Fatalf("RunRebundle: %v", err)
	}
	bundle, err := os.ReadFile(filepath.Join(directory, "vfox-plus_0.1.0.msixbundle"))
	if err != nil {
		t.Fatal(err)
	}
	files := readZip(t, bundle)
	for _, want := range []string{"vfox-plus_0.1.0_x64.msix", "vfox-plus_0.1.0_arm64.msix"} {
		if _, ok := files[want]; !ok {
			t.Errorf("the bundle misses %s", want)
		}
	}
	if manifest := string(files["AppxMetadata/AppxBundleManifest.xml"]); !strings.Contains(manifest, `Publisher="CN=Test Publisher"`) {
		t.Errorf("the bundle manifest has another publisher: %s", manifest)
	}
}

func TestRunRebundleNeedsItsArguments(t *testing.T) {
	directory := writeRebundleDir(t, true)
	if err := RunRebundle([]string{"--package", "vfox-plus", "--version", "0.1.0", directory}); err == nil {
		t.Error("RunRebundle without --publisher = nil error")
	}
	err := RunRebundle([]string{"--package", "vfox-plus", "--version", "0.1.0", "--publisher", "CN=Test Publisher", t.TempDir()})
	if err == nil || !strings.Contains(err.Error(), "no vfox-plus_0.1.0_<arch>.msix") {
		t.Errorf("RunRebundle in an empty directory = %v", err)
	}
}
