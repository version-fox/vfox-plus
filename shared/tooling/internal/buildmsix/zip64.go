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
	"encoding/binary"
	"errors"
	"fmt"
)

// The records of a ZIP archive, and the versions of the ZIP specification that
// the Microsoft tools write into them.
//
// MakeAppx and SignTool write every package as a ZIP64 archive, although no
// package is large: the classic end record holds the sentinels 0xFFFF and
// 0xFFFFFFFF, the real counts, size and offset are in the ZIP64 end record
// before it, and a locator points at that record. The deployment platform of
// Windows expects that shape. A signed package that ends with a plain ZIP32 end
// record is refused with 0x8007000B ("Opening the package ... failed"), which
// the unsigned install did not show. The central directory says "made by 4.5"
// for every entry, but the entries need only 2.0 and carry no ZIP64 extra
// field: a package with one is refused as well. (openappx measured both on an
// Xbox, with the packages of MakeAppx and SignTool as the model.)
const (
	zipCentralHeaderSignature = 0x02014b50
	zipLocalHeaderSignature   = 0x04034b50
	zipEndSignature           = 0x06054b50
	zip64EndSignature         = 0x06064b50
	zip64LocatorSignature     = 0x07064b50

	zipEndSize       = 22 // the end record, without a comment
	zip64EndSize     = 56 // the ZIP64 end record, without an extensible data sector
	zip64LocatorSize = 20
	zipCentralSize   = 46 // a central directory record, before its name

	zipVersionNeeded = 20 // what an entry needs: DEFLATE (here, stored blocks) at most
	zipVersionZip64  = 45 // the version that has ZIP64: the "made by" of every record
)

// CheckZip64 checks that an archive ends the way the packages of Microsoft do:
// the end record has the sentinels, the ZIP64 end record and its locator are
// before it, and every central directory record is made by version 4.5 and has
// no extra field. The error says what is wrong, and does not name the archive.
func CheckZip64(data []byte) error {
	le := binary.LittleEndian
	n := len(data)
	if n < zipEndSize+zip64LocatorSize+zip64EndSize {
		return errors.New("is too short to end with the ZIP64 records")
	}
	end := data[n-zipEndSize:]
	if le.Uint32(end) != zipEndSignature || le.Uint16(end[20:]) != 0 {
		return errors.New("does not end with the end record (is there a comment after it?)")
	}
	if le.Uint16(end[8:]) != 0xFFFF || le.Uint16(end[10:]) != 0xFFFF || le.Uint32(end[12:]) != 0xFFFFFFFF || le.Uint32(end[16:]) != 0xFFFFFFFF {
		return errors.New("ends with a ZIP32 end record: Windows refuses a signed package without the ZIP64 records (0x8007000B), so the counts, the size and the offset of the central directory must be the sentinels 0xFFFF and 0xFFFFFFFF there")
	}
	locatorStart := n - zipEndSize - zip64LocatorSize
	locator := data[locatorStart : locatorStart+zip64LocatorSize]
	if le.Uint32(locator) != zip64LocatorSignature || le.Uint32(locator[4:]) != 0 || le.Uint32(locator[16:]) != 1 {
		return errors.New("has no ZIP64 end record locator before the end record")
	}
	recordStart := le.Uint64(locator[8:])
	if recordStart > uint64(locatorStart-zip64EndSize) {
		return fmt.Errorf("has a ZIP64 end record at %d, which cannot end where the locator starts (%d)", recordStart, locatorStart)
	}
	record := data[recordStart:locatorStart]
	if le.Uint32(record) != zip64EndSignature {
		return fmt.Errorf("has no ZIP64 end record at %d, where the locator points", recordStart)
	}
	if size := le.Uint64(record[4:]); size != uint64(len(record)-12) || size < zip64EndSize-12 {
		return fmt.Errorf("has a ZIP64 end record of %d bytes after its size field, which does not end where the locator starts", size)
	}
	if made, needed := le.Uint16(record[12:]), le.Uint16(record[14:]); made != zipVersionZip64 || needed != zipVersionZip64 {
		return fmt.Errorf("has a ZIP64 end record made by version %d and needing version %d, want %d for both", made, needed, zipVersionZip64)
	}
	if le.Uint32(record[16:]) != 0 || le.Uint32(record[20:]) != 0 {
		return errors.New("is split over several disks")
	}
	entries, total := le.Uint64(record[24:]), le.Uint64(record[32:])
	size, offset := le.Uint64(record[40:]), le.Uint64(record[48:])
	if entries != total {
		return fmt.Errorf("lists %d entries on its disk and %d in all", entries, total)
	}
	if offset > recordStart || size != recordStart-offset {
		return fmt.Errorf("has a central directory of %d bytes at %d, which does not end where the ZIP64 end record starts (%d)", size, offset, recordStart)
	}
	position := offset
	for i := uint64(0); i < total; i++ {
		if position+zipCentralSize > recordStart || le.Uint32(data[position:]) != zipCentralHeaderSignature {
			return fmt.Errorf("has no central directory record %d at %d", i, position)
		}
		header := data[position:]
		if made := le.Uint16(header[4:]); made != zipVersionZip64 {
			return fmt.Errorf("has central directory record %d made by version %d, want %d as the Microsoft tools write", i, made, zipVersionZip64)
		}
		nameLength, extraLength, commentLength := uint64(le.Uint16(header[28:])), uint64(le.Uint16(header[30:])), uint64(le.Uint16(header[32:]))
		if extraLength != 0 {
			return fmt.Errorf("has an extra field in central directory record %d: Windows refuses ZIP64 extra fields on entries (0x8007000B)", i)
		}
		position += zipCentralSize + nameLength + extraLength + commentLength
	}
	if position != recordStart {
		return fmt.Errorf("has central directory records that end at %d, not where the ZIP64 end record starts (%d)", position, recordStart)
	}
	return nil
}
