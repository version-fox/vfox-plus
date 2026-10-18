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

	"gioui.org/layout"
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

// NavRail is the navigation rail of Material 3 Expressive
// (docs/design-system.md#components): 96 dp wide with the labels under the
// icons, or 240 dp wide with the labels next to them, with a menu button
// that switches between the two, and a floating action button.
type NavRail struct {
	// ID identifies the rail between frames, within its scope.
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
)

type navState struct {
	expand Anim
}

// W is the rail as a widget; it is as high as the space it is given.
func (r NavRail) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[navState](c, r.ID)
		theme := c.Env.Theme
		t := float32(st.expand.To(c, boolFloat(r.Expanded), theme.Motion.DefaultSpatial))
		dp := func(v float32) int { return c.DP(v) }
		width := int(Lerp(float32(dp(railCollapsed)), float32(dp(railExpanded)), t))
		height := c.Constraints.Max.Y
		if unbounded(height) {
			height = dp(400)
		}
		size := c.Constraints.Constrain(image.Pt(width, height))
		y := dp(16)

		// The menu button is centred over the collapsed rail and aligned with
		// the icons of the items when expanded.
		menuX := int(Lerp(float32(dp((railCollapsed-48)/2)), float32(dp(16)), t))
		tooltip := r.ExpandTooltip
		icon := "menu"
		if r.Expanded {
			tooltip, icon = r.CollapseTooltip, "menuOpen"
		}
		menu := c.measure(loose(image.Pt(dp(48), dp(48))), IconButton{
			ID: r.ID + ".menu", Icon: icon, Tooltip: tooltip,
			OnClick: func() { r.OnExpandedChange(!r.Expanded) },
		}.W())
		menu.put(c, image.Pt(menuX, y))
		y += menu.size().Y + dp(8)

		if r.OnFAB != nil {
			fabX := int(Lerp(float32(dp((railCollapsed-56)/2)), float32(dp(12)), t))
			fab := c.measure(loose(image.Pt(size.X, dp(56))), FAB{
				ID: r.ID + ".fab", Icon: r.FABIcon, Label: r.FABLabel, Expansion: t, OnClick: r.OnFAB,
			}.W())
			fab.put(c, image.Pt(fabX, y))
			y += fab.size().Y + dp(0)
		}
		y += dp(24)

		itemHeight := int(Lerp(64, 56, t)*float32(dp(1))) + dp(4)
		for i, d := range r.Destinations {
			r.layoutItem(c, i, d, y, width, t)
			y += itemHeight
		}
		return layout.Dimensions{Size: size}
	}
}

