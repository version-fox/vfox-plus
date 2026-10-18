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

	"gioui.org/op"
)

// showTooltip draws the plain tooltip of Material 3 below an anchor: a small
// inverse surface with a line of text.
func showTooltip(c Ctx, anchor image.Rectangle, text string) {
	colors := c.Env.Theme.Colors
	pad := c.DP(8)
	padY := c.DP(4)
	maxWidth := c.DP(280)
	label := Label(text, Type(BodySmall), colors.InverseOnSurface)
	cs := loose(image.Pt(maxWidth, c.DP(200)))
	p := c.measure(cs, label)
	size := p.size().Add(image.Pt(2*pad, 2*padY))
	// Centred below the anchor, inside the window.
	at := image.Pt(anchor.Min.X+anchor.Dx()/2-size.X/2, anchor.Max.Y+c.DP(4))
	if at.Y+size.Y > c.Env.Window.Y {
		at.Y = anchor.Min.Y - c.DP(4) - size.Y
	}
	at.X = minInt(maxInt(at.X, c.DP(8)), maxInt(c.DP(8), c.Env.Window.X-c.DP(8)-size.X))
	at.Y = maxInt(at.Y, 0)
	defer op.Offset(at).Push(c.Ops).Pop()
	FillRRect(c.Ops, image.Rectangle{Max: size}, c.Round(CornerExtraSmall), colors.InverseSurface)
	p.put(c, image.Pt(pad, padY))
}
