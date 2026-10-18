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
	"strconv"
	"testing"

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/ui/kit"
)

// rows is a list of items of the same height, each of which registers a node
// named by its index, and notes that it was laid out.
func rows(count, height int, laid map[int]bool) kit.ListSource {
	return kit.ListSource{
		Count: count,
		Item: func(c kit.Ctx, i int) layout.Dimensions {
			if laid != nil {
				laid[i] = true
			}
			size := c.Constraints.Constrain(image.Pt(c.Constraints.Max.X, height))
			c.Env.Reg.Add(kit.Node{Role: kit.RoleListItem, Name: strconv.Itoa(i), Rect: image.Rectangle{Max: size}})
			return layout.Dimensions{Size: size}
		},
	}
}

// firstVisible is the index of the first item that is in view.
func (h *harness) firstVisible(viewport image.Rectangle) int {
	first := -1
	for _, n := range h.env.Reg.Nodes() {
		if n.Role != kit.RoleListItem || !n.Rect.Overlaps(viewport) {
			continue
		}
		if index, err := strconv.Atoi(n.Name); err == nil && (first == -1 || index < first) {
			first = index
		}
	}
	return first
}

func TestListLaysOutOnlyTheItemsInViewAndScrolls(t *testing.T) {
	h := newHarness(t)
	laid := map[int]bool{}
	viewport := image.Rect(0, 0, 400, 200)
	list := kit.List("l", rows(1000, 50, laid))
	run := func(n int) {
		for range n {
			h.frameIn(layout.Exact(viewport.Size()), list)
		}
	}
	run(1)
	if len(laid) > 6 {
		t.Fatalf("a thousand items are not all laid out: %d", len(laid))
	}
	if h.firstVisible(viewport) != 0 {
		t.Errorf("got %d", h.firstVisible(viewport))
	}
	// The wheel moves the list by what it turns.
	h.scroll(image.Pt(100, 100), 520)
	run(3)
	if got := h.firstVisible(viewport); got != 10 {
		t.Errorf("got %d", got)
	}
	// Scrolling back brings the first items back, and the list does not go past
	// its start.
	h.scroll(image.Pt(100, 100), -100000)
	run(3)
	if got := h.firstVisible(viewport); got != 0 {
		t.Errorf("got %d", got)
	}
	n := h.mustNode(kit.RoleListItem, "0")
	if n.Rect.Min.Y != 0 {
		t.Errorf("the first item is at the top: %v", n.Rect)
	}
	// And the end of it is not passed either.
	h.scroll(image.Pt(100, 100), 10000000)
	run(3)
	last := h.mustNode(kit.RoleListItem, "999")
	if last.Rect.Max.Y != viewport.Dy() {
		t.Errorf("the last item ends at the bottom: %v", last.Rect)
	}
}

func TestListRemembersTheHeightOfItemsAndCopesWithVariableOnes(t *testing.T) {
	h := newHarness(t)
	source := kit.ListSource{
		Count: 200,
		Key:   func(i int) any { return "k" + strconv.Itoa(i) },
		Item: func(c kit.Ctx, i int) layout.Dimensions {
			height := 20 + i%5*10
			size := c.Constraints.Constrain(image.Pt(c.Constraints.Max.X, height))
			c.Env.Reg.Add(kit.Node{Role: kit.RoleListItem, Name: strconv.Itoa(i), Rect: image.Rectangle{Max: size}})
			return layout.Dimensions{Size: size}
		},
	}
	viewport := image.Rect(0, 0, 300, 150)
	list := kit.List("var", source)
	run := func(n int) {
		for range n {
			h.frameIn(layout.Exact(viewport.Size()), list)
		}
	}
	run(1)
	for range 30 {
		h.scroll(image.Pt(50, 50), 137)
		run(2)
	}
	if got := h.firstVisible(viewport); got < 20 {
		t.Errorf("the list scrolled down, got %d", got)
	}
	for range 60 {
		h.scroll(image.Pt(50, 50), -91)
		run(2)
	}
	if got := h.firstVisible(viewport); got != 0 {
		t.Errorf("the list scrolled back to the top, got %d", got)
	}
}

func TestListShrinksToItsItemsOrTakesTheSpaceItIsGiven(t *testing.T) {
	h := newHarness(t)
	source := rows(3, 40, nil)
	if dims := h.frameIn(loose(400, 600), kit.List("a", source)); dims.Size.Y != 600 {
		t.Errorf("a list takes its space, got %v", dims.Size)
	}
	source.Shrink = true
	if dims := h.frameIn(loose(400, 600), kit.List("b", source)); dims.Size.Y != 120 {
		t.Errorf("a list that shrinks is as high as its items, got %v", dims.Size)
	}
	if dims := h.frameIn(loose(400, 60), kit.List("c", source)); dims.Size.Y != 60 {
		t.Errorf("but not higher than the space, got %v", dims.Size)
	}
	unbounded := layout.Constraints{Max: image.Pt(400, 1000000)}
	if dims := h.frameIn(unbounded, kit.List("d", rows(3, 40, nil))); dims.Size.Y != 120 {
		t.Errorf("in unbounded space, got %v", dims.Size)
	}
	if dims := h.frameIn(loose(400, 100), kit.Scroll("s", fixed(400, 500))); dims.Size.Y != 100 {
		t.Errorf("a scrolling page takes its space, got %v", dims.Size)
	}
	if dims := h.frameIn(loose(400, 100), kit.List("empty", kit.ListSource{Shrink: true})); dims.Size.Y != 0 {
		t.Errorf("an empty list, got %v", dims.Size)
	}
}

func TestListThatSticksToTheEndFollowsGrowingContentUntilTheUserScrollsUp(t *testing.T) {
	h := newHarness(t)
	lines := 30
	list := kit.List("log", kit.ListSource{
		Count: 1, Shrink: true, StickToEnd: true,
		Item: func(c kit.Ctx, _ int) layout.Dimensions {
			size := c.Constraints.Constrain(image.Pt(c.Constraints.Max.X, lines*10))
			c.Env.Reg.Add(kit.Node{Role: kit.RoleListItem, Name: "log", Rect: image.Rectangle{Max: size}})
			return layout.Dimensions{Size: size}
		},
	})
	constraints := layout.Exact(image.Pt(300, 100))
	h.frameIn(constraints, list)
	h.frameIn(constraints, list)
	// The content is 300 px high in 100 px: its end is in view.
	if got := h.mustNode(kit.RoleListItem, "log").Rect.Max.Y; got != 100 {
		t.Errorf("got %d", got)
	}
	lines = 60
	h.frameIn(constraints, list)
	h.frameIn(constraints, list)
	if got := h.mustNode(kit.RoleListItem, "log").Rect.Max.Y; got != 100 {
		t.Errorf("the end follows the content, got %d", got)
	}
	h.scroll(image.Pt(50, 50), -150)
	h.frameIn(constraints, list)
	h.frameIn(constraints, list)
	lines = 90
	h.frameIn(constraints, list)
	h.frameIn(constraints, list)
	if got := h.mustNode(kit.RoleListItem, "log").Rect.Max.Y; got <= 100 {
		t.Errorf("once the user scrolled up, the list stays where it is, got %d", got)
	}
}
