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

// Package builddeb packages the release build of the application as a .deb
// and a .tar.gz, replacing shared/packaging/linux/build-deb.sh.
//
// The application goes to /usr/lib/vfox-plus and is started as
// /usr/bin/vfox-plus. The "vfox-plus" package conflicts with and replaces
// the old "vfox-plus-mygo" name, so installing it upgrades that package;
// any other implementation package provides, conflicts with and replaces
// "vfox-plus", so that the implementations, which share the application ID
// dev.vfox.VfoxPlus, replace each other. Set DEB_MAINTAINER to "Name <email>"
// for published packages and SOURCE_DATE_EPOCH for reproducible timestamps.
//
// The .deb is built with the nfpm library and the .tar.gz with the standard
// library's archive/tar and compress/gzip; only the desktop-file-validate
// and lintian validators (invoked by the release task, not here) still need
// external tools.
package builddeb

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"debug/elf"
	"encoding/binary"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/goreleaser/nfpm/v2"
	_ "github.com/goreleaser/nfpm/v2/deb"
	"github.com/goreleaser/nfpm/v2/files"
)

const appID = "dev.vfox.VfoxPlus"

const defaultMaintainer = "Han Li and contributors <https://github.com/version-fox/vfox-plus/issues>"

// options mirrors the command line of build-deb.sh.
type options struct {
	pkg        string
	version    string
	stage      string
	executable string
	depends    string
	recommends string
	technology string
	notice     string
	overrides  []string
	setuid     []string
	strip      bool
	output     string
}

const usage = "Usage: build-deb [options] <output directory>"

