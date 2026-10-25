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

// varRegion is one variation region: per-axis (start, peak, end) in NDC.
type varRegion struct {
	axes [][3]float64
}

// varData is one VarData subtable: rows of per-region deltas.
type varData struct {
	regionIndexes []uint16
	rows          [][]int
}

// varStore is a parsed ItemVariationStore (HVAR/VVAR style).
type varStore struct {
	regions  []varRegion
	varDatas []varData
}

// hvarTable holds the parsed HVAR (or VVAR) table: offsets are rebuilt on
// encode, so only semantic content is kept.
type hvarTable struct {
	advMap []uint32 // nil = direct glyphID mapping; else packed varidx per glyph
	lsbMap []uint32
	rsbMap []uint32
	store  varStore
}

const noVariationIndex = 0xFFFFFFFF

func parseHvar(data []byte) (*hvarTable, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("HVAR too short")
	}
	// version := getU32(data, 0)
	t := &hvarTable{}
	var err error
	if t.store, err = parseVarStore(data, int(getU32(data, 4))); err != nil {
		return nil, fmt.Errorf("HVAR VarStore: %w", err)
	}
	for i, off := range []uint32{getU32(data, 8), getU32(data, 12), getU32(data, 16)} {
		if off == 0 {
			continue
		}
		m, err := parseDeltaSetIndexMap(data, int(off))
		if err != nil {
			return nil, fmt.Errorf("HVAR map %d: %w", i, err)
		}
		switch i {
		case 0:
			t.advMap = m
		case 1:
			t.lsbMap = m
		case 2:
			t.rsbMap = m
		}
	}
	return t, nil
}

func parseVarStore(data []byte, off int) (varStore, error) {
	var vs varStore
	if off+8 > len(data) {
		return vs, fmt.Errorf("VarStore header out of range")
	}
	if getU16(data, off) != 1 {
		return vs, fmt.Errorf("unsupported VarStore format %d", getU16(data, off))
	}
	regionOff := off + int(getU32(data, off+2))
	varDataCount := int(getU16(data, off+6))
	varDataOffs := make([]int, varDataCount)
	for i := range varDataOffs {
		if off+8+4*i+4 > len(data) {
			return vs, fmt.Errorf("VarStore data offsets out of range")
		}
		varDataOffs[i] = off + int(getU32(data, off+8+4*i))
	}
	regions, err := parseVarRegionList(data, regionOff)
	if err != nil {
		return vs, err
	}
	vs.regions = regions
	for _, vo := range varDataOffs {
		vd, err := parseVarData(data, vo)
		if err != nil {
			return vs, err
		}
		vs.varDatas = append(vs.varDatas, vd)
	}
	return vs, nil
}

func parseVarRegionList(data []byte, off int) ([]varRegion, error) {
	if off+4 > len(data) {
		return nil, fmt.Errorf("VarRegionList out of range")
	}
	axisCount := int(getU16(data, off))
	regionCount := int(getU16(data, off+2))
	var out []varRegion
	for i := 0; i < regionCount; i++ {
		base := off + 4 + i*axisCount*6
		if base+axisCount*6 > len(data) {
			return nil, fmt.Errorf("VarRegion %d out of range", i)
		}
		r := varRegion{}
		for a := 0; a < axisCount; a++ {
			r.axes = append(r.axes, [3]float64{
				f2dot14ToFloat(getI16(data, base+6*a)),
				f2dot14ToFloat(getI16(data, base+6*a+2)),
				f2dot14ToFloat(getI16(data, base+6*a+4)),
			})
		}
		out = append(out, r)
	}
	return out, nil
}

