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
	// MaxWidth is the width in dp that the group has, or 0 for as much as it wants. A group that is wider than that
	// gives up what it can do without, the least visible first: 4 dp of the padding at each side of a segment (12 dp is
	// what Material 3 has), then the icons, and then 4 dp and 8 dp more. It measures the labels to know, so the fit
	// follows the language, the font and the text scale, and the page need not know how wide a group is.
	MaxWidth float32
	// Collapse makes the group a dropdown when its segments do not fit MaxWidth even when they give up all that they can:
	// the choice stays one choice, in the room that there is.
	Collapse bool
}

// segmentFit is how the segments of a group are drawn: with their icons or without, and with so much padding at each
// side.
type segmentFit struct {
	icons   bool
	padding float32
}

// segmentFits are what a group gives up to fit, from what it has at its best to the least that it can be: each is
// narrower than the one before.
var segmentFits = [...]segmentFit{
	{icons: true, padding: 16}, {icons: true, padding: 12}, {padding: 16}, {padding: 12}, {padding: 8},
}

// segmentGap is the space in dp between two segments.
const segmentGap = 2

// fit picks the first of the segmentFits in which the group is not wider than MaxWidth, or the last one when none is, and
// says whether it is.
func (g ButtonGroupSpec) fit(c Ctx) (segmentFit, bool) {
	if g.MaxWidth <= 0 || g.Expand || len(g.Segments) == 0 {
		return segmentFits[0], true
	}
	style := Type(LabelLarge)
	labels := make([]float32, len(g.Segments))
	for i, s := range g.Segments {
		labels[i], _ = c.MeasureText(0, ui.Span{Text: s.Label, Size: style.Size * c.Env.Theme.TextScale, Weight: style.Weight})
	}
	for _, fit := range segmentFits {
		width := float32(segmentGap * (len(g.Segments) - 1))
		for i, s := range g.Segments {
			inner := labels[i]
			if fit.icons && s.Icon != "" {
				inner += buttonIcon + segmentIconGap
			}
			width += max(segmentMinWidth, 2*fit.padding+inner)
		}
		if width <= g.MaxWidth {
			return fit, true
		}
	}
	return segmentFits[len(segmentFits)-1], false
}

// dropdown is the group as a dropdown, which has the choices of the segments in a menu.
func (g ButtonGroupSpec) dropdown(c Ctx) ui.Element {
	choices := make([]Choice, len(g.Segments))
	for i, s := range g.Segments {
		choices[i] = Choice{Value: s.Value, Label: s.Label}
	}
	return Dropdown(c, DropdownSpec{ID: g.ID, Value: g.Selected, Choices: choices, OnSelect: g.OnSelect, Width: g.MaxWidth, Disabled: g.OnSelect == nil})
}

// ButtonGroup is a connected button group with a single selection
// (docs/en/dev/design-system.md#components): the outer corners are fully round, the
// inner ones 8 dp; the selected button becomes fully round, and a button that is
// pressed pinches its inner corners to 4 dp.
func ButtonGroup(c Ctx, g ButtonGroupSpec) ui.Element {
	fit, fits := g.fit(c)
	if !fits && g.Collapse {
		return g.dropdown(c)
	}
	if hasKey(g.ID) {
		c.Key(g.ID)
	}
	row := ui.Row(c.Context).Gap(segmentGap).FocusGroup(ui.Horizontal)
	return row.Children(func() {
		for i, segment := range g.Segments {
			connectedButton(c, segment, segment.Value == g.Selected, i == 0, i == len(g.Segments)-1, g, fit)
		}
	})
}

// The size of a segment: the least that it is wide, and the space between its icon and its label.
const (
	segmentMinWidth = 48
	segmentIconGap  = 8
)

func connectedButton(c Ctx, s Segment, selected, first, last bool, g ButtonGroupSpec, fit segmentFit) {
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
	node.Choice = true
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
	el.Height(buttonHeight).MinWidth(segmentMinWidth).PaddingX(fit.padding).Gap(segmentIconGap).Background(container).TextColor(content)
	radii.Apply(el)
	if g.Expand {
		el.Grow(1).Basis(0)
	}
	paintPress(c, el, node, radii, stateLayer(c, el, in), content, in.keyboard)
	el.Children(func() {
		if fit.icons && s.Icon != "" {
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