// Run packages the staged application. args are the command line without the
// program name:
//
//	--package NAME    Debian package name, e.g. vfox-plus (required)
//	--version VERSION version of the application (required)
//	--stage DIRECTORY files of the application, installed to
//	                  /usr/lib/vfox-plus (required)
//	--executable PATH the program in the stage directory that
//	                  /usr/bin/vfox-plus points to (required)
//	--depends LIST    the Depends field (required)
//	--recommends LIST packages to recommend besides vfox
//	--technology TEXT the technology, for the description (required)
//	--notice FILE     third-party licenses, installed as
//	                  /usr/share/doc/<package>/THIRD-PARTY-LICENSES
//	--override TEXT   a lintian override, may be repeated
//	--setuid FILE     a program of the stage that runs as root, such as the
//	                  sandbox helper of Chromium, may be repeated
//	--strip           removes the symbols of the ELF files of the stage, for
//	                  stages that bundle unstripped native code, such as a
//	                  Java runtime
func Run(args []string) error {
	// Package files must not be group-writable, whatever the caller's umask is.
	syscall.Umask(0o022)

	opts, err := parseArgs(args)
	if err != nil {
		return err
	}

	maintainer := os.Getenv("DEB_MAINTAINER")
	if maintainer == "" {
		maintainer = defaultMaintainer
	}

	exePath := filepath.Join(opts.stage, opts.executable)
	if st, err := os.Stat(exePath); err != nil || st.Mode()&0o111 == 0 {
		return fmt.Errorf("%s is not an executable", exePath)
	}
	if opts.notice != "" {
		if st, err := os.Stat(opts.notice); err != nil || !st.Mode().IsRegular() {
			return fmt.Errorf("%s is not a file", opts.notice)
		}
		abs, err := filepath.Abs(opts.notice)
		if err != nil {
			return err
		}
		opts.notice = abs
	}

	debArch, tarArch, err := MapDebArch(unameMachine())
	if err != nil {
		return err
	}

	packagingDir, err := findPackagingDir()
	if err != nil {
		return err
	}
	repo := filepath.Dir(filepath.Dir(filepath.Dir(packagingDir)))

	if err := os.MkdirAll(opts.output, 0o755); err != nil {
		return err
	}
	output, err := filepath.Abs(opts.output)
	if err != nil {
		return err
	}

	temporary, err := os.MkdirTemp("", "build-deb-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temporary)

	// Copy the stage so stripping rewrites the copy, never the input.
	staging := filepath.Join(temporary, "stage")
	if err := copyStage(opts.stage, staging); err != nil {
		return err
	}
	if opts.strip {
		if err := stripELF(staging); err != nil {
			return err
		}
	}

	setuid := make(map[string]bool, len(opts.setuid))
	for _, file := range opts.setuid {
		target := filepath.Join(staging, file)
		if st, err := os.Stat(target); err != nil || !st.Mode().IsRegular() {
			return fmt.Errorf("%s is not in the stage", file)
		}
		setuid[filepath.ToSlash(file)] = true
	}

	epoch := sourceEpoch()
	mtime := time.Unix(epoch, 0).UTC()

	meta := filepath.Join(temporary, "meta")
	if err := os.MkdirAll(meta, 0o755); err != nil {
		return err
	}
	copyrightPath := filepath.Join(meta, "copyright")
	if err := os.WriteFile(copyrightPath, []byte(RenderCopyright(opts.pkg)), 0o644); err != nil {
		return err
	}
	changelog, err := gzipBest(RenderChangelog(opts.pkg, opts.version, maintainer, epoch))
	if err != nil {
		return err
	}
	changelogPath := filepath.Join(meta, "changelog.gz")
	if err := os.WriteFile(changelogPath, changelog, 0o644); err != nil {
		return err
	}
	overridesPath := filepath.Join(meta, "overrides")
	if err := os.WriteFile(overridesPath, []byte(RenderLintianOverrides(opts.pkg, opts.overrides)), 0o644); err != nil {
		return err
	}

	contents, err := buildContents(buildContentsConfig{
		staging:      staging,
		packagingDir: packagingDir,
		pkg:          opts.pkg,
		executable:   opts.executable,
		notice:       opts.notice,
		copyright:    copyrightPath,
		changelog:    changelogPath,
		overrides:    overridesPath,
		setuid:       setuid,
		mtime:        mtime,
	})
	if err != nil {
		return err
	}

	recommends := append([]string{"vfox"}, splitList(opts.recommends)...)
	provides, conflicts, replaces := replacementRelations(opts.pkg)
	info := &nfpm.Info{
		Name:          opts.pkg,
		Arch:          debArch,
		Platform:      "linux",
		Version:       opts.version,
		VersionSchema: "none",
		Section:       "devel",
		Priority:      "optional",
		Maintainer:    maintainer,
		Description:   debDescription(opts.technology),
		Homepage:      "https://github.com/version-fox/vfox-plus",
		MTime:         mtime,
		Overridables: nfpm.Overridables{
			Depends:    splitList(opts.depends),
			Recommends: recommends,
			Provides:   provides,
			Conflicts:  conflicts,
			Replaces:   replaces,
			Contents:   contents,
			Umask:      0o022,
		},
	}

	deb := filepath.Join(output, opts.pkg+"_"+opts.version+"_"+debArch+".deb")
	if err := writeDeb(info, deb); err != nil {
		return err
	}
	tarball := filepath.Join(output, opts.pkg+"-"+opts.version+"-linux-"+tarArch+".tar.gz")
	if err := writeTarball(tarball, staging, repo, opts.notice, setuid, epoch); err != nil {
		return err
	}
	fmt.Println(deb)
	fmt.Println(tarball)
	return nil
}

// parseArgs mirrors the option loop of build-deb.sh: flags are parsed while
// more than one argument remains, and the last argument is the output
// directory.
func parseArgs(args []string) (options, error) {
	var opts options
	rest := args
	for len(rest) > 1 {
		arg := rest[0]
		if arg == "--strip" {
			opts.strip = true
			rest = rest[1:]
			continue
		}
		var dest *string
		var list *[]string
		switch arg {
		case "--package":
			dest = &opts.pkg
		case "--version":
			dest = &opts.version
		case "--stage":
			dest = &opts.stage
		case "--executable":
			dest = &opts.executable
		case "--depends":
			dest = &opts.depends
		case "--recommends":
			dest = &opts.recommends
		case "--technology":
			dest = &opts.technology
		case "--notice":
			dest = &opts.notice
		case "--override":
			list = &opts.overrides
		case "--setuid":
			list = &opts.setuid
		default:
			return opts, fmt.Errorf("Unknown option: %s", arg)
		}
		value := rest[1]
		if dest != nil {
			*dest = value
		} else {
			*list = append(*list, value)
		}
		rest = rest[2:]
	}
	if len(rest) == 0 {
		return opts, fmt.Errorf("%s", usage)
	}
	opts.output = rest[0]
	for _, required := range []struct {
		name  string
		value string
	}{
		{"package", opts.pkg},
		{"version", opts.version},
		{"stage", opts.stage},
		{"executable", opts.executable},
		{"depends", opts.depends},
		{"technology", opts.technology},
	} {
		if required.value == "" {
			return opts, fmt.Errorf("--%s is required", required.name)
		}
	}
	return opts, nil
}

// MapDebArch maps the output of uname -m to the Debian architecture of the
// .deb and the architecture suffix of the tarball.
func MapDebArch(machine string) (debArch, tarArch string, err error) {
	switch machine {
	case "x86_64":
		return "amd64", "x64", nil
	case "aarch64":
		return "arm64", "arm64", nil
	default:
		return "", "", fmt.Errorf("Unsupported architecture: %s", machine)
	}
}

// Control holds the fields of DEBIAN/control.
type Control struct {
	Package       string
	Version       string
	DebArch       string
	Maintainer    string
	InstalledSize string
	Depends       string
	Recommends    string
	Technology    string
}

// replacementRelations returns the Provides/Conflicts/Replaces relations for
// a package name: any other implementation package provides, conflicts with
// and replaces "vfox-plus", while the "vfox-plus" package itself only
// conflicts with and replaces the old "vfox-plus-mygo" name.
func replacementRelations(pkg string) (provides, conflicts, replaces []string) {
	if pkg == "vfox-plus" {
		return nil, []string{"vfox-plus-mygo"}, []string{"vfox-plus-mygo"}
	}
	return []string{"vfox-plus"}, []string{"vfox-plus"}, []string{"vfox-plus"}
}

// RenderControl renders DEBIAN/control.
func RenderControl(c Control) string {
	recommends := "vfox"
	if c.Recommends != "" {
		recommends += ", " + c.Recommends
	}
	provides, conflicts, replaces := replacementRelations(c.Package)
	relations := ""
	for _, relation := range []struct {
		name   string
		values []string
	}{
		{"Provides", provides},
		{"Conflicts", conflicts},
		{"Replaces", replaces},
	} {
		for _, value := range relation.values {
			relations += relation.name + ": " + value + "\n"
		}
	}
	return fmt.Sprintf(`Package: %s
Version: %s
Section: devel
Priority: optional
Architecture: %s
Maintainer: %s
Installed-Size: %s
Depends: %s
Recommends: %s
%sHomepage: https://github.com/version-fox/vfox-plus
Description: Material 3 Expressive desktop GUI for vfox (%s)
 vfox+ shows the installed SDKs and the versions in use, installs and
 uninstalls versions, pins versions in projects, browses the vfox plugin
 registry and edits the vfox configuration. It drives the vfox command line
 tool, which must be installed separately.
 .
 This package is the implementation of vfox+ built with %s.
`, c.Package, c.Version, c.DebArch, c.Maintainer, c.InstalledSize, c.Depends, recommends, relations, c.Technology, c.Technology)
}

// RenderCopyright renders /usr/share/doc/<package>/copyright.
func RenderCopyright(pkg string) string {
	return fmt.Sprintf(`Format: https://www.debian.org/doc/packaging-manuals/copyright-format/1.0/
Upstream-Name: vfox-plus
Source: https://github.com/version-fox/vfox-plus

Files: *
Copyright: 2026 Han Li and contributors
License: Apache-2.0
Comment: The licenses of the bundled third-party components, including a subset
 of the Material Symbols Rounded font (Apache-2.0, Google LLC), are listed in
 /usr/share/doc/%s/THIRD-PARTY-LICENSES when that file exists.

License: Apache-2.0
 On Debian systems, the full text of the Apache License 2.0 can be found in
 /usr/share/common-licenses/Apache-2.0.
`, pkg)
}

// RenderChangelog renders the plain text of changelog.gz. The date comes from
// SOURCE_DATE_EPOCH, like `date -R -d "@$epoch"` in the script.
func RenderChangelog(pkg, version, maintainer string, epoch int64) string {
	date := time.Unix(epoch, 0).Format("Mon, 02 Jan 2006 15:04:05 -0700")
	return fmt.Sprintf(`%s (%s) unstable; urgency=medium

  * Release %s. See https://github.com/version-fox/vfox-plus/releases.

 -- %s  %s
`, pkg, version, version, maintainer, date)
}

// RenderLintianOverrides renders /usr/share/lintian/overrides/<package>.
func RenderLintianOverrides(pkg string, overrides []string) string {
	var b strings.Builder
	b.WriteString("# A graphical application without command line options.\n")
	b.WriteString(pkg + ": no-manual-page [usr/bin/vfox-plus]\n")
	for _, override := range overrides {
		b.WriteString(pkg + ": " + override + "\n")
	}
	return b.String()
}

// debDescription renders the Description field of the .deb control file. It
// holds the same text as RenderControl so that nfpm's multiline formatter
// produces the same paragraph.
func debDescription(technology string) string {
	return fmt.Sprintf(`Material 3 Expressive desktop GUI for vfox (%s)
vfox+ shows the installed SDKs and the versions in use, installs and
uninstalls versions, pins versions in projects, browses the vfox plugin
registry and edits the vfox configuration. It drives the vfox command line
tool, which must be installed separately.

This package is the implementation of vfox+ built with %s.`, technology, technology)
}

// splitList splits a comma-separated control field into its entries.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// unameMachine returns the output of uname -m, trimmed of whitespace.
func unameMachine() string {
	out, err := exec.Command("uname", "-m").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// sourceEpoch returns SOURCE_DATE_EPOCH, or the current time when unset,
// like ${SOURCE_DATE_EPOCH:-$(date +%s)} in the script.
func sourceEpoch() int64 {
	if s := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); s != "" {
		if epoch, err := strconv.ParseInt(s, 10, 64); err == nil {
			return epoch
		}
	}
	return time.Now().Unix()
}

// findPackagingDir locates shared/packaging/linux, which holds the desktop
// entry, the metainfo and the icon. It searches upward from the working
// directory, so the tool works both on the host (from a repository checkout)
// and in a sandbox container (from a copied repository).
func findPackagingDir() (string, error) {
	if dir := os.Getenv("VFOX_PLUS_PACKAGING_DIR"); dir != "" {
		if _, err := os.Stat(filepath.Join(dir, appID+".desktop")); err == nil {
			return dir, nil
		}
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "shared", "packaging", "linux")
		if _, err := os.Stat(filepath.Join(candidate, appID+".desktop")); err == nil {
			return candidate, nil
		}
		// The working directory may itself be the packaging directory.
		if _, err := os.Stat(filepath.Join(dir, appID+".desktop")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, appID+".metainfo.xml")); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
	}
	for _, candidate := range []string{"/src/shared/packaging/linux"} {
		if _, err := os.Stat(filepath.Join(candidate, appID+".desktop")); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("cannot find %s.desktop: set VFOX_PLUS_PACKAGING_DIR", appID)
}

