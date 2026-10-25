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

// variableAxes are the only axes this tool knows, mirroring the Python
// tool's parse_pin allowlist.
var variableAxes = []string{"FILL", "GRAD", "opsz", "wght"}

func isVariableAxis(tag string) bool {
	for _, a := range variableAxes {
		if a == tag {
			return true
		}
	}
	return false
}

// normalizeValue maps a user-space value to normalized -1..+1, clamping to
// the axis range. Mirrors fontTools.varLib.models.normalizeValue with
// extrapolate=False on a standard (increasing) triple.
func normalizeValue(v, lower, def, upper float64) float64 {
	if v < lower {
		v = lower
	}
	if v > upper {
		v = upper
	}
	if v == def || lower == upper {
		return 0
	}
	if v < def {
		return (v - def) / (def - lower)
	}
	return (v - def) / (upper - def)
}

// piecewiseLinearMap interpolates within a sorted from->to mapping, mirroring
// fontTools.varLib.models.piecewiseLinearMap. Callers clamp v into the key
// range (normalizeValue already yields [-1, 1] and avar maps span it).
func piecewiseLinearMap(v float64, mapping map[float64]float64) float64 {
	if len(mapping) == 0 {
		return v
	}
	if w, ok := mapping[v]; ok {
		return w
	}
	keys := make([]float64, 0, len(mapping))
	for k := range mapping {
		keys = append(keys, k)
	}
	sort.Float64s(keys)
	if v < keys[0] {
		k := keys[0]
		return v + mapping[k] - k
	}
	last := keys[len(keys)-1]
	if v > last {
		return v + mapping[last] - last
	}
	var a, b float64
	for i, k := range keys {
		if k < v {
			a = k
			_ = i
		}
		if k > v {
			b = k
			break
		}
	}
	va, vb := mapping[a], mapping[b]
	return va + (vb-va)*(v-a)/(b-a)
}

// supportScalar computes the single-axis interpolation scalar, mirroring
// fontTools.varLib.models.supportScalar for one axis with ot=True.
func supportScalar(v, lower, peak, upper float64) float64 {
	if peak == 0 {
		return 1
	}
	if v == peak {
		return 1
	}
	if v <= lower || upper <= v {
		return 0
	}
	if v < peak {
		return (v - lower) / (peak - lower)
	}
	return (v - upper) / (peak - upper)
}

// wellFormedTent reports whether a tuple support triple is usable, mirroring
// the drop checks in changeTupleVariationAxisLimit.
func wellFormedTent(lower, peak, upper float64) bool {
	if peak == 0 {
		return true
	}
	if lower > peak || peak > upper {
		return false
	}
	if lower < 0 && upper > 0 {
		return false
	}
	return true
}

// fvarAxis is one fvar axis record with user-space values.
type fvarAxis struct {
	tag      string
	min, def float64
	max      float64
	flags    uint16
	nameID   uint16
}

type fvarInstance struct {
	subfamilyID uint16
	flags       uint16
	coords      map[string]float64 // axis tag -> user value
}

type fvarTable struct {
	version   uint32
	axes      []fvarAxis
	instances []fvarInstance
}

func parseFvar(data []byte) (*fvarTable, error) {
	if len(data) < 16 {
		return nil, fmt.Errorf("fvar too short")
	}
	t := &fvarTable{version: getU32(data, 0)}
	axisOff := int(getU16(data, 4))
	axisCount := int(getU16(data, 8))
	axisSize := int(getU16(data, 10))
	instCount := int(getU16(data, 12))
	instSize := int(getU16(data, 14))
	if axisSize < 20 {
		return nil, fmt.Errorf("bad fvar axis size %d", axisSize)
	}
	for i := 0; i < axisCount; i++ {
		off := axisOff + i*axisSize
		if off+20 > len(data) {
			return nil, fmt.Errorf("fvar axis %d out of range", i)
		}
		t.axes = append(t.axes, fvarAxis{
			tag:    string(data[off : off+4]),
			min:    fixedToFloat(getI32(data, off+4)),
			def:    fixedToFloat(getI32(data, off+8)),
			max:    fixedToFloat(getI32(data, off+12)),
			flags:  getU16(data, off+16),
			nameID: getU16(data, off+18),
		})
	}
	tags := make([]string, len(t.axes))
	for i, a := range t.axes {
		tags[i] = a.tag
	}
	instOff := axisOff + axisCount*axisSize
	for i := 0; i < instCount; i++ {
		off := instOff + i*instSize
		if off+4+4*len(tags) > len(data) {
			return nil, fmt.Errorf("fvar instance %d out of range", i)
		}
		inst := fvarInstance{
			subfamilyID: getU16(data, off),
			flags:       getU16(data, off+2),
			coords:      map[string]float64{},
		}
		for j, tag := range tags {
			inst.coords[tag] = fixedToFloat(getI32(data, off+4+4*j))
		}
		t.instances = append(t.instances, inst)
	}
	return t, nil
}

