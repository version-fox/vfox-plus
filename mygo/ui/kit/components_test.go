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
	"strconv"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

func TestButtonsSnapshot(t *testing.T) {
	for _, dark := range []bool{false, true} {
		name := "buttons-light"
		if dark {
			name = "buttons-dark"
		}
		env := testEnv(t, dark)
		snapshot(t, env, name, 900, 520, func(c Ctx) {
			cs := c.Colors()
			Column(c).Fill().Background(cs.Surface).Padding(24).Gap(16).Children(func() {
				Row(c).Gap(12).Children(func() {
					Button(c, ButtonSpec{Label: "Filled", Kind: FilledButton, OnClick: func() {}})
					Button(c, ButtonSpec{Label: "Tonal", Kind: TonalButton, Icon: "add", OnClick: func() {}})
					Button(c, ButtonSpec{Label: "Outlined", Kind: OutlinedButton, OnClick: func() {}})
					Button(c, ButtonSpec{Label: "Text", Kind: TextButton, OnClick: func() {}})
					Button(c, ButtonSpec{Label: "Disabled", Kind: FilledButton, Disabled: true})
					Button(c, ButtonSpec{Label: "Delete", Kind: FilledButton, Danger: true, OnClick: func() {}})
				})
				Row(c).Gap(12).Children(func() {
					IconButton(c, IconButtonSpec{Icon: "refresh", Tooltip: "Refresh", OnClick: func() {}})
					IconButton(c, IconButtonSpec{Icon: "more", Tooltip: "More", Kind: FilledIconButton, OnClick: func() {}})
					IconButton(c, IconButtonSpec{Icon: "copy", Tooltip: "Copy", Kind: TonalIconButton, OnClick: func() {}})
					FAB(c, FABSpec{Icon: "add", Label: "Add plugin", Expansion: 1, OnClick: func() {}})
					FAB(c, FABSpec{Icon: "add", Label: "Add plugin", Expansion: 0, OnClick: func() {}})
					Tag(c, TagSpec{Label: "Plugin 0.5.1"})
					Tag(c, TagSpec{Label: "In use: 24.11.1", Icon: "current", Emphasized: true})
				})
				ButtonGroup(c, ButtonGroupSpec{
					ID: "group", Selected: "b", OnSelect: func(string) {},
					Segments: []Segment{{Value: "a", Label: "Installed (2)", Icon: "versions"}, {Value: "b", Label: "Available", Icon: "install"}, {Value: "c", Label: "Other"}},
				}).AlignSelf(ui.Start)
				SplitButton(c, SplitButtonSpec{ID: "split", Label: "Use", Icon: "check", OnClick: func() {}, MenuTooltip: "More", Items: []MenuItem{{Label: "Use globally"}}})
				ListGroup(c,
					ListItemSpec{ID: "1", Title: "maven", Subtitle: "In use: 3.9.16", OnClick: func() {}},
					ListItemSpec{ID: "2", Title: "mvn2", Subtitle: "No version in use", OnClick: func() {}, Selected: true},
					ListItemSpec{ID: "3", Title: "nodejs", Subtitle: "In use: 24.11.1"},
				).MaxWidth(420)
			})
		})
	}
}

func TestFormsSnapshot(t *testing.T) {
	env := testEnv(t, false)
	text, query, notes := "nodejs", "", "FOO=bar\nBAZ=1"
	snapshot(t, env, "forms-light", 900, 640, func(c Ctx) {
		cs := c.Colors()
		Column(c).Fill().Background(cs.Surface).Padding(24).Gap(16).Children(func() {
			Row(c).Gap(24).Children(func() {
				Switch(c, SwitchSpec{ID: "off", Label: "Off", On: false, OnChange: func(bool) {}})
				Switch(c, SwitchSpec{ID: "on", Label: "On", On: true, OnChange: func(bool) {}})
				Switch(c, SwitchSpec{ID: "dis", Label: "Disabled", On: true, Disabled: true})
				Avatar(c, AvatarSpec{Name: "nodejs", Size: 40})
				Avatar(c, AvatarSpec{Name: "maven", Size: 40})
				Avatar(c, AvatarSpec{Name: "golang", Size: 72})
				ColorSwatch(c, SwatchSpec{ID: "a", Seed: 0xFFF28C1B, Label: "Orange", Selected: true, OnClick: func() {}})
				ColorSwatch(c, SwatchSpec{ID: "b", Seed: 0xFF1B8CF2, Label: "Blue", OnClick: func() {}})
				LoadingIndicator(c, LoadingSpec{ID: "l1", Size: 48})
				LoadingIndicator(c, LoadingSpec{ID: "l2", Size: 48, Contained: true})
			})
			TextField(c, TextFieldSpec{ID: "name", Label: "Directory", Text: &text, Helper: "Where the project lives"}).MaxWidth(420)
			TextField(c, TextFieldSpec{ID: "empty", Label: "Source", Hint: "https://", Value: ""}).MaxWidth(420)
			TextField(c, TextFieldSpec{ID: "err", Label: "Directory", Text: &text, Error: "The directory does not exist"}).MaxWidth(420)
			TextField(c, TextFieldSpec{ID: "env", Label: "Environment", Text: &notes, Lines: 3, MaxLines: 6, Monospace: true}).MaxWidth(420)
			SearchBar(c, SearchBarSpec{ID: "search", Hint: "Search installed plugins", Text: &query}).MaxWidth(420)
			WavyProgress(c, WavySpec{ID: "w1", Value: 0.4}).MaxWidth(420)
			WavyProgress(c, WavySpec{ID: "w2", Indeterminate: true}).MaxWidth(420)
			Dropdown(c, DropdownSpec{ID: "dd", Value: "b", Choices: []Choice{{"a", "System"}, {"b", "English"}}, OnSelect: func(string) {}})
		})
	})
}

