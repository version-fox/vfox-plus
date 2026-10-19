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

// Icon draws an icon of the Material Symbols Rounded font, by its meaning (a
// key of shared/icons.json), in a square of size dp. fill, from 0 (outlined)
// to 1 (filled), is the fill axis of the font: the icon of the selected
// destination of the navigation is filled. A fill between the two cross-fades
// the outlined and the filled icon.
func Icon(c Ctx, name string, size float32, col ui.Color, fill float32) ui.Element {
	glyph := iconGlyph(name)
	fill = clamp01(fill)
	switch {
	case fill < 0.005:
		return iconText(c, glyph, iconFamily, size, col)
	case fill > 0.995:
		return iconText(c, glyph, iconFilledFamily, size, col)
	}
	box := ui.Box(c.Context).Size(size, size).Shrink(0)
	return box.Children(func() {
		iconText(c, glyph, iconFamily, size, col).Opacity(1 - fill)
		iconText(c, glyph, iconFilledFamily, size, col).Opacity(fill).Absolute().Top(0).Left(0)
	})
}

func iconText(c Ctx, glyph, family string, size float32, col ui.Color) ui.Element {
	return ui.Text(c.Context, glyph).Font(family).FontSize(size).FixedLineHeight(size).
		TextColor(col).NoWrap().Size(size, size).Shrink(0)
}
