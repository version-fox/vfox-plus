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

// Package msixinstall implements `vfxtool install-check`, which checks on Windows
// that the packages of a release install, register and come off again.
//
// The unsigned bundle is installed with -AllowUnsigned, as the release makes it.
// The signed packages are made the way a user makes them, with sign-msix, which
// trusts a certificate of this machine and installs the bundle without
// -AllowUnsigned. A second run must replace that certificate, not add another, and
// the certificates are taken out again at the end.
//
// Windows installs packages with the Appx cmdlets of Windows PowerShell, and keeps
// the certificates of the machine in its stores, so the checks reach them through
// the Machine interface. The checks themselves are plain Go and are tested with a
// fake machine.
//
// The runners of GitHub have UAC off, so the installs for all users run without a
// prompt. sign-msix runs with -no-relaunch: it fails instead of waiting for a prompt
// that nobody answers.
package msixinstall

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// The package and the certificate of the application.
const (
	packageName = "dev.vfox.VfoxPlus"
	certSubject = "CN=vfox-plus"
)

// Package is a registered package, with the fields that Windows names.
type Package struct {
	FullName      string `json:"PackageFullName"`
	Architecture  string `json:"Architecture"`
	SignatureKind string `json:"SignatureKind"`
}

// Certificate is a certificate of the TrustedPeople store of the machine.
type Certificate struct {
	Thumbprint string `json:"Thumbprint"`
	Subject    string `json:"Subject"`
}

// Machine is the Windows that the checks run on.
type Machine interface {
	// Install adds the package or bundle at path. With allowUnsigned, it needs no signature.
	Install(path string, allowUnsigned bool) error
	// Installed lists the registered packages of the application.
	Installed() ([]Package, error)
	// Remove removes the package with the full name.
	Remove(fullName string) error
	// Certificates lists the certificates of the application in the TrustedPeople store.
	Certificates() ([]Certificate, error)
	// RemoveCertificate removes the certificate with the thumbprint from the TrustedPeople store.
	RemoveCertificate(thumbprint string) error
	// DeploymentLog returns the last lines of the AppX deployment log.
	DeploymentLog() (string, error)
	// PackagingLog returns the last events of the AppxPackaging log.
	PackagingLog() (string, error)
	// SDKTool returns the path of the last tool of that name for x64 or arm64 in the
	// Windows SDK, or "" when the SDK has none.
	SDKTool(name string) (string, error)
	// Run runs a program with its arguments, shows its output, and returns its exit code.
	Run(program string, args ...string) (int, error)
}

// Run is the command `vfxtool install-check`:
//
//	install-check [--directory DIR] --arch x64|arm64 [--signed]
func Run(args []string) error {
	flags := flag.NewFlagSet("install-check", flag.ContinueOnError)
	directory := flags.String("directory", "msix", "the directory of the release: the bundle, the packages, vfxtool.exe and the stage directories")
	arch := flags.String("arch", "", "the architecture of this machine: x64 or arm64")
	signed := flags.Bool("signed", false, "check the signed packages that sign-msix makes, instead of the unsigned bundle")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *arch != "x64" && *arch != "arm64" {
		return fmt.Errorf("--arch must be x64 or arm64, not %q", *arch)
	}
	machine := windowsMachine{}
	if *signed {
		tool, err := os.Executable()
		if err != nil {
			return err
		}
		return checkSigned(machine, os.Stdout, tool, *directory, *arch)
	}
	bundles, err := matches(*directory, "*.msixbundle")
	if err != nil {
		return err
	}
	if len(bundles) != 1 {
		return fmt.Errorf("want one .msixbundle in %s, got %d", *directory, len(bundles))
	}
	bundle, err := filepath.Abs(bundles[0])
	if err != nil {
		return err
	}
	return checkUnsigned(machine, os.Stdout, bundle, *arch)
}

