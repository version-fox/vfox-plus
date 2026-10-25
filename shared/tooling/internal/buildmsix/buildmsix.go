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

// Package buildmsix packages Windows release builds as unsigned .msix files
// (one per architecture) and a .msixbundle for Windows 11.
//
// An MSIX package is an OPC ZIP archive: AppxManifest.xml, [Content_Types].xml
// and AppxBlockMap.xml describe the payload (the executable and its Assets).
// An MSIX bundle is another OPC ZIP that contains the .msix files together
// with AppxMetadata/AppxBundleManifest.xml listing them with their byte
// offsets and sizes.
//
// The packages are unsigned: Windows 11 installs them with
// Add-AppxPackage -AllowUnsigned (which needs administrator rights for
// packages with executable content). Unsigned packages must carry the special
// OID in their Publisher (see the Microsoft documentation on unsigned
// packages); signed packages remove it and use the certificate subject
// instead. The tool never signs: to distribute widely, sign the .msix files
// and the bundle with a trusted certificate.
//
// Everything is pure Go with no Windows SDK: the .msix files are built by
// cross-compiling (GOOS=windows GOARCH=amd64/arm64 CGO_ENABLED=0 go build)
// and the ZIPs use no compression (Store), fixed timestamps from
// SOURCE_DATE_EPOCH and sorted entries, so the output is reproducible.
package buildmsix

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const appID = "dev.vfox.VfoxPlus"

// UnsignedPublisherOID is the special OID that unsigned packages must carry
// in their Publisher so that Windows accepts them for -AllowUnsigned installs
// without conflicting with a signed package of the same identity.
const unsignedPublisherOID = "OID.2.25.311729368913984317654407730594956997722=1"

// DefaultPublisher is the Publisher of unsigned packages.
const defaultPublisher = "CN=vfox-plus, " + unsignedPublisherOID

// BlockSize is the block size of AppxBlockMap.xml: every file is split into
// 64 KiB blocks, each hashed with SHA-256.
const blockSize = 65536

// options mirrors the command line.
type options struct {
	pkg                  string
	version              string
	identity             string
	publisher            string
	displayName          string
	publisherDisplayName string
	description          string
	stageX64             string
	stageArm64           string
	executable           string
	output               string
}

const usage = "Usage: build-msix [options] <output directory>"

func defaultOptions() options {
	return options{
		identity:             appID,
		publisher:            defaultPublisher,
		displayName:          "vfox+",
		publisherDisplayName: "Han Li and contributors",
		description:          "Install and switch SDK versions with vfox",
		executable:           "vfox-plus.exe",
	}
}

// Run packages staged Windows executables as .msix and .msixbundle. args are
// the command line without the program name:
//
//	--package NAME            package file name prefix, e.g. vfox-plus (required)
//	--version VERSION         application version, e.g. 0.1.0 (required; becomes 0.1.0.0)
//	--identity ID             package identity name (default dev.vfox.VfoxPlus)
//	--publisher PUBLISHER     package publisher (default unsigned CN with the OID)
//	--display-name NAME       display name (default vfox+)
//	--publisher-display-name NAME publisher display name
//	--description TEXT        package description
//	--stage-x64 DIRECTORY     directory holding the x64 executable
//	--stage-arm64 DIRECTORY   directory holding the arm64 executable
//	--executable FILE         executable file name in the stages (default vfox-plus.exe)
//	<output directory>        where the .msix and .msixbundle are written
//
// At least one of --stage-x64 and --stage-arm64 is required. Each stage must
// hold the executable; the Assets logos are generated from the brand colour.
func Run(args []string) error {
	opts := defaultOptions()
	rest := args
	for len(rest) > 1 {
		arg := rest[0]
		var dest *string
		switch arg {
		case "--package":
			dest = &opts.pkg
		case "--version":
			dest = &opts.version
		case "--identity":
			dest = &opts.identity
		case "--publisher":
			dest = &opts.publisher
		case "--display-name":
			dest = &opts.displayName
		case "--publisher-display-name":
			dest = &opts.publisherDisplayName
		case "--description":
			dest = &opts.description
		case "--stage-x64":
			dest = &opts.stageX64
		case "--stage-arm64":
			dest = &opts.stageArm64
		case "--executable":
			dest = &opts.executable
		default:
			return fmt.Errorf("Unknown option: %s", arg)
		}
		*dest = rest[1]
		rest = rest[2:]
	}
	if len(rest) == 0 {
		return fmt.Errorf("%s", usage)
	}
	opts.output = rest[0]
	if len(rest) > 1 {
		return fmt.Errorf("Too many arguments: %q", rest[1:])
	}
	return runWithOptions(opts)
}