// setuidMode is the mode of --setuid programs, which run as root.
const setuidMode = os.FileMode(0o755) | os.ModeSetuid

// normalizeMode mirrors `chmod -R u+rwX,go+rX,go-w`: the owner always gains
// read and write permission, execute bits are added only where something
// already executes (or for directories), group and others gain read and lose
// write permission, and setuid/setgid/sticky bits are preserved.
func normalizeMode(m os.FileMode, isDir bool) os.FileMode {
	perm := m.Perm()
	perm |= 0o600
	if isDir || perm&0o111 != 0 {
		perm |= 0o111
	}
	perm |= 0o044
	perm &^= 0o022
	out := os.FileMode(perm)
	if m&os.ModeSetuid != 0 {
		out |= os.ModeSetuid
	}
	if m&os.ModeSetgid != 0 {
		out |= os.ModeSetgid
	}
	if m&os.ModeSticky != 0 {
		out |= os.ModeSticky
	}
	return out
}

// headerMode converts a file mode to the raw mode bits of a tar header.
func headerMode(m os.FileMode) int64 {
	mode := int64(m.Perm())
	if m&os.ModeSetuid != 0 {
		mode |= 0o4000
	}
	if m&os.ModeSetgid != 0 {
		mode |= 0o2000
	}
	if m&os.ModeSticky != 0 {
		mode |= 0o1000
	}
	return mode
}

