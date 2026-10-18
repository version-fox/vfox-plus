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

// atCentre is the middle of a node.
func atCentre(n kit.Node) image.Point { return n.Rect.Min.Add(n.Rect.Size().Div(2)) }

// settle lays out w a few frames, so that the input is handled and springs
// move.
func (h *harness) settle(w kit.Widget) { h.frames(4, w) }

// dynamic is a widget that is made again for every frame, like the pages of
// the application, whose state changes through the callbacks of the widgets.
func dynamic(build func() kit.Widget) kit.Widget {
	return func(c kit.Ctx) layout.Dimensions { return build()(c) }
}

func TestButtonsRunTheirActionOnAClickAndNotWhenDisabled(t *testing.T) {
	h := newHarness(t)
	clicks, disabledClicks := 0, 0
	w := kit.Row(kit.FlexOpts{Gap: 20, Min: true},
		kit.Rigid(kit.Button{ID: "go", Label: "Go", Icon: "add", OnClick: func() { clicks++ }}.W()),
		kit.Rigid(kit.Button{ID: "no", Label: "No", Disabled: true, OnClick: func() { disabledClicks++ }}.W()),
		kit.Rigid(kit.Button{ID: "text", Kind: kit.TextButton, Label: "Text"}.W()),
		kit.Rigid(kit.Button{ID: "out", Kind: kit.OutlinedButton, Label: "Out"}.W()),
		kit.Rigid(kit.Button{ID: "tonal", Kind: kit.TonalButton, Label: "Tonal"}.W()),
		kit.Rigid(kit.Button{ID: "danger", Label: "Remove", Danger: true}.W()),
	)
	h.settle(w)
	goButton := h.mustNode(kit.RoleButton, "Go")
	if !goButton.Enabled || goButton.Rect.Dy() < 40 {
		t.Errorf("got %#v", goButton)
	}
	if h.mustNode(kit.RoleButton, "No").Enabled {
		t.Error("a disabled button is not enabled")
	}
	h.click(atCentre(goButton))
	h.settle(w)
	h.click(atCentre(h.mustNode(kit.RoleButton, "No")))
	h.settle(w)
	if clicks != 1 || disabledClicks != 0 {
		t.Errorf("got %d and %d", clicks, disabledClicks)
	}
	// The node does what a click does, which the end-to-end tests rely on.
	h.mustNode(kit.RoleButton, "Go").Click()
	h.settle(w)
	if clicks != 2 {
		t.Errorf("got %d", clicks)
	}
}

func TestIconButtonsAreNamedByTheirTooltip(t *testing.T) {
	h := newHarness(t)
	clicked := false
	w := kit.Row(kit.FlexOpts{Min: true},
		kit.Rigid(kit.IconButton{ID: "refresh", Icon: "refresh", Tooltip: "Refresh", OnClick: func() { clicked = true }}.W()),
		kit.Rigid(kit.IconButton{ID: "filled", Kind: kit.FilledIconButton, Icon: "add", Tooltip: "Add", Selected: true, Rotation: 1}.W()),
		kit.Rigid(kit.IconButton{ID: "tonal", Kind: kit.TonalIconButton, Icon: "copy", Tooltip: "Copy", Disabled: true}.W()),
		kit.Rigid(kit.FAB{ID: "fab", Icon: "add", Label: "Add plugin", Expansion: 1, OnClick: func() {}}.W()),
		kit.Rigid(kit.FAB{ID: "fab2", Icon: "add", Label: "Add plugin", Tooltip: "Add", OnClick: func() {}}.W()),
		kit.Rigid(kit.Tag{Label: "LTS", Emphasized: true, Icon: "current"}.W()),
		kit.Rigid(kit.Tag{Label: "Plugin 1"}.W()),
	)
	h.settle(w)
	button := h.mustNode(kit.RoleButton, "Refresh")
	if button.Rect.Dx() < 48 || button.Tooltip != "Refresh" {
		t.Errorf("got %#v", button)
	}
	h.click(atCentre(button))
	h.settle(w)
	if !clicked {
		t.Error("expected a click")
	}
	if !h.mustNode(kit.RoleButton, "Add").Selected || h.mustNode(kit.RoleButton, "Copy").Enabled {
		t.Error("selected and disabled icon buttons")
	}
	if !h.hasText("LTS") || !h.hasText("Plugin 1") || !h.hasText("Add plugin") {
		t.Error("tags and extended buttons show their text")
	}
}

