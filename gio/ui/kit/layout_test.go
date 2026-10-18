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

package kit_test

import (
	"image"
	"testing"

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/ui/kit"
)

func loose(w, h int) layout.Constraints { return layout.Constraints{Max: image.Pt(w, h)} }

func TestRowAndColumnSizeThemselvesToTheirChildren(t *testing.T) {
	h := newHarness(t)
	row := kit.Row(kit.FlexOpts{Min: true, Gap: 10}, kit.Rigid(fixed(30, 20)), kit.Rigid(fixed(40, 50)))
	if dims := h.frameIn(loose(500, 500), row); dims.Size != image.Pt(80, 50) {
		t.Errorf("row: got %v", dims.Size)
	}
	column := kit.Column(kit.FlexOpts{Min: true, Gap: 4}, kit.Rigid(fixed(30, 20)), kit.Rigid(fixed(40, 50)))
	if dims := h.frameIn(loose(500, 500), column); dims.Size != image.Pt(40, 74) {
		t.Errorf("column: got %v", dims.Size)
	}
}

func TestFlexSharesTheSpaceThatRigidChildrenLeave(t *testing.T) {
	h := newHarness(t)
	var widths [3]int
	probe := func(i int) kit.Widget {
		return func(c kit.Ctx) layout.Dimensions {
			widths[i] = c.Constraints.Max.X
			return layout.Dimensions{Size: image.Pt(c.Constraints.Min.X, 10)}
		}
	}
	row := kit.Row(kit.FlexOpts{}, kit.Rigid(fixed(100, 10)), kit.Expanded(1, probe(0)), kit.Expanded(3, probe(1)), kit.Flexible(1, probe(2)))
	dims := h.frameIn(loose(600, 100), row)
	// 500 px are left: 100, 300 and 100.
	if widths != [3]int{100, 300, 100} {
		t.Errorf("got %v", widths)
	}
	if dims.Size.X != 600 {
		t.Errorf("a row takes its space, got %v", dims.Size)
	}
}

func TestFlexAlignsChildren(t *testing.T) {
	h := newHarness(t)
	probe := func(id string) kit.Widget {
		return func(c kit.Ctx) layout.Dimensions {
			size := c.Constraints.Constrain(image.Pt(20, 10))
			c.Env.Reg.Add(kit.Node{Role: kit.RoleButton, Name: id, Rect: image.Rectangle{Max: size}})
			return layout.Dimensions{Size: size}
		}
	}
	row := kit.Row(kit.FlexOpts{Main: kit.MainEnd, Cross: kit.CrossCenter},
		kit.Rigid(probe("a")), kit.Rigid(fixed(20, 50)), kit.Rigid(probe("b")))
	h.frameIn(loose(200, 100), row)
	// The row is 200 wide and 50 high; the children take 60 of the width.
	if got := h.mustNode(kit.RoleButton, "a").Rect; got != image.Rect(140, 20, 160, 30) {
		t.Errorf("a: got %v", got)
	}
	if got := h.mustNode(kit.RoleButton, "b").Rect; got != image.Rect(180, 20, 200, 30) {
		t.Errorf("b: got %v", got)
	}
	space := kit.Row(kit.FlexOpts{Main: kit.MainSpaceBetween}, kit.Rigid(probe("c")), kit.Rigid(probe("d")))
	h.frameIn(loose(100, 100), space)
	if got := h.mustNode(kit.RoleButton, "d").Rect; got != image.Rect(80, 0, 100, 10) {
		t.Errorf("d: got %v", got)
	}
}