func (r NavRail) layoutItem(c Ctx, index int, d NavDestination, top, width int, t float32) {
	theme := c.Env.Theme
	colors := theme.Colors
	selected := index == r.Selected
	st := UseID[navItemState](c, r.ID+".item"+strconv.Itoa(index))
	dp := func(v float32) int { return c.DP(v) }
	itemHeight := int(Lerp(64, 56, t)*float32(dp(1))) + dp(4)
	item := c.measure(tight(image.Pt(width, itemHeight)), func(c Ctx) layout.Dimensions {
		spec := pressSpec{
			role: RoleButton, name: d.Label, enabled: true, selected: selected,
			onClick: func() { r.OnSelect(index) },
		}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			size := image.Pt(width, itemHeight)
			// Collapsed: a 56 × 32 indicator with the label below it. Expanded:
			// a 56 dp high indicator around the icon and the label.
			collapsed := image.Rect(dp((railCollapsed-56)/2), dp(4), dp((railCollapsed-56)/2)+dp(56), dp(4)+dp(32))
			expanded := image.Rect(dp(12), 0, width-dp(12), dp(56))
			indicator := lerpRect(collapsed, expanded, t)
			s := float32(st.selection.To(c, boolFloat(selected), theme.Motion.FastSpatial))
			fill := float32(st.fill.To(c, boolFloat(selected), theme.Motion.FastEffects))
			// The indicator grows out of the centre of the icon.
			scale := 0.5 + 0.5*minf(maxf(s, 0), 1.2)
			grown := scaleRectX(indicator, scale)
			r := Uniform(float32(grown.Dy()) / 2)
			FillRRect(c.Ops, grown, r, Alpha(colors.SecondaryContainer, minf(maxf(s, 0), 1)))
			FillRRect(c.Ops, grown, r, st.press.stateLayer(c, in, colors.OnSurface))
			focusRing(c, in, indicator, r)

			iconColor := colors.OnSurfaceVariant
			if selected {
				iconColor = colors.OnSecondaryContainer
			}
			iconCenter := image.Pt(
				int(Lerp(float32(indicator.Min.X+indicator.Dx()/2), float32(indicator.Min.X+dp(16)+dp(12)), t)),
				indicator.Min.Y+indicator.Dy()/2)
			iconBox := image.Rect(iconCenter.X-dp(12), iconCenter.Y-dp(12), iconCenter.X+dp(12), iconCenter.Y+dp(12))
			icon := c.measure(loose(image.Pt(dp(24), dp(24))), Icon(d.Icon, 24, iconColor, fill))
			icon.put(c, iconBox.Min)
			if d.Badge > 0 {
				drawBadge(c, iconBox, d.Badge)
			}
			collapsedLabel := 1 - t*2
			expandedLabel := (t - 0.5) * 2
			if collapsedLabel > 0 {
				col := colors.OnSurfaceVariant
				weight := 500
				if selected {
					col, weight = colors.Secondary, 700
				}
				p := c.measure(loose(image.Pt(dp(railCollapsed), dp(16))), Line(d.Label, Type(LabelMedium).WithWeight(weight), col, Centered()))
				Opacity(c.Ops, collapsedLabel, func() {
					p.put(c, image.Pt((dp(railCollapsed)-p.size().X)/2, indicator.Max.Y+dp(4)))
				})
			}
			if expandedLabel > 0 {
				col := colors.OnSurfaceVariant
				weight := 500
				if selected {
					col, weight = colors.OnSecondaryContainer, 700
				}
				p := c.measure(loose(image.Pt(width-(indicator.Min.X+dp(16)+dp(24)+dp(12))-dp(24), dp(24))),
					Line(d.Label, Type(LabelLarge).WithWeight(weight), col))
				Opacity(c.Ops, expandedLabel, func() {
					p.put(c, image.Pt(indicator.Min.X+dp(16)+dp(24)+dp(12), indicator.Min.Y+indicator.Dy()/2-p.size().Y/2))
				})
			}
			return layout.Dimensions{Size: size}
		})
	})
	item.put(c, image.Pt(0, top))
}

type navItemState struct {
	press     pressable
	selection Anim
	fill      Anim
}

func lerpRect(a, b image.Rectangle, t float32) image.Rectangle {
	l := func(x, y int) int { return int(Lerp(float32(x), float32(y), t)) }
	return image.Rect(l(a.Min.X, b.Min.X), l(a.Min.Y, b.Min.Y), l(a.Max.X, b.Max.X), l(a.Max.Y, b.Max.Y))
}

func scaleRectX(r image.Rectangle, scale float32) image.Rectangle {
	center := float32(r.Min.X+r.Max.X) / 2
	half := float32(r.Dx()) / 2 * scale
	return image.Rect(int(center-half), r.Min.Y, int(center+half), r.Max.Y)
}

// drawBadge draws the count badge at the top end of an icon.
func drawBadge(c Ctx, icon image.Rectangle, count int) {
	colors := c.Env.Theme.Colors
	label := strconv.Itoa(count)
	if count > 999 {
		label = "999+"
	}
	text := c.measure(loose(image.Pt(c.DP(64), c.DP(16))), Line(label, Type(LabelSmall), colors.OnError))
	height := c.DP(16)
	width := maxInt(height, text.size().X+c.DP(8))
	left := icon.Max.X - c.DP(8)
	rect := image.Rect(left, icon.Min.Y-c.DP(4), left+width, icon.Min.Y-c.DP(4)+height)
	FillRRect(c.Ops, rect, Uniform(float32(height)/2), colors.Error)
	text.put(c, image.Pt(rect.Min.X+(width-text.size().X)/2, rect.Min.Y+(height-text.size().Y)/2))
}

