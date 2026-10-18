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

	"github.com/version-fox/vfox-plus/gio/ui/shapes"
)

// Pane is a rounded surface that a page sits on: 28 dp corners, the content
// clipped to them.
func Pane(child Widget) Widget { return PaneColor(nil, child) }

// PaneColor is a pane of a colour other than the surface container low.
func PaneColor(col *color.NRGBA, child Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		colors := c.Env.Theme.Colors
		background := colors.SurfaceContainerLow
		if col != nil {
			background = *col
		}
		cs := c.Constraints
		size := cs.Max
		if unbounded(size.X) || unbounded(size.Y) {
			p := c.measure(cs, child)
			size = p.size()
			FillRRect(c.Ops, image.Rectangle{Max: size}, c.Round(CornerExtraLarge), background)
			p.put(c, image.Point{})
			return layout.Dimensions{Size: size}
		}
		rect := image.Rectangle{Max: size}
		r := c.Round(CornerExtraLarge)
		FillRRect(c.Ops, rect, r, background)
		defer ClipRRect(c.Ops, rect, r)()
		p := c.measure(tight(size), child)
		p.put(c, image.Point{})
		return layout.Dimensions{Size: size}
	}
}

// Card is a surface with rounded corners that may be clicked.
type Card struct {
	// ID identifies the card between frames, within its scope; needed when
	// OnClick is set.
	ID string
	// Color is the colour of the card.
	Color color.NRGBA
	// Radius is the corner radius in dp.
	Radius float32
	// OnClick makes the card react to the pointer.
	OnClick func()
	// Content is the foreground colour, for the state layer.
	Content color.NRGBA
	// Name names the card for the accessibility tree.
	Name string
	// Child is what the card holds.
	Child Widget
}

type cardState struct{ press pressable }

// W is the card as a widget.
func (cd Card) W() Widget {
	return func(c Ctx) layout.Dimensions {
		draw := func(c Ctx, in interaction, st *cardState) layout.Dimensions {
			p := c.measure(c.Constraints, cd.Child)
			size := p.size()
			rect := image.Rectangle{Max: size}
			r := c.Round(cd.Radius)
			FillRRect(c.Ops, rect, r, cd.Color)
			func() {
				defer ClipRRect(c.Ops, rect, r)()
				p.put(c, image.Point{})
				if cd.OnClick != nil {
					FillRRect(c.Ops, rect, r, st.press.stateLayer(c, in, cd.Content))
				}
			}()
			focusRing(c, in, rect, r)
			return layout.Dimensions{Size: size}
		}
		if cd.OnClick == nil {
			return draw(c, interaction{}, &cardState{})
		}
		st := UseID[cardState](c, cd.ID)
		spec := pressSpec{role: RoleButton, name: cd.Name, enabled: true, onClick: cd.OnClick}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions { return draw(c, in, st) })
	}
}

// PageHeader is the header of a page: a title, a line under it, and actions
// at the end.
func PageHeader(title, subtitle string, leading Widget, actions ...Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		colors := c.Env.Theme.Colors
		text := []FlexChild{Rigid(markHeader(Line(title, Emph(HeadlineMedium), colors.OnSurface)))}
		if subtitle != "" {
			text = append(text, Rigid(Space(0, 4)), Rigid(Label(subtitle, Type(BodyMedium), colors.OnSurfaceVariant, Lines(2))))
		}
		var row []FlexChild
		if leading != nil {
			row = append(row, Rigid(leading), Rigid(Space(8, 0)))
		}
		row = append(row, Expanded(1, Column(FlexOpts{Min: true}, text...)))
		if len(actions) > 0 {
			row = append(row, Rigid(Space(16, 0)), Rigid(Wrap{Spacing: 8, CrossCenter: true, Align: MainEnd}.Layout(actions...)))
		}
		return Pad(Inset{24, 24, 16, 16}, Row(FlexOpts{Cross: CrossCenter}, row...))(c)
	}
}

// markHeader is the identity: headers are marked for assistive technologies,
// which Gio does not expose on Linux yet.
func markHeader(w Widget) Widget { return w }

// GroupHeader is the title of a group of a page.
func GroupHeader(title string, trailing Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		colors := c.Env.Theme.Colors
		row := []FlexChild{Expanded(1, markHeader(Line(title, Type(TitleSmall), colors.Primary)))}
		if trailing != nil {
			row = append(row, Rigid(trailing))
		}
		return Pad(Inset{16, 24, 4, 8}, Row(FlexOpts{Cross: CrossCenter}, row...))(c)
	}
}

