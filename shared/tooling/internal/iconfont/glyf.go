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

// Glyph flag bits (TrueType simple glyph description).
const (
	flagOnCurve       = 0x01
	flagXShort        = 0x02
	flagYShort        = 0x04
	flagRepeat        = 0x08
	flagXSame         = 0x10
	flagYSame         = 0x20
	flagOverlapSimple = 0x40
	flagCubic         = 0x80
)

// keepFlags mirrors fontTools keepFlags: only on-curve, overlap and cubic
// bits survive a decompile/recompile round-trip.
const keepFlags = flagOnCurve | flagOverlapSimple | flagCubic

// Composite component flag bits.
const (
	compArgsWords       = 0x0001
	compArgsXYValues    = 0x0002
	compRoundXYToGrid   = 0x0004
	compHaveScale       = 0x0008
	compMoreComponents  = 0x0020
	compHaveXYScale     = 0x0040
	compHave2x2         = 0x0080
	compHaveInstr       = 0x0100
	compUseMyMetrics    = 0x0200
	compOverlapCompound = 0x0400
)

// component is one glyph of a composite glyph. args holds the raw argument
// bytes (glyph index excluded): arg1/arg2 plus optional scale data.
type component struct {
	flags uint16
	gid   uint16
	xy    bool // args are x/y offsets (ARG_1_AND_2_ARE_WORDS vs bytes tracked in raw)
	raw   []byte
}

// glyph is a parsed glyf glyph: empty, simple or composite. coords holds
// absolute point coordinates (floats so instanced deltas can be applied);
// for composites, coords holds one (x, y) offset per component.
type glyph struct {
	nContours    int
	xMin, yMin   int
	xMax, yMax   int
	endPts       []uint16
	instructions []byte
	coords       [][2]float64
	flags        []byte // per point, keepFlags bits only
	composites   []component
	isComposite  bool
}

// parseGlyph parses one glyph record (without bbox recalculation).
func parseGlyph(data []byte) (*glyph, error) {
	g := &glyph{}
	if len(data) == 0 {
		return g, nil
	}
	if len(data) < 10 {
		return nil, fmt.Errorf("glyph record too short: %d bytes", len(data))
	}
	g.nContours = int(getI16(data, 0))
	g.xMin, g.yMin, g.xMax, g.yMax = int(getI16(data, 2)), int(getI16(data, 4)), int(getI16(data, 6)), int(getI16(data, 8))
	if g.nContours == 0 {
		pos := 10
		if len(data) >= pos+2 {
			n := int(getU16(data, pos))
			pos += 2
			if len(data) < pos+n {
				return nil, fmt.Errorf("glyph instructions out of range")
			}
			g.instructions = append([]byte(nil), data[pos:pos+n]...)
		}
		return g, nil
	}
	if g.nContours > 0 {
		return parseSimpleGlyph(g, data)
	}
	return parseCompositeGlyph(g, data)
}

