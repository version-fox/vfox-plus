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
	"strconv"

	"github.com/egoist/mygo/ui"
)

// NavDestination is a destination of the navigation rail or bar.
type NavDestination struct {
	// Icon is the icon, filled while the destination is selected.
	Icon string
	// Label names the destination.
	Label string
	// Badge shows a count on the icon when it is above zero.
	Badge int
}

// NavRailSpec describes the navigation rail of Material 3 Expressive
// (docs/en/dev/design-system.md#components): 96 dp wide with the labels under the
// icons, or 240 dp wide with the labels next to them, with a menu button that
// switches between the two, and a floating action button.
type NavRailSpec struct {
	// ID identifies the rail among its siblings between builds.
	ID string
	// Destinations are the destinations.
	Destinations []NavDestination
	// Selected is the index of the selected destination.
	Selected int
	// OnSelect runs with the index of the destination the user chose.
	OnSelect func(index int)
	// Expanded says whether the labels are next to the icons.
	Expanded bool
	// OnExpandedChange runs when the user presses the menu button.
	OnExpandedChange func(expanded bool)
	// ExpandTooltip and CollapseTooltip name the menu button.
	ExpandTooltip, CollapseTooltip string
	// FABIcon and FABLabel make the floating action button; it is left out
	// when OnFAB is nil.
	FABIcon, FABLabel string
	OnFAB             func()
}

const (
	railCollapsed = 96
	railExpanded  = 240
	// NavBarHeight is the height of the navigation bar in dp, where the text of the system is not larger than usual; BarHeight is
	// what it is where it is.
	NavBarHeight = 64
)

// BarHeight is the least height in dp of the navigation bar: NavBarHeight, which holds the icon and a label of 16 dp, and more where the
// labels are larger, to the size that they are read at most (barLabelMaxScale). The bar is as high as its labels are, too, and a line of
// text is higher than the one that a label asks for where the font has a higher one, or the one that Han characters fall back to: 18 dp
// for 16 in Noto Sans CJK, 22 for 20.8 in Noto Sans.
func BarHeight(c Ctx) float32 {
	return NavBarHeight + Type(LabelMedium).Line*max(min(c.Env.Theme.TextScale, barLabelMaxScale)-1, 0)
}

// barGrowth is how much higher than NavBarHeight the bar is at the most, with a line that is a few dp higher than the one that a label
// asks for (barLineSlack) at the size that the labels are read at most.
func barGrowth() float32 { return Type(LabelMedium).Line*(barLabelMaxScale-1) + barLineSlack }

// barLineSlack is how many dp higher than the line that a label asks for the line of a font may be, in the bar: 6 dp is more than the
// fonts that were seen need (2 to 3 dp), and the census looks for the bar among the buttons of the window by its height.
const barLineSlack = 6

// NavRail is the navigation rail; it is as high as the space it is given, and
// grows and shrinks between its two widths on a spatial spring.
func NavRail(c Ctx, r NavRailSpec) ui.Element {
	if hasKey(r.ID) {
		c.Key(r.ID)
	}
	rail := ui.Column(c.Context).FillHeight().Shrink(0).AlignItems(ui.Start).Padding(16, 0, 0, 0)
	t := rail.AnimateTo("expand", Flag(r.Expanded), c.Env.Theme.Motion.DefaultSpatial.Anim())
	width := Lerp(railCollapsed, railExpanded, t)
	rail.Width(width)
	return rail.Children(func() {
		// The menu button is centred over the collapsed rail and aligned with the
		// icons of the items when expanded.
		tooltip, icon := r.ExpandTooltip, "menu"
		if r.Expanded {
			tooltip, icon = r.CollapseTooltip, "menuOpen"
		}
		IconButton(c, IconButtonSpec{
			ID: "menu", Icon: icon, Tooltip: tooltip,
			OnClick: func() { r.OnExpandedChange(!r.Expanded) },
		}).Margin(0, 0, 8, Lerp((railCollapsed-48)/2, 16, t))
		if r.OnFAB != nil {
			FAB(c, FABSpec{
				ID: "fab", Icon: r.FABIcon, Label: r.FABLabel, Expansion: t, OnClick: r.OnFAB,
			}).Margin(0, 0, 24, Lerp((railCollapsed-56)/2, 12, t))
		} else {
			Gap(c, 0, 16)
		}
		// An item is as high as its label needs, which is more than the 16 dp of a line at 100 % where the text is larger.
		labelScale := railLabelScale(c, r)
		itemHeight := Lerp(64, 56, t) + 4 + Type(LabelMedium).Line*max(labelScale-1, 0)*(1-t)
		for i, d := range r.Destinations {
			railItem(c, r, i, d, t, width, itemHeight, labelScale)
		}
	})
}

