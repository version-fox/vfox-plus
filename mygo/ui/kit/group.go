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

// Segment is a button of a ButtonGroup.
type Segment struct {
	// Value identifies the segment.
	Value string
	// Label is the text of the button.
	Label string
	// Icon is shown before the label, if any.
	Icon string
	// Tooltip describes the segment.
	Tooltip string
}

// ButtonGroupSpec describes a connected button group.
type ButtonGroupSpec struct {
	// ID identifies the group among its siblings between builds.
	ID string
	// Segments are the buttons.
	Segments []Segment
	// Selected is the value of the selected segment.
	Selected string
	// OnSelect runs with the value of a segment that the user chose.
	OnSelect func(value string)
	// Expand makes the group as wide as its space, its buttons sharing it.
	Expand bool
}

// ButtonGroup is a connected button group with a single selection
// (docs/design-system.md#components): the outer corners are fully round, the
// inner ones 8 dp; the selected button becomes fully round, and a button that is
// pressed pinches its inner corners to 4 dp.
func ButtonGroup(c Ctx, g ButtonGroupSpec) ui.Element {
	if hasKey(g.ID) {
		c.Key(g.ID)
	}
	row := ui.Row(c.Context).Gap(2).FocusGroup(ui.Horizontal)
	return row.Children(func() {
		for i, segment := range g.Segments {
			connectedButton(c, segment, segment.Value == g.Selected, i == 0, i == len(g.Segments)-1, g)
		}
	})
}

func connectedButton(c Ctx, s Segment, selected, first, last bool, g ButtonGroupSpec) {
	cs := c.Colors()
	container, content := cs.SecondaryContainer, cs.OnSecondaryContainer
	if selected {
		container, content = cs.Primary, cs.OnPrimary
	}
	inert := g.OnSelect == nil
	if inert {
		container = cs.OnSurface.Alpha(0.1)
	}
	el, in, node := press(c, pressSpec{
		id: s.Value, role: RoleButton, name: s.Label, tooltip: s.Tooltip, selected: selected, disabled: inert,
		onClick: func() { g.OnSelect(s.Value) },
	})
	motion := c.Env.Theme.Motion.FastSpatial.Anim()
	selection := el.AnimateTo("selection", Flag(selected), motion)
	pressed := el.AnimateTo("shape", Flag(in.pressed), motion)
	const full = buttonHeight / 2
	inner := min(max(Lerp(Lerp(CornerSmall, full, selection), CornerExtraSmall, pressed), 0), full)
	left, right := inner, inner
	if first {
		left = full
	}
	if last {
		right = full
	}
	radii := Radii{left, right, right, left}
	el.Height(buttonHeight).MinWidth(48).PaddingX(16).Gap(8).Background(container).TextColor(content)
	radii.Apply(el)
	if g.Expand {
		el.Grow(1).Basis(0)
	}
	paintPress(c, el, node, radii, stateLayer(c, el, in), content, in.keyboard)
	el.Children(func() {
		if s.Icon != "" {
			Icon(c, s.Icon, buttonIcon, content, 0)
		}
		Line(c, s.Label, Type(LabelLarge), content).Shrink(1)
	})
}

// SplitButtonSpec describes a split button.
type SplitButtonSpec struct {
	// ID identifies the button among its siblings between builds.
	ID string
	// Label is the text of the action.
	Label string
	// Icon is the icon of the action, if any.
	Icon string
	// OnClick runs the action.
	OnClick func()
	// MenuTooltip names the trailing button.
	MenuTooltip string
	// Items are the entries of the menu.
	Items []MenuItem
	// Disabled makes both buttons inert.
	Disabled bool
}

// SplitButton is a button with an action and a menu next to it. The trailing
// button, which opens the menu, becomes a circle while the menu is open.
func SplitButton(c Ctx, b SplitButtonSpec) ui.Element {
	cs := c.Colors()
	container, content := cs.SecondaryContainer, cs.OnSecondaryContainer
	if b.Disabled {
		container, content = cs.OnSurface.Alpha(0.1), cs.OnSurface.Alpha(0.38)
	}
	motion := c.Env.Theme.Motion.FastSpatial.Anim()
	const (
		full         = buttonHeight / 2
		inner        = CornerExtraSmall
		pressedInner = CornerSmall
	)
	if hasKey(b.ID) {
		c.Key(b.ID)
	}
	return ui.Row(c.Context).Gap(2).Children(func() {
		lead, in, node := press(c, pressSpec{
			id: "action", role: RoleButton, name: b.Label, disabled: b.Disabled, onClick: b.OnClick,
		})
		pressed := lead.AnimateTo("shape", Flag(in.pressed), motion)
		radii := Radii{full, Lerp(inner, pressedInner, pressed), Lerp(inner, pressedInner, pressed), full}
		lead.Height(buttonHeight).Padding(0, 12, 0, 16).Gap(8).Background(container).TextColor(content)
		radii.Apply(lead)
		paintPress(c, lead, node, radii, stateLayer(c, lead, in), content, in.keyboard)
		lead.Children(func() {
			if b.Icon != "" {
				Icon(c, b.Icon, buttonIcon, content, 0)
			}
			Line(c, b.Label, Type(LabelLarge), content)
		})

		var open *bool
		trail, tin, tnode := press(c, pressSpec{
			id: "menu", role: RoleButton, name: b.MenuTooltip, tooltip: b.MenuTooltip, disabled: b.Disabled,
			onClick: func() { *open = !*open },
		})
		open = menuToggle(trail)
		state := trail.AnimateTo("open", Flag(*open), motion)
		tpressed := trail.AnimateTo("shape", Flag(tin.pressed), motion)
		radius := Lerp(Lerp(inner, pressedInner, tpressed), full, state)
		tradii := Radii{radius, full, full, radius}
		tnode.Selected = *open
		trail.Size(44, buttonHeight).Shrink(0).Background(container).TextColor(content)
		tradii.Apply(trail)
		paintPress(c, trail, tnode, tradii, stateLayer(c, trail, tin), content, tin.keyboard)
		trail.Children(func() {
			// The chevron turns over while the menu is open.
			Icon(c, "expandMore", 22, content, 0).Rotate(180 * state)
		})
		Menu(c, trail, open, b.Items, MenuOptions{AlignEnd: true})
	})
}
