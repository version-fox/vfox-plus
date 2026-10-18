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

	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
)

// The layout library: rows, columns, stacks, padding and alignment, with the
// semantics of Flutter's boxes: a parent gives a child constraints, the child
// picks a size within them and the parent places it. Every container tells
// the registry where it put each child (see Registry.Shift), so that the
// registry knows where every widget is when the frame is complete.

// inf is the constraint that means unbounded, as in Gio's layout.List.
const inf = 1e6

// unbounded reports whether a maximum is the unbounded one.
func unbounded(max int) bool { return max >= inf }

func dpUnit(v float32) unit.Dp { return unit.Dp(v) }
func spUnit(v float32) unit.Sp { return unit.Sp(v) }

// placed is a child that was laid out into a macro and waits for its position.
type placed struct {
	call     op.CallOp
	dims     layout.Dimensions
	from, to int
}

// measure lays out w within cs without drawing it yet.
func (c Ctx) measure(cs layout.Constraints, w Widget) placed {
	from := c.Env.Reg.Mark()
	macro := op.Record(c.Ops)
	child := c
	child.Constraints = cs
	dims := w(child)
	dims.Size = cs.Constrain(dims.Size)
	call := macro.Stop()
	return placed{call: call, dims: dims, from: from, to: c.Env.Reg.Mark()}
}

// put draws the child at the position, and moves its nodes with it.
func (p placed) put(c Ctx, at image.Point) {
	c.Env.Reg.ShiftRange(p.from, p.to, at)
	defer op.Offset(at).Push(c.Ops).Pop()
	p.call.Add(c.Ops)
}

// size is the size of the child.
func (p placed) size() image.Point { return p.dims.Size }

func loose(max image.Point) layout.Constraints { return layout.Constraints{Max: max} }

func tight(size image.Point) layout.Constraints { return layout.Constraints{Min: size, Max: size} }

func within(min, max image.Point) layout.Constraints {
	return layout.Constraints{Min: min, Max: max}
}

func axisPoint(axis layout.Axis, main, cross int) image.Point {
	if axis == layout.Horizontal {
		return image.Pt(main, cross)
	}
	return image.Pt(cross, main)
}

func axisMain(axis layout.Axis, p image.Point) int {
	if axis == layout.Horizontal {
		return p.X
	}
	return p.Y
}

func axisCross(axis layout.Axis, p image.Point) int {
	if axis == layout.Horizontal {
		return p.Y
	}
	return p.X
}

// MainAlign says where the children of a flex go along its main axis.
type MainAlign int

// The alignments along the main axis.
const (
	MainStart MainAlign = iota
	MainCenter
	MainEnd
	MainSpaceBetween
)

// CrossAlign says where the children of a flex go across its main axis.
type CrossAlign int

// The alignments across the main axis.
const (
	CrossStart CrossAlign = iota
	CrossCenter
	CrossEnd
	// CrossStretch makes the children as wide (or high) as the flex.
	CrossStretch
)

// FlexOpts are the options of a row or a column.
type FlexOpts struct {
	// Main is where the children go along the main axis when there is
	// space left.
	Main MainAlign
	// Cross is where the children go across it.
	Cross CrossAlign
	// Gap is the space between the children, in dp.
	Gap float32
	// Min makes the flex only as long as its children; by default it takes
	// all the space along the main axis.
	Min bool
}

// FlexChild is a child of a row or a column.
type FlexChild struct {
	w     Widget
	flex  float32
	tight bool
}

// Rigid is a child that has its own size along the main axis.
func Rigid(w Widget) FlexChild { return FlexChild{w: w} }

// Expanded is a child that takes a share of the space that the rigid children
// leave, in proportion to flex.
func Expanded(flex float32, w Widget) FlexChild { return FlexChild{w: w, flex: flex, tight: true} }

// Flexible is a child that may take a share of the space, but can be smaller.
func Flexible(flex float32, w Widget) FlexChild { return FlexChild{w: w, flex: flex} }

// Row lays out its children from left to right.
func Row(o FlexOpts, children ...FlexChild) Widget { return flexLayout(layout.Horizontal, o, children) }

// Column lays out its children from top to bottom.
func Column(o FlexOpts, children ...FlexChild) Widget {
	return flexLayout(layout.Vertical, o, children)
}