func railItem(c Ctx, r NavRailSpec, index int, d NavDestination, t, width, height, labelScale float32) {
	cs := c.Colors()
	selected := index == r.Selected
	el, in, node := press(c, pressSpec{
		id: strconv.Itoa(index), role: RoleButton, name: d.Label, selected: selected,
		onClick: func() { r.OnSelect(index) },
	})
	selection := el.AnimateTo("selection", Flag(selected), c.Env.Theme.Motion.FastSpatial.Anim())
	fill := el.AnimateTo("fill", Flag(selected), c.Env.Theme.Motion.FastEffects.Anim())
	layer := stateLayer(c, el, in)
	// Collapsed: a 56 × 32 indicator with the label below it. Expanded: a 56 dp
	// high indicator around the icon and the label.
	indicator := lerpRect(
		ui.Rect{X: (railCollapsed - 56) / 2, Y: 4, W: 56, H: 32},
		ui.Rect{X: 12, Y: 0, W: width - 24, H: 56}, t)
	// The indicator grows out of the centre of the icon.
	grown := scaleRectX(indicator, 0.5+0.5*min(max(selection, 0), 1.2))
	el.Size(width, height).Shrink(0).Background(ui.Transparent)
	drawNavIndicator(c, el, node, grown, indicator, selection, layer, in.keyboard)

	iconColor := cs.OnSurfaceVariant
	if selected {
		iconColor = cs.OnSecondaryContainer
	}
	iconX := Lerp(indicator.X+indicator.W/2, indicator.X+16+12, t) - 12
	iconY := indicator.Y + indicator.H/2 - 12
	el.Children(func() {
		navIcon(c, d, iconColor, fill).Absolute().Left(iconX).Top(iconY)
		if collapsed := 1 - t*2; collapsed > 0 {
			colour, weight := cs.OnSurfaceVariant, 500
			if selected {
				colour, weight = cs.Secondary, 700
			}
			Line(c, d.Label, Type(LabelMedium).WithWeight(weight), colour, Centered(), ScaleCap(labelScale)).
				Absolute().Left(0).Top(indicator.Y + indicator.H + 4).Width(railCollapsed).Opacity(collapsed)
		}
		if expanded := (t - 0.5) * 2; expanded > 0 {
			colour, weight := cs.OnSurfaceVariant, 500
			if selected {
				colour, weight = cs.OnSecondaryContainer, 700
			}
			Line(c, d.Label, Type(LabelLarge).WithWeight(weight), colour).
				Absolute().Left(indicator.X + 16 + 24 + 12).Top(indicator.Y + indicator.H/2 - 10).
				Width(max(0, width-(indicator.X+16+24+12)-24)).Opacity(expanded)
		}
	})
}

// navIcon is the icon of a destination, a 24 dp box with the badge at its top
// end.
func navIcon(c Ctx, d NavDestination, col ui.Color, fill float32) ui.Element {
	return ui.Box(c.Context).Size(24, 24).Shrink(0).Children(func() {
		Icon(c, d.Icon, 24, col, fill)
		if d.Badge > 0 {
			badge(c, d.Badge).Absolute().Left(16).Top(-4)
		}
	})
}

// badge is the count at the top end of an icon.
func badge(c Ctx, count int) ui.Element {
	cs := c.Colors()
	label := strconv.Itoa(count)
	if count > 999 {
		label = "999+"
	}
	// The count is as large as the labels are read, and no larger, and the badge is as high as its line: a count of 22 dp was drawn in a
	// badge of 16, as high as the digit.
	height := Type(LabelSmall).Line * max(min(c.Env.Theme.TextScale, barLabelMaxScale), 1)
	return Row(c).Height(height).MinWidth(height).PaddingX(4).Justify(ui.Center).Radius(height / 2).Background(cs.Error).Shrink(0).
		Children(func() {
			Line(c, label, Type(LabelSmall), cs.OnError, ScaleCap(barLabelMaxScale)).NoWrap()
		})
}

// drawNavIndicator draws the indicator of a destination, its state layer and its
// focus ring, and tells the node where the destination is.
func drawNavIndicator(c Ctx, el ui.Element, node *Node, grown, indicator ui.Rect, selection, layer float32, keyboard bool) {
	cs := c.Colors()
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		node.Rect = r
		g := ui.Rect{X: r.X + grown.X, Y: r.Y + grown.Y, W: grown.W, H: grown.H}
		radii := Uniform(g.H / 2)
		fillRRect(p, g, radii, cs.SecondaryContainer.Alpha(clamp01(selection)))
		fillRRect(p, g, radii, cs.OnSurface.Alpha(layer))
	})
	if keyboard {
		el.DrawOver(func(p *ui.Painter, r ui.Rect) {
			i := ui.Rect{X: r.X + indicator.X - 2, Y: r.Y + indicator.Y - 2, W: indicator.W + 4, H: indicator.H + 4}
			strokeRRect(p, i, Uniform(i.H/2), 3, cs.Secondary)
		})
	}
}

func lerpRect(a, b ui.Rect, t float32) ui.Rect {
	return ui.Rect{X: Lerp(a.X, b.X, t), Y: Lerp(a.Y, b.Y, t), W: Lerp(a.W, b.W, t), H: Lerp(a.H, b.H, t)}
}

func scaleRectX(r ui.Rect, scale float32) ui.Rect {
	centre := r.X + r.W/2
	half := r.W / 2 * scale
	return ui.Rect{X: centre - half, Y: r.Y, W: half * 2, H: r.H}
}

