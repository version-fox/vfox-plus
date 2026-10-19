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
