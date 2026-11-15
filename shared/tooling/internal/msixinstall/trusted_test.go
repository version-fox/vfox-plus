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

package msixinstall

import (
	"bytes"
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestTrustedInstallPassesWithoutAllowUnsigned(t *testing.T) {
	m := &fakeMachine{packages: [][]Package{{pkg("X64")}, nil}}
	var out bytes.Buffer
	if err := checkTrusted(m, &out, `C:\msix\vfox-plus_1.0.0.msixbundle`, "x64"); err != nil {
		t.Fatalf("checkTrusted: %v", err)
	}
	if want := []string{`C:\msix\vfox-plus_1.0.0.msixbundle`}; !slices.Equal(m.installs, want) {
		t.Errorf("installs = %v, want %v, without -AllowUnsigned", m.installs, want)
	}
	if !slices.Equal(m.removed, []string{testFullName}) {
		t.Errorf("removed = %v, want the package", m.removed)
	}
	if !strings.Contains(out.String(), "signature Developer") || !strings.Contains(out.String(), "uninstalled cleanly") {
		t.Errorf("the output does not show the signed install and its removal:\n%s", out.String())
	}
}

func TestTrustedInstallRefusesAnUnsignedPackage(t *testing.T) {
	unsigned := Package{FullName: testFullName, Architecture: "X64", SignatureKind: "None"}
	m := &fakeMachine{packages: [][]Package{{unsigned}, nil}}
	err := checkTrusted(m, &bytes.Buffer{}, "bundle.msixbundle", "x64")
	if err == nil || !strings.Contains(err.Error(), "has no signature") {
		t.Fatalf("checkTrusted = %v, want the unsigned package refused", err)
	}
	if !slices.Equal(m.removed, []string{testFullName}) {
		t.Errorf("removed = %v, want the package taken out again", m.removed)
	}
}

func TestTrustedInstallFailureShowsTheLog(t *testing.T) {
	m := &fakeMachine{installErr: errors.New("0x800B0109"), deployment: "deployment line"}
	var out bytes.Buffer
	err := checkTrusted(m, &out, "bundle.msixbundle", "x64")
	if err == nil || !strings.Contains(err.Error(), "Add-AppxPackage failed") {
		t.Fatalf("checkTrusted = %v, want the install failure", err)
	}
	if !strings.Contains(out.String(), "deployment line") {
		t.Errorf("the deployment log is not shown:\n%s", out.String())
	}
	if len(m.removed) != 0 {
		t.Errorf("removed %v after a failed install", m.removed)
	}
}
