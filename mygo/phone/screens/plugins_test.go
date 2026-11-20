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

package screens

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/phone/app"
	"github.com/version-fox/vfox-plus/mygo/phone/remote"
)

// pluginRegistry is an inventory with three plugins in the registry, of which go is
// official and installed; zig has a homepage that is not a web address.
func pluginRegistry() remote.Inventory {
	return remote.Inventory{
		Installed: remote.Section[[]remote.InstalledSdk]{
			Value: []remote.InstalledSdk{{Name: "go", Versions: []string{"1.23.0"}}},
		},
		Registry: remote.Section[[]remote.AvailablePlugin]{
			Value: []remote.AvailablePlugin{
				{Name: "go", Homepage: "https://github.com/version-fox/vfox-go", Official: true},
				{Name: "rust", Homepage: "https://example.com/rust/"},
				{Name: "zig", Homepage: "javascript:alert(1)"},
			},
		},
	}
}

// pluginOpenTab shows the plugin tab, and waits until the registry has loaded.
func pluginOpenTab(t *testing.T, root *Root, tt *ui.Tester) {
	t.Helper()
	root.App().SelectTab(app.TabPlugins)
	settle(tt)
	availWaitText(t, tt, "rust")
}

func TestThePluginTabListsTheRegistryWithItsSourcesAndMarks(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)
	for _, want := range []string{"Plugins", "Update all", "Add from source", "All", "Official", "Community", "version-fox/vfox-go", "example.com/rust", "zig", "Open", "Install"} {
		if !hasText(tt, want) {
			t.Errorf("the plugin tab has no %q; the frame shows %q", want, tt.Texts())
		}
	}
	// Each plugin has the avatar of its initial, which the web draws for its name.
	for _, initial := range []string{"G", "R", "Z"} {
		if !hasText(tt, initial) {
			t.Errorf("the plugin tab has no avatar %q; the frame shows %q", initial, tt.Texts())
		}
	}
	availShot(t, tt, "plugins-tab")
}

func TestThePluginSearchMatchesNamesAndHomepages(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)

	if err := tt.Click("Search plugins…"); err != nil {
		t.Fatalf("Click(search): %v", err)
	}
	tt.Type("rust")
	availWaitText(t, tt, "rust")
	if hasText(tt, "go") || hasText(tt, "zig") {
		t.Errorf("the search rust shows %q", tt.Texts())
	}
	availShot(t, tt, "plugins-search")

	tt.Key(0, ui.KeyBackspace)
	tt.Key(0, ui.KeyBackspace)
	tt.Key(0, ui.KeyBackspace)
	tt.Key(0, ui.KeyBackspace)
	tt.Type("nothing")
	availWaitText(t, tt, "No plugins match.")
	availShot(t, tt, "plugins-no-match")
}

func TestThePluginFilterKeepsTheOfficialOrTheCommunityPlugins(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)

	if err := tt.Click("Official"); err != nil {
		t.Fatalf("Click(Official): %v", err)
	}
	availWaitText(t, tt, "go")
	if hasText(tt, "rust") || hasText(tt, "zig") {
		t.Errorf("the official filter shows %q", tt.Texts())
	}

	if err := tt.Click("Community"); err != nil {
		t.Fatalf("Click(Community): %v", err)
	}
	availWaitText(t, tt, "zig")
	if hasText(tt, "go") || !hasText(tt, "rust") {
		t.Errorf("the community filter shows %q", tt.Texts())
	}
}

func TestInstallingAPluginFromTheRegistryStartsAddAndShowsTheTasks(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)

	if err := tt.Click("Install"); err != nil {
		t.Fatalf("Click(Install): %v", err)
	}
	calls := availWaitCalls(t, tt, f, 1)
	if got := calls[0]; got.Method != "POST" || got.Path != "/api/tasks" || got.Body != `{"operation":"add","source":"rust"}` {
		t.Errorf("the install sent %+v", got)
	}
	availWaitTab(t, tt, root, app.TabTasks)
}

func TestUpdateAllStartsAnUpdateOfEveryPlugin(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)

	if err := tt.Click("Update all"); err != nil {
		t.Fatalf("Click(Update all): %v", err)
	}
	calls := availWaitCalls(t, tt, f, 1)
	if got := calls[0].Body; got != `{"operation":"update"}` {
		t.Errorf("update all sent %s", got)
	}
	availWaitTab(t, tt, root, app.TabTasks)
}

func TestAnInstalledPluginOpensItsSDK(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)

	if err := tt.Click("Open"); err != nil {
		t.Fatalf("Click(Open): %v", err)
	}
	settle(tt)
	pages := root.App().Pages()
	if len(pages) != 1 || pages[0] != (app.Page{Kind: app.PageSDK, Name: "go"}) {
		t.Errorf("pages after Open: %v, want the SDK go", pages)
	}
}

