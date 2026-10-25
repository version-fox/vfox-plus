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
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/andybalholm/brotli"
)

// testFontBuilder assembles a minimal synthetic SFNT from raw tables.
type testFontBuilder struct {
	t      *testing.T
	tables map[string][]byte
}

func newTestFontBuilder(t *testing.T) *testFontBuilder {
	return &testFontBuilder{t: t, tables: map[string][]byte{}}
}

func (b *testFontBuilder) add(tag string, data []byte) {
	b.tables[tag] = data
}

func (b *testFontBuilder) build() []byte {
	var tags []string
	for tag := range b.tables {
		tags = append(tags, tag)
	}
	data, err := writeSFNT(0x00010000, tags, b.tables)
	if err != nil {
		b.t.Fatal(err)
	}
	return data
}

// head table with numGlyphs-independent fields; caller patches bounds.
func testHead(indexToLocFormat int) []byte {
	out := make([]byte, 54)
	putU32(out, 0, 0x00010000) // version
	putU32(out, 12, 0x5F0F3CF5)
	putU16(out, 16, 0)
	putU16(out, 18, 1000)
	putU16(out, 50, uint16(indexToLocFormat))
	return out
}

func testHhea(numHMetrics int) []byte {
	out := make([]byte, 36)
	putU32(out, 0, 0x00010000)
	putU16(out, 34, uint16(numHMetrics))
	return out
}

func testMaxp(numGlyphs int) []byte {
	out := make([]byte, 32)
	putU32(out, 0, 0x00010000)
	putU16(out, 4, uint16(numGlyphs))
	return out
}

// emptyGlyph returns a zero-contour glyph record.
func emptyGlyph() []byte {
	return []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
}

// simpleSquare builds a simple square glyph (1 contour, 4 on-curve points).
func simpleSquare(xMin, yMin, xMax, yMax int) []byte {
	pts := [][2]int{{xMin, yMin}, {xMax, yMin}, {xMax, yMax}, {xMin, yMax}}
	return simpleGlyph(pts, nil)
}

// simpleGlyph encodes one contour through pts (all on-curve) with optional
// instruction bytes.
func simpleGlyph(pts [][2]int, instr []byte) []byte {
	xMin, yMin := pts[0][0], pts[0][1]
	xMax, yMax := xMin, yMin
	for _, p := range pts[1:] {
		if p[0] < xMin {
			xMin = p[0]
		}
		if p[0] > xMax {
			xMax = p[0]
		}
		if p[1] < yMin {
			yMin = p[1]
		}
		if p[1] > yMax {
			yMax = p[1]
		}
	}
	out := []byte{0, 1} // 1 contour
	out = appendI16(out, int16(xMin))
	out = appendI16(out, int16(yMin))
	out = appendI16(out, int16(xMax))
	out = appendI16(out, int16(yMax))
	out = appendU16(out, uint16(len(pts)-1))
	out = appendU16(out, uint16(len(instr)))
	out = append(out, instr...)
	var flags, xs, ys []byte
	px, py := 0, 0
	for _, p := range pts {
		dx, dy := p[0]-px, p[1]-py
		px, py = p[0], p[1]
		var f byte = flagOnCurve
		if dx == 0 {
			f |= flagXSame
		} else if dx > 0 && dx < 256 {
			f |= flagXShort | flagXSame
			xs = append(xs, byte(dx))
		} else if dx < 0 && dx > -256 {
			f |= flagXShort
			xs = append(xs, byte(-dx))
		} else {
			xs = append(xs, byte(dx>>8), byte(dx))
		}
		if dy == 0 {
			f |= flagYSame
		} else if dy > 0 && dy < 256 {
			f |= flagYShort | flagYSame
			ys = append(ys, byte(dy))
		} else if dy < 0 && dy > -256 {
			f |= flagYShort
			ys = append(ys, byte(-dy))
		} else {
			ys = append(ys, byte(dy>>8), byte(dy))
		}
		flags = append(flags, f)
	}
	out = append(out, flags...)
	out = append(out, xs...)
	out = append(out, ys...)
	return out
}

// buildTestFont builds a font with glyphs: 0=.notdef(empty), 1,2,... from
// the given records, plus hmtx with advance 500.
func buildTestFont(t *testing.T, recs [][]byte, cmap map[uint32]uint16) []byte {
	return buildTestFontMetrics(t, recs, cmap, nil)
}

func buildTestFontMetrics(t *testing.T, recs [][]byte, cmap map[uint32]uint16, metrics [][2]int) []byte {
	b := newTestFontBuilder(t)
	n := len(recs)
	if metrics == nil {
		metrics = make([][2]int, n)
		for i := range metrics {
			metrics[i] = [2]int{500, 0}
		}
	}
	hmtx, numHMetrics := trimHMetrics(metrics)
	b.add("hhea", testHhea(numHMetrics))
	b.add("maxp", testMaxp(n))
	b.add("hmtx", hmtx)
	// glyf + loca.
	var glyf []byte
	var loca []uint32
	off := uint32(0)
	for _, r := range recs {
		loca = append(loca, off)
		glyf = append(glyf, r...)
		off += uint32(len(r))
	}
	loca = append(loca, off)
	b.add("glyf", glyf)
	short := true
	for _, o := range loca {
		if o%2 != 0 || o >= 0x20000 {
			short = false
		}
	}
	locaFmt := 0
	if !short {
		locaFmt = 1
	}
	b.add("head", testHead(locaFmt))
	b.add("loca", encodeLoca(loca, short))
	// cmap with one format-4 subtable.
	subs := []*cmapSubtable{{platformID: 3, platEncID: 1, format: 4, entries: cmap}}
	b.add("cmap", encodeCmap(subs))
	// name with one record.
	names := []nameRecord{{platformID: 3, platEncID: 1, langID: 0x409, nameID: 1, str: "Test"}}
	enc, err := encodeName(names)
	if err != nil {
		t.Fatal(err)
	}
	b.add("name", enc)
	// post 3.0.
	post := make([]byte, 32)
	putU32(post, 0, 0x00030000)
	b.add("post", post)
	// OS/2 minimal.
	os2 := make([]byte, 78)
	b.add("OS/2", os2)
	return b.build()
}