func TestSwitchesFlipAndButtonGroupsSelect(t *testing.T) {
	h := newHarness(t)
	on, selected := false, "b"
	w := dynamic(func() kit.Widget {
		return kit.Column(kit.FlexOpts{Min: true, Gap: 8},
			kit.Rigid(kit.Switch{ID: "sw", Label: "Dark", On: on, OnChange: func(v bool) { on = v }}.W()),
			kit.Rigid(kit.ButtonGroup{ID: "bg", Selected: selected, OnSelect: func(v string) { selected = v }, Segments: []kit.Segment{
				{Value: "a", Label: "Alpha", Icon: "versions"}, {Value: "b", Label: "Beta"}, {Value: "c", Label: "Gamma", Tooltip: "The third"},
			}}.W()),
			kit.Rigid(kit.Switch{ID: "off", Label: "Off", Disabled: true, On: true}.W()),
		)
	})
	h.settle(w)
	h.mustNode(kit.RoleSwitch, "Dark").Click()
	h.settle(w)
	if !on {
		t.Error("expected the switch to turn on")
	}
	h.click(atCentre(h.mustNode(kit.RoleButton, "Gamma")))
	h.settle(w)
	if selected != "c" {
		t.Errorf("got %q", selected)
	}
	h.settle(w)
	if !h.mustNode(kit.RoleButton, "Gamma").Selected || h.mustNode(kit.RoleButton, "Beta").Selected {
		t.Error("the selected segment is the one that was chosen")
	}
	if h.mustNode(kit.RoleSwitch, "Off").Enabled {
		t.Error("a disabled switch ignores input")
	}
	// A group without a handler cannot be changed.
	inert := kit.ButtonGroup{ID: "inert", Selected: "a", Segments: []kit.Segment{{Value: "a", Label: "Only"}}}.W()
	h.settle(inert)
	if h.mustNode(kit.RoleButton, "Only").Enabled {
		t.Error("expected a disabled segment")
	}
	h.settle(kit.ButtonGroup{ID: "wide", Selected: "a", Expand: true, Segments: []kit.Segment{{Value: "a", Label: "A"}, {Value: "b", Label: "B"}}, OnSelect: func(string) {}}.W())
	if got := h.mustNode(kit.RoleButton, "A").Rect.Dx(); got < 300 {
		t.Errorf("an expanded group fills its space, got %d", got)
	}
}

