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
	"math"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// tested is a window without a desktop around it: a tester that draws build, with
// the springs of the desktop reduced so that tests need not wait for them.
type tested struct {
	t      *testing.T
	env    *Env
	tester *ui.Tester
}

func newTested(t *testing.T, width, height int, build func(c Ctx)) *tested {
	t.Helper()
	env := testEnv(t, false)
	tester := ui.NewTester(view(env, build), width, height)
	tester.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 1})
	return &tested{t: t, env: env, tester: tester}
}

func (h *tested) click(name string) {
	h.t.Helper()
	if err := h.tester.Click(name); err != nil {
		h.t.Fatalf("click %q: %v (on screen %q)", name, err, h.tester.Texts())
	}
	h.tester.Advance(time.Second)
}

func (h *tested) mustText(text string) {
	h.t.Helper()
	if !h.tester.HasText(text) {
		h.t.Fatalf("no text %q; there is %q", text, h.tester.Texts())
	}
}

func (h *tested) node(role Role, name string) (*Node, bool) {
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == role && n.Name == name {
			return n, true
		}
	}
	return nil, false
}

func TestButtonsRunTheirActionOnAClickAndNotWhenDisabled(t *testing.T) {
	var runs []string
	h := newTested(t, 600, 200, func(c Ctx) {
		Row(c).Gap(8).Padding(16).Children(func() {
			Button(c, ButtonSpec{Label: "Run", OnClick: func() { runs = append(runs, "run") }})
			Button(c, ButtonSpec{Label: "Tonal", Kind: TonalButton, Icon: "add", OnClick: func() { runs = append(runs, "tonal") }})
			Button(c, ButtonSpec{Label: "Off", Disabled: true, OnClick: func() { runs = append(runs, "off") }})
			Button(c, ButtonSpec{Label: "Remove", Danger: true, Kind: OutlinedButton, Content: ARGB(0xFF336699), OnClick: func() { runs = append(runs, "remove") }})
		})
	})
	h.click("Run")
	h.click("Tonal")
	h.click("Remove")
	if err := h.tester.Click("Off"); err != nil {
		t.Logf("a disabled button cannot be clicked: %v", err)
	}
	if strings.Join(runs, ",") != "run,tonal,remove" {
		t.Errorf("got %v", runs)
	}
	node, ok := h.node(RoleButton, "Off")
	if !ok || node.Enabled {
		t.Errorf("a disabled button tells the registry: %+v", node)
	}
	// The registry runs the action, as a scenario would.
	node, _ = h.node(RoleButton, "Run")
	node.Click()
	if runs[len(runs)-1] != "run" {
		t.Errorf("got %v", runs)
	}
}

func TestIconButtonsAreNamedByTheirTooltip(t *testing.T) {
	clicked := 0
	h := newTested(t, 400, 200, func(c Ctx) {
		Row(c).Gap(8).Padding(16).Children(func() {
			IconButton(c, IconButtonSpec{ID: "a", Icon: "refresh", Tooltip: "Refresh", OnClick: func() { clicked++ }})
			IconButton(c, IconButtonSpec{ID: "b", Icon: "more", Tooltip: "More", Kind: FilledIconButton, Selected: true, Rotation: 90, Size: 56})
			IconButton(c, IconButtonSpec{ID: "c", Icon: "copy", Tooltip: "Copy", Kind: TonalIconButton, Disabled: true, Color: ARGB(0xFF336699)})
		})
	})
	h.click("Refresh")
	if clicked != 1 {
		t.Errorf("got %d", clicked)
	}
	for _, name := range []string{"Refresh", "More", "Copy"} {
		node, ok := h.node(RoleButton, name)
		if !ok || node.Tooltip != name {
			t.Errorf("%s: %+v", name, node)
		}
	}
	if node, _ := h.node(RoleButton, "More"); !node.Selected {
		t.Error("a toggle that is on says so")
	}
}

