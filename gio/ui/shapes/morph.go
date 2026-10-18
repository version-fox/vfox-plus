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

package shapes

import (
	"math"
	"sort"
)

// Morph is the transition between two polygons. The outlines are measured,
// their corners are matched and both are cut into the same number of curves,
// so that every curve of one has a partner on the other to interpolate with.
type Morph struct {
	match [][2]Cubic
}

// NewMorph prepares the transition from start to end.
func NewMorph(start, end *Polygon) *Morph {
	return &Morph{match: matchPolygons(start, end)}
}

// AsCubics returns the outline at progress, from 0 (start) to 1 (end). Values
// beyond overshoot, which springs do.
func (m *Morph) AsCubics(progress float64) []Cubic {
	result := make([]Cubic, 0, len(m.match))
	var first, last *Cubic
	for _, pair := range m.match {
		cubic := interpolateCubics(pair[0], pair[1], progress)
		if first == nil {
			c := cubic
			first = &c
		}
		if last != nil {
			result = append(result, *last)
		}
		c := cubic
		last = &c
	}
	if last != nil && first != nil {
		// Make sure the outline closes exactly.
		closing := *last
		closing[6], closing[7] = first[0], first[1]
		result = append(result, closing)
	}
	return result
}

// matchPolygons cuts the outlines of two polygons into pairs of curves.
func matchPolygons(p1, p2 *Polygon) [][2]Cubic {
	measured1 := measurePolygon(p1)
	measured2 := measurePolygon(p2)
	mapper := featureMapper(measured1.features, measured2.features)

	// The second outline starts where the first corner of the first one maps
	// to.
	polygon2CutPoint := mapper.mapForward(0)
	bs1 := measured1
	bs2 := measured2.cutAndShift(polygon2CutPoint)

	var result [][2]Cubic
	i1, i2 := 0, 0
	b1 := bs1.get(i1)
	i1++
	b2 := bs2.get(i2)
	i2++
	for b1 != nil && b2 != nil {
		// The progress at which each of the two current curves ends, in the
		// space of the first outline.
		b1a := b1.endProgress
		if i1 == len(bs1.cubics) {
			b1a = 1
		}
		b2a := 1.0
		if i2 != len(bs2.cubics) {
			b2a = mapper.mapBack(positiveModulo(b2.endProgress + polygon2CutPoint))
		}
		minb := math.Min(b1a, b2a)

		// Cut the curve that goes on after minb; the other one is whole.
		var seg1, seg2, newB1, newB2 *measuredCubic
		if b1a > minb+angleEpsilon {
			seg1, newB1 = b1.cutAtProgress(minb)
		} else {
			seg1, newB1 = b1, bs1.get(i1)
			i1++
		}
		if b2a > minb+angleEpsilon {
			seg2, newB2 = b2.cutAtProgress(positiveModulo(mapper.mapForward(minb) - polygon2CutPoint))
		} else {
			seg2, newB2 = b2, bs2.get(i2)
			i2++
		}
		result = append(result, [2]Cubic{seg1.cubic, seg2.cubic})
		b1, b2 = newB1, newB2
	}
	return result
}

// progressableFeature is a feature and where on the outline, from 0 to 1, it
// lies.
type progressableFeature struct {
	progress float64
	feature  feature
}

// measuredCubic is a curve of an outline, with the stretch of the outline
// (from 0 to 1) that it covers.
type measuredCubic struct {
	cubic         Cubic
	startProgress float64
	endProgress   float64
	size          float64
}

func newMeasuredCubic(cubic Cubic, start, end float64) *measuredCubic {
	return &measuredCubic{cubic: cubic, startProgress: start, endProgress: end, size: measureCubic(cubic)}
}

// cutAtProgress cuts the curve where the outline has reached progress.
func (c *measuredCubic) cutAtProgress(progress float64) (*measuredCubic, *measuredCubic) {
	bounded := math.Min(math.Max(progress, c.startProgress), c.endProgress)
	outlineSize := c.endProgress - c.startProgress
	relative := (bounded - c.startProgress) / outlineSize
	t := findCubicCutPoint(c.cubic, relative*c.size)
	first, second := c.cubic.split(t)
	return newMeasuredCubic(first, c.startProgress, bounded), newMeasuredCubic(second, bounded, c.endProgress)
}

// measuredPolygon is an outline whose curves know their share of its length.
type measuredPolygon struct {
	cubics   []*measuredCubic
	features []progressableFeature
}

func (m *measuredPolygon) get(index int) *measuredCubic {
	if index < 0 || index >= len(m.cubics) {
		return nil
	}
	return m.cubics[index]
}

// newMeasuredPolygon measures the curves given with the progress at which
// each of them (and the outline) ends.
func newMeasuredPolygon(features []progressableFeature, cubics []Cubic, outlineProgress []float64) *measuredPolygon {
	measured := &measuredPolygon{features: features}
	start := 0.0
	for i, cubic := range cubics {
		if outlineProgress[i+1]-outlineProgress[i] > distanceEpsilon {
			measured.cubics = append(measured.cubics, newMeasuredCubic(cubic, start, outlineProgress[i+1]))
			start = outlineProgress[i+1]
		}
	}
	measured.cubics[len(measured.cubics)-1].endProgress = 1
	return measured
}

