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

// Button is a Material 3 Expressive button: round at rest, and squarer
// while it is pressed (docs/design-system.md#shape-morphing-on-interaction).
type Button struct {
	// ID identifies the button between frames, within its scope.
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
	// content overrides the colour of the content.
	content *colorValue
}

const (
	buttonHeight = 40
	buttonIcon   = 20
)

// buttonColors are the colours of a style: container, content and border.
func (b Button) colors(c Ctx) (container, content, border color.NRGBA) {
	container, content, border = b.stateColors(c)
	if b.content != nil {
		content = *b.content
	}
	return container, content, border
}

// colorValue is a colour.
type colorValue = color.NRGBA

func (b Button) stateColors(c Ctx) (container, content, border color.NRGBA) {
	cs := c.Env.Theme.Colors
	switch b.Kind {
	case TonalButton:
		return cs.SecondaryContainer, cs.OnSecondaryContainer, color.NRGBA{}
	case OutlinedButton:
		return color.NRGBA{}, cs.Primary, cs.Outline
	case TextButton:
		return color.NRGBA{}, cs.Primary, color.NRGBA{}
	default:
		if b.Danger {
			return cs.Error, cs.OnError, color.NRGBA{}
		}
		return cs.Primary, cs.OnPrimary, color.NRGBA{}
	}
}

// W is the button as a widget.
func (b Button) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[buttonState](c, b.ID)
		theme := c.Env.Theme
		container, content, border := b.colors(c)
		if b.Disabled {
			content = Alpha(theme.Colors.OnSurface, 0.38)
			if b.Kind != TextButton && b.Kind != OutlinedButton {
				container = Alpha(theme.Colors.OnSurface, 0.12)
			}
			border = Alpha(theme.Colors.OnSurface, 0.12)
		}
		spec := pressSpec{role: RoleButton, name: b.Label, enabled: !b.Disabled, onClick: b.OnClick}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			// The label and the icon decide the size.
			var parts []FlexChild
			if b.Icon != "" {
				parts = append(parts, Rigid(Icon(b.Icon, buttonIcon, content, 0)))
			}
			if b.Label != "" {
				parts = append(parts, Rigid(Line(b.Label, Type(LabelLarge), content)))
			}
			body := Pad(Sym(16, 0), Row(FlexOpts{Gap: 8, Cross: CrossCenter, Min: true}, parts...))
			dims := c.measure(layout.Constraints{
				Min: image.Pt(c.DP(48), c.DP(buttonHeight)),
				Max: c.Constraints.Max,
			}, func(c Ctx) layout.Dimensions {
				d := body(c)
				d.Size.Y = maxInt(d.Size.Y, c.DP(buttonHeight))
				return d
			})
			size := dims.size()
			size.Y = maxInt(size.Y, c.DP(buttonHeight))
			rect := image.Rectangle{Max: size}
			full := float32(size.Y) / 2
			pressed := st.shape.To(c, boolFloat(in.pressed), theme.Motion.FastSpatial)
			radius := float32(Lerp(full, float32(c.DP(CornerSmall)), float32(pressed)))
			radius = minf(maxf(radius, 0), full)
			r := Uniform(radius)
			FillRRect(c.Ops, rect, r, container)
			StrokeRRect(c.Ops, rect, r, float32(c.DP(1)), border)
			FillRRect(c.Ops, rect, r, st.press.stateLayer(c, in, content))
			focusRing(c, in, rect, r)
			// Centre the body vertically in a button that is taller.
			dims.put(c, image.Pt(0, (size.Y-dims.size().Y)/2))
			return layout.Dimensions{Size: size, Baseline: dims.dims.Baseline}
		})
	}
}

type buttonState struct {
	press pressable
	shape Anim
}

func boolFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
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

// IconButton is a button that shows only an icon. Its tooltip is its
// accessible name.
type IconButton struct {
	// ID identifies the button between frames, within its scope.
	ID string
	// Kind is the style.
	Kind IconButtonKind
	// Icon is the icon.
	Icon string
	// Tooltip names the button, and shows when the pointer rests on it.
	Tooltip string
	// OnClick runs when the button is clicked.
	OnClick func()
	// Disabled makes the button inert.
	Disabled bool
	// Selected tells that the button is a toggle that is on.
	Selected bool
	// Color overrides the colour of the icon.
	Color *color.NRGBA
	// Size is the diameter of the container in dp; 40 by default. The
	// button takes at least 48 dp for the pointer.
	Size float32
	// Rotation turns the icon, in radians: the chevron of a button that
	// expands something.
	Rotation float64
}

