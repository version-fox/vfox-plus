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

// Package renamefont gives a TrueType or OpenType font another family name.
//
// The outlined and the filled icon font are two instances of Material
// Symbols Rounded and have the same family name, so that fontconfig, which
// mygo draws text with, cannot tell them apart: it picks one of the two
// faces for both. The filled font is renamed to make the two families.
//
// It replaces mygo/tools/rename_font.py without requiring fontTools: it
// parses the SFNT table directory and the name table with the standard
// library only, rewrites the records that carry the family name (IDs 1, 4
// and 16) and the ones that carry the PostScript name (IDs 3 and 6), keeps
// every other table byte-identical, and recalculates the checksums.
package renamefont

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"
)

// magic is the value that the big-endian 32-bit words of a whole font file
// must add up to; the writer sets the head checkSumAdjustment to reach it.
const magic uint32 = 0xB1B0AFBA

// fontTable is one SFNT table: its tag and its exact bytes.
type fontTable struct {
	tag  [4]byte
	data []byte
}

// parseSFNT splits data into its sfnt version and tables, keeping their
// order. It rejects collections and WOFF containers, which have no table
// directory.
func parseSFNT(data []byte) ([4]byte, []fontTable, error) {
	var version [4]byte
	if len(data) < 12 {
		return version, nil, fmt.Errorf("file too short to be a font")
	}
	copy(version[:], data[:4])
	switch string(version[:]) {
	case "ttcf":
		return version, nil, fmt.Errorf("font collections are not supported")
	case "wOFF", "wOF2":
		return version, nil, fmt.Errorf("WOFF fonts are not supported")
	}
	n := int(binary.BigEndian.Uint16(data[4:6]))
	if len(data) < 12+16*n {
		return version, nil, fmt.Errorf("file too short for %d tables", n)
	}
	tables := make([]fontTable, 0, n)
	for i := 0; i < n; i++ {
		base := 12 + 16*i
		var tag [4]byte
		copy(tag[:], data[base:base+4])
		offset := binary.BigEndian.Uint32(data[base+8 : base+12])
		length := binary.BigEndian.Uint32(data[base+12 : base+16])
		if uint64(offset)+uint64(length) > uint64(len(data)) {
			return version, nil, fmt.Errorf("table %q out of bounds", tag)
		}
		tables = append(tables, fontTable{tag: tag, data: data[offset : offset+length]})
	}
	return version, tables, nil
}

// checksum adds the big-endian 32-bit words of data, padding the tail with
// zero bytes, as the SFNT table checksums do.
func checksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i < (len(data)+3)&^3; i += 4 {
		var word uint32
		for j := i; j < i+4; j++ {
			word <<= 8
			if j < len(data) {
				word |= uint32(data[j])
			}
		}
		sum += word
	}
	return sum
}

// buildSFNT lays the tables out in their given order with 4-byte alignment,
// recalculates the directory checksums, and fixes the head
// checkSumAdjustment so that the whole file adds up to the magic value.
func buildSFNT(version [4]byte, tables []fontTable) ([]byte, error) {
	n := len(tables)
	if n == 0 {
		return nil, fmt.Errorf("font has no tables")
	}
	head := -1
	for i := range tables {
		if tables[i].tag == [4]byte{'h', 'e', 'a', 'd'} {
			head = i
			break
		}
	}
	if head < 0 {
		return nil, fmt.Errorf("font has no 'head' table")
	}
	if len(tables[head].data) < 12 {
		return nil, fmt.Errorf("'head' table too short")
	}
	// The head checksum is calculated with a zero checkSumAdjustment.
	headData := bytes.Clone(tables[head].data)
	binary.BigEndian.PutUint32(headData[8:12], 0)

	entrySelector := 0
	for 1<<(entrySelector+1) <= n {
		entrySelector++
	}
	searchRange := (1 << entrySelector) * 16
	rangeShift := n*16 - searchRange

	checksums := make([]uint32, n)
	for i := range tables {
		data := tables[i].data
		if i == head {
			data = headData
		}
		checksums[i] = checksum(data)
	}

	var out bytes.Buffer
	out.Write(version[:])
	binary.Write(&out, binary.BigEndian, uint16(n))
	binary.Write(&out, binary.BigEndian, uint16(searchRange))
	binary.Write(&out, binary.BigEndian, uint16(entrySelector))
	binary.Write(&out, binary.BigEndian, uint16(rangeShift))
	offset := 12 + 16*n
	for i := range tables {
		out.Write(tables[i].tag[:])
		binary.Write(&out, binary.BigEndian, checksums[i])
		binary.Write(&out, binary.BigEndian, uint32(offset))
		binary.Write(&out, binary.BigEndian, uint32(len(tables[i].data)))
		offset += (len(tables[i].data) + 3) &^ 3
	}
	for i := range tables {
		data := tables[i].data
		if i == head {
			data = headData
		}
		out.Write(data)
		for out.Len()%4 != 0 {
			out.WriteByte(0)
		}
	}
	file := out.Bytes()
	adjustment := magic - checksum(file)
	headOffset := 12 + 16*n
	for i := 0; i < head; i++ {
		headOffset += (len(tables[i].data) + 3) &^ 3
	}
	binary.BigEndian.PutUint32(file[headOffset+8:headOffset+12], adjustment)
	return file, nil
}

