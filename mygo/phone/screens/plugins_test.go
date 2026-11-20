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
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

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
	calls := availWaitCalls(t, tt, f)
	if got := calls[0]; got.Method != "POST" || got.Path != "/api/tasks" || got.Body != `{"operation":"add","source":"rust"}` {
		t.Errorf("the install sent %+v", got)
	}
	availWaitTab(t, tt, root)
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
	calls := availWaitCalls(t, tt, f)
	if got := calls[0].Body; got != `{"operation":"update"}` {
		t.Errorf("update all sent %s", got)
	}
	availWaitTab(t, tt, root)
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
	// The dialog is the web's: the source and the alias, and under them the upload of a file.
	for _, want := range []string{"Upload & add", "Choose a file"} {
		if !hasText(tt, want) {
			t.Errorf("the add dialog does not show %q; it shows %q", want, tt.Texts())
		}
	}
	tt.Type("https://example.com/x.git")
	if err := tt.Click("Alias (optional)"); err != nil {
		t.Fatalf("Click(Alias): %v", err)
	}
	tt.Type("xx")
	tt.Key(0, ui.KeyEnter)
	calls := availWaitCalls(t, tt, f)
	if got := calls[0]; got.Method != "POST" || got.Path != "/api/plugins" || got.Body != `{"source":"https://example.com/x.git","alias":"xx"}` {
		t.Errorf("add from source sent %+v", got)
	}
	availWaitTab(t, tt, root)
}

// pluginFile writes a file with the name and the content in a new directory, and returns its path.
func pluginFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// pickerThatChooses sets the picker of the window to one that answers as the user does, and
// returns a channel that is closed when the picker has been shown.
func pickerThatChooses(root *Root, path string, err error) <-chan struct{} {
	shown := make(chan struct{})
	root.pickFile = func(context.Context) (string, error) {
		close(shown)
		return path, err
	}
	return shown
}

// pluginOpenAddDialog opens the dialog of "Add from source" on the plugin tab.
func pluginOpenAddDialog(t *testing.T, tt *ui.Tester) {
	t.Helper()
	if err := tt.Click("Add from source"); err != nil {
		t.Fatalf("Click(Add from source): %v", err)
	}
	availWaitText(t, tt, "Choose a file")
}

func TestChoosingAFileUploadsItWithTheAlias(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	path := pluginFile(t, "demo.lua", "-- a plugin\n")
	pickerThatChooses(root, path, nil)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)
	pluginOpenAddDialog(t, tt)

	if err := tt.Click("Alias (optional)"); err != nil {
		t.Fatalf("Click(Alias): %v", err)
	}
	tt.Type("xx")
	if err := tt.Click("Choose a file"); err != nil {
		t.Fatalf("Click(Choose a file): %v", err)
	}
	calls := availWaitCalls(t, tt, f)
	got := calls[0]
	if got.Method != "POST" || got.Path != "/api/plugins" {
		t.Fatalf("choosing a file sent %+v, want a POST to /api/plugins", got)
	}
	// The form is the one that the web sends: the file under its name, and the alias that was typed.
	for _, want := range []string{`name="file"; filename="demo.lua"`, "-- a plugin", `name="alias"`, "\r\n\r\nxx\r\n"} {
		if !strings.Contains(got.Body, want) {
			t.Errorf("the form lacks %q:\n%s", want, got.Body)
		}
	}
	// As in the web, the plugin is added and the tasks tab follows; the file of the user stays.
	availWaitTab(t, tt, root)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("the file that the user chose is gone: %v", err)
	}
}

func TestGoingBackFromThePickerAddsNothing(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	shown := pickerThatChooses(root, "", nil)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)
	pluginOpenAddDialog(t, tt)

	if err := tt.Click("Choose a file"); err != nil {
		t.Fatalf("Click(Choose a file): %v", err)
	}
	select {
	case <-shown:
	case <-time.After(5 * time.Second):
		t.Fatal("the picker was not shown")
	}
	// Nothing follows the picker: the frames that are drawn after it show the same tab.
	for range 20 {
		tt.Frame()
		time.Sleep(5 * time.Millisecond)
	}
	if calls := f.recorded(); len(calls) != 0 {
		t.Errorf("a picker that chose nothing sent %+v", calls)
	}
	if root.App().Tab() != app.TabPlugins {
		t.Errorf("the window shows tab %v after a picker that chose nothing, want the plugins", root.App().Tab())
	}
	// Going back is not a failure of any kind: no message of an error is shown.
	for kind := app.KindNone + 1; kind <= app.KindOther; kind++ {
		if text := noticeText(kind); text != "" && hasText(tt, text) {
			t.Errorf("a picker that chose nothing is shown as an error: %q", text)
		}
	}
}