func parseVarData(data []byte, off int) (varData, error) {
	var vd varData
	if off+6 > len(data) {
		return vd, fmt.Errorf("VarData out of range")
	}
	itemCount := int(getU16(data, off))
	numShorts := int(getU16(data, off+2))
	regionCount := int(getU16(data, off+4))
	if off+6+2*regionCount > len(data) {
		return vd, fmt.Errorf("VarData region indexes out of range")
	}
	for i := 0; i < regionCount; i++ {
		vd.regionIndexes = append(vd.regionIndexes, getU16(data, off+6+2*i))
	}
	pos := off + 6 + 2*regionCount
	for i := 0; i < itemCount; i++ {
		row := make([]int, regionCount)
		for j := 0; j < regionCount; j++ {
			if j < numShorts {
				if pos+2 > len(data) {
					return vd, fmt.Errorf("VarData row out of range")
				}
				row[j] = int(getI16(data, pos))
				pos += 2
			} else {
				if pos+1 > len(data) {
					return vd, fmt.Errorf("VarData row out of range")
				}
				row[j] = int(int8(data[pos]))
				pos++
			}
		}
		vd.rows = append(vd.rows, row)
	}
	return vd, nil
}

// parseDeltaSetIndexMap parses format 0 or 1 maps into per-glyph varidx
// values (outer<<16|inner).
func parseDeltaSetIndexMap(data []byte, off int) ([]uint32, error) {
	if off+2 > len(data) {
		return nil, fmt.Errorf("DeltaSetIndexMap out of range")
	}
	format := data[off]
	entryFormat := data[off+1]
	var mapCount int
	hdr := 2
	if format == 0 {
		if off+4 > len(data) {
			return nil, fmt.Errorf("DeltaSetIndexMap header out of range")
		}
		mapCount = int(getU16(data, off+2))
		hdr = 4
	} else if format == 1 {
		if off+6 > len(data) {
			return nil, fmt.Errorf("DeltaSetIndexMap header out of range")
		}
		mapCount = int(getU32(data, off+2))
		hdr = 6
	} else {
		return nil, fmt.Errorf("unsupported DeltaSetIndexMap format %d", format)
	}
	innerBits := int(entryFormat&0x0F) + 1
	entrySize := int((entryFormat&0x30)>>4) + 1
	if off+hdr+entrySize*mapCount > len(data) {
		return nil, fmt.Errorf("DeltaSetIndexMap entries out of range")
	}
	out := make([]uint32, mapCount)
	for i := range out {
		var entry uint32
		for b := 0; b < entrySize; b++ {
			entry = entry<<8 | uint32(data[off+hdr+i*entrySize+b])
		}
		outer := entry >> uint(innerBits)
		inner := entry & ((1 << uint(innerBits)) - 1)
		out[i] = outer<<16 | inner
	}
	return out, nil
}

// encodeDeltaSetIndexMap writes a format 0 map with minimal entries.
func encodeDeltaSetIndexMap(m []uint32) []byte {
	// Determine inner bits and entry size.
	var maxInner, maxOuter uint32
	for _, v := range m {
		if v>>16 > maxOuter {
			maxOuter = v >> 16
		}
		if v&0xFFFF > maxInner {
			maxInner = v & 0xFFFF
		}
	}
	innerBits := 1
	for (uint32(1) << uint(innerBits)) <= maxInner {
		innerBits++
	}
	entrySize := 1
	for (uint32(1)<<uint(8*entrySize-innerBits)) <= maxOuter && entrySize < 4 {
		entrySize++
	}
	entryFormat := byte((entrySize-1)<<4) | byte(innerBits-1)
	out := []byte{0, entryFormat}
	out = appendU16(out, uint16(len(m)))
	for _, v := range m {
		entry := (v>>16)<<uint(innerBits) | (v & ((1 << uint(innerBits)) - 1))
		for b := entrySize - 1; b >= 0; b-- {
			out = append(out, byte(entry>>(8*uint(b))))
		}
	}
	return out
}

// subsetVarStore filters VarData rows to kept glyphs. used maps original GID
// to new row position (for direct mapping). Maps are handled by the caller.
func subsetVarStoreRows(vs *varStore, keep []uint16) {
	for i := range vs.varDatas {
		rows := make([][]int, 0, len(keep))
		for _, gid := range keep {
			if int(gid) < len(vs.varDatas[i].rows) {
				rows = append(rows, vs.varDatas[i].rows[gid])
			} else {
				rows = append(rows, make([]int, len(vs.varDatas[i].regionIndexes)))
			}
		}
		vs.varDatas[i].rows = rows
	}
}

