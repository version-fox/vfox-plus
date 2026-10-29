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

//go:build e2e

package ui

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/state"
)

// scenarios are the end-to-end scenarios of the application: fake (against the
// fake vfox of the fixtures), real (against the real vfox), measure (for the
// memory measurement) and screenshots.
var scenarios = map[string]func(*scenario){
	"fake":        fakeScenario,
	"real":        realScenario,
	"measure":     measureScenario,
	"screenshots": screenshotsScenario,
}

// waitForTask waits until the task with the title (in English) has succeeded;
// it fails if the task failed.
func (s *scenario) waitForTask(title string) {
	a := s.app
	s.waitUntil(fmt.Sprintf("the task %q to finish", title), func() bool {
		for _, task := range a.model.Tasks.Tasks() {
			if a.taskTitle(task.Command) != title {
				continue
			}
			if task.Status() == state.TaskFailed {
				message, _ := task.ErrorMessage()
				panic(fmt.Sprintf("%s failed: %s", title, message))
			}
			return task.Status() == state.TaskSucceeded
		}
		return false
	}, 60*time.Second)
}

// vfox runs vfox as the tests see it: in the sandbox, without colours.
func (s *scenario) vfox(args ...string) string {
	executable := s.app.environment["VFOX_PLUS_E2E_VFOX"]
	if executable == "" {
		executable = "vfox"
	}
	command := exec.Command(executable, args...) //nolint:gosec // vfox of the test.
	command.Dir = "/"
	command.Env = append(os.Environ(), "NO_COLOR=1")
	output, _ := command.Output()
	return string(output)
}

// newestTask is the task that was queued last.
func (s *scenario) newestTask() *state.VfoxTask {
	tasks := s.app.model.Tasks.Tasks()
	check(len(tasks) > 0, "there is a task")
	return tasks[0]
}

func fakeScenario(s *scenario) {
	a := s.app
	s.waitForText("Versions in use")
	s.step("home")
	executable, _ := a.model.Vfox.Executable()
	s.waitForText("vfox 1.0.13 · " + executable)
	check(executable == a.environment["VFOX_PLUS_E2E_VFOX"], "the executable from the settings is used")

	s.navigate("SDKs")
	s.clickItem("nodejs")
	s.waitForText("Node.js runtime environment.")
	s.step("detail")

	// Select 22.21.0 as the global default.
	s.clickButton("Use")
	s.waitForTask("Use nodejs@22.21.0")

	// Install a version from the available tab.
	s.clickButton("Available")
	s.waitForText("872 versions")
	s.fill("Filter versions", "24.11.0", false)
	s.waitForText("1 version")
	s.clickButtonNear("24.11.0", "Install")
	s.waitForTask("Install nodejs@24.11.0")

	tasks := a.model.Tasks.Tasks()
	for _, task := range tasks {
		check(task.Status() == state.TaskSucceeded, "every task succeeded")
	}
	progress, _ := tasks[0].Progress()
	check(progress == 1, "the download progress reached 100 %")
	check(strings.Contains(strings.Join(tasks[0].Log(), "\n"), "Install nodejs@24.11.0 success!"), "the output of vfox is kept")
}

