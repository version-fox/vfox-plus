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

// SwitchSpec describes a switch.
type SwitchSpec struct {
	// ID identifies the switch among its siblings between builds.
	ID string
	// On says whether the switch is on.
	On bool
	// OnChange runs with the new state when the user flips the switch.
	OnChange func(on bool)
	// Disabled makes the switch inert.
	Disabled bool
	// Label names the switch for assistive technology.
	Label string
}

const (
	switchWidth  = 52
	switchHeight = 32
)

// Switch is a Material 3 switch: a track with a thumb that slides, and grows
// while it is pressed.
func Switch(c Ctx, s SwitchSpec) ui.Element {
	theme := c.Env.Theme
	cs := theme.Colors
	inert := s.Disabled || s.OnChange == nil
	if hasKey(s.ID) {
		c.Key(s.ID)
	}
	on := s.On
	el := ui.SwitchBase(c.Context, &on).FocusRing(false)
	if s.Label != "" {
		el.Label(s.Label)
	}
	var in interaction
	if inert {
		el.Disabled(true)
	} else {
		in.hovered, in.pressed, in.keyboard = el.Hovered(), el.Pressed(), el.FocusVisible()
		el.OnChange(func() { s.OnChange(on) })
	}
	node := c.Env.Reg.Add(Node{
		Role: RoleSwitch, Name: s.Label, Enabled: !inert, Selected: s.On,
		Click: func() {
			if s.OnChange != nil {
				s.OnChange(!s.On)
			}
		},
	})
	motion := theme.Motion.FastSpatial.Anim()
	position := el.AnimateTo("slide", Flag(s.On), motion)
	grow := el.AnimateTo("grow", Flag(in.pressed), motion)
	layer := stateLayer(c, el, in)
	track, thumb, outline := cs.SurfaceContainerHighest, cs.Outline, cs.Outline
	if s.On {
		track, thumb = cs.Primary, cs.OnPrimary
	}
	if s.Disabled {
		track = cs.OnSurface.Alpha(0.12)
		thumb = cs.OnSurface.Alpha(0.38)
		outline = cs.OnSurface.Alpha(0.12)
		if s.On {
			thumb = cs.Surface
		}
	}
	// The thumb is 16 dp, 24 dp when on, 28 dp while pressed.
	diameter := Lerp(Lerp(16, 24, position), 28, grow)
	centreX := 4 + 24.0/2 + (switchWidth-2*4-24)*position
	layerColour := cs.OnSurface.Mix(cs.Primary, position)
	el.Size(switchWidth, switchHeight).Shrink(0)
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		node.Rect = r
		full := Uniform(r.H / 2)
		// The track changes colour with the state, which the position of the
		// thumb already tells; the colours are mixed by it.
		fillRRect(p, r, full, cs.SurfaceContainerHighest.Mix(track, position))
		if !s.On || position < 1 {
			strokeRRect(p, r, full, 2, outline.Alpha(1-position))
		}
		cx, cy := r.X+centreX, r.Y+r.H/2
		if layer > 0 {
			p.Fill(ui.Rect{X: cx - 20, Y: cy - 20, W: 40, H: 40}, layerColour.Alpha(layer), 20)
		}
		p.Fill(ui.Rect{X: cx - diameter/2, Y: cy - diameter/2, W: diameter, H: diameter}, thumb, diameter/2)
	})
	if in.keyboard {
		el.DrawOver(func(p *ui.Painter, r ui.Rect) {
			strokeRRect(p, ui.Rect{X: r.X - 2, Y: r.Y - 2, W: r.W + 4, H: r.H + 4}, Uniform(r.H/2+2), 3, cs.Secondary)
		})
	}
	return el.Children(func() {
		if position > 0.01 {
			Icon(c, "check", 16, cs.OnPrimaryContainer.Alpha(position), 0).Absolute().Left(centreX - 8).Top(switchHeight/2 - 8)
		}
	})
}
