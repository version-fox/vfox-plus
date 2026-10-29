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

// Package signmsix implements `vfxtool sign-msix`: one command that signs the
// vfox+ MSIX packages with a certificate of its own and installs the bundle on
// Windows 11, with no certificate to buy.
//
// It rebuilds the .msix and .msixbundle files with the publisher
// CN=<publisher> (a signed package carries no unsigned-package OID, and its
// publisher must equal the subject of the certificate), makes a self-signed
// code-signing certificate for that subject, signs the packages with
// osslsigncode (downloaded, pinned and checked by its SHA-256, so that no
// Windows SDK is needed), puts the signed packages into the bundle again and
// signs the bundle (the packages in a bundle need their own signatures: the
// signature of the bundle covers its manifest only), trusts the certificate for
// the local machine and installs the bundle. The key of the certificate lives for this run only: it
// is written to a temporary file for osslsigncode, deleted after, and never
// kept, so the certificate that stays in the TrustedPeople store of the machine
// can sign nothing any more. A later run replaces the certificate that an
// earlier run added instead of adding another.
//
// Trusting a certificate and installing a package need administrator rights;
// the command relaunches itself elevated, in a window of its own that waits
// for Enter at the end. Off Windows, only the signing works (pass
// --skip-install and --osslsigncode).
//
// It replaces shared/packaging/windows/Sign-Msix.ps1 and runs from the
// extracted windows artifact of CI, which holds stage-x64/, stage-arm64/, this
// vfxtool and the unsigned packages. See docs/en/dev/packaging-windows.md.
package signmsix

import (
	"archive/zip"
	"bufio"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/buildmsix"
)

const (
	defaultPublisherCN = "vfox-plus"
	defaultPackage     = "vfox-plus"
	// executableName is the file in the stages that build-msix packages.
	executableName = "vfox-plus.exe"
)

