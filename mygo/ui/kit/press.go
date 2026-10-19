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

// interaction is what a pressable area is doing.
type interaction struct {
	hovered, pressed, focused, keyboard bool
}

// pressSpec describes a pressable area to the registry and to the keyboard.
type pressSpec struct {
	// id identifies the area between builds, among its siblings; empty for the
	// place in the tree.
	id       any
	role     Role
	name     string
	tooltip  string
	selected bool
	disabled bool
	// onClick runs when the area is clicked, by a pointer, the keyboard or a
	// test.
	onClick func()
}

// hasKey tells whether an id identifies something: nil and the empty string do
// not.
func hasKey(id any) bool {
	if s, ok := id.(string); ok {
		return s != ""
	}
	return id != nil
}

// press builds the element of a pressable area: a row that takes the keyboard
// focus and reports clicks, without a look. It reads what the pointer and the
// keyboard do with it, tells the registry about it, and returns the node, which
// the drawing of the area completes with its position.
func press(c Ctx, spec pressSpec) (ui.Element, interaction, *Node) {
	if hasKey(spec.id) {
		c.Key(spec.id)
	}
	el := ui.ButtonBase(c.Context).FocusRing(false)
	name := spec.name
	if name == "" {
		name = spec.tooltip
	}
	if name != "" {
		el.Label(name)
	}
	var in interaction
	if spec.disabled {
		el.Disabled(true)
	} else {
		in.hovered, in.pressed = el.Hovered(), el.Pressed()
		in.focused = el.Focused()
		in.keyboard = el.FocusVisible()
		if spec.onClick != nil {
			el.OnClick(spec.onClick)
		}
	}
	node := c.Env.Reg.Add(Node{
		Role: spec.role, Name: name, Enabled: !spec.disabled, Selected: spec.selected,
		Tooltip: spec.tooltip, Click: spec.onClick,
	})
	if spec.tooltip != "" && !spec.disabled {
		tooltip(c, el, spec.tooltip)
	}
	return el, in, node
}

// stateLayer is the opacity of the layer that tells the state of an area, drawn
// over it in the colour of its content: 8 % when hovered, 10 % when focused with
// the keyboard or pressed. It follows an effects spring.
func stateLayer(c Ctx, el ui.Element, in interaction) float32 {
	target := float32(0)
	switch {
	case in.pressed || in.keyboard:
		target = 0.10
	case in.hovered:
		target = 0.08
	}
	return el.AnimateTo("layer", target, c.Env.Theme.Motion.FastEffects.Anim())
}

// paintPress draws the state layer of a pressable area, tells the node where
// the area is and draws the ring that shows the keyboard focus around it.
func paintPress(c Ctx, el ui.Element, node *Node, radii Radii, layer float32, content ui.Color, keyboard bool) {
	ring := c.Colors().Secondary
	el.Draw(func(p *ui.Painter, r ui.Rect) {
		if node != nil {
			node.Rect = r
		}
		if layer > 0 {
			fillRRect(p, r, radii.fit(r.W, r.H), content.Alpha(layer))
		}
	})
	if keyboard {
		el.DrawOver(func(p *ui.Painter, r ui.Rect) {
			const offset, width = 2, 3
			grown := ui.Rect{X: r.X - offset, Y: r.Y - offset, W: r.W + 2*offset, H: r.H + 2*offset}
			fit := radii.fit(r.W, r.H)
			strokeRRect(p, grown, Radii{fit[0] + offset, fit[1] + offset, fit[2] + offset, fit[3] + offset}, width, ring)
		})
	}
}

// tooltip shows a tip beside an element when the pointer rests on it, or the
// keyboard focus comes to it: a plain tooltip of Material 3, a small label on
// the inverse surface.
func tooltip(c Ctx, anchor ui.Element, text string) {
	cs := c.Colors()
	ui.TooltipBase(c.Context, anchor, func(tip ui.Element) {
		tip.Margin(6, 0, 6, 0).Padding(4, 8).Radius(CornerExtraSmall).Background(cs.InverseSurface)
		tip.Children(func() {
			Label(c, text, Type(BodySmall), cs.InverseOnSurface, Lines(2)).MaxWidth(240)
		})
	})
}

// Tip shows a tip beside an element when the pointer rests on it, and names the
// element for assistive technology, which also reads the tip.
func Tip(c Ctx, anchor ui.Element, text string) ui.Element {
	anchor.Label(text)
	tooltip(c, anchor, text)
	return anchor
}
