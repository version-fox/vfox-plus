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

// WOFF2 known table tags in flag order (WOFF2 spec 4.1).
var woff2KnownTags = []string{
	"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post",
	"cvt ", "fpgm", "glyf", "loca", "prep", "CFF ", "VORG", "EBDT",
	"EBLC", "gasp", "hdmx", "kern", "LTSH", "PCLT", "VDMX", "vhea",
	"vmtx", "BASE", "GDEF", "GPOS", "GSUB", "EBSC", "JSTF", "MATH",
	"CBDT", "CBLC", "COLR", "CPAL", "SVG ", "sbix", "acnt", "avar",
	"bdat", "bloc", "bsln", "cvar", "fdsc", "feat", "fmtx", "fvar",
	"gvar", "hsty", "just", "lcar", "mort", "morx", "opbd", "prop",
	"trak", "Zapf", "Silf", "Glat", "Gloc", "Feat", "Sill",
}

// encode255UInt16 encodes a value in the 255UInt16 variable-length format.
func encode255UInt16(v int) []byte {
	switch {
	case v < 253:
		return []byte{byte(v)}
	case v < 506:
		return []byte{255, byte(v - 253)}
	case v < 761:
		return []byte{254, byte(v - 506)}
	default:
		return []byte{253, byte(v >> 8), byte(v)}
	}
}

// encodeUIntBase128 encodes a value in UIntBase128 (shortest form).
func encodeUIntBase128(v uint32) []byte {
	if v == 0 {
		return []byte{0}
	}
	var tmp []byte
	for x := v; x > 0; x >>= 7 {
		tmp = append([]byte{byte(x & 0x7F)}, tmp...)
	}
	for i := 0; i < len(tmp)-1; i++ {
		tmp[i] |= 0x80
	}
	return tmp
}

// woff2Triplet encodes one (dx, dy) delta pair into a triplet index (bits
// 0-6) plus coordinate bytes. Only two triplet shapes are used: index 0 for
// (0,0) and indices 124-127 (16-bit X and Y) otherwise. Every index is a
// valid WOFF2 triplet, so decoding is exact. The on-curve bit (0x80) and the
// overlap bit are handled by the caller: the on-curve bit is OR'd into the
// stored flag, while overlap bits travel in overlapSimpleBitmap (a
// spec-compliant decoder overwrites bit 6 of each first flag byte with the
// bitmap, so stored bit 6 stays triplet data and widths always decode).
func woff2Triplet(dx, dy int) (index byte, bytes []byte) {
	if dx == 0 && dy == 0 {
		return 0, []byte{0}
	}
	f := byte(124)
	if dx > 0 {
		f |= 1
	}
	if dy > 0 {
		f |= 2
	}
	ax, ay := dx, dy
	if ax < 0 {
		ax = -ax
	}
	if ay < 0 {
		ay = -ay
	}
	if ax > 65535 || ay > 65535 {
		// Out of triplet range; cannot happen for valid glyf deltas after
		// instancing (coordinates stay near int16 range).
		return 0, []byte{0}
	}
	return f, []byte{byte(ax >> 8), byte(ax), byte(ay >> 8), byte(ay)}
}

// woff2Glyf holds one parsed glyph for transformation.
type woff2Glyf struct {
	g          *glyph
	nContours  int
	simple     bool
	bbox       [4]int
	emitBBox   bool
	flags      []byte // expanded per-point flags (for simple)
	endPtsLens []int  // points per contour
	instr      []byte
}

