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

package iconfont

import (
	"encoding/binary"
	"fmt"
	"sort"
)

// sfntFont is a parsed SFNT container: the sfnt version plus the raw bytes
// of every table. Rebuilt tables replace their raw bytes; untouched tables
// are copied byte-identically.
type sfntFont struct {
	sfntVersion uint32
	tables      map[string][]byte
}

// parseSFNT parses an SFNT file (offset table + table directory). Table data
// is sliced (padded length trimmed to the recorded length).
func parseSFNT(data []byte) (*sfntFont, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("file too short for SFNT header: %d bytes", len(data))
	}
	numTables := int(getU16(data, 4))
	f := &sfntFont{sfntVersion: getU32(data, 0), tables: map[string][]byte{}}
	if len(data) < 12+16*numTables {
		return nil, fmt.Errorf("file too short for %d table records", numTables)
	}
	for i := 0; i < numTables; i++ {
		rec := data[12+16*i:]
		tag := string(rec[:4])
		offset := getU32(rec, 8)
		length := getU32(rec, 12)
		if uint64(offset)+uint64(length) > uint64(len(data)) {
			return nil, fmt.Errorf("table %q out of range (offset %d, length %d, file %d)", tag, offset, length, len(data))
		}
		if _, dup := f.tables[tag]; dup {
			return nil, fmt.Errorf("duplicate table %q", tag)
		}
		f.tables[tag] = data[offset : offset+length]
	}
	return f, nil
}

// tableChecksum computes the SFNT checksum of padded table data.
func tableChecksum(data []byte) uint32 {
	var sum uint32
	n := len(data)
	for i := 0; i+4 <= n; i += 4 {
		sum += binary.BigEndian.Uint32(data[i:])
	}
	if rem := n % 4; rem != 0 {
		var last [4]byte
		copy(last[:], data[n-rem:])
		sum += binary.BigEndian.Uint32(last[:])
	}
	return sum
}

// wholeFontChecksum sums the whole file as big-endian uint32 words.
func wholeFontChecksum(data []byte) uint32 {
	return tableChecksum(data)
}

// writeSFNT serializes tables into an SFNT file with tables sorted by tag,
// 4-byte padded, with per-table checksums and head.checkSumAdjustment set so
// the whole file checksums to 0xB1B0AFBA.
func writeSFNT(sfntVersion uint32, tags []string, tables map[string][]byte) ([]byte, error) {
	final := make(map[string][]byte, len(tags))
	for _, tag := range tags {
		data := tables[tag]
		if data == nil {
			data = []byte{}
		}
		final[tag] = data
	}
	ordered := append([]string(nil), tags...)
	sortTags(ordered)
	numTables := len(ordered)
	// searchRange etc. per the SFNT spec.
	maxPow := 1
	for maxPow*2 <= numTables {
		maxPow *= 2
	}
	entrySelector := 0
	for p := maxPow; p > 1; p >>= 1 {
		entrySelector++
	}
	searchRange := uint16(maxPow * 16)
	rangeShift := uint16(numTables*16) - searchRange

	out := make([]byte, 0, 12+16*numTables)
	hdr := make([]byte, 12)
	binary.BigEndian.PutUint32(hdr[0:], sfntVersion)
	binary.BigEndian.PutUint16(hdr[4:], uint16(numTables))
	binary.BigEndian.PutUint16(hdr[6:], searchRange)
	binary.BigEndian.PutUint16(hdr[8:], uint16(entrySelector))
	binary.BigEndian.PutUint16(hdr[10:], rangeShift)
	out = append(out, hdr...)

	// Directory first to compute offsets.
	offset := 12 + 16*numTables
	records := make([]byte, 0, 16*numTables)
	for _, tag := range ordered {
		data := final[tag]
		rec := make([]byte, 16)
		copy(rec[:4], tag)
		binary.BigEndian.PutUint32(rec[4:], tableChecksum(data))
		binary.BigEndian.PutUint32(rec[8:], uint32(offset))
		binary.BigEndian.PutUint32(rec[12:], uint32(len(data)))
		records = append(records, rec...)
		offset += pad4(len(data))
	}
	out = append(out, records...)

	// Table data.
	for _, tag := range ordered {
		data := final[tag]
		out = append(out, data...)
		if p := pad4(len(data)) - len(data); p != 0 {
			out = append(out, make([]byte, p)...)
		}
	}

	// checkSumAdjustment: zero the field, compute, set so total is magic.
	// Per the OT spec the head table's own directory checksum is computed
	// with checkSumAdjustment set to zero, so the directory is left alone.
	if head, ok := final["head"]; ok {
		if len(head) < 12 {
			return nil, fmt.Errorf("head table too short: %d bytes", len(head))
		}
		var headOff uint32
		for i, tag := range ordered {
			if tag == "head" {
				headOff = binary.BigEndian.Uint32(records[i*16+8:])
				break
			}
		}
		copy(out[headOff+8:], []byte{0, 0, 0, 0})
		total := wholeFontChecksum(out)
		adj := 0xB1B0AFBA - total
		binary.BigEndian.PutUint32(out[headOff+8:], adj)
	}
	return out, nil
}

func sortTags(tags []string) {
	sort.Strings(tags)
}