func flexLayout(axis layout.Axis, o FlexOpts, all []FlexChild) Widget {
	inner := flexLayoutBounded(axis, o, all)
	return func(c Ctx) layout.Dimensions {
		if o.Cross != CrossStretch || !unbounded(axisCross(axis, c.Constraints.Max)) {
			return inner(c)
		}
		// Children stretch to the tallest (or widest) of them, as in
		// Flutter's IntrinsicHeight: lay them out once to find its size, then
		// again, tightly.
		mark := c.Env.Reg.Mark()
		macro := op.Record(c.Ops)
		probe := inner(c)
		macro.Stop()
		c.Env.Reg.Truncate(mark)
		cs := c.Constraints
		cross := axisCross(axis, probe.Size)
		if axis == layout.Horizontal {
			cs.Min.Y, cs.Max.Y = cross, cross
		} else {
			cs.Min.X, cs.Max.X = cross, cross
		}
		c.Constraints = cs
		return inner(c)
	}
}

func flexLayoutBounded(axis layout.Axis, o FlexOpts, all []FlexChild) Widget {
	return func(c Ctx) layout.Dimensions {
		var children []FlexChild
		for _, child := range all {
			if child.w != nil {
				children = append(children, child)
			}
		}
		cs := c.Constraints
		mainMax := axisMain(axis, cs.Max)
		crossMax := axisCross(axis, cs.Max)
		crossMin := axisCross(axis, cs.Min)
		gap := c.DP(o.Gap)
		totalGap := 0
		if len(children) > 1 {
			totalGap = gap * (len(children) - 1)
		}
		stretch := o.Cross == CrossStretch && !unbounded(crossMax)
		childCross := 0
		if stretch {
			childCross = crossMax
		}
		measured := make([]placed, len(children))
		used := totalGap
		var flexTotal float32
		for i, child := range children {
			if child.flex > 0 && !unbounded(mainMax) {
				flexTotal += child.flex
				continue
			}
			measured[i] = c.measure(layout.Constraints{
				Min: axisPoint(axis, 0, childCross),
				Max: axisPoint(axis, int(inf), crossMax),
			}, child.w)
			used += axisMain(axis, measured[i].size())
		}
		if flexTotal > 0 {
			space := mainMax - used
			if space < 0 {
				space = 0
			}
			given := 0
			remaining := flexTotal
			for i, child := range children {
				if child.flex <= 0 || unbounded(mainMax) {
					continue
				}
				share := int(float32(space-given) * child.flex / remaining)
				remaining -= child.flex
				given += share
				minMain := 0
				if child.tight {
					minMain = share
				}
				measured[i] = c.measure(layout.Constraints{
					Min: axisPoint(axis, minMain, childCross),
					Max: axisPoint(axis, share, crossMax),
				}, child.w)
				used += axisMain(axis, measured[i].size())
			}
		}
		cross := crossMin
		for _, p := range measured {
			if v := axisCross(axis, p.size()); v > cross {
				cross = v
			}
		}
		if stretch {
			cross = crossMax
		}
		mainSize := used
		if !o.Min && !unbounded(mainMax) {
			mainSize = mainMax
		}
		if mainMin := axisMain(axis, cs.Min); mainSize < mainMin {
			mainSize = mainMin
		}
		leftover := mainSize - used
		if leftover < 0 {
			leftover = 0
		}
		pos := 0
		between := 0
		switch o.Main {
		case MainCenter:
			pos = leftover / 2
		case MainEnd:
			pos = leftover
		case MainSpaceBetween:
			if len(children) > 1 {
				between = leftover / (len(children) - 1)
			}
		}
		for _, p := range measured {
			at := 0
			switch o.Cross {
			case CrossCenter:
				at = (cross - axisCross(axis, p.size())) / 2
			case CrossEnd:
				at = cross - axisCross(axis, p.size())
			}
			p.put(c, axisPoint(axis, pos, at))
			pos += axisMain(axis, p.size()) + gap + between
		}
		return layout.Dimensions{Size: cs.Constrain(axisPoint(axis, mainSize, cross))}
	}
}

// StackChild is a child of a stack.
type StackChild struct {
	w     Widget
	kind  int
	place func(stack, child image.Point) image.Point
}

// Stacked is a child that has its own size, which the stack grows to hold.
func Stacked(w Widget) StackChild { return StackChild{w: w} }

