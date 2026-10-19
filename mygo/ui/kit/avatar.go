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
	"math"
	"unicode"
	"unicode/utf16"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// StableHash is a hash of a text that is the same on every run and in every
// implementation of vfox-plus (FNV-1a over the UTF-16 code units), which picks
// the shape and the colours of an avatar.
func StableHash(text string) uint32 {
	hash := uint32(0x811C9DC5)
	for _, unit := range utf16.Encode([]rune(text)) {
		hash ^= uint32(unit)
		hash *= 0x01000193
	}
	return hash
}

// DecorativeShape is the shape that a name gets.
func DecorativeShape(name string) *shapes.Polygon {
	return shapes.AvatarShapes[int(StableHash(name))%len(shapes.AvatarShapes)]
}

// AvatarSpec describes the avatar of a plugin.
type AvatarSpec struct {
	// Name is the name that decides the shape and the colours.
	Name string
	// Size is the side of the avatar in dp; 40 by default.
	Size float32
	// Icon replaces the initial with an icon.
	Icon string
}

// Avatar is the avatar of a plugin: its initial on a shape picked by a hash of
// its name, in one of three colour pairs. Hovering spins the shape by 30°.
func Avatar(c Ctx, a AvatarSpec) ui.Element {
	cs := c.Colors()
	size := a.Size
	if size == 0 {
		size = 40
	}
	hash := StableHash(a.Name)
	// Avatars avoid the secondary container, which marks selected list items,
	// and the primary container, which some seeds make just as light.
	background, foreground := cs.PrimaryFixedDim, cs.OnPrimaryFixed
	switch hash % 3 {
	case 1:
		background, foreground = cs.Secondary, cs.OnSecondary
	case 2:
		background, foreground = cs.TertiaryContainer, cs.OnTertiaryContainer
	}
	box := ui.Box(c.Context).Size(size, size).Shrink(0).Center()
	rotation := box.AnimateTo("spin", Flag(box.Hovered())*math.Pi/6, c.Env.Theme.Motion.FastSpatial.Anim())
	cubics := DecorativeShape(a.Name).Cubics
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		fillShape(p, cubics, r, float64(rotation), background)
	})
	return box.Children(func() {
		if a.Icon != "" {
			Icon(c, a.Icon, size/2, foreground, 0)
			return
		}
		style := Emph(TitleMedium).WithSize(size * 0.42).WithWeight(700)
		Line(c, initial(a.Name), style, foreground)
	})
}

// initial is the first character of the name, in upper case.
func initial(name string) string {
	for _, r := range name {
		return string(unicode.ToUpper(r))
	}
	return "?"
}