func runWithOptions(opts options) error {
	if opts.pkg == "" {
		return fmt.Errorf("--package is required")
	}
	if opts.version == "" {
		return fmt.Errorf("--version is required")
	}
	if opts.stageX64 == "" && opts.stageArm64 == "" {
		return fmt.Errorf("at least one of --stage-x64 and --stage-arm64 is required")
	}
	quad, err := ToQuadVersion(opts.version)
	if err != nil {
		return err
	}
	epoch := sourceEpoch()
	mtime := time.Unix(epoch, 0).UTC()
	if err := os.MkdirAll(opts.output, 0o755); err != nil {
		return err
	}
	output, err := filepath.Abs(opts.output)
	if err != nil {
		return err
	}
	var packages []builtPackage
	for _, arch := range []string{"x64", "arm64"} {
		stage := opts.stageX64
		if arch == "arm64" {
			stage = opts.stageArm64
		}
		if stage == "" {
			continue
		}
		exePath := filepath.Join(stage, opts.executable)
		exe, err := os.ReadFile(exePath) //nolint:gosec // The caller stages the build output.
		if err != nil {
			return fmt.Errorf("%s: %w", exePath, err)
		}
		if len(exe) < 2 || exe[0] != 'M' || exe[1] != 'Z' {
			return fmt.Errorf("%s is not a Windows executable (missing MZ header)", exePath)
		}
		msix, err := WriteMsix(MsixConfig{
			Identity:             opts.identity,
			Publisher:            opts.publisher,
			Version:              quad,
			Architecture:         arch,
			DisplayName:          opts.displayName,
			PublisherDisplayName: opts.publisherDisplayName,
			Description:          opts.description,
			Executable:           opts.executable,
			ExecutableData:       exe,
			MTime:                mtime,
		})
		if err != nil {
			return err
		}
		filename := fmt.Sprintf("%s_%s_%s.msix", opts.pkg, opts.version, arch)
		if err := os.WriteFile(filepath.Join(output, filename), msix, 0o644); err != nil {
			return err
		}
		packages = append(packages, builtPackage{arch: arch, filename: filename, data: msix})
		fmt.Println(filepath.Join(output, filename))
	}
	bundle, bundleName, err := WriteBundle(BundleConfig{
		Package:   opts.pkg,
		Version:   opts.version,
		Identity:  opts.identity,
		Publisher: opts.publisher,
		Packages:  bundleInputs(packages),
		MTime:     mtime,
	})
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, bundleName), bundle, 0o644); err != nil {
		return err
	}
	fmt.Println(filepath.Join(output, bundleName))
	return nil
}

// builtPackage is one .msix built for runWithOptions before bundling.
type builtPackage struct {
	arch     string
	filename string
	data     []byte
}

func bundleInputs(packages []builtPackage) []BundlePackage {
	var out []BundlePackage
	for _, p := range packages {
		out = append(out, BundlePackage{Architecture: p.arch, Filename: p.filename, Data: p.data})
	}
	return out
}

// MapMsixArch maps a Go architecture (GOARCH) to the MSIX architecture name
// of the .msix package.
func MapMsixArch(goarch string) (string, error) {
	switch goarch {
	case "amd64", "x64":
		return "x64", nil
	case "arm64":
		return "arm64", nil
	case "386", "x86":
		return "x86", nil
	default:
		return "", fmt.Errorf("Unsupported architecture for MSIX: %s", goarch)
	}
}

