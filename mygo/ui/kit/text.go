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
	"github.com/egoist/mygo/ui"
)

// labelOptions are the options of a label.
type labelOptions struct {
	lines      int
	center     bool
	mono       bool
	selectable bool
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

// Selectable lets the user select the text and copy it.
func Selectable() LabelOption { return func(o *labelOptions) { o.selectable = true } }

// Label is text in a style and a colour. It wraps at the width it is given.
func Label(c Ctx, s string, style TextStyle, col ui.Color, options ...LabelOption) ui.Element {
	var o labelOptions
	for _, option := range options {
		option(&o)
	}
	scale := c.Env.Theme.TextScale
	e := ui.Text(c.Context, s).FontSize(style.Size * scale).FontWeight(style.Weight).
		FixedLineHeight(style.Line * scale).TextColor(col)
	if o.mono {
		e.Font(MonoFamilies)
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
		node := c.Env.Reg.Add(Node{Role: RoleText, Text: s})
		e.DrawOver(func(_ *ui.Painter, r ui.Rect) { node.Rect = r })
	}
	return e
}

// Line is a label of a single line of text that never wraps; it ends with an
// ellipsis when it is too long.
func Line(c Ctx, s string, style TextStyle, col ui.Color, options ...LabelOption) ui.Element {
	return Label(c, s, style, col, append([]LabelOption{Lines(1)}, options...)...)
}