func TestMenusOpenBelowTheirAnchorAndCloseOnTheChoiceOrAClickOutside(t *testing.T) {
	h := newHarness(t)
	chosen := ""
	w := kit.Align(layout.NW, kit.Pad(kit.All(20), kit.MenuButton{ID: "more", Icon: "more", Tooltip: "More", Items: []kit.MenuItem{
		{Label: "Open", Icon: "openInNew", OnClick: func() { chosen = "open" }},
		{Label: "Remove", Danger: true, OnClick: func() { chosen = "remove" }},
		{Label: "Disabled", Disabled: true},
	}}.W()))
	h.settle(w)
	if _, ok := h.node(kit.RoleMenuItem, "Open"); ok {
		t.Fatal("the menu is closed")
	}
	h.mustNode(kit.RoleButton, "More").Click()
	h.settle(w)
	anchor := h.mustNode(kit.RoleButton, "More")
	open := h.mustNode(kit.RoleMenuItem, "Open")
	if !open.Modal || open.Rect.Min.Y < anchor.Rect.Max.Y {
		t.Errorf("the menu is below its anchor: %v and %v", open.Rect, anchor.Rect)
	}
	if h.mustNode(kit.RoleMenuItem, "Disabled").Enabled {
		t.Error("a disabled entry")
	}
	open.Click()
	h.settle(w)
	if chosen != "open" {
		t.Errorf("got %q", chosen)
	}
	if _, ok := h.node(kit.RoleMenuItem, "Open"); ok {
		t.Error("the menu closes when an entry is chosen")
	}
	// A click outside closes it, too.
	h.mustNode(kit.RoleButton, "More").Click()
	h.settle(w)
	h.click(image.Pt(700, 500))
	h.settle(w)
	if _, ok := h.node(kit.RoleMenuItem, "Open"); ok {
		t.Error("the menu closes on a click outside")
	}
	if got := kit.Place(image.Pt(100, 100), image.Rect(80, 90, 100, 100), image.Pt(60, 50), true, 8); got.X != 32 || got.Y != 40 {
		t.Errorf("a popup stays inside the window and goes above when it does not fit below: %v", got)
	}
	if got := kit.Place(image.Pt(100, 100), image.Rect(0, 0, 10, 10), image.Pt(200, 50), false, 8); got.X != 8 || got.Y != 10 {
		t.Errorf("got %v", got)
	}
}

func TestSplitButtonRunsItsActionAndOpensItsMenu(t *testing.T) {
	h := newHarness(t)
	used, globally := 0, 0
	w := kit.Align(layout.NW, kit.SplitButton{
		ID: "use", Label: "Use", Icon: "check", MenuTooltip: "More ways", OnClick: func() { used++ },
		Items: []kit.MenuItem{{Label: "Globally", OnClick: func() { globally++ }}},
	}.W())
	h.settle(w)
	h.mustNode(kit.RoleButton, "Use").Click()
	h.settle(w)
	h.mustNode(kit.RoleButton, "More ways").Click()
	h.settle(w)
	h.mustNode(kit.RoleMenuItem, "Globally").Click()
	h.settle(w)
	if used != 1 || globally != 1 {
		t.Errorf("got %d and %d", used, globally)
	}
}

func TestDropdownsShowTheChoiceAndSelectAnother(t *testing.T) {
	h := newHarness(t)
	value, selections := "b", 0
	w := dynamic(func() kit.Widget {
		return kit.Align(layout.NW, kit.Dropdown{ID: "dd", Value: value, OnSelect: func(v string) { value = v; selections++ }, Choices: []kit.Choice{
			{Value: "a", Label: "Alpha"}, {Value: "b", Label: "Beta"}, {Value: "c", Label: "Gamma"},
		}}.W())
	})
	h.settle(w)
	if !h.hasText("Beta") {
		t.Error("the dropdown shows the chosen entry")
	}
	h.mustNode(kit.RoleButton, "Beta").Click()
	h.settle(w)
	h.mustNode(kit.RoleMenuItem, "Gamma").Click()
	h.settle(w)
	if value != "c" || selections != 1 {
		t.Errorf("got %q", value)
	}
	// Choosing the entry that is chosen already changes nothing.
	h.mustNode(kit.RoleButton, "Gamma").Click()
	h.settle(w)
	h.mustNode(kit.RoleMenuItem, "Gamma").Click()
	h.settle(w)
	if value != "c" || selections != 1 {
		t.Errorf("got %q and %d selections", value, selections)
	}
}

