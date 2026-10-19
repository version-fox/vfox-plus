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
	"sync"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

var (
	swatchMorphOnce sync.Once
	swatchMorph     *shapes.Morph

	seedColors sync.Map // seedKey -> [2]ui.Color
)

type seedKey struct {
	seed uint32
	dark bool
}

// primaryOf is the primary colour of the scheme of a seed colour, and the colour
// of what is on it. Generating a scheme is costly, and the same seeds come every
// frame, so they are kept.
func primaryOf(seed uint32, dark bool) (primary, onPrimary ui.Color) {
	if cached, ok := seedColors.Load(seedKey{seed, dark}); ok {
		colours := cached.([2]ui.Color)
		return colours[0], colours[1]
	}
	palette := PaletteFor(seed, dark)
	seedColors.Store(seedKey{seed, dark}, [2]ui.Color{palette.Primary, palette.OnPrimary})
	return palette.Primary, palette.OnPrimary
}

// SwatchSpec describes a colour swatch.
type SwatchSpec struct {
	// ID identifies the swatch among its siblings between builds.
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

// ColorSwatch is a round sample of an accent colour; the one that is selected
// morphs into a flower while it turns, with a check mark in it.
func ColorSwatch(c Ctx, s SwatchSpec) ui.Element {
	theme := c.Env.Theme
	primary, onPrimary := primaryOf(s.Seed, theme.Dark)
	swatchMorphOnce.Do(func() { swatchMorph = shapes.NewMorph(shapes.ShapeCircle, shapes.ShapeFlower) })
	el, in, node := press(c, pressSpec{
		id: s.ID, role: RoleButton, name: s.Label, tooltip: s.Label, selected: s.Selected, onClick: s.OnClick,
	})
	t := el.AnimateTo("morph", Flag(s.Selected), theme.Motion.FastSpatial.Anim())
	layer := stateLayer(c, el, in)
	cubics := swatchMorph.AsCubics(float64(clamp01(t)))
	rotation := float64(t) * 0.6
	keyboard := in.keyboard
	ring := c.Colors().Secondary
	el.Size(40, 40).Shrink(0)
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		node.Rect = r
		if layer > 0 {
			cx, cy := r.X+r.W/2, r.Y+r.H/2
			p.Fill(ui.Rect{X: cx - 24, Y: cy - 24, W: 48, H: 48}, primary.Alpha(layer), 24)
		}
		fillShape(p, cubics, r, rotation, primary)
	})
	if keyboard {
		el.DrawOver(func(p *ui.Painter, r ui.Rect) {
			strokeRRect(p, ui.Rect{X: r.X - 2, Y: r.Y - 2, W: r.W + 4, H: r.H + 4}, Uniform(22), 3, ring)
		})
	}
	return el.Children(func() {
		if t > 0.01 {
			Icon(c, "check", 20, onPrimary.Alpha(clamp01(t)), 0)
		}
	})
}
