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

	"gioui.org/gesture"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

// A vertical scrolling list that lays out only the items in view, for the
// long lists (the 800 versions of Node.js) and, with a single item, for
// whole pages.

// ListSource provides the items of a list.
type ListSource struct {
	// Count is the number of items.
	Count int
	// Key returns a key of the item that stays the same when the list changes,
	// for remembering its height; the index when nil.
	Key func(i int) any
	// Item lays out the item with the index.
	Item func(c Ctx, i int) layout.Dimensions
	// Shrink makes the list only as high as its items, up to the space it
	// is given.
	Shrink bool
	// StickToEnd keeps the end of the items in view while they grow, until
	// the user scrolls up: the output of a command. It is meant for a list
	// of a few long items.
	StickToEnd bool
}

// scrollState is the state of a scrolling list.
type scrollState struct {
	// first is the index of the first item in view, offset how many pixels of
	// it are scrolled out of view.
	first, offset int
	scroll        gesture.Scroll
	heights       map[any]int
	width         int
	known         int
	knownSum      int
	// reveal asks the next frame to scroll by that many pixels, to show the
	// widget in focus.
	reveal  int
	atStart bool
	atEnd   bool
	// pinned says that the end is in view, and stays in view when the
	// items grow; initialised tells that the list has been laid out.
	pinned, initialised bool
	// thumb fades in while the list scrolls.
	thumb      Anim
	lastScroll uint64
	// total estimates the height of all items.
	total int
}