func TestAFileThatIsNotAPluginIsRefusedAtOnce(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	pickerThatChooses(root, pluginFile(t, "notes.txt", "not a plugin"), nil)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)
	pluginOpenAddDialog(t, tt)

	if err := tt.Click("Choose a file"); err != nil {
		t.Fatalf("Click(Choose a file): %v", err)
	}
	availWaitText(t, tt, noticeText(app.KindNotPluginFile))
	if calls := f.recorded(); len(calls) != 0 {
		t.Errorf("a file that is not a plugin was sent: %+v", calls)
	}
	if root.App().Tab() != app.TabPlugins {
		t.Errorf("the window shows tab %v after a refused file, want the plugins", root.App().Tab())
	}
	availShot(t, tt, "add-plugin-refused")
}

func TestAPickerThatFailsSaysSo(t *testing.T) {
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	pickerThatChooses(root, "", errors.New("the system picker failed"))
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)
	pluginOpenAddDialog(t, tt)

	if err := tt.Click("Choose a file"); err != nil {
		t.Fatalf("Click(Choose a file): %v", err)
	}
	availWaitText(t, tt, noticeText(app.KindOther))
	if calls := f.recorded(); len(calls) != 0 {
		t.Errorf("a picker that failed sent %+v", calls)
	}
}

// TestThePickersCopyIsRemovedAfterTheUpload checks the whole way of a pick on Linux, where the
// cache of the app is the XDG_CACHE_HOME that the test sets: the copy that the picker made is
// gone when the upload ends. The other systems keep their caches elsewhere; the rule is
// tested for all of them in TestForgetPickedRemovesOnlyThePickersCopy.
func TestThePickersCopyIsRemovedAfterTheUpload(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("the cache of the app is found from XDG_CACHE_HOME on Linux only")
	}
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	copyDir := filepath.Join(cache, "picked", "1700000000000000000")
	if err := os.MkdirAll(copyDir, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(copyDir, "demo.zip")
	if err := os.WriteFile(path, []byte("PK"), 0o600); err != nil {
		t.Fatal(err)
	}
	f := availNewComputer(t)
	f.inventory = pluginRegistry()
	root := availRoot(t, f)
	pickerThatChooses(root, path, nil)
	tt := ui.NewTester(root.View, 480, 900)
	pluginOpenTab(t, root, tt)
	pluginOpenAddDialog(t, tt)

	if err := tt.Click("Choose a file"); err != nil {
		t.Fatalf("Click(Choose a file): %v", err)
	}
	availWaitCalls(t, tt, f)
	availWaitTab(t, tt, root)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err := os.Stat(copyDir); errors.Is(err, os.ErrNotExist) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the copy that the picker made is still in the cache after the upload")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestForgetPickedRemovesOnlyThePickersCopy(t *testing.T) {
	cache := t.TempDir()
	file := func(parts ...string) string {
		t.Helper()
		path := filepath.Join(parts...)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	exists := func(path string) bool {
		_, err := os.Stat(path)
		return err == nil
	}
	pick := file(cache, "picked", "1700000000000000000", "demo.zip")
	// What is not the file of one pick stays: the file of the user, a file of the cache that
	// the picker did not make, a pick that is deeper, a directory that is not a number, and a path
	// that leaves the directory of the picks.
	users := file(t.TempDir(), "demo.zip")
	other := file(cache, "other", "5", "demo.zip")
	deeper := file(cache, "picked", "7", "sub", "demo.zip")
	named := file(cache, "picked", "keep", "demo.zip")
	outside := file(cache, "outside.zip")
	for _, path := range []string{users, other, deeper, named, outside, filepath.Join(cache, "picked", "..", "outside.zip"), "", "relative.zip"} {
		forgetPickedIn(cache, path)
	}
	for _, path := range []string{users, other, deeper, named, outside} {
		if !exists(path) {
			t.Errorf("%s was removed, and it is not the copy of a pick", path)
		}
	}
	if !exists(pick) {
		t.Fatal("the copy of the pick was removed by a path that is not it")
	}
	forgetPickedIn(cache, pick)
	if exists(filepath.Dir(pick)) {
		t.Error("the directory of the pick is still there")
	}
	if !exists(filepath.Join(cache, "picked")) {
		t.Error("the directory of the picks was removed with the pick")
	}
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
