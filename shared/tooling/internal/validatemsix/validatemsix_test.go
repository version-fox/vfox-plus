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

package validatemsix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/buildmsix"
)

func TestCheckSelfSignInputs(t *testing.T) {
	dir := t.TempDir()
	err := checkSelfSignInputs(dir)
	if err == nil || !strings.Contains(err.Error(), "missing self-sign input: stage-x64/vfox-plus.exe") {
		t.Fatalf("checkSelfSignInputs on an empty directory = %v", err)
	}
	for _, rel := range selfSignInputs {
		path := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := checkSelfSignInputs(dir); err != nil {
		t.Fatalf("checkSelfSignInputs with every input: %v", err)
	}
}

// writeCertificateDir writes the packages of both architectures and their bundle
// into a new directory, with the publisher that a release gives them.
func writeCertificateDir(t *testing.T, publisher string) string {
	t.Helper()
	dir := t.TempDir()
	exe := append([]byte("MZ"), make([]byte, 4096)...)
	mtime := time.Unix(1234567890, 0).UTC()
	var packages []buildmsix.BundlePackage
	for _, arch := range []string{"x64", "arm64"} {
		msix, err := buildmsix.WriteMsix(buildmsix.MsixConfig{
			Identity:             buildmsix.DefaultIdentity,
			Publisher:            publisher,
			Version:              "0.1.0.0",
			Architecture:         arch,
			DisplayName:          "vfox+",
			PublisherDisplayName: "Test",
			Description:          "Test",
			Executable:           "vfox-plus.exe",
			ExecutableData:       exe,
			MTime:                mtime,
		})
		if err != nil {
			t.Fatal(err)
		}
		name := "vfox-plus_0.1.0_" + arch + ".msix"
		if err := os.WriteFile(filepath.Join(dir, name), msix, 0o644); err != nil {
			t.Fatal(err)
		}
		packages = append(packages, buildmsix.BundlePackage{Architecture: arch, Filename: name, Data: msix})
	}
	bundle, name, err := buildmsix.WriteBundle(buildmsix.BundleConfig{
		Package:   "vfox-plus",
		Version:   "0.1.0",
		Identity:  buildmsix.DefaultIdentity,
		Publisher: publisher,
		Packages:  packages,
		MTime:     mtime,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), bundle, 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRunChecksThePublisherOfTheCertificate(t *testing.T) {
	const subject = "CN=Test Publisher, O=Test, C=US"
	dir := writeCertificateDir(t, subject)
	if err := Run([]string{"--publisher", subject, dir}); err != nil {
		t.Fatalf("Run with the publisher: %v", err)
	}
	if err := Run([]string{"--publisher", "CN=Other", dir}); err == nil {
		t.Error("Run with another publisher = nil error")
	}
	// Without a publisher the packages must be the unsigned ones.
	if err := Run([]string{dir}); err == nil {
		t.Error("Run without --publisher accepted the packages of a certificate")
	}
}

func TestRunNeedsOneDirectory(t *testing.T) {
	if err := Run(nil); err == nil || !strings.Contains(err.Error(), "Usage: validate-msix") {
		t.Errorf("Run without a directory = %v", err)
	}
	if err := Run([]string{"--unknown", "dir"}); err == nil || !strings.Contains(err.Error(), "Usage: validate-msix") {
		t.Errorf("Run with an unknown flag = %v", err)
	}
}