// ToQuadVersion converts a dotted version to the quad notation MSIX requires
// (Major.Minor.Build.Revision). "0.1.0" becomes "0.1.0.0".
func ToQuadVersion(version string) (string, error) {
	version = strings.TrimSpace(version)
	parts := strings.Split(version, ".")
	if len(parts) < 1 || len(parts) > 4 {
		return "", fmt.Errorf("Not a valid MSIX version: %s", version)
	}
	for _, part := range parts {
		if part == "" {
			return "", fmt.Errorf("Not a valid MSIX version: %s", version)
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 || n > 65535 {
			return "", fmt.Errorf("Not a valid MSIX version: %s", version)
		}
	}
	for len(parts) < 4 {
		parts = append(parts, "0")
	}
	return strings.Join(parts, "."), nil
}

// MsixConfig holds the inputs of WriteMsix.
type MsixConfig struct {
	Identity             string
	Publisher            string
	Version              string // quad notation
	Architecture         string // x64 or arm64
	DisplayName          string
	PublisherDisplayName string
	Description          string
	Executable           string // e.g. vfox-plus.exe
	ExecutableData       []byte
	MTime                time.Time
}

// BundlePackage is one .msix embedded in a bundle.
type BundlePackage struct {
	Architecture string
	Filename     string
	Data         []byte
}

// BundleConfig holds the inputs of WriteBundle.
type BundleConfig struct {
	Package   string // file name prefix
	Version   string // original version, e.g. 0.1.0
	Identity  string
	Publisher string
	Packages  []BundlePackage
	MTime     time.Time
}

// RenderAppxManifest renders AppxManifest.xml of one .msix package.
func RenderAppxManifest(cfg MsixConfig) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10" xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10" xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities" IgnorableNamespaces="uap rescap">
  <Identity Name="%s" Publisher="%s" Version="%s" ProcessorArchitecture="%s"/>
  <Properties>
    <DisplayName>%s</DisplayName>
    <PublisherDisplayName>%s</PublisherDisplayName>
    <Description>%s</Description>
    <Logo>Assets\StoreLogo.png</Logo>
  </Properties>
  <Resources>
    <Resource Language="en-us"/>
    <Resource Language="zh-cn"/>
  </Resources>
  <Dependencies>
    <TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.22000.0" MaxVersionTested="10.0.26100.0"/>
  </Dependencies>
  <Capabilities>
    <rescap:Capability Name="runFullTrust"/>
  </Capabilities>
  <Applications>
    <Application Id="App" Executable="%s" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements DisplayName="%s" Description="%s" BackgroundColor="transparent" Square150x150Logo="Assets\Square150x150Logo.png" Square44x44Logo="Assets\Square44x44Logo.png">
        <uap:DefaultTile Wide310x150Logo="Assets\Wide310x150Logo.png"/>
      </uap:VisualElements>
    </Application>
  </Applications>
</Package>
`, xmlEscape(cfg.Identity), xmlEscape(cfg.Publisher), xmlEscape(cfg.Version), xmlEscape(cfg.Architecture), xmlEscape(cfg.DisplayName), xmlEscape(cfg.PublisherDisplayName), xmlEscape(cfg.Description), xmlEscape(cfg.Executable), xmlEscape(cfg.DisplayName), xmlEscape(cfg.Description))
}

// RenderContentTypes renders [Content_Types].xml of a .msix package.
func RenderContentTypes() string {
	return `<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="exe" ContentType="application/x-msdownload"/>
  <Default Extension="png" ContentType="image/png"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/AppxManifest.xml" ContentType="application/vnd.ms-appx.manifest+xml"/>
  <Override PartName="/AppxBlockMap.xml" ContentType="application/vnd.ms-appx.blockmap+xml"/>
</Types>
`
}

// RenderBundleContentTypes renders [Content_Types].xml of a .msixbundle.
func RenderBundleContentTypes(packages []BundlePackage) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
`)
	for _, p := range packages {
		fmt.Fprintf(&b, "  <Override PartName=\"/%s\" ContentType=\"application/vnd.ms-appx.msix\"/>\n", xmlEscape(p.Filename))
	}
	b.WriteString(`  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/AppxMetadata/AppxBundleManifest.xml" ContentType="application/vnd.ms-appx.bundlemanifest+xml"/>
  <Override PartName="/AppxBlockMap.xml" ContentType="application/vnd.ms-appx.blockmap+xml"/>
</Types>
`)
	return b.String()
}

// RenderBundleManifest renders AppxMetadata/AppxBundleManifest.xml. Offsets
// and sizes are the byte offsets of each .msix payload in the bundle file.
func RenderBundleManifest(identity, publisher, version string, entries []BundleEntry) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8" standalone="no"?>` + "\n")
	b.WriteString(`<Bundle SchemaVersion="2.0" xmlns="http://schemas.microsoft.com/appx/2013/bundle">` + "\n")
	fmt.Fprintf(&b, "  <Identity Name=\"%s\" Publisher=\"%s\" Version=\"%s\"/>\n", xmlEscape(identity), xmlEscape(publisher), xmlEscape(version))
	b.WriteString("  <Packages>\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "    <Package Type=\"application\" Version=\"%s\" Architecture=\"%s\" FileName=\"%s\" Offset=\"%d\" Size=\"%d\">\n", xmlEscape(version), xmlEscape(e.Architecture), xmlEscape(e.Filename), e.Offset, e.Size)
		b.WriteString("      <Resources>\n")
		b.WriteString("        <Resource Language=\"en-us\"/>\n")
		b.WriteString("        <Resource Language=\"zh-cn\"/>\n")
		b.WriteString("      </Resources>\n")
		b.WriteString("    </Package>\n")
	}
	b.WriteString("  </Packages>\n")
	b.WriteString("</Bundle>\n")
	return b.String()
}

// BundleEntry is one package listed in the bundle manifest.
type BundleEntry struct {
	Architecture string
	Filename     string
	Offset       int64
	Size         int64
}

func xmlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '"':
			b.WriteString("&quot;")
		case '\'':
			b.WriteString("&apos;")
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// sourceEpoch returns SOURCE_DATE_EPOCH, or the current time when unset.
func sourceEpoch() int64 {
	if s := strings.TrimSpace(os.Getenv("SOURCE_DATE_EPOCH")); s != "" {
		if epoch, err := strconv.ParseInt(s, 10, 64); err == nil {
			return epoch
		}
	}
	return time.Now().Unix()
}

// countingWriter counts the bytes written through it.
type countingWriter struct {
	w ioWriter
	n int64
}

type ioWriter interface {
	Write([]byte) (int, error)
}

func (c *countingWriter) Write(p []byte) (int, error) {
	n, err := c.w.Write(p)
	c.n += int64(n)
	return n, err
}

// zipEntry is one file of a deterministic ZIP.
type zipEntry struct {
	name string
	data []byte
	mode os.FileMode
}

// writeZip writes entries sorted by name with fixed timestamps and no
// compression (Store). Headers carry the real sizes upfront (no data
// descriptor), so parseLocalHeaders can read them back and Windows accepts
// the OPC packages.
func writeZip(entries []zipEntry, mtime time.Time) ([]byte, error) {
	sorted := append([]zipEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].name < sorted[j].name })
	dosTime, dosDate := dosDateTime(mtime)
	type central struct {
		name      string
		crc       uint32
		size      uint32
		headerOff int64
		external  uint32
	}
	var buf bytes.Buffer
	var centrals []central
	for _, e := range sorted {
		if len(e.data) > 0xFFFFFFFF {
			return nil, fmt.Errorf("%s too large", e.name)
		}
		nameBytes := []byte(e.name)
		crc := crc32.ChecksumIEEE(e.data)
		size := uint32(len(e.data))
		headerOff := int64(buf.Len())
		var h bytes.Buffer
		binary.Write(&h, binary.LittleEndian, uint32(0x04034b50))
		binary.Write(&h, binary.LittleEndian, uint16(20))
		binary.Write(&h, binary.LittleEndian, uint16(0x800))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, dosTime)
		binary.Write(&h, binary.LittleEndian, dosDate)
		binary.Write(&h, binary.LittleEndian, crc)
		binary.Write(&h, binary.LittleEndian, size)
		binary.Write(&h, binary.LittleEndian, size)
		binary.Write(&h, binary.LittleEndian, uint16(len(nameBytes)))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		h.Write(nameBytes)
		buf.Write(h.Bytes())
		buf.Write(e.data)
		centrals = append(centrals, central{name: e.name, crc: crc, size: size, headerOff: headerOff, external: uint32(e.mode.Perm())<<16 | 0x20})
	}
	centralOff := int64(buf.Len())
	for _, c := range centrals {
		nameBytes := []byte(c.name)
		var h bytes.Buffer
		binary.Write(&h, binary.LittleEndian, uint32(0x02014b50))
		binary.Write(&h, binary.LittleEndian, uint16(0x031E))
		binary.Write(&h, binary.LittleEndian, uint16(20))
		binary.Write(&h, binary.LittleEndian, uint16(0x800))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, dosTime)
		binary.Write(&h, binary.LittleEndian, dosDate)
		binary.Write(&h, binary.LittleEndian, c.crc)
		binary.Write(&h, binary.LittleEndian, c.size)
		binary.Write(&h, binary.LittleEndian, c.size)
		binary.Write(&h, binary.LittleEndian, uint16(len(nameBytes)))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, uint32(c.external))
		binary.Write(&h, binary.LittleEndian, uint32(c.headerOff))
		h.Write(nameBytes)
		buf.Write(h.Bytes())
	}
	centralSize := int64(buf.Len()) - centralOff
	var end bytes.Buffer
	binary.Write(&end, binary.LittleEndian, uint32(0x06054b50))
	binary.Write(&end, binary.LittleEndian, uint16(0))
	binary.Write(&end, binary.LittleEndian, uint16(0))
	binary.Write(&end, binary.LittleEndian, uint16(len(centrals)))
	binary.Write(&end, binary.LittleEndian, uint16(len(centrals)))
	binary.Write(&end, binary.LittleEndian, uint32(centralSize))
	binary.Write(&end, binary.LittleEndian, uint32(centralOff))
	binary.Write(&end, binary.LittleEndian, uint16(0))
	buf.Write(end.Bytes())
	return buf.Bytes(), nil
}