func TestNavSnapshot(t *testing.T) {
	env := testEnv(t, false)
	destinations := []NavDestination{{Icon: "home", Label: "Home"}, {Icon: "sdks", Label: "SDKs"}, {Icon: "plugins", Label: "Plugins"}, {Icon: "tasks", Label: "Tasks", Badge: 2}, {Icon: "settings", Label: "Settings"}}
	for _, expanded := range []bool{false, true} {
		name := "rail-collapsed"
		if expanded {
			name = "rail-expanded"
		}
		snapshot(t, env, name, 700, 520, func(c Ctx) {
			cs := c.Colors()
			Row(c).Fill().Background(cs.Surface).AlignItems(ui.Stretch).Children(func() {
				NavRail(c, NavRailSpec{
					ID: "rail", Destinations: destinations, Selected: 1, OnSelect: func(int) {}, Expanded: expanded,
					OnExpandedChange: func(bool) {}, ExpandTooltip: "Expand", CollapseTooltip: "Collapse",
					FABIcon: "add", FABLabel: "Add plugin", OnFAB: func() {},
				})
				Pane(c, func() {}).Grow(1).Margin(16)
			})
		})
	}
	snapshot(t, env, "bar", 400, 520, func(c Ctx) {
		cs := c.Colors()
		Column(c).Fill().Background(cs.Surface).Children(func() {
			Pane(c, func() {}).Grow(1).Margin(16)
			NavBar(c, NavBarSpec{ID: "bar", Destinations: destinations, Selected: 3, OnSelect: func(int) {}})
		})
	})
}