// ContentWidth limits the width of a page and centres it at the top.
func ContentWidth(maxWidth float32, child Widget) Widget {
	return Align(layout.N, MaxWidth(maxWidth, FillWidth(child)))
}

// StatusMessage is the centred message of an empty, failed or unavailable view:
// an icon on a large shape, a title, an explanation and actions.
type StatusMessage struct {
	// Icon is the icon.
	Icon string
	// Shape is the polygon the icon sits on; the cookie with nine sides by
	// default.
	Shape *shapes.Polygon
	// IsError uses the colours of errors.
	IsError bool
	// Title says what the message is about.
	Title string
	// Body explains it.
	Body string
	// Actions are the buttons that resolve it.
	Actions []Widget
}

// W is the message as a widget.
func (m StatusMessage) W() Widget {
	return func(c Ctx) layout.Dimensions {
		colors := c.Env.Theme.Colors
		shape := m.Shape
		if shape == nil {
			shape = shapes.ShapeCookie9
		}
		background, foreground := colors.SecondaryContainer, colors.OnSecondaryContainer
		if m.IsError {
			background, foreground = colors.ErrorContainer, colors.OnErrorContainer
		}
		children := []FlexChild{
			Rigid(ShapeBox{Polygon: shape, Size: 112, Color: background, Child: Center(Icon(m.Icon, 48, foreground, 1))}.W()),
			Rigid(Space(0, 24)),
			Rigid(Label(m.Title, Emph(TitleLarge), colors.OnSurface, Centered())),
		}
		if m.Body != "" {
			children = append(children, Rigid(Space(0, 8)), Rigid(Label(m.Body, Type(BodyMedium), colors.OnSurfaceVariant, Centered())))
		}
		if len(m.Actions) > 0 {
			children = append(children, Rigid(Space(0, 24)),
				Rigid(Wrap{Spacing: 8, RunSpacing: 8, Align: MainCenter}.Layout(m.Actions...)))
		}
		body := Pad(All(32), Limit(0, 0, 504, 0, Column(FlexOpts{Cross: CrossCenter, Min: true}, children...)))
		return Center(body)(c)
	}
}

// ShapeBox draws a child on a filled shape of the shape library.
type ShapeBox struct {
	// Polygon is the shape.
	Polygon *shapes.Polygon
	// Size is the side of the box in dp.
	Size float32
	// Color fills the shape.
	Color color.NRGBA
	// Rotation turns the shape, in radians.
	Rotation float64
	// Child is drawn over the shape, centred in the box.
	Child Widget
}

// W is the box as a widget.
func (s ShapeBox) W() Widget {
	return func(c Ctx) layout.Dimensions {
		px := c.DP(s.Size)
		size := c.Constraints.Constrain(image.Pt(px, px))
		FillShape(c.Ops, s.Polygon.Cubics, image.Rectangle{Max: size}, s.Rotation, s.Color)
		if s.Child != nil {
			p := c.measure(tight(size), s.Child)
			p.put(c, image.Point{})
		}
		return layout.Dimensions{Size: size}
	}
}

// LoadingMessage is a loading indicator in the middle of the space, with a
// line of text under it.
func LoadingMessage(label, accessibleLabel string) Widget {
	return func(c Ctx) layout.Dimensions {
		colors := c.Env.Theme.Colors
		children := []FlexChild{Rigid(LoadingIndicator{ID: "loading-message", Size: 64, Contained: true, Label: accessibleLabel}.W())}
		if label != "" {
			children = append(children, Rigid(Space(0, 16)), Rigid(Label(label, Type(BodyMedium), colors.OnSurfaceVariant, Centered())))
		}
		return Center(Pad(All(32), Column(FlexOpts{Cross: CrossCenter, Min: true}, children...)))(c)
	}
}

// Panel is a filled surface whose top and bottom corners are rounded
// differently, such as the output of a task, which hangs under its list item.
type Panel struct {
	// Color fills the panel.
	Color color.NRGBA
	// Top and Bottom are the radii of the corners at the top and at the
	// bottom, in dp.
	Top, Bottom float32
	// Child is what the panel holds.
	Child Widget
}

// W is the panel as a widget.
func (p Panel) W() Widget {
	return func(c Ctx) layout.Dimensions {
		child := c.measure(c.Constraints, p.Child)
		size := child.size()
		top, bottom := float32(c.DP(p.Top)), float32(c.DP(p.Bottom))
		FillRRect(c.Ops, image.Rectangle{Max: size}, Radii{TopLeft: top, TopRight: top, BottomRight: bottom, BottomLeft: bottom}, p.Color)
		child.put(c, image.Point{})
		return layout.Dimensions{Size: size}
	}
}
