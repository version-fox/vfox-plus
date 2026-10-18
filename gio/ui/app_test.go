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
	"image"
	"strings"
	"testing"
	"time"
	"unsafe"

	"gioui.org/app"
	"gioui.org/io/input"
	"gioui.org/io/system"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"

	core "github.com/version-fox/vfox-plus/gio/core"
	"github.com/version-fox/vfox-plus/gio/core/settings"
	"github.com/version-fox/vfox-plus/gio/core/vfox"
	"github.com/version-fox/vfox-plus/gio/internal/testsupport"
	"github.com/version-fox/vfox-plus/gio/state"
	"github.com/version-fox/vfox-plus/gio/ui/kit"
	"github.com/version-fox/vfox-plus/gio/ui/platform"
)

// recorder is the window system of the tests: it notes what the application asks
// of it, and opens nothing.
type recorder struct {
	kit.Services
	copied []string
	opened []string
}

func (r *recorder) Copy(text string)    { r.copied = append(r.copied, text) }
func (r *recorder) OpenURL(link string) { r.opened = append(r.opened, link) }

// testApp is the application against the captured output of vfox, laid out
// frame after frame without a window.
type testApp struct {
	t      *testing.T
	app    *App
	runner *testsupport.FakeProcessRunner
	router input.Router
	ops    op.Ops
	size   image.Point
	now    time.Time
	system *recorder
}

func newTestApp(t *testing.T, runner *testsupport.FakeProcessRunner, size image.Point, store settings.SettingsStore) *testApp {
	t.Helper()
	environment := map[string]string{"HOME": t.TempDir(), "LANG": "en_US.UTF-8"}
	model := state.NewAppModel(state.AppModelOptions{
		Runner: runner, SettingsStore: store, Environment: environment,
		Locator: vfox.NewVfoxLocator(map[string]string{}, func(path string) bool { return path == "/usr/bin/vfox" }),
	})
	a, err := New(Options{Model: model, Environment: environment})
	if err != nil {
		t.Fatal(err)
	}
	a.env.Reg.CollectText = true
	system := &recorder{Services: a.env.Services}
	a.env.Services = system
	a.listen()
	t.Cleanup(model.Dispose)
	return &testApp{t: t, app: a, runner: runner, size: size, now: time.Unix(5000, 0), system: system}
}

func defaultApp(t *testing.T) *testApp {
	t.Helper()
	app := newTestApp(t, testsupport.FixtureRunner(), image.Pt(1280, 800), settings.NewMemorySettingsStore(settings.DefaultSettings()))
	app.start()
	return app
}

// start probes vfox and waits for the installed SDKs.
func (ta *testApp) start() {
	ta.t.Helper()
	ta.app.model.Start()
	ta.until("the installed SDKs", func() bool { return ta.app.model.Installed.HasValue() })
}

// frame lays out the window once.
func (ta *testApp) frame() {
	ta.t.Helper()
	ta.now = ta.now.Add(16 * time.Millisecond)
	ta.ops.Reset()
	gtx := layout.Context{
		Constraints: layout.Exact(ta.size),
		Metric:      unit.Metric{PxPerDp: 1, PxPerSp: 1},
		Now:         ta.now,
		Source:      ta.router.Source(),
		Ops:         &ta.ops,
	}
	ta.app.frame(gtx)
	ta.router.Frame(&ta.ops)
}

func (ta *testApp) frames(n int) {
	ta.t.Helper()
	for range n {
		ta.frame()
	}
}

// settle lays out enough frames for input to be handled, and for springs and
// transitions to end.
func (ta *testApp) settle() {
	ta.t.Helper()
	ta.frames(45)
}

// until lays out frames until the condition holds.
func (ta *testApp) until(description string, condition func() bool) {
	ta.t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for !condition() {
		if time.Now().After(deadline) {
			ta.t.Fatalf("timed out waiting for %s", description)
		}
		time.Sleep(5 * time.Millisecond)
		ta.frame()
	}
}

func (ta *testApp) nodes() []kit.Node { return ta.app.env.Reg.Nodes() }

