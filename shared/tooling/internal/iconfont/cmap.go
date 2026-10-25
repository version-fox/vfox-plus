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
)

// cmapSubtable is one parsed cmap encoding record with its character map
// (character code to old glyph ID).
type cmapSubtable struct {
	platformID uint16
	platEncID  uint16
	language   uint16
	format     uint16
	entries    map[uint32]uint16
}

// isUnicode reports whether the subtable uses Unicode codepoints, mirroring
// fontTools cmap table isUnicode.
func (s *cmapSubtable) isUnicode() bool {
	return s.platformID == 0 || (s.platformID == 3 && (s.platEncID == 0 || s.platEncID == 1 || s.platEncID == 10))
}

// isSymbol reports whether the subtable is the Symbol encoding (3,0).
func (s *cmapSubtable) isSymbol() bool { return s.platformID == 3 && s.platEncID == 0 }

func parseCmap(data []byte) ([]*cmapSubtable, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("cmap too short")
	}
	num := int(getU16(data, 2))
	if len(data) < 4+8*num {
		return nil, fmt.Errorf("cmap header too short for %d subtables", num)
	}
	var out []*cmapSubtable
	for i := 0; i < num; i++ {
		plat, enc := getU16(data, 4+8*i), getU16(data, 4+8*i+2)
		off := int(getU32(data, 4+8*i+4))
		if off+2 > len(data) {
			return nil, fmt.Errorf("cmap subtable %d out of range", i)
		}
		st := &cmapSubtable{platformID: plat, platEncID: enc}
		st.language = 0
		format := getU16(data, off)
		st.format = format
		var err error
		switch format {
		case 4:
			st.entries, st.language, err = parseCmap4(data[off:])
		case 12:
			st.entries, st.language, err = parseCmap12(data[off:])
		case 0, 14, 15:
			// Dropped by the subsetter (format 0 cannot be subset easily;
			// 14/15 are variation selectors, unused here).
			continue
		default:
			return nil, fmt.Errorf("unsupported cmap format %d (platform %d encoding %d)", format, plat, enc)
		}
		if err != nil {
			return nil, fmt.Errorf("cmap subtable %d: %w", i, err)
		}
		out = append(out, st)
	}
	return out, nil
}

func parseCmap4(data []byte) (map[uint32]uint16, uint16, error) {
	if len(data) < 14 {
		return nil, 0, fmt.Errorf("cmap format 4 too short")
	}
	language := getU16(data, 4)
	segCount := int(getU16(data, 6)) / 2
	if len(data) < 16+8*segCount {
		return nil, 0, fmt.Errorf("cmap format 4 segments out of range")
	}
	ends := make([]uint16, segCount)
	for i := range ends {
		ends[i] = getU16(data, 14+2*i)
	}
	startsOff := 16 + 2*segCount
	deltasOff := startsOff + 2*segCount
	rangeOffOff := deltasOff + 2*segCount
	entries := map[uint32]uint16{}
	for i := 0; i < segCount; i++ {
		start, end := getU16(data, startsOff+2*i), ends[i]
		delta := int16(getU16(data, deltasOff+2*i))
		ro := getU16(data, rangeOffOff+2*i)
		for c := uint32(start); c <= uint32(end); c++ {
			var gid uint16
			if ro == 0 {
				gid = uint16((int32(c) + int32(delta)) & 0xFFFF)
			} else {
				p := rangeOffOff + 2*i + int(ro) + 2*int(c-uint32(start))
				if p+2 > len(data) {
					return nil, 0, fmt.Errorf("cmap format 4 glyph index out of range")
				}
				v := getU16(data, p)
				if v != 0 {
					gid = uint16((int32(v) + int32(delta)) & 0xFFFF)
				}
			}
			entries[c] = gid
		}
	}
	return entries, language, nil
}

func parseCmap12(data []byte) (map[uint32]uint16, uint16, error) {
	if len(data) < 16 {
		return nil, 0, fmt.Errorf("cmap format 12 too short")
	}
	language := getU32(data, 8)
	nGroups := int(getU32(data, 12))
	if len(data) < 16+12*nGroups {
		return nil, 0, fmt.Errorf("cmap format 12 groups out of range")
	}
	entries := map[uint32]uint16{}
	for i := 0; i < nGroups; i++ {
		start, end, gid := getU32(data, 16+12*i), getU32(data, 16+12*i+4), getU32(data, 16+12*i+8)
		for c := start; c <= end; c++ {
			entries[c] = uint16(gid)
			gid++
		}
	}
	return entries, uint16(language), nil
}

// filterCmap keeps only entries for requested unicodes whose glyph survives,
// remapping glyph IDs. It mirrors fontTools subset cmap.subset_glyphs: empty
// subtables are dropped, and an all-BMP format 12 that duplicates its BMP
// sibling is cleared (then dropped).
func filterCmap(subs []*cmapSubtable, unicodes map[uint32]bool, newGID map[uint16]uint16) []*cmapSubtable {
	sib4 := map[[2]uint16]*cmapSubtable{} // (platform,language) -> format 4
	for _, s := range subs {
		if s.format == 4 {
			sib4[[2]uint16{s.platformID, s.language}] = s
		}
	}
	var filtered []*cmapSubtable
	var bmp12 []*cmapSubtable
	for _, s := range subs {
		kept := make(map[uint32]uint16, len(s.entries))
		var max uint32
		for c, gid := range s.entries {
			if !unicodes[c] {
				continue
			}
			ng, ok := newGID[gid]
			if !ok {
				continue
			}
			kept[c] = ng
			if c > max {
				max = c
			}
		}
		s.entries = kept
		if len(kept) == 0 {
			continue
		}
		if s.format == 12 && max < 0x10000 {
			bmp12 = append(bmp12, s)
		}
		filtered = append(filtered, s)
	}
	// Drop redundant BMP-only format 12 tables duplicating a format 4 sibling.
	for _, s := range bmp12 {
		var sib *cmapSubtable
		switch {
		case s.platformID == 0 && s.platEncID == 4:
			sib = sib4[[2]uint16{0, s.language}]
		case s.platformID == 3 && s.platEncID == 10:
			sib = sib4[[2]uint16{3, s.language}]
		}
		if sib != nil && equalCmap(s.entries, sib.entries) {
			for i, f := range filtered {
				if f == s {
					filtered = append(filtered[:i], filtered[i+1:]...)
					break
				}
			}
		}
	}
	return filtered
}