func realScenario(s *scenario) {
	a := s.app
	server := a.environment["VFOX_PLUS_TEST_SERVER"]
	home := a.environment["HOME"]
	localPlugin := a.environment["VFOX_PLUS_E2E_LOCAL_PLUGIN"]

	control := func(action string) {
		response, err := http.Get(server + "/control/" + action)
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
		}
	}

	s.waitForText("Versions in use")
	installed, _ := a.model.Installed.Value()
	check(len(installed) == 0, "no plugin is installed yet")

	// Point vfox to the local registry through the vfox configuration editor.
	s.step("configure the registry")
	s.navigate("Settings")
	s.fill("Plugin registry", server+"/registry", true)
	s.waitUntil("the registry to be saved", func() bool {
		value, _ := a.model.VfoxConfig.ValueFor("registry.address")
		return value == server+"/registry"
	}, defaultWait)
	check(strings.TrimSpace(s.vfox("config", "registry.address")) == server+"/registry", "vfox stored the registry")

	// Add the plugin from the registry.
	s.step("add a plugin from the registry")
	s.navigate("Plugins")
	s.waitForText("testsdk")
	s.clickButton("Add")
	s.waitForTask("Add plugin testsdk")
	s.waitForText("Added")

	// Add a local copy of the plugin under another name. vfox moves local
	// .lua files away, so the application must hand it a copy.
	s.step("add a plugin from a local file")
	s.clickButton("Add from source…")
	s.waitForText("Add a plugin from source")
	s.fill("Source", localPlugin, false)
	s.fill("Name (optional)", "localsdk", false)
	s.clickButton("Add plugin")
	s.waitForTask("Add plugin localsdk")
	_, err := os.Stat(localPlugin)
	check(err == nil, "the source file survives vfox add --source")

	// Search and install both versions.
	s.step("install")
	s.navigate("SDKs")
	s.clickItem("testsdk")
	s.waitForText("An offline SDK for the vfox-plus end-to-end tests.")
	s.clickButton("Available")
	s.waitForText("2 versions")
	s.waitForText("LTS")
	s.waitForText("testtool 2.0")
	notes := 0
	for _, shown := range s.texts() {
		if shown == "LTS" {
			notes++
		}
	}
	check(notes == 2, "the note is shown once on the version, and once on the chip that filters by it")

	// The plugin takes an argument, which the real vfox hands to its Available hook:
	// the long-term support releases only. A chip of the notes narrows the list in the
	// same way, without asking vfox again.
	s.step("choose what to list")
	s.clickButton("LTS")
	s.waitForText("1 version")
	s.clickButton("Clear filters")
	s.waitForText("2 versions")
	s.clickButton("Channel: Every version")
	s.waitForText("Long-term support")
	s.clickItem("Long-term support")
	s.waitForText("Channel: Long-term support")
	s.waitUntil("the list of the long-term support releases", func() bool {
		versions, ok := a.detailFor("testsdk").controller.Versions.Value()
		return ok && len(versions) == 1
	}, defaultWait)
	versions, _ := a.detailFor("testsdk").controller.Versions.Value()
	check(versions[0].Version == "2.0.0" && versions[0].Note == "LTS", "vfox passed the argument to the plugin")
	check(strings.Contains(s.vfox("search", "testsdk", "lts"), "2.0.0") && !strings.Contains(s.vfox("search", "testsdk", "lts"), "1.0.0"), "vfox itself lists the same")
	s.waitForText("1 version")
	s.clickButton("Channel: Long-term support")
	s.clickItem("Every version")
	s.waitForText("2 versions")

	s.clickButtonNear("2.0.0", "Install")
	s.waitForTask("Install testsdk@2.0.0")
	progress, _ := s.newestTask().Progress()
	check(progress == 1, "vfox reports the download progress on standard error")

	// Cancel the installation of 1.0.0 during the download. From 1.0.13 vfox
	// leaves the partial installation behind, but must neither list it nor
	// refuse to install the version again.
	s.step("cancel an installation")
	control("hold")
	s.clickButtonNear("1.0.0", "Install")
	s.waitUntil("the download to start", func() bool {
		progress, ok := s.newestTask().Progress()
		return ok && progress > 0
	}, defaultWait)
	s.clickButton("Cancel")
	s.waitUntil("vfox to exit", func() bool {
		_, finished := s.newestTask().FinishedAt()
		return finished
	}, defaultWait)
	check(s.newestTask().Status() == state.TaskCancelled, "the task is cancelled")
	check(!strings.Contains(s.vfox("list", "testsdk"), "1.0.0"), "vfox does not list the partial version")
	control("release")
	pause(500)
	// Install it again by its name, which vfox looks up while it installs.
	s.step("install by name")
	s.clickButton("Install by name…")
	s.waitForText("Install testsdk by name")
	s.fill("Version name", "1.0.0", false)
	s.tapDialog(true)
	s.waitForTask("Install testsdk@1.0.0")

	// Make 2.0.0 the global default.
	s.step("use")
	s.navigate("SDKs")
	s.clickButton("Installed (2)")
	s.waitForText("1.0.0")
	s.clickButtonNear("2.0.0", "Use")
	s.waitForTask("Use testsdk@2.0.0")
	s.waitForText("In use: 2.0.0")
	toml, _ := os.ReadFile(home + "/.vfox/.vfox.toml") //nolint:gosec // A file of the sandbox.
	check(strings.Contains(string(toml), `testsdk = "2.0.0"`), "the global .vfox.toml pins 2.0.0")
	_, err = os.Lstat(home + "/.vfox/sdks/testsdk")
	check(err == nil, "the SDK is linked")

	// Uninstall 1.0.0.
	s.step("uninstall")
	s.chooseMenuItem("More actions for 1.0.0", "Uninstall")
	s.waitForText("Uninstall testsdk 1.0.0?")
	s.tapDialog(true)
	s.waitForTask("Uninstall testsdk@1.0.0")
	s.waitForText("Installed (1)")

	// Remove the second plugin.
	s.step("remove")
	s.clickItem("localsdk")
	s.waitForText("localsdk")
	s.chooseMenuItem("More actions", "Remove plugin")
	s.waitForText("Remove localsdk?")
	s.tapDialog(true)
	s.waitForTask("Remove plugin localsdk")
	s.waitForText("Select an SDK to manage its versions.")
	s.waitUntil("the removed plugin to disappear from the list", func() bool {
		sdks, _ := a.model.Installed.Value()
		for _, sdk := range sdks {
			if sdk.Name == "localsdk" {
				return false
			}
		}
		return true
	}, defaultWait)

	// Every task but the cancelled one succeeded, and vfox agrees with what
	// the application shows.
	var statuses []state.TaskStatus
	for _, task := range a.model.Tasks.Tasks() {
		if task.Status() != state.TaskSucceeded {
			statuses = append(statuses, task.Status())
		}
	}
	check(slices.Equal(statuses, []state.TaskStatus{state.TaskCancelled}), fmt.Sprintf("only the cancelled task did not succeed: %v", statuses))
	check(s.vfox("current") == "testsdk -> v2.0.0\n", "vfox uses testsdk 2.0.0")
	check(strings.Contains(s.vfox("list", "testsdk"), "2.0.0"), "vfox lists testsdk 2.0.0")
	// The copy of the local plugin is deleted once its task has finished.
	entries, _ := os.ReadDir(home + "/.vfox/tmp")
	var leftovers []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "vfox-plus-plugin-") {
			leftovers = append(leftovers, entry.Name())
		}
	}
	check(len(leftovers) == 0, fmt.Sprintf("the copy of the local plugin is deleted: %v", leftovers))
}

