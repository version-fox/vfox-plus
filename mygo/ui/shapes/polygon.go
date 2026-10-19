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

import "math"

// Rounding says how a corner of a polygon is rounded.
type Rounding struct {
	// Radius of the circle that rounds the corner, in the units of the
	// polygon.
	Radius float64
	// Smoothing from 0 to 1 spreads the curvature over a longer stretch
	// of the edges, from a circular arc to a smooth transition.
	Smoothing float64
}

// Unrounded leaves a corner sharp.
var Unrounded = Rounding{}

// featureKind is what a feature of a polygon outline is.
type featureKind int

const (
	kindEdge featureKind = iota
	kindConvexCorner
	kindConcaveCorner
)

// feature is a part of the outline of a polygon: an edge, which morphing
// ignores, or a corner, which morphing matches to a corner of the other shape.
type feature struct {
	cubics []Cubic
	kind   featureKind
}

func (f feature) isCorner() bool { return f.kind != kindEdge }

func (f feature) isConvex() bool { return f.kind == kindConvexCorner }

func (f feature) transformed(transform func(x, y float64) (float64, float64)) feature {
	cubics := make([]Cubic, len(f.cubics))
	for i, cubic := range f.cubics {
		cubics[i] = cubic.transformed(transform)
	}
	return feature{cubics: cubics, kind: f.kind}
}

// Polygon is a polygon with rounded corners, drawn as a closed outline of
// cubic Bézier curves, as in the Android library `androidx.graphics.shapes`.
type Polygon struct {
	features []feature
	// Cubics is the outline, a closed loop of curves.
	Cubics []Cubic
	// Center of the polygon, from which the shape is measured.
	Center Point
}

// calculateCenter is the average of the vertices, as a list of coordinates.
func calculateCenter(vertices []float64) Point {
	var x, y float64
	for i := 0; i < len(vertices); i += 2 {
		x += vertices[i]
		y += vertices[i+1]
	}
	count := float64(len(vertices)) / 2
	return Point{x / count, y / count}
}

// FromVertices creates the polygon with the vertices (x and y coordinates
// one after the other), all rounded with rounding, or with perVertex when
// that is not nil. The center is the average of the vertices unless given.
func FromVertices(vertices []float64, rounding Rounding, perVertex []Rounding, center *Point) *Polygon {
	if len(vertices) < 6 || len(vertices)%2 != 0 {
		panic("shapes: a polygon needs at least three vertices with two coordinates each")
	}
	if perVertex != nil && len(perVertex)*2 != len(vertices) {
		panic("shapes: perVertex needs one rounding per vertex")
	}
	n := len(vertices) / 2
	vertex := func(i int) Point { return Point{vertices[(i%n)*2], vertices[(i%n)*2+1]} }

	corners := make([]*roundedCorner, n)
	for i := 0; i < n; i++ {
		vertexRounding := rounding
		if perVertex != nil {
			vertexRounding = perVertex[i]
		}
		corners[i] = newRoundedCorner(vertex(i+n-1), vertex(i), vertex(i+1), vertexRounding)
	}

	// Rounding two corners of a side may cut more than the side is long: the
	// cuts of the corners are reduced then, first the smoothing and then the
	// rounding itself.
	type adjust struct{ roundCutRatio, cutRatio float64 }
	cutAdjusts := make([]adjust, n)
	for i := 0; i < n; i++ {
		expectedRoundCut := corners[i].expectedRoundCut + corners[(i+1)%n].expectedRoundCut
		expectedCut := corners[i].expectedCut() + corners[(i+1)%n].expectedCut()
		next := vertex(i + 1)
		sideSize := distance(vertex(i).X-next.X, vertex(i).Y-next.Y)
		switch {
		case expectedRoundCut > sideSize:
			cutAdjusts[i] = adjust{sideSize / expectedRoundCut, 0}
		case expectedCut > sideSize:
			cutAdjusts[i] = adjust{1, (sideSize - expectedRoundCut) / (expectedCut - expectedRoundCut)}
		default:
			cutAdjusts[i] = adjust{1, 1}
		}
	}

	cornerCubics := make([][]Cubic, n)
	for i := 0; i < n; i++ {
		var allowedCuts [2]float64
		for delta := 0; delta <= 1; delta++ {
			adjustment := cutAdjusts[(i+n-1+delta)%n]
			allowedCuts[delta] = corners[i].expectedRoundCut*adjustment.roundCutRatio +
				(corners[i].expectedCut()-corners[i].expectedRoundCut)*adjustment.cutRatio
		}
		cornerCubics[i] = corners[i].cubics(allowedCuts[0], allowedCuts[1])
	}

	features := make([]feature, 0, n*2)
	for i := 0; i < n; i++ {
		kind := kindConcaveCorner
		if convex(vertex(i+n-1), vertex(i), vertex(i+1)) {
			kind = kindConvexCorner
		}
		last := cornerCubics[i][len(cornerCubics[i])-1]
		nextFirst := cornerCubics[(i+1)%n][0]
		features = append(features,
			feature{cubics: cornerCubics[i], kind: kind},
			feature{
				cubics: []Cubic{straightLine(last[6], last[7], nextFirst[0], nextFirst[1])},
				kind:   kindEdge,
			})
	}
	centerPoint := calculateCenter(vertices)
	if center != nil {
		centerPoint = *center
	}
	return newPolygon(features, centerPoint)
}

