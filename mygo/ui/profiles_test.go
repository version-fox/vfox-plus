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
	"strings"
	"testing"

	core "github.com/version-fox/vfox-plus/mygo/core"
	"github.com/version-fox/vfox-plus/mygo/core/profile"
	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/core/vfox"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// javaRunner is vfox with Java installed: Temurin 21 in use, Liberica 25 with JavaFX and
// OpenJDK 25, next to the Node.js of the fixtures.
func javaRunner() *testsupport.FakeProcessRunner {
	return testsupport.FixtureRunner().
		On("list", core.ProcessOutput{Stdout: "All installed sdk versions\n├─┬java\n│ ├──21.0.12.1+1-tem\n│ ├──25.0.4+9-librca-fx\n│ └──25.0.2+10\n└─┬nodejs\n  ├──24.11.1\n  └──22.21.0\n"}).
		On("current", core.ProcessOutput{Stdout: "java -> v21.0.12.1+1-tem\nnodejs -> v24.11.1\n"}).
		On("info java --format "+vfox.PluginInfoTemplate, core.ProcessOutput{
			Stdout: "java\x1f0.6.5\x1fhttps://github.com/version-fox/vfox-java\x1f/home/u/.vfox/cache/java\x1fSupport for multiple JDK distributions, such as: Oracle, Graalvm, Eclipse & more.",
		}).
		On("info flutter --format "+vfox.PluginInfoTemplate, core.ProcessOutput{
			Stdout: "flutter\x1f0.2.5\x1fhttps://github.com/version-fox/vfox-flutter\x1f/home/u/.vfox/cache/flutter\x1fFlutter plugin, https://flutter.dev",
		})
}