// Expand is a child that is as large as the stack.
func Expand(w Widget) StackChild { return StackChild{w: w, kind: 1} }

// PlacedAt is a child that has its own size, which does not count for the size
// of the stack, and that place puts where it wants: it gets the size of the
// stack and of the child.
func PlacedAt(place func(stack, child image.Point) image.Point, w Widget) StackChild {
	return StackChild{w: w, kind: 2, place: place}
}

// Stack lays out its children on top of each other, the last one on top.
// Stacked children are positioned by align inside the stack.
func Stack(align layout.Direction, children ...StackChild) Widget {
	return func(c Ctx) layout.Dimensions {
		cs := c.Constraints
		size := cs.Min
		measured := make([]placed, len(children))
		for i, child := range children {
			if child.w == nil || child.kind != 0 {
				continue
			}
			measured[i] = c.measure(loose(cs.Max), child.w)
			size = image.Pt(max(size.X, measured[i].size().X), max(size.Y, measured[i].size().Y))
		}
		size = cs.Constrain(size)
		for i, child := range children {
			if child.w == nil {
				continue
			}
			switch child.kind {
			case 0:
				measured[i].put(c, align.Position(measured[i].size(), size))
			case 1:
				p := c.measure(tight(size), child.w)
				p.put(c, image.Point{})
			case 2:
				p := c.measure(loose(cs.Max), child.w)
				p.put(c, child.place(size, p.size()))
			}
		}
		return layout.Dimensions{Size: size}
	}
}

// Inset is padding, in dp.
type Inset struct{ Left, Top, Right, Bottom float32 }

// All is the same padding on every side.
func All(v float32) Inset { return Inset{v, v, v, v} }

// Sym is horizontal and vertical padding.
func Sym(horizontal, vertical float32) Inset {
	return Inset{horizontal, vertical, horizontal, vertical}
}

// Pad pads its child.
func Pad(in Inset, w Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		left, top, right, bottom := c.DP(in.Left), c.DP(in.Top), c.DP(in.Right), c.DP(in.Bottom)
		cs := c.Constraints
		inner := layout.Constraints{
			Min: image.Pt(max(0, cs.Min.X-left-right), max(0, cs.Min.Y-top-bottom)),
			Max: image.Pt(subtractUnbounded(cs.Max.X, left+right), subtractUnbounded(cs.Max.Y, top+bottom)),
		}
		p := c.measure(inner, w)
		p.put(c, image.Pt(left, top))
		size := cs.Constrain(p.size().Add(image.Pt(left+right, top+bottom)))
		return layout.Dimensions{Size: size, Baseline: p.dims.Baseline + top}
	}
}

func subtractUnbounded(max, v int) int {
	if unbounded(max) {
		return max
	}
	if max-v < 0 {
		return 0
	}
	return max - v
}

// Align puts its child in the space it is given, as the direction says. The
// child may be smaller than the space.
func Align(d layout.Direction, w Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		cs := c.Constraints
		p := c.measure(loose(cs.Max), w)
		size := cs.Max
		if unbounded(size.X) {
			size.X = p.size().X
		}
		if unbounded(size.Y) {
			size.Y = p.size().Y
		}
		size = cs.Constrain(size)
		p.put(c, d.Position(p.size(), size))
		return layout.Dimensions{Size: size}
	}
}

// Center puts its child in the middle of the space it is given.
func Center(w Widget) Widget { return Align(layout.Center, w) }

// Sized gives its child a fixed size, in dp. A zero width or height leaves that
// dimension to the child.
func Sized(width, height float32, w Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		cs := c.Constraints
		min, max := cs.Min, cs.Max
		if width > 0 {
			px := cs.Constrain(image.Pt(c.DP(width), 0)).X
			min.X, max.X = px, px
		}
		if height > 0 {
			px := cs.Constrain(image.Pt(0, c.DP(height))).Y
			min.Y, max.Y = px, px
		}
		p := c.measure(within(min, max), w)
		p.put(c, image.Point{})
		return layout.Dimensions{Size: p.size(), Baseline: p.dims.Baseline}
	}
}