func TestMenuAndDialogSnapshot(t *testing.T) {
	env := testEnv(t, false)
	build := func(c Ctx) {
		cs := c.Colors()
		Column(c).Fill().Background(cs.Surface).Padding(24).Gap(16).Children(func() {
			Row(c).Gap(12).Children(func() {
				IconButton(c, IconButtonSpec{ID: "more", Icon: "more", Tooltip: "More", Menu: []MenuItem{
					{Label: "Open homepage", Icon: "openInNew"}, {Label: "Remove plugin", Icon: "removePlugin", Danger: true}}, MenuAlignEnd: true})
				Button(c, ButtonSpec{ID: "dialog", Label: "Dialog", Kind: TonalButton, OnClick: func() {
					c.Env.ShowDialog("confirm", func(c Ctx, closeDialog func()) DialogSpec {
						return DialogSpec{
							Icon: "removePlugin", IconError: true, Title: "Remove plugin?",
							Content: func() {
								Label(c, "This deletes nodejs and its 2 installed versions.", Type(BodyMedium), c.Colors().OnSurfaceVariant)
							},
							Actions: []func(){
								func() { Button(c, ButtonSpec{ID: "cancel", Label: "Cancel", Kind: TextButton, OnClick: closeDialog}) },
								func() {
									Button(c, ButtonSpec{ID: "remove", Label: "Remove", Kind: FilledButton, Danger: true, OnClick: closeDialog})
								},
							},
						}
					})
				}})
			})
		})
		c.Env.LayoutDialogs(c)
		c.Env.LayoutSnackbars(c)
	}
	tt := ui.NewTester(view(env, build), 700, 420)
	if err := tt.Click("More"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	saveImage(t, "menu-open", tt.Image())
	if err := tt.Click("Open homepage"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	if err := tt.Click("Dialog"); err != nil {
		t.Fatal(err)
	}
	tt.Advance(time.Second)
	saveImage(t, "dialog-open", tt.Image())
	if !tt.HasText("Remove plugin?") {
		t.Fatalf("no dialog: %q", tt.Texts())
	}
}

// A dialog that is taller than its window scrolls its content, between the title and the actions, which stay where a person can reach
// them: a window as short as the least that the application opens in (480 dp) took the actions of the dialog of the mirrors and of
// the pairing, and nothing could close them.
func TestADialogThatIsTallerThanItsWindowScrollsItsContentAndKeepsItsActions(t *testing.T) {
	h := newTested(t, 360, 480, func(c Ctx) {
		Column(c).Fill().Padding(16).Children(func() {
			Button(c, ButtonSpec{ID: "open", Label: "Open", OnClick: func() {
				c.Env.ShowDialog("d", func(c Ctx, closeDialog func()) DialogSpec {
					return DialogSpec{
						Title: "Many lines",
						Content: func() {
							for i := 0; i < 40; i++ {
								Label(c, "Line "+strconv.Itoa(i)+" of the content", Type(BodyLarge), c.Colors().OnSurface)
							}
						},
						Actions: []func(){func() {
							Button(c, ButtonSpec{ID: "close", Label: "Close", OnClick: closeDialog})
						}},
					}
				})
			}})
		})
		c.Env.LayoutDialogs(c)
	})
	h.click("Open")
	h.tester.Advance(time.Second)
	dialog, ok := h.node(RoleDialog, "Many lines")
	if !ok || dialog.Rect.Y < 0 || dialog.Rect.Y+dialog.Rect.H > 480 {
		t.Fatalf("the dialog is in the window: %+v", dialog)
	}
	inside := func(n *Node) bool {
		return n != nil && n.Visible() && n.Rect.Y >= dialog.Rect.Y-0.5 && n.Rect.Y+n.Rect.H <= dialog.Rect.Y+dialog.Rect.H+0.5
	}
	if closeButton, _ := h.node(RoleButton, "Close"); !inside(closeButton) {
		t.Errorf("the action is in the dialog, where it can be pressed: %+v in %+v", closeButton, dialog.Rect)
	}
	last := func() *Node {
		for _, n := range h.env.Reg.Nodes() {
			if n.Role == RoleText && n.Text == "Line 39 of the content" && n.Visible() {
				return n
			}
		}
		return nil
	}
	if last() != nil {
		t.Errorf("the last line is below the fold of the content, not drawn outside the dialog: %+v", last().Rect)
	}
	for _, n := range h.env.Reg.Nodes() {
		if n.Modal && n.Role == RoleText && n.Visible() && (n.Rect.Y+n.Rect.H < dialog.Rect.Y || n.Rect.Y > dialog.Rect.Y+dialog.Rect.H) {
			t.Errorf("nothing of the dialog is drawn outside it: %q at %+v", n.Text, n.Rect)
		}
	}
	// The content scrolls to its end, and the action is still there.
	h.tester.Scroll(dialog.Rect.X+dialog.Rect.W/2, dialog.Rect.Y+dialog.Rect.H/2, 0, 100000)
	h.tester.Advance(time.Second)
	if n := last(); n == nil || !inside(n) {
		t.Errorf("the content scrolls to its last line: %+v in %+v", n, dialog.Rect)
	}
	if closeButton, _ := h.node(RoleButton, "Close"); !inside(closeButton) {
		t.Errorf("and the action stays: %+v", closeButton)
	}
}

// A menu that has more entries than the room that the window has for it scrolls, and the last entry can be reached: it went down
// the window and past it, and the entries below the edge were not there to be pressed (the addresses of a computer with a dozen
// networks, in the dropdown of the pairing).
func TestAMenuThatIsTallerThanItsWindowScrollsToItsLastEntry(t *testing.T) {
	var items []MenuItem
	for i := 0; i < 30; i++ {
		items = append(items, MenuItem{Label: "Entry " + strconv.Itoa(i), OnClick: func() {}})
	}
	h := newTested(t, 300, 300, func(c Ctx) {
		Column(c).Background(c.Colors().Surface).Padding(16).Children(func() {
			IconButton(c, IconButtonSpec{ID: "more", Icon: "more", Tooltip: "More", Menu: items})
		})
	})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	h.click("More")
	h.tester.Advance(time.Second)
	entry := func(name string) *Node {
		for _, n := range h.env.Reg.Nodes() {
			if n.Role == RoleMenuItem && n.Name == name && n.Visible() {
				return n
			}
		}
		return nil
	}
	first := entry("Entry 0")
	if first == nil || first.Rect.Y < 0 || first.Rect.Y+first.Rect.H > 300 {
		t.Fatalf("the first entry is in the window: %+v", first)
	}
	if entry("Entry 29") != nil {
		t.Errorf("the last entry is below the fold of the menu, not drawn: %+v", entry("Entry 29").Rect)
	}
	h.tester.Scroll(first.Rect.X+first.Rect.W/2, first.Rect.Y+first.Rect.H/2, 0, 100000)
	h.tester.Advance(time.Second)
	if last := entry("Entry 29"); last == nil || last.Rect.Y < 0 || last.Rect.Y+last.Rect.H > 300 {
		t.Errorf("the menu scrolls to its last entry, which is in the window: %+v", last)
	}
	// The census asks where the panel is, and it is in the window.
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == RoleMenu && n.Visible() {
			if census := TakeCensus(h.env.Reg.Nodes(), ui.Rect{W: 300, H: 300}, nil); len(census.Spilling) != 0 {
				t.Errorf("a menu that scrolls is in its window: %q", census.Spilling)
			}
			return
		}
	}
	t.Error("the panel of the menu is a node of the registry, for the census")
}

// An entry of a menu whose text is wider than the menu can be goes on more lines, and is not cut with an ellipsis: at 200 % the
// text of the system made "Use as global default" 297 dp wide in a menu that is 280 dp at the most, and what was left of it was
// "Use as global d…", and "Open homepage" was cut, too. Material's menus do the same: an entry is 48 dp high, or as high as its
// lines need.
func TestAMenuEntryThatIsWiderThanTheMenuGoesOnMoreLines(t *testing.T) {
	const long = "Use this version everywhere on this computer, in every project that has no version of its own"
	h := newTested(t, 400, 500, func(c Ctx) {
		Column(c).Background(c.Colors().Surface).Padding(16).Children(func() {
			IconButton(c, IconButtonSpec{ID: "more", Icon: "more", Tooltip: "More", Menu: []MenuItem{
				{Label: long, OnClick: func() {}},
				{Label: "Short", OnClick: func() {}},
			}})
		})
	})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	h.click("More")
	h.tester.Advance(time.Second)
	entry, ok := h.node(RoleMenuItem, long)
	if !ok || !entry.Visible() {
		t.Fatalf("the entry is drawn: %+v", entry)
	}
	var text *Node
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == RoleText && n.Text == long && n.Visible() {
			text = n
		}
	}
	if text == nil {
		t.Fatal("the text of the entry is drawn")
	}
	if text.Rect.H < 2*24-1 {
		t.Errorf("the text goes on more lines than one where it is wider than the menu: %.0f dp high", text.Rect.H)
	}
	if text.Rect.X+text.Rect.W > entry.Rect.X+entry.Rect.W+0.5 || entry.Rect.W > 280+0.5 {
		t.Errorf("the text is in its entry and the entry in a menu of 280 dp at the most: text %+v, entry %+v", text.Rect, entry.Rect)
	}
	if text.Natural != 0 {
		t.Errorf("a text that wraps is not one that an ellipsis cuts: it wants %.0f dp", text.Natural)
	}
	if short, ok := h.node(RoleMenuItem, "Short"); !ok || short.Rect.H < 48-0.5 || short.Rect.H > 48+0.5 {
		t.Errorf("an entry that is one line is 48 dp high: %+v", short)
	}
}

