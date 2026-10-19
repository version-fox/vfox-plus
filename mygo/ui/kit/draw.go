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

package kit

import (
	"math"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// Corner radii of the Material 3 shape scale, in dp
// (docs/design-system.md#corner-radius-scale).
const (
	CornerNone          float32 = 0
	CornerExtraSmall    float32 = 4
	CornerSmall         float32 = 8
	CornerMedium        float32 = 12
	CornerLarge         float32 = 16
	CornerLargeInc      float32 = 20
	CornerExtraLarge    float32 = 28
	CornerExtraLargeInc float32 = 32
	// CornerFull is a radius so large that the corners are semicircles.
	CornerFull float32 = 9999
)

// Radii are the four corner radii of a rounded rectangle, in dp: top left, top
// right, bottom right and bottom left, as mygo takes them.
type Radii [4]float32

// Uniform is the same radius for the four corners.
func Uniform(r float32) Radii { return Radii{r, r, r, r} }

// fit shrinks the radii so that they fit a rectangle of the size, as CSS does.
func (r Radii) fit(w, h float32) Radii {
	f := float32(1)
	scale := func(sum, side float32) {
		if sum > side && sum > 0 {
			f = min(f, side/sum)
		}
	}
	scale(r[0]+r[1], w)
	scale(r[3]+r[2], w)
	scale(r[0]+r[3], h)
	scale(r[1]+r[2], h)
	return Radii{r[0] * f, r[1] * f, r[2] * f, r[3] * f}
}

func (r Radii) uniform() (float32, bool) { return r[0], r[0] == r[1] && r[1] == r[2] && r[2] == r[3] }

// Apply sets the corners of the element.
func (r Radii) Apply(e ui.Element) ui.Element { return e.Radius(r[:]...) }

// kappa is the control point distance that makes a cubic curve approximate a
// quarter of a circle.
const kappa = 0.5522847498

// roundedRect is the path of a rectangle with rounded corners.
func roundedRect(r ui.Rect, radii Radii) *ui.Path {
	radii = radii.fit(r.W, r.H)
	x0, y0, x1, y1 := r.X, r.Y, r.X+r.W, r.Y+r.H
	tl, tr, br, bl := radii[0], radii[1], radii[2], radii[3]
	var p ui.Path
	p.MoveTo(x0+tl, y0)
	p.LineTo(x1-tr, y0)
	p.CubeTo(x1-tr+kappa*tr, y0, x1, y0+tr-kappa*tr, x1, y0+tr)
	p.LineTo(x1, y1-br)
	p.CubeTo(x1, y1-br+kappa*br, x1-br+kappa*br, y1, x1-br, y1)
	p.LineTo(x0+bl, y1)
	p.CubeTo(x0+bl-kappa*bl, y1, x0, y1-bl+kappa*bl, x0, y1-bl)
	p.LineTo(x0, y0+tl)
	p.CubeTo(x0, y0+tl-kappa*tl, x0+tl-kappa*tl, y0, x0+tl, y0)
	p.Close()
	return &p
}

// fillRRect fills a rounded rectangle with corners that may differ.
func fillRRect(p *ui.Painter, r ui.Rect, radii Radii, col ui.Color) {
	if col.A == 0 || r.W <= 0 || r.H <= 0 {
		return
	}
	if radius, ok := radii.uniform(); ok {
		p.Fill(r, col, min(radius, min(r.W, r.H)/2))
		return
	}
	p.FillPath(roundedRect(r, radii), col)
}

// strokeRRect draws the outline of a rounded rectangle inside the rectangle.
func strokeRRect(p *ui.Painter, r ui.Rect, radii Radii, width float32, col ui.Color) {
	if col.A == 0 || r.W <= 0 || r.H <= 0 || width <= 0 {
		return
	}
	if radius, ok := radii.uniform(); ok {
		p.Stroke(r, col, min(radius, min(r.W, r.H)/2), width)
		return
	}
	// The stroke of a path is centred on it: inset the path by half the width.
	half := width / 2
	inner := ui.Rect{X: r.X + half, Y: r.Y + half, W: r.W - width, H: r.H - width}
	shrunk := Radii{max(0, radii[0]-half), max(0, radii[1]-half), max(0, radii[2]-half), max(0, radii[3]-half)}
	p.StrokePath(roundedRect(inner, shrunk), width, col)
}

// Elevation gives an element the shadow of a surface at a level of the
// Material 3 elevation scale (0 to 5), as the two lights of the specification
// cast it: a sharp shadow under the edge, and a soft one around.
func Elevation(c Ctx, e ui.Element, level int) ui.Element {
	if level <= 0 {
		return e
	}
	shadow := c.Colors().Shadow
	l := float32(level)
	e.Shadow(0, l, l*2, 0, shadow.Alpha(0.30))
	e.Shadow(0, l*0.5, l, 0, shadow.Alpha(0.15))
	return e
}

// ShapePath is the path of a shape of the shape library, in a square: the
// shape fills the unit square of the polygon, which is put at the centre of the
// rectangle with the length of its shorter side and turned by rotation radians.
func ShapePath(cubics []shapes.Cubic, r ui.Rect, rotation float64) *ui.Path {
	side := min(r.W, r.H)
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	sin, cos := math.Sincos(rotation)
	at := func(x, y float64) (float32, float32) {
		x, y = (x-0.5)*float64(side), (y-0.5)*float64(side)
		return cx + float32(x*cos-y*sin), cy + float32(x*sin+y*cos)
	}
	var p ui.Path
	for i, cubic := range cubics {
		if i == 0 {
			p.MoveTo(at(cubic[0], cubic[1]))
		}
		x1, y1 := at(cubic[2], cubic[3])
		x2, y2 := at(cubic[4], cubic[5])
		x3, y3 := at(cubic[6], cubic[7])
		p.CubeTo(x1, y1, x2, y2, x3, y3)
	}
	p.Close()
	return &p
}

// fillShape fills a shape of the shape library inside a rectangle.
func fillShape(p *ui.Painter, cubics []shapes.Cubic, r ui.Rect, rotation float64, col ui.Color) {
	if col.A == 0 || r.W <= 0 || r.H <= 0 {
		return
	}
	p.FillPath(ShapePath(cubics, r, rotation), col)
}

// Shape is a box of a size that draws a shape of the shape library in a colour,
// with what the child builds on top of it, centred.
func Shape(c Ctx, polygon *shapes.Polygon, size float32, col ui.Color, child func()) ui.Element {
	return ShapeRotated(c, polygon.Cubics, size, col, 0, child)
}

// ShapeRotated is Shape for the outline of a polygon or of a morph (its
// cubics), turned by rotation radians.
func ShapeRotated(c Ctx, cubics []shapes.Cubic, size float32, col ui.Color, rotation float64, child func()) ui.Element {
	box := ui.Box(c.Context).Size(size, size).Shrink(0).Center().Draw(func(p *ui.Painter, r ui.Rect) {
		fillShape(p, cubics, r, rotation, col)
	})
	if child != nil {
		box.Children(child)
	}
	return box
}