// transformGlyf converts glyf records to the WOFF2 transformed format.
func transformGlyf(glyphs []*glyph, indexFormat int) (transformed []byte, locaOrigLen int, err error) {
	n := len(glyphs)
	wgs := make([]*woff2Glyf, n)
	needOverlap := false
	for i, g := range glyphs {
		wg := &woff2Glyf{g: g}
		if g.isComposite {
			wg.nContours = -1
			wg.bbox = [4]int{g.xMin, g.yMin, g.xMax, g.yMax}
			wg.emitBBox = true
			// Instructions live on the composite record.
			wg.instr = g.instructions
		} else if len(g.coords) == 0 {
			wg.nContours = 0
			if g.xMin != 0 || g.yMin != 0 || g.xMax != 0 || g.yMax != 0 {
				return nil, 0, fmt.Errorf("WOFF2: empty glyph %d has nonzero bbox", i)
			}
		} else {
			wg.nContours = g.nContours
			wg.simple = true
			wg.instr = g.instructions
			// Expanded flags.
			wg.flags = append([]byte(nil), g.flags...)
			// Points per contour.
			prev := -1
			for _, e := range g.endPts {
				wg.endPtsLens = append(wg.endPtsLens, int(e)-prev)
				prev = int(e)
			}
			// Bbox check.
			minX, minY := 1<<30, 1<<30
			maxX, maxY := -1<<30, -1<<30
			for _, p := range g.coords {
				xi, yi := otRoundInt(p[0]), otRoundInt(p[1])
				if xi < minX {
					minX = xi
				}
				if xi > maxX {
					maxX = xi
				}
				if yi < minY {
					minY = yi
				}
				if yi > maxY {
					maxY = yi
				}
			}
			wg.bbox = [4]int{minX, minY, maxX, maxY}
			if wg.bbox != [4]int{g.xMin, g.yMin, g.xMax, g.yMax} {
				wg.emitBBox = true
			}
			for _, f := range wg.flags {
				if f&flagOverlapSimple != 0 {
					needOverlap = true
					break
				}
			}
		}
		wgs[i] = wg
	}
	var nContour, nPoints, flagStream, glyphStream, compStream, bboxStream, instrStream []byte
	bboxBitmap := make([]byte, 4*((n+31)/32))
	var overlapBitmap []byte
	if needOverlap {
		overlapBitmap = make([]byte, (n+7)/8)
	}
	setBit := func(bm []byte, i int) {
		bm[i/8] |= 1 << uint(7-(i%8))
	}
	for i, wg := range wgs {
		nContour = appendI16(nContour, int16(wg.nContours))
		if wg.emitBBox {
			setBit(bboxBitmap, i)
			for _, v := range wg.bbox {
				bboxStream = appendI16(bboxStream, int16(v))
			}
		}
		if needOverlap && wg.simple && len(wg.flags) > 0 && wg.flags[0]&flagOverlapSimple != 0 {
			setBit(overlapBitmap, i)
		}
		if !wg.simple {
			continue
		}
		for _, l := range wg.endPtsLens {
			nPoints = append(nPoints, encode255UInt16(l)...)
		}
		// Flags carry the triplet index (bits 0-6) plus the on-curve bit.
		px, py := 0, 0
		for j, p := range wg.g.coords {
			xi, yi := otRoundInt(p[0]), otRoundInt(p[1])
			idx, b := woff2Triplet(xi-px, yi-py)
			fb := idx
			if wg.g.flags[j]&flagOnCurve == 0 {
				fb |= 0x80
			}
			flagStream = append(flagStream, fb)
			glyphStream = append(glyphStream, b...)
			px, py = xi, yi
		}
		glyphStream = append(glyphStream, encode255UInt16(len(wg.instr))...)
		instrStream = append(instrStream, wg.instr...)
	}
	// Composite streams.
	for _, wg := range wgs {
		if wg.nContours != -1 {
			continue
		}
		for _, c := range wg.g.composites {
			compStream = appendU16(compStream, c.flags)
			compStream = appendU16(compStream, c.gid)
			compStream = append(compStream, c.raw...)
		}
		if len(wg.instr) > 0 {
			glyphStream = append(glyphStream, encode255UInt16(len(wg.instr))...)
			instrStream = append(instrStream, wg.instr...)
		}
	}
	out := []byte{0, 0} // reserved
	if needOverlap {
		out = appendU16(out, 1)
	} else {
		out = appendU16(out, 0)
	}
	out = appendU16(out, uint16(n))
	out = appendU16(out, uint16(indexFormat))
	out = appendU32(out, uint32(len(nContour)))
	out = appendU32(out, uint32(len(nPoints)))
	out = appendU32(out, uint32(len(flagStream)))
	out = appendU32(out, uint32(len(glyphStream)))
	out = appendU32(out, uint32(len(compStream)))
	out = appendU32(out, uint32(len(bboxBitmap)+len(bboxStream)))
	out = appendU32(out, uint32(len(instrStream)))
	out = append(out, nContour...)
	out = append(out, nPoints...)
	out = append(out, flagStream...)
	out = append(out, glyphStream...)
	out = append(out, compStream...)
	out = append(out, bboxBitmap...)
	out = append(out, bboxStream...)
	out = append(out, instrStream...)
	if needOverlap {
		out = append(out, overlapBitmap...)
	}
	entrySize := 2
	if indexFormat == 1 {
		entrySize = 4
	}
	return out, (n + 1) * entrySize, nil
}