// The tracking that a test asks for widens the text that the labels draw by the width that it says, and what measures a text for a
// layout is told of it: a layout decides on the width that is drawn, and one that decided on the width without it would fit a
// group of segments into a room that its labels do not fit.
func TestTrackingWidensWhatIsDrawnAndWhatIsMeasuredAlike(t *testing.T) {
	const text = "Use as global default"
	style := Type(BodyLarge)
	widths := func(tracking float32) (natural, drawn, measured float32) {
		t.Helper()
		Tracking = tracking
		defer func() { Tracking = 0 }()
		h := newTested(t, 600, 200, func(c Ctx) {
			Row(c).Padding(16).Children(func() {
				Line(c, text, style, c.Colors().OnSurface)
				measured, _ = c.MeasureText(0, ui.Span{Text: text, Size: style.Size, Weight: style.Weight})
			})
		})
		h.tester.Frame()
		h.tester.Advance(time.Second)
		for _, n := range h.env.Reg.Nodes() {
			if n.Role == RoleText && n.Text == text {
				natural, drawn = n.Natural, n.Rect.W
			}
		}
		return natural, drawn, measured
	}
	natural, drawn, measured := widths(0)
	if natural <= 0 || math.Abs(float64(drawn-natural)) > 1 || math.Abs(float64(measured-natural)) > 1 {
		t.Fatalf("without tracking, what is drawn and what is measured are the width of the text: %.1f, %.1f, %.1f", natural, drawn, measured)
	}
	// The spacing goes after every character, or after every one but the last, as the text engine of the system has it.
	const tracking = 0.2
	each := tracking * style.Size
	characters := float32(utf8.RuneCountInString(text))
	natural2, drawn2, measured2 := widths(tracking)
	for name, got := range map[string]float32{"wanted": natural2, "drawn": drawn2, "measured": measured2} {
		if more := got - natural; more < (characters-1)*each-1 || more > characters*each+1 {
			t.Errorf("with a tracking of %.1f em, the text that is %s is %.1f dp wider, where %.1f to %.1f was asked", tracking, name, more, (characters-1)*each, characters*each)
		}
	}
	if math.Abs(float64(drawn2-measured2)) > 1 || math.Abs(float64(natural2-measured2)) > 1 {
		t.Errorf("what is drawn and what is measured are alike with tracking, too: %.1f, %.1f, %.1f", natural2, drawn2, measured2)
	}
	if Tracking != 0 {
		t.Errorf("the tracking is back to nothing: %v", Tracking)
	}
}

// A text field that has a label and a text holds both at every size of text. It was 56 dp high at every size, and at 200 % the label
// that floats above the text was drawn over it ("Version name" over "17-system"), and a label that rests in a field with a button at
// its end ran on under the button. Material's field is as high as a label and a line need: 56 dp at 100 %.
func TestATextFieldHoldsItsLabelAndItsTextAtEverySizeOfText(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		env := NewEnv(NewTheme(testSeed, false, false, scale))
		env.Reg.CollectText = true
		loadTestIcons(t)
		value := "17-system"
		var rested *string
		tester := ui.NewTester(view(env, func(c Ctx) {
			Column(c).Padding(24).Gap(24).Children(func() {
				TextField(c, TextFieldSpec{ID: "raised", Label: "Version name", Text: &value, Monospace: true})
				empty := ""
				rested = &empty
				TextField(c, TextFieldSpec{ID: "rested", Label: "SDK directory", Text: rested, Monospace: true, Trailing: func() {
					IconButton(c, IconButtonSpec{ID: "browse", Icon: "folder", Tooltip: "Browse", Size: 32})
				}})
			})
		}), 360, 600)
		tester.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: scale})
		tester.Frame()
		tester.Advance(time.Second)
		census := TakeCensus(env.Reg.Nodes(), ui.Rect{W: 360, H: 600}, nil)
		if len(census.Overlapping) != 0 {
			t.Errorf("at %.0f %% the label and the text are not drawn over one another: %q", scale*100, census.Overlapping)
		}
		var fields []*Node
		var label, browse *Node
		for _, n := range env.Reg.Nodes() {
			switch {
			case n.Role == RoleTextField && n.Visible():
				fields = append(fields, n)
			case n.Role == RoleText && n.Text == "SDK directory":
				label = n
			case n.Role == RoleButton && n.Name == "Browse":
				browse = n
			}
		}
		if len(fields) != 2 || label == nil || browse == nil {
			t.Fatalf("at %.0f %% the fields, the label and the button are drawn: %d, %v, %v", scale*100, len(fields), label, browse)
		}
		if scale == 1 && (fields[0].Rect.H < 56-0.5 || fields[0].Rect.H > 56+0.5) {
			t.Errorf("at 100 %% a field is 56 dp high: %.1f", fields[0].Rect.H)
		}
		// The fields of the test are monospaced, which is the text of 14 dp over a line of 20: 6 + 16 (the label) + 20 + 10 at 100 %,
		// which is under the 56 dp that a field is at least.
		if need := max(56, 16+36*scale); fields[0].Rect.H < need-0.5 || fields[0].Rect.H > need+0.5 {
			t.Errorf("at %.0f %% a field is as high as its label and its line need, %.0f dp: %.1f", scale*100, need, fields[0].Rect.H)
		}
		if label.Rect.X+label.Rect.W > browse.Rect.X+0.5 {
			t.Errorf("at %.0f %% the label of a field that rests ends at %.1f, where the button at its end begins at %.1f", scale*100, label.Rect.X+label.Rect.W, browse.Rect.X)
		}
	}
}