func TestDialogsAreModalAndGoWhenClosed(t *testing.T) {
	h := newHarness(t)
	confirmed := false
	page := kit.Button{ID: "page", Label: "Behind", OnClick: func() {}}.W()
	h.env.ShowDialog("ask", func(c kit.Ctx, closeDialog func()) kit.Dialog {
		return kit.Dialog{
			Icon: "uninstall", IconError: true, Title: "Delete it?",
			Content: kit.Label("This cannot be undone.", kit.Type(kit.BodyMedium), c.Env.Theme.Colors.OnSurface),
			Actions: []kit.Widget{
				kit.Button{ID: "cancel", Kind: kit.TextButton, Label: "Cancel", OnClick: closeDialog}.W(),
				kit.Button{ID: "ok", Label: "Delete", OnClick: func() { confirmed = true; closeDialog() }}.W(),
			},
		}
	})
	h.env.ShowDialog("ask", nil) // a dialog with the same id is not opened twice
	h.settle(page)
	if !h.env.HasDialog() || !h.hasText("Delete it?") || !h.hasText("This cannot be undone.") {
		t.Fatal("the dialog is shown")
	}
	dialog := h.mustNode(kit.RoleDialog, "Delete it?")
	ok := h.mustNode(kit.RoleButton, "Delete")
	if !ok.Modal || !ok.Rect.In(dialog.Rect) {
		t.Errorf("the buttons are in the dialog: %v %v", ok.Rect, dialog.Rect)
	}
	if got := h.mustNode(kit.RoleButton, "Behind"); got.Modal {
		t.Error("what is behind the dialog is not modal")
	}
	ok.Click()
	h.frames(60, page)
	if !confirmed {
		t.Error("expected the confirmation")
	}
	if h.env.HasDialog() {
		t.Error("the dialog is gone after it has closed")
	}
	if _, found := h.node(kit.RoleDialog, "Delete it?"); found {
		t.Error("the dialog has left the registry")
	}
}

func TestSnackbarsShowForAWhileAndReplaceEachOther(t *testing.T) {
	h := newHarness(t)
	acted := false
	page := fixed(10, 10)
	h.env.ShowSnackbar(kit.Snack{Text: "Saved", Action: "Undo", OnAction: func() { acted = true }, Closable: true, CloseLabel: "Close"})
	h.settle(page)
	if !h.hasText("Saved") {
		t.Fatal("the snack bar is shown")
	}
	h.mustNode(kit.RoleButton, "Undo").Click()
	h.frames(120, page)
	if !acted {
		t.Error("expected the action")
	}
	if h.hasText("Saved") {
		t.Error("the action dismisses the snack bar")
	}
	h.env.ShowSnackbar(kit.Snack{Text: "One"})
	h.env.ShowSnackbar(kit.Snack{Text: "Two", Persist: true})
	h.settle(page)
	if h.hasText("One") || !h.hasText("Two") {
		t.Error("a snack bar replaces the one before")
	}
	// A snack bar that persists stays; one that does not goes after a while.
	for range 400 {
		h.now = h.now.Add(50 * 1e6)
		h.frame(page)
	}
	if !h.hasText("Two") {
		t.Error("a snack bar that persists stays")
	}
	h.env.ShowSnackbar(kit.Snack{Text: "Brief"})
	for range 400 {
		h.now = h.now.Add(50 * 1e6)
		h.frame(page)
	}
	if h.hasText("Brief") {
		t.Error("a snack bar goes after four seconds")
	}
}

