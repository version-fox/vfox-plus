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

// Position is where an item sits in a segmented list: the outer corners of the
// list are rounded more than the corners between items.
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

// ListItemSpec describes an item of a segmented list
// (docs/en/dev/design-system.md#shape): 20 dp outer and 4 dp inner corners, the inner
// ones growing to 12 dp while the item is hovered, pressed or selected.
type ListItemSpec struct {
	// ID identifies the item among its siblings between builds.
	ID any
	// Leading builds what is shown before the text, in the colour it is given.
	Leading func(foreground ui.Color)
	// Title is the first line, in the body large style; TitleWidget replaces it
	// when set.
	Title       string
	TitleStyle  *TextStyle
	TitleWidget func(foreground ui.Color)
	// Subtitle is the second line, in the body medium style; SubtitleWidget
	// replaces it when set.
	Subtitle       string
	SubtitleLines  int
	SubtitleWidget func(foreground ui.Color)
	// Trailing builds what is shown after the text.
	Trailing func()
	// OnClick makes the item react to the pointer.
	OnClick func()
	// Selected highlights the item.
	Selected bool
	// Position is where the item sits in its list.
	Position Position
	// Padding is the padding of the content in dp, as CSS takes it (top, right,
	// bottom, left); 12 and 16 by default.
	Padding []float32
	// Wrap moves the trailing content below the text, for narrow items.
	Wrap bool
	// Name names the item for assistive technology; the title by default.
	Name string
}

// sidePadding is the padding in dp at the left and at the right of an item, from a padding as CSS takes it: top, right,
// bottom and left, or the first two, or the first three, or the first alone.
func sidePadding(pad []float32) (left, right float32) {
	switch len(pad) {
	case 1:
		return pad[0], pad[0]
	case 4:
		return pad[3], pad[1]
	default:
		return pad[1], pad[1]
	}
}

// ListItem is an item of a segmented list.
func ListItem(c Ctx, it ListItemSpec) ui.Element {
	cs := c.Colors()
	interactive := it.OnClick != nil
	background, foreground := cs.SurfaceContainer, cs.OnSurface
	secondary := cs.OnSurfaceVariant
	iconColor := cs.OnSurfaceVariant
	if it.Selected {
		background, foreground = cs.SecondaryContainer, cs.OnSecondaryContainer
		secondary, iconColor = cs.OnSecondaryContainer, cs.OnSecondaryContainer
	}
	name := it.Name
	if name == "" {
		name = it.Title
	}
	// The widgets in the item are said to be in it by its title; the name that
	// assistive technology reads may say more, as the status of a task does.
	scope := it.Title
	if scope == "" {
		scope = name
	}
	var (
		el   ui.Element
		in   interaction
		node *Node
	)
	if interactive {
		el, in, node = press(c, pressSpec{
			id: it.ID, role: RoleListItem, name: name, selected: it.Selected, onClick: it.OnClick,
		})
	} else {
		if it.ID != nil {
			c.Key(it.ID)
		}
		el = ui.Row(c.Context)
		node = c.Env.Reg.Add(Node{Role: RoleListItem, Name: name, Selected: it.Selected})
	}
	node.Position, node.Holds = it.Position, scope
	active := el.AnimateTo("active", Flag(interactive && (in.hovered || in.pressed) || it.Selected),
		c.Env.Theme.Motion.FastSpatial.Anim())
	outer := CornerLargeInc
	inner := min(max(Lerp(CornerExtraSmall, CornerMedium, active), 0), outer)
	top, bottom := inner, inner
	if it.Position == PosFirst || it.Position == PosSingle {
		top = outer
	}
	if it.Position == PosLast || it.Position == PosSingle {
		bottom = outer
	}
	radii := Radii{top, top, bottom, bottom}
	pad := it.Padding
	if len(pad) == 0 {
		pad = []float32{12, 16}
	}
	node.PadLeft, node.PadRight = sidePadding(pad)
	el.Padding(pad...).Gap(16).Background(background).TextColor(foreground)
	radii.Apply(el)
	var layer float32
	if interactive {
		layer = stateLayer(c, el, in)
	}
	paintPress(c, el, node, radii, layer, foreground, in.keyboard)
	text := func() ui.Element {
		return ui.Column(c.Context).Grow(1).Shrink(1).AlignItems(ui.Start).Children(func() {
			switch {
			case it.TitleWidget != nil:
				it.TitleWidget(foreground)
			case it.Title != "":
				style := Type(BodyLarge)
				if it.TitleStyle != nil {
					style = *it.TitleStyle
				}
				Label(c, it.Title, style, foreground)
			}
			switch {
			case it.SubtitleWidget != nil:
				it.SubtitleWidget(secondary)
			case it.Subtitle != "":
				Label(c, it.Subtitle, Type(BodyMedium), secondary, Lines(it.SubtitleLines))
			}
		})
	}
	return el.Children(func() {
		defer c.Env.Reg.EnterScope(scope)()
		if it.Leading != nil {
			it.Leading(iconColor)
		}
		if it.Wrap && it.Trailing != nil {
			ui.Column(c.Context).Grow(1).Shrink(1).AlignItems(ui.Start).Gap(8).Children(func() {
				text()
				ui.Box(c.Context).Children(it.Trailing)
			})
			return
		}
		if it.Trailing == nil {
			text()
			return
		}
		// The control at the end sits beside the text while there is room for both, the text having its own room,
		// textBasis, to be read in; when there is not, it goes under the text, at its start. The room is what the text
		// and the control need, so it follows the language, the font and the text scale, where a width at which the item
		// is told to wrap would be the one of a text of 100 %.
		ui.Row(c.Context).Grow(1).Shrink(1).Wrap().Gap(16).GapY(8).AlignItems(ui.Center).Children(func() {
			text().Basis(c.textBasis(it))
			it.Trailing()
		})
	})
}

// textRoom is the width in ems of the title of an item that a control beside it leaves to the text at the least.
const textRoom = 6

// textBasis is the width in dp that the text of the item asks for before a control may sit beside it: the width of its
// longest line when that is less than textRoom ems, and textRoom ems otherwise; a text that is built by a widget asks for
// textRoom ems.
func (c Ctx) textBasis(it ListItemSpec) float32 {
	style := Type(BodyLarge)
	if it.TitleStyle != nil {
		style = *it.TitleStyle
	}
	scale := c.Env.Theme.TextScale
	room := textRoom * style.Size * scale
	if it.TitleWidget != nil || it.SubtitleWidget != nil {
		return room
	}
	natural := float32(0)
	for _, line := range []struct {
		text  string
		style TextStyle
	}{{it.Title, style}, {it.Subtitle, Type(BodyMedium)}} {
		if line.text == "" {
			continue
		}
		width, _ := c.MeasureText(0, ui.Span{Text: line.text, Size: line.style.Size * scale, Weight: line.style.Weight})
		natural = max(natural, width)
	}
	return min(room, natural)
}

// ListGroup lays out items as a segmented list: the first and the last have
// rounded outer corners, and the items are 2 dp apart.
func ListGroup(c Ctx, items ...ListItemSpec) ui.Element {
	return ui.Column(c.Context).Gap(2).Children(func() {
		for i, item := range items {
			item.Position = PositionOf(i, len(items))
			ListItem(c, item)
		}
	})
}
