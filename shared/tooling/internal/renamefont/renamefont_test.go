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

package renamefont

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

type rawNameRec struct {
	platform, encoding, language, id uint16
	value                            []byte
}

func utf16beEncode(s string) []byte {
	units := utf16.Encode([]rune(s))
	out := make([]byte, 2*len(units))
	for i, unit := range units {
		binary.BigEndian.PutUint16(out[2*i:], unit)
	}
	return out
}

func utf16beDecode(b []byte) string {
	if len(b)%2 != 0 {
		panic("odd UTF-16BE length")
	}
	units := make([]uint16, len(b)/2)
	for i := range units {
		units[i] = binary.BigEndian.Uint16(b[2*i:])
	}
	return string(utf16.Decode(units))
}

// buildNameRaw assembles a name table without using production code, so the
// round-trip test stays independent of buildNameTable.
func buildNameRaw(format uint16, recs []rawNameRec, langTags []string) []byte {
	offsets := make([]int, len(recs))
	off := 0
	for i, r := range recs {
		offsets[i] = off
		off += len(r.value)
	}
	tagOffsets := make([]int, len(langTags))
	for i, tag := range langTags {
		tagOffsets[i] = off
		off += len(tag)
	}
	stringOffset := 6 + 12*len(recs)
	if format == 1 {
		stringOffset += 2 + 4*len(langTags)
	}
	var out bytes.Buffer
	binary.Write(&out, binary.BigEndian, format)
	binary.Write(&out, binary.BigEndian, uint16(len(recs)))
	binary.Write(&out, binary.BigEndian, uint16(stringOffset))
	for i, r := range recs {
		binary.Write(&out, binary.BigEndian, r.platform)
		binary.Write(&out, binary.BigEndian, r.encoding)
		binary.Write(&out, binary.BigEndian, r.language)
		binary.Write(&out, binary.BigEndian, r.id)
		binary.Write(&out, binary.BigEndian, uint16(len(r.value)))
		binary.Write(&out, binary.BigEndian, uint16(offsets[i]))
	}
	if format == 1 {
		binary.Write(&out, binary.BigEndian, uint16(len(langTags)))
		for i, tag := range langTags {
			binary.Write(&out, binary.BigEndian, uint16(len(tag)))
			binary.Write(&out, binary.BigEndian, uint16(tagOffsets[i]))
		}
	}
	for _, r := range recs {
		out.Write(r.value)
	}
	for _, tag := range langTags {
		out.WriteString(tag)
	}
	return out.Bytes()
}

func buildHeadRaw() []byte {
	head := make([]byte, 54)
	binary.BigEndian.PutUint32(head[0:4], 0x00010000)
	binary.BigEndian.PutUint32(head[4:8], 0x00010000)
	binary.BigEndian.PutUint32(head[12:16], 0x5F0F3CF5)
	binary.BigEndian.PutUint16(head[18:20], 1000)
	return head
}

// fixtureFont builds a font with head, an odd-length dummy table, and name,
// in that non-alphabetical order to prove the order survives.
func fixtureFont(t *testing.T, nameBytes []byte) []byte {
	t.Helper()
	font, err := buildSFNT([4]byte{0, 1, 0, 0}, []fontTable{
		{tag: [4]byte{'h', 'e', 'a', 'd'}, data: buildHeadRaw()},
		{tag: [4]byte{'D', 'U', 'M', 'M'}, data: []byte{1, 2, 3, 4, 5, 6, 7}},
		{tag: [4]byte{'n', 'a', 'm', 'e'}, data: nameBytes},
	})
	if err != nil {
		t.Fatal(err)
	}
	return font
}

func fixtureName() []byte {
	win := func(id uint16, s string) rawNameRec {
		return rawNameRec{platform: 3, encoding: 1, language: 0x409, id: id, value: utf16beEncode(s)}
	}
	mac := func(id uint16, s string) rawNameRec {
		return rawNameRec{platform: 1, encoding: 0, language: 0, id: id, value: []byte(s)}
	}
	return buildNameRaw(0, []rawNameRec{
		win(0, "Copyright Old"),
		win(1, "Old Family"),
		win(2, "Regular"),
		win(3, "OldFamily-Regular"),
		win(4, "Old Family Regular"),
		win(6, "OldFamily-Regular"),
		win(16, "Old Family"),
		mac(1, "Old Family"),
		mac(4, "Old Family Regular"),
		mac(6, "OldFamily-Regular"),
	}, nil)
}