func TestSwitchesFlipAndButtonGroupsSelect(t *testing.T) {
	on := false
	selected := "a"
	h := newTested(t, 600, 200, func(c Ctx) {
		Row(c).Gap(16).Padding(16).Children(func() {
			Switch(c, SwitchSpec{ID: "s", Label: "Sync", On: on, OnChange: func(value bool) { on = value }})
			Switch(c, SwitchSpec{ID: "d", Label: "Locked", On: true, Disabled: true})
			ButtonGroup(c, ButtonGroupSpec{
				ID: "g", Selected: selected, OnSelect: func(value string) { selected = value },
				Segments: []Segment{{Value: "a", Label: "First", Icon: "add"}, {Value: "b", Label: "Second", Tooltip: "The second"}},
			})
		})
	})
	h.click("Sync")
	if !on {
		t.Error("a click flips the switch")
	}
	h.click("Sync")
	if on {
		t.Error("and again")
	}
	h.click("Second")
	if selected != "b" {
		t.Errorf("got %q", selected)
	}
	if node, _ := h.node(RoleButton, "Second"); !node.Selected {
		t.Error("the selected segment says so")
	}
	if node, _ := h.node(RoleSwitch, "Locked"); node.Enabled || !node.Selected {
		t.Errorf("a disabled switch that is on: %+v", node)
	}
	node, _ := h.node(RoleSwitch, "Sync")
	node.Click()
	if !on {
		t.Error("the registry flips the switch")
	}
}

func TestMenusOpenBelowTheirAnchorAndCloseOnTheChoiceOrEscape(t *testing.T) {
	var chosen []string
	h := newTested(t, 500, 400, func(c Ctx) {
		Column(c).Padding(16).Children(func() {
			IconButton(c, IconButtonSpec{ID: "more", Icon: "more", Tooltip: "More actions", Menu: []MenuItem{
				{Label: "Open", Icon: "openInNew", OnClick: func() { chosen = append(chosen, "open") }},
				{Label: "Remove", Danger: true, OnClick: func() { chosen = append(chosen, "remove") }},
				{Label: "Locked", Disabled: true},
				{Label: "Current", Selected: true},
			}})
		})
	})
	h.click("More actions")
	h.mustText("Open")
	for _, name := range []string{"Open", "Remove", "Locked", "Current"} {
		node, ok := h.node(RoleMenuItem, name)
		if !ok || !node.Modal {
			t.Errorf("%s is a modal menu item: %+v", name, node)
		}
	}
	if node, _ := h.node(RoleMenuItem, "Locked"); node.Enabled {
		t.Error("a disabled entry is inert")
	}
	h.click("Open")
	if len(chosen) != 1 || chosen[0] != "open" {
		t.Errorf("got %v", chosen)
	}
	if h.tester.HasText("Remove") {
		t.Error("the menu closes on the choice")
	}
	// Escape closes it, too.
	h.click("More actions")
	h.mustText("Remove")
	h.tester.Key(0, ui.KeyEscape)
	h.tester.Advance(time.Second)
	if h.tester.HasText("Remove") {
		t.Error("Escape closes the menu")
	}
	// The registry chooses an entry.
	h.click("More actions")
	node, _ := h.node(RoleMenuItem, "Remove")
	node.Click()
	h.tester.Frame()
	if chosen[len(chosen)-1] != "remove" {
		t.Errorf("got %v", chosen)
	}
}

func TestSplitButtonRunsItsActionAndOpensItsMenu(t *testing.T) {
	var runs []string
	h := newTested(t, 500, 300, func(c Ctx) {
		Column(c).Padding(16).Children(func() {
			SplitButton(c, SplitButtonSpec{
				ID: "use", Label: "Use", Icon: "check", MenuTooltip: "Options", OnClick: func() { runs = append(runs, "use") },
				Items: []MenuItem{{Label: "Globally", OnClick: func() { runs = append(runs, "global") }}},
			}).AlignSelf(ui.Start)
			SplitButton(c, SplitButtonSpec{ID: "off", Label: "Idle", MenuTooltip: "Idle options", Disabled: true}).AlignSelf(ui.Start)
		})
	})
	h.click("Use")
	h.click("Options")
	if node, _ := h.node(RoleButton, "Options"); !node.Selected {
		t.Error("the menu button says that its menu is open")
	}
	h.click("Globally")
	if strings.Join(runs, ",") != "use,global" {
		t.Errorf("got %v", runs)
	}
	if node, _ := h.node(RoleButton, "Idle"); node.Enabled {
		t.Error("a disabled split button is inert")
	}
}

func TestDropdownsShowTheChoiceAndSelectAnother(t *testing.T) {
	value := "b"
	h := newTested(t, 500, 400, func(c Ctx) {
		Column(c).Padding(16).Children(func() {
			Dropdown(c, DropdownSpec{
				ID: "d", Value: value, Choices: []Choice{{"a", "System"}, {"b", "English"}, {"c", "Chinese"}},
				OnSelect: func(chosen string) { value = chosen },
			}).AlignSelf(ui.Start)
			Dropdown(c, DropdownSpec{ID: "off", Value: "a", Choices: []Choice{{"a", "Locked"}}, Disabled: true}).AlignSelf(ui.Start)
		})
	})
	h.mustText("English")
	h.click("English")
	h.mustText("Chinese")
	h.click("Chinese")
	if value != "c" {
		t.Errorf("got %q", value)
	}
	h.mustText("Chinese")
	// Choosing the entry that is chosen already changes nothing.
	h.click("Chinese")
	h.click("Chinese")
	if value != "c" {
		t.Errorf("got %q", value)
	}
}