func TestTextFieldsKeepWhatTheUserTypesUntilTheValueChanges(t *testing.T) {
	h := newHarness(t)
	var changes, submits []string
	value := "start"
	w := dynamic(func() kit.Widget {
		return kit.Column(kit.FlexOpts{Min: true, Gap: 8},
			kit.Rigid(kit.TextField{
				ID: "f", Label: "Name", Value: value, Helper: "Helps", OnChange: func(s string) { changes = append(changes, s) },
				OnSubmit: func(s string) { submits = append(submits, s) },
			}.W()),
			kit.Rigid(kit.TextField{ID: "e", Label: "Broken", Error: "Wrong", Lines: 3, MaxLines: 5, Monospace: true}.W()),
			kit.Rigid(kit.TextField{ID: "d", Hint: "Dense", Dense: true, LeadingIcon: "search", Trailing: fixed(24, 24), Disabled: true}.W()),
		)
	})
	h.settle(w)
	if got := kit.FieldText(h.ctx(layout.Constraints{}), "f"); got != "start" {
		t.Errorf("got %q", got)
	}
	field := h.mustNode(kit.RoleTextField, "Name")
	field.Fill("typed", true)
	h.settle(w)
	// The text that the field starts with and the text that was filled in are not
	// changes that the user made, and are announced once, by Fill.
	if len(changes) != 1 || changes[0] != "typed" || len(submits) != 1 || submits[0] != "typed" {
		t.Errorf("got %q and %q", changes, submits)
	}
	if got := h.mustNode(kit.RoleTextField, "Name").Text; got != "typed" {
		t.Errorf("got %q", got)
	}
	value = "changed from outside"
	h.settle(w)
	if got := kit.FieldText(h.ctx(layout.Constraints{}), "f"); got != "changed from outside" {
		t.Errorf("got %q", got)
	}
	kit.SetFieldText(h.ctx(layout.Constraints{}), "f", "set")
	if got := kit.FieldText(h.ctx(layout.Constraints{}), "f"); got != "set" {
		t.Errorf("got %q", got)
	}
	if !h.hasText("Helps") || !h.hasText("Wrong") {
		t.Error("helper and error text are shown")
	}
	if h.mustNode(kit.RoleTextField, "Dense").Enabled {
		t.Error("a disabled field")
	}
	h.mustNode(kit.RoleTextField, "Name").Click()
	h.settle(w)
	kit.FocusField(h.ctx(layout.Constraints{}), "f")
	h.settle(w)
	if len(changes) != 1 {
		t.Errorf("programmatic changes are not announced: %q", changes)
	}
}

func TestSearchBarsClearTheirText(t *testing.T) {
	h := newHarness(t)
	text := ""
	w := dynamic(func() kit.Widget {
		return kit.SearchBar{ID: "s", Hint: "Search", Value: text, ClearTooltip: "Clear", OnChange: func(s string) { text = s }}.W()
	})
	h.settle(w)
	if _, ok := h.node(kit.RoleButton, "Clear"); ok {
		t.Error("an empty search bar has nothing to clear")
	}
	h.mustNode(kit.RoleTextField, "Search").Fill("node", false)
	h.settle(w)
	if text != "node" {
		t.Errorf("got %q", text)
	}
	h.settle(w)
	h.mustNode(kit.RoleButton, "Clear").Click()
	h.settle(w)
	if text != "" {
		t.Errorf("got %q", text)
	}
}

func TestListItemsAndGroupsAreSegmented(t *testing.T) {
	h := newHarness(t)
	picked := ""
	items := []kit.ListItem{
		{ID: "a", Title: "First", Subtitle: "One", OnClick: func() { picked = "first" }},
		{ID: "b", Title: "Second", Selected: true, Trailing: kit.Icon("chevronRight", 24, h.env.Theme.Colors.OnSurface, 0), OnClick: func() {}},
		{ID: "c", Title: "Third", WrapTrailing: true, Trailing: fixed(100, 30)},
	}
	w := kit.ListGroup(items...)
	h.frameIn(loose(400, 600), w)
	h.settle(w)
	if h.mustNode(kit.RoleListItem, "First").Click == nil || !h.mustNode(kit.RoleListItem, "Second").Selected {
		t.Error("items that react have a click, selected ones are marked")
	}
	if h.mustNode(kit.RoleListItem, "Third").Click != nil {
		t.Error("an item without an action cannot be clicked")
	}
	h.mustNode(kit.RoleListItem, "First").Click()
	h.settle(w)
	if picked != "first" {
		t.Errorf("got %q", picked)
	}
	for _, tc := range []struct {
		index, count int
		want         kit.Position
	}{{0, 1, kit.PosSingle}, {0, 3, kit.PosFirst}, {1, 3, kit.PosMiddle}, {2, 3, kit.PosLast}} {
		if got := kit.PositionOf(tc.index, tc.count); got != tc.want {
			t.Errorf("%d of %d: got %v", tc.index, tc.count, got)
		}
	}
}

