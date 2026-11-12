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
	"math"
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

// A label that is told its limit does not grow with the text scale of the system beyond it: at 200 % it is as wide as the same
// word is at the cap of 130 %, and narrower than the one that has none. (Not a ratio of the widths: the width of a word is not
// in proportion to the size of the font in every engine, which sets a word at 18 dp with another tracking than at 28.)
func TestALabelDoesNotGrowBeyondItsScaleCap(t *testing.T) {
	width := func(scale float32, options ...LabelOption) float32 {
		h := drawnAtScale(t, 400, scale, func(c Ctx) {
			ui.Column(c.Context).Padding(16).Children(func() {
				Line(c, "Plugins", Type(LabelMedium), c.Colors().OnSurface, options...)
			})
		})
		for _, n := range h.env.Reg.Nodes() {
			if n.Text == "Plugins" && n.Natural > 0 {
				return n.Natural
			}
		}
		t.Fatalf("no label was drawn at %v", scale)
		return 0
	}
	capped, atCap, uncapped := width(2, ScaleCap(1.3)), width(1.3), width(2)
	if math.Abs(float64(capped-atCap)) > 0.01 {
		t.Errorf("a label that is capped at 130 %% is as wide at 200 %% as the same label at 130 %%: %v and %v", capped, atCap)
	}
	if capped >= uncapped {
		t.Errorf("and it is narrower than the label that has no cap: %v and %v", capped, uncapped)
	}
}

// listItemWith is a build of an item of a list in a window, with a button at its end.
func listItemWith(title, subtitle string) func(c Ctx) {
	return func(c Ctx) {
		ui.Column(c.Context).Background(c.Colors().Surface).Padding(24).Children(func() {
			ListGroup(c, ListItemSpec{
				ID: "item", Title: title, Subtitle: subtitle,
				Trailing: func() { Button(c, ButtonSpec{ID: "show", Label: "Show code", Kind: TonalButton}) },
			})
		})
	}
}

// A word that is longer than the room that its item has, an address in the way that a computer on a network has one, breaks
// over lines where the room ends: it does not stick out of the item or of the window, and it is not cut. The control that is
// at the end of the item goes under it. Where the room is enough, the word is one line, and the control is beside it.
func TestAListItemBreaksAWordThatIsLongerThanItsRoom(t *testing.T) {
	const address = "https://[2001:db8:85a3:8d3:1319:8a2e:370:7334]:8080"
	line := Type(BodyMedium).Line
	for _, scale := range []float32{1, 2} {
		h := drawnAtScale(t, 360, scale, listItemWith("Reachable", address))
		text, button := h.rectOf(RoleText, address), h.rectOf(RoleButton, "Show code")
		if text.H < 2*line*scale-1 {
			t.Errorf("at %.0f %% the address is wider than the item and breaks over lines: %+v", scale*100, text)
		}
		if edge := float32(360 - 24 - 16); text.X+text.W > edge+0.5 {
			t.Errorf("at %.0f %% the address ends at %.0f dp, where the inside of the item ends at %.0f", scale*100, text.X+text.W, edge)
		}
		if button.Y < text.Y+text.H-0.5 {
			t.Errorf("at %.0f %% the button is under the address: %+v and %+v", scale*100, button, text)
		}
		got := TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 360, 600), nil)
		if got.Overflow || len(got.Sticking) != 0 || len(got.Cut) != 0 {
			t.Errorf("at %.0f %% nothing sticks out or is cut: %v %q %q", scale*100, got.Overflowing, got.Sticking, got.Cut)
		}
	}
	wide := drawnAtScale(t, 1280, 1, listItemWith("Reachable", address))
	text, button := wide.rectOf(RoleText, address), wide.rectOf(RoleButton, "Show code")
	if text.H > line+0.5 || button.X < text.X+text.W {
		t.Errorf("in a window of 1280 dp the address is one line and the button is beside it: %+v and %+v", text, button)
	}
}

