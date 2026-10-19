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

package shapes_test

import (
	"encoding/json"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

type pt struct{ x, y float64 }

// sampleCubics samples the outline, n points per curve.
func sampleCubics(cubics []shapes.Cubic, n int) []pt {
	var points []pt
	for _, c := range cubics {
		for i := 0; i < n; i++ {
			t := float64(i) / float64(n)
			u := 1 - t
			points = append(points, pt{
				c[0]*u*u*u + c[2]*3*t*u*u + c[4]*3*t*t*u + c[6]*t*t*t,
				c[1]*u*u*u + c[3]*3*t*u*u + c[5]*3*t*t*u + c[7]*t*t*t,
			})
		}
	}
	return points
}

// parsePath samples an SVG path of absolute M, L, H, V, C and Z commands.
func parsePath(t *testing.T, path string, n int) []pt {
	t.Helper()
	fields := strings.Fields(strings.NewReplacer(
		"M", " M ", "L", " L ", "H", " H ", "V", " V ", "C", " C ", "Z", " Z ", ",", " ").Replace(path))
	var points []pt
	var current, start pt
	number := func(i int) float64 {
		value, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			t.Fatalf("bad path number %q", fields[i])
		}
		return value
	}
	for i := 0; i < len(fields); {
		switch fields[i] {
		case "M":
			current = pt{number(i + 1), number(i + 2)}
			start = current
			points = append(points, current)
			i += 3
		case "L":
			current = pt{number(i + 1), number(i + 2)}
			points = append(points, current)
			i += 3
		case "H":
			current = pt{number(i + 1), current.y}
			points = append(points, current)
			i += 2
		case "V":
			current = pt{current.x, number(i + 1)}
			points = append(points, current)
			i += 2
		case "C":
			// Several curves may follow one command letter.
			i++
			for i+5 < len(fields) && !strings.ContainsAny(fields[i], "MLHVCZ") {
				c1, c2, end := pt{number(i), number(i + 1)}, pt{number(i + 2), number(i + 3)}, pt{number(i + 4), number(i + 5)}
				for k := 1; k <= n; k++ {
					s := float64(k) / float64(n)
					u := 1 - s
					points = append(points, pt{
						current.x*u*u*u + c1.x*3*s*u*u + c2.x*3*s*s*u + end.x*s*s*s,
						current.y*u*u*u + c1.y*3*s*u*u + c2.y*3*s*s*u + end.y*s*s*s,
					})
				}
				current = end
				i += 6
			}
		case "Z":
			points = append(points, start)
			i++
		default:
			t.Fatalf("unsupported path command %q", fields[i])
		}
	}
	return points
}

// normalize scales the points into the unit square, keeping the aspect ratio
// and centring them.
func normalize(points []pt) []pt {
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, p := range points {
		minX, minY = math.Min(minX, p.x), math.Min(minY, p.y)
		maxX, maxY = math.Max(maxX, p.x), math.Max(maxY, p.y)
	}
	side := math.Max(maxX-minX, maxY-minY)
	result := make([]pt, len(points))
	for i, p := range points {
		result[i] = pt{(p.x-minX)/side + (1-(maxX-minX)/side)/2, (p.y-minY)/side + (1-(maxY-minY)/side)/2}
	}
	return result
}

// distanceToPolyline is the distance of p to the closed polyline.
func distanceToPolyline(p pt, line []pt) float64 {
	best := math.MaxFloat64
	for i := range line {
		a, b := line[i], line[(i+1)%len(line)]
		dx, dy := b.x-a.x, b.y-a.y
		length := dx*dx + dy*dy
		t := 0.0
		if length > 0 {
			t = math.Max(0, math.Min(1, ((p.x-a.x)*dx+(p.y-a.y)*dy)/length))
		}
		best = math.Min(best, math.Hypot(p.x-(a.x+t*dx), p.y-(a.y+t*dy)))
	}
	return best
}

// hausdorff is the largest distance of a point of one outline to the other.
func hausdorff(a, b []pt) float64 {
	directed := func(from, to []pt) float64 {
		worst := 0.0
		for _, p := range from {
			worst = math.Max(worst, distanceToPolyline(p, to))
		}
		return worst
	}
	return math.Max(directed(a, b), directed(b, a))
}

// The outlines of the Material shapes, from @m3e/web: the shapes drawn here
// must not be further than 2 % of their size from them.
func TestShapesMatchTheMaterialOutlines(t *testing.T) {
	content, err := os.ReadFile("testdata/shapes.json")
	if err != nil {
		t.Fatal(err)
	}
	var paths map[string]string
	if err := json.Unmarshal(content, &paths); err != nil {
		t.Fatal(err)
	}
	for name, polygon := range map[string]*shapes.Polygon{
		"4-leaf-clover": shapes.ShapeClover4, "4-sided-cookie": shapes.ShapeCookie4,
		"6-sided-cookie": shapes.ShapeCookie6, "8-leaf-clover": shapes.ShapeClover8,
		"9-sided-cookie": shapes.ShapeCookie9, "12-sided-cookie": shapes.ShapeCookie12,
		"oval": shapes.ShapeOval, "pentagon": shapes.ShapePentagon, "pill": shapes.ShapePill,
		"soft-burst": shapes.ShapeSoftBurst, "sunny": shapes.ShapeSunny, "flower": shapes.ShapeFlower,
		"gem": shapes.ShapeGem, "arch": shapes.ShapeArch, "puffy-diamond": shapes.ShapePuffyDiamond,
		"circle": shapes.ShapeCircle,
	} {
		reference, ok := paths[name]
		if !ok {
			t.Errorf("no outline for %s", name)
			continue
		}
		expected := normalize(parsePath(t, reference, 40))
		actual := normalize(sampleCubics(polygon.Cubics, 40))
		if dir := os.Getenv("SHAPES_OVERLAY"); dir != "" {
			writeOverlay(t, dir+"/"+name+".svg", expected, actual)
		}
		if distance := hausdorff(expected, actual); distance > 0.02 {
			t.Errorf("%s: %.4f away from the Material outline", name, distance)
		}
	}
}