// writeZipOrdered writes entries in the given order (no sorting) with the
// same deterministic Store layout as writeZip. Bundles use it to keep payload
// .msix files before the manifest, so the manifest offsets are stable.
func writeZipOrdered(entries []zipEntry, mtime time.Time) ([]byte, error) {
	dosTime, dosDate := dosDateTime(mtime)
	type central struct {
		name      string
		crc       uint32
		size      uint32
		headerOff int64
		external  uint32
	}
	var buf bytes.Buffer
	var centrals []central
	for _, e := range entries {
		if len(e.data) > 0xFFFFFFFF {
			return nil, fmt.Errorf("%s too large", e.name)
		}
		nameBytes := []byte(e.name)
		crc := crc32.ChecksumIEEE(e.data)
		size := uint32(len(e.data))
		headerOff := int64(buf.Len())
		var h bytes.Buffer
		binary.Write(&h, binary.LittleEndian, uint32(0x04034b50))
		binary.Write(&h, binary.LittleEndian, uint16(20))
		binary.Write(&h, binary.LittleEndian, uint16(0x800))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, dosTime)
		binary.Write(&h, binary.LittleEndian, dosDate)
		binary.Write(&h, binary.LittleEndian, crc)
		binary.Write(&h, binary.LittleEndian, size)
		binary.Write(&h, binary.LittleEndian, size)
		binary.Write(&h, binary.LittleEndian, uint16(len(nameBytes)))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		h.Write(nameBytes)
		buf.Write(h.Bytes())
		buf.Write(e.data)
		centrals = append(centrals, central{name: e.name, crc: crc, size: size, headerOff: headerOff, external: uint32(e.mode.Perm())<<16 | 0x20})
	}
	centralOff := int64(buf.Len())
	for _, c := range centrals {
		nameBytes := []byte(c.name)
		var h bytes.Buffer
		binary.Write(&h, binary.LittleEndian, uint32(0x02014b50))
		binary.Write(&h, binary.LittleEndian, uint16(0x031E))
		binary.Write(&h, binary.LittleEndian, uint16(20))
		binary.Write(&h, binary.LittleEndian, uint16(0x800))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, dosTime)
		binary.Write(&h, binary.LittleEndian, dosDate)
		binary.Write(&h, binary.LittleEndian, c.crc)
		binary.Write(&h, binary.LittleEndian, c.size)
		binary.Write(&h, binary.LittleEndian, c.size)
		binary.Write(&h, binary.LittleEndian, uint16(len(nameBytes)))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, uint16(0))
		binary.Write(&h, binary.LittleEndian, uint32(c.external))
		binary.Write(&h, binary.LittleEndian, uint32(c.headerOff))
		h.Write(nameBytes)
		buf.Write(h.Bytes())
	}
	centralSize := int64(buf.Len()) - centralOff
	var end bytes.Buffer
	binary.Write(&end, binary.LittleEndian, uint32(0x06054b50))
	binary.Write(&end, binary.LittleEndian, uint16(0))
	binary.Write(&end, binary.LittleEndian, uint16(0))
	binary.Write(&end, binary.LittleEndian, uint16(len(centrals)))
	binary.Write(&end, binary.LittleEndian, uint16(len(centrals)))
	binary.Write(&end, binary.LittleEndian, uint32(centralSize))
	binary.Write(&end, binary.LittleEndian, uint32(centralOff))
	binary.Write(&end, binary.LittleEndian, uint16(0))
	buf.Write(end.Bytes())
	return buf.Bytes(), nil
}

// dosDateTime converts a time to MS-DOS date and time fields in UTC.
func dosDateTime(t time.Time) (uint16, uint16) {
	t = t.UTC()
	year := t.Year()
	if year < 1980 {
		year = 1980
	}
	if year > 2107 {
		year = 2107
	}
	dosTime := uint16(t.Hour()<<11 | t.Minute()<<5 | t.Second()/2)
	dosDate := uint16((year-1980)<<9 | int(t.Month())<<5 | t.Day())
	return dosTime, dosDate
}

// localInfo is the layout of one file in a ZIP.
type localInfo struct {
	headerSize int64
	dataOffset int64
	dataSize   int64
}

