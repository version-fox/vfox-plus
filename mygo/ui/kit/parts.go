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

	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// Pane is a rounded surface that a page sits on: 28 dp corners, the content
// clipped to them, on the surface container low.
func Pane(c Ctx, content func()) ui.Element {
	return PaneColor(c, c.Colors().SurfaceContainerLow, content)
}

// PaneColor is a pane of a colour other than the surface container low.
func PaneColor(c Ctx, col ui.Color, content func()) ui.Element {
	return ui.Column(c.Context).Radius(CornerExtraLarge).Background(col).Clip().Children(content)
}

// CardSpec describes a card.
type CardSpec struct {
	// ID identifies the card among its siblings between builds.
	ID string
	// Color is the colour of the card.
	Color ui.Color
	// Radius is the corner radius in dp.
	Radius float32
	// OnClick makes the card react to the pointer.
	OnClick func()
	// Content is the foreground colour, for the state layer and the text.
	Content ui.Color
	// Name names the card for assistive technology.
	Name string
	// Children builds what the card holds.
	Children func()
}

// Card is a surface with rounded corners that may be clicked.
func Card(c Ctx, cd CardSpec) ui.Element {
	if cd.OnClick == nil {
		if hasKey(cd.ID) {
			c.Key(cd.ID)
		}
		el := ui.Column(c.Context).Radius(cd.Radius).Background(cd.Color).Clip().TextColor(cd.Content)
		if cd.Name != "" {
			el.Label(cd.Name)
		}
		return el.Children(cd.Children)
	}
	el, in, node := press(c, pressSpec{id: cd.ID, role: RoleButton, name: cd.Name, onClick: cd.OnClick})
	el.Radius(cd.Radius).Background(cd.Color).Clip().TextColor(cd.Content).AlignItems(ui.Stretch).Justify(ui.Start)
	paintPress(c, el, node, Uniform(cd.Radius), stateLayer(c, el, in), cd.Content, in.keyboard)
	// A card holds a column of content: the button of mygo is a row.
	return el.Children(func() {
		ui.Column(c.Context).Grow(1).Shrink(1).Children(cd.Children)
	})
}

// PageHeader is the header of a page: a title, a line under it, and actions at
// the end.
func PageHeader(c Ctx, title, subtitle string, leading func(), actions ...func()) ui.Element {
	cs := c.Colors()
	return ui.Row(c.Context).Padding(24, 16, 16, 24).Gap(8).Children(func() {
		if leading != nil {
			leading()
		}
		ui.Column(c.Context).Grow(1).Shrink(1).Gap(4).Children(func() {
			Line(c, title, Emph(HeadlineMedium), cs.OnSurface).Role(ui.RoleHeading).Level(1)
			if subtitle != "" {
				Label(c, subtitle, Type(BodyMedium), cs.OnSurfaceVariant, Lines(2))
			}
		})
		if len(actions) > 0 {
			ui.Row(c.Context).Gap(8).Wrap().Justify(ui.End).Margin(0, 0, 0, 8).Children(func() {
				for _, action := range actions {
					action()
				}
			})
		}
	})
}

// GroupHeader is the title of a group of a page.
func GroupHeader(c Ctx, title string, trailing func()) ui.Element {
	cs := c.Colors()
	return ui.Row(c.Context).Padding(24, 4, 8, 16).Children(func() {
		Line(c, title, Type(TitleSmall), cs.Primary).Grow(1).Role(ui.RoleHeading).Level(2)
		if trailing != nil {
			trailing()
		}
	})
}

// ContentWidth limits the width of a page and centres it at the top.
func ContentWidth(c Ctx, maxWidth float32, content func()) ui.Element {
	return ui.Column(c.Context).MaxWidth(maxWidth).FillWidth().AlignSelf(ui.Center).Children(content)
}

// StatusSpec describes the centred message of an empty, failed or unavailable
// view: an icon on a large shape, a title, an explanation and actions.
type StatusSpec struct {
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
	// Actions build the buttons that resolve it.
	Actions []func()
}

// StatusMessage is the message of an empty, failed or unavailable view, in the
// middle of the space.
func StatusMessage(c Ctx, m StatusSpec) ui.Element {
	cs := c.Colors()
	shape := m.Shape
	if shape == nil {
		shape = shapes.ShapeCookie9
	}
	background, foreground := cs.SecondaryContainer, cs.OnSecondaryContainer
	if m.IsError {
		background, foreground = cs.ErrorContainer, cs.OnErrorContainer
	}
	return ui.Column(c.Context).Fill().Center().Padding(32).Children(func() {
		ui.Column(c.Context).MaxWidth(504).AlignItems(ui.Center).Gap(8).Children(func() {
			Shape(c, shape, 112, background, func() {
				Icon(c, m.Icon, 48, foreground, 1)
			}).Margin(0, 0, 16, 0)
			Label(c, m.Title, Emph(TitleLarge), cs.OnSurface, Centered())
			if m.Body != "" {
				Label(c, m.Body, Type(BodyMedium), cs.OnSurfaceVariant, Centered())
			}
			if len(m.Actions) > 0 {
				ui.Row(c.Context).Gap(8).Wrap().Justify(ui.Center).Margin(16, 0, 0, 0).Children(func() {
					for _, action := range m.Actions {
						action()
					}
				})
			}
		})
	})
}

// LoadingMessage is a loading indicator in the middle of the space, with a line
// of text under it.
func LoadingMessage(c Ctx, label, accessibleLabel string) ui.Element {
	cs := c.Colors()
	return ui.Column(c.Context).Fill().Center().Padding(32).Children(func() {
		ui.Column(c.Context).AlignItems(ui.Center).Gap(16).Children(func() {
			LoadingIndicator(c, LoadingSpec{ID: "loading-message", Size: 64, Contained: true, Label: accessibleLabel})
			if label != "" {
				Label(c, label, Type(BodyMedium), cs.OnSurfaceVariant, Centered())
			}
		})
	})
}

// Panel is a filled surface whose top and bottom corners are rounded
// differently, such as the output of a task, which hangs under its list item.
func Panel(c Ctx, col ui.Color, top, bottom float32, content func()) ui.Element {
	return ui.Column(c.Context).Background(col).Radius(top, top, bottom, bottom).Children(content)
}