func parseSimpleGlyph(g *glyph, data []byte) (*glyph, error) {
	n := g.nContours
	if len(data) < 10+2*n+2 {
		return nil, fmt.Errorf("simple glyph header too short")
	}
	g.endPts = make([]uint16, n)
	for i := 0; i < n; i++ {
		g.endPts[i] = getU16(data, 10+2*i)
	}
	pos := 10 + 2*n
	instrLen := int(getU16(data, pos))
	pos += 2
	if len(data) < pos+instrLen {
		return nil, fmt.Errorf("simple glyph instructions out of range")
	}
	g.instructions = append([]byte(nil), data[pos:pos+instrLen]...)
	pos += instrLen
	nPts := int(g.endPts[n-1]) + 1
	flags := make([]byte, 0, nPts)
	for len(flags) < nPts {
		if pos >= len(data) {
			return nil, fmt.Errorf("simple glyph flags out of range")
		}
		f := data[pos]
		pos++
		rep := 1
		if f&flagRepeat != 0 {
			if pos >= len(data) {
				return nil, fmt.Errorf("simple glyph repeat out of range")
			}
			rep = int(data[pos]) + 1
			pos++
		}
		for i := 0; i < rep; i++ {
			flags = append(flags, f&^flagRepeat)
		}
	}
	if len(flags) != nPts {
		return nil, fmt.Errorf("simple glyph flag count mismatch: %d != %d", len(flags), nPts)
	}
	// X and Y coordinates live in two separate byte streams.
	nx, ny := 0, 0
	for _, f := range flags {
		switch {
		case f&flagXShort != 0:
			nx++
		case f&flagXSame == 0:
			nx += 2
		}
		switch {
		case f&flagYShort != 0:
			ny++
		case f&flagYSame == 0:
			ny += 2
		}
	}
	if pos+nx+ny > len(data) {
		return nil, fmt.Errorf("simple glyph coordinate bytes out of range")
	}
	xBytes, yBytes := data[pos:pos+nx], data[pos+nx:pos+nx+ny]
	pos += nx + ny
	xs := make([]int, nPts)
	ys := make([]int, nPts)
	xi, yi := 0, 0
	for i, f := range flags {
		switch {
		case f&flagXShort != 0:
			v := int(xBytes[xi])
			xi++
			if f&flagXSame == 0 {
				v = -v
			}
			xs[i] = v
		case f&flagXSame != 0:
			xs[i] = 0
		default:
			xs[i] = int(int16(uint16(xBytes[xi])<<8 | uint16(xBytes[xi+1])))
			xi += 2
		}
		switch {
		case f&flagYShort != 0:
			v := int(yBytes[yi])
			yi++
			if f&flagYSame == 0 {
				v = -v
			}
			ys[i] = v
		case f&flagYSame != 0:
			ys[i] = 0
		default:
			ys[i] = int(int16(uint16(yBytes[yi])<<8 | uint16(yBytes[yi+1])))
			yi += 2
		}
	}
	g.coords = make([][2]float64, nPts)
	g.flags = make([]byte, nPts)
	var x, y int
	for i := range flags {
		x += xs[i]
		y += ys[i]
		g.coords[i] = [2]float64{float64(x), float64(y)}
		g.flags[i] = flags[i] & keepFlags
	}
	return g, nil
}

func parseCompositeGlyph(g *glyph, data []byte) (*glyph, error) {
	g.isComposite = true
	pos := 10
	for {
		if len(data) < pos+4 {
			return nil, fmt.Errorf("composite component out of range")
		}
		flags := getU16(data, pos)
		gid := getU16(data, pos+2)
		if flags&0x2000 != 0 { // GID_IS_24_BIT, only in extended glyf
			return nil, fmt.Errorf("24-bit composite glyph IDs not supported")
		}
		pos += 4
		c := component{flags: flags, gid: gid, xy: flags&compArgsXYValues != 0}
		argLen := 2
		if flags&compArgsWords != 0 {
			argLen = 4
		}
		need := argLen
		switch {
		case flags&compHaveScale != 0:
			need += 2
		case flags&compHaveXYScale != 0:
			need += 4
		case flags&compHave2x2 != 0:
			need += 8
		}
		if len(data) < pos+need {
			return nil, fmt.Errorf("composite component args out of range")
		}
		c.raw = append([]byte(nil), data[pos:pos+need]...)
		pos += need
		g.composites = append(g.composites, c)
		if flags&compMoreComponents == 0 {
			break
		}
	}
	// Optional instructions on the last component.
	if len(g.composites) > 0 && g.composites[len(g.composites)-1].flags&compHaveInstr != 0 {
		if len(data) < pos+2 {
			return nil, fmt.Errorf("composite instructions out of range")
		}
		n := int(getU16(data, pos))
		pos += 2
		if len(data) < pos+n {
			return nil, fmt.Errorf("composite instruction bytes out of range")
		}
		g.instructions = append([]byte(nil), data[pos:pos+n]...)
	}
	// coords holds one (x, y) offset per XY-valued component.
	g.coords = make([][2]float64, len(g.composites))
	for i, c := range g.composites {
		if !c.xy {
			continue
		}
		if c.flags&compArgsWords != 0 {
			g.coords[i] = [2]float64{float64(getI16(c.raw, 0)), float64(getI16(c.raw, 2))}
		} else {
			g.coords[i] = [2]float64{float64(int8(c.raw[0])), float64(int8(c.raw[1]))}
		}
	}
	return g, nil
}

