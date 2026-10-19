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

// MenuItem is an entry of a menu.
type MenuItem struct {
	// Label is the text of the entry.
	Label string
	// Icon is the icon before it, if any.
	Icon string
	// Danger draws the entry in the colour of errors, for destructive
	// actions.
	Danger bool
	// Disabled makes the entry inert.
	Disabled bool
	// Selected marks the entry that holds the current value of a dropdown.
	Selected bool
	// OnClick runs when the entry is chosen, after the menu has closed.
	OnClick func()
}

// MenuOptions are the options of a menu.
type MenuOptions struct {
	// AlignEnd aligns the right edge of the menu with the right edge of its
	// anchor, for anchors at the end of a row.
	AlignEnd bool
	// MatchWidth makes the menu at least as wide as its anchor, for dropdowns.
	MatchWidth bool
}

// menuToggle is the state of the menu of an element: whether it is open.
func menuToggle(el ui.Element) *bool {
	return ui.Local(el, "menu", func() bool { return false })
}

// Menu shows the entries below the anchor while open is true. Build it right
// after the anchor, in the same parent: a press outside the menu, or Escape,
// closes it, and so does choosing an entry.
func Menu(c Ctx, anchor ui.Element, open *bool, items []MenuItem, o MenuOptions) {
	cs := c.Colors()
	ui.PopoverBase(c.Context, anchor, open, func(panel ui.Element) {
		reg := c.Env.Reg
		reg.SetModal(true)
		defer reg.SetModal(false)
		minWidth := float32(112)
		originX := float32(0)
		if o.MatchWidth {
			minWidth = max(minWidth, anchor.Bounds().W)
		}
		panel.Padding(8, 0).Radius(CornerLarge).Background(cs.SurfaceContainer).
			MinWidth(minWidth).MaxWidth(max(minWidth, 280)).FocusGroup(ui.Vertical)
		if o.AlignEnd {
			panel.AttachTo(anchor, ui.AnchorBottomRight, ui.AnchorTopRight).Margin(4, 0, 0, 0)
			originX = 1
		} else {
			panel.Margin(4, 0, 0, 0)
		}
		Elevation(c, panel, 3)
		popIn(c, panel, 0.8, originX, 0, c.Env.Theme.Motion.DefaultSpatial)
		panel.Children(func() {
			focused := false
			for _, item := range items {
				first := !focused && !item.Disabled && (item.Selected || !hasSelected(items))
				focused = focused || first
				menuItem(c, open, item, first)
			}
		})
	})
}

func hasSelected(items []MenuItem) bool {
	for _, item := range items {
		if item.Selected && !item.Disabled {
			return true
		}
	}
	return false
}

func menuItem(c Ctx, open *bool, item MenuItem, focus bool) {
	cs := c.Colors()
	foreground, iconColor := cs.OnSurface, cs.OnSurfaceVariant
	if item.Selected {
		foreground, iconColor = cs.OnSecondaryContainer, cs.OnSecondaryContainer
	}
	if item.Danger {
		foreground, iconColor = cs.Error, cs.Error
	}
	if item.Disabled {
		foreground = cs.OnSurface.Alpha(0.38)
		iconColor = foreground
	}
	el, in, node := press(c, pressSpec{
		id: item.Label, role: RoleMenuItem, name: item.Label, disabled: item.Disabled,
		onClick: func() {
			*open = false
			if item.OnClick != nil {
				item.OnClick()
			}
		},
	})
	el.MinHeight(48).PaddingX(12).Gap(12).Justify(ui.Start).TextColor(foreground)
	if item.Selected {
		el.Background(cs.SecondaryContainer)
	}
	if focus {
		el.AutoFocus()
	}
	paintPress(c, el, node, Radii{}, stateLayer(c, el, in), foreground, in.keyboard)
	el.Children(func() {
		if item.Icon != "" {
			Icon(c, item.Icon, 24, iconColor, 0)
		}
		Line(c, item.Label, Type(BodyLarge), foreground).Grow(1)
	})
}

// Choice is an entry of a dropdown.
type Choice struct {
	// Value identifies the entry.
	Value string
	// Label is the text of the entry.
	Label string
}

// DropdownSpec describes a dropdown.
type DropdownSpec struct {
	// ID identifies the dropdown among its siblings between builds.
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

// Dropdown is a filled field that shows the chosen entry and opens a menu of all
// of them.
func Dropdown(c Ctx, d DropdownSpec) ui.Element {
	cs := c.Colors()
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
	var open *bool
	el, in, node := press(c, pressSpec{
		id: d.ID, role: RoleButton, name: label, disabled: d.Disabled || d.OnSelect == nil,
		onClick: func() {
			if open != nil {
				*open = !*open
			}
		},
	})
	open = menuToggle(el)
	width := d.Width
	if width == 0 {
		width = 240
	}
	content := cs.OnSurface
	if d.Disabled {
		content = cs.OnSurface.Alpha(0.38)
	}
	turn := el.AnimateTo("turn", Flag(*open), c.Env.Theme.Motion.FastSpatial.Anim())
	el.Width(width).Height(56).PaddingX(16).Gap(8).Justify(ui.Start).Radius(CornerMedium).
		Background(cs.SurfaceContainerHighest).TextColor(content)
	if *open || in.keyboard {
		el.Border(2, cs.Primary)
	}
	paintPress(c, el, node, Uniform(CornerMedium), stateLayer(c, el, in), content, false)
	el.Children(func() {
		Line(c, label, Type(BodyLarge), content).Grow(1)
		Icon(c, "expandMore", 24, cs.OnSurfaceVariant, 0).Rotate(180 * turn)
	})
	Menu(c, el, open, items, MenuOptions{MatchWidth: true})
	return el
}