// copyStage copies the stage directory to dst, like `cp -a`, preserving
// symlinks and modes (normalized like the chmod step of the script).
func copyStage(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		switch {
		case d.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			return os.Symlink(target, out)
		case d.IsDir():
			return os.MkdirAll(out, 0o755)
		case d.Type().IsRegular():
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
				return err
			}
			if err := os.WriteFile(out, data, 0o600); err != nil {
				return err
			}
			return os.Chmod(out, normalizeMode(info.Mode(), false))
		default:
			return fmt.Errorf("%s is not a regular file", path)
		}
	})
}

// stripELF strips every ELF file under dir, like the --strip loop of the
// script, which asked `file` about each regular file. ELF files are detected
// by their magic number and stripped with the standard library, so no
// external toolchain is needed.
func stripELF(dir string) error {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode().IsRegular() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, file := range files {
		isELF, err := isELFMagic(file)
		if err != nil {
			return err
		}
		if !isELF {
			continue
		}
		if err := stripUnneeded(file); err != nil {
			return err
		}
	}
	return nil
}

// isELFMagic reports whether the file starts with the ELF magic number.
func isELFMagic(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return false, nil
		}
		return false, err
	}
	return magic == [4]byte{0x7f, 'E', 'L', 'F'}, nil
}

// stripUnneeded removes the symbols and debug information of an ELF file,
// like `strip --strip-unneeded`: section headers of sections that are not
// loaded at runtime (except the section name string table) are nulled and
// their contents are zeroed. Program headers, and therefore the running
// program, are untouched.
func stripUnneeded(path string) error {
	ef, err := elf.Open(path)
	if err != nil {
		return fmt.Errorf("elf %s: %w", path, err)
	}
	defer ef.Close()
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var order binary.ByteOrder
	switch ef.Data {
	case elf.ELFDATA2LSB:
		order = binary.LittleEndian
	case elf.ELFDATA2MSB:
		order = binary.BigEndian
	default:
		return fmt.Errorf("elf %s: unknown data encoding", path)
	}
	var shoff, shentsize, shnum, shstrndx uint64
	switch ef.Class {
	case elf.ELFCLASS32:
		if len(data) < 0x34 {
			return fmt.Errorf("elf %s: truncated header", path)
		}
		shoff = uint64(order.Uint32(data[0x20:]))
		shentsize = uint64(order.Uint16(data[0x2e:]))
		shnum = uint64(order.Uint16(data[0x30:]))
		shstrndx = uint64(order.Uint16(data[0x32:]))
	case elf.ELFCLASS64:
		if len(data) < 0x40 {
			return fmt.Errorf("elf %s: truncated header", path)
		}
		shoff = order.Uint64(data[0x28:])
		shentsize = uint64(order.Uint16(data[0x3a:]))
		shnum = uint64(order.Uint16(data[0x3c:]))
		shstrndx = uint64(order.Uint16(data[0x3e:]))
	default:
		return fmt.Errorf("elf %s: unknown class", path)
	}
	if shoff == 0 || shnum == 0 || shentsize == 0 {
		return nil
	}
	if shoff+shnum*shentsize > uint64(len(data)) {
		return fmt.Errorf("elf %s: section table out of bounds", path)
	}
	changed := false
	for i, sec := range ef.Sections {
		if uint64(i) >= shnum {
			break
		}
		if uint64(i) == shstrndx {
			continue
		}
		if sec.Flags&elf.SHF_ALLOC != 0 {
			continue
		}
		if sec.Type == elf.SHT_NULL {
			continue
		}
		if sec.Type != elf.SHT_NOBITS && sec.Size > 0 {
			if sec.Offset+sec.Size > uint64(len(data)) {
				return fmt.Errorf("elf %s: section %q out of bounds", path, sec.Name)
			}
			clear(data[sec.Offset : sec.Offset+sec.Size])
		}
		order.PutUint32(data[shoff+uint64(i)*shentsize+4:], uint32(elf.SHT_NULL))
		changed = true
	}
	if !changed {
		return nil
	}
	return os.WriteFile(path, data, 0o600)
}