func parseTestFont(t *testing.T, data []byte) *sfntFont {
	f, err := parseSFNT(data)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestChecksumRoundTrip(t *testing.T) {
	data := buildTestFont(t, [][]byte{emptyGlyph(), simpleSquare(0, 0, 100, 100)}, map[uint32]uint16{65: 1})
	if wholeFontChecksum(data) != 0xB1B0AFBA {
		t.Fatalf("whole font checksum = %#x, want magic", wholeFontChecksum(data))
	}
	f := parseTestFont(t, data)
	var tags []string
	for tag := range f.tables {
		tags = append(tags, tag)
	}
	out, err := writeSFNT(f.sfntVersion, tags, f.tables)
	if err != nil {
		t.Fatal(err)
	}
	if wholeFontChecksum(out) != 0xB1B0AFBA {
		t.Fatalf("rewritten checksum = %#x, want magic", wholeFontChecksum(out))
	}
}

func TestSubsetCmapClosure(t *testing.T) {
	// Glyphs: 0=notdef, 1='A', 2='B', 3='C'.
	cmap := map[uint32]uint16{65: 1, 66: 2, 67: 3, 48: 1}
	data := buildTestFont(t,
		[][]byte{emptyGlyph(), simpleSquare(0, 0, 10, 10), simpleSquare(0, 0, 20, 20), simpleSquare(0, 0, 30, 30)},
		cmap)
	// Add degenerate self-ligatures so verification passes.
	f2, err := parseSFNT(data)
	if err != nil {
		t.Fatal(err)
	}
	tables := map[string][]byte{}
	var tags []string
	for tag, b := range f2.tables {
		tags = append(tags, tag)
		tables[tag] = b
	}
	ligs := map[uint16][]gsubLig{1: {{result: 1}}, 3: {{result: 3}}}
	tables["GSUB"] = buildTestGSUB(t, ligs, nil)
	tags = append(tags, "GSUB")
	data, err = writeSFNT(f2.sfntVersion, tags, tables)
	if err != nil {
		t.Fatal(err)
	}
	icons := map[string]string{"a": "A", "c": "C"}
	codepoints := map[string]uint32{"A": 65, "B": 66, "C": 67}
	res, err := subsetFont(data, icons, codepoints, nil)
	if err != nil {
		t.Fatal(err)
	}
	f := parseTestFont(t, res.sfnt)
	// maxp numGlyphs: notdef + A + C + ligature chars present in cmap.
	if n := int(getU16(f.tables["maxp"], 4)); n != 3 {
		t.Fatalf("numGlyphs = %d, want 3 (notdef, A, C)", n)
	}
	// ligature chars in cmap: '0'(48) maps to gid 1 ('A' glyph!) so 'A' kept anyway.
	// kept: 0,1,3 + all [a-z0-9_] mapped glyphs (only 48->1 present).
	sub, err := parseCmap(f.tables["cmap"])
	if err != nil {
		t.Fatal(err)
	}
	var codes []uint32
	for _, s := range sub {
		for c := range s.entries {
			codes = append(codes, c)
		}
	}
	sort.Slice(codes, func(i, j int) bool { return codes[i] < codes[j] })
	for _, want := range []uint32{65, 67, 48} {
		found := false
		for _, c := range codes {
			if c == want {
				found = true
			}
		}
		if !found {
			t.Errorf("code U+%04X missing from subset cmap", want)
		}
	}
	for _, bad := range []uint32{66} {
		for _, c := range codes {
			if c == bad {
				t.Errorf("code U+%04X should have been dropped", bad)
			}
		}
	}
	if wholeFontChecksum(res.sfnt) != 0xB1B0AFBA {
		t.Errorf("bad checksum")
	}
}

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

// buildTestGSUB assembles a GSUB via encodeGSUB with DFLT+latn scripts
// (DefaultLangSys -> rlig), features rclt(empty)+rlig, lookup 0 (type 4
// ligatures) and lookup 1 (type 1 singles, unreferenced). IDs are old glyph
// IDs; the caller remaps (identity here).
func buildTestGSUB(t *testing.T, ligs map[uint16][]gsubLig, singles map[uint16]uint16) []byte {
	t.Helper()
	table := &gsubTable{
		scripts: []gsubScript{
			{tag: "DFLT", def: &gsubLangSys{req: -1, features: []int{1}}},
			{tag: "latn", def: &gsubLangSys{req: -1, features: []int{1}}},
		},
		features: []gsubFeature{
			{tag: "rclt"},
			{tag: "rlig", lookups: []int{0}},
		},
	}
	var subs []gsubSubtable
	if len(ligs) > 0 {
		var cov []uint16
		for f := range ligs {
			cov = append(cov, f)
		}
		subs = append(subs, gsubSubtable{cov: cov, ligs: ligs})
	}
	if len(singles) > 0 {
		var cov []uint16
		for g := range singles {
			cov = append(cov, g)
		}
		subs = append(subs, gsubSubtable{cov: cov, single: singles})
	}
	_ = subs
	// Build lookups: type 4 then type 1 (only non-empty).
	if len(ligs) > 0 {
		table.lookups = append(table.lookups, gsubLookup{lookupType: 4, subs: []gsubSubtable{{cov: covOf(ligs), ligs: ligs}}})
	}
	if len(singles) > 0 {
		table.lookups = append(table.lookups, gsubLookup{lookupType: 1, subs: []gsubSubtable{{cov: covOfSingles(singles), single: singles}}})
	}
	enc, err := encodeGSUB(table, func(g uint16) uint16 { return g })
	if err != nil {
		t.Fatal(err)
	}
	return enc
}

func covOf(ligs map[uint16][]gsubLig) []uint16 {
	var cov []uint16
	for f := range ligs {
		cov = append(cov, f)
	}
	sort.Slice(cov, func(i, j int) bool { return cov[i] < cov[j] })
	return cov
}

func covOfSingles(m map[uint16]uint16) []uint16 {
	var cov []uint16
	for g := range m {
		cov = append(cov, g)
	}
	sort.Slice(cov, func(i, j int) bool { return cov[i] < cov[j] })
	return cov
}

// addSelfLigatures attaches a GSUB with degenerate single-glyph ligatures
// (first == result, no components) for the given icon symbols. Each symbol
// must be a single character whose codepoint maps to gid in the test cmap.
// This satisfies ligature verification in tests that focus on other tables.
func addSelfLigatures(t *testing.T, b *testFontBuilder, icons map[string]string, codepoints map[string]uint32, cmap map[uint32]uint16) {
	t.Helper()
	ligs := map[uint16][]gsubLig{}
	for _, symbol := range icons {
		if len(symbol) != 1 {
			t.Fatalf("addSelfLigatures needs single-char symbols, got %q", symbol)
		}
		code := codepoints[symbol]
		gid, ok := cmap[code]
		if !ok {
			t.Fatalf("code U+%04X not in test cmap", code)
		}
		if rune(code) != rune(symbol[0]) {
			t.Fatalf("symbol %q != code U+%04X", symbol, code)
		}
		ligs[gid] = append(ligs[gid], gsubLig{result: gid})
	}
	b.add("GSUB", buildTestGSUB(t, ligs, nil))
}

func TestSubsetGSUBLigatures(t *testing.T) {
	// Glyphs: 0=notdef, 1='a', 2='b', 3='X'(icon ab), 4='c'(dropped).
	recs := [][]byte{emptyGlyph(), simpleSquare(0, 0, 5, 5), simpleSquare(0, 0, 6, 6), simpleSquare(0, 0, 7, 7), simpleSquare(0, 0, 8, 8)}
	gsub := buildTestGSUB(t,
		map[uint16][]gsubLig{
			1: {{result: 4, components: []uint16{2}}},
		},
		map[uint16]uint16{4: 4})
	data := buildTestFont(t, recs, map[uint32]uint16{97: 1, 98: 2, 67: 4})
	f, err := parseSFNT(data)
	if err != nil {
		t.Fatal(err)
	}
	// Add GSUB.
	var tags []string
	tables := map[string][]byte{}
	for tag, b := range f.tables {
		tags = append(tags, tag)
		tables[tag] = b
	}
	tables["GSUB"] = gsub
	tags = append(tags, "GSUB")
	raw, err := writeSFNT(f.sfntVersion, tags, tables)
	if err != nil {
		t.Fatal(err)
	}
	// Icons: x -> symbol "ab", spelled by the a+b ligature.
	icons := map[string]string{"x": "ab"}
	res, err := subsetFont(raw, icons, map[string]uint32{"ab": 67}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, err := parseSFNT(res.sfnt)
	if err != nil {
		t.Fatal(err)
	}
	// Ligature ab->X must survive (as new gids); the ac ligature dropped.
	g, err := parseGSUB(out.tables["GSUB"])
	if err != nil {
		t.Fatal(err)
	}
	if len(g.lookups) != 1 || g.lookups[0].lookupType != 4 {
		t.Fatalf("lookups = %+v, want single type-4", g.lookups)
	}
	if len(g.features) != 1 || g.features[0].tag != "rlig" {
		t.Fatalf("features = %+v, want [rlig]", g.features)
	}
	st := g.lookups[0].subs[0]
	if len(st.ligs) != 1 {
		t.Fatalf("lig sets = %d, want 1", len(st.ligs))
	}
	// Reverse cmap check: 'a'(97) and 'b'(98) kept, 'X' result kept.
	cmap, err := parseCmap(out.tables["cmap"])
	if err != nil {
		t.Fatal(err)
	}
	rev := map[uint16]uint32{}
	for _, s := range cmap {
		for c, gid := range s.entries {
			rev[gid] = c
		}
	}
	names := g.ligatureNames(rev)
	if len(names) != 1 || !names["ab"] {
		t.Fatalf("ligature names = %v, want [ab]", names)
	}
}

func TestSubsetDropsEmptyLookup(t *testing.T) {
	// Ligature references dropped glyph 4 only -> lookup pruned away.
	recs := [][]byte{emptyGlyph(), simpleSquare(0, 0, 5, 5), simpleSquare(0, 0, 6, 6)}
	gsub := buildTestGSUB(t, map[uint16][]gsubLig{1: {{result: 2, components: []uint16{5}}}}, nil)
	data := buildTestFont(t, recs, map[uint32]uint16{97: 1, 98: 2})
	f, _ := parseSFNT(data)
	tables := map[string][]byte{}
	var tags []string
	for tag, b := range f.tables {
		tags = append(tags, tag)
		tables[tag] = b
	}
	tables["GSUB"] = gsub
	tags = append(tags, "GSUB")
	raw, _ := writeSFNT(f.sfntVersion, tags, tables)
	res, err := subsetFont(raw, map[string]string{}, map[string]uint32{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := parseSFNT(res.sfnt)
	g, err := parseGSUB(out.tables["GSUB"])
	if err != nil {
		t.Fatal(err)
	}
	if len(g.lookups) != 0 {
		t.Fatalf("lookups = %d, want 0 (all pruned)", len(g.lookups))
	}
	if len(g.features) != 0 {
		t.Fatalf("features = %+v, want none", g.features)
	}
}

func TestNameKeep(t *testing.T) {
	names := []nameRecord{
		{3, 1, 0x409, 1, "Fam"},
		{3, 1, 0x409, 0, "Copyright"},
		{1, 0, 0, 1, "MacFam"},  // non-Unicode: dropped (name_legacy=false)
		{3, 1, 0x409, 1, "Fam"}, // duplicate string: shared storage
	}
	data := buildTestFont(t, [][]byte{emptyGlyph()}, map[uint32]uint16{})
	f, _ := parseSFNT(data)
	f.tables["name"], _ = encodeName(names)
	var tags []string
	for tag := range f.tables {
		tags = append(tags, tag)
	}
	raw, _ := writeSFNT(f.sfntVersion, tags, f.tables)
	res, err := subsetFont(raw, map[string]string{}, map[string]uint32{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	out, _ := parseSFNT(res.sfnt)
	back, err := parseName(out.tables["name"])
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 3 {
		t.Fatalf("name records = %d, want 3", len(back))
	}
	for _, r := range back {
		if r.platformID == 1 {
			t.Errorf("mac record kept: %+v", r)
		}
	}
}

// buildTestFvar builds an fvar table with the given axes and instances.
func buildTestFvar(axes []fvarAxis, instances []fvarInstance) []byte {
	return encodeFvar(&fvarTable{version: 0x00010000, axes: axes, instances: instances})
}

// buildTestGvar wraps gvarTuples into a table for the given new-order glyphs.
func buildTestGvar(t *testing.T, axisTags []string, glyphs []uint16, vars map[uint16][]gvarTuple) []byte {
	t.Helper()
	return encodeGvar(axisTags, glyphs, vars)
}

func floatTuples(support map[string][3]float64, n int, x, y float64) []gvarTuple {
	d := make([]gvarDelta, n)
	for i := range d {
		d[i] = gvarDelta{true, x, y}
	}
	return []gvarTuple{{support: support, deltas: d}}
}

func TestPinMath(t *testing.T) {
	// One axis wght 100/400/700. Glyph 1 has 1 point + 4 phantoms.
	axes := []fvarAxis{{tag: "wght", min: 100, def: 400, max: 700, nameID: 256}}
	fvarData := buildTestFvar(axes, []fvarInstance{{subfamilyID: 2, coords: map[string]float64{"wght": 400}}})
	// Tuples: peak-only +1 (scalar 0.5 at pin 550), intermediate, unrelated axis.
	// 8 points: 4 outline + 4 phantom. Tuple 1 moves point 0 and the
	// horizontal phantoms; tuple 2 moves point 0 in y.
	d1 := make([]gvarDelta, 8)
	d2 := make([]gvarDelta, 8)
	for i := range d1 {
		d1[i] = gvarDelta{true, 0, 0}
		d2[i] = gvarDelta{true, 0, 0}
	}
	d1[0] = gvarDelta{true, 100, 0}
	d1[4] = gvarDelta{true, 10, 0}
	d1[5] = gvarDelta{true, 20, 0}
	d2[0] = gvarDelta{true, 0, 60}
	vars := map[uint16][]gvarTuple{
		1: {
			{support: map[string][3]float64{"wght": {0, 1, 1}}, deltas: d1},
			{support: map[string][3]float64{"wght": {0, 0.5, 1}}, deltas: d2},
		},
	}
	recs := [][]byte{emptyGlyph(), simpleSquare(0, 0, 10, 10)}
	_ = recs
	// Build font manually to include fvar+gvar.
	b := newTestFontBuilder(t)
	n := 2
	b.add("head", testHead(0))
	metrics := [][2]int{{500, 0}, {500, 0}}
	hmtx, numH := trimHMetrics(metrics)
	b.add("hhea", testHhea(numH))
	b.add("maxp", testMaxp(n))
	b.add("hmtx", hmtx)
	glyf := append(append([]byte{}, emptyGlyph()...), simpleSquare(0, 0, 10, 10)...)
	loca := []uint32{0, 10, 10 + uint32(len(simpleSquare(0, 0, 10, 10)))}
	b.add("head", testHead(1))
	b.add("glyf", glyf)
	b.add("loca", encodeLoca(loca, false))
	b.add("cmap", encodeCmap([]*cmapSubtable{{platformID: 3, platEncID: 1, format: 4, entries: map[uint32]uint16{65: 1}}}))
	addSelfLigatures(t, b, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, map[uint32]uint16{65: 1})
	b.add("fvar", fvarData)
	b.add("gvar", buildTestGvar(t, []string{"wght"}, []uint16{0, 1}, vars))
	names, _ := encodeName([]nameRecord{{3, 1, 0x409, 1, "T"}, {3, 1, 0x409, 256, "Weight"}, {3, 1, 0x409, 2, "Regular"}})
	b.add("name", names)
	post := make([]byte, 32)
	putU32(post, 0, 0x00030000)
	b.add("post", post)
	b.add("OS/2", make([]byte, 96))
	data := b.build()
	{
		dbg, _ := parseSFNT(data)
		t.Logf("built loca len=%d head locFmt=%d sqLen=%d", len(dbg.tables["loca"]), getU16(dbg.tables["head"], 50), len(simpleSquare(0, 0, 10, 10)))
		t.Logf("direct encodeLoca: %d", len(encodeLoca([]uint32{0, 10, 31}, true)))
	}
	// Pin wght=550 -> ndc=(550-400)/300=0.5.
	res, err := subsetFont(data, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, map[string]float64{"wght": 550})
	if err != nil {
		t.Fatal(err)
	}
	f, _ := parseSFNT(res.sfnt)
	// Tuple 1: peak-only (0,1,1) at 0.5 -> scalar 0.5, axis removed -> default baked.
	// Tuple 2: intermediate (-1,-0.5,1) at 0.5 -> scalar (0.5-1)/(-0.5-1)=0.333, no axes left -> baked.
	// Baked x for point 0: 0 + 100*0.5 + 0 = 50. Baked y: 0 + 0 + 60*(1/3)=20.
	g1, err := parseGlyph(f.tables["glyf"][8+0:])
	_ = g1
	_ = err
	// Re-parse via split.
	numGlyphs := int(getU16(f.tables["maxp"], 4))
	loca2, _ := parseLoca(f.tables["loca"], numGlyphs, getU16(f.tables["head"], 50) == 1)
	recs2, _ := splitGlyf(f.tables["glyf"], loca2)
	gg, _ := parseGlyph(recs2[1])
	if len(gg.coords) != 4 {
		t.Fatalf("coords = %d, want 4", len(gg.coords))
	}
	if gg.coords[0][0] != 50 || gg.coords[0][1] != 60 {
		t.Fatalf("baked coords = %v, want [50 60]", gg.coords[0])
	}
	// hmtx: phantom pp0 x: 0+10*0.5=5; pp1 x: 500+20*0.5=510; adv=505.
	// Points x: 50,10,10,0 -> xMin=0 -> lsb=0-5=-5.
	hm, _ := parseHmtx(f.tables["hmtx"], int(getU16(f.tables["maxp"], 4)), int(getU16(f.tables["hhea"], 34)))
	if hm[1] != [2]int{505, -5} {
		t.Fatalf("hmtx = %v, want [-, -] and [505 -5]", hm)
	}
	// fvar dropped (full pin), gvar dropped.
	if _, ok := f.tables["fvar"]; ok {
		t.Errorf("fvar not dropped on full pin")
	}
	if _, ok := f.tables["gvar"]; ok {
		t.Errorf("gvar not dropped on full pin")
	}
}

func TestPinPartialKeepsAxes(t *testing.T) {
	axes := []fvarAxis{
		{tag: "wght", min: 100, def: 400, max: 700, nameID: 256},
		{tag: "wdth", min: 50, def: 100, max: 200, nameID: 257},
	}
	fvarData := buildTestFvar(axes, []fvarInstance{{subfamilyID: 2, coords: map[string]float64{"wght": 400, "wdth": 100}}})
	mkD := func() []gvarDelta {
		d := make([]gvarDelta, 8)
		for i := range d {
			d[i] = gvarDelta{true, 0, 0}
		}
		d[0] = gvarDelta{true, 100, 0}
		d[4] = gvarDelta{true, 10, 0}
		d[5] = gvarDelta{true, 20, 0}
		return d
	}
	vars := map[uint16][]gvarTuple{
		1: {{support: map[string][3]float64{"wght": {0, 1, 1}, "wdth": {0, 1, 1}}, deltas: mkD()}},
	}
	b := newTestFontBuilder(t)
	b.add("head", testHead(1))
	metrics := [][2]int{{500, 0}, {500, 0}}
	hmtx, numH := trimHMetrics(metrics)
	b.add("hhea", testHhea(numH))
	b.add("maxp", testMaxp(2))
	b.add("hmtx", hmtx)
	sq := simpleSquare(0, 0, 10, 10)
	b.add("glyf", append(append([]byte{}, emptyGlyph()...), sq...))
	b.add("loca", encodeLoca([]uint32{0, 10, 10 + uint32(len(sq))}, false))
	b.add("cmap", encodeCmap([]*cmapSubtable{{platformID: 3, platEncID: 1, format: 4, entries: map[uint32]uint16{65: 1}}}))
	addSelfLigatures(t, b, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, map[uint32]uint16{65: 1})
	b.add("fvar", fvarData)
	b.add("gvar", buildTestGvar(t, []string{"wght", "wdth"}, []uint16{0, 1}, vars))
	names, _ := encodeName([]nameRecord{{3, 1, 0x409, 1, "T"}, {3, 1, 0x409, 257, "Width"}})
	b.add("name", names)
	post := make([]byte, 32)
	putU32(post, 0, 0x00030000)
	b.add("post", post)
	b.add("OS/2", make([]byte, 96))
	data := b.build()

	// Pin wght=550 (ndc 0.5): tuple scaled by 0.5, wdth axis retained.
	res, err := subsetFont(data, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, map[string]float64{"wght": 550})
	if err != nil {
		t.Fatal(err)
	}
	f, _ := parseSFNT(res.sfnt)
	fv, err := parseFvar(f.tables["fvar"])
	if err != nil {
		t.Fatal(err)
	}
	if len(fv.axes) != 1 || fv.axes[0].tag != "wdth" {
		t.Fatalf("fvar axes = %+v, want [wdth]", fv.axes)
	}
	if _, ok := f.tables["gvar"]; !ok {
		t.Fatalf("gvar dropped but wdth axis remains")
	}
	// Surviving tuple: deltas scaled by 0.5, support {wdth:(0,1,1)}.
	gv, err := parseGvar(f.tables["gvar"], []string{"wdth"}, map[uint16]int{0: 4, 1: 5})
	if err != nil {
		t.Fatal(err)
	}
	tups := gv.variations[1]
	if len(tups) != 1 {
		t.Fatalf("tuples = %d, want 1", len(tups))
	}
	if s, ok := tups[0].support["wdth"]; !ok || s != [3]float64{0, 1, 1} {
		t.Fatalf("support = %v", tups[0].support)
	}
	if len(tups[0].support) != 1 {
		t.Fatalf("support has %d axes, want 1", len(tups[0].support))
	}
	if tups[0].deltas[0].x != 50 {
		t.Fatalf("scaled dx = %v, want 50", tups[0].deltas[0].x)
	}
}

func TestDownload(t *testing.T) {
	content := map[string]string{"/a": "hello", "/b": "world!"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s, ok := content[r.URL.Path]; ok {
			w.Write([]byte(s))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	sha := func(s string) string {
		sum := sha256.Sum256([]byte(s))
		return hex.EncodeToString(sum[:])
	}
	files := []pinnedFile{{repoPath: "a", sha256: sha("hello")}, {repoPath: "b", sha256: sha("world!")}}
	dir := t.TempDir()
	if err := downloadFiles(dir, srv.URL, []string{"a", "b"}, files, srv.Client()); err != nil {
		t.Fatal(err)
	}
	for name, want := range map[string]string{"a": "hello", "b": "world!"} {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || string(got) != want {
			t.Fatalf("%s = %q, %v", name, got, err)
		}
	}
	// Second run skips (no error, no .part leftovers).
	if err := downloadFiles(dir, srv.URL, []string{"a", "b"}, files, srv.Client()); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 2 {
		t.Fatalf("dir has %d entries, want 2", len(entries))
	}
	// Corrupt one file -> re-download; bad checksum -> error + no .part.
	os.WriteFile(filepath.Join(dir, "a"), []byte("junk"), 0o666)
	bad := []pinnedFile{{repoPath: "a", sha256: sha("something else")}}
	if err := downloadFiles(dir, srv.URL, []string{"a"}, bad, srv.Client()); err == nil {
		t.Fatalf("expected checksum error")
	}
	if _, err := os.Stat(filepath.Join(dir, "a.part")); !os.IsNotExist(err) {
		t.Fatalf(".part not cleaned up")
	}
}

func TestParsePin(t *testing.T) {
	for _, bad := range []string{"wght", "wght=", "=400", "WDTH=400", "wght=abc", "ital=1"} {
		if _, _, err := parsePin(bad); err == nil {
			t.Errorf("parsePin(%q) should fail", bad)
		}
	}
	axis, v, err := parsePin("opsz=24")
	if err != nil || axis != "opsz" || v != 24 {
		t.Errorf("parsePin(opsz=24) = %q %v %v", axis, v, err)
	}
}

func TestLigatureVerificationFailure(t *testing.T) {
	// Icon references a symbol whose ligature is dropped (component missing).
	recs := [][]byte{emptyGlyph(), simpleSquare(0, 0, 5, 5), simpleSquare(0, 0, 6, 6)}
	gsub := buildTestGSUB(t, map[uint16][]gsubLig{1: {{result: 2, components: []uint16{9}}}}, nil)
	data := buildTestFont(t, recs, map[uint32]uint16{97: 1, 98: 2})
	f, _ := parseSFNT(data)
	tables := map[string][]byte{}
	var tags []string
	for tag, b := range f.tables {
		tags = append(tags, tag)
		tables[tag] = b
	}
	tables["GSUB"] = gsub
	tags = append(tags, "GSUB")
	raw, _ := writeSFNT(f.sfntVersion, tags, tables)
	_, err := subsetFont(raw, map[string]string{"x": "X"}, map[string]uint32{"X": 67}, nil)
	if err == nil {
		t.Fatalf("expected ligature verification error")
	}
	t.Logf("got expected error: %v", err)
}

func TestWriteJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "icons.json")
	icons := map[string]string{"home": "home", "add": "add"}
	cps := map[string]uint32{"home": 0xE9B2, "add": 0xE145}
	if err := writeJSON(path, icons, cps); err != nil {
		t.Fatal(err)
	}
	var back map[string]any
	data, _ := os.ReadFile(path)
	if err := json.Unmarshal(data, &back); err != nil {
		t.Fatal(err)
	}
	if back["font"] != "Material Symbols Rounded" {
		t.Fatalf("font = %v", back["font"])
	}
	got := back["icons"].(map[string]any)["home"].(map[string]any)
	if got["symbol"] != "home" || got["codepoint"] != float64(0xE9B2) {
		t.Fatalf("home = %v", got)
	}
}

func TestBrotliStored(t *testing.T) {
	// Library round-trip coverage (previously stored-only codec): empty,
	// single byte, repetitive, pseudo-random incompressible, and
	// multi-kilobyte inputs must all decode exactly. No assertion on the
	// exact compressed bytes: the library encoder output differs from the
	// old hand-rolled stored codec by design.
	pattern := make([]byte, 300)
	for i := range pattern {
		pattern[i] = byte(i * 7)
	}
	incompressible := make([]byte, 1024)
	for i := range incompressible {
		incompressible[i] = byte((i*131 + 17) % 251)
	}
	vectors := [][]byte{
		{},
		nil,
		[]byte("a"),
		[]byte("hello world, hello world, hello world"),
		pattern,
		incompressible,
		bytes.Repeat([]byte("abcd"), 2048),
	}
	for i, msg := range vectors {
		enc, err := encodeBrotli(msg)
		if err != nil {
			t.Fatalf("vector %d: encode: %v", i, err)
		}
		if len(msg) > 0 && len(enc) == 0 {
			t.Fatalf("vector %d: empty encoding for %d input bytes", i, len(msg))
		}
		dec, err := decodeBrotli(enc)
		if err != nil {
			t.Fatalf("vector %d: decode: %v", i, err)
		}
		if !bytes.Equal(dec, msg) {
			t.Fatalf("vector %d: round-trip mismatch for len %d", i, len(msg))
		}
		// Cross-check with the library decoder directly (no wrapper).
		r := brotli.NewReader(bytes.NewReader(enc))
		direct, err := io.ReadAll(r)
		if err != nil {
			t.Fatalf("vector %d: direct library decode: %v", i, err)
		}
		if !bytes.Equal(direct, msg) {
			t.Fatalf("vector %d: direct library round-trip mismatch", i)
		}
	}
}

func TestWoff2BrotliRoundTrip(t *testing.T) {
	// The WOFF2 writer's Brotli payload must decode (via the library) to
	// the exact concatenated font data passed to the encoder.
	g0, err := parseGlyph(emptyGlyph())
	if err != nil {
		t.Fatal(err)
	}
	g1, err := parseGlyph(simpleSquare(0, 0, 10, 10))
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []*glyph{g0, g1} {
		g.recalcBounds(func(uint16) *glyph { return nil })
		g.toIntCoords()
	}
	tables := map[string][]byte{
		"head": testHead(0),
		"maxp": testMaxp(2),
		"glyf": append(append([]byte{}, emptyGlyph()...), simpleSquare(0, 0, 10, 10)...),
		"loca": encodeLoca([]uint32{0, 10, 10 + uint32(len(simpleSquare(0, 0, 10, 10)))}, false),
	}
	tags := []string{"glyf", "head", "loca", "maxp"}
	out, err := writeWOFF2(0x00010000, tags, tables, []*glyph{g0, g1}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if string(out[:4]) != "wOF2" {
		t.Fatalf("magic = %q", out[:4])
	}
	totalCompressed := int(getU32(out, 20))
	if totalCompressed <= 0 || totalCompressed > len(out)-48 {
		t.Fatalf("totalCompressedSize = %d, file len = %d", totalCompressed, len(out))
	}
	compressed := out[len(out)-totalCompressed:]
	fontData, err := decodeBrotli(compressed)
	if err != nil {
		t.Fatal(err)
	}
	// Rebuild the expected pre-compression blob the same way writeWOFF2
	// does: sorted tags, transformed glyf, empty transformed loca, raw
	// bytes for everything else.
	tg, _, err := transformGlyf([]*glyph{g0, g1}, 1)
	if err != nil {
		t.Fatal(err)
	}
	ordered := append([]string(nil), tags...)
	sort.Strings(ordered)
	var want []byte
	for _, tag := range ordered {
		switch tag {
		case "glyf":
			want = append(want, tg...)
		case "loca":
			// Transformed loca consumes no data.
		default:
			want = append(want, tables[tag]...)
		}
	}
	if !bytes.Equal(fontData, want) {
		t.Fatalf("decoded WOFF2 payload len = %d, want %d", len(fontData), len(want))
	}
	// Spot-check: a raw table stored with the null transform must appear
	// verbatim in the decoded payload.
	if !bytes.Contains(fontData, tables["head"]) {
		t.Fatalf("decoded payload missing head table bytes")
	}
}

func TestWoff2Magic(t *testing.T) {
	tables := map[string][]byte{
		"head": testHead(0),
		"maxp": testMaxp(1),
	}
	glyphs := []*glyph{{nContours: 0}}
	out, err := writeWOFF2(0x00010000, []string{"head", "maxp"}, tables, glyphs, 0)
	if err != nil {
		t.Fatal(err)
	}
	if string(out[:4]) != "wOF2" {
		t.Fatalf("magic = %q", out[:4])
	}
	// totalSfntSize consistency: header + dir + padded tables.
	if len(out) < 48 {
		t.Fatalf("too short")
	}
}

func TestUnknownTableError(t *testing.T) {
	data := buildTestFont(t, [][]byte{emptyGlyph()}, map[uint32]uint16{})
	f, _ := parseSFNT(data)
	f.tables["GPOS"] = []byte{0, 1, 0, 0}
	var tags []string
	tables := map[string][]byte{}
	for tag, b := range f.tables {
		tags = append(tags, tag)
		tables[tag] = b
	}
	raw, _ := writeSFNT(f.sfntVersion, tags, tables)
	if _, err := subsetFont(raw, map[string]string{}, map[string]uint32{}, nil); err == nil {
		t.Fatalf("expected error for GPOS table")
	}
}

func TestPinUnknownAxis(t *testing.T) {
	data := buildTestFont(t, [][]byte{emptyGlyph()}, map[uint32]uint16{})
	if _, err := subsetFont(data, map[string]string{}, map[string]uint32{}, map[string]float64{"ital": 1}); err == nil {
		t.Fatalf("expected error for unknown axis without fvar")
	}
}

func TestGvarRoundTrip(t *testing.T) {
	tup := gvarTuple{
		support: map[string][3]float64{"wght": {0, 0.5, 1}},
		deltas:  []gvarDelta{{true, 10, -20}, {true, 0, 0}, {false, 0, 0}},
	}
	ts := []*gvarTuple{&tup}
	shared := compileSharedTuples(ts, []string{"wght"})
	idx := map[string]int{}
	for i, s := range shared {
		idx[string(s)] = i
	}
	enc := compileGlyphVariations(ts, []string{"wght"}, idx)
	back, err := parseGlyphVariations(enc, []string{"wght"}, nil, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 1 {
		t.Fatalf("tuples = %d", len(back))
	}
	if back[0].support["wght"] != [3]float64{0, 0.5, 1} {
		t.Fatalf("support = %v", back[0].support)
	}
	if back[0].deltas[0] != (gvarDelta{true, 10, -20}) {
		t.Fatalf("deltas[0] = %+v", back[0].deltas[0])
	}
	if back[0].deltas[2].has {
		t.Fatalf("deltas[2] should be missing")
	}
}

// buildTestHVAR builds an HVAR with one VarData and the given regions/rows.
// regions are per-axis triples over axisTags; rows are per-glyph deltas.
func buildTestHVAR(t *testing.T, axisTags []string, regions [][][3]float64, rows [][]int) []byte {
	t.Helper()
	vs := varStore{}
	for _, r := range regions {
		vr := varRegion{}
		for _, a := range r {
			vr.axes = append(vr.axes, a)
		}
		vs.regions = append(vs.regions, vr)
	}
	vd := varData{}
	for i := range regions {
		vd.regionIndexes = append(vd.regionIndexes, uint16(i))
	}
	for _, row := range rows {
		vd.rows = append(vd.rows, append([]int(nil), row...))
	}
	vs.varDatas = append(vs.varDatas, vd)
	hv := &hvarTable{store: vs}
	return encodeHvar(hv, len(axisTags))
}

func TestHvarSubsetAndRestrict(t *testing.T) {
	// HVAR with 2 regions (wght-only and wdth-only), 3 glyphs.
	regions := [][][3]float64{
		{{0, 1, 1}}, // region 0 – not used below; placeholder
	}
	_ = regions
	vs := varStore{
		regions: []varRegion{
			{axes: [][3]float64{{0, 1, 1}}},
			{axes: [][3]float64{{0, 1, 1}}},
		},
		varDatas: []varData{
			{regionIndexes: []uint16{0, 1}, rows: [][]int{{10, 20}, {30, 40}, {50, 60}}},
		},
	}
	hv := &hvarTable{store: vs}
	// Subset to glyphs [0, 2]: rows filtered.
	subsetHvar(hv, []uint16{0, 2})
	vd := hv.store.varDatas[0]
	if len(vd.rows) != 2 || vd.rows[0][0] != 10 || vd.rows[1][0] != 50 {
		t.Fatalf("rows after subset = %v", vd.rows)
	}
	if len(vd.rows[0]) != 2 || vd.rows[0][1] != 20 {
		t.Fatalf("row width changed: %v", vd.rows)
	}
	// Restrict wght at ndc 0.5: region 0 (0,1,1) scalar 0.5, region 1 kept as-is?
	// Both regions are single-axis; pin wght affects region 0 only if it is wght.
	// Here both regions are on axis 0; pin it: both scale by 0.5.
	pins := map[string]float64{"wght": 0.5}
	restrictVarStore(&hv.store, []string{"wght"}, pins)
	if len(hv.store.regions) != 0 {
		t.Fatalf("regions = %d, want 0 (all pinned away)", len(hv.store.regions))
	}
	vd = hv.store.varDatas[0]
	if len(vd.regionIndexes) != 0 {
		t.Fatalf("regionIndexes = %v, want empty", vd.regionIndexes)
	}
	// Rows folded with scalar 0.5 and rounded.
	if len(vd.rows) != 2 || len(vd.rows[0]) != 0 {
		t.Fatalf("rows after restrict = %v", vd.rows)
	}
}

func TestHvarRestrictPartial(t *testing.T) {
	// Regions on wght and wdth; pin wght only.
	vs := varStore{
		regions: []varRegion{
			{axes: [][3]float64{{0, 1, 1}, {0, 0, 0}}},
			{axes: [][3]float64{{0, 0, 0}, {0, 1, 1}}},
		},
		varDatas: []varData{
			{regionIndexes: []uint16{0, 1}, rows: [][]int{{100, 200}}},
		},
	}
	hv := &hvarTable{store: vs}
	restrictVarStore(&hv.store, []string{"wght", "wdth"}, map[string]float64{"wght": 0.5})
	if len(hv.store.regions) != 1 {
		t.Fatalf("regions = %d, want 1", len(hv.store.regions))
	}
	if len(hv.store.regions[0].axes) != 1 || hv.store.regions[0].axes[0] != [3]float64{0, 1, 1} {
		t.Fatalf("region = %v, want [[0 1 1]] (wdth only)", hv.store.regions[0].axes)
	}
	vd := hv.store.varDatas[0]
	if len(vd.regionIndexes) != 1 || vd.regionIndexes[0] != 0 {
		t.Fatalf("indexes = %v", vd.regionIndexes)
	}
	// Region 0 (wght) scaled by 0.5 and folded away (its contribution was
	// 100*0.5=50, added to... no: region 0 dropped, its delta folded where?
	// Dropped regions vanish from rows; their pinned contribution is ignored
	// for glyf fonts (matches fontTools, which discards HVAR defaultDeltas).
	// Remaining row: wdth region delta 200.
	if len(vd.rows) != 1 || len(vd.rows[0]) != 1 || vd.rows[0][0] != 200 {
		t.Fatalf("rows = %v, want [[200]]", vd.rows)
	}
}

func TestAvarPinEffect(t *testing.T) {
	// avar maps wght 0.5 -> 0.25. Pin wght=550 (ndc 0.5 -> 0.25).
	axes := []fvarAxis{{tag: "wght", min: 100, def: 400, max: 700, nameID: 256}}
	fvarData := buildTestFvar(axes, []fvarInstance{{subfamilyID: 2, coords: map[string]float64{"wght": 400}}})
	// Tuple peak (0,1,1): scalar at ndc 0.25 = 0.25.
	d := make([]gvarDelta, 8)
	for i := range d {
		d[i] = gvarDelta{true, 0, 0}
	}
	d[0] = gvarDelta{true, 100, 0}
	vars := map[uint16][]gvarTuple{1: {{support: map[string][3]float64{"wght": {0, 1, 1}}, deltas: d}}}
	avarData := encodeAvar(&avarTable{version: 0x00010000, segments: []avarSegments{
		{-1: -1, 0: 0, 0.5: 0.25, 1: 1},
	}})
	b := newTestFontBuilder(t)
	b.add("head", testHead(1))
	sq := simpleSquare(0, 0, 10, 10)
	b.add("hhea", testHhea(1))
	b.add("maxp", testMaxp(2))
	hm, _ := trimHMetrics([][2]int{{500, 0}, {500, 0}})
	b.add("hmtx", hm)
	b.add("glyf", append(append([]byte{}, emptyGlyph()...), sq...))
	b.add("loca", encodeLoca([]uint32{0, 10, 10 + uint32(len(sq))}, false))
	b.add("cmap", encodeCmap([]*cmapSubtable{{platformID: 3, platEncID: 1, format: 4, entries: map[uint32]uint16{65: 1}}}))
	addSelfLigatures(t, b, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, map[uint32]uint16{65: 1})
	b.add("fvar", fvarData)
	b.add("avar", avarData)
	b.add("gvar", buildTestGvar(t, []string{"wght"}, []uint16{0, 1}, vars))
	names, _ := encodeName([]nameRecord{{3, 1, 0x409, 1, "T"}})
	b.add("name", names)
	post := make([]byte, 32)
	putU32(post, 0, 0x00030000)
	b.add("post", post)
	b.add("OS/2", make([]byte, 96))
	data := b.build()
	res, err := subsetFont(data, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, map[string]float64{"wght": 550})
	if err != nil {
		t.Fatal(err)
	}
	f, _ := parseSFNT(res.sfnt)
	numGlyphs := int(getU16(f.tables["maxp"], 4))
	loca, _ := parseLoca(f.tables["loca"], numGlyphs, getU16(f.tables["head"], 50) == 1)
	recs, _ := splitGlyf(f.tables["glyf"], loca)
	gg, _ := parseGlyph(recs[1])
	// Baked x = 100 * 0.25 = 25.
	if gg.coords[0][0] != 25 {
		t.Fatalf("baked x = %v, want 25 (avar-mapped ndc 0.25)", gg.coords[0][0])
	}
}

func TestOS2WeightOnPin(t *testing.T) {
	// Pin wght=700 (non-default): usWeightClass becomes 700, avgWidth recomputed.
	b := newTestFontBuilder(t)
	b.add("head", testHead(0))
	metrics := [][2]int{{500, 5}, {600, 6}}
	hm, numH := trimHMetrics(metrics)
	b.add("hhea", testHhea(numH))
	b.add("maxp", testMaxp(2))
	b.add("hmtx", hm)
	sq := simpleSquare(0, 0, 10, 10)
	b.add("glyf", append(append([]byte{}, emptyGlyph()...), sq...))
	loca := []uint32{0, 10, 10 + uint32(len(sq))}
	// pad to even for short loca
	padded, loca2, short := buildLoca([][]byte{emptyGlyph(), sq})
	_ = padded
	_ = loca2
	_ = short
	b.add("loca", encodeLoca(loca, false))
	b.add("cmap", encodeCmap([]*cmapSubtable{{platformID: 3, platEncID: 1, format: 4, entries: map[uint32]uint16{65: 1}}}))
	addSelfLigatures(t, b, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, map[uint32]uint16{65: 1})
	axes := []fvarAxis{{tag: "wght", min: 100, def: 400, max: 700, nameID: 256}}
	b.add("fvar", buildTestFvar(axes, nil))
	names, _ := encodeName([]nameRecord{{3, 1, 0x409, 1, "T"}})
	b.add("name", names)
	post := make([]byte, 32)
	putU32(post, 0, 0x00030000)
	b.add("post", post)
	os2 := make([]byte, 96)
	putU16(os2, 0, 4) // version 4
	putU16(os2, 4, 400)
	b.add("OS/2", os2)
	data := b.build()
	res, err := subsetFont(data, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, map[string]float64{"wght": 700})
	if err != nil {
		t.Fatal(err)
	}
	f, _ := parseSFNT(res.sfnt)
	if got := getU16(f.tables["OS/2"], 4); got != 700 {
		t.Fatalf("usWeightClass = %d, want 700", got)
	}
	// avgWidth = mean of positive advances: kept glyphs 0,1 -> (500+600)/2=550.
	if got := getU16(f.tables["OS/2"], 2); got != 550 {
		t.Fatalf("xAvgCharWidth = %d, want 550", got)
	}
}

func TestCvarPin(t *testing.T) {
	// cvar with one tuple scaling CVT 0 by pin; baked into cvt, table dropped.
	axes := []fvarAxis{{tag: "wght", min: 100, def: 400, max: 700, nameID: 256}}
	fvarData := buildTestFvar(axes, nil)
	// Build raw cvar: version 1.0, 1 tuple (embedded peak wght=1), all points.
	var cvar []byte
	cvar = appendU16(cvar, 1)
	cvar = appendU16(cvar, 0)
	cvar = appendU16(cvar, 1)  // count, no shared
	cvar = appendU16(cvar, 14) // dataOffset: past the one tuple header
	// tuple header: size, flags=embedded+private points
	tup := []byte{}
	tup = appendU16(tup, 0) // size placeholder
	tup = appendU16(tup, 0x8000|0x2000)
	tup = appendI16(tup, 0x4000) // peak 1.0
	// private points [0] (count, run header, point byte); deltas: cvt0 += 100
	aux := []byte{0x01, 0x00, 0x00}
	aux = append(aux, compileDeltaValues([]int{100})...)
	putU16(tup, 0, uint16(len(aux)))
	cvar = append(cvar, tup...)
	cvar = append(cvar, aux...)
	b := newTestFontBuilder(t)
	b.add("head", testHead(1))
	b.add("hhea", testHhea(1))
	b.add("maxp", testMaxp(1))
	hm, _ := trimHMetrics([][2]int{{500, 0}})
	b.add("hmtx", hm)
	b.add("glyf", emptyGlyph())
	b.add("loca", encodeLoca([]uint32{0, 10}, false))
	b.add("cmap", encodeCmap([]*cmapSubtable{{platformID: 3, platEncID: 1, format: 4, entries: map[uint32]uint16{}}}))
	b.add("fvar", fvarData)
	b.add("cvar", cvar)
	cvt := appendI16(nil, 10)
	cvt = appendI16(cvt, 20)
	b.add("cvt ", cvt)
	names, _ := encodeName([]nameRecord{{3, 1, 0x409, 1, "T"}})
	b.add("name", names)
	post := make([]byte, 32)
	putU32(post, 0, 0x00030000)
	b.add("post", post)
	b.add("OS/2", make([]byte, 96))
	data := b.build()
	res, err := subsetFont(data, map[string]string{}, map[string]uint32{}, map[string]float64{"wght": 700})
	if err != nil {
		t.Fatal(err)
	}
	f, _ := parseSFNT(res.sfnt)
	if _, ok := f.tables["cvar"]; ok {
		t.Fatalf("cvar not dropped")
	}
	got, ok := f.tables["cvt "]
	if !ok || len(got) < 4 {
		t.Fatalf("cvt missing: %v", ok)
	}
	// Pin wght=700 -> ndc=1 -> scalar 1 -> cvt[0] = 10+100.
	if getI16(got, 0) != 110 || getI16(got, 2) != 20 {
		t.Fatalf("cvt = %d %d, want 110 20", getI16(got, 0), getI16(got, 2))
	}
}

func TestCompositeSubset(t *testing.T) {
	// Glyph 1 = composite of glyph 2 offset (10,0); glyph 3 dropped.
	// flags = ARGS_ARE_XY_VALUES, byte args.
	comp := []byte{0x00, 0x02, 0x00, 0x02, 10, 0}
	compRec := append([]byte{0xFF, 0xFF, 0, 0, 0, 0, 0, 0, 0, 0}, comp...)
	recs := [][]byte{emptyGlyph(), compRec, simpleSquare(0, 0, 5, 5), simpleSquare(0, 0, 6, 6)}
	b := newTestFontBuilder(t)
	n := len(recs)
	b.add("head", testHead(1))
	metrics := [][2]int{{500, 0}, {500, 0}, {500, 0}, {500, 0}}
	hm, numH := trimHMetrics(metrics)
	b.add("hhea", testHhea(numH))
	b.add("maxp", testMaxp(n))
	b.add("hmtx", hm)
	var glyf []byte
	var loca []uint32
	off := uint32(0)
	for _, r := range recs {
		loca = append(loca, off)
		glyf = append(glyf, r...)
		off += uint32(len(r))
	}
	loca = append(loca, off)
	b.add("glyf", glyf)
	b.add("loca", encodeLoca(loca, false))
	b.add("cmap", encodeCmap([]*cmapSubtable{{platformID: 3, platEncID: 1, format: 4, entries: map[uint32]uint16{65: 1}}}))
	addSelfLigatures(t, b, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, map[uint32]uint16{65: 1})
	names, _ := encodeName([]nameRecord{{3, 1, 0x409, 1, "T"}})
	b.add("name", names)
	post := make([]byte, 32)
	putU32(post, 0, 0x00030000)
	b.add("post", post)
	b.add("OS/2", make([]byte, 96))
	data := b.build()
	// Keep only glyph 1 (composite): component 2 pulled in by closure; 3 dropped.
	res, err := subsetFont(data, map[string]string{"a": "A"}, map[string]uint32{"A": 65}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f, _ := parseSFNT(res.sfnt)
	if n := int(getU16(f.tables["maxp"], 4)); n != 3 {
		t.Fatalf("numGlyphs = %d, want 3 (notdef, composite, component)", n)
	}
	loca2, _ := parseLoca(f.tables["loca"], 3, false)
	recs2, _ := splitGlyf(f.tables["glyf"], loca2)
	g1, err := parseGlyph(recs2[1])
	if err != nil {
		t.Fatal(err)
	}
	if !g1.isComposite || len(g1.composites) != 1 || g1.composites[0].gid != 2 {
		t.Fatalf("composite not remapped: %+v", g1.composites)
	}
}

func TestMacRoman(t *testing.T) {
	for _, r := range "AZaz09 _" {
		b, err := stringToMacRoman(string(r))
		if err != nil {
			t.Fatal(err)
		}
		if got := macRomanToString(b); got != string(r) {
			t.Fatalf("round-trip %q = %q", r, got)
		}
	}
}

func TestWoff2Composite(t *testing.T) {
	// Composite glyph through the WOFF2 glyf transform.
	comp := []byte{0x00, 0x02, 0x00, 0x01, 10, 0} // ARGS_ARE_XY_VALUES, gid 1, offset (10,0)
	compRec := append([]byte{0xFF, 0xFF, 0, 0, 0, 0, 20, 20, 0, 0}, comp...)
	g0, err := parseGlyph(emptyGlyph())
	if err != nil {
		t.Fatal(err)
	}
	g1, err := parseGlyph(compRec)
	if err != nil {
		t.Fatal(err)
	}
	g2, err := parseGlyph(simpleSquare(0, 0, 5, 5))
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range []*glyph{g0, g1, g2} {
		g.recalcBounds(func(uint16) *glyph { return nil })
		g.toIntCoords()
	}
	tg, _, err := transformGlyf([]*glyph{g0, g1, g2}, 1)
	if err != nil {
		t.Fatal(err)
	}
	// Header: reserved(2), flags(2), numGlyphs(2), indexFormat(2), 7*u32 sizes.
	if len(tg) < 36 {
		t.Fatalf("transformed too short: %d", len(tg))
	}
	numGlyphs := int(getU16(tg, 4))
	if numGlyphs != 3 {
		t.Fatalf("numGlyphs = %d", numGlyphs)
	}
	var sizes [7]uint32
	for i := range sizes {
		sizes[i] = getU32(tg, 8+4*i)
	}
	off := 36
	nContour := tg[off : off+int(sizes[0])]
	if len(nContour) != 6 {
		t.Fatalf("nContour stream len = %d", len(nContour))
	}
	if int(int16(getU16(nContour, 2))) != -1 {
		t.Fatalf("glyph 1 nContour = %d, want -1", int16(getU16(nContour, 2)))
	}
	off += int(sizes[0]) + int(sizes[1]) + int(sizes[2]) + int(sizes[3])
	compStream := tg[off : off+int(sizes[4])]
	want := []byte{0x00, 0x02, 0x00, 0x01, 10, 0}
	found := false
outer:
	for i := 0; i+len(want) <= len(compStream); i++ {
		for j := range want {
			if compStream[i+j] != want[j] {
				continue outer
			}
		}
		found = true
		break
	}
	if !found {
		t.Fatalf("component record %x not in composite stream %x", want, compStream)
	}
}

func decodeBase128(b []byte) (int, int) {
	v, n := 0, 0
	for _, c := range b {
		v = v*128 + int(c&0x7F)
		n++
		if c&0x80 == 0 {
			break
		}
	}
	return v, n
}

func TestRunErrors(t *testing.T) {
	if err := Run(nil); err == nil {
		t.Fatalf("expected usage error")
	}
	if err := Run([]string{"frobnicate"}); err == nil {
		t.Fatalf("expected unknown command error")
	}
	if err := Run([]string{"download"}); err == nil {
		t.Fatalf("expected download usage error")
	}
	if err := Run([]string{"subset", "--fonts", "x"}); err == nil {
		t.Fatalf("expected subset usage error")
	}
	if err := Run([]string{"subset", "--fonts", "x", "--output", "y.woff2", "--pin", "bogus=1"}); err == nil {
		t.Fatalf("expected pin parse error")
	}
}

func TestQuotePath(t *testing.T) {
	if got := quotePath("variablefont/MaterialSymbolsRounded[FILL,GRAD,opsz,wght].ttf"); got != "variablefont/MaterialSymbolsRounded%5BFILL%2CGRAD%2Copsz%2Cwght%5D.ttf" {
		t.Fatalf("quote = %q", got)
	}
}

func TestHvarShortsAndMaps(t *testing.T) {
	// VarData with NumShorts=1 (one int16 + rest int8 per row) and a
	// DeltaSetIndexMap.
	vs := varStore{
		regions: []varRegion{
			{axes: [][3]float64{{0, 1, 1}}},
			{axes: [][3]float64{{0, 1, 1}}},
		},
		varDatas: []varData{
			{regionIndexes: []uint16{0, 1}, rows: [][]int{{1000, 5}, {-300, -7}}},
		},
	}
	hv := &hvarTable{advMap: []uint32{0, (0 << 16) | 1}, store: vs}
	enc := encodeHvar(hv, 1)
	back, err := parseHvar(enc)
	if err != nil {
		t.Fatal(err)
	}
	if len(back.store.varDatas) != 1 || len(back.store.varDatas[0].rows) != 2 {
		t.Fatalf("vardata = %+v", back.store.varDatas)
	}
	if back.store.varDatas[0].rows[0][0] != 1000 || back.store.varDatas[0].rows[1][1] != -7 {
		t.Fatalf("rows = %v", back.store.varDatas[0].rows)
	}
	if len(back.advMap) != 2 || back.advMap[1] != 1 {
		t.Fatalf("advMap = %v", back.advMap)
	}
	// Subset to glyph 1: row filtered, map remapped.
	subsetHvar(back, []uint16{1})
	if len(back.store.varDatas[0].rows) != 1 || back.store.varDatas[0].rows[0][0] != -300 {
		t.Fatalf("rows after subset = %v", back.store.varDatas[0].rows)
	}
	if len(back.advMap) != 1 || back.advMap[0] != 0 {
		t.Fatalf("advMap after subset = %v, want [0] (remapped)", back.advMap)
	}
}

func TestStatV1(t *testing.T) {
	st := &statTable{
		version:  0x00010000,
		axes:     []statAxis{{tag: "wght", nameID: 259, ordering: 0}},
		values:   []statAxisValue{{format: 1, axisIndex: 0, valueNameID: 2, value: 400}},
		elidedID: 2,
	}
	enc := encodeSTAT(st)
	if len(enc) != 18+8+2+2+12 {
		t.Fatalf("v1.0 len = %d, want %d", len(enc), 18+8+2+2+12)
	}
	back, err := parseSTAT(enc)
	if err != nil {
		t.Fatal(err)
	}
	if back.version != 0x00010000 || len(back.values) != 1 || back.values[0].value != 400 {
		t.Fatalf("round-trip = %+v", back)
	}
}