func TestStatusMessagesCardsAndHeadersShowTheirText(t *testing.T) {
	h := newHarness(t)
	var tapped bool
	w := kit.Column(kit.FlexOpts{Cross: kit.CrossStretch},
		kit.Rigid(kit.PageHeader("Plugins", "Every plugin adds a tool.", nil, fixed(30, 30))),
		kit.Rigid(kit.GroupHeader("Group", fixed(20, 20))),
		kit.Rigid(kit.Card{ID: "card", Color: h.env.Theme.Colors.SurfaceContainer, Radius: 16, Name: "Card", OnClick: func() { tapped = true }, Child: fixed(100, 50)}.W()),
		kit.Expanded(1, kit.StatusMessage{Icon: "empty", Title: "Nothing here", Body: "Add something.", Actions: []kit.Widget{fixed(80, 40)}}.W()),
	)
	h.settle(w)
	for _, text := range []string{"Plugins", "Every plugin adds a tool.", "Group", "Nothing here", "Add something."} {
		if !h.hasText(text) {
			t.Errorf("missing %q", text)
		}
	}
	h.mustNode(kit.RoleButton, "Card").Click()
	h.settle(w)
	if !tapped {
		t.Error("expected a tap on the card")
	}
	h.settle(kit.Pane(kit.Center(kit.LoadingMessage("Loading versions", "Loading"))))
	if !h.hasText("Loading versions") {
		t.Error("a loading message shows its text")
	}
	h.settle(kit.ContentWidth(300, fixed(1000, 20)))
	h.settle(kit.PaneColor(&h.env.Theme.Colors.Surface, fixed(10, 10)))
	h.settle(kit.Panel{Color: h.env.Theme.Colors.Surface, Top: 4, Bottom: 20, Child: fixed(100, 40)}.W())
}

func TestNavigationRailAndBarSelectDestinations(t *testing.T) {
	h := newHarness(t)
	selected, expanded, fab := 0, false, false
	destinations := []kit.NavDestination{{Icon: "home", Label: "Home"}, {Icon: "sdks", Label: "SDKs"}, {Icon: "tasks", Label: "Tasks", Badge: 3}}
	rail := dynamic(func() kit.Widget {
		return kit.NavRail{
			ID: "rail", Destinations: destinations, Selected: selected, OnSelect: func(i int) { selected = i },
			Expanded: expanded, OnExpandedChange: func(e bool) { expanded = e },
			ExpandTooltip: "Show labels", CollapseTooltip: "Hide labels", FABIcon: "add", FABLabel: "Add plugin", OnFAB: func() { fab = true },
		}.W()
	})
	h.settle(rail)
	h.mustNode(kit.RoleButton, "SDKs").Click()
	h.settle(rail)
	if selected != 1 || !h.mustNode(kit.RoleButton, "SDKs").Selected {
		t.Errorf("got %d", selected)
	}
	h.mustNode(kit.RoleButton, "Show labels").Click()
	h.frames(60, rail)
	if !expanded {
		t.Error("expected the labels next to the icons")
	}
	if got := h.mustNode(kit.RoleButton, "Home").Rect.Dx(); got < 200 {
		t.Errorf("the expanded rail is 240 px wide, got %d", got)
	}
	h.mustNode(kit.RoleButton, "Add plugin").Click()
	h.settle(rail)
	if !fab {
		t.Error("expected the action button to work")
	}
	if !h.hasText("3") {
		t.Error("the badge shows the number of tasks")
	}
	bar := dynamic(func() kit.Widget {
		return kit.NavBar{ID: "bar", Destinations: destinations, Selected: selected, OnSelect: func(i int) { selected = i }}.W()
	})
	h.settle(bar)
	h.mustNode(kit.RoleButton, "Tasks").Click()
	h.settle(bar)
	if selected != 2 {
		t.Errorf("got %d", selected)
	}
}

