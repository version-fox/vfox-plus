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

// Supported GSUB lookup types for subsetting. Other lookup types in
// feature-referenced lookups are rejected loudly rather than silently
// corrupted; unreferenced lookups are dropped without parsing.
const (
	gsubSingle    = 1
	gsubLigature  = 4
	gsubExtension = 7
)

type gsubLangSys struct {
	req      int // -1 means 0xFFFF (none)
	features []int
}

type gsubScript struct {
	tag   string
	def   *gsubLangSys
	langs []gsubLangSysEntry
}

type gsubLangSysEntry struct {
	tag string
	ls  gsubLangSys
}

type gsubFeature struct {
	tag       string
	lookups   []int
	hasParams bool
}

type gsubLig struct {
	result     uint16
	components []uint16 // excluding the first (coverage) glyph
}

// gsubSubtable is one parsed lookup subtable. Only SingleSubst,
// LigatureSubst and ExtensionSubst wrapping those are supported.
type gsubSubtable struct {
	ext     bool
	extType uint16
	// unsupported marks a subtable whose type cannot be subsetted. It is
	// dropped when unreferenced; referencing it from a kept feature is an
	// error.
	unsupported bool
	cov         []uint16 // coverage glyphs in file order
	ligs        map[uint16][]gsubLig
	single      map[uint16]uint16
}

type gsubLookup struct {
	lookupType uint16
	flag       uint16
	subs       []gsubSubtable
	// unsupported is set when the lookup type is not subsettable.
	unsupported bool
}

type gsubTable struct {
	scripts   []gsubScript
	features  []gsubFeature
	lookups   []gsubLookup
	fvRecords []gsubFVRecord // empty = GSUB 1.0
}

type gsubCondition struct {
	axisIndex int
	min, max  float64
}

type gsubSubstitution struct {
	featureIndex int
	feature      gsubFeature // alternate feature
}

type gsubFVRecord struct {
	conditions    []gsubCondition
	substitutions []gsubSubstitution
}

func parseFeatureVariations(data []byte, off int) ([]gsubFVRecord, error) {
	if off+8 > len(data) {
		return nil, fmt.Errorf("FeatureVariations out of range")
	}
	if getU32(data, off) != 0x00010000 {
		return nil, fmt.Errorf("unsupported FeatureVariations version %#x", getU32(data, off))
	}
	n := int(getU32(data, off+4))
	var out []gsubFVRecord
	for i := 0; i < n; i++ {
		if off+8+8*i+8 > len(data) {
			return nil, fmt.Errorf("FeatureVariationRecord out of range")
		}
		condOff := off + int(getU32(data, off+8+8*i))
		substOff := off + int(getU32(data, off+8+8*i+4))
		conds, err := parseConditionSet(data, condOff)
		if err != nil {
			return nil, err
		}
		substs, err := parseFeatureTableSubstitution(data, substOff)
		if err != nil {
			return nil, err
		}
		out = append(out, gsubFVRecord{conds, substs})
	}
	return out, nil
}

func parseConditionSet(data []byte, off int) ([]gsubCondition, error) {
	if off+2 > len(data) {
		return nil, fmt.Errorf("ConditionSet out of range")
	}
	n := int(getU16(data, off))
	var out []gsubCondition
	for i := 0; i < n; i++ {
		if off+2+4*i+4 > len(data) {
			return nil, fmt.Errorf("Condition offset out of range")
		}
		co := off + int(getU32(data, off+2+4*i))
		if co+8 > len(data) {
			return nil, fmt.Errorf("Condition out of range")
		}
		if getU16(data, co) != 1 {
			return nil, fmt.Errorf("unsupported Condition format %d", getU16(data, co))
		}
		out = append(out, gsubCondition{
			axisIndex: int(getU16(data, co+2)),
			min:       f2dot14ToFloat(getI16(data, co+4)),
			max:       f2dot14ToFloat(getI16(data, co+6)),
		})
	}
	return out, nil
}

func parseFeatureTableSubstitution(data []byte, off int) ([]gsubSubstitution, error) {
	if off+4 > len(data) {
		return nil, fmt.Errorf("FeatureTableSubstitution out of range")
	}
	if getU32(data, off) != 0x00010000 {
		return nil, fmt.Errorf("unsupported FeatureTableSubstitution version %#x", getU32(data, off))
	}
	n := int(getU16(data, off+4))
	var out []gsubSubstitution
	for i := 0; i < n; i++ {
		if off+6+6*i+6 > len(data) {
			return nil, fmt.Errorf("SubstitutionRecord out of range")
		}
		fi := int(getU16(data, off+6+6*i))
		f, err := parseFeature(data, off+int(getU32(data, off+6+6*i+2)))
		if err != nil {
			return nil, err
		}
		out = append(out, gsubSubstitution{fi, f})
	}
	return out, nil
}