// A word that fits the room of its item, but not with the control beside it, is not broken to keep the control there: the
// control goes under the word. The room is the least with which a control beside the text would be kept there if the word did
// not count (the text of an item asks for textRoom ems, and the control and the gap), and 8 dp more.
func TestAListItemMovesItsControlUnderAWordBeforeItBreaksTheWord(t *testing.T) {
	const key = "VFOX_JAVA_GITHUB_MIRROR"
	wide := drawnAtScale(t, 1280, 1, listItemWith("Mirror", key))
	word, control := wide.rectOf(RoleText, key).W, wide.rectOf(RoleButton, "Show code").W
	room := textRoom * Type(BodyLarge).Size
	// The inside of the item is 80 dp less than the window (24 and 16 on each side).
	inside := max(word, room+16+control) + 8
	if inside >= word+16+control || word <= room+8 {
		t.Fatalf("the word of %v dp, the control of %v dp and the room of %v dp do not make a room that tells the two apart", word, control, room)
	}
	h := drawnAtScale(t, int(math.Ceil(float64(inside)))+80, 1, listItemWith("Mirror", key))
	text, button := h.rectOf(RoleText, key), h.rectOf(RoleButton, "Show code")
	if text.H > Type(BodyMedium).Line+0.5 {
		t.Errorf("a word that fits the room is not broken to keep the control beside it: %+v", text)
	}
	if button.Y < text.Y+text.H-0.5 || button.X != text.X {
		t.Errorf("the control is under the word, at its start: %+v and %+v", button, text)
	}
}

// What holds a name that is data shows all of it, in as many lines as it needs: a chip, a tag and a dropdown break a word that
// is longer than their room, however many lines that makes, and are not cut.
func TestWhatHoldsADataLabelBreaksTheWordInsteadOfCuttingIt(t *testing.T) {
	const room = 160
	const longWord = "VFOX_JAVA_GITHUB_MIRROR"
	build := func(c Ctx) {
		ui.Column(c.Context).Background(c.Colors().Surface).Padding(16).Gap(8).Children(func() {
			ui.Row(c.Context).Width(room).Wrap().Children(func() { FilterChip(c, ChipSpec{ID: "chip", Label: longWord}) })
			ui.Row(c.Context).Width(room).Wrap().Children(func() { Tag(c, TagSpec{Label: longWord}) })
			Dropdown(c, DropdownSpec{
				ID: "hosts", Value: "v6", Width: 260,
				Choices:  []Choice{{Value: "v4", Label: "192.0.2.17"}, {Value: "v6", Label: "2001:db8:85a3:8d3:1319:8a2e:370"}},
				OnSelect: func(string) {},
			})
		})
	}
	h := drawnAtScale(t, 300, 1, build)
	chip, dropdown := h.rectOf(RoleButton, longWord), h.rectOf(RoleButton, "2001:db8:85a3:8d3:1319:8a2e:370")
	if chip.W > room+0.5 || chip.H <= chipHeight {
		t.Errorf("a chip that holds a word longer than its room is as wide as the room, with a second line: %+v", chip)
	}
	if label := h.rectOf(RoleText, "2001:db8:85a3:8d3:1319:8a2e:370"); label.H < 2*Type(BodyLarge).Line-1 {
		t.Errorf("a dropdown shows an address that is longer than its field in two lines: %+v", label)
	}
	if dropdown.W > 260.5 {
		t.Errorf("and is not wider than it was told to be: %+v", dropdown)
	}
	got := TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 300, 600), nil)
	if len(got.Cut) != 0 || got.Overflow {
		t.Errorf("nothing is cut and nothing sticks out: %q %q", got.Cut, got.Overflowing)
	}

	// A word that needs more lines than two: three in a chip, and four in a dropdown that is 160 dp wide. They are not stopped at
	// two lines with an ellipsis.
	const longer = "VFOX_JAVA_GITHUB_MIRROR_VFOX_JAVA_GITHUB_MIRROR_"
	const address = "2001:db8:85a3:8d3:1319:8a2e:370:7334"
	more := drawnAtScale(t, 300, 1, func(c Ctx) {
		ui.Column(c.Context).Background(c.Colors().Surface).Padding(16).Gap(8).Children(func() {
			ui.Row(c.Context).Width(room).Wrap().Children(func() { FilterChip(c, ChipSpec{ID: "chip", Label: longer}) })
			Dropdown(c, DropdownSpec{
				ID: "hosts", Value: "v6", Width: 160,
				Choices:  []Choice{{Value: "v6", Label: address}},
				OnSelect: func(string) {},
			})
		})
	})
	if chip := more.rectOf(RoleButton, longer); chip.H < 3*Type(LabelLarge).Line {
		t.Errorf("a chip that holds a word of three lines has them: %+v", chip)
	}
	if label := more.rectOf(RoleText, address); label.H < 3*Type(BodyLarge).Line {
		t.Errorf("a dropdown that holds an address of four lines has more than two of them: %+v", label)
	}
	if got := TakeCensus(more.env.Reg.Nodes(), rect(0, 0, 300, 600), nil); len(got.Cut) != 0 || got.Overflow {
		t.Errorf("and nothing is cut: %q %q", got.Cut, got.Overflowing)
	}
}
