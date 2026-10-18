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

// Choice is an entry of a dropdown.
type Choice struct {
	// Value identifies the entry.
	Value string
	// Label is the text of the entry.
	Label string
}

// Dropdown is a filled field that shows the chosen entry and opens a menu of
// all of them.
type Dropdown struct {
	// ID identifies the dropdown between frames, within its scope.
	ID string
	// Value is the value of the chosen entry; none is chosen when no entry has
	// it.
	Value string
	// Choices are the entries.
	Choices []Choice
	// OnSelect runs with the value of the entry the user chose.
	OnSelect func(value string)
	// Disabled makes the dropdown inert.
	Disabled bool
	// Width is the width in dp; 240 by default.
	Width float32
}

type dropdownState struct {
	press pressable
	turn  Anim
}

// W is the dropdown as a widget.
func (d Dropdown) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[dropdownState](c, d.ID)
		theme := c.Env.Theme
		colors := theme.Colors
		menuID := d.ID + ".dropdown"
		open := MenuOpen(c, menuID)
		label := ""
		items := make([]MenuItem, len(d.Choices))
		for i, choice := range d.Choices {
			if choice.Value == d.Value {
				label = choice.Label
			}
			value := choice.Value
			items[i] = MenuItem{Label: choice.Label, Selected: value == d.Value, OnClick: func() {
				if d.OnSelect != nil && value != d.Value {
					d.OnSelect(value)
				}
			}}
		}
		spec := pressSpec{
			role: RoleButton, name: label, enabled: !d.Disabled && d.OnSelect != nil, key: c.key(d.ID),
			onClick: func() { ToggleMenu(c, menuID) },
		}
		dims := st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			width := c.DP(240)
			if d.Width > 0 {
				width = c.DP(d.Width)
			}
			size := c.Constraints.Constrain(image.Pt(width, c.DP(56)))
			rect := image.Rectangle{Max: size}
			r := c.Round(CornerMedium)
			content := colors.OnSurface
			if d.Disabled {
				content = Alpha(colors.OnSurface, 0.38)
			}
			FillRRect(c.Ops, rect, r, colors.SurfaceContainerHighest)
			if open || in.keyboard {
				StrokeRRect(c.Ops, rect, r, float32(c.DP(2)), colors.Primary)
			}
			pad := c.DP(16)
			arrow := c.measure(loose(image.Pt(c.DP(24), c.DP(24))), Icon("expandMore", 24, colors.OnSurfaceVariant, 0))
			turn := float32(st.turn.To(c, boolFloat(open), theme.Motion.FastSpatial))
			func() {
				defer rotateAbout(c, image.Pt(size.X-pad-arrow.size().X/2, size.Y/2), float64(turn)*3.141592653589793)()
				arrow.put(c, image.Pt(size.X-pad-arrow.size().X, (size.Y-arrow.size().Y)/2))
			}()
			text := c.measure(loose(image.Pt(size.X-2*pad-arrow.size().X-c.DP(8), size.Y)), Line(label, Type(BodyLarge), content))
			text.put(c, image.Pt(pad, (size.Y-text.size().Y)/2))
			return layout.Dimensions{Size: size}
		})
		Menu(c, menuID, c.Anchor(d.ID), items, MenuOptions{MatchWidth: true})
		return dims
	}
}