func TestStretchedChildrenTakeTheHeightOfTheTallest(t *testing.T) {
	h := newHarness(t)
	var heights [2]int
	probe := func(i int) kit.Widget {
		return func(c kit.Ctx) layout.Dimensions {
			heights[i] = c.Constraints.Min.Y
			return layout.Dimensions{Size: c.Constraints.Constrain(image.Pt(10, 5))}
		}
	}
	// In unbounded height, as in a list: like Flutter's IntrinsicHeight.
	unbounded := layout.Constraints{Max: image.Pt(200, 1000000)}
	dims := h.frameIn(unbounded, kit.Row(kit.FlexOpts{Cross: kit.CrossStretch, Min: true}, kit.Rigid(fixed(10, 40)), kit.Rigid(probe(0)), kit.Rigid(probe(1))))
	if heights != [2]int{40, 40} || dims.Size.Y != 40 {
		t.Errorf("got %v and %v", heights, dims.Size)
	}
}

func TestPadAlignAndSizedPlaceTheirChild(t *testing.T) {
	h := newHarness(t)
	var rect image.Rectangle
	probe := func(c kit.Ctx) layout.Dimensions {
		size := c.Constraints.Constrain(image.Pt(30, 20))
		c.Env.Reg.Add(kit.Node{Role: kit.RoleButton, Name: "x", Rect: image.Rectangle{Max: size}})
		return layout.Dimensions{Size: size}
	}
	dims := h.frameIn(loose(300, 300), kit.Pad(kit.Inset{Left: 5, Top: 7, Right: 11, Bottom: 13}, probe))
	rect = h.mustNode(kit.RoleButton, "x").Rect
	if dims.Size != image.Pt(46, 40) || rect != image.Rect(5, 7, 35, 27) {
		t.Errorf("pad: %v %v", dims.Size, rect)
	}
	h.frameIn(loose(300, 200), kit.Center(probe))
	if rect = h.mustNode(kit.RoleButton, "x").Rect; rect != image.Rect(135, 90, 165, 110) {
		t.Errorf("center: %v", rect)
	}
	h.frameIn(loose(300, 200), kit.Align(layout.SE, probe))
	if rect = h.mustNode(kit.RoleButton, "x").Rect; rect != image.Rect(270, 180, 300, 200) {
		t.Errorf("south east: %v", rect)
	}
	dims = h.frameIn(loose(300, 200), kit.Sized(100, 0, kit.Sized(0, 40, probe)))
	if dims.Size != image.Pt(100, 40) {
		t.Errorf("sized: %v", dims.Size)
	}
}

func TestLimitConstrainsItsChild(t *testing.T) {
	h := newHarness(t)
	var got layout.Constraints
	probe := func(c kit.Ctx) layout.Dimensions {
		got = c.Constraints
		return layout.Dimensions{Size: c.Constraints.Min}
	}
	h.frameIn(loose(500, 400), kit.Limit(50, 60, 300, 200, probe))
	if got.Min != image.Pt(50, 60) || got.Max != image.Pt(300, 200) {
		t.Errorf("got %v", got)
	}
	h.frameIn(loose(100, 100), kit.MaxWidth(300, probe))
	if got.Max.X != 100 {
		t.Errorf("a limit never exceeds the space, got %v", got)
	}
	h.frameIn(loose(500, 400), kit.FillWidth(probe))
	if got.Min.X != 500 {
		t.Errorf("got %v", got)
	}
	h.frameIn(loose(500, 400), kit.Fill(probe))
	if got.Min != image.Pt(500, 400) {
		t.Errorf("got %v", got)
	}
	h.frameIn(layout.Exact(image.Pt(90, 80)), kit.Loose(probe))
	if got.Min != (image.Point{}) || got.Max != image.Pt(90, 80) {
		t.Errorf("got %v", got)
	}
	if dims := h.frameIn(loose(500, 400), kit.Space(12, 8)); dims.Size != image.Pt(12, 8) {
		t.Errorf("space: %v", dims.Size)
	}
}

