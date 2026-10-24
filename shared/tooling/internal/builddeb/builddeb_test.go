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

package builddeb

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/deb"
	"github.com/goreleaser/nfpm/v2/files"
)

func TestMapDebArch(t *testing.T) {
	debArch, tarArch, err := MapDebArch("x86_64")
	if err != nil {
		t.Fatal(err)
	}
	if debArch != "amd64" || tarArch != "x64" {
		t.Fatalf("x86_64 -> %q/%q, want amd64/x64", debArch, tarArch)
	}
	debArch, tarArch, err = MapDebArch("aarch64")
	if err != nil {
		t.Fatal(err)
	}
	if debArch != "arm64" || tarArch != "arm64" {
		t.Fatalf("aarch64 -> %q/%q, want arm64/arm64", debArch, tarArch)
	}
	if _, _, err := MapDebArch("riscv64"); err == nil {
		t.Fatal("expected an error for an unsupported architecture")
	} else if !strings.Contains(err.Error(), "riscv64") {
		t.Fatalf("error %q does not name the architecture", err)
	}
}

func TestRenderControl(t *testing.T) {
	got := RenderControl(Control{
		Package:       "vfox-plus-mygo",
		Version:       "1.2.3",
		DebArch:       "amd64",
		Maintainer:    "Han Li and contributors <https://github.com/version-fox/vfox-plus/issues>",
		InstalledSize: "12345",
		Depends:       "libc6, libgtk-3-0t64",
		Recommends:    "fonts-noto-cjk",
		Technology:    "mygo",
	})
	want := `Package: vfox-plus-mygo
Version: 1.2.3
Section: devel
Priority: optional
Architecture: amd64
Maintainer: Han Li and contributors <https://github.com/version-fox/vfox-plus/issues>
Installed-Size: 12345
Depends: libc6, libgtk-3-0t64
Recommends: vfox, fonts-noto-cjk
Provides: vfox-plus
Conflicts: vfox-plus
Replaces: vfox-plus
Homepage: https://github.com/version-fox/vfox-plus
Description: Material 3 Expressive desktop GUI for vfox (mygo)
 vfox+ shows the installed SDKs and the versions in use, installs and
 uninstalls versions, pins versions in projects, browses the vfox plugin
 registry and edits the vfox configuration. It drives the vfox command line
 tool, which must be installed separately.
 .
 This package is the implementation of vfox+ built with mygo.
`
	if got != want {
		t.Fatalf("RenderControl:\n%s\nwant:\n%s", got, want)
	}
}

func TestRenderControlWithoutRecommends(t *testing.T) {
	got := RenderControl(Control{Recommends: ""})
	line := ""
	for _, l := range strings.Split(got, "\n") {
		if strings.HasPrefix(l, "Recommends:") {
			line = l
		}
	}
	if line != "Recommends: vfox" {
		t.Fatalf("Recommends line = %q, want %q", line, "Recommends: vfox")
	}
}

func TestRenderChangelog(t *testing.T) {
	got := RenderChangelog("vfox-plus-mygo", "1.2.3", "Jane Doe <jane@example.com>", 0)
	if !strings.Contains(got, "vfox-plus-mygo (1.2.3) unstable; urgency=medium") {
		t.Fatalf("changelog header missing:\n%s", got)
	}
	if !strings.Contains(got, "  * Release 1.2.3. See https://github.com/version-fox/vfox-plus/releases.") {
		t.Fatalf("changelog entry missing:\n%s", got)
	}
	if !strings.Contains(got, " -- Jane Doe <jane@example.com>  ") {
		t.Fatalf("changelog trailer missing:\n%s", got)
	}
}

func TestParseArgs(t *testing.T) {
	opts, err := parseArgs([]string{
		"--package", "vfox-plus-mygo",
		"--version", "1.2.3",
		"--stage", "/tmp/stage",
		"--executable", "vfox-plus",
		"--depends", "libc6",
		"--recommends", "fonts-noto-cjk",
		"--technology", "mygo",
		"--notice", "/tmp/NOTICE",
		"--override", "a: b",
		"--override", "c: d",
		"--setuid", "helper",
		"--strip",
		"/tmp/out",
	})
	if err != nil {
		t.Fatal(err)
	}
	if opts.pkg != "vfox-plus-mygo" || !opts.strip || opts.output != "/tmp/out" {
		t.Fatalf("unexpected options: %+v", opts)
	}
	if len(opts.overrides) != 2 || len(opts.setuid) != 1 {
		t.Fatalf("repeatable flags not collected: %+v", opts)
	}
	if _, err := parseArgs([]string{"--bogus", "x", "/tmp/out"}); err == nil {
		t.Fatal("expected an error for an unknown option")
	}
	if _, err := parseArgs([]string{"--package", "p", "/tmp/out"}); err == nil {
		t.Fatal("expected an error for missing required options")
	}
	if _, err := parseArgs(nil); err == nil {
		t.Fatal("expected an error for a missing output directory")
	}
}