func TestProgressIndicatorsAndShapesAreDrawn(t *testing.T) {
	h := newHarness(t)
	w := kit.Column(kit.FlexOpts{Cross: kit.CrossStretch, Min: true, Gap: 8},
		kit.Rigid(kit.WavyProgress{ID: "a", Value: 0.5, Label: "Half"}.W()),
		kit.Rigid(kit.WavyProgress{ID: "b", Value: 0}.W()),
		kit.Rigid(kit.WavyProgress{ID: "c", Value: 1}.W()),
		kit.Rigid(kit.WavyProgress{ID: "d", Indeterminate: true}.W()),
		kit.Rigid(kit.LoadingIndicator{ID: "l", Size: 64, Contained: true, Label: "Loading"}.W()),
		kit.Rigid(kit.ColorSwatch{ID: "sw", Seed: 0xFF3F6FD8, Label: "Blue", Selected: true, OnClick: func() {}}.W()),
		kit.Rigid(kit.Avatar{Name: "nodejs", Size: 48}.W("av")),
		kit.Rigid(kit.Avatar{Name: "", Icon: "sdks"}.W("av2")),
		kit.Rigid(kit.Tip("tip", "Official", kit.Icon("verified", 18, h.env.Theme.Colors.Primary, 1))),
		kit.Rigid(kit.Collapsible("c", true, fixed(100, 40))),
		kit.Rigid(kit.Collapsible("c2", false, fixed(100, 40))),
	)
	for range 30 {
		h.frame(w)
	}
	if _, ok := h.node(kit.RoleButton, "Blue"); !ok {
		t.Error("a swatch is a button named by its label")
	}
	h.env.Theme.ReducedMotion = true
	h.settle(w)
}

func TestAvatarsAreTheSameInEveryImplementation(t *testing.T) {
	// FNV-1a over the UTF-16 code units; the other implementations compute the
	// same numbers, which is what makes an avatar the same everywhere.
	for text, want := range map[string]uint32{
		"": 0x811C9DC5, "a": 0xE40C292C, "nodejs": 0x1F748B62, "maven": 0xA6E29488, "你好": 0xD7BC7166,
		// Characters outside the Basic Multilingual Plane are two code units.
		"😀": 0xCB31C4B8,
	} {
		if got := kit.StableHash(text); got != want {
			t.Errorf("%q: got %#x, want %#x", text, got, want)
		}
	}
	if kit.StableHash("maven") == kit.StableHash("mvn2") {
		t.Error("different names have different hashes")
	}
	shape := kit.DecorativeShape("maven")
	if again := kit.DecorativeShape("maven"); again != shape {
		t.Error("a name always gets the same shape")
	}
}

func TestWindowButtonsDrawTheirSymbolsAndRunTheirAction(t *testing.T) {
	h := newHarness(t)
	var clicked []string
	row := func(glyph kit.WindowGlyph, name string) kit.FlexChild {
		return kit.Rigid(kit.WindowButton{ID: name, Glyph: glyph, Name: name, OnClick: func() { clicked = append(clicked, name) }}.W())
	}
	w := kit.Row(kit.FlexOpts{Gap: 4, Min: true},
		row(kit.GlyphMinimize, "minimize"), row(kit.GlyphMaximize, "maximize"), row(kit.GlyphRestore, "restore"), row(kit.GlyphClose, "close"))
	h.settle(w)
	for _, name := range []string{"minimize", "maximize", "restore", "close"} {
		button := h.mustNode(kit.RoleButton, name)
		if button.Rect.Dx() != 32 || button.Rect.Dy() != 32 || button.Tooltip != "" {
			t.Errorf("%s: a round button of 32 px without a tooltip, got %#v", name, button)
		}
		button.Click()
		h.settle(w)
	}
	if len(clicked) != 4 || clicked[0] != "minimize" || clicked[3] != "close" {
		t.Errorf("got %q", clicked)
	}
}

func TestTheTitleOfAWindowIsWhereItIsDragged(t *testing.T) {
	h := newHarness(t)
	dims := h.frameIn(loose(300, 40), kit.WindowDrag(fixed(200, 30)))
	if dims.Size != image.Pt(200, 30) {
		t.Errorf("dragging does not change the layout, got %v", dims.Size)
	}
}