// parseLocalHeaders walks the local file headers of a ZIP and returns each
// entry's header size and data offset. It must match what Windows reads.
func parseLocalHeaders(data []byte) (map[string]localInfo, error) {
	out := map[string]localInfo{}
	pos := int64(0)
	for pos+30 <= int64(len(data)) {
		if data[pos] != 0x50 || data[pos+1] != 0x4b || data[pos+2] != 0x03 || data[pos+3] != 0x04 {
			break // central directory or end
		}
		flags := int(data[pos+8]) | int(data[pos+9])<<8
		method := int(data[pos+10]) | int(data[pos+11])<<8
		compressedSize := int64(uint32(data[pos+18]) | uint32(data[pos+19])<<8 | uint32(data[pos+20])<<16 | uint32(data[pos+21])<<24)
		nameLen := int(data[pos+26]) | int(data[pos+27])<<8
		extraLen := int(data[pos+28]) | int(data[pos+29])<<8
		headerSize := int64(30 + nameLen + extraLen)
		if pos+headerSize > int64(len(data)) {
			return nil, fmt.Errorf("truncated header for entry at %d", pos)
		}
		name := string(data[pos+30 : pos+30+int64(nameLen)])
		dataOffset := pos + headerSize
		out[name] = localInfo{headerSize: headerSize, dataOffset: dataOffset, dataSize: compressedSize}
		if flags&0x8 != 0 {
			return nil, fmt.Errorf("data descriptor not supported for %q (method %d)", name, method)
		}
		pos = dataOffset + compressedSize
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no local headers found")
	}
	return out, nil
}

// blockHashes splits data into 64 KiB blocks and hashes each with SHA-256.
func blockHashes(data []byte) []string {
	var out []string
	for off := 0; off < len(data); off += blockSize {
		end := off + blockSize
		if end > len(data) {
			end = len(data)
		}
		sum := sha256.Sum256(data[off:end])
		out = append(out, base64.StdEncoding.EncodeToString(sum[:]))
	}
	if len(out) == 0 {
		sum := sha256.Sum256(nil)
		out = append(out, base64.StdEncoding.EncodeToString(sum[:]))
	}
	return out
}

