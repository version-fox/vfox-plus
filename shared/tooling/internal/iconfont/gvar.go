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

// Tuple variation flags, mirroring fontTools TupleVariation constants.
const (
	tupEmbeddedPeak  = 0x8000
	tupIntermediate  = 0x4000
	tupPrivatePoints = 0x2000
	tupSharedPoints  = 0x8000
	tupCountMask     = 0x0FFF
	tupIndexMask     = 0x0FFF
	delZeros         = 0x80
	delWords         = 0x40
	delLongs         = 0xC0
	delSizeMask      = 0xC0
	delRunMask       = 0x3F
	ptsWords         = 0x80
	ptsRunMask       = 0x7F
)

// gvarDelta is one point's delta; has==false marks an uninferred point.
type gvarDelta struct {
	has  bool
	x, y float64
}

// gvarTuple is one parsed tuple variation with per-point deltas.
type gvarTuple struct {
	support map[string][3]float64 // axis tag -> (lower, peak, upper)
	deltas  []gvarDelta           // parallel to glyph points (+4 phantom)
}

// usedPoints returns the explicit point set like TupleVariation.getUsedPoints:
// nil means nothing to encode (skip tuple); empty-non-nil means all points.
func (t *gvarTuple) usedPoints() ([]int, bool) {
	none := true
	for _, d := range t.deltas {
		if d.has {
			none = false
			break
		}
	}
	if none {
		return nil, false
	}
	all := true
	var pts []int
	for i, d := range t.deltas {
		if d.has {
			pts = append(pts, i)
		} else {
			all = false
		}
	}
	if all {
		return []int{}, true
	}
	return pts, true
}

type gvarTable struct {
	axes       []string // fvar axis tags
	shared     [][]float64
	variations map[uint16][]gvarTuple // old GID -> tuples (file order)
	glyphIDs   []uint16               // glyphs with variations, file order
}

// pointCountFor returns outline points + 4 phantom points for a glyph.
func pointCountFor(g *glyph) int {
	if g.isComposite {
		return len(g.composites) + 4
	}
	return len(g.coords) + 4
}

func parseGvar(data []byte, axisTags []string, points map[uint16]int) (*gvarTable, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("gvar too short")
	}
	axisCount := int(getU16(data, 4))
	sharedCount := int(getU16(data, 6))
	sharedOff := int(getU32(data, 8))
	glyphCount := int(getU16(data, 12))
	flags := getU16(data, 14)
	dataOff := int(getU32(data, 16))
	longOffsets := flags&1 != 0
	offSize := 2
	if longOffsets {
		offSize = 4
	}
	if len(data) < 20+(glyphCount+1)*offSize {
		return nil, fmt.Errorf("gvar offsets out of range")
	}
	getOff := func(i int) int {
		if longOffsets {
			return int(getU32(data, 20+4*i))
		}
		return int(getU16(data, 20+2*i)) * 2
	}
	t := &gvarTable{axes: axisTags, variations: map[uint16][]gvarTuple{}}
	for i := 0; i < sharedCount; i++ {
		off := sharedOff + i*axisCount*2
		if off+axisCount*2 > len(data) {
			return nil, fmt.Errorf("gvar shared tuples out of range")
		}
		var tup []float64
		for a := 0; a < axisCount; a++ {
			tup = append(tup, f2dot14ToFloat(getI16(data, off+2*a)))
		}
		t.shared = append(t.shared, tup)
	}
	for gid := 0; gid < glyphCount; gid++ {
		start, end := getOff(gid), getOff(gid+1)
		if end < start || dataOff+end > len(data) {
			return nil, fmt.Errorf("gvar glyph %d data out of range", gid)
		}
		if end == start {
			continue
		}
		nPts, ok := points[uint16(gid)]
		if !ok {
			return nil, fmt.Errorf("gvar glyph %d missing point count", gid)
		}
		tups, err := parseGlyphVariations(data[dataOff+start:dataOff+end], axisTags, t.shared, nPts)
		if err != nil {
			return nil, fmt.Errorf("gvar glyph %d: %w", gid, err)
		}
		if len(tups) > 0 {
			t.variations[uint16(gid)] = tups
			t.glyphIDs = append(t.glyphIDs, uint16(gid))
		}
	}
	return t, nil
}