// measurePolygon measures the outline of a polygon and where its corners are.
func measurePolygon(polygon *Polygon) *measuredPolygon {
	type cornerAt struct {
		feature feature
		cubic   int
	}
	var cubics []Cubic
	var corners []cornerAt
	for _, f := range polygon.features {
		for i, cubic := range f.cubics {
			// A corner is placed at the middle of its curves.
			if f.isCorner() && i == len(f.cubics)/2 {
				corners = append(corners, cornerAt{f, len(cubics)})
			}
			cubics = append(cubics, cubic)
		}
	}
	measures := make([]float64, len(cubics)+1)
	total := 0.0
	for i, cubic := range cubics {
		total += measureCubic(cubic)
		measures[i+1] = total
	}
	outlineProgress := make([]float64, len(measures))
	for i, measure := range measures {
		outlineProgress[i] = measure / total
	}
	features := make([]progressableFeature, len(corners))
	for i, corner := range corners {
		features[i] = progressableFeature{
			progress: positiveModulo((outlineProgress[corner.cubic] + outlineProgress[corner.cubic+1]) / 2),
			feature:  corner.feature,
		}
	}
	return newMeasuredPolygon(features, cubics, outlineProgress)
}

// cutAndShift makes the outline start at cuttingPoint (from 0 to 1).
func (m *measuredPolygon) cutAndShift(cuttingPoint float64) *measuredPolygon {
	if cuttingPoint < distanceEpsilon {
		return m
	}
	target := 0
	for i, cubic := range m.cubics {
		if cuttingPoint >= cubic.startProgress && cuttingPoint <= cubic.endProgress {
			target = i
			break
		}
	}
	b1, b2 := m.cubics[target].cutAtProgress(cuttingPoint)
	cubics := []Cubic{b2.cubic}
	for i := 1; i < len(m.cubics); i++ {
		cubics = append(cubics, m.cubics[(i+target)%len(m.cubics)].cubic)
	}
	cubics = append(cubics, b1.cubic)

	progress := make([]float64, len(m.cubics)+2)
	for i := range progress {
		switch i {
		case 0:
			progress[i] = 0
		case len(m.cubics) + 1:
			progress[i] = 1
		default:
			index := (target + i - 1) % len(m.cubics)
			progress[i] = positiveModulo(m.cubics[index].endProgress - cuttingPoint)
		}
	}
	features := make([]progressableFeature, len(m.features))
	for i, f := range m.features {
		features[i] = progressableFeature{positiveModulo(f.progress - cuttingPoint), f.feature}
	}
	return newMeasuredPolygon(features, cubics, progress)
}

// The length of a curve is measured on a few segments; this is exact enough to
// match the outlines, and the measure is the same on both.
const measureSegments = 3

// closestProgressTo walks the curve until the length threshold is reached. It
// returns the progress at that point, and the length that was walked.
func closestProgressTo(cubic Cubic, threshold float64) (float64, float64) {
	total := 0.0
	remainder := threshold
	previous := cubic.Anchor0()
	for i := 0; i <= measureSegments; i++ {
		progress := float64(i) / measureSegments
		point := cubic.pointOnCurve(progress)
		segment := point.Sub(previous).Distance()
		if segment >= remainder {
			return progress - (1-remainder/segment)/measureSegments, threshold
		}
		remainder -= segment
		total += segment
		previous = point
	}
	return 1, total
}

func measureCubic(cubic Cubic) float64 {
	_, length := closestProgressTo(cubic, math.Inf(1))
	return length
}

func findCubicCutPoint(cubic Cubic, length float64) float64 {
	progress, _ := closestProgressTo(cubic, length)
	return progress
}

// progressInRange reports whether progress lies on the stretch of the
// outline from progressFrom to progressTo, which may wrap around.
func progressInRange(progress, progressFrom, progressTo float64) bool {
	if progressTo >= progressFrom {
		return progress >= progressFrom && progress <= progressTo
	}
	return progress >= progressFrom || progress <= progressTo
}

// progressDistance is how far apart two progress values are on the loop.
func progressDistance(p1, p2 float64) float64 {
	diff := math.Abs(p1 - p2)
	return math.Min(diff, 1-diff)
}