// The labels of the navigation bar are read at the size of the text of the system up to 130 %, and smaller where the longest of them
// does not fit its fifth of the window, down to 100 %: "Settings" was cut at 130 % in a font that is wider by a third. A label that
// does not fit at 100 % is cut there, at the size of the text of the interface, and not smaller.
func TestTheLabelsOfTheNavigationBarShrinkToTheirSlotsAndNotBelowTheSizeOfText(t *testing.T) {
	destinations := func(labels ...string) []NavDestination {
		var out []NavDestination
		for _, label := range labels {
			out = append(out, NavDestination{Icon: "home", Label: label})
		}
		return out
	}
	usual := destinations("Home", "SDKs", "Plugins", "Tasks", "Settings")
	for _, c := range []struct {
		name         string
		scale        float32
		tracking     float32
		destinations []NavDestination
		fits         bool
	}{
		{"the usual labels at 100 %", 1, 0, usual, true},
		{"the usual labels at 130 %", 1.3, 0, usual, true},
		{"the usual labels at 200 %, which the bar reads at 130 %", 2, 0, usual, true},
		{"the usual labels at 200 % in a text that is wider by a third", 2, 0.16, usual, true},
		{"a label that does not fit at 100 %", 1, 0, destinations("Home", "SDKs", "A destination of a long name", "Tasks", "Settings"), false},
	} {
		Tracking = c.tracking
		var atHundred float32
		h := newTested(t, 360, 200, func(ctx Ctx) {
			// The width of the third label at 100 %, the least that the text of the interface is.
			atHundred, _ = ctx.MeasureText(0, ui.Span{Text: c.destinations[2].Label, Size: Type(LabelMedium).Size, Weight: 500})
			Column(ctx).Fill().Children(func() {
				Spacer(ctx)
				NavBar(ctx, NavBarSpec{ID: "bar", Destinations: c.destinations, Selected: 4, OnSelect: func(int) {}})
			})
		})
		h.env.Theme.TextScale = c.scale
		h.tester.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: c.scale})
		h.tester.Frame()
		h.tester.Advance(time.Second)
		Tracking = 0
		cut, drawn := 0, 0
		for _, n := range h.env.Reg.Nodes() {
			if n.Role == RoleText && n.Natural > 0 && n.Visible() {
				drawn++
				if n.Rect.W < n.Natural-cutSlack {
					cut++
				}
			}
		}
		if drawn != len(c.destinations) {
			t.Fatalf("%s: %d labels are drawn of %d", c.name, drawn, len(c.destinations))
		}
		if (cut == 0) != c.fits {
			t.Errorf("%s: %d labels are cut, and the labels fit is %v", c.name, cut, c.fits)
		}
		// The bar is as high as the labels need, and they are in it, in the window too: at 130 % a label ended 5 dp below the bar, which is
		// the foot of the window.
		census := TakeCensus(h.env.Reg.Nodes(), ui.Rect{W: 360, H: 200}, nil)
		if len(census.Outside) != 0 {
			t.Errorf("%s: the labels are in the bar: %q", c.name, census.Outside)
		}
		for _, n := range h.env.Reg.Nodes() {
			if n.Role == RoleButton && n.Visible() {
				// As high as the labels need: not less than the nominal height at that size of text, and a line that is higher than the
				// one that a label asks for makes it higher (a font that has a higher line, or Han characters, by a few dp).
				if want := NavBarHeight + Type(LabelMedium).Line*(min(c.scale, barLabelMaxScale)-1); c.scale > 1 && (n.Rect.H < want-0.5 || n.Rect.H > want+barLineSlack) {
					t.Errorf("%s: the bar is %.1f dp high where it is %.1f to %.1f at that size of text", c.name, n.Rect.H, want, want+barLineSlack)
				}
				if n.Rect.Y+n.Rect.H > 200+0.5 {
					t.Errorf("%s: the bar ends at %.1f, below the window of 200", c.name, n.Rect.Y+n.Rect.H)
				}
			}
		}
		if !c.fits {
			// What does not fit is cut at 100 %: the label wants the width of the text at that size, and is not made smaller.
			for _, n := range h.env.Reg.Nodes() {
				if n.Role == RoleText && n.Text == c.destinations[2].Label && math.Abs(float64(n.Natural-atHundred)) > 1 {
					t.Errorf("%s: the label that does not fit is set at 100 %%, %.1f dp wide, and is %.1f", c.name, atHundred, n.Natural)
				}
			}
		}
	}
}

