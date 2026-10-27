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

// ButtonKind is the style of a button.
type ButtonKind int

// The styles of buttons.
const (
	// FilledButton is the high emphasis button.
	FilledButton ButtonKind = iota
	// TonalButton is the medium emphasis button.
	TonalButton
	// OutlinedButton is a button with a border.
	OutlinedButton
	// TextButton is the low emphasis button.
	TextButton
)

// ButtonSpec describes a button.
type ButtonSpec struct {
	// ID identifies the button among its siblings between builds; needed where
	// the buttons of a collection come and go.
	ID string
	// Kind is the style of the button.
	Kind ButtonKind
	// Label is the text of the button.
	Label string
	// Icon is the icon before the label, if any.
	Icon string
	// OnClick runs when the button is clicked.
	OnClick func()
	// Disabled makes the button inert.
	Disabled bool
	// Danger makes a filled button the colour of errors, for destructive
	// actions.
	Danger bool
	// Content overrides the colour of the content when its alpha is not zero.
	Content ui.Color
}

const (
	buttonHeight = 40
	buttonIcon   = 20
)

// colors are the colours of a style: container, content and border.
func (b ButtonSpec) colors(cs *Palette) (container, content, border ui.Color) {
	switch b.Kind {
	case TonalButton:
		container, content = cs.SecondaryContainer, cs.OnSecondaryContainer
	case OutlinedButton:
		content, border = cs.Primary, cs.Outline
	case TextButton:
		content = cs.Primary
	default:
		container, content = cs.Primary, cs.OnPrimary
		if b.Danger {
			container, content = cs.Error, cs.OnError
		}
	}
	if b.Content.A != 0 {
		content = b.Content
	}
	if b.Disabled {
		content = cs.OnSurface.Alpha(0.38)
		if b.Kind == FilledButton || b.Kind == TonalButton {
			container = cs.OnSurface.Alpha(0.12)
		}
		if b.Kind == OutlinedButton {
			border = cs.OnSurface.Alpha(0.12)
		}
	}
	return container, content, border
}

// Button is a Material 3 Expressive button: round at rest, and squarer while it
// is pressed (docs/en/dev/design-system.md#shape-morphing-on-interaction).
func Button(c Ctx, b ButtonSpec) ui.Element {
	container, content, border := b.colors(c.Colors())
	el, in, node := press(c, pressSpec{
		id: b.ID, role: RoleButton, name: b.Label, disabled: b.Disabled, onClick: b.OnClick,
	})
	pressed := el.AnimateTo("shape", Flag(in.pressed), c.Env.Theme.Motion.FastSpatial.Anim())
	radius := Lerp(buttonHeight/2, CornerSmall, pressed)
	el.Height(buttonHeight).MinWidth(48).PaddingX(16).Gap(8).Radius(radius).Background(container).TextColor(content)
	if border.A != 0 {
		el.Border(1, border)
	}
	paintPress(c, el, node, Uniform(radius), stateLayer(c, el, in), content, in.keyboard)
	return el.Children(func() {
		if b.Icon != "" {
			Icon(c, b.Icon, buttonIcon, content, 0)
		}
		if b.Label != "" {
			Line(c, b.Label, Type(LabelLarge), content)
		}
	})
}

// IconButtonKind is the style of an icon button.
type IconButtonKind int

// The styles of icon buttons.
const (
	// StandardIconButton has no container.
	StandardIconButton IconButtonKind = iota
	// FilledIconButton has a primary container.
	FilledIconButton
	// TonalIconButton has a secondary container.
	TonalIconButton
)

// IconButtonSpec describes an icon button.
type IconButtonSpec struct {
	// ID identifies the button among its siblings between builds.
	ID string
	// Kind is the style.
	Kind IconButtonKind
	// Icon is the icon.
	Icon string
	// Tooltip names the button, and shows when the pointer rests on it. It is
	// the accessible name of the button.
	Tooltip string
	// OnClick runs when the button is clicked.
	OnClick func()
	// Disabled makes the button inert.
	Disabled bool
	// Selected tells that the button is a toggle that is on: its icon is
	// filled.
	Selected bool
	// Color overrides the colour of the icon when its alpha is not zero.
	Color ui.Color
	// Size is the diameter of the container in dp; 40 by default. The
	// button takes at least 48 dp for the pointer.
	Size float32
	// Rotation turns the icon, in degrees: the chevron of a button that
	// expands something.
	Rotation float32
	// Menu makes the button open a menu of entries below it, instead of
	// running OnClick; MenuAlignEnd aligns the menu with the end of the button.
	Menu         []MenuItem
	MenuAlignEnd bool
}