func parseGlyphVariations(data []byte, axisTags []string, shared [][]float64, pointCount int) ([]gvarTuple, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("glyph variation data too short")
	}
	count := int(getU16(data, 0))
	hasShared := count&tupSharedPoints != 0
	n := count & tupCountMask
	dataPos := int(getU16(data, 2))
	if dataPos > len(data) {
		return nil, fmt.Errorf("glyph variation data offset out of range")
	}
	pos := 4
	var sharedPoints []int
	if hasShared {
		var err error
		sharedPoints, dataPos, err = decompilePoints(data, dataPos, pointCount)
		if err != nil {
			return nil, err
		}
	}
	var out []gvarTuple
	for i := 0; i < n; i++ {
		if pos+4 > len(data) {
			return nil, fmt.Errorf("tuple header %d out of range", i)
		}
		dataSize := int(getU16(data, pos))
		flags := int(getU16(data, pos+2))
		pos += 4
		var peak []float64
		if flags&tupEmbeddedPeak != 0 {
			if pos+2*len(axisTags) > len(data) {
				return nil, fmt.Errorf("embedded peak out of range")
			}
			for _, a := range axisTags {
				_ = a
				peak = append(peak, f2dot14ToFloat(getI16(data, pos)))
				pos += 2
			}
		} else {
			idx := flags & tupIndexMask
			if idx >= len(shared) {
				return nil, fmt.Errorf("shared tuple index %d out of range", idx)
			}
			peak = append([]float64(nil), shared[idx]...)
		}
		var start, end []float64
		if flags&tupIntermediate != 0 {
			if pos+4*len(axisTags) > len(data) {
				return nil, fmt.Errorf("intermediate tuples out of range")
			}
			for range axisTags {
				start = append(start, f2dot14ToFloat(getI16(data, pos)))
				pos += 2
			}
			for range axisTags {
				end = append(end, f2dot14ToFloat(getI16(data, pos)))
				pos += 2
			}
		} else {
			for _, p := range peak {
				start = append(start, minFloat(p, 0))
				end = append(end, maxFloat(p, 0))
			}
		}
		if dataPos+dataSize > len(data) {
			return nil, fmt.Errorf("tuple %d data out of range", i)
		}
		tupData := data[dataPos : dataPos+dataSize]
		dataPos += dataSize
		tup := gvarTuple{support: map[string][3]float64{}, deltas: make([]gvarDelta, pointCount)}
		for a, tag := range axisTags {
			lo, pk, hi := start[a], peak[a], end[a]
			if lo == 0 && pk == 0 && hi == 0 {
				continue
			}
			tup.support[tag] = [3]float64{lo, pk, hi}
		}
		tpos := 0
		var pts []int
		if flags&tupPrivatePoints != 0 {
			var err error
			pts, tpos, err = decompilePoints(tupData, tpos, pointCount)
			if err != nil {
				return nil, fmt.Errorf("tuple %d points: %w (data %x)", i, err, tupData)
			}
		} else {
			pts = sharedPoints
		}
		dx, tpos, err := decompileDeltas(tupData, tpos, len(pts))
		if err != nil {
			return nil, fmt.Errorf("tuple %d dx: %w", i, err)
		}
		dy, _, err := decompileDeltas(tupData, tpos, len(pts))
		if err != nil {
			return nil, fmt.Errorf("tuple %d dy: %w", i, err)
		}
		for k, p := range pts {
			if p < 0 || p >= pointCount {
				continue
			}
			tup.deltas[p] = gvarDelta{true, dx[k], dy[k]}
		}
		out = append(out, tup)
	}
	return out, nil
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// decompilePoints unpacks packed point numbers. Returns points, new offset.
func decompilePoints(data []byte, pos, pointCount int) ([]int, int, error) {
	if pos >= len(data) {
		return nil, pos, fmt.Errorf("points out of range")
	}
	first := data[pos]
	pos++
	var count int
	if first == 0 {
		all := make([]int, pointCount)
		for i := range all {
			all[i] = i
		}
		return all, pos, nil
	}
	if first&ptsWords != 0 {
		if pos >= len(data) {
			return nil, pos, fmt.Errorf("point count out of range")
		}
		count = int(first&ptsRunMask)<<8 | int(data[pos])
		pos++
	} else {
		count = int(first)
	}
	var result []int
	for len(result) < count {
		if pos >= len(data) {
			return nil, pos, fmt.Errorf("point run out of range")
		}
		header := data[pos]
		pos++
		n := int(header&ptsRunMask) + 1
		if header&ptsWords != 0 {
			if pos+2*n > len(data) {
				return nil, pos, fmt.Errorf("word points out of range")
			}
			for i := 0; i < n; i++ {
				result = append(result, int(getU16(data, pos)))
				pos += 2
			}
		} else {
			if pos+n > len(data) {
				return nil, pos, fmt.Errorf("byte points out of range")
			}
			for i := 0; i < n; i++ {
				result = append(result, int(data[pos]))
				pos++
			}
		}
	}
	// Relative to absolute.
	abs := make([]int, len(result))
	cur := 0
	for i, d := range result {
		cur += d
		abs[i] = cur
	}
	return abs, pos, nil
}