// A tooltip is shown over what is under it, which is what it is for: its text is of the scope of the tooltips, which the census of
// the texts that are drawn over one another leaves out.
func TestATooltipIsATextOfTheScopeOfTooltips(t *testing.T) {
	h := newTested(t, 300, 200, func(c Ctx) {
		Row(c).Padding(16).Children(func() {
			IconButton(c, IconButtonSpec{ID: "more", Icon: "more", Tooltip: "More actions"})
		})
	})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	button, ok := h.node(RoleButton, "More actions")
	if !ok {
		t.Fatal("the button is drawn")
	}
	h.tester.Move(button.Rect.X+button.Rect.W/2, button.Rect.Y+button.Rect.H/2)
	h.tester.Advance(2 * time.Second)
	found := false
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == RoleText && n.Text == "More actions" {
			found = true
			if n.Scope != "tooltip" {
				t.Errorf("the text of a tooltip is of the scope of the tooltips: %q", n.Scope)
			}
		}
	}
	if !found {
		t.Error("the tooltip is shown where the pointer rests on its button")
	}
}

// The labels of the navigation rail fit its 96 dp at every size of text, and the items of it are as high as their labels: the rail was
// as wide and as high at 200 % as it was at 100 %, and the label of "Settings" was cut, and that of "Plugins" went down over the badge
// of the item below it.
func TestTheLabelsOfTheNavigationRailFitTheRailAndTheItemsAreAsHighAsTheirLabels(t *testing.T) {
	destinations := []NavDestination{
		{Icon: "home", Label: "Home"}, {Icon: "home", Label: "SDKs"}, {Icon: "home", Label: "Plugins"},
		{Icon: "home", Label: "Tasks", Badge: 1}, {Icon: "home", Label: "Settings"},
	}
	for _, c := range []struct {
		name     string
		scale    float32
		tracking float32
	}{
		{"100 %", 1, 0},
		{"100 % in a text that is wider by two fifths", 1, 0.19},
		{"200 %", 2, 0},
		{"200 % in a text that is wider by a fifth", 2, 0.095},
	} {
		Tracking = c.tracking
		h := newTested(t, 400, 700, func(ctx Ctx) {
			Row(ctx).Fill().AlignItems(ui.Start).Children(func() {
				NavRail(ctx, NavRailSpec{ID: "rail", Destinations: destinations, Selected: 2, OnSelect: func(int) {}})
			})
		})
		h.env.Theme.TextScale = c.scale
		h.tester.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: c.scale})
		h.tester.Frame()
		h.tester.Advance(time.Second)
		Tracking = 0
		census := TakeCensus(h.env.Reg.Nodes(), ui.Rect{W: 400, H: 700}, nil)
		if len(census.Cut) != 0 {
			t.Errorf("at %s the labels of the rail are not cut: %q", c.name, census.Cut)
		}
		if len(census.Overlapping) != 0 {
			t.Errorf("at %s the labels of the rail are not drawn over one another: %q", c.name, census.Overlapping)
		}
		// Each label is in its item, which is what keeps it from the badge of the next.
		var items, labels []*Node
		for _, n := range h.env.Reg.Nodes() {
			switch {
			case n.Role == RoleButton && n.Visible() && n.Rect.W == railCollapsed:
				items = append(items, n)
			case n.Role == RoleText && n.Visible() && n.Natural > 0 && n.Text != "1":
				labels = append(labels, n)
			}
		}
		if len(items) != len(destinations) || len(labels) != len(destinations) {
			t.Fatalf("at %s the %d items and the %d labels are drawn: %d, %d", c.name, len(destinations), len(destinations), len(items), len(labels))
		}
		for i, label := range labels {
			if label.Rect.Y+label.Rect.H > items[i].Rect.Y+items[i].Rect.H+0.5 {
				t.Errorf("at %s the label %q ends at %.1f, below its item, which ends at %.1f", c.name, label.Text, label.Rect.Y+label.Rect.H, items[i].Rect.Y+items[i].Rect.H)
			}
		}
	}
}