// RenderBlockMap renders AppxBlockMap.xml for files (name with backslashes as
// Windows writes them) with their sizes, header sizes and block hashes.
func RenderBlockMap(files []BlockMapFile) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="utf-8" standalone="no"?>` + "\n")
	b.WriteString(`<BlockMap xmlns="http://schemas.microsoft.com/appx/2010/blockmap" HashMethod="http://www.w3.org/2001/04/xmlenc#sha256">` + "\n")
	for _, f := range files {
		fmt.Fprintf(&b, "  <File Name=\"%s\" Size=\"%d\" LfhSize=\"%d\">\n", xmlEscape(toWindowsPath(f.Name)), f.Size, f.LfhSize)
		for _, h := range f.Blocks {
			fmt.Fprintf(&b, "    <Block Hash=\"%s\"/>\n", h)
		}
		b.WriteString("  </File>\n")
	}
	b.WriteString("</BlockMap>\n")
	return b.String()
}

// BlockMapFile is one file listed in AppxBlockMap.xml. Name uses forward
// slashes (the ZIP name); rendering converts them to backslashes.
type BlockMapFile struct {
	Name    string
	Size    int64
	LfhSize int64
	Blocks  []string
}

func toWindowsPath(name string) string {
	return strings.ReplaceAll(name, "/", "\\")
}

// AssetSizes are the MSIX logos generated at build time from the brand colour.
var assetSizes = []struct {
	name   string
	width  int
	height int
}{
	{"Assets/StoreLogo.png", 50, 50},
	{"Assets/Square44x44Logo.png", 44, 44},
	{"Assets/Square150x150Logo.png", 150, 150},
	{"Assets/Wide310x150Logo.png", 310, 150},
}

// GenerateAssets generates the MSIX logos: an orange gradient (the vfox brand)
// with a white "v+" mark. The icons are placeholders drawn in pure Go so that
// no binary is committed; they follow the brand colour of the SVG icon.
func GenerateAssets() (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, a := range assetSizes {
		img := drawIcon(a.width, a.height)
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
		out[a.name] = buf.Bytes()
	}
	return out, nil
}

// drawIcon draws the placeholder logo at w x h.
func drawIcon(w, h int) image.Image {
	top := color.RGBA{0xF9, 0xA5, 0x3E, 0xFF}
	bottom := color.RGBA{0xE2, 0x6A, 0x07, 0xFF}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h-1)
		if h == 1 {
			t = 0
		}
		r := uint8(float64(top.R)*(1-t) + float64(bottom.R)*t)
		g := uint8(float64(top.G)*(1-t) + float64(bottom.G)*t)
		b := uint8(float64(top.B)*(1-t) + float64(bottom.B)*t)
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{r, g, b, 0xFF})
		}
	}
	white := color.RGBA{0xFF, 0xFF, 0xFF, 0xFF}
	stroke := w / 16
	if stroke < 1 {
		stroke = 1
	}
	// "v": two diagonal bars.
	vTop := h / 4
	vBottom := h * 3 / 4
	vLeft := w / 6
	vMid := w * 5 / 12
	vRight := w * 8 / 12
	drawThickLine(img, vLeft, vTop, vMid, vBottom, stroke, white)
	drawThickLine(img, vRight, vTop, vMid, vBottom, stroke, white)
	// "+": right of the v.
	plusCx := w * 3 / 4
	plusCy := h / 2
	plusR := w / 12
	if plusR < 2 {
		plusR = 2
	}
	drawHLine(img, plusCx-plusR, plusCx+plusR, plusCy, stroke, white)
	drawVLine(img, plusCx, plusCy-plusR, plusCy+plusR, stroke, white)
	return img
}

func drawThickLine(img *image.RGBA, x0, y0, x1, y1, thickness int, c color.RGBA) {
	dx := x1 - x0
	dy := y1 - y0
	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}
	if steps == 0 {
		steps = 1
	}
	r := thickness / 2
	for i := 0; i <= steps; i++ {
		x := x0 + dx*i/steps
		y := y0 + dy*i/steps
		for oy := -r; oy <= r; oy++ {
			for ox := -r; ox <= r; ox++ {
				if ox*ox+oy*oy <= r*r+r {
					img.Set(x+ox, y+oy, c)
				}
			}
		}
	}
}

func drawHLine(img *image.RGBA, x0, x1, y, thickness int, c color.RGBA) {
	r := thickness / 2
	for x := x0; x <= x1; x++ {
		for oy := -r; oy <= r; oy++ {
			img.Set(x, y+oy, c)
		}
	}
}

func drawVLine(img *image.RGBA, x, y0, y1, thickness int, c color.RGBA) {
	r := thickness / 2
	for y := y0; y <= y1; y++ {
		for ox := -r; ox <= r; ox++ {
			img.Set(x+ox, y, c)
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// WriteMsix builds one unsigned .msix package.
func WriteMsix(cfg MsixConfig) ([]byte, error) {
	if cfg.Identity == "" || cfg.Publisher == "" || cfg.Version == "" || cfg.Architecture == "" {
		return nil, fmt.Errorf("identity, publisher, version and architecture are required")
	}
	if cfg.Executable == "" || len(cfg.ExecutableData) == 0 {
		return nil, fmt.Errorf("executable and its data are required")
	}
	if len(cfg.ExecutableData) < 2 || cfg.ExecutableData[0] != 'M' || cfg.ExecutableData[1] != 'Z' {
		return nil, fmt.Errorf("%s is not a Windows executable (missing MZ header)", cfg.Executable)
	}
	manifest := RenderAppxManifest(cfg)
	contentTypes := RenderContentTypes()
	assets, err := GenerateAssets()
	if err != nil {
		return nil, err
	}
	var entries []zipEntry
	entries = append(entries, zipEntry{name: "AppxManifest.xml", data: []byte(manifest), mode: 0o644})
	entries = append(entries, zipEntry{name: "[Content_Types].xml", data: []byte(contentTypes), mode: 0o644})
	entries = append(entries, zipEntry{name: cfg.Executable, data: cfg.ExecutableData, mode: 0o644})
	for name, data := range assets {
		entries = append(entries, zipEntry{name: name, data: data, mode: 0o644})
	}
	// BlockMap covers everything except itself. Header sizes are stable (they
	// depend only on entry names), so write once with placeholder sizes, parse
	// the real header sizes, then rewrite with the final BlockMap.
	blockFiles := func(headers map[string]int64) []BlockMapFile {
		byName := map[string][]byte{}
		byName["AppxManifest.xml"] = []byte(manifest)
		byName["[Content_Types].xml"] = []byte(contentTypes)
		byName[cfg.Executable] = cfg.ExecutableData
		for name, data := range assets {
			byName[name] = data
		}
		names := make([]string, 0, len(byName))
		for name := range byName {
			names = append(names, name)
		}
		sort.Strings(names)
		var out []BlockMapFile
		for _, name := range names {
			out = append(out, BlockMapFile{
				Name:    name,
				Size:    int64(len(byName[name])),
				LfhSize: headers[name],
				Blocks:  blockHashes(byName[name]),
			})
		}
		return out
	}
	// First pass with zero header sizes to learn the real ones.
	probeEntries := append([]zipEntry(nil), entries...)
	probeEntries = append(probeEntries, zipEntry{name: "AppxBlockMap.xml", data: []byte(RenderBlockMap(blockFiles(map[string]int64{}))), mode: 0o644})
	probeData, err := writeZip(probeEntries, cfg.MTime)
	if err != nil {
		return nil, err
	}
	probeHeaders, err := parseLocalHeaders(probeData)
	if err != nil {
		return nil, err
	}
	headers := map[string]int64{}
	for name, info := range probeHeaders {
		headers[name] = info.headerSize
	}
	blockMap := RenderBlockMap(blockFiles(headers))
	finalEntries := append([]zipEntry(nil), entries...)
	finalEntries = append(finalEntries, zipEntry{name: "AppxBlockMap.xml", data: []byte(blockMap), mode: 0o644})
	data, err := writeZip(finalEntries, cfg.MTime)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// assembleBundle writes a regular bundle: the .msix payloads (arch-sorted)
// first so their data offsets are stable, then the bundle manifest, the
// content types and the block map.
func assembleBundle(sorted []BundlePackage, manifest, contentTypes, blockMap string, mtime time.Time) ([]byte, error) {
	var entries []zipEntry
	for _, p := range sorted {
		entries = append(entries, zipEntry{name: p.Filename, data: p.Data, mode: 0o644})
	}
	entries = append(entries,
		zipEntry{name: "AppxMetadata/AppxBundleManifest.xml", data: []byte(manifest), mode: 0o644},
		zipEntry{name: "[Content_Types].xml", data: []byte(contentTypes), mode: 0o644},
		zipEntry{name: "AppxBlockMap.xml", data: []byte(blockMap), mode: 0o644},
	)
	return writeZipOrdered(entries, mtime)
}

// WriteBundle builds a regular .msixbundle containing the given .msix files.
// It returns the bundle bytes and its file name.
//
// The payload .msix entries are written first, so their data offsets are
// stable: the manifest (written after them) can embed the real offsets. Two
// passes therefore converge: probe with zero offsets to learn the real ones,
// then rewrite with the final manifest and BlockMap.
func WriteBundle(cfg BundleConfig) ([]byte, string, error) {
	if len(cfg.Packages) == 0 {
		return nil, "", fmt.Errorf("at least one .msix package is required")
	}
	quad, err := ToQuadVersion(cfg.Version)
	if err != nil {
		return nil, "", err
	}
	sorted := append([]BundlePackage(nil), cfg.Packages...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Architecture < sorted[j].Architecture })

	writeBundleWithManifest := func(manifest string, blockMap string, contentTypes string) ([]byte, error) {
		return assembleBundle(sorted, manifest, contentTypes, blockMap, cfg.MTime)
	}

	contentTypes := RenderBundleContentTypes(sorted)
	// Probe manifest with zero offsets to learn the real payload offsets.
	probeEntries := make([]BundleEntry, len(sorted))
	for i, p := range sorted {
		probeEntries[i] = BundleEntry{Architecture: p.Architecture, Filename: p.Filename}
	}
	probeManifest := RenderBundleManifest(cfg.Identity, cfg.Publisher, quad, probeEntries)
	probeBlockMap := RenderBlockMap(nil)
	probeData, err := writeBundleWithManifest(probeManifest, probeBlockMap, contentTypes)
	if err != nil {
		return nil, "", err
	}
	probeHeaders, err := parseLocalHeaders(probeData)
	if err != nil {
		return nil, "", err
	}
	var manifestEntries []BundleEntry
	for _, p := range sorted {
		info, ok := probeHeaders[p.Filename]
		if !ok {
			return nil, "", fmt.Errorf("%s missing in probe bundle", p.Filename)
		}
		manifestEntries = append(manifestEntries, BundleEntry{
			Architecture: p.Architecture,
			Filename:     p.Filename,
			Offset:       info.dataOffset,
			Size:         int64(len(p.Data)),
		})
	}
	manifest := RenderBundleManifest(cfg.Identity, cfg.Publisher, quad, manifestEntries)
	// The bundle block map lists only the bundle manifest itself: like
	// MakeAppx output, the .msix payloads are validated through their own
	// embedded block maps when the applicable slice is installed, and
	// Windows rejects a bundle block map with extra files (0x80080205).
	// Probe header sizes are stable (same names), reuse them.
	manifestInfo := probeHeaders["AppxMetadata/AppxBundleManifest.xml"]
	blockMap := RenderBlockMap([]BlockMapFile{{
		Name:    "AppxMetadata/AppxBundleManifest.xml",
		Size:    int64(len(manifest)),
		LfhSize: manifestInfo.headerSize,
		Blocks:  blockHashes([]byte(manifest)),
	}})
	finalData, err := writeBundleWithManifest(manifest, blockMap, contentTypes)
	if err != nil {
		return nil, "", err
	}
	// Verify the final payload offsets match the manifest.
	finalHeaders, err := parseLocalHeaders(finalData)
	if err != nil {
		return nil, "", err
	}
	for _, e := range manifestEntries {
		if finalHeaders[e.Filename].dataOffset != e.Offset {
			return nil, "", fmt.Errorf("bundle offset moved for %s: %d vs %d", e.Filename, finalHeaders[e.Filename].dataOffset, e.Offset)
		}
	}
	bundleName := fmt.Sprintf("%s_%s.msixbundle", cfg.Package, cfg.Version)
	return finalData, bundleName, nil
}