type dirEntry struct {
	checksum, offset, length uint32
}

// readDir parses the SFNT directory with test-local code.
func readDir(t *testing.T, data []byte) ([]string, map[string]dirEntry) {
	t.Helper()
	n := int(binary.BigEndian.Uint16(data[4:6]))
	var order []string
	entries := map[string]dirEntry{}
	for i := 0; i < n; i++ {
		base := 12 + 16*i
		tag := string(data[base : base+4])
		order = append(order, tag)
		entries[tag] = dirEntry{
			checksum: binary.BigEndian.Uint32(data[base+4 : base+8]),
			offset:   binary.BigEndian.Uint32(data[base+8 : base+12]),
			length:   binary.BigEndian.Uint32(data[base+12 : base+16]),
		}
	}
	return order, entries
}

// checksumLocal recomputes a table checksum independently of checksum().
func checksumLocal(data []byte) uint32 {
	var sum uint32
	for i := 0; i < len(data); i += 4 {
		var word uint32
		for j := 0; j < 4; j++ {
			word <<= 8
			if i+j < len(data) {
				word |= uint32(data[i+j])
			}
		}
		sum += word
	}
	return sum
}

func tableSlice(data []byte, e dirEntry) []byte {
	return data[e.offset : e.offset+e.length]
}

func TestRenameRoundTrip(t *testing.T) {
	before := fixtureFont(t, fixtureName())
	after, err := renameFont(before, "New Family")
	if err != nil {
		t.Fatal(err)
	}

	// Table order and the untouched table survive byte-identically.
	order, entries := readDir(t, after)
	wantOrder := []string{"head", "DUMM", "name"}
	if strings.Join(order, ",") != strings.Join(wantOrder, ",") {
		t.Errorf("table order = %v, want %v", order, wantOrder)
	}
	_, beforeEntries := readDir(t, before)
	if !bytes.Equal(tableSlice(after, entries["DUMM"]), tableSlice(before, beforeEntries["DUMM"])) {
		t.Error("DUMM table changed")
	}
	headBefore := tableSlice(before, beforeEntries["head"])
	headAfter := tableSlice(after, entries["head"])
	maskedBefore := append(bytes.Clone(headBefore[:8]), headBefore[12:]...)
	maskedAfter := append(bytes.Clone(headAfter[:8]), headAfter[12:]...)
	if !bytes.Equal(maskedBefore, maskedAfter) {
		t.Error("head table changed outside checkSumAdjustment")
	}

	// Name records carry the new family and PostScript names.
	nameTable, err := parseNameTable(tableSlice(after, entries["name"]))
	if err != nil {
		t.Fatal(err)
	}
	family := utf16beEncode("New Family")
	postscript := utf16beEncode("NewFamily-Regular")
	for _, r := range nameTable.records {
		var want []byte
		switch {
		case r.platform == 3 && (r.id == 1 || r.id == 4 || r.id == 16):
			want = family
		case r.platform == 3 && (r.id == 3 || r.id == 6):
			want = postscript
		case r.platform == 1 && (r.id == 1 || r.id == 4 || r.id == 16):
			want = []byte("New Family")
		case r.platform == 1 && (r.id == 3 || r.id == 6):
			want = []byte("NewFamily-Regular")
		case r.id == 0:
			want = utf16beEncode("Copyright Old")
		case r.id == 2:
			want = utf16beEncode("Regular")
		default:
			t.Fatalf("unexpected record %+v", r)
		}
		if !bytes.Equal(r.value, want) {
			t.Errorf("record platform=%d id=%d = %q, want %q", r.platform, r.id, r.value, want)
		}
	}

	// Directory checksums match the table bytes; the head checksum covers a
	// zeroed adjustment.
	for tag, e := range entries {
		data := tableSlice(after, e)
		if tag == "head" {
			data = bytes.Clone(data)
			binary.BigEndian.PutUint32(data[8:12], 0)
		}
		if got := checksumLocal(data); got != e.checksum {
			t.Errorf("table %q checksum = %#x, want %#x", tag, e.checksum, got)
		}
	}

	// The whole file adds up to the magic value.
	if got := checksumLocal(after); got != magic {
		t.Errorf("file checksum = %#x, want %#x", got, magic)
	}
}