func TestStackLaysChildrenOnTopOfEachOther(t *testing.T) {
	h := newHarness(t)
	probe := func(name string, w, hgt int) kit.Widget {
		return func(c kit.Ctx) layout.Dimensions {
			size := c.Constraints.Constrain(image.Pt(w, hgt))
			c.Env.Reg.Add(kit.Node{Role: kit.RoleButton, Name: name, Rect: image.Rectangle{Max: size}})
			return layout.Dimensions{Size: size}
		}
	}
	stack := kit.Stack(layout.Center,
		kit.Stacked(probe("big", 100, 60)),
		kit.Stacked(probe("small", 20, 20)),
		kit.Expand(probe("fill", 1, 1)),
		kit.PlacedAt(func(stack, child image.Point) image.Point { return stack.Sub(child) }, probe("corner", 10, 10)),
	)
	dims := h.frameIn(loose(300, 300), stack)
	if dims.Size != image.Pt(100, 60) {
		t.Fatalf("got %v", dims.Size)
	}
	for name, want := range map[string]image.Rectangle{
		"small":  image.Rect(40, 20, 60, 40),
		"fill":   image.Rect(0, 0, 100, 60),
		"corner": image.Rect(90, 50, 100, 60),
	} {
		if got := h.mustNode(kit.RoleButton, name).Rect; got != want {
			t.Errorf("%s: got %v, want %v", name, got, want)
		}
	}
}

func TestWrapStartsANewRowWhereOneIsFull(t *testing.T) {
	h := newHarness(t)
	wrap := kit.Wrap{Spacing: 10, RunSpacing: 5}.Layout(fixed(40, 20), fixed(40, 30), fixed(40, 10), fixed(40, 10))
	// 100 px fit two children of 40 and the space between them.
	if dims := h.frameIn(loose(100, 500), wrap); dims.Size != image.Pt(90, 30+5+10) {
		t.Errorf("got %v", dims.Size)
	}
	centered := kit.Wrap{Align: kit.MainCenter, CrossCenter: true}.Layout(fixed(40, 20), fixed(40, 30))
	if dims := h.frameIn(loose(100, 500), centered); dims.Size != image.Pt(80, 30) {
		t.Errorf("got %v", dims.Size)
	}
}

func TestRegistryMovesTheNodesOfAChildWithIt(t *testing.T) {
	h := newHarness(t)
	probe := func(c kit.Ctx) layout.Dimensions {
		c.Env.Reg.Add(kit.Node{Role: kit.RoleButton, Name: "deep", Rect: image.Rect(0, 0, 10, 10)})
		return layout.Dimensions{Size: image.Pt(10, 10)}
	}
	nested := kit.Pad(kit.All(10), kit.Column(kit.FlexOpts{Min: true}, kit.Rigid(fixed(5, 30)), kit.Rigid(kit.Pad(kit.Inset{Left: 20, Top: 3}, probe))))
	h.frameIn(loose(300, 300), nested)
	// 10 of padding, 30 of the first child, 3 and 20 of the inner padding.
	if got := h.mustNode(kit.RoleButton, "deep").Rect; got != image.Rect(30, 43, 40, 53) {
		t.Errorf("got %v", got)
	}
	reg := h.env.Reg
	mark := reg.Mark()
	reg.Add(kit.Node{Name: "extra", Rect: image.Rect(0, 0, 4, 4)})
	reg.Shift(mark, image.Pt(7, 8))
	reg.Shift(mark, image.Point{})
	if got, ok := reg.Find(nil); !ok || got.Name != "extra" || got.Rect != image.Rect(7, 8, 11, 12) {
		t.Errorf("got %v %v", got, ok)
	}
	reg.Truncate(mark)
	if len(reg.Nodes()) != mark {
		t.Errorf("got %d nodes", len(reg.Nodes()))
	}
	if !kit.Visible(kit.Node{Rect: image.Rect(10, 10, 20, 20)}, image.Pt(15, 15)) || kit.Visible(kit.Node{Rect: image.Rect(10, 10, 20, 20)}, image.Pt(5, 5)) {
		t.Error("visibility")
	}
	if kit.Visible(kit.Node{}, image.Pt(5, 5)) {
		t.Error("an empty node is not visible")
	}
}
