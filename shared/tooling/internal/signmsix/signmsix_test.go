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

package signmsix

import (
	"archive/zip"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// The tests run this test binary as osslsigncode: with fakeToolEnv set, it
// checks that it is called the way the tool is called, that the certificate and
// the key that it is given belong together, and appends a line with the serial
// number of the certificate to the file, in place of a signature. It refuses a
// file that has such a line already, as osslsigncode refuses a file that is
// signed. The value of fakeFailEnv, when set, is a name that it fails for.
const (
	fakeToolEnv = "VFXTOOL_FAKE_OSSLSIGNCODE"
	fakeFailEnv = "VFXTOOL_FAKE_FAIL_FOR"
	trailer     = "SIGNED-BY serial="
)

func TestMain(m *testing.M) {
	if os.Getenv(fakeToolEnv) != "" {
		if err := fakeTool(os.Args[1:]); err != nil {
			fmt.Fprintln(os.Stderr, "fake osslsigncode:", err)
			os.Exit(1)
		}
		return
	}
	os.Exit(m.Run())
}

func fakeTool(args []string) error {
	if len(args) != 9 || args[0] != "sign" || args[1] != "-certs" || args[3] != "-key" || args[5] != "-in" || args[7] != "-out" {
		return fmt.Errorf("unexpected arguments %q", args)
	}
	chain, err := os.ReadFile(args[2])
	if err != nil {
		return err
	}
	private, err := os.ReadFile(args[4])
	if err != nil {
		return err
	}
	pair, err := tls.X509KeyPair(chain, private)
	if err != nil {
		return fmt.Errorf("the certificate and the key do not belong together: %w", err)
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		return err
	}
	if fail := os.Getenv(fakeFailEnv); fail != "" && strings.Contains(args[6], fail) {
		return fmt.Errorf("cannot sign %s", filepath.Base(args[6]))
	}
	data, err := os.ReadFile(args[6])
	if err != nil {
		return err
	}
	if signedBy(data) != "" {
		return errors.New("the file is signed already")
	}
	data = append(data, []byte("\n"+trailer+hex.EncodeToString(leaf.SerialNumber.Bytes())+"\n")...)
	return os.WriteFile(args[8], data, 0o644)
}

// signedBy returns the serial number that the fake signature at the end of a
// file names, or "": the file is signed when its last line is one. (The packages
// in a signed bundle have such a line of their own, inside.)
func signedBy(data []byte) string {
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	last := lines[len(lines)-1]
	if serial, found := strings.CutPrefix(last, trailer); found {
		return strings.TrimSpace(serial)
	}
	return ""
}

// fakeSystem records what the command asks of the system.
type fakeSystem struct {
	isElevated  bool
	exitCode    int
	relaunchErr error
	installErr  error
	// onStore runs when a certificate is added, to look at what exists then.
	onStore func(der []byte)

	relaunched [][]string
	stored     [][]byte
	replaces   func(*x509.Certificate) bool
	installed  []string
}

func (f *fakeSystem) elevated() bool { return f.isElevated }

func (f *fakeSystem) relaunchElevated(args []string) (int, error) {
	f.relaunched = append(f.relaunched, args)
	return f.exitCode, f.relaunchErr
}

func (f *fakeSystem) addToStore(der []byte, replaces func(*x509.Certificate) bool) error {
	f.stored = append(f.stored, der)
	f.replaces = replaces
	if f.onStore != nil {
		f.onStore(der)
	}
	return nil
}

func (f *fakeSystem) install(bundle string) error {
	f.installed = append(f.installed, bundle)
	return f.installErr
}

// artifact makes the directory of the extracted windows artifact: the stages
// that arches name, and the unsigned bundle of the version (when there is one).
func artifact(t *testing.T, version string, arches ...string) string {
	t.Helper()
	directory := t.TempDir()
	for _, arch := range arches {
		stage := filepath.Join(directory, "stage-"+arch)
		if err := os.MkdirAll(stage, 0o755); err != nil {
			t.Fatal(err)
		}
		executable := append([]byte("MZ"), bytes.Repeat([]byte{0x90}, 2000)...)
		if err := os.WriteFile(filepath.Join(stage, executableName), executable, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if version != "" {
		if err := os.WriteFile(filepath.Join(directory, "vfox-plus_"+version+".msixbundle"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return directory
}

// run is execute with the defaults of the command line, and this test binary as
// the signing tool.
func run(t *testing.T, directory string, sys system, mutate func(*options)) error {
	t.Helper()
	t.Setenv(fakeToolEnv, "1")
	o := options{directory: directory, publisherCN: defaultPublisherCN, pkg: defaultPackage, tool: os.Args[0]}
	if mutate != nil {
		mutate(&o)
	}
	return execute(o, sys)
}

func TestSignRebuildsSignsTrustsAndInstalls(t *testing.T) {
	directory := artifact(t, "0.1.0", "x64", "arm64")
	signed := filepath.Join(directory, "signed")
	// What an earlier run of another version left: signed already, and not
	// rebuilt by this run, so that signing it again would fail.
	if err := os.MkdirAll(signed, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(signed, "vfox-plus_0.0.9_x64.msix")
	if err := os.WriteFile(stale, []byte("old"+"\n"+trailer+"01\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sys := &fakeSystem{isElevated: true}
	var storedBefore []string
	sys.onStore = func(der []byte) {
		// The certificate is trusted only once everything is signed with it.
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			t.Error(err)
			return
		}
		for _, name := range []string{"vfox-plus_0.1.0_x64.msix", "vfox-plus_0.1.0_arm64.msix", "vfox-plus_0.1.0.msixbundle"} {
			data, err := os.ReadFile(filepath.Join(signed, name))
			if err != nil || signedBy(data) != hex.EncodeToString(cert.SerialNumber.Bytes()) {
				t.Errorf("%s is not signed by the certificate that is added: %v", name, err)
			}
			storedBefore = append(storedBefore, name)
		}
	}
	if err := run(t, directory, sys, nil); err != nil {
		t.Fatal(err)
	}
	if len(storedBefore) != 3 {
		t.Errorf("the certificate was not added after the signing: %v", storedBefore)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("the package of an earlier run is still there: %v", err)
	}
	entries, err := os.ReadDir(signed)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	if want := []string{"vfox-plus_0.1.0.msixbundle", "vfox-plus_0.1.0_arm64.msix", "vfox-plus_0.1.0_x64.msix"}; !slices.Equal(names, want) {
		t.Errorf("signed/ holds %v, want %v", names, want)
	}
	// The packages carry the publisher of the signature, and not the OID of the
	// unsigned ones; the files are stored, not compressed, so it can be read.
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(signed, name))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(data, []byte(`Publisher="CN=vfox-plus"`)) || bytes.Contains(data, []byte("OID.2.25")) {
			t.Errorf("%s has the wrong publisher", name)
		}
	}
	checkBundleHoldsSignedPackages(t, signed, "vfox-plus_0.1.0.msixbundle")
	if len(sys.stored) != 1 {
		t.Fatalf("%d certificates added", len(sys.stored))
	}
	if want := []string{filepath.Join(signed, "vfox-plus_0.1.0.msixbundle")}; !slices.Equal(sys.installed, want) {
		t.Errorf("installed %v, want %v", sys.installed, want)
	}
	if len(sys.relaunched) != 0 {
		t.Error("an elevated process relaunched itself")
	}

	// The certificate that an earlier run added is what this run replaces.
	cert, err := x509.ParseCertificate(sys.stored[0])
	if err != nil {
		t.Fatal(err)
	}
	earlier, err := newSigner("vfox-plus", cert.NotBefore)
	if err != nil {
		t.Fatal(err)
	}
	earlierCert, _ := x509.ParseCertificate(earlier.der)
	if !sys.replaces(earlierCert) || sys.replaces(cert) {
		t.Error("the replacement rule is wrong")
	}
}

func TestSignWithoutInstalling(t *testing.T) {
	directory := artifact(t, "", "x64")
	sys := &fakeSystem{isElevated: true}
	err := run(t, directory, sys, func(o *options) { o.skipInstall = true; o.version = "1.2.3" })
	if err != nil {
		t.Fatal(err)
	}
	// Trusted all the same: a bundle that is not trusted cannot be double-clicked.
	if len(sys.stored) != 1 || len(sys.installed) != 0 {
		t.Errorf("%d certificates added, installed %v", len(sys.stored), sys.installed)
	}
	if _, err := os.Stat(filepath.Join(directory, "signed", "vfox-plus_1.2.3.msixbundle")); err != nil {
		t.Error(err)
	}
	// One architecture only.
	if _, err := os.Stat(filepath.Join(directory, "signed", "vfox-plus_1.2.3_arm64.msix")); !os.IsNotExist(err) {
		t.Errorf("an arm64 package without an arm64 stage: %v", err)
	}
}

func TestSignAnotherPublisherAndPackage(t *testing.T) {
	directory := artifact(t, "", "arm64")
	sys := &fakeSystem{isElevated: true}
	err := run(t, directory, sys, func(o *options) {
		o.skipInstall = true
		o.version = "2.0.0"
		o.publisherCN = "Example Corp."
		o.pkg = "example"
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(directory, "signed", "example_2.0.0.msixbundle"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`Publisher="CN=Example Corp."`)) {
		t.Error("the bundle has the wrong publisher")
	}
	cert, _ := x509.ParseCertificate(sys.stored[0])
	if cert.Subject.String() != "CN=Example Corp." {
		t.Errorf("the certificate is for %s", cert.Subject)
	}
}

func TestSigningFailureStopsBeforeTrust(t *testing.T) {
	directory := artifact(t, "0.1.0", "x64")
	t.Setenv(fakeFailEnv, ".msixbundle")
	sys := &fakeSystem{isElevated: true}
	err := run(t, directory, sys, nil)
	if err == nil || !strings.Contains(err.Error(), "osslsigncode failed for vfox-plus_0.1.0.msixbundle") {
		t.Fatalf("got %v", err)
	}
	// Nothing was trusted or installed, and no half-signed file is left.
	if len(sys.stored) != 0 || len(sys.installed) != 0 {
		t.Errorf("added %d certificates, installed %v", len(sys.stored), sys.installed)
	}
	matches, _ := filepath.Glob(filepath.Join(directory, "signed", "*.signed-tmp"))
	if len(matches) != 0 {
		t.Errorf("temporary files are left: %v", matches)
	}
}

func TestInstallFailureIsReported(t *testing.T) {
	directory := artifact(t, "0.1.0", "x64")
	sys := &fakeSystem{isElevated: true, installErr: errors.New("0x80073CF3")}
	if err := run(t, directory, sys, nil); err == nil || !strings.Contains(err.Error(), "0x80073CF3") {
		t.Fatalf("got %v", err)
	}
}

func TestSignNeedsStagesAndAVersion(t *testing.T) {
	sys := &fakeSystem{isElevated: true}
	if err := run(t, t.TempDir(), sys, nil); err == nil || !strings.Contains(err.Error(), "no vfox-plus.exe") {
		t.Errorf("no stages: %v", err)
	}
	if err := run(t, artifact(t, "", "x64"), sys, nil); err == nil || !strings.Contains(err.Error(), "pass -version") {
		t.Errorf("no version: %v", err)
	}
	if len(sys.stored)+len(sys.installed) != 0 {
		t.Error("the system was touched")
	}
}

func TestElevationRelaunch(t *testing.T) {
	directory := artifact(t, "0.1.0", "x64")
	sys := &fakeSystem{}
	err := run(t, directory, sys, func(o *options) { o.version = "0.1.0"; o.force = true })
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-directory", directory, "-publisher-cn", "vfox-plus", "-package", "vfox-plus", "-version", "0.1.0", "-force", "-osslsigncode", os.Args[0], "-no-relaunch", "-pause"}
	if len(sys.relaunched) != 1 || !slices.Equal(sys.relaunched[0], want) {
		t.Errorf("relaunched with %q, want %q", sys.relaunched, want)
	}
	// The copy does the work: this one does none.
	if _, err := os.Stat(filepath.Join(directory, "signed")); !os.IsNotExist(err) {
		t.Errorf("the process without rights signed: %v", err)
	}

	sys = &fakeSystem{exitCode: 1}
	if err := run(t, directory, sys, nil); err == nil || !strings.Contains(err.Error(), "exited with code 1") {
		t.Errorf("a failed copy: %v", err)
	}
	sys = &fakeSystem{relaunchErr: errors.New("The operation was canceled by the user.")}
	if err := run(t, directory, sys, nil); err == nil || !strings.Contains(err.Error(), "canceled by the user") {
		t.Errorf("a refused prompt: %v", err)
	}
}

func TestNoRelaunchFailsWithoutRights(t *testing.T) {
	directory := artifact(t, "0.1.0", "x64")
	sys := &fakeSystem{}
	err := run(t, directory, sys, func(o *options) { o.noRelaunch = true })
	if err == nil || !strings.Contains(err.Error(), "administrator rights are needed") {
		t.Fatalf("got %v", err)
	}
	if len(sys.relaunched) != 0 {
		t.Error("it relaunched although it was told not to")
	}
	if _, err := os.Stat(filepath.Join(directory, "signed")); !os.IsNotExist(err) {
		t.Errorf("it signed without the rights: %v", err)
	}
	// With the rights it does not matter.
	sys = &fakeSystem{isElevated: true}
	if err := run(t, directory, sys, func(o *options) { o.noRelaunch = true; o.skipInstall = true }); err != nil {
		t.Fatal(err)
	}
}

func TestPauseWaitsAfterTheWork(t *testing.T) {
	var told []error
	previous := waitBeforeClosing
	waitBeforeClosing = func(failure error) error { told = append(told, failure); return failure }
	t.Cleanup(func() { waitBeforeClosing = previous })

	directory := artifact(t, "0.1.0", "x64")
	sys := &fakeSystem{isElevated: true}
	if err := run(t, directory, sys, func(o *options) { o.pause = true; o.skipInstall = true }); err != nil {
		t.Fatal(err)
	}
	if err := run(t, t.TempDir(), sys, func(o *options) { o.pause = true }); err == nil {
		t.Fatal("no error without stages")
	}
	if len(told) != 2 || told[0] != nil || told[1] == nil {
		t.Errorf("the window was told %v", told)
	}
}

func TestParseOptions(t *testing.T) {
	directory := t.TempDir()
	got, err := parseOptions([]string{"-directory", directory})
	if err != nil {
		t.Fatal(err)
	}
	if got.directory != directory || got.publisherCN != "vfox-plus" || got.pkg != "vfox-plus" || got.version != "" || got.force || got.skipInstall || got.noRelaunch || got.pause {
		t.Errorf("defaults: %+v", got)
	}
	got, err = parseOptions([]string{"--directory", directory, "--publisher-cn", "Han Li", "--package", "p", "--version", "1.2.3.4", "--force", "--skip-install", "--osslsigncode", "x", "--no-relaunch", "--pause"})
	if err != nil {
		t.Fatal(err)
	}
	if got.publisherCN != "Han Li" || got.pkg != "p" || got.version != "1.2.3.4" || !got.force || !got.skipInstall || got.tool != "x" || !got.noRelaunch || !got.pause {
		t.Errorf("parsed: %+v", got)
	}
	// The directory defaults to where the program is, as the script did.
	if got, err = parseOptions(nil); err != nil || !filepath.IsAbs(got.directory) {
		t.Errorf("default directory %q, %v", got.directory, err)
	}
	// A relative directory is made absolute, so that the elevated copy finds it.
	if got, err = parseOptions([]string{"-directory", "relative"}); err != nil || !filepath.IsAbs(got.directory) {
		t.Errorf("relative directory %q, %v", got.directory, err)
	}
	for name, args := range map[string][]string{
		"comma":        {"-publisher-cn", "Han Li, Inc."},
		"empty":        {"-publisher-cn", ""},
		"leading dot":  {"-publisher-cn", ".vfox"},
		"quote":        {"-publisher-cn", `a"b`},
		"package path": {"-package", "a/b"},
		"no package":   {"-package", ""},
		"version":      {"-version", "v1"},
		"stray":        {"stray"},
		"unknown flag": {"-nope"},
	} {
		if _, err := parseOptions(args); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestChildArgsRoundTrip(t *testing.T) {
	directory := t.TempDir()
	original, err := parseOptions([]string{"-directory", directory, "-publisher-cn", "Han Li", "-package", "p", "-version", "3.2.1", "-force", "-skip-install", "-osslsigncode", "tool"})
	if err != nil {
		t.Fatal(err)
	}
	child, err := parseOptions(original.childArgs())
	if err != nil {
		t.Fatal(err)
	}
	// The copy is told not to relaunch, and to wait for Enter at the end.
	original.noRelaunch, original.pause = true, true
	if child != original {
		t.Errorf("the copy got %+v, want %+v", child, original)
	}
}

func TestDetectVersion(t *testing.T) {
	directory := t.TempDir()
	if _, err := detectVersion(directory, "vfox-plus"); err == nil || !strings.Contains(err.Error(), "pass -version") {
		t.Errorf("no bundle: %v", err)
	}
	for _, name := range []string{"vfox-plus_0.1.0_x64.msix", "other_9.9.9.msixbundle", "vfox-plus.msixbundle"} {
		if err := os.WriteFile(filepath.Join(directory, name), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := detectVersion(directory, "vfox-plus"); err == nil {
		t.Error("a package that is not a bundle gave a version")
	}
	if err := os.WriteFile(filepath.Join(directory, "vfox-plus_10.20.3.msixbundle"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if got, err := detectVersion(directory, "vfox-plus"); err != nil || got != "10.20.3" {
		t.Errorf("got %q, %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(directory, "vfox-plus_0.1.0-rc1.msixbundle"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := detectVersion(directory, "vfox-plus"); err == nil || !strings.Contains(err.Error(), "cannot read a version") {
		t.Errorf("an unreadable version: %v", err)
	}
}

func TestStagesIn(t *testing.T) {
	if got, err := stagesIn(artifact(t, "", "arm64")); err != nil || !slices.Equal(got, []string{"arm64"}) {
		t.Errorf("arm64: %v, %v", got, err)
	}
	if got, err := stagesIn(artifact(t, "", "x64", "arm64")); err != nil || !slices.Equal(got, []string{"x64", "arm64"}) {
		t.Errorf("both: %v, %v", got, err)
	}
	// A directory of that name is no executable.
	directory := t.TempDir()
	if err := os.MkdirAll(filepath.Join(directory, "stage-x64", executableName), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := stagesIn(directory); err == nil {
		t.Error("a directory counted as the executable")
	}
}

// checkBundleHoldsSignedPackages checks what makes a signed bundle installable,
// with the fake signatures of these tests (see checkBundleHoldsPackages).
func checkBundleHoldsSignedPackages(t *testing.T, dir, bundleName string) {
	t.Helper()
	checkBundleHoldsPackages(t, dir, bundleName, func(t *testing.T, name string, data []byte) {
		if signedBy(data) == "" {
			t.Errorf("%s is not signed", name)
		}
	})
}

// checkBundleHoldsPackages checks what makes a signed bundle installable: the
// signature of the bundle covers its manifest only, so the packages in it are the
// signed files (not the unsigned ones that the bundle was first built of), each
// one where the manifest says, as large as it says. look sees each package of
// the directory once, to check its signature.
func checkBundleHoldsPackages(t *testing.T, dir, bundleName string, look func(t *testing.T, name string, data []byte)) {
	t.Helper()
	bundle, err := os.ReadFile(filepath.Join(dir, bundleName))
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		// A fake signature is a line after the end of the archive: read it as the signed file.
		reader = nil
	}
	manifest := bundle
	if reader != nil {
		for _, file := range reader.File {
			if file.Name == "AppxMetadata/AppxBundleManifest.xml" {
				entry, err := file.Open()
				if err != nil {
					t.Fatal(err)
				}
				manifest, _ = io.ReadAll(entry)
				_ = entry.Close()
			}
		}
	}
	listed := regexp.MustCompile(`FileName="([^"]+)"[^>]*Offset="(\d+)" Size="(\d+)"`).FindAllStringSubmatch(string(manifest), -1)
	if len(listed) == 0 {
		t.Fatalf("%s lists no package", bundleName)
	}
	for _, match := range listed {
		offset, _ := strconv.Atoi(match[2])
		size, _ := strconv.Atoi(match[3])
		alone, err := os.ReadFile(filepath.Join(dir, match[1]))
		if err != nil {
			t.Fatal(err)
		}
		look(t, match[1], alone)
		if size != len(alone) || offset+size > len(bundle) || !bytes.Equal(bundle[offset:offset+size], alone) {
			t.Errorf("%s: the bundle does not hold the signed file at %d, %d bytes", match[1], offset, size)
		}
	}
}

// A file whose entry keeps neither its size nor its CRC is what osslsigncode made
// of a stored [Content_Types].xml: it is found where it is made.
func TestCheckArchiveFindsACorruptEntry(t *testing.T) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	good, err := archive.Create("good.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = good.Write([]byte("<a/>"))
	// The entry says Store and 479 bytes, and holds 243 bytes of deflated data.
	bad, err := archive.CreateRaw(&zip.FileHeader{Name: "[Content_Types].xml", Method: zip.Store, CRC32: 0x75c98439, CompressedSize64: 243, UncompressedSize64: 479})
	if err != nil {
		t.Fatal(err)
	}
	_, _ = bad.Write(bytes.Repeat([]byte{0x5a}, 243))
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "corrupt.msix")
	if err := os.WriteFile(path, buffer.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	err = checkArchive(path)
	if err == nil || !strings.Contains(err.Error(), "[Content_Types].xml is a corrupt entry") {
		t.Fatalf("got %v", err)
	}

	var whole bytes.Buffer
	valid := zip.NewWriter(&whole)
	entry, _ := valid.Create("a.xml")
	_, _ = entry.Write([]byte("<a/>"))
	_ = valid.Close()
	good2 := filepath.Join(t.TempDir(), "good.msix")
	if err := os.WriteFile(good2, whole.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := checkArchive(good2); err != nil {
		t.Errorf("a good archive: %v", err)
	}
	if err := checkArchive(filepath.Join(t.TempDir(), "missing.msix")); err == nil {
		t.Error("a missing file passed")
	}
}
