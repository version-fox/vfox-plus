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

	"gioui.org/layout"
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

// ButtonGroup is a connected button group with a single selection
// (docs/design-system.md#components): the outer corners are fully round, the
// inner ones 8 dp; the selected button becomes fully round, and a button that
// is pressed pinches its inner corners to 4 dp.
type ButtonGroup struct {
	// ID identifies the group between frames, within its scope.
	ID string
	// Segments are the buttons.
	Segments []Segment
	// Selected is the value of the selected segment.
	Selected string
	// OnSelect runs with the value of a segment that the user chose.
	OnSelect func(value string)
	// Expand makes the group as wide as its space.
	Expand bool
}

// W is the group as a widget.
func (g ButtonGroup) W() Widget {
	return func(c Ctx) layout.Dimensions {
		bounded := !unbounded(c.Constraints.Max.X)
		children := make([]FlexChild, 0, len(g.Segments))
		for i, segment := range g.Segments {
			w := connectedButton(g.ID, segment, segment.Value == g.Selected, i == 0, i == len(g.Segments)-1, g.OnSelect)
			switch {
			case g.Expand:
				children = append(children, Expanded(1, w))
			case bounded:
				children = append(children, Flexible(1, w))
			default:
				children = append(children, Rigid(w))
			}
		}
		return Row(FlexOpts{Gap: 2, Min: !g.Expand, Cross: CrossCenter}, children...)(c)
	}
}

func connectedButton(group string, s Segment, selected, first, last bool, onSelect func(string)) Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[segmentState](c, group+"/"+s.Value)
		theme := c.Env.Theme
		colors := theme.Colors
		container, content := colors.SecondaryContainer, colors.OnSecondaryContainer
		if selected {
			container, content = colors.Primary, colors.OnPrimary
		}
		spec := pressSpec{
			role: RoleButton, name: s.Label, tooltip: s.Tooltip, selected: selected, enabled: onSelect != nil,
			onClick: func() { onSelect(s.Value) },
		}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			var parts []FlexChild
			if s.Icon != "" {
				parts = append(parts, Rigid(Icon(s.Icon, 20, content, 0)))
			}
			parts = append(parts, Flexible(1, Line(s.Label, Type(LabelLarge), content)))
			body := Pad(Sym(16, 0), Row(FlexOpts{Gap: 8, Cross: CrossCenter, Min: true}, parts...))
			p := c.measure(layout.Constraints{Min: image.Pt(maxInt(c.DP(48), c.Constraints.Min.X), c.DP(40)), Max: c.Constraints.Max}, body)
			size := image.Pt(maxInt(p.size().X, c.Constraints.Min.X), maxInt(p.size().Y, c.DP(40)))
			full := float32(size.Y) / 2
			selection := float32(st.selection.To(c, boolFloat(selected), theme.Motion.FastSpatial))
			press := float32(st.shape.To(c, boolFloat(in.pressed), theme.Motion.FastSpatial))
			inner := Lerp(Lerp(float32(c.DP(CornerSmall)), full, selection), float32(c.DP(CornerExtraSmall)), press)
			inner = minf(maxf(inner, 0), full)
			left, right := inner, inner
			if first {
				left = full
			}
			if last {
				right = full
			}
			rect := image.Rectangle{Max: size}
			r := Radii{TopLeft: left, TopRight: right, BottomRight: right, BottomLeft: left}
			if onSelect == nil {
				container = Alpha(colors.OnSurface, 0.1)
			}
			FillRRect(c.Ops, rect, r, container)
			FillRRect(c.Ops, rect, r, st.press.stateLayer(c, in, content))
			focusRing(c, in, rect, r)
			p.put(c, image.Pt((size.X-p.size().X)/2, (size.Y-p.size().Y)/2))
			return layout.Dimensions{Size: size}
		})
	}
}

type segmentState struct {
	press     pressable
	selection Anim
	shape     Anim
}