func encodeFvar(t *fvarTable) []byte {
	axisCount := len(t.axes)
	instSize := 4 + 4*axisCount
	out := make([]byte, 0, 16+20*axisCount+instSize*len(t.instances))
	out = appendU32(out, t.version)
	out = appendU16(out, 16) // offsetToAxesArray
	out = appendU16(out, 2)  // countSizePairs
	out = appendU16(out, uint16(axisCount))
	out = appendU16(out, 20) // axisSize
	out = appendU16(out, uint16(len(t.instances)))
	out = appendU16(out, uint16(instSize))
	for _, a := range t.axes {
		out = append(out, a.tag[0], a.tag[1], a.tag[2], a.tag[3])
		out = appendU32(out, uint32(floatToFixed(a.min)))
		out = appendU32(out, uint32(floatToFixed(a.def)))
		out = appendU32(out, uint32(floatToFixed(a.max)))
		out = appendU16(out, a.flags)
		out = appendU16(out, a.nameID)
	}
	for _, in := range t.instances {
		out = appendU16(out, in.subfamilyID)
		out = appendU16(out, in.flags)
		for _, a := range t.axes {
			out = appendU32(out, uint32(floatToFixed(in.coords[a.tag])))
		}
	}
	return out
}

// avarTable is an axis-variations table with per-axis segment maps in
// normalized coordinates (axis order follows fvar).
type avarTable struct {
	version  uint32
	segments []avarSegments // one per fvar axis, in order
}

type avarSegments map[float64]float64

func parseAvar(data []byte, axisTags []string) (*avarTable, error) {
	if len(data) < 8 {
		return nil, fmt.Errorf("avar too short")
	}
	t := &avarTable{version: getU32(data, 0)}
	if t.version>>16 != 1 {
		return nil, fmt.Errorf("unsupported avar version %#x", t.version)
	}
	count := int(getU16(data, 6))
	off := 8
	for i := 0; i < count; i++ {
		if off+2 > len(data) {
			return nil, fmt.Errorf("avar segment %d out of range", i)
		}
		n := int(getU16(data, off))
		off += 2
		seg := avarSegments{}
		for j := 0; j < n; j++ {
			if off+4 > len(data) {
				return nil, fmt.Errorf("avar mapping %d/%d out of range", i, j)
			}
			from := f2dot14ToFloat(getI16(data, off))
			to := f2dot14ToFloat(getI16(data, off+2))
			seg[from] = to
			off += 4
		}
		t.segments = append(t.segments, seg)
	}
	for len(t.segments) < len(axisTags) {
		t.segments = append(t.segments, avarSegments{})
	}
	return t, nil
}

func encodeAvar(t *avarTable) []byte {
	out := make([]byte, 0, 64)
	out = appendU32(out, t.version)
	out = appendU16(out, 0) // reserved
	out = appendU16(out, uint16(len(t.segments)))
	for _, seg := range t.segments {
		keys := make([]float64, 0, len(seg))
		for k := range seg {
			keys = append(keys, k)
		}
		sort.Float64s(keys)
		out = appendU16(out, uint16(len(keys)))
		for _, k := range keys {
			out = appendI16(out, floatToF2Dot14(k))
			out = appendI16(out, floatToF2Dot14(seg[k]))
		}
	}
	return out
}

// isValidAvarSegmentMap mirrors fontTools _isValidAvarSegmentMap.
func isValidAvarSegmentMap(seg avarSegments) bool {
	if len(seg) == 0 {
		return true
	}
	if seg[-1.0] != -1.0 || seg[0.0] != 0.0 || seg[1.0] != 1.0 {
		return false
	}
	keys := make([]float64, 0, len(seg))
	for k := range seg {
		keys = append(keys, k)
	}
	sort.Float64s(keys)
	for i := 1; i < len(keys); i++ {
		if seg[keys[i-1]] > seg[keys[i]] {
			return false
		}
	}
	return true
}

