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
)

// Tables the subsetter understands. Anything else (except the drop list) is
// rejected loudly rather than silently corrupted. This mirrors fontTools
// subset with passthrough_tables=False, except the tool errors instead of
// warning-and-dropping so unsupported fonts fail visibly.
var supportedTables = map[string]bool{
	"head": true, "hhea": true, "maxp": true, "OS/2": true,
	"hmtx": true, "vmtx": true, "vhea": true, "cmap": true,
	"loca": true, "glyf": true, "name": true, "post": true,
	"gasp": true, "prep": true, "GSUB": true, "HVAR": true,
	"VVAR": true, "fvar": true, "avar": true, "gvar": true,
	"STAT": true, "cvar": true, "cvt ": true,
}

// subsetter holds the parsed font and the subset state.
type subsetter struct {
	font      *sfntFont
	numGlyphs int

	head []byte
	hhea []byte
	maxp []byte
	os2  []byte
	post []byte
	gasp []byte
	prep []byte
	cvt  []int16

	hmtx   [][2]int
	vmtx   [][2]int // nil unless present
	glyphs []*glyph
	loca   []uint32

	cmap  []*cmapSubtable
	names []nameRecord
	gsub  *gsubTable
	hvar  *hvarTable
	vvar  *hvarTable
	fvar  *fvarTable
	avar  *avarTable
	gvar  *gvarTable
	stat  *statTable
	cvarN int // cvt count

	keep      []bool
	newIndex  []int
	keptOrder []uint16 // new GID -> old GID

	// Symbols for ligature verification.
	icons map[string]string
	// Requested unicodes (icon codepoints + ligature characters).
	unicodes map[uint32]bool
	pinned   bool

	// Subset results (subset order, pre-pinning).
	outGlyphs []*glyph
	outHmtx   [][2]int
	outVmtx   [][2]int
	outCmap   []*cmapSubtable
	outGSUB   *gsubTable
	outHvar   *hvarTable
	outVvar   *hvarTable
	outGvar   map[uint16][]gvarTuple // old GID -> tuples
	outNames  []nameRecord
	outFvar   *fvarTable
	outAvar   *avarTable
	outStat   *statTable
	outCvt    []int16
}