// javaApp is the application with Java installed.
func javaApp(t *testing.T) *testApp {
	t.Helper()
	app := newTestApp(t, javaRunner(), windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	app.start()
	return app
}

// addMore chooses one of the ways to add an SDK that the list does not offer, in the menu at the end of the row above
// the available versions.
func (ta *testApp) addMore(item string) {
	ta.t.Helper()
	ta.click(kit.RoleButton, "More ways to add")
	ta.click(kit.RoleMenuItem, item)
}

// openTab goes to a tab of the details of a plugin: its button, or, where the room is too short for the tabs and they are a
// dropdown that shows the tab that is open (the first, Installed, when the details have just been opened), the dropdown and
// the entry of the tab in its menu.
func (ta *testApp) openTab(name string) {
	ta.t.Helper()
	if _, ok := ta.find(kit.RoleButton, name); ok {
		ta.button(name)
		return
	}
	for _, n := range ta.nodes() {
		if n.Role == kit.RoleButton && n.Visible() && strings.HasPrefix(n.Name, "Installed") {
			ta.click(kit.RoleButton, n.Name)
			ta.click(kit.RoleMenuItem, name)
			return
		}
	}
	ta.t.Fatalf("no tab %q and no dropdown of the tabs (on screen: %q)", name, ta.texts())
}

// openAvailable opens the details of a plugin and its available versions, which it waits for.
func (ta *testApp) openAvailable(plugin string) {
	ta.t.Helper()
	ta.button("SDKs")
	ta.item(plugin)
	ta.openTab("Available")
	ta.until("the versions", func() bool { return ta.app.detailFor(plugin).controller.Versions.HasValue() })
	ta.settle()
}

// pythonRunner is the runner of a machine that has Python installed next to the others of the
// profile plugins (the captures of VFOX_PLUS_FAKE_INVENTORY=python).
func pythonRunner() *testsupport.FakeProcessRunner {
	return testsupport.FixtureRunner().
		On("list", testsupport.FixtureOutput("list-python", 0)).
		On("current", testsupport.FixtureOutput("current-python", 0))
}

// phpRunner is the runner of a machine that has PHP 8.5.12 installed next to the others. Its
// answers are a few lines that vfox 1.1.0 gives for the plugin php 0.3.4 on Linux, the census
// needs no more of them.
func phpRunner() *testsupport.FakeProcessRunner {
	return testsupport.FixtureRunner().
		On("info php --format "+vfox.PluginInfoTemplate, core.ProcessOutput{Stdout: "php\x1f0.3.4\x1fhttps://github.com/version-fox/vfox-php\x1f/home/u/.vfox/cache/php\x1fphp plugin"}).
		On("list", core.ProcessOutput{Stdout: "All installed sdk versions\n└─┬php\n  └──8.5.12\n"}).
		On("current", core.ProcessOutput{Stdout: "php -> v8.5.12\n"}).
		On("alias list php", testsupport.FixtureOutput("alias-list", 0)).
		On("search php", core.ProcessOutput{Stdout: "Available versions:\n - 8.5.12\n - 8.5.11\n - 8.5.10\n"})
}

// profileRunner is vfox with one of the plugins added after Maven and PHP
// installed on the machine. Its answers are the few lines the census needs to
// reach the options of the plugin and its install-by-name dialog.
func profileRunner(plugin string) *testsupport.FakeProcessRunner {
	var version, pluginVersion, description, search string
	switch plugin {
	case "dart":
		version, pluginVersion, description = "3.14.0", "0.1.2", "Dart SDK"
		search = "Available versions:\n - 3.14.0 (stable)\n - 3.14.0 (dev)\n - 3.13.0 (beta)\n"
	case "kotlin":
		version, pluginVersion, description = "2.1.0", "0.1.2", "Kotlin compiler"
		search = "Available versions:\n - 2.1.0\n - 2.0.0\n - 1.9.24\n"
	case "dotnet":
		version, pluginVersion, description = "8.0.34", "0.3.2", ".NET SDK"
		search = "Available versions:\n - 8.0.34 (LTS)\n - 8.0.34-runtime (Runtime - LTS)\n - 8.0.34-aspnetcore (ASP.NET Core Runtime - LTS)\n - 11.0.0-rc.3\n"
	default:
		panic("no profile machine for " + plugin)
	}
	return testsupport.FixtureRunner().
		On("info "+plugin+" --format "+vfox.PluginInfoTemplate, core.ProcessOutput{Stdout: plugin + "\x1f" + pluginVersion + "\x1fhttps://github.com/version-fox/vfox-" + plugin + "\x1f/home/u/.vfox/cache/" + plugin + "\x1f" + description}).
		On("list", core.ProcessOutput{Stdout: "All installed sdk versions\n└─┬" + plugin + "\n  └──" + version + "\n"}).
		On("current", core.ProcessOutput{Stdout: plugin + " -> v" + version + "\n"}).
		On("alias list "+plugin, testsupport.FixtureOutput("alias-list", 0)).
		On("search "+plugin, core.ProcessOutput{Stdout: search})
}

// pythonApp is the application on that machine.
func pythonApp(t *testing.T) *testApp {
	t.Helper()
	app := newTestApp(t, pythonRunner(), windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	app.start()
	return app
}

func TestThePythonOptionsSetTheInstallMethodAndShowTheVersionFiles(t *testing.T) {
	ta := pythonApp(t)
	ta.button("SDKs")
	ta.item("python")
	ta.button("Options")
	ta.until("the configuration", func() bool { return ta.app.model.VfoxConfig.HasValue() })
	ta.settle()
	for _, text := range []string{"Plugin variables", "python.org mirror", "Install method", "uv-build mirror", "Version files", ".python-version", "Notes"} {
		ta.mustText(text)
	}
	ta.snapshot("python-options")
	// The install method is one of two choices. The row shows the choice that was saved, with its value.
	ta.buttonNear("Install method", "Change")
	ta.button("uv-build only")
	ta.button("Save")
	ta.mustText("uv-build only · 1")
	ta.snapshot("python-install-method")
}

func TestThePythonVersionsFilterOnTheGraalPyBuilds(t *testing.T) {
	ta := pythonApp(t)
	ta.openAvailable("python")
	ta.mustText("999 versions")
	for _, chip := range []string{"GraalPy", "GraalPy Community"} {
		if _, ok := ta.find(kit.RoleButton, chip); !ok {
			t.Fatalf("no chip for %s: %q", chip, ta.texts())
		}
	}
	ta.button("GraalPy Community")
	if node, _ := ta.find(kit.RoleButton, "GraalPy Community"); !node.Selected {
		t.Error("the chip is on")
	}
	ta.mustText("18 versions")
	ta.snapshot("python-graalpy-community")
	// The chips of one row are alternatives: the two builds together are 49 versions.
	ta.button("GraalPy")
	ta.mustText("49 versions")
	ta.button("Clear filters")
	ta.mustText("999 versions")
}

func TestTheVersionsOfJavaSayWhichDistributionTheyAreOf(t *testing.T) {
	ta := javaApp(t)
	ta.button("SDKs")
	ta.item("java")
	ta.until("the plugin information", func() bool { _, ok := ta.app.detailFor("java").controller.Info.Value(); return ok })
	ta.settle()
	for _, text := range []string{"Installed (3)", "21.0.12.1+1-tem", "Temurin", "25.0.4+9-librca-fx", "Liberica · JavaFX", "25.0.2+10", "OpenJDK"} {
		ta.mustText(text)
	}
	ta.snapshot("java-installed")
}

func TestAnotherDistributionOfJavaIsChosenFromAListThatAFieldFilters(t *testing.T) {
	ta := javaApp(t)
	ta.openAvailable("java")
	// The plugin lists OpenJDK, as it does without an argument, and the button says so.
	ta.mustText("75 versions")
	if calls := ta.runner.Calls("search java"); len(calls) != 1 || len(calls[0].Arguments) != 2 {
		t.Fatalf("OpenJDK is searched with no argument: %v", calls)
	}
	ta.snapshot("java-available")
	ta.button("Distribution: OpenJDK")
	ta.mustText("Distribution")
	field, ok := ta.find(kit.RoleTextField, "Search by name or short name")
	if !ok {
		t.Fatalf("the choices have a filter field: %q", ta.texts())
	}
	ta.snapshot("java-choose")
	ta.fill(field, "adoptium", false)
	ta.mustText("Temurin")
	if ta.text("Zulu") {
		t.Error("the filter looks at the maker as well as the name")
	}
	ta.fill(field, "no such distribution", false)
	ta.mustText("Nothing matches.")
	ta.fill(field, "tem", false)
	ta.item("Temurin")
	ta.until("the versions of Temurin", func() bool {
		return len(ta.runner.Calls("search java tem")) == 1 && ta.app.detailFor("java").controller.Versions.HasValue()
	})
	ta.settle()
	ta.mustText("123 versions")
	if _, ok := ta.find(kit.RoleButton, "Distribution: Temurin"); !ok {
		t.Errorf("the button names the choice: %q", ta.texts())
	}
	if got := ta.app.model.RememberedSearch("java"); got != "tem" {
		t.Errorf("the choice is kept in the settings: %q", got)
	}
	ta.snapshot("java-temurin")

	// The choice outlives the details: they are built again for the next visit.
	ta.app.showSdk("")
	ta.settle()
	ta.item("java")
	ta.button("Available")
	ta.until("the versions of Temurin again", func() bool { return ta.app.detailFor("java").controller.Versions.HasValue() })
	ta.settle()
	if _, ok := ta.find(kit.RoleButton, "Distribution: Temurin"); !ok {
		t.Errorf("the choice was forgotten: %q", ta.texts())
	}
	// The list of Temurin is the one that vfox keeps for the argument, so a visit does not pay twice.
	node, ok := ta.find(kit.RoleButton, "Install")
	if !ok {
		t.Fatalf("no Install button: %q", ta.texts())
	}
	node.Click()
	ta.until("the installation", func() bool { return len(ta.runner.Calls("install --yes java@27.0.1-tem")) == 1 })
}

func TestTheFilterChipsNarrowTheVersionsThatAreListed(t *testing.T) {
	ta := javaApp(t)
	ta.openAvailable("java")
	ta.button("Distribution: OpenJDK")
	ta.item("Temurin")
	ta.until("the versions of Temurin", func() bool { return ta.app.detailFor("java").controller.Versions.HasValue() })
	ta.settle()
	ta.mustText("Support")
	node, ok := ta.find(kit.RoleButton, "LTS")
	if !ok || node.Selected {
		t.Fatalf("a chip for LTS that is off: %v %q", node, ta.texts())
	}
	ta.snapshot("java-chips-off")
	ta.button("LTS")
	if node, _ = ta.find(kit.RoleButton, "LTS"); !node.Selected {
		t.Error("the chip is on")
	}
	if _, ok := ta.find(kit.RoleButton, "Clear filters"); !ok {
		t.Errorf("a button takes the choices back: %q", ta.texts())
	}
	for _, text := range ta.texts() {
		if text == "123 versions" {
			t.Fatal("LTS narrows the list")
		}
	}
	ta.snapshot("java-chips-on")
	ta.button("Clear filters")
	ta.mustText("123 versions")
	// A search of every distribution says which one a version is of.
	ta.button("Distribution: Temurin")
	field, _ := ta.find(kit.RoleTextField, "Search by name or short name")
	ta.fill(field, "all dist", false)
	ta.item("All distributions")
	ta.until("the versions of every distribution", func() bool {
		return len(ta.runner.Calls("search java all")) == 1 && ta.app.detailFor("java").controller.Versions.HasValue()
	})
	ta.settle()
	ta.mustText("2,683 versions")
	ta.mustText("OpenJDK") // the tag of the first versions, which are OpenJDK's
	ta.mustText("Flavor")
	ta.snapshot("java-all")
}

func TestAnErrorOfASearchLeavesTheChoiceOfAnotherOne(t *testing.T) {
	ta := javaApp(t)
	ta.runner.On("search java bsg", core.ProcessOutput{ExitCode: 1, Stderr: "no available version"})
	ta.button("SDKs")
	ta.item("java")
	ta.button("Available")
	ta.until("OpenJDK", func() bool { return ta.app.detailFor("java").controller.Versions.HasValue() })
	ta.settle()
	ta.button("Distribution: OpenJDK")
	field, _ := ta.find(kit.RoleTextField, "Search by name or short name")
	ta.fill(field, "bsg", false)
	ta.item("BiSheng JDK")
	ta.until("the error", func() bool { return ta.app.detailFor("java").controller.Versions.Error() != nil })
	ta.settle()
	ta.mustText("no available version")
	// The button stays, so that another choice can be made.
	if _, ok := ta.find(kit.RoleButton, "Distribution: BiSheng JDK"); !ok {
		t.Errorf("the buttons stay while the search failed: %q", ta.texts())
	}
	ta.button("Distribution: BiSheng JDK")
	ta.item("Temurin")
	ta.until("Temurin", func() bool { return ta.app.detailFor("java").controller.Versions.HasValue() })
}

func TestAVersionIsInstalledByTheNameThatTheUserTypes(t *testing.T) {
	ta := defaultApp(t)
	ta.openAvailable("nodejs")
	ta.addMore("Install by name…")
	ta.mustText("Install nodejs by name")
	ta.mustText("24, 24.21 or 24.21.0 install the newest release of that line; latest the newest of any line. There is no lts alias: name the line.")
	ta.snapshot("install-by-name")
	// Nothing, and a name with a space, are refused with a reason.
	field, ok := ta.find(kit.RoleTextField, "Version name")
	if !ok {
		t.Fatalf("no field: %q", ta.texts())
	}
	ta.fill(field, "", true)
	ta.mustText("Type a version name.")
	ta.fill(field, "24 lts", true)
	ta.mustText("A version name has no spaces and does not start with a dash.")
	ta.fill(field, "--all", true)
	ta.mustText("A version name has no spaces and does not start with a dash.")
	if calls := ta.runner.Calls("install"); len(calls) != 0 {
		t.Errorf("nothing was installed: %v", calls)
	}
	// A name that the profile suggests fills the field, and says what it stands for.
	ta.button("latest")
	ta.mustText("The newest release of any line, LTS or not")
	if field, _ = ta.find(kit.RoleTextField, "Version name"); field.Value() != "latest" {
		t.Errorf("the field holds %q", field.Value())
	}
	ta.fill(field, "24", true)
	ta.until("the installation", func() bool { return len(ta.runner.Calls("install --yes nodejs@24")) == 1 })
	if ta.app.shell.destination != destTasks {
		t.Errorf("the task is shown: %v", ta.app.shell.destination)
	}
}

func TestAPluginWithoutAProfileTakesNamesToo(t *testing.T) {
	// mvn2 is an alias of a plugin that vfox-plus has no profile for, and its
	// information does not name the plugin: it gets no options tab and no names
	// of its own. (maven has a profile now.)
	ta := defaultApp(t)
	ta.runner.
		On("info mvn2 --format "+vfox.PluginInfoTemplate, testsupport.FixtureOutput("info-unknown", 1)).
		On("search mvn2", testsupport.FixtureOutput("search-maven-installed", 0))
	ta.button("SDKs")
	ta.item("mvn2")
	if _, ok := ta.find(kit.RoleButton, "Options"); ok {
		t.Error("mvn2 has no options tab")
	}
	ta.button("Available")
	ta.until("the versions", func() bool { return ta.app.detailFor("mvn2").controller.Versions.HasValue() })
	ta.settle()
	if strings.Contains(strings.Join(ta.texts(), "|"), "Distribution") {
		t.Error("mvn2 has no distribution to choose")
	}
	ta.addMore("Install by name…")
	ta.mustText("Install mvn2 by name")
	if ta.text("Names to try") {
		t.Error("mvn2 has no names to suggest")
	}
	field, _ := ta.find(kit.RoleTextField, "Version name")
	ta.fill(field, "3.9.16", true)
	ta.until("the installation", func() bool { return len(ta.runner.Calls("install --yes mvn2@3.9.16")) == 1 })
}

func TestAPluginAddedUnderAnAliasHasTheChoicesOfThePluginItWasMadeFrom(t *testing.T) {
	ta := javaApp(t)
	ta.runner.
		On("list", core.ProcessOutput{Stdout: "All installed sdk versions\n└─┬coffee\n  └──21.0.12.1+1-tem\n"}).
		On("current", core.ProcessOutput{Stdout: "coffee -> v21.0.12.1+1-tem\n"}).
		On("info coffee --format "+vfox.PluginInfoTemplate, core.ProcessOutput{
			Stdout: "java\x1f0.6.5\x1fhttps://github.com/version-fox/vfox-java\x1f/home/u/.vfox/cache/coffee\x1fSupport for multiple JDK distributions.",
		}).
		On("search coffee", testsupport.FixtureOutput("search-java", 0)).
		On("search coffee tem", testsupport.FixtureOutput("search-java-tem", 0))
	ta.app.model.Installed.Load()
	ta.until("coffee", func() bool { _, ok := ta.app.model.InstalledSdk("coffee"); return ok })
	ta.button("SDKs")
	ta.item("coffee")
	// The plugin is called coffee, and says that it is java: the details learn it from the
	// information, and show what they show for java.
	ta.until("the plugin information", func() bool { _, ok := ta.app.detailFor("coffee").controller.Info.Value(); return ok })
	ta.settle()
	ta.mustText("Temurin")
	ta.button("Options")
	ta.mustText("GitHub mirror")
	ta.button("Available")
	ta.until("the versions", func() bool { return ta.app.detailFor("coffee").controller.Versions.HasValue() })
	ta.settle()
	ta.button("Distribution: OpenJDK")
	field, ok := ta.find(kit.RoleTextField, "Search by name or short name")
	if !ok {
		t.Fatalf("the choices have a filter field: %q", ta.texts())
	}
	ta.fill(field, "tem", false)
	ta.item("Temurin")
	ta.until("the versions of Temurin", func() bool {
		return len(ta.runner.Calls("search coffee tem")) == 1 && ta.app.detailFor("coffee").controller.Versions.HasValue()
	})
	// The choice is the one of the plugin that has this name, not of java.
	if got := ta.app.model.RememberedSearch("coffee"); got != "tem" || ta.app.model.RememberedSearch("java") != "" {
		t.Errorf("the choice is kept for coffee: %q, java %q", got, ta.app.model.RememberedSearch("java"))
	}
}

func TestTheChannelsAndTheArchitecturesOfFlutterAreFilterChips(t *testing.T) {
	ta := javaApp(t)
	ta.runner.On("list", core.ProcessOutput{Stdout: "All installed sdk versions\n└─┬flutter\n  └──3.47.7\n"}).
		On("current", core.ProcessOutput{Stdout: "flutter -> v3.47.7\n"})
	ta.app.model.Installed.Load()
	ta.until("flutter", func() bool { _, ok := ta.app.model.InstalledSdk("flutter"); return ok })
	ta.openAvailable("flutter")
	ta.mustText("460 versions")
	for _, chip := range []string{"stable", "beta", "dev", "OpenHarmony", "x64"} {
		if _, ok := ta.find(kit.RoleButton, chip); !ok {
			t.Errorf("no chip %q: %q", chip, ta.texts())
		}
	}
	if _, ok := ta.find(kit.RoleButton, "arm64"); ok {
		t.Error("the list of this machine has no arm64 builds")
	}
	ta.button("OpenHarmony")
	ta.mustText("25 versions")
	ta.button("OpenHarmony")
	ta.button("x64")
	ta.mustText("314 versions")
	ta.snapshot("flutter-chips")
}

func TestTheOptionsOfNodejsSetTheMirrorAndShowTheVersionFiles(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("nodejs")
	ta.button("Options")
	ta.until("the configuration", func() bool { return ta.app.model.VfoxConfig.HasValue() })
	ta.settle()
	for _, text := range []string{
		"Plugin variables", "Download mirror", "C library", "Advanced", "Version files", "Read legacy version files",
		".node-version", ".nvmrc", "Aliases", "Notes", "Documentation",
	} {
		ta.mustText(text)
	}
	// What the dialog of the names says, or the buttons of the page, is not said again here.
	for _, text := range []string{"Version names", "Names you can type", "Good to know", "No aliases yet."} {
		if ta.text(text) {
			t.Errorf("the options say %q", text)
		}
	}
	ta.snapshot("options")
	ta.mustText("VFOX_NODEJS_MIRROR")
	ta.mustText("Default")

	// An address that is not one is refused, and nothing changes.
	ta.button("Change")
	field, ok := ta.find(kit.RoleTextField, "Address")
	if !ok {
		t.Fatalf("no field for the address: %q", ta.texts())
	}
	ta.snapshot("options-mirror")
	ta.fill(field, "npmmirror.com", true)
	ta.mustText("Use an address that starts with http:// or https://.")
	if got := ta.app.model.PluginVariable("VFOX_NODEJS_MIRROR"); got != "" {
		t.Errorf("stored %q", got)
	}
	// A mirror that was checked is picked with a click.
	ta.button("npmmirror")
	if field, _ = ta.find(kit.RoleTextField, "Address"); field.Value() != "https://npmmirror.com/mirrors/node" {
		t.Errorf("the field holds %q", field.Value())
	}
	ta.button("Save")
	if got := ta.app.model.PluginVariable("VFOX_NODEJS_MIRROR"); got != "https://npmmirror.com/mirrors/node" {
		t.Errorf("stored %q", got)
	}
	ta.mustText("npmmirror · https://npmmirror.com/mirrors/node")
	// vfox runs with it from now on, and the settings page shows it among the other variables.
	ta.until("vfox to run with the mirror", func() bool {
		return ta.app.model.Cli.LaunchConfig().ExtraEnvironment["VFOX_NODEJS_MIRROR"] == "https://npmmirror.com/mirrors/node"
	})
	// Taking it away goes back to the default.
	ta.button("Change")
	ta.button("Default (not set)")
	ta.button("Save")
	ta.mustText("VFOX_NODEJS_MIRROR")
	ta.mustText("Default")

	// A choice takes its values only.
	nodejs, _ := ta.app.model.Profiles.Lookup("nodejs")
	ta.app.askVariable(nodejs, mustVariable(t, ta, "nodejs", "VFOX_NODEJS_LIBC"))
	ta.settle()
	ta.button("musl")
	ta.button("Save")
	if got := ta.app.model.PluginVariable("VFOX_NODEJS_LIBC"); got != "musl" {
		t.Errorf("stored %q", got)
	}
}

func mustVariable(t *testing.T, ta *testApp, plugin, key string) (variable profile.Variable) {
	t.Helper()
	known, ok := ta.app.model.Profiles.Lookup(plugin)
	if !ok {
		t.Fatalf("no profile for %s", plugin)
	}
	variable, ok = known.Variable(key)
	if !ok {
		t.Fatalf("%s does not read %s", plugin, key)
	}
	return variable
}

func TestTheVersionFilesAreReadThroughThePluginIfVfoxReadsThem(t *testing.T) {
	ta := javaApp(t)
	ta.button("SDKs")
	ta.item("java")
	ta.button("Options")
	ta.until("the configuration", func() bool { return ta.app.model.VfoxConfig.HasValue() })
	ta.settle()
	for _, text := range []string{".sdkmanrc", "GitHub mirror", "Read legacy version files"} {
		ta.mustText(text)
	}
	// Java reads the file as it is: the strategy of vfox makes no difference, so there is none to choose.
	if _, ok := ta.find(kit.RoleListItem, "Version used from legacy files"); ok || ta.text("Version used from legacy files") {
		t.Error("there is no strategy to choose for Java")
	}
	ta.snapshot("options-java")
	// The switch changes the configuration of vfox itself.
	ta.runner.On("config legacyVersionFile.enable false", core.ProcessOutput{})
	ta.item("Read legacy version files")
	ta.until("the setting", func() bool { return len(ta.runner.Calls("config legacyVersionFile.enable false")) == 1 })

	// The documentation opens in the browser, in the language of the application.
	ta.item("Versions and names")
	if len(ta.desktop.opened) != 1 || ta.desktop.opened[0] != "https://github.com/version-fox/vfox-java/blob/main/docs/versions.md" {
		t.Errorf("opened %v", ta.desktop.opened)
	}
}

func TestAShorthandFillsTheFieldOfTheNames(t *testing.T) {
	ta := javaApp(t)
	ta.openAvailable("java")
	ta.addMore("Install by name…")
	ta.button("21-tem")
	ta.mustText("The newest Temurin 21")
	field, ok := ta.find(kit.RoleTextField, "Version name")
	if !ok || field.Value() != "21-tem" {
		t.Fatalf("the chip fills the field: %v %q", ok, ta.texts())
	}
	ta.button("Install")
	ta.until("the installation", func() bool { return len(ta.runner.Calls("install --yes java@21-tem")) == 1 })
}

func TestTheOptionsFollowTheLanguage(t *testing.T) {
	ta := defaultApp(t)
	ta.app.updateSettings(func(s *settings.AppSettings) { s.Language = settings.LanguageSimplifiedChinese })
	ta.settle()
	ta.button("SDK")
	ta.item("nodejs")
	ta.button("选项")
	ta.until("the configuration", func() bool { return ta.app.model.VfoxConfig.HasValue() })
	ta.settle()
	for _, text := range []string{"插件变量", "下载镜像", "版本文件", "别名", "须知", "文档"} {
		if !ta.text(text) && !strings.Contains(strings.Join(ta.texts(), "\n"), text) {
			t.Errorf("no %q: %q", text, ta.texts())
		}
	}
	ta.snapshot("options-zh")
}

func TestInstallByNameOffersUseAfterInstall(t *testing.T) {
	ta := defaultApp(t)
	ta.openAvailable("nodejs")
	ta.addMore("Install by name…")
	// The switch says what it is for, in the text of the dialog and not only for assistive technology.
	ta.mustText("Use after install")
	// Off by default: plain install.
	field, ok := ta.find(kit.RoleTextField, "Version name")
	if !ok {
		t.Fatalf("no field: %q", ta.texts())
	}
	ta.fill(field, "24", true)
	ta.until("the installation", func() bool { return len(ta.runner.Calls("install --yes nodejs@24")) == 1 })
	ta.app.showSdk("")
	ta.settle()
	// On: install and use.
	ta.openAvailable("nodejs")
	ta.addMore("Install by name…")
	ta.mustText("Use after install")
	ta.click(kit.RoleSwitch, "Use after install")
	field, _ = ta.find(kit.RoleTextField, "Version name")
	ta.fill(field, "24", true)
	ta.until("the install and use", func() bool { return len(ta.runner.Calls("install --yes --use nodejs@24")) == 1 })
}

func TestVersionsListedOncePerChannelGetTheirOwnRows(t *testing.T) {
	// vfox-dart lists a version once for every channel it is published on;
	// the rows share a version but are different entries of the list.
	ta := newTestApp(t, profileRunner("dart"), windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.start()
	ta.openAvailable("dart")
	ta.settle()
	rows := 0
	for _, n := range ta.nodes() {
		if n.Role == kit.RoleListItem && n.Name == "3.14.0" {
			rows++
		}
	}
	if rows != 2 {
		t.Errorf("3.14.0 is listed once for stable and once for dev, got %d rows: %q", rows, ta.texts())
	}
}

func TestAvailableVersionsOfferInstallAndUseAndLink(t *testing.T) {
	ta := defaultApp(t)
	ta.openAvailable("nodejs")
	ta.click(kit.RoleButton, "More ways to install this version")
	ta.mustText("Install and use")
	ta.click(kit.RoleMenuItem, "Install and use")
	// The other ways to add an SDK are the items of one menu, and not buttons of their own.
	for _, text := range []string{"Install by name…", "Detect and link", "Link existing…"} {
		if ta.text(text) {
			t.Errorf("the row above the list shows %q", text)
		}
	}
	ta.addMore("Detect and link")
	ta.until("the detection", func() bool { return len(ta.runner.Calls("link --detect")) == 1 })
}

func TestManualLinkNeedsAVersionAndADirectory(t *testing.T) {
	ta := defaultApp(t)
	ta.openAvailable("nodejs")
	ta.addMore("Link existing…")
	ta.mustText("Link nodejs at a directory")
	ta.mustText("Repoint an already linked version")
	// Submit empty: stays open with errors.
	ta.button("Link")
	ta.mustText("Type a version name.")
	ta.mustText("Link nodejs at a directory")
}

func TestAnAliasIsNamedForAVersion(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("nodejs")
	ta.button("Options")
	ta.mustText("Aliases")
	ta.until("the aliases", func() bool { return ta.text("Name alias…") })
	ta.button("Name alias…")
	ta.mustText("Alias")
	field, ok := ta.find(kit.RoleTextField, "Alias")
	if !ok {
		t.Fatalf("no alias field: %q", ta.texts())
	}
	ta.fill(field, "lts", false)
	ver, ok := ta.find(kit.RoleTextField, "Version name")
	if !ok {
		t.Fatalf("no version field: %q", ta.texts())
	}
	ta.fill(ver, "24.11.1", true)
	ta.until("the alias", func() bool { return len(ta.runner.Calls("alias set")) == 1 })
}

func TestAliasesAreListedAndRemoved(t *testing.T) {
	runner := testsupport.FixtureRunner().
		On("alias list nodejs", core.ProcessOutput{Stdout: "nodejs@lts -> nodejs@22.21.0\n"})
	ta := newTestApp(t, runner, windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	ta.start()
	ta.button("SDKs")
	ta.item("nodejs")
	// An installed version says the alias that names it, after what the plugin says of it.
	ta.until("the alias of a version", func() bool { return strings.Contains(strings.Join(ta.texts(), "\n"), "@lts") })
	ta.button("Options")
	ta.until("the aliases", func() bool { return ta.text("lts") })
	ta.mustText("nodejs@22.21.0")
	ta.button("Remove")
	ta.until("the removal", func() bool { return len(ta.runner.Calls("alias unset nodejs lts")) == 1 })
	// The task says which alias it removes, and of what.
	ta.button("Tasks")
	ta.mustText("Remove alias lts of nodejs")
}

func TestAvailableRowOffersInstallAndUse(t *testing.T) {
	ta := defaultApp(t)
	ta.openAvailable("nodejs")
	// The first row (26.13.0) has Install, and next to it the menu that has Install and use.
	ta.click(kit.RoleButton, "More ways to install this version")
	ta.click(kit.RoleMenuItem, "Install and use")
	ta.until("the install and use", func() bool {
		for _, call := range ta.runner.Calls("install") {
			if len(call.Arguments) >= 4 && call.Arguments[2] == "--use" {
				return true
			}
		}
		return false
	})
}
