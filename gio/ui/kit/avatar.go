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
	"math"
	"unicode"
	"unicode/utf16"

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op/clip"

	"github.com/version-fox/vfox-plus/gio/ui/shapes"
)

// StableHash is a hash of a text that is the same on every run and in every
// implementation of vfox-plus (FNV-1a over the UTF-16 code units), which
// picks the shape and the colours of an avatar.
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

// Avatar is the avatar of a plugin: its initial on a shape picked by a hash of
// its name, in one of three colour pairs. Hovering spins the shape by 30°.
type Avatar struct {
	// Name is the name that decides the shape and the colours.
	Name string
	// Size is the side of the avatar in dp; 40 by default.
	Size float32
	// Icon replaces the initial with an icon.
	Icon string
}

type avatarState struct {
	hover  gesture.Hover
	rotate Anim
}

// W is the avatar as a widget. id identifies it between frames within its
// scope.
func (a Avatar) W(id string) Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[avatarState](c, id)
		colors := c.Env.Theme.Colors
		sizeDp := a.Size
		if sizeDp == 0 {
			sizeDp = 40
		}
		px := c.DP(sizeDp)
		size := c.Constraints.Constrain(image.Pt(px, px))
		hash := StableHash(a.Name)
		// Avatars avoid the secondary container, which marks selected list
		// items, and the primary container, which some seeds make just as light.
		background, foreground := colors.PrimaryFixedDim, colors.OnPrimaryFixed
		switch hash % 3 {
		case 1:
			background, foreground = colors.Secondary, colors.OnSecondary
		case 2:
			background, foreground = colors.TertiaryContainer, colors.OnTertiaryContainer
		}
		hovered := st.hover.Update(c.Source)
		rotation := st.rotate.To(c, boolFloat(hovered)*math.Pi/6, c.Env.Theme.Motion.FastSpatial)
		FillShape(c.Ops, DecorativeShape(a.Name).Cubics, image.Rectangle{Max: size}, rotation, background)
		area := clip.Rect(image.Rectangle{Max: size}).Push(c.Ops)
		st.hover.Add(c.Ops)
		area.Pop()
		var inner Widget
		if a.Icon != "" {
			inner = Center(Icon(a.Icon, sizeDp/2, foreground, 0))
		} else {
			style := Emph(TitleMedium).WithSize(sizeDp * 0.42).WithWeight(700)
			inner = Center(Line(initial(a.Name), style, foreground))
		}
		p := c.measure(tight(size), inner)
		p.put(c, image.Point{})
		return layout.Dimensions{Size: size}
	}
}

// initial is the first character of the name, in upper case.
func initial(name string) string {
	for _, r := range name {
		return string(unicode.ToUpper(r))
	}
	return "?"
}