func TestDialogsAreModalAndGoWhenClosed(t *testing.T) {
	closed := false
	h := newTested(t, 700, 500, func(c Ctx) {
		Column(c).Fill().Padding(16).Children(func() {
			Button(c, ButtonSpec{ID: "open", Label: "Open", OnClick: func() {
				c.Env.ShowDialog("d", func(c Ctx, closeDialog func()) DialogSpec {
					return DialogSpec{
						Icon: "info", Title: "Hello", MaxWidth: 400,
						Content: func() { Label(c, "A dialog.", Type(BodyMedium), c.Colors().OnSurfaceVariant) },
						Actions: []func(){func() {
							Button(c, ButtonSpec{ID: "close", Label: "Close", OnClick: func() { closed = true; closeDialog() }})
						}},
					}
				})
			}})
		})
		c.Env.LayoutDialogs(c)
	})
	h.click("Open")
	h.mustText("Hello")
	if !h.env.HasDialog() {
		t.Error("the dialog is open")
	}
	// Opening it again leaves the one that is open.
	h.env.ShowDialog("d", nil)
	if len(h.env.dialogs) != 1 {
		t.Errorf("one dialog: %d", len(h.env.dialogs))
	}
	if node, ok := h.node(RoleDialog, "Hello"); !ok || !node.Modal {
		t.Errorf("the registry knows the dialog: %+v", node)
	}
	if node, _ := h.node(RoleButton, "Close"); node == nil || node.Scope != "dialog" || !node.Modal {
		t.Errorf("its buttons are in its scope: %+v", node)
	}
	h.click("Close")
	h.tester.Advance(time.Second)
	if !closed || h.tester.HasText("Hello") || h.env.HasDialog() {
		t.Errorf("the dialog goes: closed %v, texts %q", closed, h.tester.Texts())
	}
	// Escape closes a dialog, too.
	h.click("Open")
	h.mustText("Hello")
	h.tester.Key(0, ui.KeyEscape)
	h.tester.Advance(time.Second)
	if h.tester.HasText("Hello") {
		t.Error("Escape closes the dialog")
	}
}

func TestSnackbarsShowForAWhileAndReplaceEachOther(t *testing.T) {
	acted := 0
	h := newTested(t, 700, 400, func(c Ctx) {
		Column(c).Fill().Children(func() {})
		c.Env.LayoutSnackbars(c)
	})
	h.env.ShowSnackbar(Snack{Text: "Saved"})
	h.tester.Frame()
	h.mustText("Saved")
	// A new one replaces it.
	h.env.ShowSnackbar(Snack{Text: "Failed", Action: "View", OnAction: func() { acted++ }, Persist: true, Closable: true, CloseLabel: "Dismiss"})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	h.mustText("Failed")
	if h.tester.HasText("Saved") {
		t.Error("one snack bar at a time")
	}
	if !h.env.PendingSnack() == false {
		t.Error("nothing waits")
	}
	// A persistent snack bar stays.
	h.tester.Advance(30 * time.Second)
	h.mustText("Failed")
	h.click("View")
	if acted != 1 {
		t.Errorf("got %d", acted)
	}
	h.tester.Advance(time.Second)
	if h.tester.HasText("Failed") {
		t.Error("the action closes the snack bar")
	}
	// A plain one goes by itself, unless the pointer rests on it (the pointer is
	// still where the action was), and the close button closes the other kind.
	h.tester.Move(5, 5)
	h.env.ShowSnackbar(Snack{Text: "Copied"})
	h.tester.Frame()
	h.mustText("Copied")
	h.tester.Advance(10 * time.Second)
	if h.tester.HasText("Copied") {
		t.Error("a snack bar goes after a few seconds")
	}
	h.env.ShowSnackbar(Snack{Text: "Stay", Persist: true, Closable: true, CloseLabel: "Dismiss"})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	node, _ := h.node(RoleButton, "Dismiss")
	if node == nil {
		t.Fatalf("no close button: %q", h.tester.Texts())
	}
	node.Click()
	h.tester.Frame()
	h.tester.Advance(time.Second)
	if h.tester.HasText("Stay") {
		t.Error("the registry closes the snack bar")
	}
}