// recalcBounds recomputes the bbox from coords (otRound each bound), or
// zeroes it for empty glyphs. Composite bounds need the component glyphs;
// flattenComposite resolves them.
func (g *glyph) recalcBounds(resolve func(gid uint16) *glyph) {
	if len(g.coords) == 0 && !g.isComposite {
		g.xMin, g.yMin, g.xMax, g.yMax = 0, 0, 0, 0
		return
	}
	if g.isComposite {
		// Flattened bounds like fontTools tryRecalcBoundsComposite with
		// fallback to full flattening (only integer translates supported
		// exactly; others fall back to stored bounds).
		if b, ok := flattenBounds(g, resolve, map[uint16]bool{}); ok {
			g.xMin, g.yMin, g.xMax, g.yMax = b[0], b[1], b[2], b[3]
		}
		return
	}
	minX, minY := g.coords[0][0], g.coords[0][1]
	maxX, maxY := minX, minY
	for _, p := range g.coords[1:] {
		if p[0] < minX {
			minX = p[0]
		}
		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}
	g.xMin, g.yMin, g.xMax, g.yMax = otRoundInt(minX), otRoundInt(minY), otRoundInt(maxX), otRoundInt(maxY)
}

// flattenBounds computes composite bounds by resolving components. Only
// components with pure integer XY translation are resolved exactly (mirrors
// fontTools tryRecalcBoundsComposite); others report failure.
func flattenBounds(g *glyph, resolve func(gid uint16) *glyph, seen map[uint16]bool) ([4]int, bool) {
	var b [4]int
	have := false
	for _, c := range g.composites {
		if c.flags&(compHaveScale|compHaveXYScale|compHave2x2) != 0 || !c.xy {
			return b, false
		}
		if seen[c.gid] {
			return b, false
		}
		base := resolve(c.gid)
		if base == nil {
			return b, false
		}
		seen[c.gid] = true
		bb, ok := [4]int{}, true
		if base.isComposite {
			bb, ok = flattenBounds(base, resolve, seen)
		} else {
			bb = [4]int{base.xMin, base.yMin, base.xMax, base.yMax}
		}
		delete(seen, c.gid)
		if !ok {
			return b, false
		}
		if bb[0] == bb[1] && bb[2] == bb[3] && bb[0] == 0 {
			// Empty component does not update bounds (mirrors fontTools).
			continue
		}
		var dx, dy int
		if c.flags&compArgsWords != 0 {
			dx, dy = int(getI16(c.raw, 0)), int(getI16(c.raw, 2))
		} else {
			dx, dy = int(int8(c.raw[0])), int(int8(c.raw[1]))
		}
		pts := [][2]int{{bb[0] + dx, bb[1] + dy}, {bb[2] + dx, bb[3] + dy}}
		for _, p := range pts {
			if !have {
				b = [4]int{p[0], p[1], p[0], p[1]}
				have = true
				continue
			}
			if p[0] < b[0] {
				b[0] = p[0]
			}
			if p[1] < b[1] {
				b[1] = p[1]
			}
			if p[0] > b[2] {
				b[2] = p[0]
			}
			if p[1] > b[3] {
				b[3] = p[1]
			}
		}
	}
	if !have {
		return [4]int{0, 0, 0, 0}, true
	}
	return b, true
}

// toIntCoords rounds absolute coordinates in place (fontTools toInt).
func (g *glyph) toIntCoords() {
	for i := range g.coords {
		g.coords[i][0] = otRound(g.coords[i][0])
		g.coords[i][1] = otRound(g.coords[i][1])
	}
}