// statAxisValue is one parsed STAT axis value table.
type statAxisValue struct {
	format int
	// formats 1-3:
	axisIndex   int
	flags       uint16
	valueNameID uint16
	value       float64
	// format 2:
	nominal float64
	minV    float64
	maxV    float64
	// format 3:
	linked float64
	// format 4:
	records [][2]float64 // (axisIndex, value)
}

type statTable struct {
	version  uint32
	axes     []statAxis // design axes
	values   []statAxisValue
	elidedID uint16
}

type statAxis struct {
	tag      string
	nameID   uint16
	ordering uint16
}

func parseSTAT(data []byte) (*statTable, error) {
	if len(data) < 18 {
		return nil, fmt.Errorf("STAT too short")
	}
	t := &statTable{version: getU32(data, 0)}
	axisSize := int(getU16(data, 4))
	axisCount := int(getU16(data, 6))
	axesOff := int(getU32(data, 8))
	valueCount := int(getU16(data, 12))
	valueOffsOff := int(getU32(data, 14))
	if t.version >= 0x00010001 {
		if len(data) < 20 {
			return nil, fmt.Errorf("STAT v1.1+ too short")
		}
		t.elidedID = getU16(data, 18)
	}
	if axisSize < 8 {
		return nil, fmt.Errorf("bad STAT axis size %d", axisSize)
	}
	for i := 0; i < axisCount; i++ {
		off := axesOff + i*axisSize
		if off+8 > len(data) {
			return nil, fmt.Errorf("STAT axis %d out of range", i)
		}
		t.axes = append(t.axes, statAxis{string(data[off : off+4]), getU16(data, off+4), getU16(data, off+6)})
	}
	for i := 0; i < valueCount; i++ {
		if valueOffsOff+2*i+2 > len(data) {
			return nil, fmt.Errorf("STAT value offset %d out of range", i)
		}
		off := valueOffsOff + int(getU16(data, valueOffsOff+2*i))
		v, err := parseSTATValue(data, off)
		if err != nil {
			return nil, fmt.Errorf("STAT value %d: %w", i, err)
		}
		t.values = append(t.values, v)
	}
	return t, nil
}

func parseSTATValue(data []byte, off int) (statAxisValue, error) {
	var v statAxisValue
	if off+2 > len(data) {
		return v, fmt.Errorf("value out of range")
	}
	v.format = int(getU16(data, off))
	switch v.format {
	case 1:
		if off+12 > len(data) {
			return v, fmt.Errorf("format 1 out of range")
		}
		v.axisIndex = int(getU16(data, off+2))
		v.flags = getU16(data, off+4)
		v.valueNameID = getU16(data, off+6)
		v.value = fixedToFloat(getI32(data, off+8))
	case 2:
		if off+18 > len(data) {
			return v, fmt.Errorf("format 2 out of range")
		}
		v.axisIndex = int(getU16(data, off+2))
		v.flags = getU16(data, off+4)
		v.valueNameID = getU16(data, off+6)
		v.nominal = fixedToFloat(getI32(data, off+8))
		v.minV = fixedToFloat(getI32(data, off+12))
		v.maxV = fixedToFloat(getI32(data, off+16))
	case 3:
		if off+16 > len(data) {
			return v, fmt.Errorf("format 3 out of range")
		}
		v.axisIndex = int(getU16(data, off+2))
		v.flags = getU16(data, off+4)
		v.valueNameID = getU16(data, off+6)
		v.value = fixedToFloat(getI32(data, off+8))
		v.linked = fixedToFloat(getI32(data, off+12))
	case 4:
		if off+8 > len(data) {
			return v, fmt.Errorf("format 4 out of range")
		}
		n := int(getU16(data, off+2))
		v.flags = getU16(data, off+4)
		v.valueNameID = getU16(data, off+6)
		if off+8+6*n > len(data) {
			return v, fmt.Errorf("format 4 records out of range")
		}
		for i := 0; i < n; i++ {
			ai := int(getU16(data, off+8+6*i))
			vv := fixedToFloat(getI32(data, off+8+6*i+2))
			v.records = append(v.records, [2]float64{float64(ai), vv})
		}
	default:
		return v, fmt.Errorf("unsupported STAT axis value format %d", v.format)
	}
	return v, nil
}