// decompileDeltas unpacks packed deltas for count logical values.
func decompileDeltas(data []byte, pos, count int) ([]float64, int, error) {
	result := make([]float64, 0, count)
	for len(result) < count {
		if pos >= len(data) {
			return nil, pos, fmt.Errorf("delta run out of range")
		}
		header := data[pos]
		pos++
		n := int(header&delRunMask) + 1
		switch header & delSizeMask {
		case delZeros:
			for i := 0; i < n; i++ {
				result = append(result, 0)
			}
		case delLongs:
			if pos+4*n > len(data) {
				return nil, pos, fmt.Errorf("long deltas out of range")
			}
			for i := 0; i < n; i++ {
				result = append(result, float64(getI32(data, pos)))
				pos += 4
			}
		case delWords:
			if pos+2*n > len(data) {
				return nil, pos, fmt.Errorf("word deltas out of range")
			}
			for i := 0; i < n; i++ {
				result = append(result, float64(getI16(data, pos)))
				pos += 2
			}
		default:
			if pos+n > len(data) {
				return nil, pos, fmt.Errorf("byte deltas out of range")
			}
			for i := 0; i < n; i++ {
				result = append(result, float64(int8(data[pos])))
				pos++
			}
		}
	}
	return result, pos, nil
}

// compilePoints packs a point set like TupleVariation.compilePoints.
func compilePoints(points []int) []byte {
	if len(points) == 0 {
		return []byte{0}
	}
	sorted := append([]int(nil), points...)
	sort.Ints(sorted)
	out := []byte{}
	if len(sorted) < 0x80 {
		out = append(out, byte(len(sorted)))
	} else {
		out = append(out, byte(len(sorted)>>8)|0x80, byte(len(sorted)))
	}
	last, pos := 0, 0
	for pos < len(sorted) {
		hdrPos := len(out)
		out = append(out, 0)
		runLen := 0
		useBytes := -1 // unknown
		for pos < len(sorted) && runLen <= 127 {
			delta := sorted[pos] - last
			if useBytes < 0 {
				if delta >= 0 && delta <= 0xFF {
					useBytes = 1
				} else {
					useBytes = 0
				}
			}
			if useBytes == 1 && (delta > 0xFF || delta < 0) {
				break
			}
			if useBytes == 1 {
				out = append(out, byte(delta))
			} else {
				out = appendU16(out, uint16(delta))
			}
			last = sorted[pos]
			pos++
			runLen++
		}
		if useBytes == 0 {
			out[hdrPos] = byte(runLen-1) | ptsWords
		} else {
			out[hdrPos] = byte(runLen - 1)
		}
	}
	return out
}

// compileDeltaValues packs deltas like TupleVariation.compileDeltaValues_
// with optimizeSize=True.
func compileDeltaValues(deltas []int) []byte {
	var out []byte
	pos, n := 0, len(deltas)
	for pos < n {
		v := deltas[pos]
		switch {
		case v == 0:
			pos = encodeZeroRun(deltas, pos, &out)
		case v >= -128 && v <= 127:
			pos = encodeByteRun(deltas, pos, &out)
		case v >= -32768 && v <= 32767:
			pos = encodeWordRun(deltas, pos, &out)
		default:
			pos = encodeLongRun(deltas, pos, &out)
		}
	}
	return out
}

