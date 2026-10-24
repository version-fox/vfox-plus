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

// instantiate applies variable-font pinning, mirroring fontTools
// instantiateVariableFont for pin-only axis limits (plus the subset-then-pin
// order used by icon_font.py). Tables the instancer would touch but this
// font family does not contain (MVAR, GDEF/GPOS variations, COLR, CFF2,
// VARC) are rejected if present instead of being silently mishandled.
func (s *subsetter) instantiate(pins map[string]float64) error {
	if s.fvar == nil {
		return fmt.Errorf("cannot pin: no fvar table")
	}
	// Presence checks mirroring sanityCheckVariableTables.
	if _, ok := s.font.tables["MVAR"]; ok {
		return fmt.Errorf("MVAR instancing not supported")
	}
	if _, ok := s.font.tables["VARC"]; ok {
		return fmt.Errorf("VARC instancing not supported")
	}
	if _, ok := s.font.tables["CFF2"]; ok {
		return fmt.Errorf("CFF2 instancing not supported")
	}
	if _, ok := s.font.tables["COLR"]; ok {
		return fmt.Errorf("COLR instancing not supported")
	}
	fvarTags := map[string]bool{}
	for _, a := range s.outFvar.axes {
		fvarTags[a.tag] = true
	}
	// Unknown axes are rejected like fontTools ("Cannot limit ...").
	var unknown []string
	for tag := range pins {
		if !fvarTags[tag] {
			unknown = append(unknown, tag)
		}
	}
	if len(unknown) > 0 {
		sort.Strings(unknown)
		return fmt.Errorf("cannot limit: %v not present in fvar", unknown)
	}
	if s.outAvar != nil && s.outAvar.version>>16 != 1 {
		return fmt.Errorf("unsupported avar version %#x", s.outAvar.version)
	}
	// Clamp pins and normalize (user -> NDC with avar, F2DOT14-quantized).
	norm := map[string]pinResult{}
	axisByTag := map[string]*fvarAxis{}
	for i := range s.outFvar.axes {
		axisByTag[s.outFvar.axes[i].tag] = &s.outFvar.axes[i]
	}
	segByTag := map[string]avarSegments{}
	if s.outAvar != nil {
		for i, a := range s.outFvar.axes {
			if i < len(s.outAvar.segments) {
				segByTag[a.tag] = s.outAvar.segments[i]
			}
		}
	}
	for tag, v := range pins {
		a := axisByTag[tag]
		c := v
		if c < a.min {
			c = a.min
		}
		if c > a.max {
			c = a.max
		}
		ndc := normalizeValue(c, a.min, a.def, a.max)
		if seg, ok := segByTag[tag]; ok && len(seg) > 0 {
			ndc = piecewiseLinearMap(ndc, seg)
		}
		norm[tag] = pinResult{user: c, ndc: floatToF2Dot14Float(ndc)}
	}
	ndcPins := map[string]float64{}
	for tag, p := range norm {
		ndcPins[tag] = p.ndc
	}
	axisTags := make([]string, len(s.outFvar.axes))
	for i, a := range s.outFvar.axes {
		axisTags[i] = a.tag
	}
	fullPin := len(norm) == len(s.outFvar.axes)

	// gvar (+hmtx) instancing per glyph, in component-depth order so that
	// composite bounds resolve against already-instanced components
	// (mirrors fontTools instantiateGvar).
	if s.outGvar != nil {
		byOld := map[uint16]*glyph{}
		for newGID, g := range s.outGlyphs {
			byOld[s.keptOrder[newGID]] = g
		}
		depth := map[uint16]int{}
		var getDepth func(gid uint16, seen map[uint16]bool) int
		getDepth = func(gid uint16, seen map[uint16]bool) int {
			if d, ok := depth[gid]; ok {
				return d
			}
			if seen[gid] {
				return 0
			}
			seen[gid] = true
			d := 0
			if g := byOld[gid]; g != nil && g.isComposite {
				for _, c := range g.composites {
					if cd := getDepth(c.gid, seen); cd+1 > d {
						d = cd + 1
					}
				}
			}
			delete(seen, gid)
			depth[gid] = d
			return d
		}
		order := append([]uint16(nil), s.keptOrder...)
		sort.SliceStable(order, func(i, j int) bool {
			return getDepth(order[i], map[uint16]bool{}) < getDepth(order[j], map[uint16]bool{})
		})
		resolve := func(gid uint16) *glyph { return byOld[gid] }
		remap := s.newIndexToMap()
		for _, oldGID := range order {
			tups := s.outGvar[oldGID]
			if len(tups) == 0 {
				continue
			}
			newGID := int(remap[oldGID])
			remaining, err := instanceGlyphTups(s.outGlyphs[newGID], tups, s.outHmtx[newGID], ndcPins, axisTags, resolve)
			if err != nil {
				return fmt.Errorf("glyph %d: %w", oldGID, err)
			}
			s.outGlyphs[newGID] = remaining.glyph
			byOld[oldGID] = remaining.glyph
			s.outHmtx[newGID] = remaining.hmtx
			if len(remaining.tuples) == 0 {
				delete(s.outGvar, oldGID)
			} else {
				s.outGvar[oldGID] = remaining.tuples
			}
		}
	}
	// HVAR/VVAR: drop on full pin, else restrict.
	if s.outHvar != nil {
		if fullPin {
			s.outHvar = nil
		} else {
			restrictVarStore(&s.outHvar.store, axisTags, ndcPins)
		}
	}
	if s.outVvar != nil {
		if fullPin {
			s.outVvar = nil
		} else {
			restrictVarStore(&s.outVvar.store, axisTags, ndcPins)
		}
	}
	// cvar instancing.
	if _, ok := s.font.tables["cvar"]; ok {
		if err := s.instantiateCvar(norm); err != nil {
			return err
		}
	}
	// avar: drop pinned maps; drop table on full pin.
	if s.outAvar != nil {
		pinnedSet := map[string]bool{}
		for tag := range norm {
			pinnedSet[tag] = true
		}
		allFvar := true
		for _, a := range s.outFvar.axes {
			if !pinnedSet[a.tag] {
				allFvar = false
			}
		}
		allSeg := true
		for i := range s.outAvar.segments {
			if i >= len(s.outFvar.axes) || !pinnedSet[s.outFvar.axes[i].tag] {
				allSeg = false
			}
		}
		if allFvar || allSeg {
			s.outAvar = nil
		} else {
			var segs []avarSegments
			for i, a := range s.outFvar.axes {
				if pinnedSet[a.tag] {
					continue
				}
				var seg avarSegments
				if i < len(s.outAvar.segments) {
					seg = s.outAvar.segments[i]
				}
				if !isValidAvarSegmentMap(seg) {
					continue
				}
				segs = append(segs, seg)
			}
			s.outAvar.segments = segs
		}
	}
	// Name pruning IDs before fvar/STAT instancing.
	beforeIDs := variationNameIDs(s.outFvar, s.outStat)
	for _, in := range s.outFvar.instances {
		if in.subfamilyID > 255 {
			beforeIDs[in.subfamilyID] = true
		}
	}
	// fvar instancing.
	{
		pinnedUsers := map[string]float64{}
		for tag, p := range norm {
			pinnedUsers[tag] = p.user
		}
		var axes []fvarAxis
		for _, a := range s.outFvar.axes {
			if _, pinned := norm[a.tag]; !pinned {
				axes = append(axes, a)
			}
		}
		var instances []fvarInstance
		for _, in := range s.outFvar.instances {
			keep := true
			for tag, v := range pinnedUsers {
				if cv, ok := in.coords[tag]; !ok {
					if dv := axisByTag[tag].def; dv != v {
						keep = false
						break
					}
				} else if cv != v {
					keep = false
					break
				}
			}
			if !keep {
				continue
			}
			coords := map[string]float64{}
			for tag, v := range in.coords {
				if _, pinned := norm[tag]; !pinned {
					coords[tag] = v
				}
			}
			instances = append(instances, fvarInstance{subfamilyID: in.subfamilyID, flags: in.flags, coords: coords})
		}
		s.outFvar.axes = axes
		s.outFvar.instances = instances
	}
	// STAT pruning.
	if s.outStat != nil {

		limits := map[string][2]float64{}
		for tag, p := range norm {
			limits[tag] = [2]float64{p.user, p.user}
		}
		// Axis indexes refer to design axes, which are in fvar order here
		// only if STAT design axes match fvar order; map by tag instead.
		_ = limits
		s.outStat.values = filterSTATValuesByTag(s.outStat, s.statAxesTags(), pinnedUsers(norm))
	}
	// Drop fully-instanced tables.
	if fullPin {
		s.outFvar = nil
		s.outGvar = nil
		s.outAvar = nil
	}
	// Name pruning.
	afterIDs := variationNameIDs(s.outFvar, s.outStat)
	if s.outFvar != nil {
		for _, in := range s.outFvar.instances {
			if in.subfamilyID > 255 {
				afterIDs[in.subfamilyID] = true
			}
		}
	}
	exclude := map[uint16]bool{}
	for id := range beforeIDs {
		if !afterIDs[id] {
			exclude[id] = true
		}
	}
	s.outNames = pruneNames(s.outNames, exclude)
	// OS/2 updates.
	if len(s.os2) >= 58 {
		avg := recalcAvgCharWidth(s.outHmtx)
		putU16(s.os2, 2, uint16(avg))
		// NOTE: os2 here is the working copy? No - s.os2 is the parsed copy;
		// the write phase copies from it. Update in place.
	}
	updateOS2ForPins(s.os2, s.post, pinnedUsers(norm))
	// Overlap flags on static instances.
	if fullPin {
		for _, g := range s.outGlyphs {
			if g.isComposite {
				if len(g.composites) > 0 {
					g.composites[0].flags |= compOverlapCompound
				}
			} else if g.nContours > 0 && len(g.flags) > 0 {
				g.flags[0] |= flagOverlapSimple
			}
		}
	}
	return nil
}

