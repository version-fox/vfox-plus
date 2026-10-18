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

// Switch is a Material 3 switch: a track with a thumb that slides, and grows
// while it is pressed.
type Switch struct {
	// ID identifies the switch between frames, within its scope.
	ID string
	// On says whether the switch is on.
	On bool
	// OnChange runs with the new state when the user flips the switch.
	OnChange func(on bool)
	// Disabled makes the switch inert.
	Disabled bool
	// Label names the switch for the accessibility tree.
	Label string
}

type switchState struct {
	press pressable
	slide Anim
	grow  Anim
}

// W is the switch as a widget.
func (s Switch) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[switchState](c, s.ID)
		theme := c.Env.Theme
		colors := theme.Colors
		spec := pressSpec{
			role: RoleSwitch, name: s.Label, enabled: !s.Disabled && s.OnChange != nil, selected: s.On,
			onClick: func() { s.OnChange(!s.On) },
		}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			width, height := c.DP(52), c.DP(32)
			size := image.Pt(width, height)
			position := float32(st.slide.To(c, boolFloat(s.On), theme.Motion.FastSpatial))
			grow := float32(st.grow.To(c, boolFloat(in.pressed), theme.Motion.FastSpatial))
			track, thumb, outline := colors.SurfaceContainerHighest, colors.Outline, colors.Outline
			if s.On {
				track, thumb = colors.Primary, colors.OnPrimary
			}
			if s.Disabled {
				track = Alpha(colors.OnSurface, 0.12)
				thumb = Alpha(colors.OnSurface, 0.38)
				outline = Alpha(colors.OnSurface, 0.12)
				if s.On {
					thumb = colors.Surface
				}
			}
			rect := image.Rectangle{Max: size}
			full := Uniform(float32(height) / 2)
			// The track changes colour with the state, which the position of
			// the thumb already tells; the colours are mixed by it.
			FillRRect(c.Ops, rect, full, Mix(colors.SurfaceContainerHighest, track, position))
			if !s.On || position < 1 {
				StrokeRRect(c.Ops, rect, full, float32(c.DP(2)), Alpha(outline, 1-position))
			}
			// The thumb is 16 dp, 24 dp when on, 28 dp while pressed.
			diameter := Lerp(Lerp(16, 24, position), 28, grow)
			d := float32(c.DP(1)) * diameter
			travel := float32(width) - 2*float32(c.DP(4)) - float32(c.DP(24))
			centerX := float32(c.DP(4)) + float32(c.DP(24))/2 + travel*position
			cx, cy := int(centerX), height/2
			state := st.press.stateLayer(c, in, Mix(colors.OnSurface, colors.Primary, position))
			if state.A > 0 {
				FillCircle(c.Ops, image.Pt(cx, cy), c.DP(20), state)
			}
			FillCircle(c.Ops, image.Pt(cx, cy), int(d/2), thumb)
			if position > 0.01 {
				icon := c.measure(loose(size), Icon("check", 16, Alpha(colors.OnPrimaryContainer, position), 0))
				icon.put(c, image.Pt(cx-icon.size().X/2, cy-icon.size().Y/2))
			}
			focusRing(c, in, rect, full)
			return layout.Dimensions{Size: size}
		})
	}
}
