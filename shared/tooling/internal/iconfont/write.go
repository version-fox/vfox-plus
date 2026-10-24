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
)

// verifyLigatures checks that every icon symbol still has its ligature,
// mirroring the subset step of icon_font.py.
func (s *subsetter) verifyLigatures() error {
	names := map[string]bool{}
	if s.outGSUB != nil {
		// One glyph can be mapped from several codes (e.g. the test
		// font maps both '0' and 'A' to the same glyph). Range order
		// over a Go map is random, so collect the pairs and apply
		// them in ascending code order: deterministic, and the
		// highest code wins, matching fontTools' getBestCmap order
		// for format-4 cmaps (entries ascend, later overwrites).
		type codeGid struct {
			code uint32
			gid  uint16
		}
		var pairs []codeGid
		for _, sub := range s.outCmap {
			for code, gid := range sub.entries {
				pairs = append(pairs, codeGid{code, gid})
			}
		}
		sort.Slice(pairs, func(i, j int) bool { return pairs[i].code < pairs[j].code })
		reverse := map[uint16]uint32{}
		for _, p := range pairs {
			reverse[p.gid] = p.code
		}
		names = s.outGSUB.ligatureNames(reverse)
	}
	var missing []string
	for _, symbol := range s.icons {
		if !names[symbol] {
			missing = append(missing, symbol)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return fmt.Errorf("the subset lost the ligatures of: %s", strings.Join(missing, ", "))
	}
	return nil
}

// buildTables assembles the final table set.
func (s *subsetter) buildTables() (map[string][]byte, []string, error) {
	tables := map[string][]byte{}
	var tags []string
	add := func(tag string, data []byte) {
		tables[tag] = data
		tags = append(tags, tag)
	}
	// glyf (+loca): encode with GID remap, pad, build loca.
	gidRemap := map[uint16]uint16{}
	for newGID, oldGID := range s.keptOrder {
		gidRemap[oldGID] = uint16(newGID)
	}
	recs := make([][]byte, len(s.outGlyphs))
	for i, g := range s.outGlyphs {
		enc, err := encodeGlyph(g, gidRemap)
		if err != nil {
			return nil, nil, fmt.Errorf("glyph %d: %w", s.keptOrder[i], err)
		}
		recs[i] = enc
	}
	padded, loca, short := buildLoca(recs)
	glyfData := []byte{}
	for _, r := range padded {
		glyfData = append(glyfData, r...)
	}
	add("glyf", glyfData)
	if short {
		putU16(s.head, 50, 0)
	} else {
		putU16(s.head, 50, 1)
	}
	add("loca", encodeLoca(loca, short))
	// hmtx (+hhea).
	hmtxBytes, numHMetrics := trimHMetrics(s.outHmtx)
	add("hmtx", hmtxBytes)
	hhea := append([]byte(nil), s.hhea...)
	putU16(hhea, 34, uint16(numHMetrics))
	s.recalcHhea(hhea)
	add("hhea", hhea)
	if s.vmtx != nil {
		vmtxBytes, numVMetrics := trimHMetrics(s.outVmtx)
		add("vmtx", vmtxBytes)
		if vhea, ok := s.font.tables["vhea"]; ok && len(vhea) >= 36 {
			vh := append([]byte(nil), vhea...)
			putU16(vh, 34, uint16(numVMetrics))
			add("vhea", vh)
		}
	}
	// maxp + head bounds.
	maxp, head, err := s.recalcMaxp()
	if err != nil {
		return nil, nil, err
	}
	add("maxp", maxp)
	add("head", head)
	// cmap.
	add("cmap", encodeCmap(s.outCmap))
	// GSUB (already in new glyph ID space).
	if s.outGSUB != nil {
		enc, err := encodeGSUB(s.outGSUB, func(g uint16) uint16 { return g })
		if err != nil {
			return nil, nil, fmt.Errorf("GSUB: %w", err)
		}
		add("GSUB", enc)
	}
	// HVAR/VVAR.
	if s.outHvar != nil {
		axisCount := 0
		if s.outFvar != nil {
			axisCount = len(s.outFvar.axes)
		}
		add("HVAR", encodeHvar(s.outHvar, axisCount))
	}
	if s.outVvar != nil {
		axisCount := 0
		if s.outFvar != nil {
			axisCount = len(s.outFvar.axes)
		}
		add("VVAR", encodeHvar(s.outVvar, axisCount))
	}
	// fvar/avar/gvar/STAT.
	if s.outFvar != nil {
		add("fvar", encodeFvar(s.outFvar))
	}
	if s.outAvar != nil {
		add("avar", encodeAvar(s.outAvar))
	}
	if s.outGvar != nil {
		var axisTags []string
		if s.outFvar != nil {
			for _, a := range s.outFvar.axes {
				axisTags = append(axisTags, a.tag)
			}
		}
		add("gvar", encodeGvar(axisTags, s.keptOrder, s.outGvar))
	}
	if s.outStat != nil {
		add("STAT", encodeSTAT(s.outStat))
	}
	// cvar: dropped when pinned (see instantiateCvar), else copied.
	if _, ok := s.font.tables["cvar"]; ok {
		add("cvar", append([]byte(nil), s.font.tables["cvar"]...))
	}
	// name/post/OS/2/gasp/prep/cvt.
	enc, err := encodeName(s.outNames)
	if err != nil {
		return nil, nil, fmt.Errorf("name: %w", err)
	}
	add("name", enc)
	// post 3.0: 32-byte header only, glyph names dropped.
	post := append([]byte(nil), s.post[:32]...)
	putU32(post, 0, 0x00030000)
	add("post", post)
	os2 := append([]byte(nil), s.os2...)
	var codes []uint32
	for _, sub := range s.outCmap {
		for code := range sub.entries {
			codes = append(codes, code)
		}
	}
	updateFirstLastCharIndex(os2, codes)
	add("OS/2", os2)
	if s.gasp != nil {
		add("gasp", s.gasp)
	}
	if s.prep != nil {
		add("prep", s.prep)
	}
	if len(s.outCvt) > 0 {
		cvt := make([]byte, 0, 2*len(s.outCvt))
		for _, v := range s.outCvt {
			cvt = appendI16(cvt, v)
		}
		add("cvt ", cvt)
	}
	return tables, tags, nil
}

// recalcMaxp recomputes numGlyphs, font bounds and max* values, mirroring
// maxp.recalc (run at save time because recalcBBoxes is on).
func (s *subsetter) recalcMaxp() ([]byte, []byte, error) {
	maxp := append([]byte(nil), s.maxp...)
	head := append([]byte(nil), s.head...)
	numGlyphs := len(s.outGlyphs)
	putU16(maxp, 4, uint16(numGlyphs))
	const inf = 1 << 30
	xMin, yMin, xMax, yMax := inf, inf, -inf, -inf
	allXMinIsLsb := true
	maxPoints, maxContours := 0, 0
	maxCompPoints, maxCompContours, maxCompElements, maxCompDepth := 0, 0, 0, 0
	byOld := map[uint16]*glyph{}
	for i, g := range s.outGlyphs {
		byOld[s.keptOrder[i]] = g
	}
	for newGID, g := range s.outGlyphs {
		if g.nContours == 0 && !g.isComposite {
			continue
		}
		if s.outHmtx[newGID][1] != g.xMin {
			allXMinIsLsb = false
		}
		if g.xMin < xMin {
			xMin = g.xMin
		}
		if g.yMin < yMin {
			yMin = g.yMin
		}
		if g.xMax > xMax {
			xMax = g.xMax
		}
		if g.yMax > yMax {
			yMax = g.yMax
		}
		if g.nContours > 0 {
			if len(g.coords) > maxPoints {
				maxPoints = len(g.coords)
			}
			if len(g.endPts) > maxContours {
				maxContours = len(g.endPts)
			}
		} else if g.isComposite {
			np, nc, depth := compositeMaxp(g, byOld, 1, map[uint16]bool{})
			if np > maxCompPoints {
				maxCompPoints = np
			}
			if nc > maxCompContours {
				maxCompContours = nc
			}
			if len(g.composites) > maxCompElements {
				maxCompElements = len(g.composites)
			}
			if depth > maxCompDepth {
				maxCompDepth = depth
			}
		}
	}
	if xMin == inf {
		xMin, yMin, xMax, yMax = 0, 0, 0, 0
	}
	putI16 := func(b []byte, off int, v int) {
		putU16(b, off, uint16(int16(v)))
	}
	putI16(head, 36, xMin)
	putI16(head, 38, yMin)
	putI16(head, 40, xMax)
	putI16(head, 42, yMax)
	flags := getU16(head, 16)
	if allXMinIsLsb {
		flags |= 0x2
	} else {
		flags &^= 0x2
	}
	putU16(head, 16, flags)
	if len(maxp) >= 32 {
		putU16(maxp, 6, uint16(maxPoints))
		putU16(maxp, 8, uint16(maxContours))
		putU16(maxp, 10, uint16(maxCompPoints))
		putU16(maxp, 12, uint16(maxCompContours))
		putU16(maxp, 16, uint16(maxCompElements))
		putU16(maxp, 18, uint16(maxCompDepth))
	}
	return maxp, head, nil
}

func compositeMaxp(g *glyph, byOld map[uint16]*glyph, depth int, seen map[uint16]bool) (np, nc, maxDepth int) {
	maxDepth = depth
	for _, c := range g.composites {
		base := byOld[c.gid]
		if base == nil || base.nContours == 0 {
			continue
		}
		if base.nContours > 0 {
			np += len(base.coords)
			nc += len(base.endPts)
		} else if !seen[c.gid] {
			seen[c.gid] = true
			p2, c2, d2 := compositeMaxp(base, byOld, depth+1, seen)
			delete(seen, c.gid)
			np += p2
			nc += c2
			if d2 > maxDepth {
				maxDepth = d2
			}
		}
	}
	return np, nc, maxDepth
}

// writeSFNT serializes the subset font as SFNT.
func (s *subsetter) writeSFNT() ([]byte, error) {
	tables, tags, err := s.buildTables()
	if err != nil {
		return nil, err
	}
	return writeSFNT(s.font.sfntVersion, tags, tables)
}

// writeWOFF2 serializes the subset font as WOFF2.
func (s *subsetter) writeWOFF2() ([]byte, error) {
	tables, tags, err := s.buildTables()
	if err != nil {
		return nil, err
	}
	indexFormat := 0
	if getU16(tables["head"], 50) == 1 {
		indexFormat = 1
	}
	return writeWOFF2(s.font.sfntVersion, tags, tables, s.outGlyphs, indexFormat)
}

// recalcHhea recomputes advanceWidthMax, minLeftSideBearing,
// minRightSideBearing and xMaxExtent, mirroring hhea.compile with
// recalcBBoxes on.
func (s *subsetter) recalcHhea(hhea []byte) {
	if len(hhea) < 36 || len(s.outHmtx) == 0 {
		return
	}
	maxAdv := s.outHmtx[0][0]
	for _, m := range s.outHmtx[1:] {
		if m[0] > maxAdv {
			maxAdv = m[0]
		}
	}
	putU16(hhea, 10, uint16(maxAdv))
	const inf = 1 << 30
	minLSB, minRSB, xMaxExtent := inf, inf, -inf
	for i, g := range s.outGlyphs {
		if g.nContours == 0 && !g.isComposite {
			continue
		}
		adv, lsb := s.outHmtx[i][0], s.outHmtx[i][1]
		boundsWidth := g.xMax - g.xMin
		rsb := adv - lsb - boundsWidth
		extent := lsb + boundsWidth
		if lsb < minLSB {
			minLSB = lsb
		}
		if rsb < minRSB {
			minRSB = rsb
		}
		if extent > xMaxExtent {
			xMaxExtent = extent
		}
	}
	if minLSB == inf {
		minLSB, minRSB, xMaxExtent = 0, 0, 0
	}
	putU16(hhea, 12, uint16(int16(minLSB)))
	putU16(hhea, 14, uint16(int16(minRSB)))
	putU16(hhea, 16, uint16(int16(xMaxExtent)))
}
