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

// The shapes of the Material 3 Expressive shape library, in the unit square
// (`MaterialShapes` of `androidx.compose.material3`). Only the ones that
// vfox-plus draws are here.

// pointAndRound is a vertex with the rounding of its corner.
type pointAndRound struct {
	p Point
	r Rounding
}

func pr(x, y float64) pointAndRound { return pointAndRound{p: Point{x, y}} }

func prr(x, y, radius float64) pointAndRound {
	return pointAndRound{p: Point{x, y}, r: Rounding{Radius: radius}}
}

func prs(x, y, radius, smoothing float64) pointAndRound {
	return pointAndRound{p: Point{x, y}, r: Rounding{Radius: radius, Smoothing: smoothing}}
}

// customPolygon repeats the vertices reps times around the center of the unit
// square, turning them, or with mirroring turning and mirroring them.
func customPolygon(points []pointAndRound, reps int, mirroring bool) *Polygon {
	center := Point{0.5, 0.5}
	repeated := repeat(points, reps, center, mirroring)
	vertices := make([]float64, 0, len(repeated)*2)
	perVertex := make([]Rounding, 0, len(repeated))
	for _, point := range repeated {
		vertices = append(vertices, point.p.X, point.p.Y)
		perVertex = append(perVertex, point.r)
	}
	return FromVertices(vertices, Unrounded, perVertex, &center)
}

func repeat(points []pointAndRound, reps int, center Point, mirroring bool) []pointAndRound {
	var result []pointAndRound
	if !mirroring {
		for i := 0; i < len(points)*reps; i++ {
			point := points[i%len(points)]
			result = append(result, pointAndRound{
				p: point.p.Rotate(float64(i/len(points))*360/float64(reps), center),
				r: point.r,
			})
		}
		return result
	}
	type measure struct{ angle, distance float64 }
	measures := make([]measure, len(points))
	for i, point := range points {
		offset := point.p.Sub(center)
		measures[i] = measure{offset.AngleRadians(), offset.Distance()}
	}
	actualReps := reps * 2
	sectionAngle := math.Pi * 2 / float64(actualReps)
	for r := 0; r < actualReps; r++ {
		for index := range points {
			i := index
			if r%2 != 0 {
				i = len(points) - 1 - index
			}
			if i > 0 || r%2 == 0 {
				angle := sectionAngle * float64(r)
				if r%2 == 0 {
					angle += measures[i].angle
				} else {
					angle += sectionAngle - measures[i].angle + 2*measures[0].angle
				}
				point := Point{math.Cos(angle), math.Sin(angle)}.Mul(measures[i].distance).Add(center)
				result = append(result, pointAndRound{p: point, r: points[i].r})
			}
		}
	}
	return result
}

// scaleThenRotate scales and then turns, by degrees.
func scaleThenRotate(scaleX, scaleY, degrees float64) func(x, y float64) (float64, float64) {
	turn := rotation(degrees)
	return func(x, y float64) (float64, float64) { return turn(x*scaleX, y*scaleY) }
}

var (
	cornerRound15  = Rounding{Radius: 0.15}
	cornerRound20  = Rounding{Radius: 0.2}
	cornerRound50  = Rounding{Radius: 0.5}
	cornerRound100 = Rounding{Radius: 1}
	unitCenter     = Point{0.5, 0.5}
	origin         = Point{}
)

func star(vertices int, innerRadius float64, rounding Rounding, degrees float64) *Polygon {
	polygon := Star(vertices, 1, innerRadius, rounding, nil, origin)
	if degrees != 0 {
		polygon = polygon.Rotated(degrees)
	}
	return polygon.Normalized()
}

// The shapes. They are polygons of the unit square.
var (
	// ShapeCircle is a circle.
	ShapeCircle = Circle(10, 0.5, unitCenter)
	// ShapeOval is a tilted ellipse.
	ShapeOval = Circle(8, 1, origin).Transformed(scaleThenRotate(1, 0.64, -45)).Normalized()
	// ShapePill is a pill, with the long side horizontal.
	ShapePill = customPolygon([]pointAndRound{prr(0.961, 0.039, 0.426), pr(1.001, 0.428), prr(1, 0.609, 1)}, 2, true).Normalized()
	// ShapePentagon is a pentagon with rounded corners.
	ShapePentagon = customPolygon([]pointAndRound{prr(0.5, -0.009, 0.172), prr(1.03, 0.365, 0.164), prr(0.828, 0.97, 0.169)}, 1, true).Normalized()
	// ShapeGem is a gem.
	ShapeGem = customPolygon([]pointAndRound{
		prs(0.499, 1.023, 0.241, 0.778), prr(-0.005, 0.792, 0.208), prr(0.073, 0.258, 0.228), prr(0.433, -0, 0.491),
	}, 1, true).Normalized()
	// ShapeArch is a rounded arch.
	ShapeArch = RegularPolygon(4, 1, origin, Unrounded,
		[]Rounding{cornerRound100, cornerRound100, cornerRound20, cornerRound20}).Rotated(-135).Normalized()
	// ShapeSunny is a sun with eight rays.
	ShapeSunny = star(8, 0.8, cornerRound15, 0)
	// ShapeCookie4 is a cookie with four sides.
	ShapeCookie4 = customPolygon([]pointAndRound{prr(1.237, 1.236, 0.258), prr(0.5, 0.918, 0.233)}, 4, false).Normalized()
	// ShapeCookie6 is a cookie with six sides.
	ShapeCookie6 = customPolygon([]pointAndRound{prr(0.723, 0.884, 0.394), prr(0.5, 1.099, 0.398)}, 6, false).Normalized()
	// ShapeCookie9 is a cookie with nine sides.
	ShapeCookie9 = star(9, 0.8, cornerRound50, -90)
	// ShapeCookie12 is a cookie with twelve sides, the shape of the
	// application icon.
	ShapeCookie12 = star(12, 0.8, cornerRound50, -90)
	// ShapeClover4 is a clover with four leaves.
	ShapeClover4 = customPolygon([]pointAndRound{pr(0.5, 0.074), prr(0.725, -0.099, 0.476)}, 4, true).Normalized()
	// ShapeClover8 is a clover with eight leaves.
	ShapeClover8 = customPolygon([]pointAndRound{pr(0.5, 0.036), prr(0.758, -0.101, 0.209)}, 8, false).Normalized()
	// ShapeSoftBurst is a burst with ten rounded points.
	ShapeSoftBurst = customPolygon([]pointAndRound{prr(0.193, 0.277, 0.053), prr(0.176, 0.055, 0.053)}, 10, false).Normalized()
	// ShapeFlower is a flower with eight petals.
	ShapeFlower = customPolygon([]pointAndRound{pr(0.370, 0.187), prr(0.416, 0.049, 0.381), prr(0.479, 0.001, 0.095)}, 8, true).Normalized()
	// ShapePuffyDiamond is a diamond with puffy sides.
	ShapePuffyDiamond = customPolygon([]pointAndRound{prr(0.87, 0.13, 0.146), pr(0.818, 0.357), prr(1, 0.332, 0.853)}, 4, true).Normalized()
)

// AvatarShapes are the shapes of the avatars of plugins, in the order of the
// design system: a stable hash of the name picks one.
var AvatarShapes = []*Polygon{
	ShapeCookie9, ShapeClover4, ShapeSunny, ShapeSoftBurst, ShapeCookie6, ShapeGem,
	ShapePentagon, ShapeFlower, ShapePuffyDiamond, ShapeCookie4, ShapeArch, ShapeClover8,
}