func encodeZeroRun(deltas []int, off int, out *[]byte) int {
	pos := off
	for pos < len(deltas) && deltas[pos] == 0 {
		pos++
	}
	run := pos - off
	for run >= 64 {
		*out = append(*out, delZeros|63)
		run -= 64
	}
	if run > 0 {
		*out = append(*out, delZeros|byte(run-1))
	}
	return pos
}

func encodeByteRun(deltas []int, off int, out *[]byte) int {
	pos := off
	for pos < len(deltas) {
		v := deltas[pos]
		if v < -128 || v > 127 {
			break
		}
		if v == 0 && pos+1 < len(deltas) && deltas[pos+1] == 0 {
			break
		}
		pos++
	}
	emit := func(chunk []int) {
		*out = append(*out, byte(len(chunk)-1))
		for _, v := range chunk {
			*out = append(*out, byte(int8(v)))
		}
	}
	remaining := deltas[off:pos]
	for len(remaining) >= 64 {
		emit(remaining[:64])
		remaining = remaining[64:]
	}
	if len(remaining) > 0 {
		emit(remaining)
	}
	return pos
}

func encodeWordRun(deltas []int, off int, out *[]byte) int {
	pos := off
	for pos < len(deltas) {
		v := deltas[pos]
		if v == 0 {
			break
		}
		if v >= -128 && v <= 127 && pos+1 < len(deltas) && deltas[pos+1] >= -128 && deltas[pos+1] <= 127 {
			break
		}
		if v < -32768 || v > 32767 {
			break
		}
		pos++
	}
	emit := func(chunk []int) {
		*out = append(*out, delWords|byte(len(chunk)-1))
		for _, v := range chunk {
			*out = appendI16(*out, int16(v))
		}
	}
	remaining := deltas[off:pos]
	for len(remaining) >= 64 {
		emit(remaining[:64])
		remaining = remaining[64:]
	}
	if len(remaining) > 0 {
		emit(remaining)
	}
	return pos
}

func encodeLongRun(deltas []int, off int, out *[]byte) int {
	pos := off
	for pos < len(deltas) {
		v := deltas[pos]
		if v >= -32768 && v <= 32767 {
			break
		}
		pos++
	}
	remaining := deltas[off:pos]
	for len(remaining) >= 64 {
		*out = append(*out, delLongs|63)
		for _, v := range remaining[:64] {
			*out = appendU32(*out, uint32(int32(v)))
		}
		remaining = remaining[64:]
	}
	if len(remaining) > 0 {
		*out = append(*out, delLongs|byte(len(remaining)-1))
		for _, v := range remaining {
			*out = appendU32(*out, uint32(int32(v)))
		}
	}
	return pos
}

// peakCoord returns the peak tuple bytes for a tuple over axisTags.
func peakCoord(t *gvarTuple, axisTags []string) []byte {
	var out []byte
	for _, tag := range axisTags {
		p := 0.0
		if s, ok := t.support[tag]; ok {
			p = s[1]
		}
		out = appendI16(out, floatToF2Dot14(p))
	}
	return out
}

// needsIntermediate reports whether a tuple needs intermediate records,
// mirroring compileIntermediateCoord.
func needsIntermediate(t *gvarTuple) bool {
	for _, s := range t.support {
		minV, v, maxV := s[0], s[1], s[2]
		if minV != minFloat(v, 0.0) || maxV != maxFloat(v, 0.0) {
			return true
		}
	}
	return false
}

// compileTupleData encodes one tuple's header records (peak + intermediate).
func compileTupleData(t *gvarTuple, axisTags []string, sharedIdx map[string]int) (flags int, recs []byte) {
	coord := string(peakCoord(t, axisTags))
	if idx, ok := sharedIdx[coord]; ok {
		flags = idx
	} else {
		flags = tupEmbeddedPeak
		recs = append(recs, peakCoord(t, axisTags)...)
	}
	if needsIntermediate(t) {
		flags |= tupIntermediate
		for _, tag := range axisTags {
			s, ok := t.support[tag]
			if !ok {
				s = [3]float64{0, 0, 0}
			}
			recs = append(recs, appendI16(nil, floatToF2Dot14(s[0]))...)
		}
		for _, tag := range axisTags {
			s, ok := t.support[tag]
			if !ok {
				s = [3]float64{0, 0, 0}
			}
			recs = append(recs, appendI16(nil, floatToF2Dot14(s[2]))...)
		}
	}
	return flags, recs
}

