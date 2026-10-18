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

// Tolerances of the shape maths, the same as those of the library this is
// ported from.
const (
	distanceEpsilon = 1e-5
	angleEpsilon    = 1e-6
)

// Point is a point, or a vector, in the plane.
type Point struct{ X, Y float64 }

// Add returns p + o.
func (p Point) Add(o Point) Point { return Point{p.X + o.X, p.Y + o.Y} }

// Sub returns p - o.
func (p Point) Sub(o Point) Point { return Point{p.X - o.X, p.Y - o.Y} }

// Mul returns p scaled by f.
func (p Point) Mul(f float64) Point { return Point{p.X * f, p.Y * f} }

// Div returns p divided by f.
func (p Point) Div(f float64) Point { return Point{p.X / f, p.Y / f} }

// Rotate90 returns p turned by 90 degrees.
func (p Point) Rotate90() Point { return Point{-p.Y, p.X} }

// Rotate returns p turned around center by degrees.
func (p Point) Rotate(degrees float64, center Point) Point {
	radians := degrees * math.Pi / 180
	off := p.Sub(center)
	cos, sin := math.Cos(radians), math.Sin(radians)
	return Point{off.X*cos - off.Y*sin, off.X*sin + off.Y*cos}.Add(center)
}

// AngleRadians is the angle of the vector p from the X axis.
func (p Point) AngleRadians() float64 { return math.Atan2(p.Y, p.X) }

// Distance is the length of the vector p.
func (p Point) Distance() float64 { return math.Sqrt(p.X*p.X + p.Y*p.Y) }

// DistanceSquared is the squared length of the vector p.
func (p Point) DistanceSquared() float64 { return p.X*p.X + p.Y*p.Y }

// Dot is the dot product of p and o.
func (p Point) Dot(o Point) float64 { return p.X*o.X + p.Y*o.Y }

// Clockwise reports whether o is clockwise of p.
func (p Point) Clockwise(o Point) bool { return p.X*o.Y-p.Y*o.X > 0 }

// Direction is the unit vector in the direction of p.
func (p Point) Direction() Point { return p.Div(p.Distance()) }

func distance(x, y float64) float64 { return math.Sqrt(x*x + y*y) }

func distanceSquared(x, y float64) float64 { return x*x + y*y }

// directionVector is the unit vector in the direction of (x, y).
func directionVector(x, y float64) Point {
	d := distance(x, y)
	return Point{x / d, y / d}
}

// radialToCartesian is the point at radius and angle (radians) from center.
func radialToCartesian(radius, angleRadians float64, center Point) Point {
	return Point{math.Cos(angleRadians), math.Sin(angleRadians)}.Mul(radius).Add(center)
}

func square(x float64) float64 { return x * x }

func lerp(start, stop, fraction float64) float64 {
	return start*(1-fraction) + stop*fraction
}

func interpolate(start, stop Point, fraction float64) Point {
	return Point{lerp(start.X, stop.X, fraction), lerp(start.Y, stop.Y, fraction)}
}

// positiveModulo is num modulo 1, never negative.
func positiveModulo(num float64) float64 {
	return math.Mod(math.Mod(num, 1)+1, 1)
}

// convex reports whether the corner at current, between previous and next,
// is convex.
func convex(previous, current, next Point) bool {
	return current.Sub(previous).Clockwise(next.Sub(current))
}
