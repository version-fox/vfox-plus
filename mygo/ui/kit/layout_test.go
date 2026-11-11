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
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// drawnAtScale is a build that is drawn in a window of width dp, with the text scale of the system.
func drawnAtScale(t *testing.T, width int, scale float32, build func(c Ctx)) *tested {
	t.Helper()
	h := newTested(t, width, 600, build)
	h.tester.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: scale})
	h.env.Theme = NewTheme(testSeed, false, false, scale)
	h.tester.Frame()
	h.tester.Advance(time.Second)
	return h
}

func (h *tested) rectOf(role Role, name string) ui.Rect {
	h.t.Helper()
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == role && (n.Name == name || n.Text == name) {
			return n.Rect
		}
	}
	h.t.Fatalf("no %s %q", role, name)
	return ui.Rect{}
}

// An item of a list keeps its control beside the text while there is room for both, and moves it under the text, at the start
// of the text, when there is not: at the size of the text that the system asks for.
func TestAListItemMovesItsControlUnderItsTextWhenThereIsNoRoomBesideIt(t *testing.T) {
	item := func(c Ctx) {
		ui.Column(c.Context).Background(c.Colors().Surface).Padding(24).Children(func() {
			ListGroup(c, ListItemSpec{
				ID: "health", Title: "No problems found.",
				Leading:  func(col ui.Color) { Icon(c, "check", 24, col, 0) },
				Trailing: func() { Button(c, ButtonSpec{ID: "check", Label: "Check health", Kind: TextButton}) },
			})
		})
	}
	beside := drawnAtScale(t, 360, 1, item)
	title, button := beside.rectOf(RoleText, "No problems found."), beside.rectOf(RoleButton, "Check health")
	if button.X < title.X+title.W || button.Y >= title.Y+title.H {
		t.Errorf("at 100 %% the button is beside the text: %+v and %+v", button, title)
	}
	under := drawnAtScale(t, 360, 1.5, item)
	title, button = under.rectOf(RoleText, "No problems found."), under.rectOf(RoleButton, "Check health")
	if button.Y < title.Y+title.H || button.X != title.X {
		t.Errorf("at 150 %% the button is under the text, at its start: %+v and %+v", button, title)
	}
	got := TakeCensus(under.env.Reg.Nodes(), rect(0, 0, 360, 600), nil)
	if len(got.Sticking) != 0 || got.Overflow {
		t.Errorf("and it is inside the padding of the row: %q", got.Sticking)
	}
}

// The actions of the header of a page are beside its title where there is room, and under it where there is not.
func TestThePageHeaderPutsItsActionsUnderTheTitleWhereThereIsNoRoomBesideIt(t *testing.T) {
	header := func(c Ctx) {
		ui.Column(c.Context).Background(c.Colors().Surface).Children(func() {
			PageHeader(c, "Plugins", "", nil,
				func() { IconButton(c, IconButtonSpec{ID: "reload", Icon: "refresh", Tooltip: "Reload"}) },
				func() { Button(c, ButtonSpec{ID: "add", Label: "Add from source…", Kind: TonalButton, Icon: "add"}) })
		})
	}
	wide := drawnAtScale(t, 700, 1, header)
	title, add := wide.rectOf(RoleText, "Plugins"), wide.rectOf(RoleButton, "Add from source…")
	if add.X < title.X+title.W || add.Y >= title.Y+title.H {
		t.Errorf("in a window of 700 dp the actions are beside the title: %+v and %+v", add, title)
	}
	narrow := drawnAtScale(t, 360, 1.3, header)
	title, add = narrow.rectOf(RoleText, "Plugins"), narrow.rectOf(RoleButton, "Add from source…")
	if add.Y < title.Y+title.H || add.X+add.W > 360 {
		t.Errorf("in a window of 360 dp at 130 %% they are under it, in the window: %+v and %+v", add, title)
	}
}

// A group of segments that does not fit the width that it has, even when it gives up all it can, is a dropdown if it may be.
func TestAButtonGroupThatMayCollapseIsADropdownWhenItDoesNotFit(t *testing.T) {
	segments := []Segment{{Value: "system", Label: "System"}, {Value: "english", Label: "English"}, {Value: "chinese", Label: "简体中文"}}
	group := func(room float32, collapse bool) func(c Ctx) {
		return func(c Ctx) {
			ui.Column(c.Context).Padding(16).Children(func() {
				ButtonGroup(c, ButtonGroupSpec{ID: "language", Segments: segments, Selected: "english", OnSelect: func(string) {}, MaxWidth: room, Collapse: collapse})
			})
		}
	}
	buttons := func(h *tested) (names []string) {
		for _, n := range h.env.Reg.Nodes() {
			if n.Role == RoleButton {
				names = append(names, n.Name)
			}
		}
		return names
	}
	if got := buttons(drawnAtScale(t, 400, 1, group(300, true))); len(got) != 3 {
		t.Errorf("in the room of 300 dp the three segments fit: %q", got)
	}
	if got := buttons(drawnAtScale(t, 400, 1, group(150, true))); len(got) != 1 || got[0] != "English" {
		t.Errorf("in the room of 150 dp the group is a dropdown that shows the choice: %q", got)
	}
	if got := buttons(drawnAtScale(t, 400, 1, group(150, false))); len(got) != 3 {
		t.Errorf("a group that may not collapse stays a group: %q", got)
	}
}

// A label that is told its limit does not grow with the text scale of the system beyond it: the same word, at 200 % and at the
// cap of 130 %, is 130 / 200 as wide.
func TestALabelDoesNotGrowBeyondItsScaleCap(t *testing.T) {
	h := drawnAtScale(t, 400, 2, func(c Ctx) {
		ui.Column(c.Context).Padding(16).Gap(8).Children(func() {
			Line(c, "Plugins", Type(LabelMedium), c.Colors().OnSurface)
			Line(c, "Plugins", Type(LabelMedium), c.Colors().OnSurface, ScaleCap(1.3))
		})
	})
	var widths []float32
	for _, n := range h.env.Reg.Nodes() {
		if n.Text == "Plugins" {
			widths = append(widths, n.Natural)
		}
	}
	if len(widths) != 2 || widths[0] <= 0 {
		t.Fatalf("two labels were drawn: %v", widths)
	}
	if ratio := widths[1] / widths[0]; ratio < 0.64 || ratio > 0.66 {
		t.Errorf("the label with a cap is %.2f of the other, want 0.65: %v", ratio, widths)
	}
}