// nameRecord is one record of the name table with its string bytes.
type nameRecord struct {
	platform, encoding, language, id uint16
	value                            []byte
}

// langTag is one language-tag record of a format 1 name table.
type langTag struct {
	length, offset uint16
	value          []byte
}

// nameTable is a parsed name table: its format, name records, and, for
// format 1, its language tags.
type nameTable struct {
	format  uint16
	records []nameRecord
	tags    []langTag
}

// parseNameTable parses the name table, keeping the format 1 language tags
// so that they survive the rewrite.
func parseNameTable(data []byte) (*nameTable, error) {
	if len(data) < 6 {
		return nil, fmt.Errorf("table too short")
	}
	format := binary.BigEndian.Uint16(data[0:2])
	if format > 1 {
		return nil, fmt.Errorf("unsupported name table format %d", format)
	}
	count := int(binary.BigEndian.Uint16(data[2:4]))
	stringOffset := int(binary.BigEndian.Uint16(data[4:6]))
	head := 6 + 12*count
	tagCount := 0
	if format == 1 {
		if len(data) < head+2 {
			return nil, fmt.Errorf("table too short for language tags")
		}
		tagCount = int(binary.BigEndian.Uint16(data[head : head+2]))
		head += 2 + 4*tagCount
	}
	if len(data) < head {
		return nil, fmt.Errorf("table too short for %d records", count)
	}
	if stringOffset < head || stringOffset > len(data) {
		return nil, fmt.Errorf("bad string offset %d", stringOffset)
	}
	table := &nameTable{format: format}
	for i := 0; i < count; i++ {
		base := 6 + 12*i
		length := int(binary.BigEndian.Uint16(data[base+8 : base+10]))
		offset := int(binary.BigEndian.Uint16(data[base+10 : base+12]))
		if stringOffset+offset+length > len(data) {
			return nil, fmt.Errorf("record %d out of bounds", i)
		}
		table.records = append(table.records, nameRecord{
			platform: binary.BigEndian.Uint16(data[base : base+2]),
			encoding: binary.BigEndian.Uint16(data[base+2 : base+4]),
			language: binary.BigEndian.Uint16(data[base+4 : base+6]),
			id:       binary.BigEndian.Uint16(data[base+6 : base+8]),
			value:    bytes.Clone(data[stringOffset+offset : stringOffset+offset+length]),
		})
	}
	for i := 0; i < tagCount; i++ {
		base := 6 + 12*count + 2 + 4*i
		length := int(binary.BigEndian.Uint16(data[base : base+2]))
		offset := int(binary.BigEndian.Uint16(data[base+2 : base+4]))
		if stringOffset+offset+length > len(data) {
			return nil, fmt.Errorf("language tag %d out of bounds", i)
		}
		table.tags = append(table.tags, langTag{
			length: uint16(length),
			offset: uint16(offset),
			value:  bytes.Clone(data[stringOffset+offset : stringOffset+offset+length]),
		})
	}
	return table, nil
}

