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
	"sync"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/version-fox/vfox-plus/gio/ui/shapes"
)

var (
	swatchMorphOnce sync.Once
	swatchMorph     *shapes.Morph
)

// ColorSwatch is a round sample of an accent colour; the one that is selected
// morphs into a flower while it turns, with a check mark in it.
type ColorSwatch struct {
	// ID identifies the swatch between frames, within its scope.
	ID string
	// Seed is the accent colour as ARGB.
	Seed uint32
	// Label names the swatch.
	Label string
	// Selected says that the swatch shows the colour in use.
	Selected bool
	// OnClick runs when the user chooses the swatch.
	OnClick func()
}

type swatchState struct {
	press    pressable
	morph    Anim
	seed     uint32
	dark     bool
	ready    bool
	primary  color.NRGBA
	onColour color.NRGBA
}

// W is the swatch as a widget.
func (s ColorSwatch) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[swatchState](c, s.ID)
		theme := c.Env.Theme
		if !st.ready || st.seed != s.Seed || st.dark != theme.Dark {
			palette := PaletteFor(s.Seed, theme.Dark)
			st.primary, st.onColour = palette.Primary, palette.OnPrimary
			st.seed, st.dark, st.ready = s.Seed, theme.Dark, true
		}
		swatchMorphOnce.Do(func() { swatchMorph = shapes.NewMorph(shapes.ShapeCircle, shapes.ShapeFlower) })
		spec := pressSpec{role: RoleButton, name: s.Label, tooltip: s.Label, selected: s.Selected, enabled: true, key: c.key(s.ID), onClick: s.OnClick}
		return st.press.layout(c, spec, func(c Ctx, in interaction) layout.Dimensions {
			px := c.DP(40)
			size := c.Constraints.Constrain(image.Pt(px, px))
			center := image.Pt(size.X/2, size.Y/2)
			t := st.morph.To(c, boolFloat(s.Selected), theme.Motion.FastSpatial)
			if layer := st.press.stateLayer(c, in, st.primary); layer.A > 0 {
				FillCircle(c.Ops, center, c.DP(24), layer)
			}
			cubics := swatchMorph.AsCubics(clampFloat(t, 0, 1))
			func() {
				path := ShapePath(c.Ops, cubics, f32.Pt(float32(center.X), float32(center.Y)), float32(px), t*0.6)
				defer clip.Outline{Path: path}.Op().Push(c.Ops).Pop()
				paint.Fill(c.Ops, st.primary)
			}()
			if t > 0.01 {
				icon := c.measure(loose(image.Pt(px, px)), Icon("check", 20, Alpha(st.onColour, float32(clampFloat(t, 0, 1))), 0))
				icon.put(c, image.Pt(center.X-icon.size().X/2, center.Y-icon.size().Y/2))
			}
			focusRing(c, in, image.Rect(center.X-px/2, center.Y-px/2, center.X+px/2, center.Y+px/2), Uniform(float32(px)/2))
			return layout.Dimensions{Size: size}
		})
	}
}

func clampFloat(v, low, high float64) float64 {
	switch {
	case v < low:
		return low
	case v > high:
		return high
	default:
		return v
	}
}
