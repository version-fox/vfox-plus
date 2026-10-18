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
)

// Position is where an item sits in a segmented list: the outer corners of
// the list are rounded more than the corners between items.
type Position int

// The positions of items of a segmented list.
const (
	PosSingle Position = iota
	PosFirst
	PosMiddle
	PosLast
)

// PositionOf is the position of the item with the index in a list of count
// items.
func PositionOf(index, count int) Position {
	switch {
	case count <= 1:
		return PosSingle
	case index == 0:
		return PosFirst
	case index == count-1:
		return PosLast
	default:
		return PosMiddle
	}
}

// ListItem is an item of a segmented list (docs/design-system.md#shape):
// 20 dp outer and 4 dp inner corners, the inner ones growing to 12 dp while the
// item is hovered, pressed or selected.
type ListItem struct {
	// ID identifies the item between frames, within its scope.
	ID string
	// Leading is shown before the text.
	Leading func(foreground color.NRGBA) Widget
	// Title is the first line, in the body large style; TitleWidget replaces it
	// when set.
	Title       string
	TitleStyle  *TextStyle
	TitleWidget func(foreground color.NRGBA) Widget
	// Subtitle is the second line, in the body medium style; SubtitleWidget
	// replaces it when set.
	Subtitle       string
	SubtitleLines  int
	SubtitleWidget func(foreground color.NRGBA) Widget
	// Trailing is shown after the text.
	Trailing Widget
	// OnClick makes the item react to the pointer.
	OnClick func()
	// Selected highlights the item.
	Selected bool
	// Position is where the item sits in its list.
	Position Position
	// Padding is the padding of the content, in dp; 16 and 12 by default.
	Padding *Inset
	// WrapTrailing moves the trailing widget below the text when the item is
	// narrow.
	WrapTrailing bool
	// Tooltip names the item for the accessibility tree.
	Name string
}

type listItemState struct {
	press  pressable
	active Anim
}

// wrapWidth is the width below which a trailing widget that may wrap moves
// below the text, in dp.
const wrapWidth = 560

// W is the item as a widget.
func (it ListItem) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[listItemState](c, it.ID)
		theme := c.Env.Theme
		colors := theme.Colors
		interactive := it.OnClick != nil
		background, foreground := colors.SurfaceContainer, colors.OnSurface
		secondary, iconColor := colors.OnSurfaceVariant, colors.OnSurfaceVariant
		if it.Selected {
			background, foreground = colors.SecondaryContainer, colors.OnSecondaryContainer
			secondary, iconColor = colors.OnSecondaryContainer, colors.OnSecondaryContainer
		}
		name := it.Name
		if name == "" {
			name = it.Title
		}
		spec := pressSpec{role: RoleListItem, name: name, enabled: interactive, selected: it.Selected, onClick: it.OnClick}
		draw := func(c Ctx, in interaction) layout.Dimensions {
			pad := Sym(16, 12)
			if it.Padding != nil {
				pad = *it.Padding
			}
			text := it.textColumn(foreground, secondary)
			width := c.Constraints.Max.X
			wrap := it.WrapTrailing && it.Trailing != nil && width < c.DP(wrapWidth)
			var row []FlexChild
			if it.Leading != nil {
				row = append(row, Rigid(it.Leading(iconColor)), Rigid(Space(16, 0)))
			}
			if wrap {
				row = append(row, Expanded(1, Column(FlexOpts{Cross: CrossStart, Min: true},
					Rigid(text), Rigid(Space(0, 8)), Rigid(it.Trailing))))
			} else {
				row = append(row, Expanded(1, text))
				if it.Trailing != nil {
					row = append(row, Rigid(Space(16, 0)), Rigid(it.Trailing))
				}
			}
			body := Pad(pad, Row(FlexOpts{Cross: CrossCenter}, row...))
			p := c.measure(layout.Constraints{Min: image.Pt(c.Constraints.Max.X, 0), Max: c.Constraints.Max}, body)
			size := image.Pt(maxInt(p.size().X, c.Constraints.Min.X), p.size().Y)
			active := float32(st.active.To(c, boolFloat(interactive && (in.hovered || in.pressed) || it.Selected), theme.Motion.FastSpatial))
			outer := float32(c.DP(CornerLargeInc))
			inner := minf(maxf(Lerp(float32(c.DP(CornerExtraSmall)), float32(c.DP(CornerMedium)), active), 0), outer)
			top, bottom := inner, inner
			if it.Position == PosFirst || it.Position == PosSingle {
				top = outer
			}
			if it.Position == PosLast || it.Position == PosSingle {
				bottom = outer
			}
			rect := image.Rectangle{Max: size}
			r := Radii{TopLeft: top, TopRight: top, BottomRight: bottom, BottomLeft: bottom}
			FillRRect(c.Ops, rect, r, background)
			if interactive {
				FillRRect(c.Ops, rect, r, st.press.stateLayer(c, in, foreground))
			}
			focusRing(c, in, rect, r)
			p.put(c, image.Point{})
			return layout.Dimensions{Size: size}
		}
		if interactive {
			return st.press.layout(c, spec, draw)
		}
		dims := draw(c, interaction{})
		c.Env.Reg.Add(Node{Role: RoleListItem, Name: name, Rect: image.Rectangle{Max: dims.Size}, Selected: it.Selected})
		return dims
	}
}

func (it ListItem) textColumn(foreground, secondary color.NRGBA) Widget {
	var children []FlexChild
	switch {
	case it.TitleWidget != nil:
		children = append(children, Rigid(it.TitleWidget(foreground)))
	case it.Title != "":
		style := Type(BodyLarge)
		if it.TitleStyle != nil {
			style = *it.TitleStyle
		}
		children = append(children, Rigid(Label(it.Title, style, foreground)))
	}
	switch {
	case it.SubtitleWidget != nil:
		children = append(children, Rigid(it.SubtitleWidget(secondary)))
	case it.Subtitle != "":
		lines := it.SubtitleLines
		children = append(children, Rigid(Label(it.Subtitle, Type(BodyMedium), secondary, Lines(lines))))
	}
	return Column(FlexOpts{Cross: CrossStart, Min: true}, children...)
}

// ListGroup lays out items as a segmented list: the first and the last have
// rounded outer corners, and the items are 2 dp apart.
func ListGroup(items ...ListItem) Widget {
	children := make([]FlexChild, 0, len(items))
	for i, item := range items {
		item.Position = PositionOf(i, len(items))
		children = append(children, Rigid(item.W()))
	}
	return Column(FlexOpts{Gap: 2, Cross: CrossStretch}, children...)
}