func encodeSTAT(t *statTable) []byte {
	// Layout: header, design axes, value offsets, values. Version 1.0
	// headers omit the elided fallback name ID.
	axisSize := 8
	axesOff := 20 // v1.1+ header size
	if t.version < 0x00010001 {
		axesOff = 18
	}
	valOffsOff := axesOff + axisSize*len(t.axes)
	out := make([]byte, 0, 128)
	out = appendU32(out, t.version)
	out = appendU16(out, uint16(axisSize))
	out = appendU16(out, uint16(len(t.axes)))
	out = appendU32(out, uint32(axesOff))
	out = appendU16(out, uint16(len(t.values)))
	out = appendU32(out, uint32(valOffsOff))
	if t.version >= 0x00010001 {
		out = appendU16(out, t.elidedID)
	}
	for _, a := range t.axes {
		out = append(out, a.tag[0], a.tag[1], a.tag[2], a.tag[3])
		out = appendU16(out, a.nameID)
		out = appendU16(out, a.ordering)
	}
	var bodies [][]byte
	for _, v := range t.values {
		bodies = append(bodies, encodeSTATValue(v))
	}
	// Axis value offsets of 0 read back as NULL, so the value data never
	// starts at offset 0: reserve two pad bytes after the offset array.
	// (fontTools uses the same NULL convention when reading.)
	off := 2*len(bodies) + 2
	for _, b := range bodies {
		out = appendU16(out, uint16(off))
		off += len(b)
	}
	if len(bodies) > 0 {
		out = appendU16(out, 0) // pad; keeps all value offsets nonzero
	}
	for _, b := range bodies {
		out = append(out, b...)
	}
	return out
}

func encodeSTATValue(v statAxisValue) []byte {
	out := appendU16(nil, uint16(v.format))
	switch v.format {
	case 1:
		out = appendU16(out, uint16(v.axisIndex))
		out = appendU16(out, v.flags)
		out = appendU16(out, v.valueNameID)
		out = appendU32(out, uint32(floatToFixed(v.value)))
	case 2:
		out = appendU16(out, uint16(v.axisIndex))
		out = appendU16(out, v.flags)
		out = appendU16(out, v.valueNameID)
		out = appendU32(out, uint32(floatToFixed(v.nominal)))
		out = appendU32(out, uint32(floatToFixed(v.minV)))
		out = appendU32(out, uint32(floatToFixed(v.maxV)))
	case 3:
		out = appendU16(out, uint16(v.axisIndex))
		out = appendU16(out, v.flags)
		out = appendU16(out, v.valueNameID)
		out = appendU32(out, uint32(floatToFixed(v.value)))
		out = appendU32(out, uint32(floatToFixed(v.linked)))
	case 4:
		out = appendU16(out, uint16(len(v.records)))
		out = appendU16(out, v.flags)
		out = appendU16(out, v.valueNameID)
		for _, r := range v.records {
			out = appendU16(out, uint16(int(r[0])))
			out = appendU32(out, uint32(floatToFixed(r[1])))
		}
	}
	return out
}

// filterSTATValues drops axis value tables outside pinned limits, mirroring
// fontTools axisValuesFromAxisLimits. limits maps axis tags to (min, max)
// user-space pins; axes absent from limits are unrestricted.
func filterSTATValues(t *statTable, limits map[string][2]float64) []statAxisValue {
	outside := func(axisIdx int, value float64) bool {
		if axisIdx < 0 || axisIdx >= len(t.axes) {
			return false
		}
		lim, ok := limits[t.axes[axisIdx].tag]
		if !ok {
			return false
		}
		return value < lim[0] || value > lim[1]
	}
	var kept []statAxisValue
	for _, v := range t.values {
		switch v.format {
		case 1, 2, 3:
			val := v.value
			if v.format == 2 {
				// Format 2 tables are matched by nominal value.
				val = v.nominal
			}
			if outside(v.axisIndex, val) {
				continue
			}
			kept = append(kept, v)
		case 4:
			drop := false
			for _, r := range v.records {
				if outside(int(r[0]), r[1]) {
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
