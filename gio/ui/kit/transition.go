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
	"time"

	"gioui.org/layout"
	"gioui.org/op"
)

// Transition is how a view gives way to another one
// (docs/design-system.md#motion).
type Transition int

const (
	// FadeThrough fades the old view out, then fades the new one in while it
	// grows a little: for views that have nothing to do with each other, such as
	// the pages of the navigation.
	FadeThrough Transition = iota
	// SharedAxisX slides the views along the horizontal axis while they fade:
	// for views that belong together, such as a list and the details of one of
	// its items.
	SharedAxisX
)

// The easing curves of the Material motion system.
func accelerateCurve(x float64) float64    { return cubicBezier(0.4, 0, 1, 1, x) }
func decelerateCurve(x float64) float64    { return cubicBezier(0, 0, 0.2, 1, x) }
func fastOutSlowInCurve(x float64) float64 { return cubicBezier(0.4, 0, 0.2, 1, x) }

type transitionState struct {
	key     any
	child   Widget
	prev    Widget
	started time.Time
	reverse bool
	running bool
}

// SwitchView lays out child, the view that belongs to key. When the key changes,
// the view of the old key gives way on a transition, as long as a spring needs
// to settle. reverse says that the user goes back, which mirrors the
// movement of SharedAxisX. Keys must be comparable.
func SwitchView(id string, kind Transition, reverse bool, key any, child Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[transitionState](c, id)
		if st.child != nil && st.key != key && !c.Env.Reduced() {
			st.prev, st.started, st.reverse, st.running = st.child, c.Now, reverse, true
		}
		st.key, st.child = key, child
		if !st.running {
			return child(c)
		}
		duration := c.Env.Theme.Motion.SlowEffects.SettleTime()
		t := float64(c.Now.Sub(st.started)) / float64(duration)
		if t >= 1 || t < 0 {
			st.running, st.prev = false, nil
			return child(c)
		}
		c.Invalidate()
		return st.transition(c, kind, t, child)
	}
}

func (st *transitionState) transition(c Ctx, kind Transition, t float64, child Widget) layout.Dimensions {
	// The old view is below the new one, and ignores the input; its nodes are
	// not in the registry, which lists what the user can reach.
	mark, popups := c.Env.Reg.Mark(), len(c.Env.popups)
	oldMacro := op.Record(c.Ops)
	old := c
	old.Context = c.Disabled()
	st.prev(old)
	oldCall := oldMacro.Stop()
	c.Env.Reg.Truncate(mark)
	// Nor does the old view open menus.
	c.Env.popups = c.Env.popups[:popups]

	newMacro := op.Record(c.Ops)
	dims := child(c)
	newCall := newMacro.Stop()

	fadeOut := float32(1 - accelerateCurve(interval(t, 0, 0.3, func(x float64) float64 { return x })))
	fadeIn := float32(interval(t, 0.3, 1, decelerateCurve))
	switch kind {
	case SharedAxisX:
		direction := float32(1)
		if st.reverse {
			direction = -1
		}
		travel := float32(c.DP(30)) * direction
		move := float32(fastOutSlowInCurve(t))
		Opacity(c.Ops, fadeOut, func() {
			defer translated(c, f32Pt(-travel*move, 0))()
			oldCall.Add(c.Ops)
		})
		Opacity(c.Ops, fadeIn, func() {
			defer translated(c, f32Pt(travel*(1-move), 0))()
			newCall.Add(c.Ops)
		})
	default:
		Opacity(c.Ops, fadeOut, func() { oldCall.Add(c.Ops) })
		scale := 0.92 + 0.08*fadeIn
		pivot := image.Pt(dims.Size.X/2, dims.Size.Y/2)
		Opacity(c.Ops, fadeIn, func() {
			defer scaleAbout(c, pivot, scale)()
			newCall.Add(c.Ops)
		})
	}
	return dims
}

type collapsibleState struct{ open Anim }

// Collapsible shows child under a height that follows a spring between 0 and
// the height of the child, so that content appears and disappears by growing.
func Collapsible(id string, open bool, child Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[collapsibleState](c, id)
		t := float32(st.open.To(c, boolFloat(open), c.Env.Theme.Motion.DefaultSpatial))
		if t <= 0.001 && !open {
			return layout.Dimensions{Size: c.Constraints.Constrain(image.Point{})}
		}
		cs := c.Constraints
		cs.Min.Y = 0
		p := c.measure(cs, child)
		height := int(float32(p.size().Y) * minf(t, 1))
		size := image.Pt(p.size().X, height)
		defer clipRect(c, image.Rectangle{Max: size})()
		p.put(c, image.Point{})
		return layout.Dimensions{Size: size}
	}
}