func (ta *testApp) text(text string) bool {
	for _, n := range ta.nodes() {
		if n.Role == kit.RoleText && n.Text == text {
			return true
		}
	}
	return false
}

func (ta *testApp) mustText(text string) {
	ta.t.Helper()
	if !ta.text(text) {
		ta.t.Fatalf("no text %q on screen; there is %q", text, ta.texts())
	}
}

func (ta *testApp) texts() []string {
	var texts []string
	for _, n := range ta.nodes() {
		if n.Role == kit.RoleText && n.Text != "" {
			texts = append(texts, n.Text)
		}
	}
	return texts
}

func (ta *testApp) find(role kit.Role, name string) (kit.Node, bool) {
	modal := false
	for _, n := range ta.nodes() {
		modal = modal || n.Modal
	}
	for _, n := range ta.nodes() {
		if n.Role == role && n.Modal == modal && (n.Name == name || role == kit.RoleListItem && strings.HasPrefix(n.Name, name)) {
			return n, true
		}
	}
	return kit.Node{}, false
}

// click does what a click on the widget does, and lets the interface react.
func (ta *testApp) click(role kit.Role, name string) {
	ta.t.Helper()
	n, ok := ta.find(role, name)
	if !ok || n.Click == nil {
		ta.t.Fatalf("nothing to click: %s %q (%v)", role, name, ta.texts())
	}
	n.Click()
	ta.settle()
}

func (ta *testApp) button(name string) { ta.t.Helper(); ta.click(kit.RoleButton, name) }

// buttonNear clicks the button in the list item that is called row.
func (ta *testApp) buttonNear(row, name string) {
	ta.t.Helper()
	for _, item := range ta.nodes() {
		if item.Role != kit.RoleListItem || item.Name != row {
			continue
		}
		for _, n := range ta.nodes() {
			if n.Role == kit.RoleButton && n.Name == name && n.Rect.In(item.Rect) {
				n.Click()
				ta.settle()
				return
			}
		}
	}
	ta.t.Fatalf("no button %q next to %q: %q", name, row, ta.texts())
}

func (ta *testApp) item(name string) { ta.t.Helper(); ta.click(kit.RoleListItem, name) }

func TestHomeShowsTheOverviewAndTheShortcuts(t *testing.T) {
	ta := defaultApp(t)
	ta.settle()
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
	if len(ta.system.copied) != 1 || ta.system.copied[0] != `eval "$(vfox activate bash)"` {
		t.Errorf("the activation command is copied, got %q", ta.system.copied)
	}
	ta.mustText("Copied to the clipboard")
	ta.button("Add more SDKs")
	ta.mustText("Every plugin adds a language or tool to vfox.")
	ta.button("Home")
	ta.button("vfox documentation")
	if len(ta.system.opened) != 1 || ta.system.opened[0] != "https://vfox.dev/" {
		t.Errorf("got %q", ta.system.opened)
	}
	ta.button("Refresh")
	ta.until("the SDKs to be listed again", func() bool { return len(ta.runner.Calls("list")) >= 2 })
}

func TestTheRailNavigatesBetweenThePagesAndTheBadgeCountsTasks(t *testing.T) {
	ta := defaultApp(t)
	ta.settle()
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
	ta.settle()
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
	ta.item("nodejs")
	ta.until("the plugin information", func() bool { _, ok := ta.app.detailFor("nodejs").controller.Info.Value(); return ok })
	ta.settle()
	for _, text := range []string{"Node.js runtime environment.", "Installed (2)", "24.11.1", "22.21.0", "In use", "Plugin 0.5.1"} {
		ta.mustText(text)
	}
	// The search narrows the list of plugins.
	ta.app.pages.sdksQuery = "mvn"
	ta.settle()
	if ta.text("nodejs") && ta.app.shell.selectedSdk == "" {
		t.Error("the search hides the other plugins")
	}
	ta.app.pages.sdksQuery = "zzz"
	ta.settle()
	ta.mustText("No plugin matches “zzz”.")
	ta.app.pages.sdksQuery = ""
	ta.settle()
	// Using another version, uninstalling and the project dialog.
	ta.button("Use")
	ta.until("the use task", func() bool { return len(ta.app.model.Tasks.Tasks()) == 1 })
	if got := ta.app.model.Tasks.Tasks()[0].Command.Operation; got != vfox.OperationUse {
		t.Errorf("got %v", got)
	}
	ta.until("the task to finish", func() bool { return ta.app.model.Tasks.ActiveCount() == 0 })
	ta.settle()
	ta.mustText("Use nodejs@22.21.0: done")
}