// newPolygon creates a polygon from its features and builds the outline.
func newPolygon(features []feature, center Point) *Polygon {
	polygon := &Polygon{features: features, Center: center}
	polygon.initCubics()
	return polygon
}

// initCubics joins the cubics of the features into the outline, leaving out
// the ones without length. A corner of three cubics at the start is cut in the
// middle of its arc, so that the outline starts and ends in the same corner.
func (p *Polygon) initCubics() {
	var firstCubic, lastCubic *Cubic
	var firstSplitStart, firstSplitEnd []Cubic
	if len(p.features) > 0 && len(p.features[0].cubics) == 3 {
		start, end := p.features[0].cubics[1].split(0.5)
		firstSplitStart = []Cubic{p.features[0].cubics[0], start}
		firstSplitEnd = []Cubic{end, p.features[0].cubics[2]}
	}
	for i := 0; i <= len(p.features); i++ {
		var featureCubics []Cubic
		switch {
		case i == 0 && firstSplitEnd != nil:
			featureCubics = firstSplitEnd
		case i == len(p.features):
			if firstSplitStart == nil {
				continue
			}
			featureCubics = firstSplitStart
		default:
			featureCubics = p.features[i].cubics
		}
		for _, cubic := range featureCubics {
			if !cubic.zeroLength() {
				if lastCubic != nil {
					p.Cubics = append(p.Cubics, *lastCubic)
				}
				c := cubic
				lastCubic = &c
				if firstCubic == nil {
					first := cubic
					firstCubic = &first
				}
			} else if lastCubic != nil {
				// Move the end of the last curve to where the empty one ends.
				lastCubic[6], lastCubic[7] = cubic[6], cubic[7]
			}
		}
	}
	if lastCubic != nil && firstCubic != nil {
		closing := *lastCubic
		closing[6], closing[7] = firstCubic[0], firstCubic[1]
		p.Cubics = append(p.Cubics, closing)
		return
	}
	p.Cubics = append(p.Cubics, Cubic{p.Center.X, p.Center.Y, p.Center.X, p.Center.Y, p.Center.X, p.Center.Y, p.Center.X, p.Center.Y})
}