// linearMap maps x from the progress values xValues to yValues, linearly in
// between.
func linearMap(xValues, yValues []float64, x float64) float64 {
	segmentStart := -1
	for i := range xValues {
		if progressInRange(x, xValues[i], xValues[(i+1)%len(xValues)]) {
			segmentStart = i
			break
		}
	}
	if segmentStart == -1 {
		panic("shapes: no segment for the progress")
	}
	segmentEnd := (segmentStart + 1) % len(xValues)
	segmentSizeX := positiveModulo(xValues[segmentEnd] - xValues[segmentStart])
	segmentSizeY := positiveModulo(yValues[segmentEnd] - yValues[segmentStart])
	position := 0.5
	if segmentSizeX >= 0.001 {
		position = positiveModulo(x-xValues[segmentStart]) / segmentSizeX
	}
	return positiveModulo(yValues[segmentStart] + segmentSizeY*position)
}

// doubleMapper maps progress values of one outline to those of the other and
// back, through a list of pairs of corresponding values.
type doubleMapper struct {
	source, target []float64
}

func newDoubleMapper(mappings [][2]float64) doubleMapper {
	mapper := doubleMapper{make([]float64, len(mappings)), make([]float64, len(mappings))}
	for i, pair := range mappings {
		mapper.source[i], mapper.target[i] = pair[0], pair[1]
	}
	return mapper
}

func (m doubleMapper) mapForward(x float64) float64 { return linearMap(m.source, m.target, x) }

func (m doubleMapper) mapBack(x float64) float64 { return linearMap(m.target, m.source, x) }

// featureMapper matches the corners of two outlines: nearby corners of the
// same kind are mapped to each other first, as long as the order of the
// corners along the outline stays the same.
func featureMapper(features1, features2 []progressableFeature) doubleMapper {
	corners := func(features []progressableFeature) []progressableFeature {
		var result []progressableFeature
		for _, f := range features {
			if f.feature.isCorner() {
				result = append(result, f)
			}
		}
		return result
	}
	return newDoubleMapper(doMapping(corners(features1), corners(features2)))
}

type distanceVertex struct {
	distance float64
	f1, f2   int
}

func doMapping(features1, features2 []progressableFeature) [][2]float64 {
	var vertices []distanceVertex
	for i, f1 := range features1 {
		for j, f2 := range features2 {
			if d := featureDistSquared(f1.feature, f2.feature); d != math.MaxFloat64 {
				vertices = append(vertices, distanceVertex{d, i, j})
			}
		}
	}
	sort.SliceStable(vertices, func(a, b int) bool { return vertices[a].distance < vertices[b].distance })
	switch len(vertices) {
	case 0:
		return [][2]float64{{0, 0}, {0.5, 0.5}}
	case 1:
		f1, f2 := features1[vertices[0].f1].progress, features2[vertices[0].f2].progress
		return [][2]float64{{f1, f2}, {math.Mod(f1+0.5, 1), math.Mod(f2+0.5, 1)}}
	}
	helper := mappingHelper{usedF1: map[int]bool{}, usedF2: map[int]bool{}}
	for _, v := range vertices {
		helper.add(features1, features2, v.f1, v.f2)
	}
	return helper.mapping
}

// mappingHelper collects the pairs of corners that are mapped to each other.
type mappingHelper struct {
	mapping        [][2]float64
	usedF1, usedF2 map[int]bool
}

func (h *mappingHelper) add(features1, features2 []progressableFeature, i1, i2 int) {
	if h.usedF1[i1] || h.usedF2[i2] {
		return
	}
	p1, p2 := features1[i1].progress, features2[i2].progress
	// Where the pair goes, to keep the first values ascending.
	index := sort.Search(len(h.mapping), func(i int) bool { return h.mapping[i][0] >= p1 })
	if index < len(h.mapping) && h.mapping[index][0] == p1 {
		panic("shapes: two corners cannot be at the same progress")
	}
	n := len(h.mapping)
	if n >= 1 {
		before := h.mapping[(index+n-1)%n]
		after := h.mapping[index%n]
		// Too close to a pair that exists: skip.
		if progressDistance(p1, before[0]) < distanceEpsilon || progressDistance(p1, after[0]) < distanceEpsilon ||
			progressDistance(p2, before[1]) < distanceEpsilon || progressDistance(p2, after[1]) < distanceEpsilon {
			return
		}
		// The second values must keep their order as well.
		if n > 1 && !progressInRange(p2, before[1], after[1]) {
			return
		}
	}
	h.mapping = append(h.mapping, [2]float64{})
	copy(h.mapping[index+1:], h.mapping[index:])
	h.mapping[index] = [2]float64{p1, p2}
	h.usedF1[i1], h.usedF2[i2] = true, true
}

// featureDistSquared is the squared distance between the corners, or the
// largest number when one is convex and the other concave.
func featureDistSquared(f1, f2 feature) float64 {
	if f1.isCorner() && f2.isCorner() && f1.isConvex() != f2.isConvex() {
		return math.MaxFloat64
	}
	return representativePoint(f1).Sub(representativePoint(f2)).DistanceSquared()
}

// representativePoint is the middle between the start and the end of the
// feature.
func representativePoint(f feature) Point {
	first, last := f.cubics[0], f.cubics[len(f.cubics)-1]
	return Point{(first[0] + last[6]) / 2, (first[1] + last[7]) / 2}
}