// measureScenario opens the Home page and the versions of Node.js, so that the
// harness can measure the memory of the application at both points (see
// docs/en/dev/memory.md).
func measureScenario(s *scenario) {
	s.waitForText("Versions in use")
	pause(3000)
	s.step("MEASURE home")
	pause(5000)
	s.navigate("SDKs")
	s.clickItem("nodejs")
	s.clickButton("Available")
	s.waitForText("872 versions")
	pause(3000)
	s.step("MEASURE nodejs")
	pause(5000)
	s.navigate("Home")
}

// shot waits for the page to settle and asks the harness to capture the screen.
func (s *scenario) shot(name string) {
	pause(1500)
	s.step("SHOT " + name)
	pause(1500)
}

// resize changes the size of the window, in dp.
func (s *scenario) resize(width, height float32) {
	s.app.win.SetContentSize(int(width), int(height))
	pause(500)
}

// screenshotsScenario shows every page against the fake vfox; the harness
// captures the screen at every `e2e: SHOT <name>` (see
// `vfxtool screenshots`). With VFOX_PLUS_FAKE_DELAY, it shows an
// installation in progress instead.
func screenshotsScenario(s *scenario) {
	a := s.app
	s.waitForText("Versions in use")
	// The startup overlay fades out after vfox is found; the first shot must
	// wait for it, or the Home page is still dimmed.
	s.waitUntil("the startup overlay to go away", func() bool {
		for _, shown := range s.texts() {
			if shown == "Looking for vfox…" {
				return false
			}
		}
		return true
	}, defaultWait)
	if a.environment["VFOX_PLUS_FAKE_DELAY"] != "" {
		s.navigate("SDKs")
		s.clickItem("nodejs")
		s.clickButton("Available")
		s.waitForText("872 versions")
		s.clickButtonNear("26.12.0", "Install")
		s.waitUntil("the download to start", func() bool {
			progress, ok := s.newestTask().Progress()
			return ok && progress > 0
		}, defaultWait)
		s.shot("installing")
		return
	}
	s.shot("home")
	s.navigate("SDKs")
	s.clickItem("nodejs")
	s.waitForText("Node.js runtime environment.")
	s.shot("sdks")
	s.clickButton("Available")
	s.waitForText("872 versions")
	s.shot("available")
	s.fill("Filter versions", "24.11.0", false)
	s.waitForText("1 version")
	s.clickButtonNear("24.11.0", "Install")
	s.waitForTask("Install nodejs@24.11.0")
	s.navigate("Tasks")
	s.clickButton("Show output")
	s.shot("tasks")
	s.navigate("Plugins")
	s.waitForText("dotnet")
	s.shot("plugins")
	s.clickButton("Add from source…")
	s.waitForText("Add a plugin from source")
	s.shot("dialog")
	s.tapDialog(false)
	s.navigate("Settings")
	s.waitForText("Plugin registry")
	s.shot("settings")
	a.model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		current.Theme, current.RailExpanded = settings.ThemeDark, true
		return current
	})
	s.navigate("Home")
	s.shot("dark-expanded")
	s.navigate("SDKs")
	s.clickItem("maven")
	s.waitForText("3.8.9")
	s.shot("dark-sdks")
	s.clickItem("nodejs")
	s.waitForText("Node.js runtime environment.")
	s.clickButton("Available")
	s.waitForText("872 versions")
	s.shot("dark-available")
	a.model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		current.Theme, current.RailExpanded, current.Language = settings.ThemeLight, false, settings.LanguageSimplifiedChinese
		return current
	})
	s.navigate("首页")
	s.waitForText("正在使用的版本")
	s.shot("zh-home")
	s.navigate("设置")
	s.waitForText("插件仓库")
	s.shot("zh-settings")
	a.model.Settings.Update(func(current settings.AppSettings) settings.AppSettings {
		current.Language = settings.LanguageEnglish
		return current
	})
	s.resize(412, 800)
	s.navigate("Home")
	s.waitForText("Versions in use")
	s.shot("compact-home")
	// Maven is still selected on the SDKs page, which compact windows would
	// show; the versions in use on the Home page lead to their SDK instead.
	s.clickItem("nodejs")
	s.waitForText("Node.js runtime environment.")
	s.shot("compact-detail")
}