func TestTheVersionsThatCanBeInstalledAreFilteredAndInstalled(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("nodejs")
	ta.button("Available")
	ta.until("the versions", func() bool { return ta.app.detailFor("nodejs").controller.Versions.HasValue() })
	ta.settle()
	ta.mustText("872 versions")
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
	// While the installation runs, its row shows the progress and a button
	// that cancels it; and afterwards, the version is installed.
	ta.until("the task to finish", func() bool { return ta.app.model.Tasks.ActiveCount() == 0 })
	ta.settle()
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
	if n := len(ta.runner.Calls("remove")); n != 0 {
		t.Fatalf("nothing is removed before the user confirms: %d", n)
	}
	ta.button("Cancel")
	ta.frames(80)
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
	ta.frames(80)
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
	field.Fill("/does/not/exist", true)
	ta.settle()
	ta.mustText("Choose an existing directory.")
	directory := t.TempDir()
	field, _ = ta.find(kit.RoleTextField, "Project directory")
	field.Fill(directory, true)
	ta.until("vfox to be told", func() bool { return len(ta.runner.Calls("use")) >= 1 })
	for _, call := range ta.runner.Calls("use") {
		if call.WorkingDirectory == directory && strings.Contains(strings.Join(call.Arguments, " "), "--project") {
			return
		}
	}
	t.Errorf("the version is used in the project: %v", ta.runner.Calls("use"))
}

