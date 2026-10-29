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
	"encoding/binary"
	"strings"
	"testing"
	"time"
)

// zip64Samples are a package and a bundle, written by this package.
func zip64Samples(t *testing.T) map[string][]byte {
	t.Helper()
	msix, err := WriteMsix(testMsixConfig())
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := WriteBundle(BundleConfig{
		Package:   "vfox-plus",
		Version:   "0.1.0",
		Identity:  "dev.vfox.VfoxPlus",
		Publisher: defaultPublisher,
		Packages:  []BundlePackage{{Architecture: "x64", Filename: "vfox-plus_0.1.0_x64.msix", Data: msix}},
		MTime:     time.Unix(1234567890, 0).UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	return map[string][]byte{"package": msix, "bundle": bundle}
}

// The end of every archive is the end of a package of Microsoft: this is what
// SignedUntrustedCert-CERT_E_CHAINING.appx of microsoft/msix-packaging (signed by
// SignTool) ends with, field by field. Windows refuses a signed package with a
// plain ZIP32 end (0x8007000B), so the shape is pinned here.
func TestZipEndsLikeThePackagesOfMicrosoft(t *testing.T) {
	le := binary.LittleEndian
	for name, data := range zip64Samples(t) {
		reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		n := len(data)

		// The end record has the sentinels, and no comment.
		wantEnd := []byte{'P', 'K', 5, 6, 0, 0, 0, 0, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0, 0}
		if end := data[n-22:]; !bytes.Equal(end, wantEnd) {
			t.Errorf("%s: the end record is % x, want % x", name, end, wantEnd)
		}

		// The locator of the ZIP64 end record: one disk, which holds it.
		locator := data[n-42 : n-22]
		if !bytes.Equal(locator[:8], []byte{'P', 'K', 6, 7, 0, 0, 0, 0}) || le.Uint32(locator[16:]) != 1 {
			t.Errorf("%s: the locator is % x", name, locator)
		}

		// The ZIP64 end record, made by and needing 4.5, that ends where the locator starts.
		start := le.Uint64(locator[8:])
		record := data[start : n-42]
		if len(record) != 56 || !bytes.Equal(record[:12], []byte{'P', 'K', 6, 6, 44, 0, 0, 0, 0, 0, 0, 0}) {
			t.Fatalf("%s: the ZIP64 end record is % x", name, record)
		}
		if !bytes.Equal(record[12:24], []byte{45, 0, 45, 0, 0, 0, 0, 0, 0, 0, 0, 0}) {
			t.Errorf("%s: the versions and the disks are % x", name, record[12:24])
		}
		entries, total, size, offset := le.Uint64(record[24:]), le.Uint64(record[32:]), le.Uint64(record[40:]), le.Uint64(record[48:])
		if entries != uint64(len(reader.File)) || total != entries {
			t.Errorf("%s: %d and %d entries, the archive has %d", name, entries, total, len(reader.File))
		}
		if offset+size != start {
			t.Errorf("%s: the central directory (%d bytes at %d) does not end where the ZIP64 end record starts (%d)", name, size, offset, start)
		}

		// Every entry is made by 4.5 for MS-DOS, needs 2.0, and has no extra
		// field, no comment and no attributes, like the entries of the packages
		// of openappx that an Xbox installed once they were signed.
		for _, file := range reader.File {
			if file.CreatorVersion != 45 || file.ReaderVersion != 20 || file.Flags != 0 || len(file.Extra) != 0 || file.Comment != "" || file.ExternalAttrs != 0 {
				t.Errorf("%s: %s is made by %#x, needs %d, has flags %#x, extra % x, comment %q, attributes %#x", name, file.Name, file.CreatorVersion, file.ReaderVersion, file.Flags, file.Extra, file.Comment, file.ExternalAttrs)
			}
		}
		if err := CheckZip64(data); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestZipFlagsNonASCIINamesAsUTF8(t *testing.T) {
	data, err := writeZip([]zipEntry{{name: "plain.txt", data: []byte("a")}, {name: "café.txt", data: []byte("b")}}, time.Unix(1234567890, 0).UTC())
	if err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range reader.File {
		want := uint16(0)
		if file.Name != "plain.txt" {
			want = 0x800
		}
		if file.Flags != want {
			t.Errorf("%s has the flags %#x, want %#x", file.Name, file.Flags, want)
		}
	}
}

func TestZipRefusesWhatNeedsZip64ExtraFields(t *testing.T) {
	_, err := writeZip([]zipEntry{{name: strings.Repeat("a", 0x10000), data: []byte("a")}}, time.Unix(1234567890, 0).UTC())
	if err == nil || !strings.Contains(err.Error(), "does not fit") {
		t.Errorf("a name of 65536 bytes: %v", err)
	}
}

// CheckZip64 says what is wrong with an archive that Windows would refuse once
// it is signed.
func TestCheckZip64(t *testing.T) {
	le := binary.LittleEndian
	samples := zip64Samples(t)

	var zip32 bytes.Buffer
	writer := zip.NewWriter(&zip32)
	entry, err := writer.Create("AppxManifest.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = entry.Write([]byte("<Package/>"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	// mutate copies the package and lets a function change it. The offsets are
	// those of the ZIP64 end record and of the first central directory record.
	mutate := func(change func(data []byte, record, central int) []byte) []byte {
		data := append([]byte(nil), samples["package"]...)
		record := int(le.Uint64(data[len(data)-42+8:])) // The locator holds the offset of the record.
		return change(data, record, int(le.Uint64(data[record+48:])))
	}
	for name, test := range map[string]struct {
		data []byte
		want string // a part of the error, or "" for no error
	}{
		"a package":   {samples["package"], ""},
		"a bundle":    {samples["bundle"], ""},
		"a ZIP32 one": {zip32.Bytes(), "ZIP32 end record"},
		"a comment": {append(append([]byte(nil), samples["package"]...), 'x'),
			"does not end with the end record"},
		"no locator": {mutate(func(data []byte, record, central int) []byte {
			data[len(data)-42] = 'X'
			return data
		}), "no ZIP64 end record locator"},
		"a locator that points elsewhere": {mutate(func(data []byte, record, central int) []byte {
			le.PutUint64(data[len(data)-42+8:], 3)
			return data
		}), "has no ZIP64 end record at 3"},
		"a ZIP64 end record made by 2.0": {mutate(func(data []byte, record, central int) []byte {
			le.PutUint16(data[record+12:], 20)
			return data
		}), "made by version 20"},
		"several disks": {mutate(func(data []byte, record, central int) []byte {
			le.PutUint32(data[record+16:], 1)
			return data
		}), "several disks"},
		"entries that differ": {mutate(func(data []byte, record, central int) []byte {
			le.PutUint64(data[record+24:], 1)
			return data
		}), "entries on its disk"},
		"a central directory that ends elsewhere": {mutate(func(data []byte, record, central int) []byte {
			le.PutUint64(data[record+40:], le.Uint64(data[record+40:])-1)
			return data
		}), "does not end where the ZIP64 end record starts"},
		"an entry made by Unix": {mutate(func(data []byte, record, central int) []byte {
			le.PutUint16(data[central+4:], 0x031E)
			return data
		}), "made by version 798"},
		"an extra field": {mutate(func(data []byte, record, central int) []byte {
			le.PutUint16(data[central+30:], 4)
			return data
		}), "extra field in central directory record 0"},
		"a record that is not one": {mutate(func(data []byte, record, central int) []byte {
			data[central] = 'X'
			return data
		}), "no central directory record 0"},
	} {
		err := CheckZip64(test.data)
		switch {
		case test.want == "" && err != nil:
			t.Errorf("%s: %v", name, err)
		case test.want != "" && (err == nil || !strings.Contains(err.Error(), test.want)):
			t.Errorf("%s: got %v, want an error with %q", name, err, test.want)
		}
	}

	// A truncated archive is refused, and never makes the check run out of its bounds.
	for _, data := range samples {
		for cut := 0; cut < len(data); cut += 1 + cut/50 {
			if CheckZip64(data[:cut]) == nil {
				t.Fatalf("a package cut to %d of %d bytes passes", cut, len(data))
			}
		}
		for cut := len(data) - 200; cut < len(data); cut++ {
			if CheckZip64(data[:cut]) == nil {
				t.Fatalf("a package cut to %d of %d bytes passes", cut, len(data))
			}
		}
	}
}

// What the validator of the packages (and so the build) refuses: the ZIP32 end.
func TestValidateRefusesAZip32Package(t *testing.T) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, name := range []string{"AppxManifest.xml", "[Content_Types].xml", "AppxBlockMap.xml"} {
		entry, err := writer.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store, Modified: time.Unix(1234567890, 0).UTC()})
		if err != nil {
			t.Fatal(err)
		}
		_, _ = entry.Write([]byte("<a/>"))
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	err := ValidateMsix(buffer.Bytes(), "zip32.msix")
	if err == nil || !strings.Contains(err.Error(), "zip32.msix ends with a ZIP32 end record") {
		t.Errorf("got %v", err)
	}
}
