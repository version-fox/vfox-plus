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

package ui

import (
	"os"
	"strings"
	"testing"

	mui "github.com/egoist/mygo/ui"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/i18n"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

func (ta *testApp) item(name string) { ta.t.Helper(); ta.click(kit.RoleListItem, name) }

// buttonNear clicks the button in the list item that is called row.
func (ta *testApp) buttonNear(row, name string) {
	ta.t.Helper()
	for _, n := range ta.nodes() {
		if n.Role == kit.RoleButton && n.Name == name && n.Scope == row {
			n.Click()
			ta.settle()
			return
		}
	}
	ta.t.Fatalf("no button %q next to %q: %q", name, row, ta.texts())
}

// fieldIn finds the text field of the list item with the name.
func (ta *testApp) fieldIn(name string) (*kit.Node, bool) {
	for _, n := range ta.nodes() {
		if n.Role == kit.RoleTextField && n.Scope == name {
			return n, true
		}
	}
	return nil, false
}

// fill puts text into a field and lets the interface react.
func (ta *testApp) fill(n *kit.Node, text string, submit bool) {
	ta.t.Helper()
	if n == nil || n.Fill == nil {
		ta.t.Fatalf("nothing to fill in (%q)", ta.texts())
	}
	n.Fill(text, submit)
	ta.settle()
}

func TestHomeShowsTheOverviewAndTheShortcuts(t *testing.T) {
	ta := defaultApp(t)
	for _, text := range []string{
		"vfox+", "Versions in use", "Plugins", "Installed versions", "In use", "maven", "nodejs",
		"Use the versions in a terminal", `eval "$(vfox activate bash)"`, "Add more SDKs", "vfox documentation",
	} {
		ta.mustText(text)
	}
	if !ta.text("vfox 1.0.13 · /usr/bin/vfox") {
		t.Errorf("the subtitle names the version and the executable: %q", ta.texts())
	}
	// The tiles count what vfox lists: three plugins, four installed versions.
	for _, count := range []string{"3", "4", "2"} {
		ta.mustText(count)
	}
	ta.button("Copy")
	if len(ta.desktop.copied) != 1 || ta.desktop.copied[0] != `eval "$(vfox activate bash)"` {
		t.Errorf("the activation command is copied, got %q", ta.desktop.copied)
	}
	ta.mustText("Copied to the clipboard")
	ta.snapshot("home")
	ta.button("Add more SDKs")
	ta.mustText("Every plugin adds a language or tool to vfox.")
	ta.button("Home")
	ta.button("vfox documentation")
	if len(ta.desktop.opened) != 1 || ta.desktop.opened[0] != "https://vfox.dev/" {
		t.Errorf("got %q", ta.desktop.opened)
	}
	ta.button("Refresh")
	ta.until("the SDKs to be listed again", func() bool { return len(ta.runner.Calls("list")) >= 2 })
}

// The tester draws in memory, as a window does where no GPU works, and the transitions
// that scale then only fade (kit.Env.Software).
func TestAWindowDrawnByTheCPUTellsItsTransitionsToOnlyFade(t *testing.T) {
	ta := defaultApp(t)
	if !ta.app.env.Software {
		t.Error("the painter says that nothing accelerates the frame, and the environment follows it")
	}
}

func TestTheRailNavigatesBetweenThePagesAndTheBadgeCountsTasks(t *testing.T) {
	ta := defaultApp(t)
	for destination, title := range map[string]string{"SDKs": "Select an SDK to manage its versions.", "Tasks": "Nothing is running", "Settings": "Appearance"} {
		ta.button(destination)
		ta.mustText(title)
	}
	ta.button("Plugins")
	ta.until("the registry", func() bool { return ta.app.model.Registry.HasValue() })
	ta.settle()
	ta.mustText("Every plugin adds a language or tool to vfox.")
	ta.button("Home")
	ta.mustText("Versions in use")
	// The action button of the rail leads to the plugins, and the menu button
	// shows the labels.
	ta.button("Add plugin")
	ta.mustText("Every plugin adds a language or tool to vfox.")
	ta.button("Show labels")
	if !ta.app.model.Settings.Settings().RailExpanded {
		t.Error("the rail is expanded")
	}
	ta.snapshot("rail-expanded")
	ta.button("Hide labels")
	if ta.app.model.Settings.Settings().RailExpanded {
		t.Error("the rail is collapsed again")
	}
}

func TestTheSdkPageSearchesAndShowsTheVersionsOfThePluginThatWasChosen(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	for _, text := range []string{"maven", "mvn2", "nodejs", "In use: 24.11.1\n2 versions installed", "No version in use\nNo version installed"} {
		ta.mustText(text)
	}
	ta.snapshot("sdks-list")
	ta.item("nodejs")
	ta.until("the plugin information", func() bool { _, ok := ta.app.detailFor("nodejs").controller.Info.Value(); return ok })
	ta.settle()
	for _, text := range []string{"Node.js runtime environment.", "Installed (2)", "24.11.1", "22.21.0", "In use", "Plugin 0.5.1"} {
		ta.mustText(text)
	}
	ta.snapshot("sdks")
	// The search narrows the list of plugins.
	ta.app.pages.sdksQuery = "mvn"
	ta.settle()
	if ta.text("In use: 24.11.1\n2 versions installed") && ta.app.shell.selectedSdk == "" {
		t.Error("the search hides the other plugins")
	}
	ta.app.pages.sdksQuery = "zzz"
	ta.settle()
	ta.mustText("No plugin matches “zzz”.")
	ta.app.pages.sdksQuery = ""
	ta.settle()
	// Using another version.
	ta.button("Use")
	ta.until("the use task", func() bool { return len(ta.app.model.Tasks.Tasks()) == 1 })
	if got := ta.app.model.Tasks.Tasks()[0].Command.Operation; got != vfox.OperationUse {
		t.Errorf("got %v", got)
	}
	ta.until("the task to finish", func() bool { return ta.app.model.Tasks.ActiveCount() == 0 })
	if got := ta.app.model.Tasks.Tasks()[0].Status(); got != state.TaskSucceeded {
		t.Errorf("got %v", got)
	}
}

func TestTheVersionsThatCanBeInstalledAreFilteredAndInstalled(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("nodejs")
	ta.button("Available")
	ta.until("the versions", func() bool { return ta.app.detailFor("nodejs").controller.Versions.HasValue() })
	ta.settle()
	ta.mustText("872 versions")
	ta.snapshot("available")
	d := ta.app.detailFor("nodejs")
	d.controller.SetQuery("24.11.0")
	ta.settle()
	ta.mustText("1 version")
	ta.button("Installed")
	if d.controller.Filter() != state.VersionFilterInstalled {
		t.Errorf("got %v", d.controller.Filter())
	}
	d.controller.SetFilter(state.VersionFilterAll)
	ta.settle()
	node, ok := ta.find(kit.RoleButton, "Install")
	if !ok {
		t.Fatalf("no Install button: %q", ta.texts())
	}
	node.Click()
	ta.until("the installation", func() bool { return len(ta.runner.Calls("install --yes nodejs@24.11.0")) == 1 })
	ta.until("the task to finish", func() bool { return ta.app.model.Tasks.ActiveCount() == 0 })
	ta.settle()
}

func TestTheFilterFieldOfTheVersionsTakesTheTextTheUserTypes(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("nodejs")
	ta.button("Available")
	ta.until("the versions", func() bool { return ta.app.detailFor("nodejs").controller.Versions.HasValue() })
	ta.settle()
	field, ok := ta.find(kit.RoleTextField, "Filter versions")
	if !ok {
		t.Fatalf("no filter field: %q", ta.texts())
	}
	ta.fill(field, "22.21.0", false)
	ta.mustText("1 version")
	if got := ta.app.detailFor("nodejs").controller.Query(); got != "22.21.0" {
		t.Errorf("got %q", got)
	}
	// The filter that the user typed in stays while the page is built again.
	ta.settle()
	if field, _ = ta.find(kit.RoleTextField, "Filter versions"); field.Value() != "22.21.0" {
		t.Errorf("the field holds %q", field.Value())
	}
}

func TestRemovingAPluginAsksForConfirmationFirst(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("maven")
	ta.settle()
	ta.click(kit.RoleButton, "More actions")
	ta.click(kit.RoleMenuItem, "Remove plugin")
	ta.mustText("Remove maven?")
	ta.mustText("The plugin and its 2 installed versions will be deleted.")
	ta.snapshot("dialog-remove")
	if n := len(ta.runner.Calls("remove")); n != 0 {
		t.Fatalf("nothing is removed before the user confirms: %d", n)
	}
	ta.button("Cancel")
	ta.settle()
	if ta.app.env.HasDialog() {
		t.Error("the dialog closes")
	}
	ta.click(kit.RoleButton, "More actions")
	ta.click(kit.RoleMenuItem, "Remove plugin")
	ta.button("Remove")
	ta.until("the removal", func() bool { return len(ta.runner.Calls("remove --yes maven")) == 1 })
	if ta.app.shell.selectedSdk != "" {
		t.Error("the page leaves the plugin that is removed")
	}
}

func TestVersionsAreUninstalledAfterAConfirmation(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("nodejs")
	ta.settle()
	ta.click(kit.RoleButton, "More actions for 22.21.0")
	ta.click(kit.RoleMenuItem, "Uninstall")
	ta.mustText("Uninstall nodejs 22.21.0?")
	ta.mustText("The files of this version will be deleted.")
	ta.button("Uninstall")
	ta.until("the uninstallation", func() bool { return len(ta.runner.Calls("uninstall")) == 1 })
	// The version that is in use warns that vfox switches to another.
	ta.click(kit.RoleButton, "More actions for 24.11.1")
	ta.click(kit.RoleMenuItem, "Uninstall")
	ta.mustText("This version is in use. After it is deleted, vfox switches to another installed version, if there is one.")
	ta.button("Cancel")
	ta.settle()
}

func TestUsingAVersionInAProjectAsksForTheDirectory(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("nodejs")
	ta.settle()
	ta.click(kit.RoleButton, "More ways to use this version")
	ta.click(kit.RoleMenuItem, "Use in a project…")
	ta.mustText("Use nodejs 22.21.0 in a project")
	field, ok := ta.find(kit.RoleTextField, "Project directory")
	if !ok {
		t.Fatalf("no field: %q", ta.texts())
	}
	ta.fill(field, "/does/not/exist", true)
	ta.mustText("Choose an existing directory.")
	directory := t.TempDir()
	field, _ = ta.find(kit.RoleTextField, "Project directory")
	ta.fill(field, directory, true)
	ta.until("vfox to be told", func() bool { return len(ta.runner.Calls("use")) >= 1 })
	for _, call := range ta.runner.Calls("use") {
		if call.WorkingDirectory == directory && strings.Contains(strings.Join(call.Arguments, " "), "--project") {
			return
		}
	}
	t.Errorf("the version is used in the project: %v", ta.runner.Calls("use"))
}

func TestTheProjectDirectoryCanBeChosenWithTheChooser(t *testing.T) {
	ta := defaultApp(t)
	ta.desktop.chosen = t.TempDir()
	ta.button("SDKs")
	ta.item("nodejs")
	ta.settle()
	ta.click(kit.RoleButton, "More ways to use this version")
	ta.click(kit.RoleMenuItem, "Use in a project…")
	ta.button("Browse…")
	ta.until("the chooser", func() bool { return len(ta.desktop.asked) == 1 })
	if options := ta.desktop.asked[0]; !options.Directory {
		t.Errorf("the chooser asks for a directory: %+v", options)
	}
	ta.until("the directory in the field", func() bool {
		field, ok := ta.find(kit.RoleTextField, "Project directory")
		return ok && field.Value() == ta.desktop.chosen
	})
}

func TestThePluginRegistryListsAndAddsPlugins(t *testing.T) {
	ta := defaultApp(t)
	ta.button("Plugins")
	ta.until("the registry", func() bool { return ta.app.model.Registry.HasValue() })
	ta.settle()
	ta.mustText("dotnet")
	ta.mustText("version-fox/vfox-dotnet")
	ta.snapshot("plugins")
	// The filter and the search narrow the grid.
	ta.button("Community")
	if ta.app.pages.pluginsFilter != filterCommunity {
		t.Errorf("got %q", ta.app.pages.pluginsFilter)
	}
	ta.app.pages.pluginsFilter = filterAll
	ta.app.pages.pluginsQuery = "nothing like this"
	ta.settle()
	ta.mustText("No plugin matches your search.")
	ta.app.pages.pluginsQuery = ""
	ta.settle()
	add, ok := ta.find(kit.RoleButton, "Add")
	if !ok {
		t.Fatalf("no Add button: %q", ta.texts())
	}
	add.Click()
	ta.until("the plugin to be added", func() bool { return len(ta.runner.Calls("add ")) >= 1 })
	ta.settle()
	// Adding from a source takes a dialog, which refuses an empty source and a
	// name that is not valid.
	ta.button("Add from source…")
	ta.mustText("Add a plugin from source")
	ta.snapshot("dialog-source")
	ta.button("Add plugin")
	ta.mustText("Enter a URL or a path.")
	source, _ := ta.find(kit.RoleTextField, "Source")
	ta.fill(source, "https://example.com/p.zip", false)
	alias, _ := ta.find(kit.RoleTextField, "Name (optional)")
	ta.fill(alias, "-bad name", false)
	ta.button("Add plugin")
	ta.mustText("Use letters, digits, dots, dashes and underscores only.")
	alias, _ = ta.find(kit.RoleTextField, "Name (optional)")
	ta.fill(alias, "mine", false)
	ta.button("Add plugin")
	ta.until("the plugin to be added from the source", func() bool {
		for _, call := range ta.runner.Calls("add --source https://example.com/p.zip") {
			if strings.HasSuffix(strings.Join(call.Arguments, " "), "--alias mine") {
				return true
			}
		}
		return false
	})
	ta.button("Update all plugins")
	ta.until("the update", func() bool { return len(ta.runner.Calls("update --all")) == 1 })
}

func TestTasksShowTheirStateAndOutput(t *testing.T) {
	runner := testsupport.FixtureRunner()
	process := testsupport.NewFakeRunningProcess()
	runner.OnStart("install --yes maven@3.9.16", func() *testsupport.FakeRunningProcess { return process })
	runner.On("update java", core.ProcessOutput{ExitCode: 1, Stderr: "java plugin not installed\n"})
	ta := newTestApp(t, runner, windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.start()
	command, err := ta.app.model.Cli.Install("maven", "3.9.16")
	if err != nil {
		t.Fatal(err)
	}
	running := ta.app.model.Run(command)
	update, err := ta.app.model.Cli.UpdatePlugin("java")
	if err != nil {
		t.Fatal(err)
	}
	queued := ta.app.model.Run(update)
	ta.button("Tasks")
	ta.mustText("Install maven@3.9.16")
	ta.mustText("Update plugin java")
	ta.mustText("Waiting")
	process.WriteStderr("\rDownloading...  42% [===>  ]\r")
	process.WriteStdout("Preinstalling maven@3.9.16...\n")
	ta.until("the progress", func() bool { progress, ok := running.Progress(); return ok && progress == 0.42 })
	ta.settle()
	ta.mustText("Downloading… 42%")
	ta.buttonNear("Install maven@3.9.16", "Show output")
	ta.mustText("$ vfox install --yes maven@3.9.16\nPreinstalling maven@3.9.16...")
	ta.snapshot("tasks")
	ta.button("Copy output")
	if len(ta.desktop.copied) != 1 || !strings.Contains(ta.desktop.copied[0], "Preinstalling") {
		t.Errorf("got %q", ta.desktop.copied)
	}
	process.Exit(0)
	ta.until("the tasks", func() bool { return ta.app.model.Tasks.ActiveCount() == 0 })
	ta.settle()
	ta.mustText("Done")
	if queued.Status() != state.TaskFailed {
		t.Fatalf("got %v", queued.Status())
	}
	ta.mustText("Failed · java plugin not installed")
	// A failure can be tried again, and finished tasks cleared.
	ta.button("Retry")
	ta.until("the retry", func() bool { return len(ta.runner.Calls("update java")) == 2 })
	ta.until("the retry to finish", func() bool { return ta.app.model.Tasks.ActiveCount() == 0 })
	ta.button("Clear finished")
	ta.settle()
	ta.mustText("Nothing is running")
}

func TestSnackbarsAnnounceFinishedTasksAndFailuresStay(t *testing.T) {
	runner := testsupport.FixtureRunner()
	runner.On("update java", core.ProcessOutput{ExitCode: 1, Stderr: "java plugin not installed\n"})
	ta := newTestApp(t, runner, windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.start()
	update, _ := ta.app.model.Cli.UpdatePlugin("java")
	ta.app.model.Run(update)
	ta.until("the announcement", func() bool {
		ta.app.runPending()
		return ta.text("Update plugin java: failed\njava plugin not installed")
	})
	ta.settle()
	ta.snapshot("snackbar")
	ta.button("View")
	if ta.app.shell.destination != destTasks {
		t.Error("the action of the snack bar leads to the tasks")
	}
	ta.mustText("Failed · java plugin not installed")
}

func TestSettingsChangeTheAppearanceTheLanguageAndTheEnvironment(t *testing.T) {
	store := settings.NewMemorySettingsStore(settings.DefaultSettings())
	ta := newTestApp(t, testsupport.FixtureRunner(), windowWidth, windowHeight, store)
	ta.start()
	ta.button("Settings")
	ta.until("the vfox configuration", func() bool { return ta.app.model.VfoxConfig.HasValue() })
	ta.settle()
	for _, text := range []string{"Appearance", "vfox", "vfox configuration", "Plugin registry", "About", "Source code", "Open-source licenses"} {
		ta.mustText(text)
	}
	ta.snapshot("settings")
	ta.button("Dark")
	if got := ta.app.model.Settings.Settings().Theme; got != settings.ThemeDark {
		t.Errorf("got %v", got)
	}
	if !ta.app.env.Theme.Dark {
		t.Error("the colours follow the theme")
	}
	ta.snapshot("settings-dark")
	ta.button("Accent color 3")
	current := ta.app.model.Settings.Settings()
	if current.UseSystemColor || current.SeedColor != seedColors[2] {
		t.Errorf("a colour was chosen: %#v", current)
	}
	ta.click(kit.RoleSwitch, "Use the desktop accent color")
	if !ta.app.model.Settings.Settings().UseSystemColor {
		t.Error("the switch turns the desktop colour back on")
	}
	ta.button("简体中文")
	ta.mustText("外观")
	ta.snapshot("settings-zh")
	ta.button("English")
	ta.mustText("Appearance")
	// Environment variables are validated before they are stored.
	field, _ := ta.find(kit.RoleTextField, "VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node")
	ta.fill(field, "A=1\nnot valid\nB=2", false)
	ta.button("Save")
	ta.mustText("Line 2 is not KEY=VALUE.")
	field, _ = ta.find(kit.RoleTextField, "VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node")
	ta.fill(field, "A=1\nB=2", false)
	ta.button("Save")
	if got := ta.app.model.Settings.Settings().ExtraEnvironment; got["A"] != "1" || got["B"] != "2" {
		t.Errorf("got %v", got)
	}
	ta.mustText("Saved")
	// The configuration of vfox is edited key by key.
	registry, ok := ta.fieldIn("Plugin registry")
	if !ok {
		t.Fatalf("no field for the registry: %q", ta.texts())
	}
	ta.fill(registry, "https://registry.example/", true)
	ta.until("the key to be set", func() bool { return len(ta.runner.Calls("config registry.address https://registry.example/")) == 1 })
	ta.click(kit.RoleSwitch, "Download through a proxy")
	ta.until("the switch to be saved", func() bool { return len(ta.runner.Calls("config proxy.enable")) >= 1 })
	// The licenses open in a dialog.
	ta.item("Open-source licenses")
	ta.mustText("Open-source licenses")
	ta.button("Close")
	ta.settle()
	ta.item("Source code")
	if len(ta.desktop.opened) != 1 || ta.desktop.opened[0] != linkRepository {
		t.Errorf("got %q", ta.desktop.opened)
	}
}

func TestTheVfoxExecutableCanBeChosenAndDetectedAgain(t *testing.T) {
	store := settings.NewMemorySettingsStore(settings.DefaultSettings())
	ta := newTestApp(t, testsupport.FixtureRunner(), windowWidth, windowHeight, store)
	ta.desktop.chosen = "/usr/local/bin/vfox"
	ta.start()
	ta.button("Settings")
	ta.settle()
	ta.button("Browse…")
	ta.until("the executable", func() bool {
		return ta.app.model.Settings.Settings().VfoxExecutable == "/usr/local/bin/vfox"
	})
	ta.settle()
	ta.button("Detect automatically")
	ta.until("the choice to be forgotten", func() bool { return ta.app.model.Settings.Settings().VfoxExecutable == "" })
}

func TestCompactWindowsGetTheBottomBarAndOnePaneAtATime(t *testing.T) {
	ta := newTestApp(t, testsupport.FixtureRunner(), 412, 800, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.start()
	ta.settle()
	ta.mustText("Versions in use")
	ta.snapshot("compact-home")
	ta.button("SDKs")
	ta.mustText("nodejs")
	ta.item("nodejs")
	ta.settle()
	ta.mustText("Node.js runtime environment.")
	ta.snapshot("compact-detail")
	// The back button leads to the list again.
	ta.button("Close")
	ta.mustText("In use: 24.11.1\n2 versions installed")
	if !ta.app.env.Compact {
		t.Error("snack bars stay above the navigation bar")
	}
	// A wide window has the rail and two panes.
	ta.tester.SetSize(1400, windowHeight)
	ta.settle()
	if ta.app.env.Compact {
		t.Error("wide windows have a rail")
	}
}

func TestUnavailableVfoxShowsHowToFixIt(t *testing.T) {
	runner := testsupport.FixtureRunner()
	runner.SetMissingExecutable(true)
	ta := newTestApp(t, runner, windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.app.model.Start()
	ta.until("vfox to be missing", func() bool { return ta.app.model.Vfox.Availability() == state.VfoxNotFound })
	ta.settle()
	ta.mustText("vfox is not installed")
	ta.snapshot("vfox-missing")
	ta.button("Installation guide")
	if len(ta.desktop.opened) != 1 || ta.desktop.opened[0] != linkVfoxInstall {
		t.Errorf("got %q", ta.desktop.opened)
	}
	ta.button("Retry")
	// Settings stay reachable, so that the executable can be chosen.
	ta.button("Settings")
	ta.mustText("Appearance")
	runner.SetMissingExecutable(false)
	runner.On("--version", core.ProcessOutput{Stdout: "hello"})
	ta.button("Home")
	ta.button("Retry")
	ta.until("vfox to be broken", func() bool { return ta.app.model.Vfox.Availability() == state.VfoxBroken })
	ta.settle()
	ta.mustText("vfox does not respond")
}

func TestLoadingAndFailedListsOfSdks(t *testing.T) {
	runner := testsupport.FixtureRunner()
	gate := make(chan struct{})
	runner.OnDynamic("list", func() core.ProcessOutput {
		<-gate
		return core.ProcessOutput{ExitCode: 1, Stderr: "boom\n"}
	})
	runner.On("current", core.ProcessOutput{ExitCode: 1, Stderr: "boom\n"})
	ta := newTestApp(t, runner, windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.app.model.Start()
	ta.until("vfox to be ready", func() bool { return ta.app.model.Vfox.Availability() == state.VfoxReady })
	ta.settle()
	if !ta.app.model.Installed.IsLoading() {
		t.Fatal("the SDKs are loading")
	}
	close(gate)
	ta.until("the failure", func() bool { return ta.app.model.Installed.Error() != nil })
	ta.settle()
	ta.mustText("Could not list the installed SDKs.")
	ta.button("SDKs")
	ta.mustText("Could not list the installed SDKs.")
	ta.button("Retry")
}

func TestTheLanguageFollowsTheDesktopLocale(t *testing.T) {
	store := settings.NewMemorySettingsStore(settings.DefaultSettings())
	ta := newTestApp(t, testsupport.FixtureRunner(), windowWidth, windowHeight, store)
	ta.app.locales = []string{"zh_CN.UTF-8"}
	ta.start()
	ta.settle()
	ta.mustText("正在使用的版本")
	if got := ta.app.T("homeStatPlugins", "count", 1); got != "插件" {
		t.Errorf("got %q", got)
	}
}

// Windows sets none of the variables of the locale: mygo tells the display languages
// of the user once the application is ready, and they come before the environment.
func TestTheLanguageFollowsTheDisplayLanguagesOfTheSystem(t *testing.T) {
	cases := map[string]struct {
		preference  settings.LanguagePreference
		environment []string
		system      []string
		want        i18n.SupportedLanguage
	}{
		"Chinese Windows, no variable":              {settings.LanguageSystem, nil, []string{"zh-Hans-CN", "en-US"}, i18n.SimplifiedChinese},
		"Traditional Chinese has the one catalogue": {settings.LanguageSystem, nil, []string{"zh-Hant-TW"}, i18n.SimplifiedChinese},
		"English Windows":                           {settings.LanguageSystem, nil, []string{"en-US"}, i18n.English},
		"the first language that has a catalogue":   {settings.LanguageSystem, nil, []string{"de-DE", "zh-Hans-CN", "en-US"}, i18n.SimplifiedChinese},
		"none of them has one":                      {settings.LanguageSystem, nil, []string{"de-DE", "fr-FR"}, i18n.English},
		"the system lists none":                     {settings.LanguageSystem, []string{"zh_CN.UTF-8"}, nil, i18n.SimplifiedChinese},
		"the system comes first":                    {settings.LanguageSystem, []string{"en_US.UTF-8"}, []string{"zh-Hans-CN"}, i18n.SimplifiedChinese},
		"a language of the settings wins":           {settings.LanguageEnglish, nil, []string{"zh-Hans-CN"}, i18n.English},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			defaults := settings.DefaultSettings()
			defaults.Language = c.preference
			ta := newTestApp(t, testsupport.FixtureRunner(), windowWidth, windowHeight, settings.NewMemorySettingsStore(defaults))
			ta.app.locales = c.environment
			ta.app.useSystemLanguages(c.system)
			// The settings of the user are read when the model starts.
			ta.start()
			ta.settle()
			if got := ta.app.l10n.Load().Language; got != c.want {
				t.Fatalf("the language is %q, want %q", got, c.want)
			}
			if c.want == i18n.SimplifiedChinese {
				ta.mustText("正在使用的版本")
			} else {
				ta.mustText("Versions in use")
			}
		})
	}
}

func TestTheAccentColourOfTheDesktopIsTheSeedOfTheScheme(t *testing.T) {
	store := settings.NewMemorySettingsStore(settings.DefaultSettings())
	ta := newTestApp(t, testsupport.FixtureRunner(), windowWidth, windowHeight, store)
	ta.start()
	before := ta.app.env.Theme.Colors.Primary
	ta.tester.SetPreferences(mui.Preferences{Accent: mui.RGB(0x1B, 0x8C, 0xF2), TextScale: 1.25})
	ta.settle()
	if ta.app.env.Theme.Colors.Primary == before {
		t.Error("the scheme follows the accent colour of the desktop")
	}
	if ta.app.env.Theme.TextScale != 1.25 {
		t.Errorf("the text follows the scale of the desktop: %v", ta.app.env.Theme.TextScale)
	}
	ta.tester.SetDark(true)
	ta.settle()
	if !ta.app.env.Theme.Dark {
		t.Error("the scheme follows the dark appearance of the desktop")
	}
}

func TestTheVersionOfTheApplicationIsTheOneInTheVersionFile(t *testing.T) {
	content, err := os.ReadFile("../VERSION")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(content)); got != Version {
		t.Errorf("VERSION says %q, the application %q", got, Version)
	}
}