// compileSharedTuples selects shared peak tuples like
// TupleVariation.compileSharedTuples: coords used more than once, sorted by
// (-count, coord bytes).
func compileSharedTuples(tuples []*gvarTuple, axisTags []string) [][]byte {
	count := map[string]int{}
	order := []string{}
	for _, t := range tuples {
		c := string(peakCoord(t, axisTags))
		if _, ok := count[c]; !ok {
			order = append(order, c)
		}
		count[c]++
	}
	type kv struct {
		c string
		n int
	}
	var cands []kv
	for _, c := range order {
		if count[c] > 1 {
			cands = append(cands, kv{c, count[c]})
		}
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].n != cands[j].n {
			return cands[i].n > cands[j].n
		}
		return cands[i].c < cands[j].c
	})
	var out [][]byte
	for _, k := range cands {
		out = append(out, []byte(k.c))
		if len(out) >= tupIndexMask+1 {
			break
		}
	}
	return out
}

// compileGlyphVariations encodes one glyph's tuples with shared points
// selection, mirroring compileTupleVariationStore.
func compileGlyphVariations(tuples []*gvarTuple, axisTags []string, sharedIdx map[string]int) []byte {
	// Filter to encodable tuples and collect point sets.
	type item struct {
		t   *gvarTuple
		pts []int
		key string
	}
	var items []item
	pointCount := map[string]int{} // key -> occurrences
	var keyOrder []string
	compiled := map[string][]byte{}
	for _, t := range tuples {
		pts, ok := t.usedPoints()
		if !ok {
			continue
		}
		var key string
		if len(pts) == 0 {
			key = "all"
		} else {
			// Canonical key for the set.
			cp := append([]int(nil), pts...)
			sort.Ints(cp)
			for _, p := range cp {
				key += fmt.Sprintf("%d,", p)
			}
		}
		if _, seen := pointCount[key]; !seen {
			keyOrder = append(keyOrder, key)
		}
		pointCount[key]++
		items = append(items, item{t, pts, key})
	}
	if len(items) == 0 {
		return nil
	}
	for _, k := range keyOrder {
		var pts []int
		for _, it := range items {
			if it.key == k {
				pts = it.pts
				break
			}
		}
		compiled[k] = compilePoints(pts)
	}
	// Shared points: maximize saved bytes; first maximum wins.
	sharedKey := ""
	best := -1
	for _, k := range keyOrder {
		saved := len(compiled[k]) * (pointCount[k] - 1)
		if saved > best {
			best = saved
			sharedKey = k
		}
	}
	var data []byte
	tupleCount := len(items)
	if sharedKey != "" && best > 0 {
		tupleCount |= tupSharedPoints
		data = append(data, compiled[sharedKey]...)
	}
	var headers, blob []byte
	for _, it := range items {
		var pointData []byte
		if it.key != sharedKey || best <= 0 {
			pointData = compiled[it.key]
		}
		tup, aux := compileOneTuple(it.t, it.pts, pointData, axisTags, sharedIdx)
		headers = append(headers, tup...)
		blob = append(blob, aux...)
	}
	out := appendU16(nil, uint16(tupleCount))
	// dataOffset points at the start of the data block, which begins with
	// the shared points (if any), followed by per-tuple data.
	dataOff := 2 + 2 + len(headers)
	out = appendU16(out, uint16(dataOff))
	out = append(out, headers...)
	out = append(out, data...)
	out = append(out, blob...)
	if len(out)%2 == 1 {
		out = append(out, 0)
	}
	return out
}

