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

	"gioui.org/f32"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// WindowGlyph is the symbol on a button of the title bar.
type WindowGlyph int

// The symbols of the window buttons.
const (
	// GlyphMinimize is a horizontal line.
	GlyphMinimize WindowGlyph = iota
	// GlyphMaximize is a square.
	GlyphMaximize
	// GlyphRestore is two squares, one behind the other.
	GlyphRestore
	// GlyphClose is a cross.
	GlyphClose
)

// windowButtonSize is the diameter of a window button in dp.
const windowButtonSize = 32

// WindowButton is a button of the title bar that the application draws itself,
// where the desktop does not: it minimises, maximises, restores or closes the
// window. It is a circle that shows its state when the pointer is on it, the one
// that closes the window in the colour of errors.
type WindowButton struct {
	// ID identifies the button between frames, within its scope.
	ID string
	// Glyph is the symbol on the button.
	Glyph WindowGlyph
	// Name names the button for the tests; it is not shown, because the
	// symbols are those of every desktop.
	Name string
	// OnClick runs when the button is clicked.
	OnClick func()
}

// W is the button as a widget.
func (b WindowButton) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[buttonState](c, b.ID)
		cs := c.Env.Theme.Colors
		content := cs.OnSurfaceVariant
		tint := content
		if b.Glyph == GlyphClose {
			tint = cs.Error
		}
		spec := pressSpec{role: RoleButton, name: b.Name, enabled: true, key: c.key(b.ID), onClick: b.OnClick}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			d := c.DP(windowButtonSize)
			size := c.Constraints.Constrain(image.Pt(d, d))
			rect := image.Rectangle{Max: size}
			FillRRect(c.Ops, rect, Uniform(float32(d)/2), st.press.stateLayer(c, in, tint))
			focusRing(c, in, rect, Uniform(float32(d)/2))
			drawWindowGlyph(c, b.Glyph, image.Pt(size.X/2, size.Y/2), content)
			return layout.Dimensions{Size: size}
		})
	}
}

// drawWindowGlyph draws a symbol of 10 dp around a centre.
func drawWindowGlyph(c Ctx, glyph WindowGlyph, centre image.Point, col color.NRGBA) {
	half := c.DP(5)
	width := float32(maxInt(1, c.DP(1.5)))
	rect := image.Rect(centre.X-half, centre.Y-half, centre.X+half, centre.Y+half)
	pt := func(x, y int) f32.Point { return f32.Pt(float32(x), float32(y)) }
	switch glyph {
	case GlyphMinimize:
		strokeLines(c, col, width, [][]f32.Point{{pt(rect.Min.X, centre.Y), pt(rect.Max.X, centre.Y)}})
	case GlyphMaximize:
		StrokeRRect(c.Ops, rect, Uniform(float32(c.DP(1))), width, col)
	case GlyphRestore:
		shift := c.DP(3)
		front := image.Rect(rect.Min.X, rect.Min.Y+shift, rect.Max.X-shift, rect.Max.Y)
		StrokeRRect(c.Ops, front, Uniform(float32(c.DP(1))), width, col)
		// The part of the square behind that the one in front does not hide.
		strokeLines(c, col, width, [][]f32.Point{{
			pt(rect.Min.X+shift, rect.Min.Y+shift), pt(rect.Min.X+shift, rect.Min.Y), pt(rect.Max.X, rect.Min.Y),
			pt(rect.Max.X, rect.Max.Y-shift), pt(rect.Max.X-shift, rect.Max.Y-shift),
		}})
	case GlyphClose:
		strokeLines(c, col, width, [][]f32.Point{
			{pt(rect.Min.X, rect.Min.Y), pt(rect.Max.X, rect.Max.Y)},
			{pt(rect.Min.X, rect.Max.Y), pt(rect.Max.X, rect.Min.Y)},
		})
	}
}

// strokeLines strokes the paths, each of which is a series of straight lines.
func strokeLines(c Ctx, col color.NRGBA, width float32, lines [][]f32.Point) {
	var path clip.Path
	path.Begin(c.Ops)
	for _, line := range lines {
		path.MoveTo(line[0])
		for _, point := range line[1:] {
			path.LineTo(point)
		}
	}
	defer clip.Stroke{Path: path.End(), Width: width}.Op().Push(c.Ops).Pop()
	paint.Fill(c.Ops, col)
}

// WindowDrag lays out child and makes its area one that moves the window when
// the user drags it: the title of a title bar that the application draws.
func WindowDrag(child Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		dims := child(c)
		defer clip.Rect{Max: dims.Size}.Push(c.Ops).Pop()
		system.ActionInputOp(system.ActionMove).Add(c.Ops)
		return dims
	}
}