func TestTextFieldsKeepWhatTheUserTypesUntilTheValueChanges(t *testing.T) {
	var changed, submitted, blurred []string
	value := "first"
	owned := "owned"
	h := newTested(t, 600, 700, func(c Ctx) {
		Column(c).Padding(16).Gap(16).Children(func() {
			TextField(c, TextFieldSpec{
				ID: "name", Label: "Name", Value: value, Helper: "Helper text",
				OnChange: func(text string) { changed = append(changed, text) },
				OnSubmit: func(text string) { submitted = append(submitted, text) },
				OnBlur:   func(text string) { blurred = append(blurred, text) },
			})
			TextField(c, TextFieldSpec{ID: "owned", Label: "Owned", Text: &owned, Error: "Not valid", LeadingIcon: "search"})
			TextField(c, TextFieldSpec{ID: "hint", Hint: "A hint", Dense: true, Trailing: func() { Icon(c, "close", 24, c.Colors().OnSurface, 0) }})
			TextField(c, TextFieldSpec{ID: "lines", Label: "Notes", Lines: 2, MaxLines: 4, Monospace: true, AutoFocus: false})
			TextField(c, TextFieldSpec{ID: "off", Label: "Off", Value: "x", Disabled: true})
		})
	})
	h.mustText("Helper text")
	h.mustText("Not valid")
	node, ok := h.node(RoleTextField, "Name")
	if !ok || node.Value() != "first" || node.Text != "first" {
		t.Fatalf("the field shows its value: %+v", node)
	}
	h.click("Name")
	h.tester.Type("abc")
	h.tester.Advance(time.Second)
	node, _ = h.node(RoleTextField, "Name")
	if node.Value() != "firstabc" {
		t.Errorf("got %q", node.Value())
	}
	if len(changed) == 0 || changed[len(changed)-1] != "firstabc" {
		t.Errorf("got %v", changed)
	}
	h.tester.Key(0, ui.KeyEnter)
	h.tester.Advance(time.Second)
	if len(submitted) != 1 || submitted[0] != "firstabc" {
		t.Errorf("got %v", submitted)
	}
	// The text stays while the value stays, and follows it when it changes.
	h.tester.Frame()
	if node, _ = h.node(RoleTextField, "Name"); node.Value() != "firstabc" {
		t.Errorf("got %q", node.Value())
	}
	value = "second"
	h.tester.Frame()
	if node, _ = h.node(RoleTextField, "Name"); node.Value() != "second" {
		t.Errorf("got %q", node.Value())
	}
	// Leaving the field says so.
	h.click("Owned")
	h.tester.Advance(time.Second)
	if len(blurred) == 0 || blurred[len(blurred)-1] != "second" {
		t.Errorf("got %v", blurred)
	}
	// A field keeps its text in the string it is given.
	h.tester.Type("!")
	if owned != "owned!" {
		t.Errorf("got %q", owned)
	}
	// The registry fills a field in, as a scenario does.
	node, _ = h.node(RoleTextField, "Name")
	changed, submitted = nil, nil
	node.Fill("filled", true)
	if len(changed) != 1 || changed[0] != "filled" || len(submitted) != 1 {
		t.Errorf("got %v and %v", changed, submitted)
	}
	if node, _ = h.node(RoleTextField, "Off"); node.Enabled {
		t.Error("a disabled field is inert")
	}
}

func TestSearchBarsClearTheirText(t *testing.T) {
	query := "node"
	var changes []string
	h := newTested(t, 500, 200, func(c Ctx) {
		Column(c).Padding(16).Children(func() {
			SearchBar(c, SearchBarSpec{
				ID: "search", Hint: "Search", Text: &query, ClearTooltip: "Clear",
				OnChange: func(text string) { changes = append(changes, text) },
			})
			SearchBar(c, SearchBarSpec{ID: "other", Hint: "Other", Value: "x"})
		})
	})
	h.click("Clear")
	if query != "" || len(changes) != 1 || changes[0] != "" {
		t.Errorf("got %q and %v", query, changes)
	}
	if _, ok := h.node(RoleButton, "Clear"); ok {
		t.Error("the clear button goes with the text")
	}
	node, _ := h.node(RoleTextField, "Search")
	node.Fill("go", false)
	if query != "go" {
		t.Errorf("got %q", query)
	}
	if node, _ = h.node(RoleTextField, "Other"); node.Value() != "x" {
		t.Errorf("got %q", node.Value())
	}
}