// renamedValue returns the new string for the name IDs that carry the family
// name or the PostScript name, and false for every other ID.
func renamedValue(id uint16, family, postscript string) (string, bool) {
	switch id {
	case 1, 4, 16:
		return family, true
	case 3, 6:
		return postscript, true
	}
	return "", false
}

// buildNameTable rebuilds the name table, encoding the renamed records in
// their own encoding and copying the other records and the language tags
// verbatim.
func buildNameTable(table *nameTable, family, postscript string) ([]byte, error) {
	var storage bytes.Buffer
	type dirEntry struct {
		platform, encoding, language, id, length, offset uint16
	}
	entries := make([]dirEntry, 0, len(table.records))
	for _, record := range table.records {
		value := record.value
		if renamed, ok := renamedValue(record.id, family, postscript); ok {
			encoded, err := encodeNameString(record.platform, record.encoding, renamed)
			if err != nil {
				return nil, fmt.Errorf("nameID %d: %w", record.id, err)
			}
			value = encoded
		}
		if len(value) > 0xFFFF || storage.Len()+len(value) > 0xFFFF {
			return nil, fmt.Errorf("name table string storage overflow")
		}
		offset := storage.Len()
		storage.Write(value)
		entries = append(entries, dirEntry{
			platform: record.platform,
			encoding: record.encoding,
			language: record.language,
			id:       record.id,
			length:   uint16(len(value)),
			offset:   uint16(offset),
		})
	}
	tags := make([]dirEntry, 0, len(table.tags))
	for _, tag := range table.tags {
		if storage.Len()+len(tag.value) > 0xFFFF {
			return nil, fmt.Errorf("name table string storage overflow")
		}
		offset := storage.Len()
		storage.Write(tag.value)
		tags = append(tags, dirEntry{length: tag.length, offset: uint16(offset)})
	}
	stringOffset := 6 + 12*len(entries)
	if table.format == 1 {
		stringOffset += 2 + 4*len(tags)
	}
	var out bytes.Buffer
	binary.Write(&out, binary.BigEndian, table.format)
	binary.Write(&out, binary.BigEndian, uint16(len(entries)))
	binary.Write(&out, binary.BigEndian, uint16(stringOffset))
	for _, e := range entries {
		binary.Write(&out, binary.BigEndian, e.platform)
		binary.Write(&out, binary.BigEndian, e.encoding)
		binary.Write(&out, binary.BigEndian, e.language)
		binary.Write(&out, binary.BigEndian, e.id)
		binary.Write(&out, binary.BigEndian, e.length)
		binary.Write(&out, binary.BigEndian, e.offset)
	}
	if table.format == 1 {
		binary.Write(&out, binary.BigEndian, uint16(len(tags)))
		for _, t := range tags {
			binary.Write(&out, binary.BigEndian, t.length)
			binary.Write(&out, binary.BigEndian, t.offset)
		}
	}
	out.Write(storage.Bytes())
	return out.Bytes(), nil
}

// encodeNameString encodes s the way the name record's platform and encoding
// require: UTF-16BE for Unicode and Windows Unicode records, MacRoman for
// Macintosh Roman records.
func encodeNameString(platform, encoding uint16, s string) ([]byte, error) {
	switch {
	case platform == 0 || platform == 3 && (encoding == 0 || encoding == 1 || encoding == 10):
		units := utf16.Encode([]rune(s))
		out := make([]byte, 2*len(units))
		for i, unit := range units {
			binary.BigEndian.PutUint16(out[2*i:], unit)
		}
		return out, nil
	case platform == 1 && encoding == 0:
		return encodeMacRoman(s)
	default:
		return nil, fmt.Errorf("unsupported encoding: platform %d, encoding %d", platform, encoding)
	}
}

