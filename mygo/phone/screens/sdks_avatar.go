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

package screens

import (
	"unicode"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/ui/kit"
	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// sdkNameHash is the hash that picks the colour and the shape of the avatar of a
// plugin. It is the hash of the web interface (mygo/web/src/dom.ts, hashName):
// h = h*31 + the code point, over the code points of the name, in 32 bits. The
// toolkit's own avatars hash the name differently (kit.StableHash), so the phone
// does not use them: a plugin has the same colour here and on the web.
func sdkNameHash(name string) uint32 {
	var h uint32
	for _, c := range name {
		h = h*31 + uint32(c)
	}
	return h
}

// sdkTone is the colour of an avatar, from 0 to 2: the web's av-p, av-s and av-t,
// which are chosen by the hash modulo 3.
func sdkTone(hash uint32) int {
	return int(hash % 3)
}

// sdkShape is the shape of an avatar. The web gives an avatar one of twelve border
// radii (av-0 to av-11), chosen by the hash modulo 12; the toolkit has twelve
// shapes of its own, which the phone takes in the same order. The shapes are not
// the same figures as the radii.
func sdkShape(hash uint32) *shapes.Polygon {
	return shapes.AvatarShapes[hash%uint32(len(shapes.AvatarShapes))]
}

// sdkInitial is the letter of an avatar: the first character of the name in upper
// case, or "?" for a name that is empty.
func sdkInitial(name string) string {
	for _, c := range name {
		return string(unicode.ToUpper(c))
	}
	return "?"
}

// sdkTonePair returns the background and the content colour of a tone. The primary
// tone uses the palette's fixed primary colours, which are the web's primary-fixed
// ones. The palette has no fixed secondary or tertiary colours, so those tones use
// the containers, which are the same colours in the light scheme; their content
// colour is the palette's, not the web's fixed one.
func sdkTonePair(cs *kit.Palette, tone int) (background, foreground ui.Color) {
	switch tone {
	case 1:
		return cs.SecondaryContainer, cs.OnSecondaryContainer
	case 2:
		return cs.TertiaryContainer, cs.OnTertiaryContainer
	}
	return cs.PrimaryFixed, cs.OnPrimaryFixed
}

// sdkAvatar is the avatar of a plugin: the initial of its name, on a shape and in
// a colour that the name's hash picks, as the web interface draws it.
func sdkAvatar(k kit.Ctx, name string, size float32) ui.Element {
	hash := sdkNameHash(name)
	background, foreground := sdkTonePair(k.Colors(), sdkTone(hash))
	style := kit.Emph(kit.TitleMedium).WithSize(kit.ScaleSize(size * 0.42)).WithWeight(700)
	letter := sdkInitial(name)
	return kit.Shape(k, sdkShape(hash), size, background, func() {
		kit.Line(k, letter, style, foreground, kit.InitialText())
	})
}