// buildContentsConfig holds the inputs of buildContents.
type buildContentsConfig struct {
	staging      string
	packagingDir string
	pkg          string
	executable   string
	notice       string
	copyright    string
	changelog    string
	overrides    string
	setuid       map[string]bool
	mtime        time.Time
}

// docFile describes a generated file installed outside /usr/lib/vfox-plus.
type docFile struct {
	source      string
	destination string
}

// buildContents assembles the nfpm file tree: the staged application under
// /usr/lib/vfox-plus, the /usr/bin/vfox-plus symlink, the desktop entry,
// the metainfo, the icon and the documentation. Every entry carries an
// explicit owner, mode and timestamp so the .deb is reproducible.
func buildContents(cfg buildContentsConfig) (files.Contents, error) {
	root := func(owner, group string, mode os.FileMode) *files.ContentFileInfo {
		return &files.ContentFileInfo{Owner: owner, Group: group, Mode: mode, MTime: cfg.mtime}
	}
	var contents []*files.Content
	err := filepath.WalkDir(cfg.staging, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(cfg.staging, path)
		if err != nil {
			return err
		}
		if rel == "." {
			contents = append(contents, &files.Content{
				Destination: "/usr/lib/vfox-plus",
				Type:        files.TypeDir,
				FileInfo:    root("root", "root", 0o755),
			})
			return nil
		}
		slash := filepath.ToSlash(rel)
		destination := "/usr/lib/vfox-plus/" + slash
		switch {
		case d.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			contents = append(contents, &files.Content{
				Source:      filepath.ToSlash(target),
				Destination: destination,
				Type:        files.TypeSymlink,
				FileInfo:    root("root", "root", 0o777),
			})
		case d.IsDir():
			info, err := d.Info()
			if err != nil {
				return err
			}
			contents = append(contents, &files.Content{
				Destination: destination,
				Type:        files.TypeDir,
				FileInfo:    root("root", "root", normalizeMode(info.Mode(), true)),
			})
		case d.Type().IsRegular():
			info, err := d.Info()
			if err != nil {
				return err
			}
			mode := normalizeMode(info.Mode(), false)
			if cfg.setuid[slash] {
				mode = setuidMode
			}
			contents = append(contents, &files.Content{
				Source:      path,
				Destination: destination,
				FileInfo:    root("root", "root", mode),
			})
		default:
			return fmt.Errorf("%s is not a regular file", path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	docs := []docFile{
		{filepath.Join(cfg.packagingDir, appID+".desktop"), "/usr/share/applications/" + appID + ".desktop"},
		{filepath.Join(cfg.packagingDir, appID+".metainfo.xml"), "/usr/share/metainfo/" + appID + ".metainfo.xml"},
		{filepath.Join(cfg.packagingDir, appID+".svg"), "/usr/share/icons/hicolor/scalable/apps/" + appID + ".svg"},
		{cfg.copyright, "/usr/share/doc/" + cfg.pkg + "/copyright"},
		{cfg.changelog, "/usr/share/doc/" + cfg.pkg + "/changelog.gz"},
		{cfg.overrides, "/usr/share/lintian/overrides/" + cfg.pkg},
	}
	if cfg.notice != "" {
		docs = append(docs, docFile{cfg.notice, "/usr/share/doc/" + cfg.pkg + "/THIRD-PARTY-LICENSES"})
	}
	for _, doc := range docs {
		contents = append(contents, &files.Content{
			Source:      doc.source,
			Destination: doc.destination,
			FileInfo:    root("root", "root", 0o644),
		})
	}
	contents = append(contents, &files.Content{
		Source:      "../lib/vfox-plus/" + cfg.executable,
		Destination: "/usr/bin/vfox-plus",
		Type:        files.TypeSymlink,
		FileInfo:    root("root", "root", 0o777),
	})
	return contents, nil
}

// writeDeb builds the .deb with the nfpm library.
func writeDeb(info *nfpm.Info, debPath string) error {
	packager, err := nfpm.Get("deb")
	if err != nil {
		return err
	}
	out, err := os.Create(debPath)
	if err != nil {
		return err
	}
	if err := packager.Package(info, out); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// writeTarball builds the .tar.gz with the standard library, holding the
// same entries as the `tar --owner=0 --group=0` invocation of the script:
// the stage tree, the LICENSE of the repository and the notice file.
func writeTarball(tarball, staging, repo, notice string, setuid map[string]bool, epoch int64) error {
	license := filepath.Join(repo, "LICENSE")
	licenseInfo, err := os.Stat(license)
	if err != nil || !licenseInfo.Mode().IsRegular() {
		return fmt.Errorf("LICENSE is not in %s", repo)
	}
	mtime := time.Unix(epoch, 0).UTC()
	type entry struct {
		header *tar.Header
		source string
	}
	var entries []entry
	add := func(name string, mode os.FileMode, size int64, typeflag byte, linkname, source string) {
		entries = append(entries, entry{&tar.Header{
			Name:     name,
			Mode:     headerMode(mode),
			Size:     size,
			ModTime:  mtime,
			Typeflag: typeflag,
			Linkname: linkname,
			Uid:      0,
			Gid:      0,
			Uname:    "root",
			Gname:    "root",
			Format:   tar.FormatGNU,
		}, source})
	}
	add("./", 0o755, 0, tar.TypeDir, "", "")
	err = filepath.WalkDir(staging, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(staging, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		slash := filepath.ToSlash(rel)
		switch {
		case d.Type()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			add("./"+slash, 0o777, 0, tar.TypeSymlink, target, "")
		case d.IsDir():
			info, err := d.Info()
			if err != nil {
				return err
			}
			add("./"+slash+"/", normalizeMode(info.Mode(), true), 0, tar.TypeDir, "", "")
		case d.Type().IsRegular():
			info, err := d.Info()
			if err != nil {
				return err
			}
			mode := normalizeMode(info.Mode(), false)
			if setuid[slash] {
				mode = setuidMode
			}
			add("./"+slash, mode, info.Size(), tar.TypeReg, "", path)
		default:
			return fmt.Errorf("%s is not a regular file", path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	add("LICENSE", licenseInfo.Mode(), licenseInfo.Size(), tar.TypeReg, "", license)
	if notice != "" {
		noticeInfo, err := os.Stat(notice)
		if err != nil {
			return err
		}
		add(filepath.ToSlash(filepath.Base(notice)), noticeInfo.Mode(), noticeInfo.Size(), tar.TypeReg, "", notice)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].header.Name < entries[j].header.Name })

	out, err := os.Create(tarball)
	if err != nil {
		return err
	}
	gz, err := gzip.NewWriterLevel(out, gzip.BestCompression)
	if err != nil {
		out.Close()
		return err
	}
	gz.Header.ModTime = mtime
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		if err := tw.WriteHeader(e.header); err != nil {
			tw.Close()
			gz.Close()
			out.Close()
			return err
		}
		if e.source != "" {
			f, err := os.Open(e.source)
			if err != nil {
				tw.Close()
				gz.Close()
				out.Close()
				return err
			}
			_, copyErr := io.Copy(tw, f)
			closeErr := f.Close()
			if copyErr != nil {
				tw.Close()
				gz.Close()
				out.Close()
				return copyErr
			}
			if closeErr != nil {
				tw.Close()
				gz.Close()
				out.Close()
				return closeErr
			}
		}
	}
	if err := tw.Close(); err != nil {
		gz.Close()
		out.Close()
		return err
	}
	if err := gz.Close(); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// gzipBest compresses data like gzip -9n: best compression without a file
// name or timestamp in the header, so the output is reproducible.
func gzipBest(data string) ([]byte, error) {
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	w.Header.ModTime = time.Unix(0, 0).UTC()
	if _, err := w.Write([]byte(data)); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
