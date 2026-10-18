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
	"image"
	"image/color"
	"math"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/version-fox/vfox-plus/gio/ui/shapes"
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
	CornerFull          float32 = 9999
)

// Radii are the four corner radii of a rounded rectangle, in pixels.
type Radii struct{ TopLeft, TopRight, BottomRight, BottomLeft float32 }

// Uniform is the same radius for the four corners.
func Uniform(r float32) Radii { return Radii{r, r, r, r} }

// radii converts corner radii in dp, where CornerFull stands for fully
// rounded, to pixels.
func (c Ctx) radii(tl, tr, br, bl float32) Radii {
	px := func(v float32) float32 {
		if v >= CornerFull {
			return 1e6
		}
		return v * float32(c.DP(1))
	}
	return Radii{px(tl), px(tr), px(br), px(bl)}
}

// Round is the radii of a uniform corner in dp.
func (c Ctx) Round(dp float32) Radii { return c.radii(dp, dp, dp, dp) }

// Fit shrinks radii so that they fit a rectangle of the size.
func (r Radii) fit(size image.Point) Radii {
	w, h := float32(size.X), float32(size.Y)
	f := float32(1)
	scale := func(sum, side float32) {
		if sum > side && sum > 0 {
			f = minf(f, side/sum)
		}
	}
	scale(r.TopLeft+r.TopRight, w)
	scale(r.BottomLeft+r.BottomRight, w)
	scale(r.TopLeft+r.BottomLeft, h)
	scale(r.TopRight+r.BottomRight, h)
	return Radii{r.TopLeft * f, r.TopRight * f, r.BottomRight * f, r.BottomLeft * f}
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// kappa is the control point distance that makes a cubic curve approximate a
// quarter of a circle.
const kappa = 0.5522847498

// roundedRect is the path of a rectangle with rounded corners.
func roundedRect(ops *op.Ops, rect image.Rectangle, r Radii) clip.PathSpec {
	r = r.fit(rect.Size())
	x0, y0 := float32(rect.Min.X), float32(rect.Min.Y)
	x1, y1 := float32(rect.Max.X), float32(rect.Max.Y)
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(f32.Pt(x0+r.TopLeft, y0))
	p.LineTo(f32.Pt(x1-r.TopRight, y0))
	p.CubeTo(f32.Pt(x1-r.TopRight+kappa*r.TopRight, y0), f32.Pt(x1, y0+r.TopRight-kappa*r.TopRight), f32.Pt(x1, y0+r.TopRight))
	p.LineTo(f32.Pt(x1, y1-r.BottomRight))
	p.CubeTo(f32.Pt(x1, y1-r.BottomRight+kappa*r.BottomRight), f32.Pt(x1-r.BottomRight+kappa*r.BottomRight, y1), f32.Pt(x1-r.BottomRight, y1))
	p.LineTo(f32.Pt(x0+r.BottomLeft, y1))
	p.CubeTo(f32.Pt(x0+r.BottomLeft-kappa*r.BottomLeft, y1), f32.Pt(x0, y1-r.BottomLeft+kappa*r.BottomLeft), f32.Pt(x0, y1-r.BottomLeft))
	p.LineTo(f32.Pt(x0, y0+r.TopLeft))
	p.CubeTo(f32.Pt(x0, y0+r.TopLeft-kappa*r.TopLeft), f32.Pt(x0+r.TopLeft-kappa*r.TopLeft, y0), f32.Pt(x0+r.TopLeft, y0))
	p.Close()
	return p.End()
}

// ClipRRect clips drawing to a rounded rectangle until the returned function
// is called.
func ClipRRect(ops *op.Ops, rect image.Rectangle, r Radii) func() {
	stack := clip.Outline{Path: roundedRect(ops, rect, r)}.Op().Push(ops)
	return stack.Pop
}

// FillRRect fills a rounded rectangle.
func FillRRect(ops *op.Ops, rect image.Rectangle, r Radii, col color.NRGBA) {
	if col.A == 0 || rect.Empty() {
		return
	}
	defer ClipRRect(ops, rect, r)()
	paint.Fill(ops, col)
}

// StrokeRRect draws the outline of a rounded rectangle, inside the rectangle.
func StrokeRRect(ops *op.Ops, rect image.Rectangle, r Radii, width float32, col color.NRGBA) {
	if col.A == 0 || rect.Empty() || width <= 0 {
		return
	}
	half := width / 2
	inner := image.Rect(rect.Min.X, rect.Min.Y, rect.Max.X, rect.Max.Y)
	// The stroke is centred on the path: inset the path by half its width.
	fr := r.fit(rect.Size())
	shrunk := Radii{
		maxf(0, fr.TopLeft-half), maxf(0, fr.TopRight-half),
		maxf(0, fr.BottomRight-half), maxf(0, fr.BottomLeft-half),
	}
	path := roundedRectF(ops, rectF{float32(inner.Min.X) + half, float32(inner.Min.Y) + half,
		float32(inner.Max.X) - half, float32(inner.Max.Y) - half}, shrunk)
	defer clip.Stroke{Path: path, Width: width}.Op().Push(ops).Pop()
	paint.Fill(ops, col)
}

// rectF is a rectangle with fractional coordinates.
type rectF struct{ x0, y0, x1, y1 float32 }

// roundedRectF is roundedRect with fractional coordinates.
func roundedRectF(ops *op.Ops, rect rectF, r Radii) clip.PathSpec {
	x0, y0, x1, y1 := rect.x0, rect.y0, rect.x1, rect.y1
	var p clip.Path
	p.Begin(ops)
	p.MoveTo(f32.Pt(x0+r.TopLeft, y0))
	p.LineTo(f32.Pt(x1-r.TopRight, y0))
	p.CubeTo(f32.Pt(x1-r.TopRight+kappa*r.TopRight, y0), f32.Pt(x1, y0+r.TopRight-kappa*r.TopRight), f32.Pt(x1, y0+r.TopRight))
	p.LineTo(f32.Pt(x1, y1-r.BottomRight))
	p.CubeTo(f32.Pt(x1, y1-r.BottomRight+kappa*r.BottomRight), f32.Pt(x1-r.BottomRight+kappa*r.BottomRight, y1), f32.Pt(x1-r.BottomRight, y1))
	p.LineTo(f32.Pt(x0+r.BottomLeft, y1))
	p.CubeTo(f32.Pt(x0+r.BottomLeft-kappa*r.BottomLeft, y1), f32.Pt(x0, y1-r.BottomLeft+kappa*r.BottomLeft), f32.Pt(x0, y1-r.BottomLeft))
	p.LineTo(f32.Pt(x0, y0+r.TopLeft))
	p.CubeTo(f32.Pt(x0, y0+r.TopLeft-kappa*r.TopLeft), f32.Pt(x0+r.TopLeft-kappa*r.TopLeft, y0), f32.Pt(x0+r.TopLeft, y0))
	p.Close()
	return p.End()
}

// FillCircle fills a circle.
func FillCircle(ops *op.Ops, center image.Point, radius int, col color.NRGBA) {
	FillRRect(ops, image.Rect(center.X-radius, center.Y-radius, center.X+radius, center.Y+radius), Uniform(float32(radius)), col)
}

// Elevation draws the shadow of a surface at the given level, in dp: the
// surface it is drawn under is the rounded rectangle given. Gio has no blur,
// so the shadow is a few stacked translucent shapes that fall off like one.
func (c Ctx) Elevation(rect image.Rectangle, r Radii, level float32) {
	if level <= 0 {
		return
	}
	dp := float32(c.DP(1))
	const layers = 7
	blur := level * 2 * dp
	offset := level * 0.5 * dp
	base := Alpha(c.Env.Theme.Colors.Shadow, 0.32/layers)
	for i := 1; i <= layers; i++ {
		spread := blur * float32(i) / layers
		grown := image.Rect(rect.Min.X-int(spread), rect.Min.Y-int(spread-offset), rect.Max.X+int(spread), rect.Max.Y+int(spread+offset))
		FillRRect(c.Ops, grown, Radii{
			r.TopLeft + spread, r.TopRight + spread, r.BottomRight + spread, r.BottomLeft + spread,
		}, Alpha(base, float32(layers-i+1)/float32(layers)*2))
	}
}

// ShapePath is the path of a shape of the shape library, in a square: the
// shape fills the unit square of the polygon, which is put at center with the
// given side length and turned by rotation radians.
func ShapePath(ops *op.Ops, cubics []shapes.Cubic, center f32.Point, side float32, rotation float64) clip.PathSpec {
	sin, cos := math.Sincos(rotation)
	at := func(x, y float64) f32.Point {
		x, y = (x-0.5)*float64(side), (y-0.5)*float64(side)
		return f32.Pt(center.X+float32(x*cos-y*sin), center.Y+float32(x*sin+y*cos))
	}
	var p clip.Path
	p.Begin(ops)
	for i, cubic := range cubics {
		if i == 0 {
			p.MoveTo(at(cubic[0], cubic[1]))
		}
		p.CubeTo(at(cubic[2], cubic[3]), at(cubic[4], cubic[5]), at(cubic[6], cubic[7]))
	}
	p.Close()
	return p.End()
}

// FillShape fills a shape of the shape library inside a rectangle.
func FillShape(ops *op.Ops, cubics []shapes.Cubic, rect image.Rectangle, rotation float64, col color.NRGBA) {
	if col.A == 0 || rect.Empty() {
		return
	}
	center := f32.Pt(float32(rect.Min.X+rect.Max.X)/2, float32(rect.Min.Y+rect.Max.Y)/2)
	side := float32(minInt(rect.Dx(), rect.Dy()))
	defer clip.Outline{Path: ShapePath(ops, cubics, center, side, rotation)}.Op().Push(ops).Pop()
	paint.Fill(ops, col)
}

// Opacity draws what the function draws with the given opacity (0 to 1).
func Opacity(ops *op.Ops, opacity float32, draw func()) {
	if opacity >= 1 {
		draw()
		return
	}
	defer paint.PushOpacity(ops, clamp01(opacity)).Pop()
	draw()
}

// scaleAbout scales what is drawn until the returned function is called, about
// the pivot, in pixels.
func scaleAbout(c Ctx, pivot image.Point, scale float32) func() {
	t := f32.Affine2D{}.Scale(f32.Pt(float32(pivot.X), float32(pivot.Y)), f32.Pt(scale, scale))
	return op.Affine(t).Push(c.Ops).Pop
}

// translated moves what is drawn until the returned function is called.
func translated(c Ctx, by f32.Point) func() {
	return op.Affine(f32.Affine2D{}.Offset(by)).Push(c.Ops).Pop
}

// rotateAbout turns what is drawn until the returned function is called, about
// the pivot, in pixels, by radians.
func rotateAbout(c Ctx, pivot image.Point, radians float64) func() {
	t := f32.Affine2D{}.Rotate(f32.Pt(float32(pivot.X), float32(pivot.Y)), float32(radians))
	return op.Affine(t).Push(c.Ops).Pop
}

// f32Pt makes a point of pixel offsets.
func f32Pt(x, y float32) f32.Point { return f32.Pt(x, y) }

// clipRect clips drawing to a rectangle until the returned function is called.
func clipRect(c Ctx, rect image.Rectangle) func() {
	return clip.Rect(rect).Push(c.Ops).Pop
}