// encodeGlyph serializes a glyph with the greedy flag packing of fontTools
// Glyph.compileCoordinates(optimizeSize=True). Coordinates must be integral.
func encodeGlyph(g *glyph, gidRemap map[uint16]uint16) ([]byte, error) {
	out := make([]byte, 0, 64)
	out = appendI16(out, int16(g.nContours))
	out = appendI16(out, int16(g.xMin))
	out = appendI16(out, int16(g.yMin))
	out = appendI16(out, int16(g.xMax))
	out = appendI16(out, int16(g.yMax))
	if g.nContours == 0 && !g.isComposite {
		out = appendU16(out, uint16(len(g.instructions)))
		out = append(out, g.instructions...)
		return out, nil
	}
	if g.isComposite {
		return encodeComposite(g, out, gidRemap)
	}
	for _, e := range g.endPts {
		out = appendU16(out, e)
	}
	out = appendU16(out, uint16(len(g.instructions)))
	out = append(out, g.instructions...)
	// Relative integer deltas from rounded absolute coordinates
	// (mirrors Glyph.compileCoordinates: toInt then absoluteToRelative).
	xs := make([]int, len(g.coords))
	ys := make([]int, len(g.coords))
	px, py := 0, 0
	for i, p := range g.coords {
		xi, yi := otRoundInt(p[0]), otRoundInt(p[1])
		xs[i], ys[i] = xi-px, yi-py
		px, py = xi, yi
	}
	var cflags, cxs, cys []byte
	var lastflag byte
	repeat := 0
	first := true
	for i := range g.coords {
		flag := g.flags[i] & keepFlags
		x, y := xs[i], ys[i]
		if x == 0 {
			flag |= flagXSame
		} else if x >= -255 && x <= 255 {
			flag |= flagXShort
			if x > 0 {
				flag |= flagXSame
				cxs = append(cxs, byte(x))
			} else {
				cxs = append(cxs, byte(-x))
			}
		} else {
			if x < -32768 || x > 32767 {
				return nil, fmt.Errorf("x delta out of range: %d", x)
			}
			cxs = append(cxs, byte(x>>8), byte(x))
		}
		if y == 0 {
			flag |= flagYSame
		} else if y >= -255 && y <= 255 {
			flag |= flagYShort
			if y > 0 {
				flag |= flagYSame
				cys = append(cys, byte(y))
			} else {
				cys = append(cys, byte(-y))
			}
		} else {
			if y < -32768 || y > 32767 {
				return nil, fmt.Errorf("y delta out of range: %d", y)
			}
			cys = append(cys, byte(y>>8), byte(y))
		}
		if !first && flag == lastflag && repeat != 255 {
			repeat++
			if repeat == 1 {
				cflags = append(cflags, flag)
			} else {
				cflags[len(cflags)-2] = flag | flagRepeat
				cflags[len(cflags)-1] = byte(repeat)
			}
		} else {
			repeat = 0
			cflags = append(cflags, flag)
			first = false
		}
		lastflag = flag
	}
	out = append(out, cflags...)
	out = append(out, cxs...)
	out = append(out, cys...)
	return out, nil
}

func encodeComposite(g *glyph, out []byte, gidRemap map[uint16]uint16) ([]byte, error) {
	for i, c := range g.composites {
		flags := c.flags
		if i < len(g.composites)-1 {
			flags |= compMoreComponents
		} else {
			flags &^= compMoreComponents
		}
		newGID := c.gid
		if gidRemap != nil {
			m, ok := gidRemap[c.gid]
			if !ok {
				return nil, fmt.Errorf("composite component GID %d dropped by subset", c.gid)
			}
			newGID = m
		}
		out = appendU16(out, flags)
		out = appendU16(out, newGID)
		raw := c.raw
		if c.xy {
			// Refresh XY offsets from coords (instancing may have moved them).
			raw = append([]byte(nil), c.raw...)
			if c.flags&compArgsWords != 0 {
				if len(raw) < 4 {
					return nil, fmt.Errorf("composite args too short")
				}
				xi, yi := otRoundInt(g.coords[i][0]), otRoundInt(g.coords[i][1])
				putU16(raw, 0, uint16(int16(xi)))
				putU16(raw, 2, uint16(int16(yi)))
			} else {
				if len(raw) < 2 {
					return nil, fmt.Errorf("composite args too short")
				}
				raw[0] = byte(int8(otRoundInt(g.coords[i][0])))
				raw[1] = byte(int8(otRoundInt(g.coords[i][1])))
			}
		}
		out = append(out, raw...)
	}
	if len(g.composites) > 0 && g.composites[len(g.composites)-1].flags&compHaveInstr != 0 {
		out = appendU16(out, uint16(len(g.instructions)))
		out = append(out, g.instructions...)
	}
	return out, nil
}

// splitGlyf splits concatenated glyf data using loca offsets.
func splitGlyf(data []byte, loca []uint32) ([][]byte, error) {
	if len(loca) == 0 {
		return nil, fmt.Errorf("empty loca table")
	}
	n := len(loca) - 1
	recs := make([][]byte, n)
	for i := 0; i < n; i++ {
		if loca[i+1] < loca[i] || int(loca[i+1]) > len(data) {
			return nil, fmt.Errorf("bad loca entry %d: %d..%d (glyf %d)", i, loca[i], loca[i+1], len(data))
		}
		recs[i] = data[loca[i]:loca[i+1]]
	}
	return recs, nil
}