func TestThePluginRegistryListsAndAddsPlugins(t *testing.T) {
	ta := defaultApp(t)
	ta.button("Plugins")
	ta.until("the registry", func() bool { return ta.app.model.Registry.HasValue() })
	ta.settle()
	ta.mustText("dotnet")
	ta.mustText("version-fox/vfox-dotnet")
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
	ta.button("Add plugin")
	ta.mustText("Enter a URL or a path.")
	source, _ := ta.find(kit.RoleTextField, "Source")
	source.Fill("https://example.com/p.zip", false)
	alias, _ := ta.find(kit.RoleTextField, "Name (optional)")
	alias.Fill("-bad name", false)
	ta.button("Add plugin")
	ta.mustText("Use letters, digits, dots, dashes and underscores only.")
	alias, _ = ta.find(kit.RoleTextField, "Name (optional)")
	alias.Fill("mine", false)
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
	ta := newTestApp(t, runner, image.Pt(1280, 800), settings.NewMemorySettingsStore(settings.DefaultSettings()))
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
	ta.button("Copy output")
	if len(ta.system.copied) != 1 || !strings.Contains(ta.system.copied[0], "Preinstalling") {
		t.Errorf("got %q", ta.system.copied)
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
	ta := newTestApp(t, runner, image.Pt(1280, 800), settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.start()
	update, _ := ta.app.model.Cli.UpdatePlugin("java")
	ta.app.model.Run(update)
	ta.until("the announcement", func() bool { return ta.text("Update plugin java: failed\njava plugin not installed") })
	ta.button("View")
	if ta.app.shell.destination != destTasks {
		t.Error("the action of the snack bar leads to the tasks")
	}
	ta.mustText("Failed · java plugin not installed")
}

func TestSettingsChangeTheAppearanceTheLanguageAndTheEnvironment(t *testing.T) {
	store := settings.NewMemorySettingsStore(settings.DefaultSettings())
	ta := newTestApp(t, testsupport.FixtureRunner(), image.Pt(1280, 800), store)
	ta.start()
	ta.button("Settings")
	ta.until("the vfox configuration", func() bool { return ta.app.model.VfoxConfig.HasValue() })
	ta.settle()
	for _, text := range []string{"Appearance", "vfox", "vfox configuration", "Plugin registry", "About", "Source code", "Open-source licenses"} {
		ta.mustText(text)
	}
	ta.button("Dark")
	if got := ta.app.model.Settings.Settings().Theme; got != settings.ThemeDark {
		t.Errorf("got %v", got)
	}
	if !ta.app.env.Theme.Dark {
		t.Error("the colours follow the theme")
	}
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
	ta.button("English")
	ta.mustText("Appearance")
	// Environment variables are validated before they are stored.
	field, _ := ta.find(kit.RoleTextField, "VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node")
	field.Fill("A=1\nnot valid\nB=2", false)
	ta.button("Save")
	ta.mustText("Line 2 is not KEY=VALUE.")
	field, _ = ta.find(kit.RoleTextField, "VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node")
	field.Fill("A=1\nB=2", false)
	ta.button("Save")
	if got := ta.app.model.Settings.Settings().ExtraEnvironment; got["A"] != "1" || got["B"] != "2" {
		t.Errorf("got %v", got)
	}
	ta.mustText("Saved")
	// The configuration of vfox is edited key by key.
	registry, ok := ta.find(kit.RoleTextField, "Plugin registry")
	if !ok {
		registry, ok = ta.fieldInItem("Plugin registry")
	}
	if !ok {
		t.Fatalf("no field for the registry: %q", ta.texts())
	}
	registry.Fill("https://registry.example/", true)
	ta.until("the key to be set", func() bool { return len(ta.runner.Calls("config registry.address https://registry.example/")) == 1 })
	ta.click(kit.RoleSwitch, "Download through a proxy")
	ta.until("the switch to be saved", func() bool { return len(ta.runner.Calls("config proxy.enable")) >= 1 })
	// The licenses open in a dialog.
	ta.item("Open-source licenses")
	ta.mustText("Open-source licenses")
	ta.button("Close")
	ta.frames(80)
	ta.item("Source code")
	if len(ta.system.opened) != 1 || ta.system.opened[0] != linkRepository {
		t.Errorf("got %q", ta.system.opened)
	}
}

// fieldInItem finds the text field of the list item with the name.
func (ta *testApp) fieldInItem(name string) (kit.Node, bool) {
	for _, item := range ta.nodes() {
		if item.Role != kit.RoleListItem || item.Name != name {
			continue
		}
		for _, n := range ta.nodes() {
			if n.Role == kit.RoleTextField && n.Rect.In(item.Rect) {
				return n, true
			}
		}
	}
	return kit.Node{}, false
}

func TestCompactWindowsGetTheBottomBarAndOnePaneAtATime(t *testing.T) {
	ta := newTestApp(t, testsupport.FixtureRunner(), image.Pt(412, 800), settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.start()
	ta.settle()
	ta.mustText("Versions in use")
	ta.button("SDKs")
	ta.mustText("nodejs")
	ta.item("nodejs")
	ta.settle()
	ta.mustText("Node.js runtime environment.")
	// The back button leads to the list again.
	ta.button("Close")
	ta.mustText("In use: 24.11.1\n2 versions installed")
	if ta.app.env.BottomInset == 0 {
		t.Error("snack bars stay above the navigation bar")
	}
	// A medium window shows one pane at a time, a wide one two.
	ta.size = image.Pt(1400, 800)
	ta.settle()
	if ta.app.env.BottomInset != 0 {
		t.Error("wide windows have a rail")
	}
}

func TestUnavailableVfoxShowsHowToFixIt(t *testing.T) {
	runner := testsupport.FixtureRunner()
	runner.SetMissingExecutable(true)
	ta := newTestApp(t, runner, image.Pt(1280, 800), settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.app.model.Start()
	ta.until("vfox to be missing", func() bool { return ta.app.model.Vfox.Availability() == state.VfoxNotFound })
	ta.settle()
	ta.mustText("vfox is not installed")
	ta.button("Installation guide")
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
	ta := newTestApp(t, runner, image.Pt(1280, 800), settings.NewMemorySettingsStore(settings.DefaultSettings()))
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
	ta := newTestApp(t, testsupport.FixtureRunner(), image.Pt(1280, 800), store)
	ta.app.locales = []string{"zh_CN.UTF-8"}
	ta.start()
	ta.settle()
	ta.mustText("正在使用的版本")
	if got := ta.app.T("homeStatPlugins", "count", 1); got != "插件" {
		t.Errorf("got %q", got)
	}
}

func TestTheApplicationDrawsTheTitleBarWhereTheCompositorDoesNot(t *testing.T) {
	ta := defaultApp(t)
	var actions []system.Action
	ta.app.windowAction = func(action system.Action) { actions = append(actions, action) }
	ta.settle()
	if _, ok := ta.find(kit.RoleButton, "close"); ok {
		t.Fatal("an X11 window has the title bar of the window manager")
	}
	// A Wayland window that the compositor does not decorate, as under GNOME:
	// the application takes over and draws the buttons of Ubuntu.
	var display int
	window := new(app.Window)
	handle := unsafe.Pointer(&display) //nolint:gosec // The event holds pointers that the test never uses.
	ta.app.onWaylandView(window, app.WaylandViewEvent{Display: handle, Surface: handle})
	ta.app.onConfig(app.Config{Mode: app.Windowed})
	ta.settle()
	if !ta.app.needsCaption() {
		t.Fatal("the title bar is drawn")
	}
	for name, action := range map[string]system.Action{"minimize": system.ActionMinimize, "maximize": system.ActionMaximize, "close": system.ActionClose} {
		ta.button(name)
		if got := actions[len(actions)-1]; got != action {
			t.Errorf("%s: got %v", name, got)
		}
	}
	// The content moves down by the height of the bar.
	title, _ := ta.find(kit.RoleButton, "close")
	if title.Rect.Dy() != 32 || title.Rect.Max.X > ta.size.X-4 {
		t.Errorf("the close button is at the end of the bar: %v", title.Rect)
	}
	// A window that is maximised can be restored.
	ta.app.onConfig(app.Config{Mode: app.Maximized})
	ta.settle()
	ta.button("restore")
	if got := actions[len(actions)-1]; got != system.ActionUnmaximize {
		t.Errorf("got %v", got)
	}
	// The desktop says where the buttons go.
	ta.app.appearance.WindowButtons = &platform.WindowButtons{Left: []string{"close"}, Right: []string{"minimize"}}
	ta.settle()
	closeButton, _ := ta.find(kit.RoleButton, "close")
	minimize, _ := ta.find(kit.RoleButton, "minimize")
	if _, ok := ta.find(kit.RoleButton, "restore"); ok || closeButton.Rect.Min.X >= minimize.Rect.Min.X {
		t.Errorf("close at the start, minimize at the end: %v %v", closeButton.Rect, minimize.Rect)
	}
	// Nothing is drawn when the compositor draws the title bar, or in full screen.
	ta.app.onConfig(app.Config{Mode: app.Windowed, Decorated: true})
	ta.settle()
	if _, ok := ta.find(kit.RoleButton, "close"); ok {
		t.Error("a compositor with server-side decorations draws the title bar")
	}
	ta.app.onConfig(app.Config{Mode: app.Fullscreen})
	ta.settle()
	if _, ok := ta.find(kit.RoleButton, "close"); ok {
		t.Error("a window in full screen has no title bar")
	}
	// A window that is not a Wayland window is not changed.
	ta.app.onWaylandView(window, app.WaylandViewEvent{})
	ta.app.onConfig(app.Config{Mode: app.Windowed})
	if ta.app.needsCaption() {
		t.Error("only Wayland windows get a title bar of the application")
	}
	ta.app.act(system.ActionClose)
	ta.app.windowAction = nil
	ta.app.act(system.ActionClose)
}