// macRoman holds the 0x80-0xFF half of Mac OS Roman; 0x00-0x7F is ASCII.
var macRoman = [128]rune{
	0x00C4, 0x00C5, 0x00C7, 0x00C9, 0x00D1, 0x00D6, 0x00DC, 0x00E1,
	0x00E0, 0x00E2, 0x00E4, 0x00E3, 0x00E5, 0x00E7, 0x00E9, 0x00E8,
	0x00EA, 0x00EB, 0x00ED, 0x00EC, 0x00EE, 0x00EF, 0x00F1, 0x00F3,
	0x00F2, 0x00F4, 0x00F6, 0x00F5, 0x00FA, 0x00F9, 0x00FB, 0x00FC,
	0x2020, 0x00B0, 0x00A2, 0x00A3, 0x00A7, 0x2022, 0x00B6, 0x00DF,
	0x00AE, 0x00A9, 0x2122, 0x00B4, 0x00A8, 0x2260, 0x00C6, 0x00D8,
	0x221E, 0x00B1, 0x2264, 0x2265, 0x00A5, 0x00B5, 0x2202, 0x2211,
	0x220F, 0x03C0, 0x222B, 0x00AA, 0x00BA, 0x03A9, 0x00E6, 0x00F8,
	0x00BF, 0x00A1, 0x00AC, 0x221A, 0x0192, 0x2248, 0x2206, 0x00AB,
	0x00BB, 0x2026, 0x00A0, 0x00C0, 0x00C3, 0x00D5, 0x0152, 0x0153,
	0x2013, 0x2014, 0x201C, 0x201D, 0x2018, 0x2019, 0x00F7, 0x25CA,
	0x00FF, 0x0178, 0x2044, 0x00A4, 0x2039, 0x203A, 0xFB01, 0xFB02,
	0x2021, 0x00B7, 0x201A, 0x201E, 0x2030, 0x00C2, 0x00CA, 0x00C1,
	0x00CB, 0x00C8, 0x00CD, 0x00CE, 0x00CF, 0x00CC, 0x00D3, 0x00F4,
	0xF8FF, 0x00D2, 0x00DA, 0x00DB, 0x00D9, 0x0131, 0x02C6, 0x02DC,
	0x00AF, 0x02D8, 0x02D9, 0x02DA, 0x00B8, 0x02DD, 0x02DB, 0x02C7,
}

// macRomanEncode maps runes back to their MacRoman bytes.
var macRomanEncode = func() map[rune]byte {
	m := make(map[rune]byte, len(macRoman))
	for i, r := range macRoman {
		if _, ok := m[r]; !ok {
			m[r] = byte(0x80 + i)
		}
	}
	return m
}()

// encodeMacRoman encodes s in Mac OS Roman.
func encodeMacRoman(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r < 0x80:
			out = append(out, byte(r))
		default:
			b, ok := macRomanEncode[r]
			if !ok {
				return nil, fmt.Errorf("character U+%04X not representable in MacRoman", r)
			}
			out = append(out, b)
		}
	}
	return out, nil
}

// renameFont returns a copy of the font data with the family renamed to
// family and its PostScript name set to the family without spaces plus
// "-Regular".
func renameFont(data []byte, family string) ([]byte, error) {
	version, tables, err := parseSFNT(data)
	if err != nil {
		return nil, err
	}
	name := -1
	for i := range tables {
		if tables[i].tag == [4]byte{'n', 'a', 'm', 'e'} {
			name = i
			break
		}
	}
	if name < 0 {
		return nil, fmt.Errorf("font has no 'name' table")
	}
	table, err := parseNameTable(tables[name].data)
	if err != nil {
		return nil, fmt.Errorf("invalid 'name' table: %w", err)
	}
	postscript := strings.ReplaceAll(family, " ", "") + "-Regular"
	rebuilt, err := buildNameTable(table, family, postscript)
	if err != nil {
		return nil, err
	}
	tables[name].data = rebuilt
	return buildSFNT(version, tables)
}

// Run renames the family of the font at FONT to FAMILY. It accepts FONT
// FAMILY.
func Run(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: renamefont FONT FAMILY")
	}
	path, family := args[0], args[1]
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	renamed, err := renameFont(data, family)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if err := os.WriteFile(path, renamed, 0o666); err != nil {
		return err
	}
	fmt.Printf("Renamed the family of %s to %s\n", path, family)
	return nil
}