// IconButton is a button that shows only an icon. Its tooltip is its
// accessible name.
func IconButton(c Ctx, b IconButtonSpec) ui.Element {
	cs := c.Colors()
	container, content := ui.Transparent, cs.OnSurfaceVariant
	switch b.Kind {
	case FilledIconButton:
		container, content = cs.Primary, cs.OnPrimary
	case TonalIconButton:
		container, content = cs.SecondaryContainer, cs.OnSecondaryContainer
	}
	if b.Color.A != 0 {
		content = b.Color
	}
	if b.Disabled {
		content = cs.OnSurface.Alpha(0.38)
		if container.A != 0 {
			container = cs.OnSurface.Alpha(0.12)
		}
	}
	diameter := b.Size
	if diameter == 0 {
		diameter = 40
	}
	box := max(48, diameter)
	onClick := b.OnClick
	var open *bool
	if len(b.Menu) > 0 {
		onClick = func() { *open = !*open }
	}
	el, in, node := press(c, pressSpec{
		id: b.ID, role: RoleButton, name: b.Tooltip, tooltip: b.Tooltip, selected: b.Selected,
		disabled: b.Disabled, onClick: onClick,
	})
	if len(b.Menu) > 0 {
		open = menuToggle(el)
	}
	pressed := el.AnimateTo("shape", Flag(in.pressed), c.Env.Theme.Motion.FastSpatial.Anim())
	radius := Lerp(diameter/2, CornerMedium, pressed)
	layer := stateLayer(c, el, in)
	el.Size(box, box).Shrink(0)
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		node.Rect = r
		disc := ui.Rect{X: r.X + (r.W-diameter)/2, Y: r.Y + (r.H-diameter)/2, W: diameter, H: diameter}
		fillRRect(p, disc, Uniform(radius), container)
		fillRRect(p, disc, Uniform(radius), content.Alpha(layer))
	})
	if in.keyboard {
		el.DrawOver(func(p *ui.Painter, r ui.Rect) {
			disc := ui.Rect{X: r.X + (r.W-diameter)/2 - 2, Y: r.Y + (r.H-diameter)/2 - 2, W: diameter + 4, H: diameter + 4}
			strokeRRect(p, disc, Uniform(radius+2), 3, cs.Secondary)
		})
	}
	el.Children(func() {
		icon := Icon(c, b.Icon, 24, content, Flag(b.Selected))
		if b.Rotation != 0 {
			icon.Rotate(b.Rotation)
		}
	})
	if open != nil {
		Menu(c, el, open, b.Menu, MenuOptions{AlignEnd: b.MenuAlignEnd})
	}
	return el
}

// FABSpec describes a floating action button.
type FABSpec struct {
	// ID identifies the button among its siblings between builds.
	ID string
	// Icon is the icon.
	Icon string
	// Label is shown next to the icon of an extended button, as much of it as
	// Expansion (0 to 1) reveals.
	Label string
	// Expansion is how far the button is extended: 0 is the icon alone.
	Expansion float32
	// Tooltip names the button when it shows only its icon.
	Tooltip string
	// OnClick runs when the button is clicked.
	OnClick func()
	// Disabled makes the button inert.
	Disabled bool
}

// FAB is a floating action button: a primary container with an icon, and a label
// when it is extended.
func FAB(c Ctx, b FABSpec) ui.Element {
	cs := c.Colors()
	tip := b.Tooltip
	if b.Expansion < 0.5 && b.Label != "" {
		tip = b.Label
	}
	el, in, node := press(c, pressSpec{
		id: b.ID, role: RoleButton, name: b.Label, tooltip: tip, disabled: b.Disabled, onClick: b.OnClick,
	})
	expansion := clamp01(b.Expansion)
	labelWidth, _ := c.MeasureText(0, ui.Span{Text: b.Label, Weight: Type(TitleSmall).Weight, Size: Type(TitleSmall).Size * c.Env.Theme.TextScale})
	el.Height(56).PaddingX(16).Radius(CornerLarge).Background(cs.PrimaryContainer).TextColor(cs.OnPrimaryContainer)
	el.Justify(ui.Start).Shrink(0)
	Elevation(c, el, 1)
	paintPress(c, el, node, Uniform(CornerLarge), stateLayer(c, el, in), cs.OnPrimaryContainer, in.keyboard)
	return el.Children(func() {
		Icon(c, b.Icon, 24, cs.OnPrimaryContainer, 0)
		if b.Label != "" && expansion > 0 {
			room := (labelWidth + 12 + 4) * expansion
			ui.Box(c.Context).Width(room).Shrink(0).ClipX().Children(func() {
				Line(c, b.Label, Type(TitleSmall), cs.OnPrimaryContainer).MarginStart(12).
					Opacity(clamp01((expansion - 0.5) * 2)).Shrink(0).NoWrap()
			})
		}
	})
}

// TagSpec describes a tag.
type TagSpec struct {
	// Label is the text.
	Label string
	// Icon is an icon before the text, if any.
	Icon string
	// Emphasized uses the tertiary container instead of the neutral one.
	Emphasized bool
}

// Tag is a small label in a pill: the current version, a note such as LTS.
func Tag(c Ctx, t TagSpec) ui.Element {
	cs := c.Colors()
	background, foreground := cs.SurfaceContainerHighest, cs.OnSurfaceVariant
	if t.Emphasized {
		background, foreground = cs.TertiaryContainer, cs.OnTertiaryContainer
	}
	return Row(c).Padding(3, 10).Gap(4).Radius(CornerFull).Background(background).Shrink(0).Children(func() {
		if t.Icon != "" {
			Icon(c, t.Icon, 16, foreground, 1)
		}
		Line(c, t.Label, Type(LabelMedium), foreground).Shrink(1)
	})
}