func TestListItemsAndGroupsAreSegmented(t *testing.T) {
	cases := []struct {
		index, count int
		want         Position
	}{{0, 1, PosSingle}, {0, 3, PosFirst}, {1, 3, PosMiddle}, {2, 3, PosLast}}
	for _, c := range cases {
		if got := PositionOf(c.index, c.count); got != c.want {
			t.Errorf("%d of %d: got %v", c.index, c.count, got)
		}
	}
	clicked := ""
	wrapped := ListItemSpec{ID: "w", Title: "Wrapped", Wrap: true, Trailing: func() {}, Padding: []float32{8, 16}}
	h := newTested(t, 500, 600, func(c Ctx) {
		Column(c).Padding(16).Gap(8).Children(func() {
			ListGroup(c,
				ListItemSpec{ID: "1", Title: "One", Subtitle: "first\nsecond", SubtitleLines: 2, OnClick: func() { clicked = "one" }},
				ListItemSpec{ID: "2", Title: "Two", Selected: true, Leading: func(foreground ui.Color) { Icon(c, "add", 24, foreground, 0) }},
				ListItemSpec{ID: "3", TitleWidget: func(ui.Color) { Label(c, "Custom", Type(BodyLarge), c.Colors().OnSurface) }, Name: "Three",
					SubtitleWidget: func(ui.Color) { Label(c, "Widget", Type(BodyMedium), c.Colors().OnSurface) }},
			)
			ListItem(c, wrapped)
			Card(c, CardSpec{ID: "card", Color: c.Colors().SurfaceContainer, Radius: CornerLarge, Name: "Card", OnClick: func() { clicked = "card" },
				Children: func() { Label(c, "Inside the card", Type(BodyMedium), c.Colors().OnSurface) }})
		})
	})
	h.click("One")
	if clicked != "one" {
		t.Errorf("got %q", clicked)
	}
	h.click("Card")
	if clicked != "card" {
		t.Errorf("got %q", clicked)
	}
	for _, name := range []string{"One", "Two", "Three", "Wrapped"} {
		if _, ok := h.node(RoleListItem, name); !ok {
			t.Errorf("no list item %q", name)
		}
	}
	if node, _ := h.node(RoleListItem, "Two"); !node.Selected {
		t.Error("the selected item says so")
	}
	h.mustText("Inside the card")
}

func TestStatusMessagesPanesAndHeadersShowTheirText(t *testing.T) {
	h := newTested(t, 700, 700, func(c Ctx) {
		Column(c).Fill().Children(func() {
			PageHeader(c, "Title", "Subtitle", func() { Icon(c, "back", 24, c.Colors().OnSurface, 0) },
				func() { Button(c, ButtonSpec{ID: "action", Label: "Action"}) })
			GroupHeader(c, "Group", func() { Label(c, "Trailing", Type(LabelSmall), c.Colors().Primary) })
			ContentWidth(c, 300, func() { Label(c, "Narrow content", Type(BodyMedium), c.Colors().OnSurface) })
			Pane(c, func() {
				StatusMessage(c, StatusSpec{
					Icon: "empty", IsError: true, Title: "Nothing", Body: "Nothing here.",
					Actions: []func(){func() { Button(c, ButtonSpec{ID: "retry", Label: "Retry"}) }},
				})
			}).Grow(1)
		})
	})
	for _, text := range []string{"Title", "Subtitle", "Group", "Trailing", "Narrow content", "Nothing", "Nothing here.", "Retry", "Action"} {
		h.mustText(text)
	}
	h2 := newTested(t, 300, 300, func(c Ctx) {
		Column(c).Fill().Children(func() {
			LoadingMessage(c, "Loading versions", "Loading").Grow(1)
		})
	})
	h2.mustText("Loading versions")
	h3 := newTested(t, 300, 200, func(c Ctx) {
		Panel(c, c.Colors().SurfaceContainerHighest, CornerExtraSmall, CornerLargeInc, func() {
			Label(c, "Output", Type(BodySmall), c.Colors().OnSurface)
		})
	})
	h3.mustText("Output")
}