// restrictVarStore applies pins to the VarStore in place, mirroring
// instantiateItemVariationStore for the HVAR case: regions on pinned axes
// are evaluated and removed; surviving regions on remaining axes keep their
// deltas (scaled by the pinned-axis scalar). axisTags is the full fvar axis
// order.
func restrictVarStore(vs *varStore, axisTags []string, pins map[string]float64) {
	// Restrict each region: scale by pinned axes, keep remaining triples.
	type restricted struct {
		axes   [][3]float64
		scalar float64
		keep   bool
	}
	regions := make([]restricted, len(vs.regions))
	for i, r := range vs.regions {
		rg := restricted{scalar: 1, keep: true}
		for a, triple := range r.axes {
			tag := ""
			if a < len(axisTags) {
				tag = axisTags[a]
			}
			if ndc, pinned := pins[tag]; pinned {
				// Mirror changeTupleVariationAxisLimit: peak-0 axes drop
				// out, malformed regions are dropped, others scale.
				if triple[1] == 0 {
					continue
				}
				if triple[0] > triple[1] || triple[1] > triple[2] ||
					(triple[0] < 0 && triple[2] > 0) {
					rg.keep = false
					break
				}
				rg.scalar *= supportScalar(ndc, triple[0], triple[1], triple[2])
			} else {
				rg.axes = append(rg.axes, triple)
			}
		}
		if rg.scalar == 0 {
			rg.keep = false
		}
		regions[i] = rg
	}
	keyOf := func(axes [][3]float64) string {
		s := ""
		for _, a := range axes {
			s += fmt.Sprintf("%d,%d,%d;", floatToF2Dot14(a[0]), floatToF2Dot14(a[1]), floatToF2Dot14(a[2]))
		}
		return s
	}
	// Regions left with no participating axis (all peaks zero) are dropped:
	// their pinned contribution folds into default deltas, which glyf fonts
	// ignore (matching fontTools, which drops empty-support tuples).
	for i := range regions {
		if !regions[i].keep {
			continue
		}
		participates := false
		for _, a := range regions[i].axes {
			if a[1] != 0 {
				participates = true
				break
			}
		}
		if !participates {
			regions[i].keep = false
		}
	}
	// New region list in first-appearance order, deduplicated.
	newRegions := []varRegion{}
	indexOf := map[string]int{}
	for _, rg := range regions {
		if !rg.keep {
			continue
		}
		key := keyOf(rg.axes)
		if _, ok := indexOf[key]; !ok {
			indexOf[key] = len(newRegions)
			newRegions = append(newRegions, varRegion{axes: rg.axes})
		}
	}
	// Rebuild each VarData: remap region indexes, fold scaled deltas.
	for di := range vs.varDatas {
		vd := &vs.varDatas[di]
		var newIndexes []uint16
		posMap := make([]int, len(vd.regionIndexes))
		for i := range posMap {
			posMap[i] = -1
		}
		for pos, ri := range vd.regionIndexes {
			if int(ri) >= len(regions) || !regions[ri].keep {
				continue
			}
			ni := indexOf[keyOf(regions[ri].axes)]
			found := -1
			for k, v := range newIndexes {
				if int(v) == ni {
					found = k
					break
				}
			}
			if found < 0 {
				found = len(newIndexes)
				newIndexes = append(newIndexes, uint16(ni))
			}
			posMap[pos] = found
		}
		newRows := make([][]int, len(vd.rows))
		for r, row := range vd.rows {
			newRow := make([]int, len(newIndexes))
			for oldPos, delta := range row {
				if oldPos >= len(posMap) || posMap[oldPos] < 0 {
					continue
				}
				sc := 1.0
				if ri := vd.regionIndexes[oldPos]; int(ri) < len(regions) {
					sc = regions[ri].scalar
				}
				newRow[posMap[oldPos]] += otRoundInt(float64(delta) * sc)
			}
			newRows[r] = newRow
		}
		vd.regionIndexes = newIndexes
		vd.rows = newRows
	}
	vs.regions = newRegions
}