func parseGSUB(data []byte) (*gsubTable, error) {
	if len(data) < 10 {
		return nil, fmt.Errorf("GSUB too short")
	}
	version := getU32(data, 0)
	if version != 0x00010000 && version != 0x00010001 {
		return nil, fmt.Errorf("unsupported GSUB version %#x", version)
	}
	t := &gsubTable{}
	var err error
	if t.scripts, err = parseScriptList(data, int(getU16(data, 4))); err != nil {
		return nil, fmt.Errorf("GSUB ScriptList: %w", err)
	}
	if t.features, err = parseFeatureList(data, int(getU16(data, 6))); err != nil {
		return nil, fmt.Errorf("GSUB FeatureList: %w", err)
	}
	if t.lookups, err = parseLookupList(data, int(getU16(data, 8))); err != nil {
		return nil, fmt.Errorf("GSUB LookupList: %w", err)
	}
	if version == 0x00010001 {
		if len(data) < 14 {
			return nil, fmt.Errorf("GSUB 1.1 too short for FeatureVariations offset")
		}
		if t.fvRecords, err = parseFeatureVariations(data, int(getU32(data, 10))); err != nil {
			return nil, fmt.Errorf("GSUB FeatureVariations: %w", err)
		}
	}
	return t, nil
}

func parseScriptList(data []byte, off int) ([]gsubScript, error) {
	if off+2 > len(data) {
		return nil, fmt.Errorf("ScriptList out of range")
	}
	n := int(getU16(data, off))
	var out []gsubScript
	for i := 0; i < n; i++ {
		if off+2+6*i+6 > len(data) {
			return nil, fmt.Errorf("ScriptRecord out of range")
		}
		tag := string(data[off+2+6*i : off+2+6*i+4])
		recOff := off + int(getU16(data, off+2+6*i+4))
		s, err := parseScript(data, recOff)
		if err != nil {
			return nil, fmt.Errorf("script %q: %w", tag, err)
		}
		s.tag = tag
		out = append(out, s)
	}
	return out, nil
}

func parseScript(data []byte, off int) (gsubScript, error) {
	var s gsubScript
	if off+4 > len(data) {
		return s, fmt.Errorf("Script out of range")
	}
	defOff := int(getU16(data, off))
	n := int(getU16(data, off+2))
	if defOff != 0 {
		ls, err := parseLangSys(data, off+defOff)
		if err != nil {
			return s, err
		}
		s.def = &ls
	}
	for i := 0; i < n; i++ {
		if off+4+6*i+6 > len(data) {
			return s, fmt.Errorf("LangSysRecord out of range")
		}
		tag := string(data[off+4+6*i : off+4+6*i+4])
		ls, err := parseLangSys(data, off+int(getU16(data, off+4+6*i+4)))
		if err != nil {
			return s, err
		}
		s.langs = append(s.langs, gsubLangSysEntry{tag, ls})
	}
	return s, nil
}

func parseLangSys(data []byte, off int) (gsubLangSys, error) {
	var ls gsubLangSys
	if off+6 > len(data) {
		return ls, fmt.Errorf("LangSys out of range")
	}
	// lookupOrder offset must be NULL for our purposes; skip it.
	req := int(getU16(data, off+2))
	if req == 0xFFFF {
		ls.req = -1
	} else {
		ls.req = req
	}
	n := int(getU16(data, off+4))
	if off+6+2*n > len(data) {
		return ls, fmt.Errorf("LangSys features out of range")
	}
	for i := 0; i < n; i++ {
		ls.features = append(ls.features, int(getU16(data, off+6+2*i)))
	}
	return ls, nil
}

func parseFeatureList(data []byte, off int) ([]gsubFeature, error) {
	if off+2 > len(data) {
		return nil, fmt.Errorf("FeatureList out of range")
	}
	n := int(getU16(data, off))
	var out []gsubFeature
	for i := 0; i < n; i++ {
		if off+2+6*i+6 > len(data) {
			return nil, fmt.Errorf("FeatureRecord out of range")
		}
		tag := string(data[off+2+6*i : off+2+6*i+4])
		f, err := parseFeature(data, off+int(getU16(data, off+2+6*i+4)))
		if err != nil {
			return nil, fmt.Errorf("feature %q: %w", tag, err)
		}
		f.tag = tag
		out = append(out, f)
	}
	return out, nil
}

func parseFeature(data []byte, off int) (gsubFeature, error) {
	var f gsubFeature
	if off+4 > len(data) {
		return f, fmt.Errorf("Feature out of range")
	}
	f.hasParams = getU16(data, off) != 0
	n := int(getU16(data, off+2))
	if off+4+2*n > len(data) {
		return f, fmt.Errorf("Feature lookups out of range")
	}
	for i := 0; i < n; i++ {
		f.lookups = append(f.lookups, int(getU16(data, off+4+2*i)))
	}
	return f, nil
}