func TestNavigationRailAndBarSelectDestinations(t *testing.T) {
	selected := 0
	expanded := false
	fabs := 0
	destinations := []NavDestination{{Icon: "home", Label: "Home"}, {Icon: "sdks", Label: "SDKs"}, {Icon: "tasks", Label: "Tasks", Badge: 1200}}
	rail := newTested(t, 700, 500, func(c Ctx) {
		Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
			NavRail(c, NavRailSpec{
				ID: "rail", Destinations: destinations, Selected: selected, OnSelect: func(i int) { selected = i },
				Expanded: expanded, OnExpandedChange: func(value bool) { expanded = value },
				ExpandTooltip: "Show labels", CollapseTooltip: "Hide labels", FABIcon: "add", FABLabel: "Add", OnFAB: func() { fabs++ },
			})
		})
	})
	rail.click("SDKs")
	if selected != 1 {
		t.Errorf("got %d", selected)
	}
	rail.click("Show labels")
	if !expanded {
		t.Error("the menu button expands the rail")
	}
	rail.click("Hide labels")
	if expanded {
		t.Error("and collapses it")
	}
	rail.click("Add")
	if fabs != 1 {
		t.Errorf("got %d", fabs)
	}
	for _, name := range []string{"Home", "SDKs", "Tasks"} {
		if _, ok := rail.node(RoleButton, name); !ok {
			t.Errorf("no destination %q", name)
		}
	}
	if node, _ := rail.node(RoleButton, "SDKs"); !node.Selected {
		t.Error("the selected destination says so")
	}
	rail.mustText("999+")
	bare := newTested(t, 700, 500, func(c Ctx) {
		Row(c).Fill().Children(func() {
			NavRail(c, NavRailSpec{ID: "rail", Destinations: destinations, OnSelect: func(int) {}, OnExpandedChange: func(bool) {}})
		})
	})
	if _, ok := bare.node(RoleButton, "Add"); ok {
		t.Error("a rail without an action has no action button")
	}
	bar := newTested(t, 400, 300, func(c Ctx) {
		Column(c).Fill().Children(func() {
			mid := ui.Box(c.Context).Grow(1)
			_ = mid
			NavBar(c, NavBarSpec{ID: "bar", Destinations: destinations, Selected: selected, OnSelect: func(i int) { selected = i }})
		})
	})
	bar.click("Home")
	if selected != 0 {
		t.Errorf("got %d", selected)
	}
	bar.mustText("999+")
}

func TestProgressIndicatorsShapesAndAvatarsAreDrawn(t *testing.T) {
	env := testEnv(t, false)
	draw := func(build func(c Ctx)) *image.RGBA {
		return ui.Render(view(env, func(c Ctx) {
			Column(c).Fill().Background(c.Colors().Surface).Padding(8).Children(func() { build(c) })
		}), 200, 100, 1)
	}
	blank := draw(func(Ctx) {})
	differs := func(name string, build func(c Ctx)) {
		if img := draw(build); string(img.Pix) == string(blank.Pix) {
			t.Errorf("%s draws nothing", name)
		}
	}
	differs("a determinate wave", func(c Ctx) { WavyProgress(c, WavySpec{ID: "a", Value: 0.5, Label: "Half"}) })
	differs("an empty wave", func(c Ctx) { WavyProgress(c, WavySpec{ID: "b", Value: 0}) })
	differs("a full wave", func(c Ctx) { WavyProgress(c, WavySpec{ID: "c", Value: 1}) })
	differs("an indeterminate wave", func(c Ctx) { WavyProgress(c, WavySpec{ID: "d", Indeterminate: true}) })
	differs("a loading indicator", func(c Ctx) { LoadingIndicator(c, LoadingSpec{ID: "l", Size: 64, Contained: true, Label: "Loading"}) })
	differs("a plain loading indicator", func(c Ctx) { LoadingIndicator(c, LoadingSpec{Color: ARGB(0xFF3F6FD8)}) })
	differs("a swatch", func(c Ctx) {
		ColorSwatch(c, SwatchSpec{ID: "sw", Seed: 0xFF3F6FD8, Label: "Blue", Selected: true, OnClick: func() {}})
	})
	differs("an avatar", func(c Ctx) { Avatar(c, AvatarSpec{Name: "nodejs", Size: 48}) })
	differs("an avatar with an icon", func(c Ctx) { Avatar(c, AvatarSpec{Icon: "sdks"}) })
	differs("a shape", func(c Ctx) { Shape(c, DecorativeShape("maven"), 64, c.Colors().Primary, nil) })
	differs("an icon", func(c Ctx) { Icon(c, "home", 24, c.Colors().Primary, 0.5) })
	differs("an elevated button", func(c Ctx) { Elevation(c, Box(c).Size(40, 40), 3) })
	differs("a tag", func(c Ctx) { Tag(c, TagSpec{Label: "LTS", Icon: "current", Emphasized: true}) })
	differs("a floating action button", func(c Ctx) { FAB(c, FABSpec{Icon: "add", Label: "Add", Expansion: 0.6}) })
	differs("a collapsed section", func(c Ctx) {
		Collapsible(c, "x", true, func() { Label(c, "Open", Type(BodyMedium), c.Colors().OnSurface) })
	})
	if img := draw(func(c Ctx) {
		Collapsible(c, "y", false, func() { Label(c, "Shut", Type(BodyMedium), c.Colors().OnSurface) })
	}); string(img.Pix) != string(blank.Pix) {
		t.Error("a closed section shows nothing")
	}
	if HasIcon("no-such-icon") || !HasIcon("home") {
		t.Error("the icon fonts know the icons of the interface")
	}
}