// RegularPolygon creates the polygon with numVertices vertices on a circle of
// radius around center, all rounded with rounding, or with perVertex when that
// is not nil.
func RegularPolygon(numVertices int, radius float64, center Point, rounding Rounding, perVertex []Rounding) *Polygon {
	if numVertices < 3 {
		panic("shapes: a polygon needs at least three vertices")
	}
	vertices := make([]float64, 0, numVertices*2)
	for i := 0; i < numVertices; i++ {
		vertex := radialToCartesian(radius, math.Pi/float64(numVertices)*2*float64(i), center)
		vertices = append(vertices, vertex.X, vertex.Y)
	}
	return FromVertices(vertices, rounding, perVertex, &center)
}

// Circle creates a polygon that approximates a circle with numVertices
// rounded vertices.
func Circle(numVertices int, radius float64, center Point) *Polygon {
	if numVertices < 3 {
		panic("shapes: a circle needs at least three vertices")
	}
	// Rounding the corners of the polygon with the radius of the circle makes
	// the polygon the circle, once its vertices lie on a larger circle.
	theta := math.Pi / float64(numVertices)
	polygonRadius := radius / math.Cos(theta)
	return RegularPolygon(numVertices, polygonRadius, center, Rounding{Radius: radius}, nil)
}

// Rectangle creates the rectangle of the width and the height around center,
// rounded with rounding, or with perVertex when that is not nil. The vertices
// are the bottom right, bottom left, top left and top right ones.
func Rectangle(width, height float64, rounding Rounding, perVertex []Rounding, center Point) *Polygon {
	left, top := center.X-width/2, center.Y-height/2
	right, bottom := center.X+width/2, center.Y+height/2
	return FromVertices([]float64{right, bottom, left, bottom, left, top, right, top}, rounding, perVertex, &center)
}

// Star creates the polygon of a star with numVerticesPerRadius points on a
// circle of radius, between which the vertices of innerRadius lie. The points
// are rounded with rounding and the inner vertices with innerRounding, if it
// is not nil.
func Star(numVerticesPerRadius int, radius, innerRadius float64, rounding Rounding, innerRounding *Rounding, center Point) *Polygon {
	if radius <= 0 || innerRadius <= 0 || innerRadius >= radius {
		panic("shapes: a star needs 0 < innerRadius < radius")
	}
	var perVertex []Rounding
	if innerRounding != nil {
		for i := 0; i < numVerticesPerRadius; i++ {
			perVertex = append(perVertex, rounding, *innerRounding)
		}
	}
	vertices := make([]float64, 0, numVerticesPerRadius*4)
	for i := 0; i < numVerticesPerRadius; i++ {
		outer := radialToCartesian(radius, math.Pi/float64(numVerticesPerRadius)*2*float64(i), Point{})
		inner := radialToCartesian(innerRadius, math.Pi/float64(numVerticesPerRadius)*float64(2*i+1), Point{})
		vertices = append(vertices, outer.X+center.X, outer.Y+center.Y, inner.X+center.X, inner.Y+center.Y)
	}
	return FromVertices(vertices, rounding, perVertex, &center)
}

// Transformed applies transform to every point of the polygon.
func (p *Polygon) Transformed(transform func(x, y float64) (float64, float64)) *Polygon {
	features := make([]feature, len(p.features))
	for i, f := range p.features {
		features[i] = f.transformed(transform)
	}
	x, y := transform(p.Center.X, p.Center.Y)
	return newPolygon(features, Point{x, y})
}

// Rotated turns the polygon around the origin by degrees.
func (p *Polygon) Rotated(degrees float64) *Polygon {
	return p.Transformed(rotation(degrees))
}

func rotation(degrees float64) func(x, y float64) (float64, float64) {
	radians := degrees * math.Pi / 180
	cos, sin := math.Cos(radians), math.Sin(radians)
	return func(x, y float64) (float64, float64) { return x*cos - y*sin, x*sin + y*cos }
}

// Bounds returns the bounding box (left, top, right, bottom) of the outline.
// The approximation includes the control points and is never too small.
func (p *Polygon) Bounds(approximate bool) [4]float64 {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, cubic := range p.Cubics {
		bounds := cubic.calculateBounds(approximate)
		minX, minY = math.Min(minX, bounds[0]), math.Min(minY, bounds[1])
		maxX, maxY = math.Max(maxX, bounds[2]), math.Max(maxY, bounds[3])
	}
	return [4]float64{minX, minY, maxX, maxY}
}

