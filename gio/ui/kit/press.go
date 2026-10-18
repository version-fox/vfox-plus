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

	"gioui.org/layout"
	"gioui.org/widget"
)

// pressable is the state of an area that reacts to the pointer and the
// keyboard: a button, a list item, a switch.
type pressable struct {
	click widget.Clickable
	// pointerFocus is set while the focus came from a click, which shows no
	// focus ring.
	pointerFocus bool
	// layer animates the state layer.
	layer Anim
}

// interaction is what a pressable area is doing.
type interaction struct {
	hovered, pressed, focused, keyboard, clicked bool
}

// pressSpec describes a pressable area to the registry and the keyboard.
type pressSpec struct {
	role     Role
	name     string
	tooltip  string
	selected bool
	enabled  bool
	// key identifies the area for popups; nil for none.
	key any
	// onClick runs when the area is clicked, by a pointer, the keyboard or a
	// test.
	onClick func()
}

// layout lays out draw inside the area, which is as large as what it draws.
// A disabled area draws, and does not react.
func (st *pressable) layout(c Ctx, spec pressSpec, draw func(c Ctx, in interaction) layout.Dimensions) layout.Dimensions {
	var in interaction
	if spec.enabled {
		in.clicked = st.click.Clicked(c.Context)
		in.hovered = st.click.Hovered()
		in.pressed = st.click.Pressed()
		in.focused = c.Focused(&st.click)
		if in.pressed {
			st.pointerFocus = true
		}
		if !in.focused {
			st.pointerFocus = false
		}
		in.keyboard = in.focused && !st.pointerFocus
	}
	var dims layout.Dimensions
	if spec.enabled {
		dims = st.click.Layout(c.Context, func(gtx layout.Context) layout.Dimensions {
			inner := c
			inner.Context = gtx
			return draw(inner, in)
		})
	} else {
		dims = draw(c, in)
	}
	name := spec.name
	if name == "" {
		name = spec.tooltip
	}
	c.Env.Reg.Add(Node{
		Key: spec.key, Role: spec.role, Name: name, Rect: image.Rectangle{Max: dims.Size},
		Enabled: spec.enabled, Selected: spec.selected, Hovered: in.hovered, Focused: in.focused,
		Tooltip: spec.tooltip,
		Click: func() {
			st.click.Click()
			c.Env.Services.Invalidate()
		},
	})
	if spec.tooltip != "" {
		c.Env.noteHover(spec.key, spec.tooltip, in.hovered, image.Rectangle{Max: dims.Size}, c.Env.Reg.Mark()-1)
	}
	if in.clicked && spec.onClick != nil {
		spec.onClick()
	}
	return dims
}

// stateLayer is the colour of the layer that tells the state of an area, drawn
// over it in the colour of its content: 8 % when hovered, 10 % when focused
// with the keyboard or pressed.
func (st *pressable) stateLayer(c Ctx, in interaction, content color.NRGBA) color.NRGBA {
	target := 0.0
	switch {
	case in.pressed || in.keyboard:
		target = 0.10
	case in.hovered:
		target = 0.08
	}
	return Alpha(content, float32(st.layer.To(c, target, c.Env.Theme.Motion.FastEffects)))
}

// focusRing draws the ring that shows the keyboard focus around a shape.
func focusRing(c Ctx, in interaction, rect image.Rectangle, r Radii) {
	if !in.keyboard {
		return
	}
	width := float32(c.DP(2))
	grow := c.DP(2)
	outer := image.Rect(rect.Min.X-grow, rect.Min.Y-grow, rect.Max.X+grow, rect.Max.Y+grow)
	StrokeRRect(c.Ops, outer, Radii{r.TopLeft + float32(grow), r.TopRight + float32(grow),
		r.BottomRight + float32(grow), r.BottomLeft + float32(grow)}, width, c.Env.Theme.Colors.Secondary)
}