// pinResult is one normalized pin: the clamped user-space value and the
// F2DOT14-quantized NDC value (avar applied).
type pinResult struct {
	user float64
	ndc  float64
}

func pinnedUsers(norm map[string]pinResult) map[string]float64 {
	out := map[string]float64{}
	for tag, p := range norm {
		out[tag] = p.user
	}
	return out
}

func (s *subsetter) statAxesTags() []string {
	if s.outStat == nil {
		return nil
	}
	tags := make([]string, len(s.outStat.axes))
	for i, a := range s.outStat.axes {
		tags[i] = a.tag
	}
	return tags
}

// filterSTATValuesByTag prunes axis values outside pinned limits, mapping
// axis indexes through design-axis tags.
func filterSTATValuesByTag(t *statTable, designTags []string, pins map[string]float64) []statAxisValue {
	tagOf := func(idx int) string {
		if idx >= 0 && idx < len(designTags) {
			return designTags[idx]
		}
		return ""
	}
	var kept []statAxisValue
	for _, v := range t.values {
		switch v.format {
		case 1, 2, 3:
			val := v.value
			if v.format == 2 {
				val = v.nominal
			}
			if p, ok := pins[tagOf(v.axisIndex)]; ok && (val < p || val > p) {
				continue
			}
			kept = append(kept, v)
		case 4:
			drop := false
			for _, r := range v.records {
				if p, ok := pins[tagOf(int(r[0]))]; ok && (r[1] < p || r[1] > p) {
					drop = true
					break
				}
			}
			if !drop {
				kept = append(kept, v)
			}
		}
	}
	return kept
}