func parseLookupList(data []byte, off int) ([]gsubLookup, error) {
	if off+2 > len(data) {
		return nil, fmt.Errorf("LookupList out of range")
	}
	n := int(getU16(data, off))
	var out []gsubLookup
	for i := 0; i < n; i++ {
		if off+2+2*i+2 > len(data) {
			return nil, fmt.Errorf("Lookup offset out of range")
		}
		l, err := parseLookup(data, off+int(getU16(data, off+2+2*i)))
		if err != nil {
			return nil, fmt.Errorf("lookup %d: %w", i, err)
		}
		out = append(out, l)
	}
	return out, nil
}

func parseLookup(data []byte, off int) (gsubLookup, error) {
	var l gsubLookup
	if off+6 > len(data) {
		return l, fmt.Errorf("Lookup header out of range")
	}
	l.lookupType = getU16(data, off)
	l.flag = getU16(data, off+2)
	n := int(getU16(data, off+4))
	if l.flag&0x10 != 0 {
		return l, fmt.Errorf("MarkFilteringSet lookups not supported (lookup type %d)", l.lookupType)
	}
	switch l.lookupType {
	case gsubLigature, gsubSingle, gsubExtension:
		// Parseable; subtables parsed below.
	default:
		// Unsupported type: leave subtables unparsed. pruneGSUB drops
		// unreferenced lookups without parsing and errors on referenced
		// ones, instead of silently corrupting them.
		l.unsupported = true
		return l, nil
	}
	for i := 0; i < n; i++ {
		if off+6+2*i+2 > len(data) {
			return l, fmt.Errorf("Lookup subtable offset out of range")
		}
		st, err := parseSubtable(data, off+int(getU16(data, off+6+2*i)), l.lookupType)
		if err != nil {
			return l, err
		}
		l.subs = append(l.subs, st)
	}
	return l, nil
}

// parseSubtable parses one subtable of the given lookup type. Unsupported
// types are recorded, not parsed; the caller errors if such a lookup is
// referenced by a kept feature.
func parseSubtable(data []byte, off int, lookupType uint16) (gsubSubtable, error) {
	var st gsubSubtable
	switch lookupType {
	case gsubLigature:
		return parseLigature(data, off)
	case gsubSingle:
		return parseSingle(data, off)
	case gsubExtension:
		if off+8 > len(data) {
			return st, fmt.Errorf("ExtensionSubst out of range")
		}
		if getU16(data, off) != 1 {
			return st, fmt.Errorf("unsupported ExtensionSubst format %d", getU16(data, off))
		}
		extType := getU16(data, off+2)
		innerOff := off + int(getU32(data, off+4))
		switch extType {
		case gsubLigature, gsubSingle:
			// Parseable; errors surface if a kept feature references it.
		default:
			return gsubSubtable{ext: true, extType: extType, unsupported: true}, nil
		}
		inner, err := parseSubtable(data, innerOff, extType)
		if err != nil {
			return st, err
		}
		inner.ext = true
		inner.extType = extType
		return inner, nil
	default:
		return st, fmt.Errorf("unsupported GSUB lookup type %d", lookupType)
	}
}