// W is the button as a widget.
func (b IconButton) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[buttonState](c, b.ID)
		theme := c.Env.Theme
		cs := theme.Colors
		container, content := color.NRGBA{}, cs.OnSurfaceVariant
		switch b.Kind {
		case FilledIconButton:
			container, content = cs.Primary, cs.OnPrimary
		case TonalIconButton:
			container, content = cs.SecondaryContainer, cs.OnSecondaryContainer
		}
		if b.Color != nil {
			content = *b.Color
		}
		if b.Disabled {
			content = Alpha(cs.OnSurface, 0.38)
			if container.A != 0 {
				container = Alpha(cs.OnSurface, 0.12)
			}
		}
		diameter := b.Size
		if diameter == 0 {
			diameter = 40
		}
		spec := pressSpec{
			role: RoleButton, name: b.Tooltip, tooltip: b.Tooltip, enabled: !b.Disabled,
			selected: b.Selected, key: c.key(b.ID), onClick: b.OnClick,
		}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			box := maxInt(c.DP(48), c.DP(diameter))
			size := c.Constraints.Constrain(image.Pt(box, box))
			d := c.DP(diameter)
			rect := image.Rect((size.X-d)/2, (size.Y-d)/2, (size.X+d)/2, (size.Y+d)/2)
			pressed := st.shape.To(c, boolFloat(in.pressed), theme.Motion.FastSpatial)
			radius := minf(maxf(float32(Lerp(float32(d)/2, float32(c.DP(CornerMedium)), float32(pressed))), 0), float32(d)/2)
			r := Uniform(radius)
			FillRRect(c.Ops, rect, r, container)
			FillRRect(c.Ops, rect, r, st.press.stateLayer(c, in, content))
			focusRing(c, in, rect, r)
			icon := c.measure(loose(image.Pt(box, box)), Icon(b.Icon, 24, content, boolFloat32(b.Selected)))
			func() {
				if b.Rotation != 0 {
					defer rotateAbout(c, image.Pt(size.X/2, size.Y/2), b.Rotation)()
				}
				icon.put(c, image.Pt((size.X-icon.size().X)/2, (size.Y-icon.size().Y)/2))
			}()
			return layout.Dimensions{Size: size}
		})
	}
}

func boolFloat32(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// FAB is a floating action button: a primary container with an icon, and
// a label when it is extended.
type FAB struct {
	// ID identifies the button between frames, within its scope.
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

// W is the button as a widget.
func (b FAB) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[buttonState](c, b.ID)
		cs := c.Env.Theme.Colors
		spec := pressSpec{
			role: RoleButton, name: b.Label, tooltip: tooltipIf(b.Expansion < 0.5, b.Label, b.Tooltip),
			enabled: !b.Disabled, key: c.key(b.ID), onClick: b.OnClick,
		}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			height := c.DP(56)
			iconBox := c.DP(24)
			labelP := c.measure(loose(image.Pt(c.DP(240), height)), Line(b.Label, Type(TitleSmall), cs.OnPrimaryContainer))
			labelWidth := int(float32(labelP.size().X+c.DP(12)+c.DP(4)) * clamp01(b.Expansion))
			width := c.DP(16)*2 + iconBox + labelWidth
			size := c.Constraints.Constrain(image.Pt(width, height))
			rect := image.Rectangle{Max: size}
			r := c.Round(CornerLarge)
			c.Elevation(rect, r, 1)
			FillRRect(c.Ops, rect, r, cs.PrimaryContainer)
			FillRRect(c.Ops, rect, r, st.press.stateLayer(c, in, cs.OnPrimaryContainer))
			focusRing(c, in, rect, r)
			icon := c.measure(loose(image.Pt(height, height)), Icon(b.Icon, 24, cs.OnPrimaryContainer, 0))
			icon.put(c, image.Pt(c.DP(16), (height-icon.size().Y)/2))
			if b.Expansion > 0.5 {
				fade := (b.Expansion - 0.5) * 2
				Opacity(c.Ops, fade, func() {
					labelP.put(c, image.Pt(c.DP(16)+iconBox+c.DP(12), (height-labelP.size().Y)/2))
				})
			}
			return layout.Dimensions{Size: size}
		})
	}
}

func tooltipIf(show bool, text, fallback string) string {
	if show {
		return text
	}
	return fallback
}

// Tag is a small label in a pill: the current version, a note such as LTS.
type Tag struct {
	// Label is the text.
	Label string
	// Icon is an icon before the text, if any.
	Icon string
	// Emphasized uses the tertiary container instead of the neutral one.
	Emphasized bool
}

// W is the tag as a widget.
func (t Tag) W() Widget {
	return func(c Ctx) layout.Dimensions {
		cs := c.Env.Theme.Colors
		background, foreground := cs.SurfaceContainerHighest, cs.OnSurfaceVariant
		if t.Emphasized {
			background, foreground = cs.TertiaryContainer, cs.OnTertiaryContainer
		}
		var parts []FlexChild
		if t.Icon != "" {
			parts = append(parts, Rigid(Icon(t.Icon, 16, foreground, 1)))
		}
		parts = append(parts, Flexible(1, Line(t.Label, Type(LabelMedium), foreground)))
		body := Pad(Inset{10, 3, 10, 3}, Row(FlexOpts{Gap: 4, Cross: CrossCenter, Min: true}, parts...))
		p := c.measure(layout.Constraints{Max: c.Constraints.Max}, body)
		size := p.size()
		FillRRect(c.Ops, image.Rectangle{Max: size}, Uniform(float32(size.Y)/2), background)
		p.put(c, image.Point{})
		return layout.Dimensions{Size: size, Baseline: p.dims.Baseline}
	}
}
