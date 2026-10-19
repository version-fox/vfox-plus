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

// Cubic is a cubic Bézier curve: the coordinates of the first anchor, the two
// control points and the second anchor, in this order.
type Cubic [8]float64

// Anchor0 is the start of the curve.
func (c Cubic) Anchor0() Point { return Point{c[0], c[1]} }

// Control0 is the control point of the start.
func (c Cubic) Control0() Point { return Point{c[2], c[3]} }

// Control1 is the control point of the end.
func (c Cubic) Control1() Point { return Point{c[4], c[5]} }

// Anchor1 is the end of the curve.
func (c Cubic) Anchor1() Point { return Point{c[6], c[7]} }

func newCubic(anchor0, control0, control1, anchor1 Point) Cubic {
	return Cubic{anchor0.X, anchor0.Y, control0.X, control0.Y, control1.X, control1.Y, anchor1.X, anchor1.Y}
}

// straightLine is the cubic that draws the line from (x0, y0) to (x1, y1).
func straightLine(x0, y0, x1, y1 float64) Cubic {
	return Cubic{
		x0, y0,
		lerp(x0, x1, 1.0/3), lerp(y0, y1, 1.0/3),
		lerp(x0, x1, 2.0/3), lerp(y0, y1, 2.0/3),
		x1, y1,
	}
}

// circularArc is the cubic that approximates the arc of the circle around
// (centerX, centerY) from (x0, y0) to (x1, y1).
func circularArc(centerX, centerY, x0, y0, x1, y1 float64) Cubic {
	p0d := directionVector(x0-centerX, y0-centerY)
	p1d := directionVector(x1-centerX, y1-centerY)
	rotatedP0 := p0d.Rotate90()
	rotatedP1 := p1d.Rotate90()
	clockwise := rotatedP0.Dot(Point{x1 - centerX, y1 - centerY}) >= 0
	cosa := p0d.Dot(p1d)
	if cosa > 0.999 {
		// The arc is so short that a line is as good.
		return straightLine(x0, y0, x1, y1)
	}
	direction := -1.0
	if clockwise {
		direction = 1
	}
	k := distance(x0-centerX, y0-centerY) * 4 / 3 *
		(math.Sqrt(2*(1-cosa)) - math.Sqrt(1-cosa*cosa)) / (1 - cosa) * direction
	return Cubic{
		x0, y0,
		x0 + rotatedP0.X*k, y0 + rotatedP0.Y*k,
		x1 - rotatedP1.X*k, y1 - rotatedP1.Y*k,
		x1, y1,
	}
}

// pointOnCurve is the point of the curve at t, from 0 to 1.
func (c Cubic) pointOnCurve(t float64) Point {
	u := 1 - t
	return Point{
		c[0]*(u*u*u) + c[2]*(3*t*u*u) + c[4]*(3*t*t*u) + c[6]*(t*t*t),
		c[1]*(u*u*u) + c[3]*(3*t*u*u) + c[5]*(3*t*t*u) + c[7]*(t*t*t),
	}
}

// zeroLength reports whether the curve starts and ends at the same point.
func (c Cubic) zeroLength() bool {
	return math.Abs(c[0]-c[6]) < distanceEpsilon && math.Abs(c[1]-c[7]) < distanceEpsilon
}

// calculateBounds returns the bounding box (left, top, right, bottom) of the
// curve. With approximate, the control points count as part of it, which is
// faster and never too small.
func (c Cubic) calculateBounds(approximate bool) [4]float64 {
	if c.zeroLength() {
		return [4]float64{c[0], c[1], c[0], c[1]}
	}
	minX, minY := math.Min(c[0], c[6]), math.Min(c[1], c[7])
	maxX, maxY := math.Max(c[0], c[6]), math.Max(c[1], c[7])
	if approximate {
		return [4]float64{
			math.Min(minX, math.Min(c[2], c[4])), math.Min(minY, math.Min(c[3], c[5])),
			math.Max(maxX, math.Max(c[2], c[4])), math.Max(maxY, math.Max(c[3], c[5])),
		}
	}
	// The extrema are where the derivative is zero, a quadratic per axis.
	extend := func(a0, c0, c1, a1 float64, axis int, lo, hi *float64) {
		a := -a0 + 3*c0 - 3*c1 + a1
		b := 2*a0 - 4*c0 + 2*c1
		cc := -a0 + c0
		consider := func(t float64) {
			if t >= 0 && t <= 1 {
				value := c.pointOnCurve(t).X
				if axis == 1 {
					value = c.pointOnCurve(t).Y
				}
				*lo = math.Min(*lo, value)
				*hi = math.Max(*hi, value)
			}
		}
		if math.Abs(a) < distanceEpsilon {
			if b != 0 {
				consider(2 * cc / (-2 * b))
			}
			return
		}
		if discriminant := b*b - 4*a*cc; discriminant >= 0 {
			root := math.Sqrt(discriminant)
			consider((-b + root) / (2 * a))
			consider((-b - root) / (2 * a))
		}
	}
	extend(c[0], c[2], c[4], c[6], 0, &minX, &maxX)
	extend(c[1], c[3], c[5], c[7], 1, &minY, &maxY)
	return [4]float64{minX, minY, maxX, maxY}
}

// split cuts the curve at t into two curves that draw the same.
func (c Cubic) split(t float64) (Cubic, Cubic) {
	u := 1 - t
	point := c.pointOnCurve(t)
	return Cubic{
		c[0], c[1],
		c[0]*u + c[2]*t, c[1]*u + c[3]*t,
		c[0]*(u*u) + c[2]*(2*u*t) + c[4]*(t*t), c[1]*(u*u) + c[3]*(2*u*t) + c[5]*(t*t),
		point.X, point.Y,
	}, Cubic{
		point.X, point.Y,
		c[2]*(u*u) + c[4]*(2*u*t) + c[6]*(t*t), c[3]*(u*u) + c[5]*(2*u*t) + c[7]*(t*t),
		c[4]*u + c[6]*t, c[5]*u + c[7]*t,
		c[6], c[7],
	}
}

// reverse draws the curve backwards.
func (c Cubic) reverse() Cubic {
	return Cubic{c[6], c[7], c[4], c[5], c[2], c[3], c[0], c[1]}
}

// transformed applies f to the four points of the curve.
func (c Cubic) transformed(f func(x, y float64) (float64, float64)) Cubic {
	var result Cubic
	for i := range 4 {
		result[2*i], result[2*i+1] = f(c[2*i], c[2*i+1])
	}
	return result
}

// interpolateCubics is the curve between a and b, at progress from 0 to 1.
func interpolateCubics(a, b Cubic, progress float64) Cubic {
	var result Cubic
	for i := range result {
		result[i] = lerp(a[i], b[i], progress)
	}
	return result
}
