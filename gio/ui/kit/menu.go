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
	"strconv"

	"gioui.org/io/key"
	"gioui.org/layout"
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

// menuState is the state of a menu.
type menuState struct {
	open  bool
	scale Anim
	// shown tells that the menu was open in the previous frame, so that the
	// animation starts from the closed state when it opens.
	shown bool
}

// Anchor is the key that popups use to find a widget laid out with the given
// id in this scope: the key of an IconButton or a Button with that id.
func (c Ctx) Anchor(id string) any { return c.key(id) }

// ToggleMenu opens the menu with the id, or closes it.
func ToggleMenu(c Ctx, id string) {
	st := UseID[menuState](c, id+".menu")
	st.open = !st.open
	c.Env.Services.Invalidate()
}

// MenuOpen reports whether the menu with the id is open.
func MenuOpen(c Ctx, id string) bool { return UseID[menuState](c, id+".menu").open }

// MenuOptions are the options of a menu.
type MenuOptions struct {
	// AlignEnd aligns the right edge of the menu with the right edge of its
	// anchor, for anchors at the end of a row.
	AlignEnd bool
	// MatchWidth makes the menu at least as wide as its anchor, for dropdowns.
	MatchWidth bool
}

// Menu declares the menu with the id: while it is open, it shows below the
// widget with the key anchor (see Anchor). Call it every frame, after the
// anchor was laid out.
func Menu(c Ctx, id string, anchor any, items []MenuItem, o MenuOptions) {
	st := UseID[menuState](c, id+".menu")
	if !st.open {
		st.shown = false
		st.scale.Snap(c, 0)
		return
	}
	scope := c.Scope(id + ".items")
	c.Env.RequestPopup(anchor, func(pc Ctx, rect image.Rectangle) layout.Dimensions {
		return layoutMenu(scope.withContext(pc), st, items, rect, o)
	})
}

// withContext is the scope of c with the layout context of other.
func (c Ctx) withContext(other Ctx) Ctx {
	other.scope = c.scope
	return other
}

func layoutMenu(c Ctx, st *menuState, items []MenuItem, anchor image.Rectangle, o MenuOptions) layout.Dimensions {
	theme := c.Env.Theme
	colors := theme.Colors
	dismiss := func() {
		st.open = false
		c.Env.Services.Invalidate()
	}
	reg := c.Env.Reg
	reg.SetModal(true)
	defer reg.SetModal(false)

	// Escape closes the menu, a press elsewhere, too.
	for {
		if _, ok := c.Event(key.Filter{Name: key.NameEscape}); !ok {
			break
		}
		dismiss()
	}
	scrim(c, st, dismiss)

	if !st.shown {
		st.scale.Snap(c, 0)
		st.shown = true
	}
	open := st.scale.F(c, 1, theme.Motion.DefaultSpatial)
	opacity := minf(1, maxf(0, open*1.5))

	// Measure the menu.
	maxWidth := minInt(c.DP(280), c.Env.Window.X-c.DP(16))
	list := menuColumn(c, items, dismiss)
	minWidth := c.DP(112)
	if o.MatchWidth {
		minWidth = minInt(maxInt(minWidth, anchor.Dx()), c.Env.Window.X-c.DP(16))
		maxWidth = maxInt(maxWidth, minWidth)
	}
	p := c.measure(layout.Constraints{Min: image.Pt(minWidth, 0), Max: image.Pt(maxWidth, c.Env.Window.Y)}, list)
	size := p.size()
	at := Place(c.Env.Window, anchor, size, o.AlignEnd, c.DP(8))
	// Open from the corner next to the anchor.
	scale := Lerp(0.8, 1, minf(open, 1.1))
	pivot := image.Pt(at.X, at.Y)
	if o.AlignEnd {
		pivot.X = at.X + size.X
	}
	Opacity(c.Ops, opacity, func() {
		defer scaleAbout(c, pivot, scale)()
		rect := image.Rectangle{Min: at, Max: at.Add(size)}
		radii := c.Round(CornerLarge)
		c.Elevation(rect, radii, 3)
		FillRRect(c.Ops, rect, radii, colors.SurfaceContainer)
		p.put(c, at)
	})
	return layout.Dimensions{Size: size}
}

func menuColumn(c Ctx, items []MenuItem, close func()) Widget {
	colors := c.Env.Theme.Colors
	children := make([]FlexChild, 0, len(items))
	for i, item := range items {
		children = append(children, Rigid(menuItem(item, i, close, colors)))
	}
	return Pad(Sym(0, 4), Column(FlexOpts{Cross: CrossStretch, Min: true}, children...))
}

func menuItem(item MenuItem, index int, close func(), colors Palette) Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[buttonState](c, "item"+strconv.Itoa(index))
		foreground := colors.OnSurface
		iconColor := colors.OnSurfaceVariant
		if item.Selected {
			foreground, iconColor = colors.OnSecondaryContainer, colors.OnSecondaryContainer
		}
		if item.Danger {
			foreground, iconColor = colors.Error, colors.Error
		}
		if item.Disabled {
			foreground = Alpha(colors.OnSurface, 0.38)
			iconColor = foreground
		}
		spec := pressSpec{role: RoleMenuItem, name: item.Label, enabled: !item.Disabled, onClick: func() {
			close()
			if item.OnClick != nil {
				item.OnClick()
			}
		}}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			var parts []FlexChild
			if item.Icon != "" {
				parts = append(parts, Rigid(Icon(item.Icon, 24, iconColor, 0)))
			}
			parts = append(parts, Flexible(1, Line(item.Label, Type(BodyLarge), foreground)))
			p := c.measure(layout.Constraints{Min: image.Pt(c.Constraints.Min.X, c.DP(48)), Max: c.Constraints.Max},
				Pad(Sym(12, 0), Row(FlexOpts{Gap: 12, Cross: CrossCenter}, parts...)))
			size := image.Pt(maxInt(p.size().X, c.Constraints.Min.X), maxInt(p.size().Y, c.DP(48)))
			rect := image.Rectangle{Max: size}
			if item.Selected {
				FillRRect(c.Ops, rect, Radii{}, colors.SecondaryContainer)
			}
			FillRRect(c.Ops, rect, Radii{}, st.press.stateLayer(c, in, foreground))
			p.put(c, image.Pt(0, (size.Y-p.size().Y)/2))
			return layout.Dimensions{Size: size}
		})
	}
}

// MenuButton is an icon button that opens a menu below it.
type MenuButton struct {
	// ID identifies the button between frames, within its scope.
	ID string
	// Icon is the icon of the button.
	Icon string
	// Tooltip names the button.
	Tooltip string
	// Items are the entries of the menu.
	Items []MenuItem
	// AlignEnd aligns the menu with the end of the button.
	AlignEnd bool
	// Disabled makes the button inert.
	Disabled bool
}

// W is the button as a widget.
func (m MenuButton) W() Widget {
	return func(c Ctx) layout.Dimensions {
		dims := IconButton{
			ID: m.ID, Icon: m.Icon, Tooltip: m.Tooltip, Disabled: m.Disabled,
			OnClick: func() { ToggleMenu(c, m.ID) },
		}.W()(c)
		Menu(c, m.ID, c.Anchor(m.ID), m.Items, MenuOptions{AlignEnd: m.AlignEnd})
		return dims
	}
}