func newSubsetter(f *sfntFont, icons map[string]string, codepoints map[string]uint32) (*subsetter, error) {
	s := &subsetter{font: f, icons: icons}
	t := f.tables
	get := func(tag string) ([]byte, bool) { b, ok := t[tag]; return b, ok }
	// Reject unsupported tables (after applying the drop list).
	for tag := range t {
		dropped := false
		for _, d := range subsetDropTables {
			if tag == d {
				dropped = true
				break
			}
		}
		if !dropped && !supportedTables[tag] {
			return nil, fmt.Errorf("unsupported table %q", tag)
		}
	}
	var err error
	if s.numGlyphs, err = parseNumGlyphs(t); err != nil {
		return nil, err
	}
	if s.head, err = requiredTable(t, "head", 54); err != nil {
		return nil, err
	}
	if s.hhea, err = requiredTable(t, "hhea", 36); err != nil {
		return nil, err
	}
	if s.maxp, err = requiredTable(t, "maxp", 6); err != nil {
		return nil, err
	}
	if s.os2, err = requiredTable(t, "OS/2", 78); err != nil {
		return nil, err
	}
	if s.post, err = requiredTable(t, "post", 32); err != nil {
		return nil, err
	}
	if s.gasp, _ = get("gasp"); s.gasp != nil {
		s.gasp = append([]byte(nil), s.gasp...)
	}
	if s.prep, _ = get("prep"); s.prep != nil {
		s.prep = append([]byte(nil), s.prep...)
	}
	numHMetrics := int(getU16(s.hhea, 34))
	if b, ok := get("hmtx"); ok {
		if s.hmtx, err = parseHmtx(b, s.numGlyphs, numHMetrics); err != nil {
			return nil, fmt.Errorf("hmtx: %w", err)
		}
	} else {
		return nil, fmt.Errorf("missing hmtx table")
	}
	if b, ok := get("vmtx"); ok {
		numVMetrics := s.numGlyphs
		if vhea, ok2 := get("vhea"); ok2 && len(vhea) >= 34 {
			numVMetrics = int(getU16(vhea, 34))
		}
		if s.vmtx, err = parseHmtx(b, s.numGlyphs, numVMetrics); err != nil {
			return nil, fmt.Errorf("vmtx: %w", err)
		}
	}
	longLoca := getU16(s.head, 50) == 1
	if b, ok := get("glyf"); ok {
		lb, ok2 := get("loca")
		if !ok2 {
			return nil, fmt.Errorf("glyf without loca")
		}
		if s.loca, err = parseLoca(lb, s.numGlyphs, longLoca); err != nil {
			return nil, fmt.Errorf("loca: %w", err)
		}
		recs, err := splitGlyf(b, s.loca)
		if err != nil {
			return nil, err
		}
		for i, r := range recs {
			g, err := parseGlyph(r)
			if err != nil {
				return nil, fmt.Errorf("glyph %d: %w", i, err)
			}
			s.glyphs = append(s.glyphs, g)
		}
	} else {
		return nil, fmt.Errorf("missing glyf table")
	}
	// head bounds etc. are recomputed later (recalcBBoxes behavior).
	if b, ok := get("cmap"); ok {
		subs, err := parseCmap(b)
		if err != nil {
			return nil, fmt.Errorf("cmap: %w", err)
		}
		for _, sub := range subs {
			if sub.isUnicode() && !sub.isSymbol() && sub.format != 0 {
				s.cmap = append(s.cmap, sub)
			}
		}
	} else {
		return nil, fmt.Errorf("missing cmap table")
	}
	if b, ok := get("name"); ok {
		recs, err := parseName(b)
		if err != nil {
			return nil, fmt.Errorf("name: %w", err)
		}
		// name_legacy=False: keep Unicode records only.
		for _, r := range recs {
			if isUnicodeName(r.platformID, r.platEncID) {
				s.names = append(s.names, r)
			}
		}
	}
	if b, ok := get("GSUB"); ok {
		if s.gsub, err = parseGSUB(b); err != nil {
			return nil, fmt.Errorf("GSUB: %w", err)
		}
	}
	if b, ok := get("HVAR"); ok {
		if s.hvar, err = parseHvar(b); err != nil {
			return nil, fmt.Errorf("HVAR: %w", err)
		}
	}
	if b, ok := get("VVAR"); ok {
		if s.vvar, err = parseHvar(b); err != nil {
			return nil, fmt.Errorf("VVAR: %w", err)
		}
	}
	if b, ok := get("fvar"); ok {
		if s.fvar, err = parseFvar(b); err != nil {
			return nil, fmt.Errorf("fvar: %w", err)
		}
	}
	if b, ok := get("avar"); ok {
		var tags []string
		if s.fvar != nil {
			for _, a := range s.fvar.axes {
				tags = append(tags, a.tag)
			}
		}
		if s.avar, err = parseAvar(b, tags); err != nil {
			return nil, fmt.Errorf("avar: %w", err)
		}
	}
	if b, ok := get("gvar"); ok {
		points := map[uint16]int{}
		for i, g := range s.glyphs {
			points[uint16(i)] = pointCountFor(g)
		}
		var tags []string
		if s.fvar != nil {
			for _, a := range s.fvar.axes {
				tags = append(tags, a.tag)
			}
		}
		if s.gvar, err = parseGvar(b, tags, points); err != nil {
			return nil, fmt.Errorf("gvar: %w", err)
		}
	}
	if b, ok := get("STAT"); ok {
		if s.stat, err = parseSTAT(b); err != nil {
			return nil, fmt.Errorf("STAT: %w", err)
		}
	}
	if b, ok := get("cvt "); ok {
		if len(b)%2 != 0 {
			return nil, fmt.Errorf("cvt odd length")
		}
		for i := 0; i < len(b); i += 2 {
			s.cvt = append(s.cvt, getI16(b, i))
		}
		s.outCvt = append([]int16(nil), s.cvt...)
	}
	if _, ok := get("cvar"); ok {
		// cvar parses like gvar with width-1 tuples; handled in the pin step.
		s.cvarN = len(s.cvt)
	}
	return s, nil
}

func parseNumGlyphs(t map[string][]byte) (int, error) {
	b, ok := t["maxp"]
	if !ok || len(b) < 6 {
		return 0, fmt.Errorf("missing maxp table")
	}
	return int(getU16(b, 4)), nil
}

func requiredTable(t map[string][]byte, tag string, minLen int) ([]byte, error) {
	b, ok := t[tag]
	if !ok {
		return nil, fmt.Errorf("missing %s table", tag)
	}
	if len(b) < minLen {
		return nil, fmt.Errorf("%s table too short: %d", tag, len(b))
	}
	return append([]byte(nil), b...), nil
}