// Limit constrains its child to at most maxWidth and maxHeight, in dp (zero means
// no limit) and at least minWidth and minHeight.
func Limit(minWidth, minHeight, maxWidth, maxHeight float32, w Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		cs := c.Constraints
		min, max := cs.Min, cs.Max
		if maxWidth > 0 {
			max.X = minInt(max.X, c.DP(maxWidth))
		}
		if maxHeight > 0 {
			max.Y = minInt(max.Y, c.DP(maxHeight))
		}
		if minWidth > 0 {
			min.X = minInt(max.X, maxInt(min.X, c.DP(minWidth)))
		}
		if minHeight > 0 {
			min.Y = minInt(max.Y, maxInt(min.Y, c.DP(minHeight)))
		}
		min.X, min.Y = minInt(min.X, max.X), minInt(min.Y, max.Y)
		p := c.measure(layout.Constraints{Min: min, Max: max}, w)
		p.put(c, image.Point{})
		return layout.Dimensions{Size: p.size(), Baseline: p.dims.Baseline}
	}
}

// MaxWidth is Limit with only a maximum width, in dp.
func MaxWidth(width float32, w Widget) Widget { return Limit(0, 0, width, 0, w) }

// Loose lets its child be smaller than the space it is given.
func Loose(w Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		cs := c.Constraints
		cs.Min = image.Point{}
		c.Constraints = cs
		return w(c)
	}
}

// Fill makes its child as large as the space it is given, where that is
// bounded.
func Fill(w Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		cs := c.Constraints
		if !unbounded(cs.Max.X) {
			cs.Min.X = cs.Max.X
		}
		if !unbounded(cs.Max.Y) {
			cs.Min.Y = cs.Max.Y
		}
		c.Constraints = cs
		return w(c)
	}
}

// FillWidth makes its child as wide as the space it is given, where that is
// bounded.
func FillWidth(w Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		cs := c.Constraints
		if !unbounded(cs.Max.X) {
			cs.Min.X = cs.Max.X
		}
		c.Constraints = cs
		return w(c)
	}
}

// Space is empty space of the given size, in dp.
func Space(width, height float32) Widget {
	return func(c Ctx) layout.Dimensions {
		return layout.Dimensions{Size: c.Constraints.Constrain(image.Pt(c.DP(width), c.DP(height)))}
	}
}

// Wrap lays out its children in rows, starting a new row where one is full.
type Wrap struct {
	// Spacing is the space between the children of a row, in dp.
	Spacing float32
	// RunSpacing is the space between the rows, in dp.
	RunSpacing float32
	// Align puts the rows horizontally: MainStart, MainCenter or MainEnd.
	Align MainAlign
	// CrossCenter centres the children of a row vertically, instead of
	// putting them at its top.
	CrossCenter bool
}

// Layout lays out the children.
func (wr Wrap) Layout(children ...Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		cs := c.Constraints
		maxWidth := cs.Max.X
		spacing, runSpacing := c.DP(wr.Spacing), c.DP(wr.RunSpacing)
		type run struct {
			children []placed
			width    int
			height   int
		}
		var runs []run
		current := run{}
		for _, w := range children {
			if w == nil {
				continue
			}
			p := c.measure(loose(image.Pt(maxWidth, cs.Max.Y)), w)
			next := current.width + p.size().X
			if len(current.children) > 0 {
				next += spacing
			}
			if len(current.children) > 0 && !unbounded(maxWidth) && next > maxWidth {
				runs = append(runs, current)
				current = run{}
				next = p.size().X
			}
			current.children = append(current.children, p)
			current.width = next
			current.height = max(current.height, p.size().Y)
		}
		if len(current.children) > 0 {
			runs = append(runs, current)
		}
		width, y := 0, 0
		for _, r := range runs {
			width = max(width, r.width)
		}
		if !unbounded(maxWidth) && wr.Align != MainStart {
			width = max(width, cs.Min.X)
		}
		for i, r := range runs {
			x := 0
			switch wr.Align {
			case MainCenter:
				x = (width - r.width) / 2
			case MainEnd:
				x = width - r.width
			}
			for _, p := range r.children {
				at := 0
				if wr.CrossCenter {
					at = (r.height - p.size().Y) / 2
				}
				p.put(c, image.Pt(x, y+at))
				x += p.size().X + spacing
			}
			y += r.height
			if i < len(runs)-1 {
				y += runSpacing
			}
		}
		return layout.Dimensions{Size: cs.Constrain(image.Pt(width, y))}
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