func equalCmap(a, b map[uint32]uint16) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// compileCmap4 builds a format 4 subtable from sorted entries. Segments use
// idDelta when the mapping is uniform and idRangeOffset otherwise.
func compileCmap4(entries map[uint32]uint16, language uint16) []byte {
	var codes []uint32
	for c := range entries {
		if c <= 0xFFFF {
			codes = append(codes, c)
		}
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	// Build segments of consecutive codes.
	type seg struct{ start, end uint32 }
	var segs []seg
	for _, c := range codes {
		if n := len(segs); n > 0 && c == segs[n-1].end+1 {
			segs[n-1].end = c
		} else {
			segs = append(segs, seg{c, c})
		}
	}
	segs = append(segs, seg{0xFFFF, 0xFFFF})
	segCount := len(segs)
	out := make([]byte, 0, 16+8*segCount)
	out = appendU16(out, 4)
	out = appendU16(out, 0) // length placeholder
	out = appendU16(out, language)
	out = appendU16(out, uint16(segCount*2))
	// searchRange/entrySelector/rangeShift for segCount.
	maxPow := 1
	for maxPow*2 <= segCount {
		maxPow *= 2
	}
	sel := 0
	for p := maxPow; p > 1; p >>= 1 {
		sel++
	}
	out = appendU16(out, uint16(maxPow*2))
	out = appendU16(out, uint16(sel))
	out = appendU16(out, uint16(segCount*2-maxPow*2))
	for _, s := range segs {
		out = appendU16(out, uint16(s.end))
	}
	out = appendU16(out, 0) // reservedPad
	for _, s := range segs {
		out = appendU16(out, uint16(s.start))
	}
	uniform := make([]bool, len(segs))
	deltas := make([]int, len(segs))
	for i, s := range segs {
		if s.start == 0xFFFF {
			deltas[i] = 1
			uniform[i] = true
			continue
		}
		d := int(entries[s.start]) - int(s.start)
		deltas[i], uniform[i] = d, true
		for c := s.start; c <= s.end; c++ {
			if int(entries[c])-int(c) != d {
				uniform[i] = false
				break
			}
		}
	}
	for i := range segs {
		if uniform[i] {
			out = appendU16(out, uint16(int16(deltas[i])))
		} else {
			out = appendU16(out, 0)
		}
	}
	roOff := len(out)
	for range segs {
		out = appendU16(out, 0)
	}
	for i, s := range segs {
		if uniform[i] {
			continue
		}
		here := roOff + 2*i
		putU16(out, here, uint16(len(out)-here))
		for c := s.start; c <= s.end; c++ {
			out = appendU16(out, entries[c])
		}
	}
	putU16(out, 2, uint16(len(out)))
	return out
}

// compileCmap12 builds a format 12 subtable.
func compileCmap12(entries map[uint32]uint16, language uint16) []byte {
	var codes []uint32
	for c := range entries {
		codes = append(codes, c)
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	type grp struct{ start, end, gid uint32 }
	var groups []grp
	for _, c := range codes {
		if n := len(groups); n > 0 && c == groups[n-1].end+1 && uint32(entries[c]) == groups[n-1].gid+(c-groups[n-1].start) {
			groups[n-1].end = c
		} else {
			groups = append(groups, grp{c, c, uint32(entries[c])})
		}
	}
	out := make([]byte, 0, 16+12*len(groups))
	out = appendU16(out, 12)
	out = appendU16(out, 0)
	out = appendU32(out, uint32(16+12*len(groups)))
	out = appendU32(out, uint32(language))
	out = appendU32(out, uint32(len(groups)))
	for _, g := range groups {
		out = appendU32(out, g.start)
		out = appendU32(out, g.end)
		out = appendU32(out, g.gid)
	}
	return out
}

// encodeCmap serializes filtered subtables. Identical subtable bodies share
// one offset, mirroring fontTools cmap compilation.
func encodeCmap(subs []*cmapSubtable) []byte {
	n := len(subs)
	heads := appendU16(nil, 0)
	heads = appendU16(heads, uint16(n))
	off := 4 + 8*n
	var bodies []byte
	seen := map[string]int{} // body bytes -> offset
	for _, s := range subs {
		var body []byte
		switch s.format {
		case 4:
			body = compileCmap4(s.entries, s.language)
		case 12:
			body = compileCmap12(s.entries, s.language)
		}
		heads = appendU16(heads, s.platformID)
		heads = appendU16(heads, s.platEncID)
		if prev, ok := seen[string(body)]; ok {
			heads = appendU32(heads, uint32(prev))
			continue
		}
		seen[string(body)] = off
		heads = appendU32(heads, uint32(off))
		off += len(body)
		bodies = append(bodies, body...)
	}
	return append(heads, bodies...)
}