// NavBarSpec describes the navigation bar of Material 3 for compact windows: the
// destinations at the bottom, the selected one on an indicator.
type NavBarSpec struct {
	// ID identifies the bar among its siblings between builds.
	ID string
	// Destinations are the destinations.
	Destinations []NavDestination
	// Selected is the index of the selected destination.
	Selected int
	// OnSelect runs with the index of the destination the user chose.
	OnSelect func(index int)
}

// NavBar is the navigation bar: as wide as the space it is given.
func NavBar(c Ctx, b NavBarSpec) ui.Element {
	cs := c.Colors()
	if hasKey(b.ID) {
		c.Key(b.ID)
	}
	return ui.Row(c.Context).MinHeight(BarHeight(c)).Shrink(0).AlignItems(ui.Stretch).Background(cs.SurfaceContainer).
		Children(func() {
			labelScale := barLabelScale(c, b)
			for i, d := range b.Destinations {
				barItem(c, b, i, d, labelScale)
			}
		})
}

// barLabelMaxScale is the text scale beyond which the labels of the navigation bar do not grow: each destination has a
// fifth of the window to be read in, and the bar is as high as it is.
const barLabelMaxScale = 1.3

// navLabelMargin is the room in dp that a label of a destination leaves in its slot, between itself and what is beside it.
const navLabelMargin = 8

// fitLabelScale is the text scale of the labels of the destinations, which have a room that does not grow with the text: the text
// scale of the system, up to limit, and less where the longest of the labels does not fit the room, down to 100 %. A label that does
// not fit at that size is cut with an ellipsis, as the text of the interface is where it has not the room, and the size of the text
// is not made less than it is.
func fitLabelScale(c Ctx, destinations []NavDestination, style TextStyle, room, limit float32) float32 {
	floor := min(1, c.Env.Theme.TextScale)
	scale := min(c.Env.Theme.TextScale, limit)
	for _, d := range destinations {
		// The label of the selected destination is bold, which is the widest that a label is.
		natural, _ := c.MeasureText(0, ui.Span{Text: d.Label, Size: style.Size, Weight: 700})
		if natural > 0 {
			scale = min(scale, room/natural)
		}
	}
	return max(scale, floor)
}

// barLabelScale is the text scale of the labels of the bar: each has a fifth of the window.
func barLabelScale(c Ctx, b NavBarSpec) float32 {
	windowWidth, _ := c.Size()
	room := windowWidth/float32(max(1, len(b.Destinations))) - navLabelMargin
	return fitLabelScale(c, b.Destinations, Type(LabelMedium), room, barLabelMaxScale)
}

// railLabelScale is the text scale of the labels of the rail when it is collapsed, which are under the icons, in the width of the rail.
func railLabelScale(c Ctx, r NavRailSpec) float32 {
	return fitLabelScale(c, r.Destinations, Type(LabelMedium), railCollapsed-navLabelMargin, c.Env.Theme.TextScale)
}

func barItem(c Ctx, b NavBarSpec, index int, d NavDestination, labelScale float32) {
	cs := c.Colors()
	selected := index == b.Selected
	el, in, node := press(c, pressSpec{
		id: strconv.Itoa(index), role: RoleButton, name: d.Label, selected: selected,
		onClick: func() { b.OnSelect(index) },
	})
	selection := el.AnimateTo("selection", Flag(selected), c.Env.Theme.Motion.FastSpatial.Anim())
	fill := el.AnimateTo("fill", Flag(selected), c.Env.Theme.Motion.FastEffects.Anim())
	layer := stateLayer(c, el, in)
	iconColor, labelColor, weight := cs.OnSurfaceVariant, cs.OnSurfaceVariant, 500
	if selected {
		iconColor, labelColor, weight = cs.OnSecondaryContainer, cs.OnSurface, 700
	}
	el.Grow(1).Basis(0).AlignItems(ui.Start).Justify(ui.Start)
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		node.Rect = r
		indicator := ui.Rect{X: r.X + (r.W-64)/2, Y: r.Y + 12, W: 64, H: 32}
		grown := scaleRectX(indicator, 0.5+0.5*min(max(selection, 0), 1.2))
		radii := Uniform(grown.H / 2)
		fillRRect(p, grown, radii, cs.SecondaryContainer.Alpha(clamp01(selection)))
		fillRRect(p, grown, radii, cs.OnSurface.Alpha(layer))
		if in.keyboard {
			ring := ui.Rect{X: indicator.X - 2, Y: indicator.Y - 2, W: indicator.W + 4, H: indicator.H + 4}
			strokeRRect(p, ring, Uniform(ring.H/2), 3, cs.Secondary)
		}
	})
	el.Children(func() {
		ui.Column(c.Context).FillWidth().AlignItems(ui.Center).Padding(16, 0, 0, 0).Gap(8).Children(func() {
			navIcon(c, d, iconColor, fill)
			Line(c, d.Label, Type(LabelMedium).WithWeight(weight), labelColor, Centered(), ScaleCap(labelScale)).MaxWidth(100)
		})
	})
}
