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
	"fmt"
	"sort"
	"strings"
	"unicode/utf16"
)

// nameRecord is one name table record.
type nameRecord struct {
	platformID uint16
	platEncID  uint16
	langID     uint16
	nameID     uint16
	str        string
}

func parseName(data []byte) ([]nameRecord, error) {
	if len(data) < 6 {
		return nil, fmt.Errorf("name table too short")
	}
	count := int(getU16(data, 2))
	strOff := int(getU16(data, 4))
	if len(data) < 6+12*count {
		return nil, fmt.Errorf("name records out of range")
	}
	var out []nameRecord
	for i := 0; i < count; i++ {
		rec := data[6+12*i:]
		r := nameRecord{
			platformID: getU16(rec, 0),
			platEncID:  getU16(rec, 2),
			langID:     getU16(rec, 4),
			nameID:     getU16(rec, 6),
		}
		length := int(getU16(rec, 8))
		offset := int(getU16(rec, 10))
		if strOff+offset+length > len(data) {
			return nil, fmt.Errorf("name record %d string out of range", i)
		}
		raw := data[strOff+offset : strOff+offset+length]
		s, err := decodeNameString(r.platformID, r.platEncID, raw)
		if err != nil {
			return nil, fmt.Errorf("name record %d: %w", i, err)
		}
		r.str = s
		out = append(out, r)
	}
	return out, nil
}

func decodeNameString(platformID, platEncID uint16, raw []byte) (string, error) {
	switch {
	case platformID == 0 || (platformID == 3 && (platEncID == 0 || platEncID == 1)):
		if len(raw)%2 != 0 {
			return "", fmt.Errorf("odd length UTF-16 string")
		}
		u16s := make([]uint16, len(raw)/2)
		for i := range u16s {
			u16s[i] = getU16(raw, 2*i)
		}
		return string(utf16.Decode(u16s)), nil
	case platformID == 1 && platEncID == 0:
		return macRomanToString(raw), nil
	default:
		return "", fmt.Errorf("unsupported name encoding platform %d encoding %d", platformID, platEncID)
	}
}

func encodeNameString(platformID, platEncID uint16, s string) ([]byte, error) {
	switch {
	case platformID == 0 || (platformID == 3 && (platEncID == 0 || platEncID == 1)):
		u16s := utf16.Encode([]rune(s))
		out := make([]byte, 0, 2*len(u16s))
		for _, u := range u16s {
			out = appendU16(out, u)
		}
		return out, nil
	case platformID == 1 && platEncID == 0:
		return stringToMacRoman(s)
	default:
		return nil, fmt.Errorf("unsupported name encoding platform %d encoding %d", platformID, platEncID)
	}
}

// isUnicodeName mirrors fontTools NameRecord.isUnicode for the name_legacy=False
// filter: only Unicode records are kept.
func isUnicodeName(platformID, platEncID uint16) bool {
	return platformID == 0 || (platformID == 3 && (platEncID == 0 || platEncID == 1))
}

// encodeName sorts records (platform, encoding, language, ID, string bytes —
// mirroring NameRecord.__lt__), dedups identical strings and packs the table.
func encodeName(records []nameRecord) ([]byte, error) {
	type key struct {
		r nameRecord
		b []byte
	}
	keys := make([]key, 0, len(records))
	for _, r := range records {
		b, err := encodeNameString(r.platformID, r.platEncID, r.str)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key{r, b})
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.r.platformID != b.r.platformID {
			return a.r.platformID < b.r.platformID
		}
		if a.r.platEncID != b.r.platEncID {
			return a.r.platEncID < b.r.platEncID
		}
		if a.r.langID != b.r.langID {
			return a.r.langID < b.r.langID
		}
		if a.r.nameID != b.r.nameID {
			return a.r.nameID < b.r.nameID
		}
		return string(a.b) < string(b.b)
	})
	strOff := 6 + 12*len(keys)
	out := appendU16(nil, 0)
	out = appendU16(out, uint16(len(keys)))
	out = appendU16(out, uint16(strOff))
	done := map[string][2]int{}
	var pool []byte
	for _, k := range keys {
		sk := string(k.b)
		loc, ok := done[sk]
		if !ok {
			loc = [2]int{len(pool), len(k.b)}
			done[sk] = loc
			pool = append(pool, k.b...)
		}
		out = appendU16(out, k.r.platformID)
		out = appendU16(out, k.r.platEncID)
		out = appendU16(out, k.r.langID)
		out = appendU16(out, k.r.nameID)
		out = appendU16(out, uint16(loc[1]))
		out = appendU16(out, uint16(loc[0]))
	}
	return append(out, pool...), nil
}

// variationNameIDs collects referenced name IDs > 255 from fvar and STAT,
// mirroring fontTools instancer.names.getVariationNameIDs.
func variationNameIDs(fvar *fvarTable, stat *statTable) map[uint16]bool {
	used := map[uint16]bool{}
	if fvar != nil {
		for _, a := range fvar.axes {
			used[a.nameID] = true
		}
		// NOTE: instances are handled by the caller, which compares
		// before/after sets; postscriptNameID 0xFFFF is skipped.
	}
	_ = used
	out := map[uint16]bool{}
	add := func(id uint16) {
		if id > 255 {
			out[id] = true
		}
	}
	if fvar != nil {
		for _, a := range fvar.axes {
			add(a.nameID)
		}
	}
	if stat != nil {
		for _, a := range stat.axes {
			add(a.nameID)
		}
		for _, v := range stat.values {
			add(v.valueNameID)
		}
		add(stat.elidedID)
	}
	return out
}