// instanceResult is the outcome of pinning one glyph's tuples.
type instanceResult struct {
	glyph  *glyph
	hmtx   [2]int
	tuples []gvarTuple
}

// instanceGlyphTups mirrors _instantiateGvarGlyph + instantiateTupleVariationStore.
func instanceGlyphTups(g *glyph, tuples []gvarTuple, hmtx [2]int, pins map[string]float64, axisTags []string, resolve func(uint16) *glyph) (*instanceResult, error) {
	// Base coordinates + phantom points (mirrors _getCoordinatesAndControls).
	base := make([][2]float64, 0, len(g.coords)+4)
	base = append(base, g.coords...)
	xMin := g.xMin
	adv := hmtx[0]
	lsb := hmtx[1]
	leftX := float64(xMin - lsb)
	rightX := leftX + float64(adv)
	base = append(base,
		[2]float64{leftX, 0}, [2]float64{rightX, 0},
		[2]float64{0, 0}, [2]float64{0, 0})
	orig := append([][2]float64(nil), base...)
	var ends []int
	if g.isComposite {
		for i := range g.composites {
			ends = append(ends, i)
		}
	} else {
		for _, e := range g.endPts {
			ends = append(ends, int(e))
		}
	}
	// Scale/drop per pinned axis (sorted order like the original).
	var tags []string
	for tag := range pins {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	working := make([]gvarTuple, len(tuples))
	copy(working, tuples)
	for _, tag := range tags {
		var next []gvarTuple
		for i := range working {
			if applyPin(&working[i], tag, pins[tag]) {
				next = append(next, working[i])
			}
		}
		working = next
	}
	// Inferred deltas (IUP), mirroring calcInferredDeltas.
	for i := range working {
		have := make([]bool, len(base))
		deltas := make([][2]float64, len(base))
		for j, d := range working[i].deltas {
			if j >= len(base) {
				break
			}
			if d.has {
				have[j] = true
				deltas[j] = [2]float64{d.x, d.y}
			}
		}
		filled := iupDelta(deltas, have, orig, ends)
		for j := range working[i].deltas {
			if j < len(filled) {
				working[i].deltas[j] = gvarDelta{true, filled[j][0], filled[j][1]}
			}
		}
	}
	// Merge tuples with identical support.
	merged := map[string]*gvarTuple{}
	var morder []string
	for i := range working {
		key := supportKey(&working[i])
		if m, ok := merged[key]; ok {
			for j := range m.deltas {
				m.deltas[j].x += working[i].deltas[j].x
				m.deltas[j].y += working[i].deltas[j].y
			}
			continue
		}
		cp := working[i]
		merged[key] = &cp
		morder = append(morder, key)
	}
	// Split off the unconditional (empty-support) tuple as default deltas.
	var defaultDeltas [][2]float64
	var survivors []gvarTuple
	for _, key := range morder {
		t := merged[key]
		if len(t.support) == 0 {
			defaultDeltas = make([][2]float64, len(t.deltas))
			for j, d := range t.deltas {
				defaultDeltas[j] = [2]float64{d.x, d.y}
			}
			continue
		}
		// roundDeltas.
		for j := range t.deltas {
			t.deltas[j].x = otRound(t.deltas[j].x)
			t.deltas[j].y = otRound(t.deltas[j].y)
		}
		survivors = append(survivors, *t)
	}
	// Bake into base (mirrors coordinates += defaultDeltas; Nones are zero).
	ng := *g
	ng.coords = append([][2]float64(nil), g.coords...)
	if len(defaultDeltas) > 0 {
		for j := range base {
			if j < len(defaultDeltas) {
				base[j][0] += defaultDeltas[j][0]
				base[j][1] += defaultDeltas[j][1]
			}
		}
	}
	// _setCoordinates equivalent.
	nPts := len(g.coords)
	newCoords := append([][2]float64(nil), base[:nPts]...)
	pp := base[nPts:]
	ng.coords = newCoords
	ng.toIntCoords()
	ng.recalcBounds(resolve)
	advOut := otRoundInt(pp[1][0] - pp[0][0])
	if advOut < 0 {
		advOut = 0
	}
	lsbOut := otRoundInt(float64(ng.xMin) - pp[0][0])
	hm := [2]int{advOut, lsbOut}
	if g.isComposite {
		// Refresh XY offsets from instanced coords.
		for i := range ng.composites {
			if ng.composites[i].xy && i < len(newCoords) {
				if ng.composites[i].flags&compArgsWords != 0 {
					ng.composites[i].raw = append([]byte(nil), ng.composites[i].raw...)
					putU16(ng.composites[i].raw, 0, uint16(int16(otRoundInt(newCoords[i][0]))))
					putU16(ng.composites[i].raw, 2, uint16(int16(otRoundInt(newCoords[i][1]))))
				} else if len(ng.composites[i].raw) >= 2 {
					ng.composites[i].raw = append([]byte(nil), ng.composites[i].raw...)
					ng.composites[i].raw[0] = byte(int8(otRoundInt(newCoords[i][0])))
					ng.composites[i].raw[1] = byte(int8(otRoundInt(newCoords[i][1])))
				}
			}
		}
	}
	return &instanceResult{glyph: &ng, hmtx: hm, tuples: survivors}, nil
}

// instantiateCvar bakes pinned cvar deltas into the CVT table.
func (s *subsetter) instantiateCvar(norm map[string]pinResult) error {
	raw, ok := s.font.tables["cvar"]
	if !ok {
		return nil
	}
	// cvar header: major, minor, count+flags, dataOffset, headers...
	if len(raw) < 8 {
		return fmt.Errorf("cvar too short")
	}
	axisTags := make([]string, len(s.outFvar.axes))
	for i, a := range s.outFvar.axes {
		axisTags[i] = a.tag
	}
	// Reuse the gvar parser with width-1 tuples: wrap CVT count as points.
	// Parse manually here (single deltas, no inference).
	count := int(getU16(raw, 4))
	hasShared := count&tupSharedPoints != 0
	n := count & tupCountMask
	dataPos := int(getU16(raw, 6))
	pos := 8
	var shared []int
	if hasShared {
		var err error
		shared, dataPos, err = decompilePoints(raw, dataPos, s.cvarN)
		if err != nil {
			return err
		}
	}
	pins := map[string]float64{}
	for tag, p := range norm {
		pins[tag] = p.ndc
	}
	var tags []string
	for tag := range pins {
		tags = append(tags, tag)
	}
	sort.Strings(tags)
	deltas := make([]float64, s.cvarN)
	for i := 0; i < n; i++ {
		if pos+4 > len(raw) {
			return fmt.Errorf("cvar tuple %d out of range", i)
		}
		dataSize := int(getU16(raw, pos))
		flags := int(getU16(raw, pos+2))
		pos += 4
		var peak []float64
		if flags&tupEmbeddedPeak != 0 {
			if pos+2*len(axisTags) > len(raw) {
				return fmt.Errorf("cvar peak out of range")
			}
			for range axisTags {
				peak = append(peak, f2dot14ToFloat(getI16(raw, pos)))
				pos += 2
			}
		} else {
			// cvar has no shared tuples; embedded only.
			return fmt.Errorf("cvar without embedded peak")
		}
		support := map[string][3]float64{}
		for a, tag := range axisTags {
			lo, pk := minFloat(peak[a], 0), peak[a]
			hi := maxFloat(peak[a], 0)
			if lo == 0 && pk == 0 && hi == 0 {
				continue
			}
			support[tag] = [3]float64{lo, pk, hi}
		}
		if dataPos+dataSize > len(raw) {
			return fmt.Errorf("cvar tuple %d data out of range", i)
		}
		tupData := raw[dataPos : dataPos+dataSize]
		dataPos += dataSize
		tpos := 0
		var pts []int
		if flags&tupPrivatePoints != 0 {
			var err error
			pts, tpos, err = decompilePoints(tupData, tpos, s.cvarN)
			if err != nil {
				return err
			}
		} else {
			pts = shared
		}
		d, _, err := decompileDeltas(tupData, tpos, len(pts))
		if err != nil {
			return err
		}
		scalar := 1.0
		axesLeft := map[string][3]float64{}
		for tag2, sup := range support {
			axesLeft[tag2] = sup
		}
		for _, tag := range tags {
			sup, ok := support[tag]
			if !ok {
				continue
			}
			if sup[1] == 0 {
				delete(axesLeft, tag)
				continue
			}
			if !wellFormedTent(sup[0], sup[1], sup[2]) {
				axesLeft = nil
				break
			}
			sc := supportScalar(pins[tag], sup[0], sup[1], sup[2])
			if sc == 0 {
				axesLeft = nil
				break
			}
			scalar *= sc
			delete(axesLeft, tag)
		}
		if axesLeft == nil {
			continue
		}
		if len(axesLeft) == 0 {
			// Fully pinned: bake.
			for k, p := range pts {
				if p >= 0 && p < s.cvarN {
					deltas[p] += d[k] * scalar
				}
			}
		} else {
			return fmt.Errorf("cvar with surviving axes not supported")
		}
	}
	for i := range s.outCvt {
		s.outCvt[i] = int16(otRoundInt(float64(s.outCvt[i]) + deltas[i]))
	}
	// Drop cvar (all tuples consumed by pins; survivors unsupported).
	delete(s.font.tables, "cvar")
	return nil
}