// The keys move through the entries of a menu, which a scroll container holds, and Enter chooses the one that has the focus: the
// first when the menu opens, the next after Down, and the one before it after Up.
func TestTheKeysMoveThroughTheEntriesOfAMenuThatScrolls(t *testing.T) {
	chosen := ""
	entry := func(label string) MenuItem { return MenuItem{Label: label, OnClick: func() { chosen = label }} }
	for _, c := range []struct {
		keys []ui.Key
		want string
	}{
		{[]ui.Key{ui.KeyEnter}, "One"},
		{[]ui.Key{ui.KeyDown, ui.KeyEnter}, "Two"},
		{[]ui.Key{ui.KeyDown, ui.KeyDown, ui.KeyEnter}, "Three"},
		{[]ui.Key{ui.KeyDown, ui.KeyDown, ui.KeyUp, ui.KeyEnter}, "Two"},
	} {
		chosen = ""
		h := newTested(t, 300, 300, func(ctx Ctx) {
			Column(ctx).Padding(16).Children(func() {
				IconButton(ctx, IconButtonSpec{ID: "more", Icon: "more", Tooltip: "More", Menu: []MenuItem{entry("One"), entry("Two"), entry("Three")}})
			})
		})
		h.tester.Frame()
		h.tester.Advance(time.Second)
		h.click("More")
		h.tester.Advance(time.Second)
		for _, key := range c.keys {
			h.tester.Key(0, key)
			h.tester.Advance(time.Second)
		}
		if chosen != c.want {
			t.Errorf("the keys %v choose %q, want %q", c.keys, chosen, c.want)
		}
	}
}

// The title of a row is a name, and goes on as many lines as it needs: it is not held to a line, so that a version that is longer than the
// room is read whole, and the census has nothing to find cut in it.
func TestTheTitleOfARowGoesOnAsManyLinesAsItNeeds(t *testing.T) {
	h := newTested(t, 360, 400, func(c Ctx) {
		Column(c).Padding(16).Children(func() {
			ListGroup(c, ListItemSpec{ID: "row", Title: "21.0.12.1+1-tem-and-a-long-name", Subtitle: "Temurin"})
		})
	})
	h.env.Theme.TextScale = 2
	h.tester.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 2})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	var title *Node
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == RoleText && n.Text == "21.0.12.1+1-tem-and-a-long-name" {
			title = n
		}
	}
	if title == nil || !title.Visible() {
		t.Fatalf("the title of the row is drawn: %+v", title)
	}
	if title.Lines != 0 || title.Needs != 0 {
		t.Errorf("the title of a row has no limit of lines, which a name would be cut by: %d lines, %d needed", title.Lines, title.Needs)
	}
	if title.Rect.H < 2*Type(BodyLarge).Line*2-1 {
		t.Errorf("the title of the row goes on more than one line at 200 %%: %.1f dp high", title.Rect.H)
	}
	if got := TakeCensus(h.env.Reg.Nodes(), ui.Rect{W: 360, H: 400}, nil); len(got.Cut) != 0 {
		t.Errorf("the title of a row is not cut: %q", got.Cut)
	}
}

// A label of a field that has not the room at rest goes on as many lines as it needs, and is not cut: "SDK directory" wants 271 dp at 200 % in
// a text that is wider by two fifths, and the field of the dialog that links a version has 232.
func TestALabelOfAFieldThatHasNotTheRoomGoesOnAsManyLinesAsItNeeds(t *testing.T) {
	for _, c := range []struct {
		width float32
		lines int
	}{
		{300, 1}, // 268 dp of room: the label is one line
		{200, 2}, // 168 dp: two, and the field is as high as they need
	} {
		value := ""
		h := newTested(t, 400, 300, func(ctx Ctx) {
			Column(ctx).Padding(24).Children(func() {
				TextField(ctx, TextFieldSpec{ID: "path", Label: "SDK directory", Text: &value, Monospace: true}).Width(c.width)
			})
		})
		h.env.Theme.TextScale = 2
		h.tester.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 2})
		for range 3 {
			h.tester.Frame()
			h.tester.Advance(time.Second)
		}
		var label *Node
		var field *Node
		for _, n := range h.env.Reg.Nodes() {
			switch {
			case n.Role == RoleText && n.Text == "SDK directory":
				label = n
			case n.Role == RoleTextField:
				field = n
			}
		}
		if label == nil || !label.Visible() || field == nil {
			t.Fatalf("the label and the field at %.0f dp are drawn: %+v, %+v", c.width, label, field)
		}
		// A label of one line wants a width (Natural), and one of more says how many lines it may have and needs.
		if c.lines == 1 && (label.Lines != 0 || label.Natural > label.Rect.W+0.5) {
			t.Errorf("the label at %.0f dp is one line of %.1f dp in %.1f: %+v", c.width, label.Natural, label.Rect.W, label)
		}
		if c.lines > 1 && (label.Lines != c.lines || label.Needs > label.Lines) {
			t.Errorf("the label at %.0f dp is on %d lines of the %d that it needs, where it should be on %d", c.width, label.Lines, label.Needs, c.lines)
		}
		if received := label.Rect.H; received < float32(c.lines)*Type(BodyLarge).Line*2-1 {
			t.Errorf("the label at %.0f dp is %.1f dp high where %d lines are %.1f", c.width, received, c.lines, float32(c.lines)*Type(BodyLarge).Line*2)
		}
		if c.lines > 1 && field.Rect.H < float32(c.lines)*Type(BodyLarge).Line*2 {
			t.Errorf("the field at %.0f dp is %.1f dp high, where its label needs %d lines", c.width, field.Rect.H, c.lines)
		}
		if got := TakeCensus(h.env.Reg.Nodes(), ui.Rect{W: 400, H: 300}, nil); len(got.Cut) != 0 {
			t.Errorf("the label of a field is not cut: %q", got.Cut)
		}
	}
}

