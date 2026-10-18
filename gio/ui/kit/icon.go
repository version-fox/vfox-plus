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

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
)

// Icon draws an icon of the Material Symbols Rounded font, by its meaning (a
// key of shared/icons.json), in a square of size dp. fill, from 0 (outlined)
// to 1 (filled), is the fill axis of the font: the icon of the selected
// destination of the navigation is filled.
func Icon(name string, size float32, col color.NRGBA, fill float32) Widget {
	return func(c Ctx) layout.Dimensions {
		theme := c.Env.Theme
		px := c.DP(size)
		glyph := string(theme.Fonts.IconRune(name))
		label := widget.Label{MaxLines: 1, LineHeight: unit.Sp(size), LineHeightScale: 1}
		// The label is laid out in a box of exactly the size of the icon, and
		// its glyph is centred in it.
		cs := c.Constraints
		cs.Min, cs.Max = image.Point{}, image.Pt(px*4, px*4)
		inner := c
		inner.Constraints = cs
		macro := op.Record(c.Ops)
		dims := label.Layout(inner.Context, theme.Shaper(), theme.Fonts.Icon(fill), unit.Sp(size), glyph, colorOp(c.Ops, col))
		call := macro.Stop()
		offset := image.Pt((px-dims.Size.X)/2, (px-dims.Size.Y)/2+iconBaselineShift(px))
		stack := op.Offset(offset).Push(c.Ops)
		call.Add(c.Ops)
		stack.Pop()
		return layout.Dimensions{Size: c.Constraints.Constrain(image.Pt(px, px))}
	}
}

// iconBaselineShift moves the glyphs of the icon font down: they sit a little
// above the middle of their line.
func iconBaselineShift(px int) int { return 0 }