// NavBar is the navigation bar of Material 3 for compact windows: the
// destinations at the bottom, the selected one on an indicator.
type NavBar struct {
	// ID identifies the bar between frames, within its scope.
	ID string
	// Destinations are the destinations.
	Destinations []NavDestination
	// Selected is the index of the selected destination.
	Selected int
	// OnSelect runs with the index of the destination the user chose.
	OnSelect func(index int)
}

// NavBarHeight is the height of the navigation bar in dp.
const NavBarHeight = 64

// W is the bar as a widget: as wide as the space it is given.
func (b NavBar) W() Widget {
	return func(c Ctx) layout.Dimensions {
		theme := c.Env.Theme
		colors := theme.Colors
		dp := func(v float32) int { return c.DP(v) }
		width := c.Constraints.Max.X
		size := image.Pt(width, dp(NavBarHeight))
		FillRRect(c.Ops, image.Rectangle{Max: size}, Radii{}, colors.SurfaceContainer)
		n := len(b.Destinations)
		for i, d := range b.Destinations {
			i, d := i, d
			selected := i == b.Selected
			st := UseID[navItemState](c, b.ID+".item"+strconv.Itoa(i))
			left, right := width*i/n, width*(i+1)/n
			item := c.measure(tight(image.Pt(right-left, size.Y)), func(c Ctx) layout.Dimensions {
				spec := pressSpec{role: RoleButton, name: d.Label, enabled: true, selected: selected, onClick: func() { b.OnSelect(i) }}
				return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
					itemWidth := right - left
					s := float32(st.selection.To(c, boolFloat(selected), theme.Motion.FastSpatial))
					fill := float32(st.fill.To(c, boolFloat(selected), theme.Motion.FastEffects))
					indicator := image.Rect((itemWidth-dp(64))/2, dp(12), (itemWidth+dp(64))/2, dp(12)+dp(32))
					grown := scaleRectX(indicator, 0.5+0.5*minf(maxf(s, 0), 1.2))
					r := Uniform(float32(grown.Dy()) / 2)
					FillRRect(c.Ops, grown, r, Alpha(colors.SecondaryContainer, minf(maxf(s, 0), 1)))
					FillRRect(c.Ops, grown, r, st.press.stateLayer(c, in, colors.OnSurface))
					focusRing(c, in, indicator, r)
					iconColor, labelColor, weight := colors.OnSurfaceVariant, colors.OnSurfaceVariant, 500
					if selected {
						iconColor, labelColor, weight = colors.OnSecondaryContainer, colors.OnSurface, 700
					}
					iconBox := image.Rect(itemWidth/2-dp(12), indicator.Min.Y+dp(4), itemWidth/2+dp(12), indicator.Min.Y+dp(28))
					icon := c.measure(loose(image.Pt(dp(24), dp(24))), Icon(d.Icon, 24, iconColor, fill))
					icon.put(c, iconBox.Min)
					if d.Badge > 0 {
						drawBadge(c, iconBox, d.Badge)
					}
					label := c.measure(loose(image.Pt(itemWidth, dp(16))), Line(d.Label, Type(LabelMedium).WithWeight(weight), labelColor, Centered()))
					label.put(c, image.Pt((itemWidth-label.size().X)/2, indicator.Max.Y+dp(4)))
					return layout.Dimensions{Size: image.Pt(itemWidth, size.Y)}
				})
			})
			item.put(c, image.Pt(left, 0))
		}
		return layout.Dimensions{Size: size}
	}
}