// compileOneTuple encodes a single tuple variation.
func compileOneTuple(t *gvarTuple, pts []int, pointData []byte, axisTags []string, sharedIdx map[string]int) (header, aux []byte) {
	flags, recs := compileTupleData(t, axisTags, sharedIdx)
	if len(pointData) > 0 {
		flags |= tupPrivatePoints
		aux = append(aux, pointData...)
	}
	// Deltas in point order; all-points when pts empty.
	var order []int
	if len(pts) == 0 {
		for i := range t.deltas {
			order = append(order, i)
		}
	} else {
		order = pts
	}
	dx := make([]int, len(order))
	dy := make([]int, len(order))
	for i, p := range order {
		dx[i] = otRoundInt(t.deltas[p].x)
		dy[i] = otRoundInt(t.deltas[p].y)
	}
	aux = append(aux, compileDeltaValues(dx)...)
	aux = append(aux, compileDeltaValues(dy)...)
	header = appendU16(nil, uint16(len(aux)))
	header = appendU16(header, uint16(flags))
	header = append(header, recs...)
	return header, aux
}

// encodeGvar rebuilds the gvar table from per-glyph tuples in subset glyph
// order. shared tuples are recomputed; offsets use short format when they fit.
func encodeGvar(axisTags []string, glyphs []uint16, variations map[uint16][]gvarTuple) []byte {
	shared := compileSharedTuples(collectTuples(variations, glyphs), axisTags)
	sharedIdx := map[string]int{}
	for i, s := range shared {
		sharedIdx[string(s)] = i
	}
	compiled := make([][]byte, len(glyphs))
	for i, gid := range glyphs {
		var ts []*gvarTuple
		for j := range variations[gid] {
			t := variations[gid][j]
			ts = append(ts, &t)
		}
		compiled[i] = compileGlyphVariations(ts, axisTags, sharedIdx)
	}
	offsets := make([]int, len(glyphs)+1)
	for i, c := range compiled {
		offsets[i+1] = offsets[i] + len(c)
	}
	long := offsets[len(offsets)-1] > 0xFFFF*2
	var flags uint16
	if long {
		flags = 1
	}
	// Header.
	out := make([]byte, 0, 1024)
	out = appendU16(out, 1) // major
	out = appendU16(out, 0) // minor
	out = appendU16(out, uint16(len(axisTags)))
	out = appendU16(out, uint16(len(shared)))
	sharedOffPos := len(out)
	out = appendU32(out, 0)
	out = appendU16(out, uint16(len(glyphs))) // glyphCount
	out = appendU16(out, flags)
	dataOffPos := len(out)
	out = appendU32(out, 0)
	if !long {
		for _, o := range offsets {
			out = appendU16(out, uint16(o/2))
		}
	} else {
		for _, o := range offsets {
			out = appendU32(out, uint32(o))
		}
	}
	putU32(out, sharedOffPos, uint32(len(out)))
	for _, s := range shared {
		out = append(out, s...)
	}
	putU32(out, dataOffPos, uint32(len(out)))
	for _, c := range compiled {
		out = append(out, c...)
	}
	return out
}

func collectTuples(variations map[uint16][]gvarTuple, glyphs []uint16) []*gvarTuple {
	var all []*gvarTuple
	for _, gid := range glyphs {
		for i := range variations[gid] {
			t := variations[gid][i]
			all = append(all, &t)
		}
	}
	return all
}

// applyPin scales one tuple for a pinned axis, returning false to drop it.
// Mirrors changeTupleVariationAxisLimit for pins.
func applyPin(t *gvarTuple, axis string, ndc float64) bool {
	s, ok := t.support[axis]
	if !ok {
		return true
	}
	lower, peak, upper := s[0], s[1], s[2]
	if peak == 0 {
		delete(t.support, axis)
		return true
	}
	if !wellFormedTent(lower, peak, upper) {
		return false
	}
	scalar := supportScalar(ndc, lower, peak, upper)
	if scalar == 0 {
		return false
	}
	if scalar != 1 {
		for i := range t.deltas {
			if t.deltas[i].has {
				t.deltas[i].x *= scalar
				t.deltas[i].y *= scalar
			}
		}
	}
	delete(t.support, axis)
	return true
}

// supportKey returns a canonical support signature for merging.
func supportKey(t *gvarTuple) string {
	keys := make([]string, 0, len(t.support))
	for k := range t.support {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	key := ""
	for _, k := range keys {
		s := t.support[k]
		key += fmt.Sprintf("%s:%d,%d,%d;", k, floatToF2Dot14(s[0]), floatToF2Dot14(s[1]), floatToF2Dot14(s[2]))
	}
	return key
}