// publisherName is what a publisher common name may look like: no comma, which
// would end the name inside Publisher="CN=...".
var publisherName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9 ._-]*$`)

// bundleVersion is a version, as it stands in the name of a bundle.
var bundleVersion = regexp.MustCompile(`^[0-9]+(\.[0-9]+){0,3}$`)

// options is the command line.
type options struct {
	directory   string // holds stage-x64/, stage-arm64/ and the unsigned bundle
	publisherCN string
	pkg         string
	version     string
	force       bool
	skipInstall bool
	tool        string // an osslsigncode to use instead of the download
	noRelaunch  bool   // fail instead of relaunching elevated
	pause       bool   // set by the elevated copy: wait for Enter at the end
}

const usage = "Usage: sign-msix [-directory DIR] [-publisher-cn NAME] [-package NAME] [-version VERSION] [-force] [-skip-install] [-osslsigncode PATH] [-no-relaunch]"

func parseOptions(args []string) (options, error) {
	var o options
	flags := flag.NewFlagSet("sign-msix", flag.ContinueOnError)
	flags.StringVar(&o.directory, "directory", "", "`directory` that holds stage-x64/, stage-arm64/ and the unsigned bundle (default: the directory of vfxtool)")
	flags.StringVar(&o.publisherCN, "publisher-cn", defaultPublisherCN, "common `name` of the certificate and of the package publisher, like the CN= of Publisher=\"CN=...\"; letters, digits, space, dot, underscore and hyphen")
	flags.StringVar(&o.pkg, "package", defaultPackage, "package file name `prefix`, as passed to build-msix --package")
	flags.StringVar(&o.version, "version", "", "application `version`, like 0.1.0 (default: read from the unsigned <package>_<version>.msixbundle in the directory)")
	flags.BoolVar(&o.force, "force", false, "download osslsigncode again even when it is there")
	flags.BoolVar(&o.skipInstall, "skip-install", false, "rebuild and sign, but do not install the bundle")
	flags.StringVar(&o.tool, "osslsigncode", "", "osslsigncode `executable` to use instead of the download")
	flags.BoolVar(&o.noRelaunch, "no-relaunch", false, "fail when there are no administrator rights, instead of relaunching elevated (for scripts and CI, where nobody can answer the prompt)")
	flags.BoolVar(&o.pause, "pause", false, "wait for Enter before exiting (used by the elevated copy, which has a window of its own)")
	if err := flags.Parse(args); err != nil {
		return o, err
	}
	if flags.NArg() != 0 {
		return o, fmt.Errorf("unexpected argument %q\n%s", flags.Arg(0), usage)
	}
	if !publisherName.MatchString(o.publisherCN) {
		return o, fmt.Errorf("-publisher-cn must look like a common name without commas: %q", o.publisherCN)
	}
	if o.pkg == "" || strings.ContainsAny(o.pkg, `/\`) {
		return o, fmt.Errorf("bad -package %q", o.pkg)
	}
	if o.version != "" && !bundleVersion.MatchString(o.version) {
		return o, fmt.Errorf("bad -version %q (want like 0.1.0)", o.version)
	}
	if o.directory == "" {
		executable, err := os.Executable()
		if err != nil {
			return o, err
		}
		o.directory = filepath.Dir(executable)
	}
	directory, err := filepath.Abs(o.directory)
	if err != nil {
		return o, err
	}
	o.directory = directory
	return o, nil
}

// childArgs is the command line of the elevated copy, without the command
// name: the same options, with the directory made absolute, and -pause. The copy
// does not relaunch: if it still has no rights, it says so, and does not ask
// again for ever.
func (o options) childArgs() []string {
	args := []string{"-directory", o.directory, "-publisher-cn", o.publisherCN, "-package", o.pkg}
	if o.version != "" {
		args = append(args, "-version", o.version)
	}
	if o.force {
		args = append(args, "-force")
	}
	if o.skipInstall {
		args = append(args, "-skip-install")
	}
	if o.tool != "" {
		args = append(args, "-osslsigncode", o.tool)
	}
	return append(args, "-no-relaunch", "-pause")
}

// Run implements `sign-msix`. The flags:
//
//	-directory DIR        the extracted windows artifact (default: where vfxtool is)
//	-publisher-cn NAME    the common name of the certificate and the publisher (default vfox-plus)
//	-package NAME         the package file name prefix (default vfox-plus)
//	-version VERSION      the version (default: read from the unsigned bundle)
//	-force                download osslsigncode again
//	-skip-install         rebuild and sign, but do not install
//	-osslsigncode PATH    use this osslsigncode instead of the download
//	-no-relaunch          fail without administrator rights, instead of relaunching elevated
func Run(args []string) error {
	o, err := parseOptions(args)
	if err != nil {
		return err
	}
	return execute(o, newSystem())
}

// execute runs the command in the privileges that it needs: here when they
// are there, in an elevated copy when not.
func execute(o options, sys system) error {
	if !sys.elevated() {
		if o.noRelaunch {
			return errors.New("administrator rights are needed to trust the certificate and to install: run this from an elevated shell")
		}
		fmt.Println("Relaunching elevated (trusting the certificate and installing need administrator rights)...")
		code, err := sys.relaunchElevated(o.childArgs())
		if err != nil {
			return err
		}
		if code != 0 {
			return fmt.Errorf("the elevated copy exited with code %d (its window showed why)", code)
		}
		return nil
	}
	err := sign(o, sys)
	if o.pause {
		return waitBeforeClosing(err)
	}
	return err
}

// waitBeforeClosing keeps the window of the elevated copy open until Enter, so
// that its output can be read. A failure is told here and ends the process
// with it, because the caller would print it a second time.
var waitBeforeClosing = func(failure error) error {
	if failure != nil {
		fmt.Fprintf(os.Stderr, "vfxtool sign-msix: %v\n", failure)
	} else {
		fmt.Println("Done.")
	}
	fmt.Print("Press Enter to close this window.")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	if failure != nil {
		os.Exit(1)
	}
	return nil
}

// sign does the work, with the privileges that it needs.
func sign(o options, sys system) error {
	stages, err := stagesIn(o.directory)
	if err != nil {
		return err
	}
	version := o.version
	if version == "" {
		if version, err = detectVersion(o.directory, o.pkg); err != nil {
			return err
		}
	}
	fmt.Printf("Version: %s   Architectures: %s   Publisher: CN=%s\n", version, strings.Join(stages, ", "), o.publisherCN)

	// 1. The signing tool.
	tool, err := ensureTool(o.directory, o.tool, o.force)
	if err != nil {
		return err
	}

	// 2. The packages, with the publisher of the signature (the unsigned ones
	// carry the OID of unsigned packages, which would not match).
	signed := filepath.Join(o.directory, "signed")
	if err := rebuild(o, version, stages, signed); err != nil {
		return err
	}

	// 3. The signature, with a certificate that lives for this run. SignTool
	// signs the packages of a bundle and then the bundle, made of the signed
	// packages, and so does this: the signature of a bundle covers its manifest
	// only, so the packages in it have to be signed themselves.
	signer, err := newSigner(o.publisherCN, time.Now())
	if err != nil {
		return err
	}
	if err := signPackagesAndBundle(tool, signer, o, version, stages, signed); err != nil {
		return err
	}

	// 4. Trust for the machine, which is what sideloading looks for. Only now
	// that something is signed with it.
	fmt.Println("Trusting the signing certificate for the local machine (TrustedPeople)...")
	if err := sys.addToStore(signer.der, func(c *x509.Certificate) bool { return signer.replaces(c) }); err != nil {
		return err
	}

	bundle := filepath.Join(signed, fmt.Sprintf("%s_%s.msixbundle", o.pkg, version))
	if _, err := os.Stat(bundle); err != nil {
		return fmt.Errorf("the signed bundle is not in %s: %w", signed, err)
	}
	if o.skipInstall {
		fmt.Println("Signed bundle:", bundle)
		return nil
	}
	fmt.Printf("Installing %s...\n", filepath.Base(bundle))
	if err := sys.install(bundle); err != nil {
		return err
	}
	fmt.Println("Installed. Launch vfox+ from the Start menu.")
	return nil
}

// stagesIn lists the architectures that the directory has an executable for.
func stagesIn(directory string) ([]string, error) {
	var stages []string
	for _, arch := range []string{"x64", "arm64"} {
		info, err := os.Stat(filepath.Join(directory, "stage-"+arch, executableName))
		if err == nil && info.Mode().IsRegular() {
			stages = append(stages, arch)
		}
	}
	if len(stages) == 0 {
		return nil, fmt.Errorf("no %s in stage-x64 or stage-arm64 of %s: run this from the extracted windows artifact directory, or pass -directory", executableName, directory)
	}
	return stages, nil
}

// detectVersion reads the version from the name of the unsigned bundle that CI
// built, <package>_<version>.msixbundle.
func detectVersion(directory, pkg string) (string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", err
	}
	prefix := pkg + "_"
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, ".msixbundle") {
			continue
		}
		version := strings.TrimSuffix(strings.TrimPrefix(name, prefix), ".msixbundle")
		if !bundleVersion.MatchString(version) {
			return "", fmt.Errorf("cannot read a version from %s: pass -version", name)
		}
		return version, nil
	}
	return "", fmt.Errorf("no %s*.msixbundle in %s: pass -version", prefix, directory)
}

// rebuild writes the packages with the publisher of the signature into dir.
func rebuild(o options, version string, stages []string, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// The files of an earlier run are signed already, and a signed package
	// cannot be signed again: start with an empty directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && (strings.HasSuffix(name, ".msix") || strings.HasSuffix(name, ".msixbundle")) {
			if err := os.Remove(filepath.Join(dir, name)); err != nil {
				return err
			}
		}
	}
	args := []string{"--package", o.pkg, "--version", version, "--identity", buildmsix.DefaultIdentity, "--publisher", "CN=" + o.publisherCN, "--executable", executableName}
	for _, arch := range stages {
		args = append(args, "--stage-"+arch, filepath.Join(o.directory, "stage-"+arch))
	}
	fmt.Println("Rebuilding the packages with the publisher of the signature...")
	return buildmsix.Run(append(args, dir))
}

// packagesIn lists the .msix files in dir.
func packagesIn(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var packages []string
	for _, entry := range entries {
		if name := entry.Name(); !entry.IsDir() && strings.HasSuffix(name, ".msix") {
			packages = append(packages, filepath.Join(dir, name))
		}
	}
	sort.Strings(packages)
	if len(packages) == 0 {
		return nil, errors.New("nothing to sign: " + dir + " has no packages")
	}
	return packages, nil
}

// signPackagesAndBundle signs every .msix in dir, puts the signed packages into
// the bundle again (the offsets and the sizes in its manifest are theirs now),
// and signs the bundle.
func signPackagesAndBundle(tool string, s *signer, o options, version string, stages []string, dir string) error {
	packages, err := packagesIn(dir)
	if err != nil {
		return err
	}
	session, err := s.open(tool)
	if err != nil {
		return err
	}
	defer session.close()
	for _, file := range packages {
		if err := signAndCheck(session, file); err != nil {
			return err
		}
	}
	fmt.Println("Putting the signed packages into the bundle...")
	bundle, err := buildmsix.Rebundle(buildmsix.RebundleConfig{
		Dir:           dir,
		Package:       o.pkg,
		Version:       version,
		Publisher:     "CN=" + o.publisherCN,
		Architectures: stages,
	})
	if err != nil {
		return err
	}
	return signAndCheck(session, bundle)
}

// signAndCheck signs a file and checks what osslsigncode made of it.
func signAndCheck(session *signing, file string) error {
	if err := session.sign(file); err != nil {
		return err
	}
	if err := checkArchive(file); err != nil {
		return fmt.Errorf("osslsigncode made a corrupt archive of %s: %w", filepath.Base(file), err)
	}
	return nil
}

// checkArchive reads every entry of a ZIP archive to its end, which fails on a
// size or a CRC that the entry does not keep. osslsigncode rewrites a part of the
// archive when it signs, and a version that does it wrongly makes a file that its
// own verification accepts and that Windows refuses (0x8007000D, at the
// installation): the result is checked here, where the cause is still known.
func checkArchive(path string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer func() { _ = reader.Close() }()
	for _, file := range reader.File {
		entry, err := file.Open()
		if err != nil {
			return fmt.Errorf("%s: %w", file.Name, err)
		}
		read, err := io.Copy(io.Discard, entry)
		_ = entry.Close()
		if err != nil || uint64(read) != file.UncompressedSize64 {
			return fmt.Errorf("%s is a corrupt entry (method %d, %d of %d bytes): %v", file.Name, file.Method, read, file.UncompressedSize64, err)
		}
	}
	return nil
}