// pruneNames drops records whose name IDs are in exclude.
func pruneNames(records []nameRecord, exclude map[uint16]bool) []nameRecord {
	var out []nameRecord
	for _, r := range records {
		if !exclude[r.nameID] {
			out = append(out, r)
		}
	}
	return out
}

// WDTH_VALUE_TO_OS2_WIDTH_CLASS mirrors the fontTools table used by
// set_default_weight_width_slant.
var wdthToWidthClass = [][2]float64{
	{50, 1}, {62.5, 2}, {75, 3}, {87.5, 4}, {100, 5},
	{112.5, 6}, {125, 7}, {150, 8}, {200, 9},
}

func wdthToClass(v float64) int {
	m := map[float64]float64{}
	for _, kv := range wdthToWidthClass {
		m[kv[0]] = kv[1]
	}
	return otRoundInt(piecewiseLinearMap(v, m))
}

// updateOS2ForPins mirrors set_default_weight_width_slant: pinned wght sets
// usWeightClass, pinned wdth sets usWidthClass, pinned slnt sets italicAngle.
// OS/2 and post are fixed-size headers edited in place.
func updateOS2ForPins(os2 []byte, post []byte, pins map[string]float64) {
	if w, ok := pins["wght"]; ok && len(os2) >= 6 {
		v := otRoundInt(w)
		if v < 1 {
			v = 1
		}
		if v > 1000 {
			v = 1000
		}
		putU16(os2, 4, uint16(v))
	}
	if w, ok := pins["wdth"]; ok && len(os2) >= 8 {
		v := w
		if v < 50 {
			v = 50
		}
		if v > 200 {
			v = 200
		}
		putU16(os2, 6, uint16(wdthToClass(v)))
	}
	if s, ok := pins["slnt"]; ok && len(post) >= 8 {
		a := s
		if a < -90 {
			a = -90
		}
		if a > 90 {
			a = 90
		}
		putU32(post, 4, uint32(floatToFixed(a)))
	}
}

// recalcAvgCharWidth mirrors OS/2.recalcAvgCharWidth: otRound of the mean of
// positive advance widths.
func recalcAvgCharWidth(metrics [][2]int) int {
	sum, n := 0, 0
	for _, m := range metrics {
		if m[0] > 0 {
			sum += m[0]
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return otRoundInt(float64(sum) / float64(n))
}

// updateFirstLastCharIndex mirrors OS/2.compile's updateFirstAndLastCharIndex.
// usFirstCharIndex is at offset 64 and usLastCharIndex at 66 in all OS/2
// versions.
func updateFirstLastCharIndex(os2 []byte, codes []uint32) {
	if len(os2) < 68 {
		return
	}
	if len(codes) == 0 {
		return
	}
	minC, maxC := codes[0], codes[0]
	for _, c := range codes[1:] {
		if c < minC {
			minC = c
		}
		if c > maxC {
			maxC = c
		}
	}
	if minC > 0xFFFF {
		minC = 0xFFFF
	}
	if maxC > 0xFFFF {
		maxC = 0xFFFF
	}
	putU16(os2, 64, uint16(minC))
	putU16(os2, 66, uint16(maxC))
}

// macRoman tables for (1,0) name records. Only ASCII round-trips; other bytes
// map through a minimal table sufficient for decoding.
func macRomanToString(raw []byte) string {
	var sb strings.Builder
	for _, b := range raw {
		if b < 0x80 {
			sb.WriteByte(b)
		} else {
			sb.WriteRune(macRomanDecode(b))
		}
	}
	return sb.String()
}

func stringToMacRoman(s string) ([]byte, error) {
	var out []byte
	for _, r := range s {
		if r < 0x80 {
			out = append(out, byte(r))
		} else if b, ok := macRomanEncode(r); ok {
			out = append(out, b)
		} else {
			return nil, fmt.Errorf("rune %U not encodable in Mac Roman", r)
		}
	}
	return out, nil
}

// macRomanDecode maps the 0x80-0xFF range (standard Mac Roman).
func macRomanDecode(b byte) rune {
	const table = "ÄÅÇÉÑÖÜáàâäãåçéèêëíìîïñóòôöõúùûü†°¢£§•¶ß®©™´¨≠ÆØ∞±≤≥¥µ∂∑∏π∫ªºΩæø¿¡¬√ƒ≈∆«»… ÀÃÕŒœ–—“”‘’÷◊ÿŸ⁄€‹›ﬁﬂ‡·‚„‰ÂÊÁËÈÍÎÏÌÓÔÒÚÛÙıˆ˜¯˘˙˚¸˝˛ˇ"
	if b < 0x80 {
		return rune(b)
	}
	for i, r := range table {
		if 0x80+i == int(b) {
			return r
		}
	}
	return rune(b)
}

func macRomanEncode(r rune) (byte, bool) {
	const table = "ÄÅÇÉÑÖÜáàâäãåçéèêëíìîïñóòôöõúùûü†°¢£§•¶ß®©™´¨≠ÆØ∞±≤≥¥µ∂∑∏π∫ªºΩæø¿¡¬√ƒ≈∆«»… ÀÃÕŒœ–—“”‘’÷◊ÿŸ⁄€‹›ﬁﬂ‡·‚„‰ÂÊÁËÈÍÎÏÌÓÔÒÚÛÙıˆ˜¯˘˙˚¸˝˛ˇ"
	if r < 0x80 {
		return byte(r), true
	}
	for i, c := range table {
		if c == r {
			return byte(0x80 + i), true
		}
	}
	return 0, false
}