func TestOutlinesAreClosedLoops(t *testing.T) {
	all := append([]*shapes.Polygon{shapes.ShapeCircle, shapes.ShapeOval, shapes.ShapePill}, shapes.AvatarShapes...)
	for index, polygon := range all {
		cubics := polygon.Cubics
		if len(cubics) < 3 {
			t.Fatalf("shape %d has %d curves", index, len(cubics))
		}
		previous := cubics[len(cubics)-1]
		for i, c := range cubics {
			if math.Abs(c[0]-previous[6]) > 1e-4 || math.Abs(c[1]-previous[7]) > 1e-4 {
				t.Errorf("shape %d: curve %d does not start where the one before ends", index, i)
			}
			previous = c
		}
		// Normalized shapes fit the unit square.
		bounds := polygon.Bounds(false)
		if bounds[0] < -0.02 || bounds[1] < -0.02 || bounds[2] > 1.02 || bounds[3] > 1.02 {
			t.Errorf("shape %d leaves the unit square: %v", index, bounds)
		}
	}
}

func TestMorphStartsAtTheFirstShapeAndEndsAtTheSecond(t *testing.T) {
	pairs := [][2]*shapes.Polygon{
		{shapes.ShapeSoftBurst, shapes.ShapeCookie9}, {shapes.ShapeCookie9, shapes.ShapePentagon},
		{shapes.ShapePentagon, shapes.ShapePill}, {shapes.ShapePill, shapes.ShapeSunny},
		{shapes.ShapeSunny, shapes.ShapeCookie4}, {shapes.ShapeCookie4, shapes.ShapeOval},
		{shapes.ShapeOval, shapes.ShapeSoftBurst}, {shapes.ShapeCircle, shapes.ShapeFlower},
	}
	for index, pair := range pairs {
		morph := shapes.NewMorph(pair[0], pair[1])
		for progress, target := range map[float64]*shapes.Polygon{0: pair[0], 1: pair[1]} {
			outline := morph.AsCubics(progress)
			if len(outline) < 3 {
				t.Fatalf("pair %d: %d curves", index, len(outline))
			}
			distance := hausdorff(sampleCubics(outline, 32), sampleCubics(target.Cubics, 32))
			if distance > 0.01 {
				t.Errorf("pair %d at %v: %.4f away from the shape", index, progress, distance)
			}
		}
		// The halfway shape is a shape of its own, inside the unit square.
		middle := morph.AsCubics(0.5)
		for _, p := range sampleCubics(middle, 8) {
			if p.x < -0.1 || p.x > 1.1 || p.y < -0.1 || p.y > 1.1 {
				t.Fatalf("pair %d: %v leaves the square", index, p)
			}
		}
		// The outline stays closed.
		last, first := middle[len(middle)-1], middle[0]
		if math.Abs(last[6]-first[0]) > 1e-9 || math.Abs(last[7]-first[1]) > 1e-9 {
			t.Errorf("pair %d: the outline is not closed", index)
		}
	}
}

func TestPolygonConstructionRules(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("a polygon needs three vertices")
		}
	}()
	shapes.FromVertices([]float64{0, 0, 1, 1}, shapes.Unrounded, nil, nil)
}

func TestRectangleAndStarShapes(t *testing.T) {
	square := shapes.Rectangle(2, 2, shapes.Rounding{Radius: 0.5}, nil, shapes.Point{})
	if len(square.Cubics) < 8 {
		t.Errorf("a rounded square has curves for its corners, got %d", len(square.Cubics))
	}
	star := shapes.Star(5, 1, 0.5, shapes.Unrounded, nil, shapes.Point{})
	if len(star.Cubics) != 10 {
		t.Errorf("a star with five points has ten edges, got %d", len(star.Cubics))
	}
	rotated := star.Rotated(90)
	if math.Abs(rotated.Center.X) > 1e-9 || math.Abs(rotated.Center.Y) > 1e-9 {
		t.Errorf("got %v", rotated.Center)
	}
}

// writeOverlay draws the reference outline in black and the shape in red, as an
// SVG: `SHAPES_OVERLAY=dir go test` writes one per shape, to look at.
func writeOverlay(t *testing.T, path string, reference, actual []pt) {
	t.Helper()
	line := func(points []pt, color string) string {
		var out strings.Builder
		out.WriteString(`<polygon fill="none" stroke="` + color + `" stroke-width="0.004" points="`)
		for _, p := range points {
			out.WriteString(strconv.FormatFloat(p.x, 'f', 4, 64) + "," + strconv.FormatFloat(p.y, 'f', 4, 64) + " ")
		}
		return out.String() + `"/>`
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="-0.05 -0.05 1.1 1.1" width="300" height="300">` +
		`<rect x="-0.05" y="-0.05" width="1.1" height="1.1" fill="white"/>` +
		line(reference, "black") + line(actual, "red") + `</svg>`
	err := os.WriteFile(path, []byte(svg), 0o600) //nolint:gosec // A file for the developer to look at.
	if err != nil {
		t.Fatal(err)
	}
}