// computeKeep determines the kept glyph set: glyph 0, cmap closure over the
// icon codepoints plus ligature characters, and composite closure.
func (s *subsetter) computeKeep(icons map[string]string, codepoints map[string]uint32) {
	s.keep = make([]bool, s.numGlyphs)
	unicodes := map[uint32]bool{}
	for _, symbol := range icons {
		unicodes[codepoints[symbol]] = true
	}
	for _, c := range ligatureChars {
		unicodes[uint32(c)] = true
	}
	s.unicodes = unicodes
	for _, sub := range s.cmap {
		for code, gid := range sub.entries {
			if unicodes[code] && int(gid) < s.numGlyphs {
				s.keep[gid] = true
			}
		}
	}
	s.keep[0] = true
	// Composite closure.
	for changed := true; changed; {
		changed = false
		for gid, keep := range s.keep {
			if !keep || !s.glyphs[gid].isComposite {
				continue
			}
			for _, c := range s.glyphs[gid].composites {
				if int(c.gid) < s.numGlyphs && !s.keep[c.gid] {
					s.keep[c.gid] = true
					changed = true
				}
			}
		}
	}
	s.newIndex = make([]int, s.numGlyphs)
	for i := range s.newIndex {
		s.newIndex[i] = -1
	}
	for gid, keep := range s.keep {
		if keep {
			s.newIndex[gid] = len(s.keptOrder)
			s.keptOrder = append(s.keptOrder, uint16(gid))
		}
	}
}

// run executes subsetting then optional pinning.
func (s *subsetter) run(icons map[string]string, codepoints map[string]uint32, pins map[string]float64) error {
	s.computeKeep(icons, codepoints)
	if err := s.subsetTables(); err != nil {
		return err
	}
	if len(pins) > 0 {
		s.pinned = true
		if err := s.instantiate(pins); err != nil {
			return err
		}
	}
	return s.verifyLigatures()
}

func (s *subsetter) subsetTables() error {
	// Glyphs: expand (already parsed), recalc bounds (recalcBBoxes=True),
	// keep subset in original order.
	for _, gid := range s.keptOrder {
		g := s.glyphs[gid]
		g.recalcBounds(s.resolveGlyph)
		s.outGlyphs = append(s.outGlyphs, g)
		s.outHmtx = append(s.outHmtx, s.hmtx[gid])
		if s.vmtx != nil {
			s.outVmtx = append(s.outVmtx, s.vmtx[gid])
		}
	}
	// cmap filter.
	newIndex := s.newIndexToMap()
	s.outCmap = filterCmap(s.cmap, s.unicodes, newIndex)
	// Remap composite component glyphs to new IDs (closure already ran).
	for _, gid := range s.keptOrder {
		g := s.glyphs[gid]
		if !g.isComposite {
			continue
		}
		for i := range g.composites {
			ng, ok := newIndex[g.composites[i].gid]
			if !ok {
				return fmt.Errorf("composite component GID %d dropped by subset", g.composites[i].gid)
			}
			g.composites[i].gid = ng
		}
	}
	// GSUB prune + remap to new glyph IDs.
	if s.gsub != nil {
		keep := map[uint16]bool{}
		for gid, k := range s.keep {
			if k {
				keep[uint16(gid)] = true
			}
		}
		pruned, err := pruneGSUB(s.gsub, keep)
		if err != nil {
			return err
		}
		remapGSUB(pruned, newIndex)
		s.outGSUB = pruned
	}
	// HVAR/VVAR subset.
	if s.hvar != nil {
		hv := *s.hvar
		hv.store = cloneVarStore(&s.hvar.store)
		subsetHvar(&hv, s.keptOrder)
		s.outHvar = &hv
	}
	if s.vvar != nil {
		vv := *s.vvar
		vv.store = cloneVarStore(&s.vvar.store)
		subsetHvar(&vv, s.keptOrder)
		s.outVvar = &vv
	}
	// gvar filter (re-encoded at write).
	if s.gvar != nil {
		s.outGvar = map[uint16][]gvarTuple{}
		for _, gid := range s.keptOrder {
			if tups, ok := s.gvar.variations[gid]; ok {
				cp := make([]gvarTuple, len(tups))
				copy(cp, tups)
				s.outGvar[uint16(gid)] = cp
			}
		}
	}
	// name kept whole (re-encoded at write).
	s.outNames = append([]nameRecord(nil), s.names...)
	// fvar/avar/STAT kept whole until pinning.
	s.outFvar = s.fvar
	s.outAvar = s.avar
	s.outStat = s.stat
	if _, ok := s.font.tables["cvar"]; ok {
		// Parsed lazily in the pin step.
	}
	return nil
}

