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
)

// labelOptions are the options of a label.
type labelOptions struct {
	lines      int
	center     bool
	mono       bool
	selectable bool
	scaleCap   float32
	breakable  bool
}

// LabelOption changes how a label lays out its text.
type LabelOption func(*labelOptions)

// Lines limits the label to a number of lines, and ends the last with an
// ellipsis.
func Lines(n int) LabelOption { return func(o *labelOptions) { o.lines = n } }

// Centered centres the lines of the label.
func Centered() LabelOption { return func(o *labelOptions) { o.center = true } }

// Monospace draws the label in the monospaced font.
func Monospace() LabelOption { return func(o *labelOptions) { o.mono = true } }

// ScaleCap keeps the label from growing with the text scale of the system beyond that scale: what has a room that does not
// grow with the text, such as the label of a destination of the navigation bar, is read at 130 % at the most.
func ScaleCap(scale float32) LabelOption { return func(o *labelOptions) { o.scaleCap = scale } }

// Breakable lets a word that is longer than the room the label has break at any character, where it would stick out of what
// holds it or be cut: a name, an address, a path or a link is one word that has no space to break at. Words that can break at a
// space still do, and only a word that cannot fit a line of its own is broken. What holds the label has to leave it that room:
// a container that is a flex item has to be told MinWidth(0), as its least width is that of the longest word.
func Breakable() LabelOption { return func(o *labelOptions) { o.breakable = true } }

// Selectable lets the user select the text and copy it.
func Selectable() LabelOption { return func(o *labelOptions) { o.selectable = true } }

// Label is text in a style and a colour. It wraps at the width it is given.
func Label(c Ctx, s string, style TextStyle, col ui.Color, options ...LabelOption) ui.Element {
	var o labelOptions
	for _, option := range options {
		option(&o)
	}
	scale := c.Env.Theme.TextScale
	if o.scaleCap > 0 {
		scale = min(scale, o.scaleCap)
	}
	size := style.Size * scale
	spacing := Tracking * size
	e := ui.Text(c.Context, s).FontSize(size).FontWeight(style.Weight).
		FixedLineHeight(style.Line * scale).TextColor(col)
	if spacing != 0 {
		e.LetterSpacing(spacing)
	}
	if o.mono {
		e.Font(MonoFamilies)
	}
	if o.breakable {
		// The width that a text asks for is that of its longest word, which a box that is aligned to the start of its parent is
		// as wide as: the box is held to the width of its parent, and the line breaks inside the word.
		e.MinWidth(0).MaxWidthPercent(100)
	}
	if o.lines == 1 {
		e.SingleLine()
	} else if o.lines > 1 {
		e.MaxLines(o.lines)
	}
	if o.center {
		e.TextAlign(ui.Center)
	}
	if o.selectable {
		e.Selectable()
	}
	if c.Env.Reg.CollectText && s != "" {
		node := c.Env.Reg.Add(Node{Role: RoleText, Text: s, Size: style.Size, Weight: style.Weight, Color: col, Mono: o.mono})
		span := ui.Span{Text: s, Size: size, Weight: style.Weight, LetterSpacing: spacing}
		if o.mono {
			span.Font = MonoFamilies
		}
		switch {
		case o.lines == 1:
			node.Natural, _ = c.MeasureText(0, span)
		case o.lines > 1:
			node.Lines = o.lines
		}
		e.DrawOver(func(p *ui.Painter, r ui.Rect) {
			node.Rect, node.Seen = r, intersection(r, p.Visible())
			if node.Lines > 1 {
				node.Needs = linesNeeded(p, span, r.W)
			}
		})
	}
	return e
}

// intersection is the part of a that is in b: empty where there is none.
func intersection(a, b ui.Rect) ui.Rect {
	x, y := max(a.X, b.X), max(a.Y, b.Y)
	right, bottom := min(a.X+a.W, b.X+b.W), min(a.Y+a.H, b.Y+b.H)
	if right <= x || bottom <= y {
		return ui.Rect{}
	}
	return ui.Rect{X: x, Y: y, W: right - x, H: bottom - y}
}

// linesNeeded is how many lines the span takes when it wraps at the width: the height that it has there over the height of a
// line of the same font.
func linesNeeded(p *ui.Painter, span ui.Span, width float32) int {
	_, height := p.MeasureText(width, span)
	one := span
	one.Text = "x"
	_, line := p.MeasureText(0, one)
	if line <= 0 {
		return 0
	}
	return int(math.Round(float64(height / line)))
}

// Line is a label of a single line of text that never wraps; it ends with an
// ellipsis when it is too long.
func Line(c Ctx, s string, style TextStyle, col ui.Color, options ...LabelOption) ui.Element {
	return Label(c, s, style, col, append([]LabelOption{Lines(1)}, options...)...)
}