// encodeHvar serializes an HVAR table with rebuilt offsets. axisCount is the
// number of remaining fvar axes.
//
// Known deltas vs fontTools: VarData rows always use int16 deltas
// (NumShorts == region count) instead of mixing int8/int16, and the store is
// never re-optimized. Rendering is identical; the table may be marginally
// larger.
func encodeHvar(t *hvarTable, axisCount int) []byte {
	varStoreBytes, regionCount := encodeVarStore(&t.store, axisCount)
	out := make([]byte, 0, 64)
	out = appendU32(out, 0x00010000)
	out = appendU32(out, 20) // varStore offset
	advPos, lsbPos, rsbPos := len(out), len(out)+4, len(out)+8
	out = appendU32(out, 0)
	out = appendU32(out, 0)
	out = appendU32(out, 0)
	var adv, lsb, rsb []byte
	if t.advMap != nil {
		adv = encodeDeltaSetIndexMap(t.advMap)
	}
	if t.lsbMap != nil {
		lsb = encodeDeltaSetIndexMap(t.lsbMap)
	}
	if t.rsbMap != nil {
		rsb = encodeDeltaSetIndexMap(t.rsbMap)
	}
	_ = regionCount
	out = append(out, varStoreBytes...)
	if adv != nil {
		putU32(out, advPos, uint32(len(out)))
		out = append(out, adv...)
	}
	if lsb != nil {
		putU32(out, lsbPos, uint32(len(out)))
		out = append(out, lsb...)
	}
	if rsb != nil {
		putU32(out, rsbPos, uint32(len(out)))
		out = append(out, rsb...)
	}
	return out
}

func encodeVarStore(vs *varStore, axisCount int) ([]byte, int) {
	// VarRegionList.
	regions := []byte{}
	regions = appendU16(regions, uint16(axisCount))
	regions = appendU16(regions, uint16(len(vs.regions)))
	for _, r := range vs.regions {
		for a := 0; a < axisCount; a++ {
			var triple [3]float64
			if a < len(r.axes) {
				triple = r.axes[a]
			}
			regions = appendI16(regions, floatToF2Dot14(triple[0]))
			regions = appendI16(regions, floatToF2Dot14(triple[1]))
			regions = appendI16(regions, floatToF2Dot14(triple[2]))
		}
	}
	out := []byte{0, 1} // format
	regionListOff := 2 + 4 + 2 + 4*len(vs.varDatas)
	out = appendU32(out, uint32(regionListOff))
	out = appendU16(out, uint16(len(vs.varDatas)))
	dataBase := regionListOff + len(regions)
	for _, vd := range vs.varDatas {
		out = appendU32(out, uint32(dataBase))
		dataBase += varDataSize(&vd)
	}
	out = append(out, regions...)
	for i := range vs.varDatas {
		out = append(out, encodeVarData(&vs.varDatas[i])...)
	}
	return out, len(vs.regions)
}

func varDataSize(vd *varData) int {
	return 6 + 2*len(vd.regionIndexes) + len(vd.rows)*2*len(vd.regionIndexes)
}

func encodeVarData(vd *varData) []byte {
	out := appendU16(nil, uint16(len(vd.rows)))
	// NumShorts = regionCount: all deltas stored as int16 (always valid).
	out = appendU16(out, uint16(len(vd.regionIndexes)))
	out = appendU16(out, uint16(len(vd.regionIndexes)))
	for _, ri := range vd.regionIndexes {
		out = appendU16(out, ri)
	}
	for _, row := range vd.rows {
		for j := range vd.regionIndexes {
			v := 0
			if j < len(row) {
				v = row[j]
			}
			out = appendI16(out, int16(v))
		}
	}
	return out
}

// remapDeltaSetIndexMap remaps map entries through old->new varidx.
func remapDeltaSetIndexMap(m []uint32, remap map[uint32]uint32) []uint32 {
	out := make([]uint32, len(m))
	for i, v := range m {
		if nv, ok := remap[v]; ok {
			out[i] = nv
		} else {
			out[i] = v
		}
	}
	return out
}