// checkUnsigned installs the unsigned bundle, checks the one package that Windows
// registers, and removes it again.
func checkUnsigned(m Machine, out io.Writer, bundle, arch string) error {
	if err := m.Install(bundle, true); err != nil {
		fmt.Fprintln(out, "--- AppX deployment log (tail) ---")
		showLog(out, m.DeploymentLog)
		return fmt.Errorf("Add-AppxPackage failed: %w", err)
	}
	pkg, err := registered(m, arch, "install")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "installed %s arch %s\n", pkg.FullName, pkg.Architecture)
	if err := remove(m, pkg); err != nil {
		return err
	}
	fmt.Fprintln(out, "uninstalled cleanly")
	return nil
}

// checkSigned makes the signed packages with sign-msix, as a user does, and checks
// that Windows installs them and trusts their certificate. A second run must replace
// that certificate. The certificates are taken out again at the end.
func checkSigned(m Machine, out io.Writer, tool, dir, arch string) error {
	trusted, err := m.Certificates()
	if err != nil {
		return err
	}
	if len(trusted) != 0 {
		return fmt.Errorf("the runner trusts %s already", certSubject)
	}

	code, err := m.Run(tool, "sign-msix", "--directory", dir, "--no-relaunch")
	if err != nil {
		return err
	}
	if code != 0 {
		diagnose(m, out, dir, arch)
		return fmt.Errorf("sign-msix failed with exit code %d", code)
	}
	pkg, err := registered(m, arch, "the signed install")
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "installed %s arch %s, signature %s\n", pkg.FullName, pkg.Architecture, pkg.SignatureKind)
	trusted, err = m.Certificates()
	if err != nil {
		return err
	}
	if len(trusted) != 1 {
		return fmt.Errorf("want one trusted %s, got %d", certSubject, len(trusted))
	}
	if err := remove(m, pkg); err != nil {
		return err
	}

	code, err = m.Run(tool, "sign-msix", "--directory", dir, "--no-relaunch", "--skip-install")
	if err != nil {
		return err
	}
	if code != 0 {
		return fmt.Errorf("the second sign-msix failed with exit code %d", code)
	}
	trusted, err = m.Certificates()
	if err != nil {
		return err
	}
	if len(trusted) != 1 {
		return fmt.Errorf("want one trusted %s after two runs, got %d", certSubject, len(trusted))
	}
	for _, cert := range trusted {
		if err := m.RemoveCertificate(cert.Thumbprint); err != nil {
			return err
		}
	}
	fmt.Fprintln(out, "signed, installed and uninstalled cleanly")
	return nil
}

// registered returns the one package that Windows registers for the application,
// after the install named by when, and checks its architecture.
func registered(m Machine, arch, when string) (Package, error) {
	pkgs, err := m.Installed()
	if err != nil {
		return Package{}, err
	}
	switch len(pkgs) {
	case 0:
		return Package{}, fmt.Errorf("package not registered after %s", when)
	case 1:
	default:
		return Package{}, fmt.Errorf("want one registered package, got %d", len(pkgs))
	}
	if !strings.EqualFold(pkgs[0].Architecture, arch) {
		return Package{}, fmt.Errorf("installed arch %s, want %s", pkgs[0].Architecture, arch)
	}
	return pkgs[0], nil
}

// remove removes the package, and checks that Windows no longer registers the application.
func remove(m Machine, pkg Package) error {
	if err := m.Remove(pkg.FullName); err != nil {
		return err
	}
	left, err := m.Installed()
	if err != nil {
		return err
	}
	if len(left) != 0 {
		return errors.New("package still registered after removal")
	}
	return nil
}

// showLog prints the log that the machine returns, or why it cannot be read.
func showLog(out io.Writer, read func() (string, error)) {
	text, err := read()
	if err != nil {
		fmt.Fprintf(out, "(cannot read the log: %v)\n", err)
		return
	}
	fmt.Fprintln(out, strings.TrimRight(text, "\r\n"))
}

// matches returns the files of dir that match pattern, sorted by name.
func matches(dir, pattern string) ([]string, error) {
	return filepath.Glob(filepath.Join(dir, pattern))
}