func TestAddFromSourceSendsTheAddressAndTheAlias(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)

	if err := tt.Click("Add from source"); err != nil {
		t.Fatalf("Click(Add from source): %v", err)
	}
	availWaitText(t, tt, "Source URL")
	availShot(t, tt, "add-plugin-dialog")
	if hasText(tt, "Upload & add") {
		t.Error("the add dialog offers an upload, which the phone does not do")
	}
	tt.Type("https://example.com/x.git")
	if err := tt.Click("Alias (optional)"); err != nil {
		t.Fatalf("Click(Alias): %v", err)
	}
	tt.Type("xx")
	tt.Key(0, ui.KeyEnter)
	calls := availWaitCalls(t, tt, f, 1)
	if got := calls[0]; got.Method != "POST" || got.Path != "/api/plugins" || got.Body != `{"source":"https://example.com/x.git","alias":"xx"}` {
		t.Errorf("add from source sent %+v", got)
	}
	availWaitTab(t, tt, root, app.TabTasks)
}

func TestAddFromSourceNeedsASource(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)

	if err := tt.Click("Add from source"); err != nil {
		t.Fatalf("Click(Add from source): %v", err)
	}
	availWaitText(t, tt, "Source URL")
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if calls := f.recorded(); len(calls) != 0 {
		t.Errorf("an empty source was sent: %+v", calls)
	}
}

func TestAHomepageThatIsNotAWebAddressIsNotOpened(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)

	if err := tt.Click("Search plugins…"); err != nil {
		t.Fatalf("Click(search): %v", err)
	}
	tt.Type("zig")
	availWaitText(t, tt, "zig")
	if err := tt.Click("Homepage"); err != nil {
		t.Fatalf("Click(Homepage): %v", err)
	}
	availWaitText(t, tt, "This address is not a web address, so it is not opened.")
	if opened := tt.OpenedURLs(); len(opened) != 0 {
		t.Errorf("a refused homepage was opened: %v", opened)
	}
}

func TestAWebHomepageIsOpenedInTheBrowser(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)

	if err := tt.Click("Search plugins…"); err != nil {
		t.Fatalf("Click(search): %v", err)
	}
	tt.Type("vfox-go")
	availWaitText(t, tt, "go")
	if err := tt.Click("Homepage"); err != nil {
		t.Fatalf("Click(Homepage): %v", err)
	}
	tt.Frame()
	opened := tt.OpenedURLs()
	if len(opened) != 1 || opened[0] != "https://github.com/version-fox/vfox-go" {
		t.Errorf("the homepage opened %v", opened)
	}
}

func TestThePluginTabShowsItsLoadingUntilTheRegistryAnswers(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	release := f.holdReads(t)
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	root.App().SelectTab(app.TabPlugins)
	settle(tt)
	if !hasText(tt, "Loading…") {
		t.Errorf("the plugin tab does not show its loading; the frame shows %q", tt.Texts())
	}
	availShot(t, tt, "plugins-loading")
	release()
	availWaitText(t, tt, "rust")
}

func TestAFailedRegistryShowsItsErrorAndARetry(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = remote.Inventory{
		Registry: remote.Section[[]remote.AvailablePlugin]{Error: "the registry is offline"},
	}
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	root.App().SelectTab(app.TabPlugins)
	settle(tt)
	availWaitText(t, tt, "Request failed.")
	if !hasText(tt, "the registry is offline") || !hasText(tt, "Retry") {
		t.Errorf("the failure of the registry shows %q", tt.Texts())
	}
	availShot(t, tt, "plugins-failed")
}

func TestTheRegistryShowsItsFirstPluginsAndAddsMore(t *testing.T) {
	f := availNewComputer(t)
	var plugins []remote.AvailablePlugin
	for i := 0; i < 250; i++ {
		plugins = append(plugins, remote.AvailablePlugin{Name: fmt.Sprintf("plugin-%03d", i)})
	}
	f.inventory = remote.Inventory{Registry: remote.Section[[]remote.AvailablePlugin]{Value: plugins}}
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	root.App().SelectTab(app.TabPlugins)
	settle(tt)
	availWaitText(t, tt, "plugin-000")
	if !hasText(tt, "Show 50 more") || hasText(tt, "plugin-200") {
		t.Fatalf("the registry does not show its first 200 plugins with a button for the rest")
	}
	tt.Scroll(240, 500, 0, 40000)
	tt.Frame()
	if err := tt.Click("Show 50 more"); err != nil {
		t.Fatalf("Click(show more): %v", err)
	}
	availWaitText(t, tt, "plugin-249")
}

func TestAnEmptyRegistrySaysNoPluginMatches(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = remote.Inventory{Registry: remote.Section[[]remote.AvailablePlugin]{Value: []remote.AvailablePlugin{}}}
	root := availRoot(t, f)
	tt := ui.NewTester(root.View, 480, 900)
	root.App().SelectTab(app.TabPlugins)
	settle(tt)
	availWaitText(t, tt, "No plugins match.")
	availShot(t, tt, "plugins-empty")
}