func parseCoverage(data []byte, off int) ([]uint16, error) {
	if off+4 > len(data) {
		return nil, fmt.Errorf("Coverage out of range")
	}
	format := getU16(data, off)
	switch format {
	case 1:
		n := int(getU16(data, off+2))
		if off+4+2*n > len(data) {
			return nil, fmt.Errorf("Coverage format 1 out of range")
		}
		out := make([]uint16, n)
		for i := range out {
			out[i] = getU16(data, off+4+2*i)
		}
		return out, nil
	case 2:
		n := int(getU16(data, off+2))
		if off+4+6*n > len(data) {
			return nil, fmt.Errorf("Coverage format 2 out of range")
		}
		var out []uint16
		for i := 0; i < n; i++ {
			start, end := getU16(data, off+4+6*i), getU16(data, off+4+6*i+2)
			for g := start; ; g++ {
				out = append(out, g)
				if g == end {
					break
				}
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unsupported Coverage format %d", format)
	}
}

func parseLigature(data []byte, off int) (gsubSubtable, error) {
	var st gsubSubtable
	if off+6 > len(data) {
		return st, fmt.Errorf("LigatureSubst out of range")
	}
	if getU16(data, off) != 1 {
		return st, fmt.Errorf("unsupported LigatureSubst format %d", getU16(data, off))
	}
	n := int(getU16(data, off+4))
	if off+6+2*n > len(data) {
		return st, fmt.Errorf("LigatureSubst sets out of range")
	}
	cov, err := parseCoverage(data, off+int(getU16(data, off+2)))
	if err != nil {
		return st, err
	}
	st.cov = cov
	st.ligs = map[uint16][]gsubLig{}
	for i := 0; i < n; i++ {
		setOff := off + int(getU16(data, off+6+2*i))
		if setOff+2 > len(data) {
			return st, fmt.Errorf("LigatureSet out of range")
		}
		m := int(getU16(data, setOff))
		if setOff+2+2*m > len(data) {
			return st, fmt.Errorf("LigatureSet entries out of range")
		}
		for j := 0; j < m; j++ {
			ligOff := setOff + int(getU16(data, setOff+2+2*j))
			if ligOff+4 > len(data) {
				return st, fmt.Errorf("Ligature out of range")
			}
			result := getU16(data, ligOff)
			cc := int(getU16(data, ligOff+2))
			if ligOff+4+2*(cc-1) > len(data) || cc < 1 {
				return st, fmt.Errorf("Ligature components out of range")
			}
			lig := gsubLig{result: result}
			for k := 0; k < cc-1; k++ {
				lig.components = append(lig.components, getU16(data, ligOff+4+2*k))
			}
			// Attribute to coverage glyph cyclically like fontTools does
			// implicitly: the i-th set belongs to cov[i] when present.
			first := uint16(0)
			if i < len(cov) {
				first = cov[i]
			}
			st.ligs[first] = append(st.ligs[first], lig)
		}
	}
	return st, nil
}

func parseSingle(data []byte, off int) (gsubSubtable, error) {
	var st gsubSubtable
	if off+6 > len(data) {
		return st, fmt.Errorf("SingleSubst out of range")
	}
	format := getU16(data, off)
	st.single = map[uint16]uint16{}
	switch format {
	case 1:
		cov, err := parseCoverage(data, off+int(getU16(data, off+2)))
		if err != nil {
			return st, err
		}
		delta := int16(getU16(data, off+4))
		st.cov = cov
		for _, g := range cov {
			st.single[g] = uint16((int32(g) + int32(delta)) & 0xFFFF)
		}
	case 2:
		cov, err := parseCoverage(data, off+int(getU16(data, off+2)))
		if err != nil {
			return st, err
		}
		n := int(getU16(data, off+4))
		if off+6+2*n > len(data) {
			return st, fmt.Errorf("SingleSubst format 2 out of range")
		}
		st.cov = cov
		for i, g := range cov {
			if i < n {
				st.single[g] = getU16(data, off+6+2*i)
			}
		}
	default:
		return st, fmt.Errorf("unsupported SingleSubst format %d", format)
	}
	return st, nil
}

// pruneGSUB applies fontTools subsetter GSUB semantics with
// layout_features=["*"] and layout_closure=false:
//   - each feature-referenced lookup keeps only subtables whose ligature
//     (or single-subst) entries are fully covered by kept glyphs;
//   - lookups left empty are dropped, features left empty (unless "pref" or
//     sized params) are dropped, ScriptList entries are kept (GSUB retains
//     empty scripts) with remapped feature indices.
//
// keep maps old glyph IDs to true. It returns the pruned table with lookups
// in original order and feature indices remapped.
func pruneGSUB(t *gsubTable, keep map[uint16]bool) (*gsubTable, error) {
	// Collect lookups referenced by any feature.
	ref := map[int]bool{}
	for _, f := range t.features {
		for _, l := range f.lookups {
			ref[l] = true
		}
	}
	keptLookup := make([]bool, len(t.lookups))
	for i, l := range t.lookups {
		if !ref[i] {
			continue
		}
		if l.unsupported {
			return nil, fmt.Errorf("unsupported GSUB lookup type %d referenced by a kept feature", l.lookupType)
		}
		pruned, err := pruneLookup(l, keep)
		if err != nil {
			return nil, err
		}
		if len(pruned.subs) > 0 {
			keptLookup[i] = true
			t.lookups[i] = pruned
		}
	}
	// Filter features to kept lookups; drop empties (keep "pref" and
	// parameterized features like fontTools does for "size").
	remap := make([]int, len(t.lookups))
	newIdx := 0
	for i, k := range keptLookup {
		remap[i] = -1
		if k {
			remap[i] = newIdx
			newIdx++
		}
	}
	// FeatureVariations substitution pruning: filter alternate features to
	// kept lookups (remapped), collecting referenced feature indices.
	var fvKept []int
	for ri := range t.fvRecords {
		var substs []gsubSubstitution
		for _, s := range t.fvRecords[ri].substitutions {
			var ls []int
			for _, l := range s.feature.lookups {
				if l < len(remap) && remap[l] >= 0 {
					ls = append(ls, remap[l])
				}
			}
			if len(ls) > 0 {
				s.feature.lookups = ls
				substs = append(substs, s)
				fvKept = append(fvKept, s.featureIndex)
			}
		}
		t.fvRecords[ri].substitutions = substs
	}
	mainKept := map[int]bool{}
	for i, f := range t.features {
		var ls []int
		for _, l := range f.lookups {
			if l < len(remap) && remap[l] >= 0 {
				ls = append(ls, remap[l])
			}
		}
		if len(ls) > 0 || f.tag == "pref" || f.hasParams {
			f.lookups = ls
			mainKept[i] = true
		}
	}
	keptSet := map[int]bool{}
	for i := range mainKept {
		keptSet[i] = true
	}
	for _, i := range fvKept {
		keptSet[i] = true
	}
	var keptFeatureIndices []int
	for i := range t.features {
		if keptSet[i] {
			keptFeatureIndices = append(keptFeatureIndices, i)
		}
	}
	var features []gsubFeature
	for i, f := range t.features {
		if mainKept[i] {
			features = append(features, f)
		}
	}
	// FV.subset_features: drop substitution records whose feature was
	// dropped, remap the rest, then pop trailing empty records (mirroring
	// fontTools, which only prunes empties at the end).
	var fvRecords []gsubFVRecord
	for _, r := range t.fvRecords {
		var substs []gsubSubstitution
		for _, s := range r.substitutions {
			idx := -1
			for k, fi := range keptFeatureIndices {
				if fi == s.featureIndex {
					idx = k
					break
				}
			}
			if idx >= 0 {
				s.featureIndex = idx
				substs = append(substs, s)
			}
		}
		r.substitutions = substs
		fvRecords = append(fvRecords, r)
	}
	for len(fvRecords) > 0 && len(fvRecords[len(fvRecords)-1].substitutions) == 0 {
		fvRecords = fvRecords[:len(fvRecords)-1]
	}
	// Remap ScriptList.
	featKept := make([]bool, len(t.features))
	for _, i := range keptFeatureIndices {
		featKept[i] = true
	}
	scripts := make([]gsubScript, len(t.scripts))
	for i, s := range t.scripts {
		scripts[i].tag = s.tag
		if s.def != nil {
			d := remapLangSys(*s.def, featKept)
			scripts[i].def = &d
		}
		for _, e := range s.langs {
			scripts[i].langs = append(scripts[i].langs, gsubLangSysEntry{e.tag, remapLangSys(e.ls, featKept)})
		}
	}
	var lookups []gsubLookup
	for i, k := range keptLookup {
		if k {
			lookups = append(lookups, t.lookups[i])
		}
	}
	return &gsubTable{scripts: scripts, features: features, lookups: lookups, fvRecords: fvRecords}, nil
}

// remapGSUB rewrites all glyph ID references from old to new IDs. It must
// run after pruning (which works in old ID space); afterwards the table is
// in new ID space for verification and encoding.
func remapGSUB(t *gsubTable, newIndex map[uint16]uint16) {
	remap := func(g uint16) uint16 {
		if n, ok := newIndex[g]; ok {
			return n
		}
		return g
	}
	for li := range t.lookups {
		for si := range t.lookups[li].subs {
			st := &t.lookups[li].subs[si]
			for i, g := range st.cov {
				st.cov[i] = remap(g)
			}
			fixed := map[uint16][]gsubLig{}
			for first, ligs := range st.ligs {
				nf := remap(first)
				for _, lig := range ligs {
					lig.result = remap(lig.result)
					for i, c := range lig.components {
						lig.components[i] = remap(c)
					}
					fixed[nf] = append(fixed[nf], lig)
				}
			}
			st.ligs = fixed
			for g, v := range st.single {
				delete(st.single, g)
				st.single[remap(g)] = remap(v)
			}
		}
	}
}

func remapLangSys(ls gsubLangSys, featKept []bool) gsubLangSys {
	out := gsubLangSys{req: -1}
	// Build old->new feature index map.
	newIdx := map[int]int{}
	n := 0
	for i, k := range featKept {
		if k {
			newIdx[i] = n
			n++
		}
	}
	if ls.req >= 0 {
		if v, ok := newIdx[ls.req]; ok {
			out.req = v
		}
	}
	for _, f := range ls.features {
		if v, ok := newIdx[f]; ok {
			out.features = append(out.features, v)
		}
	}
	return out
}

// pruneLookup drops subtable entries referencing dropped glyphs, mirroring
// fontTools LigatureSubst/SingleSubst subset_glyphs with layout_closure=false.
func pruneLookup(l gsubLookup, keep map[uint16]bool) (gsubLookup, error) {
	out := gsubLookup{lookupType: l.lookupType, flag: l.flag}
	for _, st := range l.subs {
		if st.unsupported {
			return out, fmt.Errorf("unsupported GSUB subtable (lookup type %d, extension %d) referenced by a kept feature", l.lookupType, st.extType)
		}
		if st.ext && st.extType != gsubLigature && st.extType != gsubSingle {
			return out, fmt.Errorf("unsupported GSUB extension inner type %d", st.extType)
		}
		innerType := l.lookupType
		if st.ext {
			innerType = st.extType
		}
		switch innerType {
		case gsubLigature:
			var cov []uint16
			ligs := map[uint16][]gsubLig{}
			for _, first := range st.cov {
				if !keep[first] {
					continue
				}
				for _, lig := range st.ligs[first] {
					if !keep[lig.result] {
						continue
					}
					ok := true
					for _, c := range lig.components {
						if !keep[c] {
							ok = false
							break
						}
					}
					if ok {
						ligs[first] = append(ligs[first], lig)
					}
				}
				if len(ligs[first]) > 0 {
					cov = append(cov, first)
				}
			}
			if len(cov) > 0 {
				st.cov = cov
				st.ligs = ligs
				out.subs = append(out.subs, st)
			}
		case gsubSingle:
			var cov []uint16
			m := map[uint16]uint16{}
			for _, g := range st.cov {
				v, ok := st.single[g]
				if !ok || !keep[g] || !keep[v] {
					continue
				}
				m[g] = v
				cov = append(cov, g)
			}
			if len(cov) > 0 {
				st.cov = cov
				st.single = m
				out.subs = append(out.subs, st)
			}
		default:
			return out, fmt.Errorf("unsupported GSUB lookup type %d", l.lookupType)
		}
	}
	return out, nil
}

// ligatureNames returns the set of component-character strings that the
// table's ligature lookups produce, mirroring icon_font.ligature_names.
func (t *gsubTable) ligatureNames(reverse map[uint16]uint32) map[string]bool {
	names := map[string]bool{}
	for _, l := range t.lookups {
		for _, st := range l.subs {
			inner := l.lookupType
			if st.ext {
				inner = st.extType
			}
			if inner != gsubLigature {
				continue
			}
			for first, ligs := range st.ligs {
				for _, lig := range ligs {
					s := string(rune(reverse[first]))
					ok := true
					for _, c := range lig.components {
						r, found := reverse[c]
						if !found {
							ok = false
							break
						}
						s += string(rune(r))
					}
					if ok {
						names[s] = true
					}
				}
			}
		}
	}
	return names
}

// compileCoverage builds a Coverage table, picking the smaller of formats 1/2.
func compileCoverage(glyphs []uint16) []byte {
	if len(glyphs) == 0 {
		return []byte{0, 1, 0, 0}
	}
	sorted := append([]uint16(nil), glyphs...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })
	// Count ranges for format 2.
	ranges := 1
	for i := 1; i < len(sorted); i++ {
		if sorted[i] != sorted[i-1]+1 {
			ranges++
		}
	}
	if 4+6*ranges < 4+2*len(sorted) {
		out := []byte{0, 2}
		out = appendU16(out, uint16(ranges))
		for i := 0; i < len(sorted); {
			j := i
			for j+1 < len(sorted) && sorted[j+1] == sorted[j]+1 {
				j++
			}
			out = appendU16(out, sorted[i])
			out = appendU16(out, sorted[j])
			out = appendU16(out, uint16(i))
			i = j + 1
		}
		return out
	}
	out := []byte{0, 1}
	out = appendU16(out, uint16(len(sorted)))
	for _, g := range sorted {
		out = appendU16(out, g)
	}
	return out
}

// encodeGSUB serializes a pruned GSUB table with new glyph IDs. Version 1.1
// with FeatureVariations is written only when records survive.
//
// Known deltas vs fontTools: coverage tables use whichever of format 1/2 is
// smaller (fontTools uses its own heuristic); identical scripts share one
// blob like the fontTools writer; FeatureParams are not preserved (written
// as NULL), so fonts relying on parameterized features (e.g. "size") lose
// those parameters — the Material Symbols source has none.

func encodeGSUB(t *gsubTable, gid func(old uint16) uint16) ([]byte, error) {
	// Pre-encode lookups (need lengths for offsets).
	type blob struct {
		typ  uint16
		flag uint16
		subs [][]byte
	}
	blobs := make([]blob, len(t.lookups))
	for i, l := range t.lookups {
		b := blob{typ: l.lookupType, flag: l.flag}
		for _, st := range l.subs {
			var body []byte
			var err error
			inner := l.lookupType
			if st.ext {
				inner = st.extType
			}
			switch inner {
			case gsubLigature:
				body, err = encodeLigature(st, gid)
			case gsubSingle:
				body, err = encodeSingle(st, gid)
			default:
				err = fmt.Errorf("unsupported GSUB lookup type %d", l.lookupType)
			}
			if err != nil {
				return nil, err
			}
			if st.ext {
				wrap := []byte{0, 1}
				wrap = appendU16(wrap, st.extType)
				// Extension offset from start of the ExtensionSubst subtable.
				wrap = appendU32(wrap, 8)
				body = append(wrap, body...)
			}
			b.subs = append(b.subs, body)
		}
		blobs[i] = b
	}

	hasFV := len(t.fvRecords) > 0
	out := []byte{0, 1, 0, 0} // version
	if hasFV {
		out[3] = 1
	}
	out = appendU16(out, 10) // scriptList offset (placeholder)
	out = appendU16(out, 0)  // featureList offset
	out = appendU16(out, 0)  // lookupList offset
	fvOffPos := -1
	if hasFV {
		fvOffPos = len(out)
		out = appendU32(out, 0) // FeatureVariations offset placeholder
	}

	// ScriptList. Identical scripts share one blob, mirroring the
	// fontTools writer's subtable sharing.
	scriptOff := len(out)
	putU16(out, 4, uint16(scriptOff))
	out = appendU16(out, uint16(len(t.scripts)))
	scriptRecPos := len(out)
	for _, s := range t.scripts {
		out = append(out, s.tag[0], s.tag[1], s.tag[2], s.tag[3], 0, 0)
	}
	scriptBlobs := map[string]int{}
	encodeScript := func(s gsubScript) []byte {
		var body []byte
		body = appendU16(body, 0) // defaultLangSys offset placeholder
		body = appendU16(body, uint16(len(s.langs)))
		langPos := len(body)
		for _, e := range s.langs {
			body = append(body, e.tag[0], e.tag[1], e.tag[2], e.tag[3], 0, 0)
		}
		if s.def != nil {
			putU16(body, 0, uint16(len(body)))
			body = append(body, encodeLangSys(*s.def)...)
		}
		for j, e := range s.langs {
			putU16(body, langPos+j*6+4, uint16(len(body)))
			body = append(body, encodeLangSys(e.ls)...)
		}
		return body
	}
	for i, s := range t.scripts {
		body := encodeScript(s)
		if prev, ok := scriptBlobs[string(body)]; ok {
			putU16(out, scriptRecPos+i*6+4, uint16(prev-scriptOff))
			continue
		}
		putU16(out, scriptRecPos+i*6+4, uint16(len(out)-scriptOff))
		scriptBlobs[string(body)] = len(out)
		out = append(out, body...)
	}

	// FeatureList.
	featOff := len(out)
	putU16(out, 6, uint16(featOff))
	out = appendU16(out, uint16(len(t.features)))
	featRecPos := len(out)
	for _, f := range t.features {
		out = append(out, f.tag[0], f.tag[1], f.tag[2], f.tag[3], 0, 0)
	}
	for i, f := range t.features {
		putU16(out, featRecPos+i*6+4, uint16(len(out)-featOff))
		out = appendU16(out, 0) // featureParams NULL
		out = appendU16(out, uint16(len(f.lookups)))
		for _, l := range f.lookups {
			out = appendU16(out, uint16(l))
		}
	}

	// LookupList.
	lookOff := len(out)
	putU16(out, 8, uint16(lookOff))
	out = appendU16(out, uint16(len(blobs)))
	lookRecPos := len(out)
	for range blobs {
		out = appendU16(out, 0)
	}
	for i, b := range blobs {
		putU16(out, lookRecPos+i*2, uint16(len(out)-lookOff))
		base := len(out)
		out = appendU16(out, b.typ)
		out = appendU16(out, b.flag)
		out = appendU16(out, uint16(len(b.subs)))
		subPos := len(out)
		for range b.subs {
			out = appendU16(out, 0)
		}
		for j, body := range b.subs {
			putU16(out, subPos+j*2, uint16(len(out)-base))
			out = append(out, body...)
		}
	}
	if hasFV {
		putU32(out, fvOffPos, uint32(len(out)))
		out = append(out, encodeFVRecords(t.fvRecords)...)
	}
	return out, nil
}

func encodeFVRecords(records []gsubFVRecord) []byte {
	out := appendU32(nil, 0x00010000)
	out = appendU32(out, uint32(len(records)))
	recPos := len(out)
	for range records {
		out = appendU32(out, 0) // conditionSet offset
		out = appendU32(out, 0) // substitution offset
	}
	for i, r := range records {
		base := len(out)
		// ConditionSet.
		putU32(out, recPos+i*8, uint32(len(out)-base))
		csBase := len(out)
		out = appendU16(out, uint16(len(r.conditions)))
		coPos := len(out)
		for range r.conditions {
			out = appendU32(out, 0)
		}
		for j, c := range r.conditions {
			putU32(out, coPos+j*4, uint32(len(out)-csBase))
			out = appendU16(out, 1)
			out = appendU16(out, uint16(c.axisIndex))
			out = appendI16(out, floatToF2Dot14(c.min))
			out = appendI16(out, floatToF2Dot14(c.max))
		}
		// FeatureTableSubstitution.
		putU32(out, recPos+i*8+4, uint32(len(out)-base))
		ftsBase := len(out)
		out = appendU32(out, 0x00010000)
		out = appendU16(out, uint16(len(r.substitutions)))
		srPos := len(out)
		for range r.substitutions {
			out = appendU16(out, 0)
			out = appendU32(out, 0)
		}
		for j, sub := range r.substitutions {
			putU16(out, srPos+j*6, uint16(sub.featureIndex))
			putU32(out, srPos+j*6+2, uint32(len(out)-ftsBase))
			out = appendU16(out, 0) // featureParams NULL
			out = appendU16(out, uint16(len(sub.feature.lookups)))
			for _, l := range sub.feature.lookups {
				out = appendU16(out, uint16(l))
			}
		}
	}
	return out
}

func encodeLangSys(ls gsubLangSys) []byte {
	out := []byte{0, 0} // lookupOrder NULL
	if ls.req < 0 {
		out = appendU16(out, 0xFFFF)
	} else {
		out = appendU16(out, uint16(ls.req))
	}
	out = appendU16(out, uint16(len(ls.features)))
	for _, f := range ls.features {
		out = appendU16(out, uint16(f))
	}
	return out
}

func encodeLigature(st gsubSubtable, gid func(old uint16) uint16) ([]byte, error) {
	// Coverage glyphs and LigatureSets must use the same order; both are
	// emitted sorted by new glyph ID for determinism.
	type set struct {
		first uint16
		ligs  []gsubLig
	}
	var sets []set
	for _, first := range st.cov {
		if len(st.ligs[first]) > 0 {
			sets = append(sets, set{gid(first), st.ligs[first]})
		}
	}
	sort.Slice(sets, func(i, j int) bool { return sets[i].first < sets[j].first })
	cov := make([]uint16, len(sets))
	for i, s := range sets {
		cov[i] = s.first
	}
	covBytes := compileCoverage(cov)
	// LigatureSets in coverage order.
	body := []byte{0, 1}
	body = appendU16(body, 0) // coverage offset placeholder
	body = appendU16(body, uint16(len(cov)))
	setPos := len(body)
	for range cov {
		body = appendU16(body, 0)
	}
	covOff := len(body)
	body = append(body, covBytes...)
	putU16(body, 2, uint16(covOff))
	for i, s := range sets {
		putU16(body, setPos+i*2, uint16(len(body)))
		setBase := len(body)
		ligs := s.ligs
		body = appendU16(body, uint16(len(ligs)))
		ligPos := len(body)
		for range ligs {
			body = appendU16(body, 0)
		}
		for j, lig := range ligs {
			putU16(body, ligPos+j*2, uint16(len(body)-setBase))
			body = appendU16(body, gid(lig.result))
			body = appendU16(body, uint16(len(lig.components)+1))
			for _, c := range lig.components {
				body = appendU16(body, gid(c))
			}
		}
	}
	// Offsets in LigatureSubst are from the subtable start; body[0] is it.
	return body, nil
}

func encodeSingle(st gsubSubtable, gid func(old uint16) uint16) ([]byte, error) {
	// Emit format 1 when the mapping is a uniform delta in new ID space,
	// else format 2. Both are semantically identical.
	cov := make([]uint16, len(st.cov))
	subs := make([]uint16, len(st.cov))
	for i, g := range st.cov {
		cov[i] = gid(g)
		subs[i] = gid(st.single[g])
	}
	// Coverage and substitutes share one order; sort by new glyph ID.
	ord := make([]int, len(cov))
	for i := range ord {
		ord[i] = i
	}
	sort.Slice(ord, func(i, j int) bool { return cov[ord[i]] < cov[ord[j]] })
	scov := make([]uint16, len(cov))
	ssub := make([]uint16, len(cov))
	for i, o := range ord {
		scov[i], ssub[i] = cov[o], subs[o]
	}
	cov, subs = scov, ssub
	uniform := len(cov) > 0
	var delta int
	for i := range cov {
		d := int(subs[i]) - int(cov[i])
		if i == 0 {
			delta = d
		} else if d != delta {
			uniform = false
			break
		}
	}
	if uniform && delta >= -32768 && delta <= 32767 {
		body := []byte{0, 1}
		body = appendU16(body, 6) // coverage right after header
		body = appendU16(body, uint16(int16(delta)))
		body = append(body, compileCoverage(cov)...)
		return body, nil
	}
	body := []byte{0, 2}
	body = appendU16(body, uint16(6+2+2*len(cov)))
	body = appendU16(body, uint16(len(cov)))
	for _, s := range subs {
		body = appendU16(body, s)
	}
	body = append(body, compileCoverage(cov)...)
	return body, nil
}
