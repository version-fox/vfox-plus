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
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const (
	testFullName = "dev.vfox.VfoxPlus_1.0.0.0_x64__abc"
	testTool     = `C:\release\vfxtool.exe`
)

// fakeMachine answers the checks from scripted results. A result list that runs
// out repeats its last value, the way a machine that keeps its state does.
type fakeMachine struct {
	installErr   error
	installs     []string
	packages     [][]Package
	removed      []string
	certs        [][]Certificate
	removedCerts []string
	codes        []int
	calls        [][]string
	deployment   string
	packaging    string
	tools        map[string]string
}

func next[T any](queue *[][]T) []T {
	if len(*queue) == 0 {
		return nil
	}
	first := (*queue)[0]
	if len(*queue) > 1 {
		*queue = (*queue)[1:]
	}
	return first
}

func (f *fakeMachine) Install(path string, allowUnsigned bool) error {
	f.installs = append(f.installs, path)
	if allowUnsigned {
		f.installs[len(f.installs)-1] += " unsigned"
	}
	return f.installErr
}

func (f *fakeMachine) Installed() ([]Package, error) { return next(&f.packages), nil }

func (f *fakeMachine) Remove(fullName string) error {
	f.removed = append(f.removed, fullName)
	return nil
}

func (f *fakeMachine) Certificates() ([]Certificate, error) { return next(&f.certs), nil }

func (f *fakeMachine) RemoveCertificate(thumbprint string) error {
	f.removedCerts = append(f.removedCerts, thumbprint)
	return nil
}

func (f *fakeMachine) DeploymentLog() (string, error) { return f.deployment, nil }
func (f *fakeMachine) PackagingLog() (string, error)  { return f.packaging, nil }

func (f *fakeMachine) SDKTool(name string) (string, error) { return f.tools[name], nil }

func (f *fakeMachine) Run(program string, args ...string) (int, error) {
	f.calls = append(f.calls, append([]string{program}, args...))
	if len(f.codes) == 0 {
		return 0, nil
	}
	code := f.codes[0]
	f.codes = f.codes[1:]
	return code, nil
}

func pkg(arch string) Package {
	return Package{FullName: testFullName, Architecture: arch, SignatureKind: "Developer"}
}

func cert(thumbprint string) Certificate {
	return Certificate{Thumbprint: thumbprint, Subject: certSubject}
}

func TestUnsignedInstallPasses(t *testing.T) {
	m := &fakeMachine{packages: [][]Package{{pkg("X64")}, nil}}
	var out bytes.Buffer
	if err := checkUnsigned(m, &out, `C:\msix\vfox-plus_1.0.0.msixbundle`, "x64"); err != nil {
		t.Fatalf("checkUnsigned: %v", err)
	}
	if want := []string{`C:\msix\vfox-plus_1.0.0.msixbundle unsigned`}; !slices.Equal(m.installs, want) {
		t.Errorf("installs = %v, want %v", m.installs, want)
	}
	if !slices.Equal(m.removed, []string{testFullName}) {
		t.Errorf("removed = %v, want the package", m.removed)
	}
	if !strings.Contains(out.String(), "uninstalled cleanly") {
		t.Errorf("output does not say the removal is clean:\n%s", out.String())
	}
}

func TestUnsignedInstallFailureShowsTheLog(t *testing.T) {
	m := &fakeMachine{installErr: errors.New("0x80073CF3"), deployment: "deployment line"}
	var out bytes.Buffer
	err := checkUnsigned(m, &out, "bundle.msixbundle", "x64")
	if err == nil || !strings.Contains(err.Error(), "Add-AppxPackage failed") {
		t.Fatalf("checkUnsigned = %v, want the install failure", err)
	}
	if !strings.Contains(out.String(), "deployment line") {
		t.Errorf("the deployment log is not shown:\n%s", out.String())
	}
	if len(m.removed) != 0 {
		t.Errorf("removed %v after a failed install", m.removed)
	}
}

func TestUnsignedArchitectureMustMatch(t *testing.T) {
	m := &fakeMachine{packages: [][]Package{{pkg("Arm64")}}}
	err := checkUnsigned(m, &bytes.Buffer{}, "bundle.msixbundle", "x64")
	if err == nil || !strings.Contains(err.Error(), "installed arch Arm64, want x64") {
		t.Fatalf("checkUnsigned = %v, want the architecture refused", err)
	}
}

func TestUnsignedNotRegisteredAfterInstall(t *testing.T) {
	m := &fakeMachine{}
	err := checkUnsigned(m, &bytes.Buffer{}, "bundle.msixbundle", "x64")
	if err == nil || !strings.Contains(err.Error(), "package not registered after install") {
		t.Fatalf("checkUnsigned = %v, want the package not registered", err)
	}
}

func TestUnsignedRemovalMustLeaveNothing(t *testing.T) {
	m := &fakeMachine{packages: [][]Package{{pkg("X64")}}}
	err := checkUnsigned(m, &bytes.Buffer{}, "bundle.msixbundle", "x64")
	if err == nil || !strings.Contains(err.Error(), "package still registered after removal") {
		t.Fatalf("checkUnsigned = %v, want the package still registered", err)
	}
}