// List lays out a scrolling list that fills the space it is given, which must
// be bounded in height.
func List(id string, src ListSource) Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[scrollState](c, id)
		cs := c.Constraints
		size := cs.Max
		if unbounded(size.Y) {
			// No bounded height to scroll in: the list is as tall as its items.
			size.Y = 1 << 20
		}
		if st.heights == nil || st.width != size.X {
			st.heights = make(map[any]int)
			st.width = size.X
			st.known, st.knownSum = 0, 0
		}
		key := func(i int) any {
			if src.Key != nil {
				return src.Key(i)
			}
			return i
		}
		height := func(i int) (int, bool) {
			h, ok := st.heights[key(i)]
			return h, ok
		}
		average := func() int {
			if st.known == 0 {
				return c.DP(72)
			}
			return st.knownSum / st.known
		}

		// The input of this frame.
		delta := st.scroll.Update(c.Metric, c.Source, c.Now, gesture.Vertical,
			pointer.ScrollRange{}, pointer.ScrollRange{Min: -boolRange(!st.atStart), Max: boolRange(!st.atEnd)})
		st.offset += delta + st.reveal
		if delta != 0 {
			st.lastScroll = c.Env.frame
		}
		st.reveal = 0
		if src.StickToEnd {
			if delta < 0 {
				st.pinned = false
			}
			if st.pinned || !st.initialised {
				// Far beyond the end: the loop below brings it back to it.
				st.first, st.offset = maxInt(0, src.Count-1), 1<<30
			}
		}
		st.initialised = true

		type visible struct {
			index int
			y     int
			p     placed
		}
		var items []visible
		itemCs := layout.Constraints{Min: image.Pt(size.X, 0), Max: image.Pt(size.X, int(inf))}
		fill := func() (endY int) {
			items = items[:0]
			y := -st.offset
			for i := st.first; i < src.Count && y < size.Y; i++ {
				p := c.measure(itemCs, func(c Ctx) layout.Dimensions { return src.Item(c, i) })
				h := p.size().Y
				if old, ok := height(i); ok {
					st.knownSum += h - old
				} else {
					st.known++
					st.knownSum += h
				}
				st.heights[key(i)] = h
				items = append(items, visible{i, y, p})
				y += h
			}
			return y
		}
		// normalize makes the first item the one at the top of the list: it
		// moves back through the items before it when the list was scrolled up
		// beyond the first, and skips the items that are entirely above the list
		// when it was scrolled down, by their heights when they are known, and
		// by the average height when the jump is far, which spares measuring
		// the items in between.
		normalize := func() {
			for st.offset < 0 && st.first > 0 {
				st.first--
				h, ok := height(st.first)
				if !ok {
					h = average()
				}
				st.offset += h
			}
			if st.offset < 0 {
				st.offset = 0
			}
			for st.first+1 < src.Count {
				h, ok := height(st.first)
				if !ok {
					if st.offset <= 2*size.Y {
						break
					}
					h = maxInt(1, average())
				}
				if st.offset < h {
					break
				}
				st.offset -= h
				st.first++
			}
		}
		mark := c.Env.Reg.Mark()
		for attempt := 0; attempt < 8; attempt++ {
			normalize()
			c.Env.Reg.Truncate(mark)
			endY := fill()
			adjusted := false
			switch {
			case len(items) > 0 && st.offset >= items[0].p.size().Y && st.first+1 < src.Count:
				// The first item is entirely above the list: its height was not
				// known yet.
				for _, item := range items {
					if st.offset < item.p.size().Y || st.first+1 >= src.Count {
						break
					}
					st.offset -= item.p.size().Y
					st.first++
				}
				adjusted = true
			case endY < size.Y && (st.first > 0 || st.offset > 0):
				// The items end above the bottom of the list: scroll back.
				st.offset -= size.Y - endY
				adjusted = true
			}
			if !adjusted {
				break
			}
		}
		st.atStart = st.first == 0 && st.offset == 0
		defer func() {
			if src.StickToEnd && st.atEnd {
				st.pinned = true
			}
		}()
		last := items
		st.atEnd = len(last) == 0 || last[len(last)-1].index == src.Count-1 && last[len(last)-1].y+last[len(last)-1].p.size().Y <= size.Y
		contentHeight := 0
		for _, v := range items {
			contentHeight = maxInt(contentHeight, v.y+v.p.size().Y)
		}
		out := image.Pt(size.X, minInt(size.Y, maxInt(cs.Min.Y, contentHeight)))
		switch {
		case cs.Max.Y >= inf, src.Shrink:
			out.Y = minInt(size.Y, contentHeight)
		default:
			out.Y = size.Y
		}
		// Estimate the total for the scroll bar from the average height.
		before := st.first * average()
		st.total = src.Count * average()
		if st.total < before+contentHeight {
			st.total = before + contentHeight
		}

		// Draw, clipped to the list, with the area that receives the scroll
		// below the items.
		defer clip.Rect(image.Rectangle{Max: out}).Push(c.Ops).Pop()
		st.scroll.Add(c.Ops)
		for _, v := range items {
			v.p.put(c, image.Pt(0, v.y))
		}
		// Keep the widget in focus in view.
		for _, n := range c.Env.Reg.Nodes()[mark:] {
			if !n.Focused {
				continue
			}
			switch {
			case n.Rect.Min.Y < 0:
				st.reveal = n.Rect.Min.Y - c.DP(8)
				c.Invalidate()
			case n.Rect.Max.Y > out.Y:
				st.reveal = n.Rect.Max.Y - out.Y + c.DP(8)
				c.Invalidate()
			}
			break
		}
		// The scroll bar: a thin thumb that shows while the list moves.
		if st.total > out.Y {
			visibleFor := c.Env.frame - st.lastScroll
			target := 0.0
			if st.lastScroll != 0 && visibleFor < 90 {
				target = 1
			}
			opacity := st.thumb.To(c, target, c.Env.Theme.Motion.DefaultEffects)
			if visibleFor < 90 && st.lastScroll != 0 {
				c.Invalidate()
			}
			if opacity > 0.01 {
				thumb := maxInt(c.DP(32), out.Y*out.Y/st.total)
				scrolled := before + st.offset
				track := out.Y - thumb
				pos := 0
				if st.total > out.Y {
					pos = track * scrolled / (st.total - out.Y)
				}
				pos = minInt(maxInt(pos, 0), track)
				width := c.DP(4)
				rect := image.Rect(out.X-width-c.DP(2), pos, out.X-c.DP(2), pos+thumb)
				FillRRect(c.Ops, rect, Uniform(float32(width)/2), Alpha(c.Env.Theme.Colors.OnSurfaceVariant, float32(opacity)*0.5))
			}
		}
		return layout.Dimensions{Size: out}
	}
}

func boolRange(b bool) int {
	if b {
		return 1 << 20
	}
	return 0
}

// Scroll makes a page that is taller than its space scroll.
func Scroll(id string, child Widget) Widget {
	return List(id, ListSource{Count: 1, Item: func(c Ctx, _ int) layout.Dimensions { return child(c) }})
}