func TestRenameFormat1KeepsLangTags(t *testing.T) {
	name := buildNameRaw(1, []rawNameRec{
		{platform: 3, encoding: 1, language: 0x409, id: 1, value: utf16beEncode("Old Family")},
	}, []string{"en"})
	after, err := renameFont(fixtureFont(t, name), "New Family")
	if err != nil {
		t.Fatal(err)
	}
	_, entries := readDir(t, after)
	table, err := parseNameTable(tableSlice(after, entries["name"]))
	if err != nil {
		t.Fatal(err)
	}
	if table.format != 1 {
		t.Errorf("format = %d, want 1", table.format)
	}
	if len(table.tags) != 1 || string(table.tags[0].value) != "en" {
		t.Errorf("lang tags = %+v, want [en]", table.tags)
	}
	if len(table.records) != 1 || utf16beDecode(table.records[0].value) != "New Family" {
		t.Errorf("records = %+v, want the renamed family", table.records)
	}
	if got := checksumLocal(after); got != magic {
		t.Errorf("file checksum = %#x, want %#x", got, magic)
	}
}

func TestRenameErrors(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{"short", []byte{0, 1}, "too short"},
		{"collection", bytes.Repeat([]byte("ttcf"), 8), "collections"},
		{"woff", append([]byte("wOFF"), make([]byte, 12)...), "WOFF"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := renameFont(tc.data, "New Family"); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("renameFont = %v, want an error containing %q", err, tc.want)
			}
		})
	}

	t.Run("no name table", func(t *testing.T) {
		font, err := buildSFNT([4]byte{0, 1, 0, 0}, []fontTable{
			{tag: [4]byte{'h', 'e', 'a', 'd'}, data: buildHeadRaw()},
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := renameFont(font, "New Family"); err == nil || !strings.Contains(err.Error(), "no 'name' table") {
			t.Errorf("renameFont = %v, want a missing name table error", err)
		}
	})

	t.Run("unsupported encoding", func(t *testing.T) {
		name := buildNameRaw(0, []rawNameRec{
			{platform: 2, encoding: 1, language: 0, id: 1, value: []byte("Old")},
		}, nil)
		if _, err := renameFont(fixtureFont(t, name), "New Family"); err == nil || !strings.Contains(err.Error(), "unsupported encoding") {
			t.Errorf("renameFont = %v, want an unsupported encoding error", err)
		}
	})

	t.Run("unrepresentable macroman", func(t *testing.T) {
		name := buildNameRaw(0, []rawNameRec{
			{platform: 1, encoding: 0, language: 0, id: 1, value: []byte("Old")},
		}, nil)
		// U+4E2D has no MacRoman byte.
		if _, err := renameFont(fixtureFont(t, name), "Fa\u4e2dmily"); err == nil || !strings.Contains(err.Error(), "MacRoman") {
			t.Errorf("renameFont = %v, want a MacRoman error", err)
		}
	})
}

func TestRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "font.ttf")
	if err := os.WriteFile(path, fixtureFont(t, fixtureName()), 0o666); err != nil {
		t.Fatal(err)
	}
	if err := Run([]string{path, "New Family"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	_, entries := readDir(t, data)
	table, err := parseNameTable(tableSlice(data, entries["name"]))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range table.records {
		if r.platform == 3 && r.id == 1 && utf16beDecode(r.value) != "New Family" {
			t.Errorf("family record = %q", utf16beDecode(r.value))
		}
	}
	if got := checksumLocal(data); got != magic {
		t.Errorf("file checksum = %#x, want %#x", got, magic)
	}

	if err := Run([]string{path}); err == nil {
		t.Error("Run with one arg = nil, want an error")
	}
	if err := Run([]string{filepath.Join(t.TempDir(), "missing.ttf"), "F"}); err == nil {
		t.Error("Run with a missing file = nil, want an error")
	}
}