func (s *subsetter) newIndexToMap() map[uint16]uint16 {
	out := map[uint16]uint16{}
	for old, n := range s.newIndex {
		if n >= 0 {
			out[uint16(old)] = uint16(n)
		}
	}
	return out
}

func (s *subsetter) resolveGlyph(gid uint16) *glyph {
	if int(gid) < len(s.glyphs) {
		return s.glyphs[gid]
	}
	return nil
}

func cloneVarStore(vs *varStore) varStore {
	out := varStore{}
	for _, r := range vs.regions {
		out.regions = append(out.regions, varRegion{axes: append([][3]float64(nil), r.axes...)})
	}
	for _, vd := range vs.varDatas {
		nvd := varData{regionIndexes: append([]uint16(nil), vd.regionIndexes...)}
		for _, row := range vd.rows {
			nvd.rows = append(nvd.rows, append([]int(nil), row...))
		}
		out.varDatas = append(out.varDatas, nvd)
	}
	return out
}

// subsetHvar filters VarData rows and DeltaSetIndexMaps to kept glyphs,
// mirroring fontTools HVAR.subset_glyphs (without retain_gids/emptied).
func subsetHvar(t *hvarTable, keep []uint16) {
	// Filter maps to kept glyphs (new order).
	filterMap := func(m []uint32) []uint32 {
		if m == nil {
			return nil
		}
		out := make([]uint32, len(keep))
		for i, gid := range keep {
			if int(gid) < len(m) {
				out[i] = m[gid]
			} else {
				out[i] = noVariationIndex
			}
		}
		return out
	}
	t.advMap = filterMap(t.advMap)
	t.lsbMap = filterMap(t.lsbMap)
	t.rsbMap = filterMap(t.rsbMap)
	// Collect used (outer, inner) varidx values.
	used := map[uint32]bool{}
	if t.advMap == nil {
		// Direct mapping: (0, gid) per kept glyph.
		for _, gid := range keep {
			used[uint32(gid)] = true
		}
	} else {
		for _, v := range t.advMap {
			used[v] = true
		}
	}
	for _, m := range [][]uint32{t.lsbMap, t.rsbMap} {
		for _, v := range m {
			used[v] = true
		}
	}
	delete(used, noVariationIndex)
	// Prune VarData rows to used inners per outer, remap map entries.
	innerRemap := make([]map[int]int, len(t.store.varDatas))
	for di := range t.store.varDatas {
		vd := &t.store.varDatas[di]
		innerRemap[di] = map[int]int{}
		// Direct mapping addresses VarData 0 by glyph ID.
		var keepRows []int
		if t.advMap == nil && di == 0 {
			for _, gid := range keep {
				if int(gid) < len(vd.rows) {
					innerRemap[di][int(gid)] = len(keepRows)
					keepRows = append(keepRows, int(gid))
				} else {
					innerRemap[di][int(gid)] = len(keepRows)
					keepRows = append(keepRows, -1)
				}
			}
		} else {
			for row := range vd.rows {
				if used[uint32(di)<<16|uint32(row)] {
					innerRemap[di][row] = len(keepRows)
					keepRows = append(keepRows, row)
				}
			}
		}
		newRows := make([][]int, len(keepRows))
		for i, r := range keepRows {
			if r >= 0 {
				newRows[i] = vd.rows[r]
			} else {
				newRows[i] = make([]int, len(vd.regionIndexes))
			}
		}
		vd.rows = newRows
	}
	remapEntry := func(v uint32) uint32 {
		if v == noVariationIndex {
			return v
		}
		outer, inner := int(v>>16), int(v&0xFFFF)
		if outer < len(innerRemap) {
			if nv, ok := innerRemap[outer][inner]; ok {
				return uint32(outer)<<16 | uint32(nv)
			}
		}
		return v
	}
	for i := range t.advMap {
		t.advMap[i] = remapEntry(t.advMap[i])
	}
	for i := range t.lsbMap {
		t.lsbMap[i] = remapEntry(t.lsbMap[i])
	}
	for i := range t.rsbMap {
		t.rsbMap[i] = remapEntry(t.rsbMap[i])
	}
}