func TestAvatarsAreTheSameInEveryImplementation(t *testing.T) {
	// FNV-1a over the UTF-16 code units; the other implementations compute the
	// same numbers, which is what makes an avatar the same everywhere.
	for text, want := range map[string]uint32{
		"": 0x811C9DC5, "a": 0xE40C292C, "nodejs": 0x1F748B62, "maven": 0xA6E29488, "你好": 0xD7BC7166,
		// Characters outside the Basic Multilingual Plane are two code units.
		"😀": 0xCB31C4B8,
	} {
		if got := StableHash(text); got != want {
			t.Errorf("%q: got %#x, want %#x", text, got, want)
		}
	}
	if StableHash("maven") == StableHash("mvn2") {
		t.Error("different names have different hashes")
	}
	shape := DecorativeShape("maven")
	if again := DecorativeShape("maven"); again != shape {
		t.Error("a name always gets the same shape")
	}
	if initial("nodejs") != "N" || initial("") != "?" || initial("élan") != "É" {
		t.Error("the initial is the first letter in upper case")
	}
}

func TestSwitchViewKeepsTheViewOfTheKeyAndGivesWayToTheNext(t *testing.T) {
	key := "a"
	h := newTested(t, 400, 300, func(c Ctx) {
		Column(c).Fill().Children(func() {
			SwitchView(c, FadeThrough, 0, key, func() { Label(c, "View "+key, Type(BodyLarge), c.Colors().OnSurface) }).Grow(1)
		})
	})
	h.mustText("View a")
	key = "b"
	h.tester.Frame()
	h.tester.Advance(2 * time.Second)
	h.mustText("View b")
	if h.tester.HasText("View a") {
		t.Error("the old view has gone")
	}
	shared := newTested(t, 400, 300, func(c Ctx) {
		Column(c).Fill().Children(func() {
			SwitchView(c, SharedAxisX, 1, key, func() { Label(c, "Detail "+key, Type(BodyLarge), c.Colors().OnSurface) }).Grow(1)
		})
	})
	shared.mustText("Detail b")
}

func TestSpringsReachTheirTargetAndOnlyBounceWhenUnderdamped(t *testing.T) {
	bouncy := Spring{Damping: 0.5, Stiffness: 400}
	critical := Spring{Damping: 1, Stiffness: 400}
	heavy := Spring{Damping: 2, Stiffness: 400}
	for name, spring := range map[string]Spring{"bouncy": bouncy, "critical": critical, "heavy": heavy} {
		if got := spring.Position(0); math.Abs(got) > 1e-9 {
			t.Errorf("%s starts at 0, got %v", name, got)
		}
		if got := spring.Position(10); math.Abs(got-1) > 1e-6 {
			t.Errorf("%s settles at 1, got %v", name, got)
		}
	}
	overshoot := func(s Spring) float64 {
		peak := 0.0
		for i := range 400 {
			peak = math.Max(peak, s.Position(float64(i)/200))
		}
		return peak
	}
	if overshoot(bouncy) <= 1.1 {
		t.Error("an underdamped spring overshoots")
	}
	if overshoot(critical) > 1+1e-9 || overshoot(heavy) > 1+1e-9 {
		t.Error("the others do not")
	}
}

func TestSettleTimeGrowsWithSoftnessAndIsRemembered(t *testing.T) {
	stiff := Spring{Damping: 1, Stiffness: 3800}.SettleTime()
	soft := Spring{Damping: 1, Stiffness: 200}.SettleTime()
	if stiff <= 0 || soft <= stiff {
		t.Errorf("got %v and %v", stiff, soft)
	}
	if again := (Spring{Damping: 1, Stiffness: 200}).SettleTime(); again != soft {
		t.Errorf("got %v and %v", again, soft)
	}
	if never := (Spring{Damping: 0.001, Stiffness: 1}).SettleTime(); never != 5*time.Second {
		t.Errorf("a spring that does not settle is cut off, got %v", never)
	}
	if Expressive.FastSpatial.Damping >= 1 || Expressive.DefaultEffects.Damping != 1 {
		t.Error("spatial springs bounce, effects springs do not")
	}
}