// MaxBounds is the bounding box (left, top, right, bottom) of the circle around
// the center that reaches the point of the outline farthest from it: the
// polygon stays inside it whatever it is turned by.
func (p *Polygon) MaxBounds() [4]float64 {
	maxDistance := 0.0
	for _, cubic := range p.Cubics {
		anchor := distanceSquared(cubic[0]-p.Center.X, cubic[1]-p.Center.Y)
		middle := cubic.pointOnCurve(0.5)
		maxDistance = math.Max(maxDistance, math.Max(anchor, distanceSquared(middle.X-p.Center.X, middle.Y-p.Center.Y)))
	}
	radius := math.Sqrt(maxDistance)
	return [4]float64{p.Center.X - radius, p.Center.Y - radius, p.Center.X + radius, p.Center.Y + radius}
}

// FitToCircle scales the polygon around its center so that the circle that
// holds it fills the unit square, which keeps it inside its bounds while it
// spins.
func (p *Polygon) FitToCircle() *Polygon {
	bounds := p.MaxBounds()
	scale := 0.5 / ((bounds[2] - bounds[0]) / 2)
	cx, cy := p.Center.X, p.Center.Y
	return p.Transformed(func(x, y float64) (float64, float64) {
		return (x-cx)*scale + 0.5, (y-cy)*scale + 0.5
	})
}

// Normalized scales and moves the polygon into the unit square, keeping its
// aspect ratio and centring it.
func (p *Polygon) Normalized() *Polygon {
	bounds := p.Bounds(true)
	width, height := bounds[2]-bounds[0], bounds[3]-bounds[1]
	side := math.Max(width, height)
	offsetX := (side-width)/2 - bounds[0]
	offsetY := (side-height)/2 - bounds[1]
	return p.Transformed(func(x, y float64) (float64, float64) {
		return (x + offsetX) / side, (y + offsetY) / side
	})
}

// roundedCorner computes the curves that round one corner of a polygon.
type roundedCorner struct {
	p0, p1, p2       Point
	rounding         Rounding
	d1, d2           Point
	cornerRadius     float64
	smoothing        float64
	cosAngle         float64
	sinAngle         float64
	expectedRoundCut float64
	center           Point
}

func newRoundedCorner(p0, p1, p2 Point, rounding Rounding) *roundedCorner {
	c := &roundedCorner{p0: p0, p1: p1, p2: p2, rounding: rounding}
	v01, v21 := p0.Sub(p1), p2.Sub(p1)
	d01, d21 := v01.Distance(), v21.Distance()
	if d01 > 0 && d21 > 0 {
		c.d1, c.d2 = v01.Div(d01), v21.Div(d21)
		c.cornerRadius = rounding.Radius
		c.smoothing = rounding.Smoothing
		// The corner of the two edges: how far from the vertex the round
		// part starts.
		c.cosAngle = c.d1.Dot(c.d2)
		c.sinAngle = math.Sqrt(1 - square(c.cosAngle))
		if c.sinAngle > 1e-3 {
			c.expectedRoundCut = c.cornerRadius * (c.cosAngle + 1) / c.sinAngle
		}
	}
	return c
}

// expectedCut is how far the rounding, with its smoothing, reaches along the
// edges, if nothing is in its way.
func (c *roundedCorner) expectedCut() float64 { return (1 + c.smoothing) * c.expectedRoundCut }