// parseLoca parses short or long loca.
func parseLoca(data []byte, numGlyphs int, long bool) ([]uint32, error) {
	if long {
		if len(data) < 4*(numGlyphs+1) {
			return nil, fmt.Errorf("long loca too short: %d", len(data))
		}
		out := make([]uint32, numGlyphs+1)
		for i := range out {
			out[i] = getU32(data, 4*i)
		}
		return out, nil
	}
	if len(data) < 2*(numGlyphs+1) {
		return nil, fmt.Errorf("short loca too short: %d", len(data))
	}
	out := make([]uint32, numGlyphs+1)
	for i := range out {
		out[i] = uint32(getU16(data, 2*i)) * 2
	}
	return out, nil
}

// buildLoca pads odd-length glyph records when that allows short offsets
// (mirroring fontTools glyf.compile padding==1), then builds the loca array
// and reports whether short format applies (max < 0x20000 and all even),
// mirroring fontTools loca.compile.
func buildLoca(recs [][]byte) (padded [][]byte, loca []uint32, short bool) {
	padded = recs
	var total int
	for _, r := range recs {
		total += len(r)
	}
	if total < 0x20000 {
		need := 0
		for _, r := range recs {
			if len(r)%2 == 1 {
				need++
			}
		}
		if need > 0 && total+need < 0x20000 {
			padded = make([][]byte, len(recs))
			for i, r := range recs {
				if len(r)%2 == 1 {
					r = append(append([]byte(nil), r...), 0)
				}
				padded[i] = r
			}
		}
	}
	loca = make([]uint32, len(padded)+1)
	short = true
	for i, r := range padded {
		loca[i+1] = loca[i] + uint32(len(r))
		if loca[i+1]%2 != 0 || loca[i+1] >= 0x20000 {
			short = false
		}
	}
	return padded, loca, short
}

// encodeLoca encodes offsets in short or long format.
func encodeLoca(loca []uint32, short bool) []byte {
	if short {
		out := make([]byte, 0, 2*len(loca))
		for _, o := range loca {
			out = appendU16(out, uint16(o/2))
		}
		return out
	}
	out := make([]byte, 0, 4*len(loca))
	for _, o := range loca {
		out = appendU32(out, o)
	}
	return out
}

// trimHMetrics applies the fontTools hmtx.compile trimming rule and returns
// the table bytes plus the numberOfHMetrics value for hhea.
func trimHMetrics(metrics [][2]int) ([]byte, int) {
	adv := make([]int, len(metrics))
	lsb := make([]int, len(metrics))
	for i, m := range metrics {
		adv[i], lsb[i] = m[0], m[1]
	}
	last := len(metrics)
	if last == 0 {
		return []byte{}, 0
	}
	lastAdvance := adv[last-1]
	lastIndex := last
	for lastIndex-2 >= 0 && adv[lastIndex-2] == lastAdvance {
		lastIndex--
		if lastIndex <= 1 {
			lastIndex = 1
			break
		}
	}
	out := make([]byte, 0, 4*lastIndex+2*(last-lastIndex))
	for i := 0; i < lastIndex; i++ {
		out = appendU16(out, uint16(otRoundInt(float64(adv[i]))))
		out = appendI16(out, int16(otRoundInt(float64(lsb[i]))))
	}
	for i := lastIndex; i < last; i++ {
		out = appendI16(out, int16(otRoundInt(float64(lsb[i]))))
	}
	return out, lastIndex
}

// parseHmtx parses hmtx given numGlyphs and numberOfHMetrics.
func parseHmtx(data []byte, numGlyphs, numHMetrics int) ([][2]int, error) {
	if numHMetrics > numGlyphs {
		numHMetrics = numGlyphs
	}
	if len(data) < 4*numHMetrics+2*(numGlyphs-numHMetrics) {
		return nil, fmt.Errorf("hmtx too short: %d", len(data))
	}
	out := make([][2]int, numGlyphs)
	pos := 0
	for i := 0; i < numHMetrics; i++ {
		out[i] = [2]int{int(getU16(data, pos)), int(getI16(data, pos+2))}
		pos += 4
	}
	lastAdv := 0
	if numHMetrics > 0 {
		lastAdv = out[numHMetrics-1][0]
	}
	for i := numHMetrics; i < numGlyphs; i++ {
		out[i] = [2]int{lastAdv, int(getI16(data, pos))}
		pos += 2
	}
	return out, nil
}
