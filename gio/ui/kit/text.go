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

	giofont "gioui.org/font"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
)

// colorOp records the material that draws text in a colour.
func colorOp(ops *op.Ops, col color.NRGBA) op.CallOp {
	macro := op.Record(ops)
	paint.ColorOp{Color: col}.Add(ops)
	return macro.Stop()
}

// labelOptions are the options of a label.
type labelOptions struct {
	lines int
	align text.Alignment
	mono  bool
}

// LabelOption changes how a label lays out its text.
type LabelOption func(*labelOptions)

// Lines limits the label to a number of lines, and ends the last with an
// ellipsis.
func Lines(n int) LabelOption { return func(o *labelOptions) { o.lines = n } }

// Centered centres the lines of the label.
func Centered() LabelOption { return func(o *labelOptions) { o.align = text.Middle } }

// Monospace draws the label in the monospaced font.
func Monospace() LabelOption { return func(o *labelOptions) { o.mono = true } }

// Label is text in a style and a colour. It wraps at the width it is given.
func Label(s string, style TextStyle, col color.NRGBA, options ...LabelOption) Widget {
	var o labelOptions
	for _, option := range options {
		option(&o)
	}
	return func(c Ctx) layout.Dimensions {
		theme := c.Env.Theme
		face := theme.Fonts.UI(style.Weight)
		if o.mono {
			face = theme.Fonts.Mono()
		}
		label := widget.Label{
			MaxLines:   o.lines,
			Truncator:  "…",
			Alignment:  o.align,
			LineHeight: unit.Sp(style.Line),
			// Gio scales the line height by 1.2 unless told otherwise.
			LineHeightScale: 1,
		}
		material := colorOp(c.Ops, col)
		dims := label.Layout(c.Context, theme.Shaper(), face, unit.Sp(style.Size), s, material)
		if c.Env.Reg.CollectText && s != "" {
			c.Env.Reg.Add(Node{Role: RoleText, Text: s, Rect: image.Rectangle{Max: dims.Size}})
		}
		return dims
	}
}

// Line is a label of a single line of text that never wraps; it ends with an
// ellipsis when it is too long.
func Line(s string, style TextStyle, col color.NRGBA, options ...LabelOption) Widget {
	return Label(s, style, col, append([]LabelOption{Lines(1)}, options...)...)
}

// selectableState is the state of selectable text.
type selectableState struct {
	selectable widget.Selectable
}

// SelectableText is text that the user can select and copy. id identifies it
// between frames.
func SelectableText(id string, s string, style TextStyle, col color.NRGBA, options ...LabelOption) Widget {
	var o labelOptions
	for _, option := range options {
		option(&o)
	}
	return func(c Ctx) layout.Dimensions {
		st := UseID[selectableState](c, id)
		theme := c.Env.Theme
		face := theme.Fonts.UI(style.Weight)
		if o.mono {
			face = theme.Fonts.Mono()
		}
		if st.selectable.Text() != s {
			st.selectable.SetText(s)
		}
		st.selectable.LineHeight = unit.Sp(style.Line)
		st.selectable.LineHeightScale = 1
		st.selectable.MaxLines = o.lines
		st.selectable.Alignment = o.align
		material := colorOp(c.Ops, col)
		selection := colorOp(c.Ops, Alpha(theme.Colors.Primary, 0.3))
		dims := st.selectable.Layout(c.Context, theme.Shaper(), face, unit.Sp(style.Size), material, selection)
		if c.Env.Reg.CollectText && s != "" {
			c.Env.Reg.Add(Node{Role: RoleText, Text: s, Rect: image.Rectangle{Max: dims.Size}})
		}
		return dims
	}
}

// Font is the font that the style asks for.
func (t *Theme) Font(style TextStyle) giofont.Font { return t.Fonts.UI(style.Weight) }