// cubics returns the curves that round the corner, given how much of each
// of the two edges may be used: a flanking curve, the circular arc and the
// other flanking curve.
func (c *roundedCorner) cubics(allowedCut0, allowedCut1 float64) []Cubic {
	allowedCut := math.Min(allowedCut0, allowedCut1)
	if c.expectedRoundCut < distanceEpsilon || allowedCut < distanceEpsilon || c.cornerRadius < distanceEpsilon {
		// Not rounded: the corner is a single point.
		c.center = c.p1
		return []Cubic{straightLine(c.p1.X, c.p1.Y, c.p1.X, c.p1.Y)}
	}
	// Use the cut that fits, and scale the radius with it.
	actualRoundCut := math.Min(allowedCut, c.expectedRoundCut)
	actualSmoothing0 := c.actualSmoothing(allowedCut0)
	actualSmoothing1 := c.actualSmoothing(allowedCut1)
	actualR := c.cornerRadius * actualRoundCut / c.expectedRoundCut
	centerDistance := math.Sqrt(square(actualR) + square(actualRoundCut))
	c.center = c.p1.Add(c.d1.Add(c.d2).Div(2).Direction().Mul(centerDistance))
	circleIntersection0 := c.p1.Add(c.d1.Mul(actualRoundCut))
	circleIntersection2 := c.p1.Add(c.d2.Mul(actualRoundCut))
	flanking0 := c.flankingCurve(actualRoundCut, actualSmoothing0, c.p1, c.p0,
		circleIntersection0, circleIntersection2, c.center, actualR)
	flanking2 := c.flankingCurve(actualRoundCut, actualSmoothing1, c.p1, c.p2,
		circleIntersection2, circleIntersection0, c.center, actualR).reverse()
	return []Cubic{
		flanking0,
		circularArc(c.center.X, c.center.Y, flanking0[6], flanking0[7], flanking2[0], flanking2[1]),
		flanking2,
	}
}

// actualSmoothing is the smoothing that fits in what an edge allows.
func (c *roundedCorner) actualSmoothing(allowedCut float64) float64 {
	switch {
	case allowedCut > c.expectedCut():
		return c.smoothing
	case allowedCut > c.expectedRoundCut:
		return c.smoothing * (allowedCut - c.expectedRoundCut) / (c.expectedCut() - c.expectedRoundCut)
	default:
		return 0
	}
}

// flankingCurve is the curve that joins an edge to the circular arc.
func (c *roundedCorner) flankingCurve(actualRoundCut, actualSmoothing float64,
	corner, sideStart, circleSegmentIntersection, otherCircleSegmentIntersection, circleCenter Point,
	actualR float64,
) Cubic {
	sideDirection := sideStart.Sub(corner).Direction()
	curveStart := corner.Add(sideDirection.Mul(actualRoundCut * (1 + actualSmoothing)))
	// The end of the curve lies on the circle, between the intersection of
	// the circle with the edge and with the other edge.
	p := interpolate(circleSegmentIntersection,
		circleSegmentIntersection.Add(otherCircleSegmentIntersection).Div(2), actualSmoothing)
	curveEnd := circleCenter.Add(directionVector(p.X-circleCenter.X, p.Y-circleCenter.Y).Mul(actualR))
	circleTangent := curveEnd.Sub(circleCenter).Rotate90()
	anchorEnd, ok := lineIntersection(sideStart, sideDirection, curveEnd, circleTangent)
	if !ok {
		anchorEnd = circleSegmentIntersection
	}
	anchorStart := curveStart.Add(anchorEnd.Mul(2)).Div(3)
	return newCubic(curveStart, anchorStart, anchorEnd, curveEnd)
}

// lineIntersection is where the line through p0 in the direction d0 meets the
// line through p1 in the direction d1, if they are not parallel.
func lineIntersection(p0, d0, p1, d1 Point) (Point, bool) {
	rotatedD1 := d1.Rotate90()
	den := d0.Dot(rotatedD1)
	if math.Abs(den) < distanceEpsilon {
		return Point{}, false
	}
	num := p1.Sub(p0).Dot(rotatedD1)
	if math.Abs(den) < distanceEpsilon*math.Abs(num) {
		return Point{}, false
	}
	return p0.Add(d0.Mul(num / den)), true
}