// SplitButton is a button with an action and a menu next to it. The trailing
// button, which opens the menu, becomes a circle while the menu is open.
type SplitButton struct {
	// ID identifies the button between frames, within its scope.
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

// W is the button as a widget.
func (b SplitButton) W() Widget {
	return func(c Ctx) layout.Dimensions {
		theme := c.Env.Theme
		colors := theme.Colors
		container, content := colors.SecondaryContainer, colors.OnSecondaryContainer
		if b.Disabled {
			container, content = Alpha(colors.OnSurface, 0.1), Alpha(colors.OnSurface, 0.38)
		}
		menuID := b.ID + ".menu"
		open := MenuOpen(c, menuID)
		lead := UseID[segmentState](c, b.ID+"/lead")
		trail := UseID[segmentState](c, b.ID+"/trail")
		height := c.DP(40)
		full := float32(height) / 2
		inner := float32(c.DP(CornerExtraSmall))
		pressedInner := float32(c.DP(CornerSmall))

		leading := func(c Ctx) layout.Dimensions {
			spec := pressSpec{role: RoleButton, name: b.Label, enabled: !b.Disabled, onClick: b.OnClick}
			return lead.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
				var parts []FlexChild
				if b.Icon != "" {
					parts = append(parts, Rigid(Icon(b.Icon, 20, content, 0)))
				}
				parts = append(parts, Rigid(Line(b.Label, Type(LabelLarge), content)))
				p := c.measure(loose(c.Constraints.Max), Pad(Inset{16, 0, 12, 0}, Row(FlexOpts{Gap: 8, Cross: CrossCenter, Min: true}, parts...)))
				size := image.Pt(p.size().X, height)
				press := float32(lead.shape.To(c, boolFloat(in.pressed), theme.Motion.FastSpatial))
				r := Radii{TopLeft: full, BottomLeft: full, TopRight: Lerp(inner, pressedInner, press), BottomRight: Lerp(inner, pressedInner, press)}
				rect := image.Rectangle{Max: size}
				FillRRect(c.Ops, rect, r, container)
				FillRRect(c.Ops, rect, r, lead.press.stateLayer(c, in, content))
				focusRing(c, in, rect, r)
				p.put(c, image.Pt(0, (height-p.size().Y)/2))
				return layout.Dimensions{Size: size}
			})
		}
		trailing := func(c Ctx) layout.Dimensions {
			spec := pressSpec{
				role: RoleButton, name: b.MenuTooltip, tooltip: b.MenuTooltip, enabled: !b.Disabled,
				key: c.key(b.ID + "/menubutton"), selected: open,
				onClick: func() { ToggleMenu(c, menuID) },
			}
			return trail.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
				size := image.Pt(c.DP(44), height)
				state := float32(trail.selection.To(c, boolFloat(open), theme.Motion.FastSpatial))
				press := float32(trail.shape.To(c, boolFloat(in.pressed), theme.Motion.FastSpatial))
				radius := Lerp(Lerp(inner, pressedInner, press), full, state)
				r := Radii{TopLeft: radius, BottomLeft: radius, TopRight: full, BottomRight: full}
				rect := image.Rectangle{Max: size}
				FillRRect(c.Ops, rect, r, container)
				FillRRect(c.Ops, rect, r, trail.press.stateLayer(c, in, content))
				focusRing(c, in, rect, r)
				// The chevron turns over while the menu is open.
				icon := c.measure(loose(size), Icon("expandMore", 22, content, 0))
				pivot := image.Pt(size.X/2, size.Y/2)
				pop := rotateAbout(c, pivot, float64(state)*3.141592653589793)
				icon.put(c, image.Pt((size.X-icon.size().X)/2, (size.Y-icon.size().Y)/2))
				pop()
				return layout.Dimensions{Size: size}
			})
		}
		dims := Row(FlexOpts{Gap: 2, Cross: CrossCenter, Min: true}, Rigid(leading), Rigid(trailing))(c)
		Menu(c, menuID, c.Anchor(b.ID+"/menubutton"), b.Items, MenuOptions{})
		return dims
	}
}