func TestSignedInstallPasses(t *testing.T) {
	m := &fakeMachine{
		// Before the first run, after it, and after the second run.
		certs:    [][]Certificate{nil, {cert("AAA")}, {cert("BBB")}},
		packages: [][]Package{{pkg("X64")}, nil},
	}
	var out bytes.Buffer
	if err := checkSigned(m, &out, testTool, "msix", "x64"); err != nil {
		t.Fatalf("checkSigned: %v", err)
	}
	want := [][]string{
		{testTool, "sign-msix", "--directory", "msix", "--no-relaunch"},
		{testTool, "sign-msix", "--directory", "msix", "--no-relaunch", "--skip-install"},
	}
	if len(m.calls) != len(want) {
		t.Fatalf("calls = %v, want %v", m.calls, want)
	}
	for i := range want {
		if !slices.Equal(m.calls[i], want[i]) {
			t.Errorf("call %d = %v, want %v", i, m.calls[i], want[i])
		}
	}
	if !slices.Equal(m.removedCerts, []string{"BBB"}) {
		t.Errorf("removed certificates = %v, want the one of the second run", m.removedCerts)
	}
	if !slices.Equal(m.removed, []string{testFullName}) {
		t.Errorf("removed packages = %v, want the package", m.removed)
	}
	if !strings.Contains(out.String(), "signed, installed and uninstalled cleanly") {
		t.Errorf("output does not say the check passed:\n%s", out.String())
	}
}

func TestSignedRefusesARunnerThatTrustsTheCertificate(t *testing.T) {
	m := &fakeMachine{certs: [][]Certificate{{cert("AAA")}}}
	err := checkSigned(m, &bytes.Buffer{}, testTool, "msix", "x64")
	if err == nil || !strings.Contains(err.Error(), "the runner trusts CN=vfox-plus already") {
		t.Fatalf("checkSigned = %v, want the runner refused", err)
	}
	if len(m.calls) != 0 {
		t.Errorf("sign-msix ran on a runner that trusts the certificate: %v", m.calls)
	}
}

func TestSignedSecondRunMustReplaceTheCertificate(t *testing.T) {
	m := &fakeMachine{
		certs:    [][]Certificate{nil, {cert("AAA")}, {cert("BBB"), cert("CCC")}},
		packages: [][]Package{{pkg("X64")}, nil},
	}
	err := checkSigned(m, &bytes.Buffer{}, testTool, "msix", "x64")
	if err == nil || !strings.Contains(err.Error(), "want one trusted CN=vfox-plus after two runs, got 2") {
		t.Fatalf("checkSigned = %v, want two certificates refused", err)
	}
}

func TestSignedSecondRunFailure(t *testing.T) {
	m := &fakeMachine{
		codes:    []int{0, 1},
		certs:    [][]Certificate{nil, {cert("AAA")}},
		packages: [][]Package{{pkg("X64")}, nil},
	}
	err := checkSigned(m, &bytes.Buffer{}, testTool, "msix", "x64")
	if err == nil || !strings.Contains(err.Error(), "the second sign-msix failed with exit code 1") {
		t.Fatalf("checkSigned = %v, want the second run refused", err)
	}
}

func TestSignedSignFailureShowsTheDiagnosis(t *testing.T) {
	dir := t.TempDir()
	signed := filepath.Join(dir, "signed")
	if err := os.MkdirAll(signed, 0o755); err != nil {
		t.Fatal(err)
	}
	alone := filepath.Join(signed, "vfox-plus_1.0.0_x64.msix")
	if err := os.WriteFile(alone, bytes.Repeat([]byte{0xAB}, 120), 0o644); err != nil {
		t.Fatal(err)
	}
	m := &fakeMachine{
		codes:      []int{1},
		certs:      [][]Certificate{nil},
		deployment: "deployment line",
		packaging:  "packaging line",
		tools:      map[string]string{"signtool.exe": `C:\sdk\x64\signtool.exe`},
	}
	var out bytes.Buffer
	err := checkSigned(m, &out, testTool, dir, "x64")
	if err == nil || !strings.Contains(err.Error(), "sign-msix failed with exit code 1") {
		t.Fatalf("checkSigned = %v, want the sign-msix failure", err)
	}
	for _, want := range []string{
		"--- AppX deployment log (tail) ---", "deployment line",
		"--- AppxPackaging log (tail) ---", "packaging line",
		"vfox-plus_1.0.0_x64.msix installs",
		"--- the last 98 bytes of vfox-plus_1.0.0_x64.msix ---",
		"--- makeappx.exe is not on this runner ---",
	} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("the diagnosis lacks %q:\n%s", want, out.String())
		}
	}
	verify := []string{`C:\sdk\x64\signtool.exe`, "verify", "/pa", "/v", alone}
	if !slices.ContainsFunc(m.calls, func(call []string) bool { return slices.Equal(call, verify) }) {
		t.Errorf("signtool did not verify the signed package: %v", m.calls)
	}
}

func TestLastBytesAreShownAsHex(t *testing.T) {
	path := filepath.Join(t.TempDir(), "p.msix")
	if err := os.WriteFile(path, []byte{0x01, 0xAB, 0xFF}, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := lastBytes(path, 2)
	if err != nil || got != "ab ff" {
		t.Errorf("lastBytes = %q, %v; want \"ab ff\"", got, err)
	}
}