// writeWOFF2 packages tables into a WOFF2 file. glyf must be present along
// with its parsed glyphs; the glyf/loca pair gets the transform, everything
// else is stored with the null transform and the concatenated font data is
// compressed with the Brotli library.
func writeWOFF2(sfntVersion uint32, tags []string, tables map[string][]byte, glyphs []*glyph, indexFormat int) ([]byte, error) {
	// Table order: sorted by tag. "loca" sorts after "glyf", which
	// satisfies the WOFF2 order constraints for single fonts.
	ordered := append([]string(nil), tags...)
	sort.Strings(ordered)
	known := map[string]int{}
	for i, t := range woff2KnownTags {
		known[t] = i
	}
	type entry struct {
		flags        byte
		tag          string
		origLength   uint32
		transformLen uint32
		hasTransform bool
		data         []byte
		reconsLen    int // reconstructed length for totalSfntSize
	}
	var entries []entry
	var fontData []byte
	totalSfnt := 12 + 16*len(ordered)
	for _, tag := range ordered {
		data := tables[tag]
		e := entry{tag: tag, origLength: uint32(len(data)), reconsLen: pad4(len(data))}
		idx, ok := known[tag]
		if !ok {
			e.flags = 63
		} else {
			e.flags = byte(idx)
		}
		switch tag {
		case "glyf":
			tg, locaLen, err := transformGlyf(glyphs, indexFormat)
			if err != nil {
				return nil, err
			}
			e.hasTransform = true
			e.transformLen = uint32(len(tg))
			// origLength stays informational; use the pre-transform length.
			e.reconsLen = pad4(len(data))
			_ = locaLen
			e.data = tg
		case "loca":
			// Transformed loca consumes no data; origLength is the
			// reconstructed size (checked by decoders).
			e.hasTransform = true
			e.transformLen = 0
			e.reconsLen = pad4(int(e.origLength))
			e.data = nil
		default:
			e.data = data
		}
		entries = append(entries, e)
		fontData = append(fontData, e.data...)
		totalSfnt += e.reconsLen
	}
	compressed, err := encodeBrotli(fontData)
	if err != nil {
		return nil, err
	}
	out := make([]byte, 0, 48)
	out = append(out, 'w', 'O', 'F', '2')
	out = appendU32(out, sfntVersion)
	out = appendU32(out, 0) // length placeholder
	out = appendU16(out, uint16(len(entries)))
	out = appendU16(out, 0) // reserved
	out = appendU32(out, uint32(totalSfnt))
	out = appendU32(out, uint32(len(compressed)))
	out = appendU16(out, 0) // majorVersion
	out = appendU16(out, 0) // minorVersion
	out = appendU32(out, 0) // metaOffset
	out = appendU32(out, 0) // metaLength
	out = appendU32(out, 0) // metaOrigLength
	out = appendU32(out, 0) // privOffset
	out = appendU32(out, 0) // privLength
	for _, e := range entries {
		// All tables use transform version 0 (null transform for regular
		// tables; the glyf/loca transform for that pair).
		flags := e.flags
		out = append(out, flags)
		if e.flags == 63 {
			out = append(out, e.tag[0], e.tag[1], e.tag[2], e.tag[3])
		}
		out = append(out, encodeUIntBase128(e.origLength)...)
		if e.hasTransform {
			// Transformed glyf carries its length; transformed loca
			// consumes no data, so its length is always zero.
			out = append(out, encodeUIntBase128(e.transformLen)...)
		}
	}
	out = append(out, compressed...)
	putU32(out, 8, uint32(len(out)))
	return out, nil
}