func TestSpringsAreGivenToMygoAsADampingCoefficient(t *testing.T) {
	// The design system says the damping ratio; mygo takes the coefficient, which
	// is twice the ratio times the square root of the stiffness for a unit mass.
	spring := Spring{Damping: 0.6, Stiffness: 800}.Anim()
	options := spring.Spring
	if options.Mass != 1 || options.Stiffness != 800 || math.Abs(options.Damping-2*0.6*math.Sqrt(800)) > 1e-9 {
		t.Errorf("got %+v", options)
	}
	if Flag(true) != 1 || Flag(false) != 0 || Lerp(2, 4, 0.25) != 2.5 || clamp01(2) != 1 || clamp01(-1) != 0 {
		t.Error("helpers")
	}
}

func TestTheThemeFollowsTheSchemeAndTheTextScale(t *testing.T) {
	light := NewTheme(testSeed, false, false, 0)
	dark := NewTheme(testSeed, true, true, 1.5)
	if light.TextScale != 1 || dark.TextScale != 1.5 || !dark.ReducedMotion || light.Dark || !dark.Dark {
		t.Errorf("got %+v and %+v", light, dark)
	}
	if light.Colors.Primary == dark.Colors.Primary || light.Colors.Surface == dark.Colors.Surface {
		t.Error("the schemes differ")
	}
	mygo := dark.UI()
	if !mygo.Dark || mygo.Background != dark.Colors.Surface || mygo.Accent != dark.Colors.Primary || mygo.FontSize != 14*1.5 {
		t.Errorf("mygo's own widgets follow the scheme: %+v", mygo)
	}
	if style := Type(BodyLarge).WithSize(32).WithWeight(700); style.Size != 32 || style.Line != 48 || style.Weight != 700 {
		t.Errorf("got %+v", style)
	}
	if Emph(TitleMedium).Weight != 700 || Type(TitleMedium).Weight != 500 {
		t.Error("emphasized text is heavier")
	}
	if got := ARGB(0x80102030); got != (ui.Color{R: 0x10, G: 0x20, B: 0x30, A: 0x80}) {
		t.Errorf("got %+v", got)
	}
}

func TestTheRegistryScopesAndCollectsText(t *testing.T) {
	var registry Registry
	registry.begin()
	registry.Add(Node{Role: RoleButton, Name: "outside"})
	leave := registry.EnterScope("row")
	registry.Add(Node{Role: RoleButton, Name: "inside"})
	registry.SetModal(true)
	registry.Add(Node{Role: RoleMenuItem, Name: "menu", Scope: "own"})
	registry.SetModal(false)
	leave()
	registry.Add(Node{Role: RoleButton, Name: "after"})
	nodes := registry.Nodes()
	if len(nodes) != 4 || nodes[0].Scope != "" || nodes[1].Scope != "row" || nodes[2].Scope != "own" || nodes[3].Scope != "" {
		t.Errorf("got %+v", nodes)
	}
	if nodes[0].Modal || !nodes[2].Modal || nodes[3].Modal {
		t.Error("only the nodes added while modal are modal")
	}
	outer := &Node{Rect: ui.Rect{X: 0, Y: 0, W: 100, H: 100}}
	inner := &Node{Rect: ui.Rect{X: 10, Y: 10, W: 20, H: 20}}
	if !inner.Inside(outer) || outer.Inside(inner) || (&Node{}).Visible() || !inner.Visible() {
		t.Error("nodes lie inside each other once they are drawn")
	}
	registry.begin()
	if len(registry.Nodes()) != 0 || registry.Builds() != 2 {
		t.Errorf("a build starts again: %d nodes, %d builds", len(registry.Nodes()), registry.Builds())
	}
}

func TestLabelsCollectTheirTextAndFollowTheTextScale(t *testing.T) {
	env := testEnv(t, false)
	env.Theme = NewTheme(testSeed, false, false, 1.25)
	tester := ui.NewTester(view(env, func(c Ctx) {
		Column(c).Padding(8).Children(func() {
			Label(c, "Plain", Type(BodyMedium), c.Colors().OnSurface)
			Label(c, "Centred", Type(BodyMedium), c.Colors().OnSurface, Centered(), Lines(2))
			Label(c, "Mono", Type(BodySmall), c.Colors().OnSurface, Monospace(), Selectable())
			Line(c, "Single", Type(BodyMedium), c.Colors().OnSurface)
		})
	}), 300, 200)
	for _, text := range []string{"Plain", "Centred", "Mono", "Single"} {
		if !tester.HasText(text) {
			t.Errorf("no text %q", text)
		}
	}
	collected := 0
	for _, n := range env.Reg.Nodes() {
		if n.Role == RoleText {
			collected++
			if !n.Visible() {
				t.Errorf("the text %q has no place", n.Text)
			}
		}
	}
	if collected != 4 {
		t.Errorf("got %d", collected)
	}
}