// A tooltip is the name of its button, and goes on as many lines as it needs: at 200 % a name of 240 dp wants three, and the two that it was
// held to cut it. It is drawn in the window, too.
func TestATooltipGoesOnAsManyLinesAsItNeedsAndIsDrawnInTheWindow(t *testing.T) {
	const tip = "More ways to install this version of the SDK, from a plugin or from a link"
	h := newTested(t, 360, 480, func(c Ctx) {
		Column(c).Fill().Padding(16).Children(func() {
			Spacer(c)
			Row(c).Justify(ui.End).Children(func() {
				IconButton(c, IconButtonSpec{ID: "more", Icon: "more", Tooltip: tip})
			})
		})
	})
	h.tester.SetPreferences(ui.Preferences{ReduceMotion: true, TextScale: 2})
	h.env.Theme.TextScale = 2
	h.tester.Frame()
	h.tester.Advance(time.Second)
	button, ok := h.node(RoleButton, tip)
	if !ok {
		t.Fatal("the button is drawn")
	}
	h.tester.Move(button.Rect.X+button.Rect.W/2, button.Rect.Y+button.Rect.H/2)
	h.tester.Advance(2 * time.Second)
	var text *Node
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == RoleText && n.Text == tip && n.Scope == "tooltip" {
			text = n
		}
	}
	if text == nil || !text.Visible() {
		t.Fatalf("the tooltip is drawn: %+v", text)
	}
	if text.Rect.H < 3*Type(BodySmall).Line*2-1 {
		t.Errorf("the tooltip goes on the three lines that it needs at 200 %%: %.1f dp high", text.Rect.H)
	}
	if text.Rect.X < 0 || text.Rect.Y < 0 || text.Rect.X+text.Rect.W > 360.5 || text.Rect.Y+text.Rect.H > 480.5 {
		t.Errorf("the tooltip is in the window: %+v", text.Rect)
	}
	if got := TakeCensus(h.env.Reg.Nodes(), ui.Rect{W: 360, H: 480}, nil); len(got.Cut) != 0 {
		t.Errorf("the text of a tooltip is not cut: %q", got.Cut)
	}
}

// The title of a group is copy, and is not cut where the room is short: it goes on as many lines as it needs, where a name that is
// data may end with an ellipsis (it was held to two lines, and "Use the versions i…", a title of three at 200 % in a wide font, was cut).
func TestTheTitleOfAGroupGoesOnAsManyLinesAsItNeeds(t *testing.T) {
	const title = "A title of a group that is far too long for the room that it has"
	h := newTested(t, 300, 400, func(c Ctx) {
		ui.Column(c.Context).Background(c.Colors().Surface).Width(120).Children(func() {
			GroupHeader(c, title, nil)
		})
	})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == RoleText && n.Text == title {
			if lines := n.Rect.H / Type(TitleSmall).Line; lines < 3 || n.Rect.W > 120 {
				t.Errorf("a title that is far too long for 120 dp has as many lines as it needs, in the width of its room: %+v, %.1f lines", n.Rect, lines)
			}
			return
		}
	}
	t.Fatalf("the title was not drawn: %q", h.tester.Texts())
}

// What does not fit in the room that it has wraps, in the components that hold a label: a chip, a tag, and a button that wraps.
func TestWhatHoldsALabelWrapsItWhereTheRoomIsShort(t *testing.T) {
	const room = 160
	h := newTested(t, 300, 400, func(c Ctx) {
		cs := c.Colors()
		ui.Column(c.Context).Background(cs.Surface).Padding(16).Gap(8).Children(func() {
			ui.Row(c.Context).Width(room).Wrap().Gap(8).Children(func() {
				FilterChip(c, ChipSpec{ID: "long", Label: "Beijing Foreign Studies University"})
			})
			ui.Row(c.Context).Width(room).Wrap().Gap(8).Children(func() {
				Tag(c, TagSpec{Label: "In use: 21.0.12.1+1-tem"})
			})
			ui.Row(c.Context).Width(room).Children(func() {
				Button(c, ButtonSpec{ID: "two", Label: "Distribution: OpenJDK", Kind: TonalButton}).Shrink(1)
			})
			ui.Row(c.Context).Width(room).Children(func() {
				Button(c, ButtonSpec{ID: "one", Label: "Fits", Kind: TonalButton})
			})
		})
	})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	var chip, two, one *Node
	for _, n := range h.env.Reg.Nodes() {
		switch {
		case n.Role == RoleButton && n.Name == "Beijing Foreign Studies University":
			chip = n
		case n.Role == RoleButton && n.Name == "Distribution: OpenJDK":
			two = n
		case n.Role == RoleButton && n.Name == "Fits":
			one = n
		}
	}
	if chip == nil || two == nil || one == nil {
		t.Fatalf("no chip, or button: %v %v %v", chip, two, one)
	}
	if chip.Rect.W > room+0.5 || chip.Rect.H <= chipHeight {
		t.Errorf("a chip that is too long is as wide as the room and has a second line: %+v", chip.Rect)
	}
	if two.Rect.W > room+0.5 || two.Rect.H <= buttonHeight {
		t.Errorf("a button with two lines is as wide as the room and has a second line: %+v", two.Rect)
	}
	if one.Rect.H != buttonHeight || chip.Rect.X < 0 {
		t.Errorf("a button with one line is 40 dp high: %+v", one.Rect)
	}
	got := TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 300, 400), nil)
	if len(got.Cut) != 0 || got.Overflow {
		t.Errorf("nothing is cut and nothing is across the edge: %q", got.Cut)
	}
}
