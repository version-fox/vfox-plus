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
	"time"

	"gioui.org/f32"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	"github.com/version-fox/vfox-plus/gio/assets"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
)

// harness lays out widgets frame after frame without a window, and feeds them
// input, like the window does.
type harness struct {
	t      *testing.T
	env    *kit.Env
	router input.Router
	ops    op.Ops
	size   image.Point
	now    time.Time
	// invalidations counts the requests for another frame.
	invalidations int
	copied        []string
	opened        []string
}

type services struct{ h *harness }

func (s services) Invalidate()      { s.h.invalidations++ }
func (s services) Copy(text string) { s.h.copied = append(s.h.copied, text) }
func (s services) OpenURL(a string) { s.h.opened = append(s.h.opened, a) }

func newHarness(t *testing.T) *harness {
	t.Helper()
	iconFont, err := assets.Files.ReadFile("fonts/MaterialSymbolsRounded.ttf")
	if err != nil {
		t.Fatal(err)
	}
	codes, err := assets.Files.ReadFile("icons.json")
	if err != nil {
		t.Fatal(err)
	}
	fonts, err := kit.LoadFonts(iconFont, codes)
	if err != nil {
		t.Fatal(err)
	}
	h := &harness{t: t, size: image.Pt(800, 600), now: time.Unix(1000, 0)}
	h.env = kit.NewEnv(kit.NewTheme(0xFFF28C1B, false, false, fonts), services{h})
	h.env.Reg.CollectText = true
	return h
}

// ctx makes the context of a frame.
func (h *harness) ctx(constraints layout.Constraints) kit.Ctx {
	gtx := layout.Context{
		Constraints: constraints,
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Now:         h.now,
		Source:      h.router.Source(),
		Ops:         &h.ops,
	}
	return kit.Ctx{Context: gtx, Env: h.env}
}

// frame lays out w in the window, 16 ms after the frame before.
func (h *harness) frame(w kit.Widget) {
	h.t.Helper()
	h.frameIn(layout.Exact(h.size), w)
}

func (h *harness) frameIn(constraints layout.Constraints, w kit.Widget) layout.Dimensions {
	h.t.Helper()
	h.now = h.now.Add(16 * time.Millisecond)
	h.ops.Reset()
	h.env.BeginFrame(h.size)
	c := h.ctx(constraints)
	dims := w(c)
	h.env.LayoutOverlays(c)
	h.router.Frame(&h.ops)
	return dims
}

// frames lays out w a number of times.
func (h *harness) frames(n int, w kit.Widget) {
	h.t.Helper()
	for range n {
		h.frame(w)
	}
}

// click presses and releases the primary button of the mouse at p.
func (h *harness) click(p image.Point) {
	at := f32.Pt(float32(p.X), float32(p.Y))
	h.router.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: at},
		pointer.Event{Kind: pointer.Press, Source: pointer.Mouse, Position: at, Buttons: pointer.ButtonPrimary},
		pointer.Event{Kind: pointer.Release, Source: pointer.Mouse, Position: at},
	)
}

// scroll turns the wheel by dy pixels at p.
func (h *harness) scroll(p image.Point, dy float32) {
	at := f32.Pt(float32(p.X), float32(p.Y))
	h.router.Queue(
		pointer.Event{Kind: pointer.Move, Source: pointer.Mouse, Position: at},
		pointer.Event{Kind: pointer.Scroll, Source: pointer.Mouse, Position: at, Scroll: f32.Pt(0, dy)},
	)
}

// node finds the node with the role and the name in the last frame.
func (h *harness) node(role kit.Role, name string) (kit.Node, bool) {
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == role && n.Name == name {
			return n, true
		}
	}
	return kit.Node{}, false
}

// mustNode is node that fails the test when there is none.
func (h *harness) mustNode(role kit.Role, name string) kit.Node {
	h.t.Helper()
	n, ok := h.node(role, name)
	if !ok {
		h.t.Fatalf("no %s %q in %d nodes", role, name, len(h.env.Reg.Nodes()))
	}
	return n
}

// hasText reports whether a label shows exactly text.
func (h *harness) hasText(text string) bool {
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == kit.RoleText && n.Text == text {
			return true
		}
	}
	return false
}

// fixed is a widget of a given size that draws nothing.
func fixed(width, height int) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions {
		return layout.Dimensions{Size: c.Constraints.Constrain(image.Pt(width, height))}
	}
}
