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

	"gioui.org/io/event"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

// Overlays are drawn after the interface, on top of it: menus, tooltips and
// dialogs. A widget that opens one only says so while it is laid out; the
// popup is laid out later in the frame (Env.LayoutOverlays), when every widget
// knows where it is, so that it can be put at its anchor and kept inside the
// window.

// popup is a popup that was requested during the layout of a frame.
type popup struct {
	// anchor is the key of the widget the popup opens at.
	anchor any
	// layout draws the popup, given the rectangle of the anchor.
	layout func(c Ctx, anchor image.Rectangle) layout.Dimensions
}

// RequestPopup asks for a popup at the widget with the key anchor.
func (e *Env) RequestPopup(anchor any, draw func(c Ctx, anchor image.Rectangle) layout.Dimensions) {
	e.popups = append(e.popups, popup{anchor: anchor, layout: draw})
}

// hoverState tracks what the pointer rests on, for tooltips.
type hoverState struct {
	key     any
	text    string
	since   time.Time
	rect    image.Rectangle
	node    int
	hovered bool
	frame   uint64
}

// tooltipDelay is how long the pointer rests on a widget before its tooltip
// shows.
const tooltipDelay = 500 * time.Millisecond

// noteHover tells the environment that the pointer is (or is not) over a widget
// that has a tooltip.
func (e *Env) noteHover(key any, text string, hovered bool, rect image.Rectangle, node int) {
	if !hovered {
		if e.hover.key == key && e.hover.key != nil {
			e.hover = hoverState{}
		}
		return
	}
	if e.hover.key != key || e.hover.text != text {
		e.hover = hoverState{key: key, text: text, since: time.Time{}}
	}
	e.hover.rect = rect
	e.hover.node = node
	e.hover.hovered = true
	e.hover.frame = e.frame
}

// LayoutOverlays lays out the popups of the frame, then the tooltip, on top
// of everything laid out so far. It is called with the context of the window.
func (e *Env) LayoutOverlays(c Ctx) {
	for _, p := range e.popups {
		anchor, ok := e.Reg.Find(p.anchor)
		if !ok {
			continue
		}
		p.layout(c, anchor.Rect)
	}
	e.layoutTooltip(c)
	e.layoutDialogs(c)
	e.layoutSnackbar(c)
}

func (e *Env) layoutTooltip(c Ctx) {
	h := &e.hover
	if h.key == nil || h.frame != e.frame || len(e.popups) > 0 {
		return
	}
	if h.since.IsZero() {
		h.since = c.Now
	}
	if wait := tooltipDelay - c.Now.Sub(h.since); wait > 0 {
		c.InvalidateAt(c.Now.Add(wait))
		return
	}
	node, ok := e.Reg.Find(h.key)
	if !ok {
		return
	}
	showTooltip(c, node.Rect, h.text)
}

// Place puts a popup of the given size next to an anchor: below it, aligned to
// its start edge, or its end edge with alignEnd, and moved to stay inside the
// window, above the anchor if it does not fit below.
func Place(window image.Point, anchor image.Rectangle, size image.Point, alignEnd bool, margin int) image.Point {
	x := anchor.Min.X
	if alignEnd {
		x = anchor.Max.X - size.X
	}
	y := anchor.Max.Y
	if y+size.Y > window.Y-margin {
		if above := anchor.Min.Y - size.Y; above >= margin {
			y = above
		} else {
			y = window.Y - margin - size.Y
		}
	}
	x = minInt(maxInt(x, margin), maxInt(margin, window.X-margin-size.X))
	y = maxInt(y, margin)
	return image.Pt(x, y)
}

// scrim covers the window and sends a press on it to onPress, which is how a
// popup dismisses on a click outside of it. It consumes the pointer events of
// everything below it.
func scrim(c Ctx, tag any, onPress func()) {
	defer clip.Rect(image.Rectangle{Max: c.Env.Window}).Push(c.Ops).Pop()
	event.Op(c.Ops, tag)
	for {
		ev, ok := c.Event(pointer.Filter{Target: tag, Kinds: pointer.Press | pointer.Scroll})
		if !ok {
			break
		}
		if e, ok := ev.(pointer.Event); ok && e.Kind == pointer.Press {
			onPress()
		}
	}
}