// fixtureRoot creates a repository layout with a stage, a packaging
// directory and a LICENSE, and moves the test into it so findPackagingDir
// resolves the fixture.
func fixtureRoot(t *testing.T) (stage, out string) {
	t.Helper()
	root := t.TempDir()
	packaging := filepath.Join(root, "shared", "packaging", "linux")
	if err := os.MkdirAll(packaging, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{appID + ".desktop", appID + ".metainfo.xml", appID + ".svg"} {
		if err := os.WriteFile(filepath.Join(packaging, name), []byte("test "+name+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "LICENSE"), []byte("license\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stage = filepath.Join(root, "stage")
	if err := os.MkdirAll(filepath.Join(stage, "bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, f := range []struct {
		path string
		mode os.FileMode
		body string
	}{
		{"vfox-plus", 0o755, "#!/bin/sh\necho hi\n"},
		{filepath.Join("bin", "helper"), 0o755, "#!/bin/sh\necho helper\n"},
		{"data.txt", 0o644, "data\n"},
	} {
		if err := os.WriteFile(filepath.Join(stage, f.path), []byte(f.body), f.mode); err != nil {
			t.Fatal(err)
		}
	}
	out = filepath.Join(root, "out")
	t.Chdir(root)
	t.Setenv("SOURCE_DATE_EPOCH", "1234567890")
	t.Setenv("DEB_MAINTAINER", "Test User <test@example.com>")
	return stage, out
}

func runBuild(t *testing.T, args ...string) {
	t.Helper()
	if err := Run(args); err != nil {
		t.Fatal(err)
	}
}

func globOne(t *testing.T, dir, pattern string) string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected one %s in %s, got %v", pattern, dir, matches)
	}
	return matches[0]
}

// readAr parses a .deb (an ar archive) into its members.
func readAr(t *testing.T, path string) map[string][]byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 8 || string(data[:8]) != "!<arch>\n" {
		t.Fatalf("%s is not an ar archive", path)
	}
	members := map[string][]byte{}
	off := 8
	for off < len(data) {
		if off+60 > len(data) {
			t.Fatalf("truncated ar header in %s", path)
		}
		header := data[off : off+60]
		if header[58] != '`' || header[59] != '\n' {
			t.Fatalf("bad ar header in %s", path)
		}
		name := strings.TrimSpace(string(header[:16]))
		size, err := strconv.ParseInt(strings.TrimSpace(string(header[48:58])), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		off += 60
		if off+int(size) > len(data) {
			t.Fatalf("truncated ar member %s in %s", name, path)
		}
		members[name] = data[off : off+int(size)]
		off += int(size)
		if size%2 == 1 {
			off++
		}
	}
	return members
}

// readTarGz reads a gzip-compressed tar archive into name -> header.
func readTarGz(t *testing.T, data []byte) map[string]*tar.Header {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	members := map[string]*tar.Header{}
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		members[header.Name] = header
		if _, err := io.Copy(io.Discard, tr); err != nil {
			t.Fatal(err)
		}
	}
	return members
}

// readTarGzFile returns the content of one file in a gzip-compressed tar.
func readTarGzFile(t *testing.T, data []byte, name string) []byte {
	t.Helper()
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			t.Fatalf("no %s in archive", name)
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Name == name {
			content, err := io.ReadAll(tr)
			if err != nil {
				t.Fatal(err)
			}
			return content
		}
		if _, err := io.Copy(io.Discard, tr); err != nil {
			t.Fatal(err)
		}
	}
}

func controlField(t *testing.T, control, field string) string {
	t.Helper()
	for _, line := range strings.Split(control, "\n") {
		if strings.HasPrefix(line, field+":") {
			return strings.TrimSpace(strings.TrimPrefix(line, field+":"))
		}
	}
	t.Fatalf("control field %s missing:\n%s", field, control)
	return ""
}

func TestRunBuildsDebAndTarball(t *testing.T) {
	stage, out := fixtureRoot(t)
	notice := filepath.Join(stage, "NOTICE.txt")
	if err := os.WriteFile(notice, []byte("third-party\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	debArch, tarArch, err := MapDebArch(unameMachine())
	if err != nil {
		t.Fatal(err)
	}
	runBuild(t,
		"--package", "vfox-plus-test",
		"--version", "1.2.3",
		"--stage", stage,
		"--executable", "vfox-plus",
		"--depends", "libc6",
		"--recommends", "fonts-noto-cjk",
		"--technology", "testtech",
		"--notice", notice,
		"--override", "some-tag [usr/bin/vfox-plus]",
		out,
	)

	deb := globOne(t, out, "*.deb")
	if want := filepath.Join(out, "vfox-plus-test_1.2.3_"+debArch+".deb"); deb != want {
		t.Fatalf("deb = %s, want %s", deb, want)
	}
	tarball := globOne(t, out, "*.tar.gz")
	if want := filepath.Join(out, "vfox-plus-test-1.2.3-linux-"+tarArch+".tar.gz"); tarball != want {
		t.Fatalf("tarball = %s, want %s", tarball, want)
	}

	members := readAr(t, deb)
	controlData, ok := members["control.tar.gz"]
	if !ok {
		t.Fatalf("control.tar.gz missing in %s: %v", deb, members)
	}
	var dataData []byte
	for name, content := range members {
		if strings.HasPrefix(name, "data.tar.") {
			dataData = content
		}
	}
	if dataData == nil {
		t.Fatalf("data tarball missing in %s", deb)
	}
	control := string(readTarGzFile(t, controlData, "./control"))
	for _, want := range []string{
		"Package: vfox-plus-test",
		"Version: 1.2.3",
		"Architecture: " + debArch,
		"Maintainer: Test User <test@example.com>",
		"Depends: libc6",
		"Recommends: vfox, fonts-noto-cjk",
		"Provides: vfox-plus",
		"Conflicts: vfox-plus",
		"Replaces: vfox-plus",
		"Description: Material 3 Expressive desktop GUI for vfox (testtech)",
	} {
		if !strings.Contains(control, want) {
			t.Fatalf("control missing %q:\n%s", want, control)
		}
	}
	if field := controlField(t, control, "Installed-Size"); field == "" {
		t.Fatalf("Installed-Size empty:\n%s", control)
	}

	data := readTarGz(t, dataData)
	for _, name := range []string{
		"./usr/lib/vfox-plus/vfox-plus",
		"./usr/lib/vfox-plus/bin/helper",
		"./usr/lib/vfox-plus/data.txt",
		"./usr/share/applications/" + appID + ".desktop",
		"./usr/share/metainfo/" + appID + ".metainfo.xml",
		"./usr/share/icons/hicolor/scalable/apps/" + appID + ".svg",
		"./usr/share/doc/vfox-plus-test/copyright",
		"./usr/share/doc/vfox-plus-test/changelog.gz",
		"./usr/share/doc/vfox-plus-test/THIRD-PARTY-LICENSES",
		"./usr/share/lintian/overrides/vfox-plus-test",
	} {
		if _, ok := data[name]; !ok {
			t.Fatalf("%s missing in data tarball", name)
		}
	}
	symlink, ok := data["./usr/bin/vfox-plus"]
	if !ok {
		t.Fatal("./usr/bin/vfox-plus missing in data tarball")
	}
	if symlink.Typeflag != tar.TypeSymlink || symlink.Linkname != "../lib/vfox-plus/vfox-plus" {
		t.Fatalf("bad ./usr/bin/vfox-plus entry: %+v", symlink)
	}
	changelogGz := readTarGzFile(t, dataData, "./usr/share/doc/vfox-plus-test/changelog.gz")
	gz, err := gzip.NewReader(bytes.NewReader(changelogGz))
	if err != nil {
		t.Fatal(err)
	}
	changelog, err := io.ReadAll(gz)
	if err != nil {
		t.Fatal(err)
	}
	gz.Close()
	if !strings.Contains(string(changelog), "vfox-plus-test (1.2.3) unstable; urgency=medium") {
		t.Fatalf("changelog content wrong:\n%s", changelog)
	}

	tarballData, err := os.ReadFile(tarball)
	if err != nil {
		t.Fatal(err)
	}
	tarballMembers := readTarGz(t, tarballData)
	for _, name := range []string{
		"./",
		"./vfox-plus",
		"./bin/helper",
		"./data.txt",
		"LICENSE",
		"NOTICE.txt",
	} {
		header, ok := tarballMembers[name]
		if !ok {
			t.Fatalf("%s missing in tarball", name)
		}
		if header.Uid != 0 || header.Gid != 0 {
			t.Fatalf("%s has owner %d:%d, want 0:0", name, header.Uid, header.Gid)
		}
	}
	if got := string(readTarGzFile(t, tarballData, "LICENSE")); got != "license\n" {
		t.Fatalf("LICENSE content = %q", got)
	}
}

// TestNfpmPreservesSetuid proves that a setuid mode survives the nfpm files
// package and the deb packager instead of assuming it.
func TestNfpmPreservesSetuid(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "helper")
	if err := os.WriteFile(source, []byte("helper\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	mtime := time.Unix(1234567890, 0).UTC()
	info := &nfpm.Info{
		Name:          "vfox-plus-test",
		Arch:          "amd64",
		Platform:      "linux",
		Version:       "1.2.3",
		VersionSchema: "none",
		MTime:         mtime,
		Overridables: nfpm.Overridables{
			Contents: files.Contents{
				{
					Source:      source,
					Destination: "/usr/lib/vfox-plus/helper",
					FileInfo:    &files.ContentFileInfo{Owner: "root", Group: "root", Mode: os.FileMode(0o755) | os.ModeSetuid, MTime: mtime},
				},
			},
		},
	}
	packager, err := nfpm.Get("deb")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := packager.Package(info, &buf); err != nil {
		t.Fatal(err)
	}
	deb := filepath.Join(dir, "setuid.deb")
	if err := os.WriteFile(deb, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	members := readAr(t, deb)
	var dataData []byte
	for name, content := range members {
		if strings.HasPrefix(name, "data.tar.") {
			dataData = content
		}
	}
	if dataData == nil {
		t.Fatal("data tarball missing")
	}
	header, ok := readTarGz(t, dataData)["./usr/lib/vfox-plus/helper"]
	if !ok {
		t.Fatal("helper missing in data tarball")
	}
	if header.Mode&0o7777 != 0o4755 {
		t.Fatalf("helper mode = %o, want 4755", header.Mode)
	}
}

func TestSetuidBit(t *testing.T) {
	stage, out := fixtureRoot(t)
	runBuild(t,
		"--package", "vfox-plus-test",
		"--version", "1.2.3",
		"--stage", stage,
		"--executable", "vfox-plus",
		"--depends", "libc6",
		"--technology", "testtech",
		"--setuid", "bin/helper",
		out,
	)
	deb := globOne(t, out, "*.deb")
	members := readAr(t, deb)
	var dataData []byte
	for name, content := range members {
		if strings.HasPrefix(name, "data.tar.") {
			dataData = content
		}
	}
	if dataData == nil {
		t.Fatal("data tarball missing")
	}
	data := readTarGz(t, dataData)
	header, ok := data["./usr/lib/vfox-plus/bin/helper"]
	if !ok {
		t.Fatal("helper missing in data tarball")
	}
	if header.Mode&0o7777 != 0o4755 {
		t.Fatalf("helper mode = %o, want 4755", header.Mode)
	}
	plain, ok := data["./usr/lib/vfox-plus/vfox-plus"]
	if !ok {
		t.Fatal("vfox-plus missing in data tarball")
	}
	if plain.Mode&0o7777 != 0o755 {
		t.Fatalf("vfox-plus mode = %o, want 755", plain.Mode)
	}

	tarballData, err := os.ReadFile(globOne(t, out, "*.tar.gz"))
	if err != nil {
		t.Fatal(err)
	}
	tarHeader, ok := readTarGz(t, tarballData)["./bin/helper"]
	if !ok {
		t.Fatal("helper missing in tarball")
	}
	if tarHeader.Mode&0o7777 != 0o4755 {
		t.Fatalf("tarball helper mode = %o, want 4755", tarHeader.Mode)
	}
}

func TestReproducibleBuild(t *testing.T) {
	buildOnce := func(t *testing.T, stamp time.Time) (debBytes, tarballBytes []byte) {
		t.Helper()
		stage, out := fixtureRoot(t)
		// Source timestamps must not leak into the packages.
		stampFile := func(path string) {
			if err := os.Chtimes(path, stamp, stamp); err != nil {
				t.Fatal(err)
			}
		}
		stampFile(stage)
		filepath.Walk(stage, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				t.Fatal(err)
			}
			return os.Chtimes(path, stamp, stamp)
		})
		runBuild(t,
			"--package", "vfox-plus-test",
			"--version", "1.2.3",
			"--stage", stage,
			"--executable", "vfox-plus",
			"--depends", "libc6",
			"--technology", "testtech",
			"--setuid", "bin/helper",
			out,
		)
		debBytes, err := os.ReadFile(globOne(t, out, "*.deb"))
		if err != nil {
			t.Fatal(err)
		}
		tarballBytes, err = os.ReadFile(globOne(t, out, "*.tar.gz"))
		if err != nil {
			t.Fatal(err)
		}
		return debBytes, tarballBytes
	}
	firstDeb, firstTarball := buildOnce(t, time.Unix(1000000000, 0).UTC())
	secondDeb, secondTarball := buildOnce(t, time.Unix(1100000000, 0).UTC())
	if !bytes.Equal(firstDeb, secondDeb) {
		t.Fatal("two builds with the same SOURCE_DATE_EPOCH produced different .deb files")
	}
	if !bytes.Equal(firstTarball, secondTarball) {
		t.Fatal("two builds with the same SOURCE_DATE_EPOCH produced different .tar.gz files")
	}
}
