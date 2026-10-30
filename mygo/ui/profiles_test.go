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

// javaApp is the application with Java installed: Temurin 21 in use, Liberica 25 with
// JavaFX and OpenJDK 25, next to the Node.js of the fixtures.
func javaApp(t *testing.T) *testApp {
	t.Helper()
	runner := testsupport.FixtureRunner().
		On("list", core.ProcessOutput{Stdout: "All installed sdk versions\n├─┬java\n│ ├──21.0.12.1+1-tem\n│ ├──25.0.4+9-librca-fx\n│ └──25.0.2+10\n└─┬nodejs\n  ├──24.11.1\n  └──22.21.0\n"}).
		On("current", core.ProcessOutput{Stdout: "java -> v21.0.12.1+1-tem\nnodejs -> v24.11.1\n"}).
		On("info java --format "+vfox.PluginInfoTemplate, core.ProcessOutput{
			Stdout: "java\x1f0.6.3\x1fhttps://github.com/version-fox/vfox-java\x1f/home/u/.vfox/cache/java\x1fSupport for multiple JDK distributions, such as: Oracle, Graalvm, Eclipse & more.",
		}).
		On("info flutter --format "+vfox.PluginInfoTemplate, core.ProcessOutput{
			Stdout: "flutter\x1f0.2.5\x1fhttps://github.com/version-fox/vfox-flutter\x1f/home/u/.vfox/cache/flutter\x1fFlutter plugin, https://flutter.dev",
		})
	app := newTestApp(t, runner, windowWidth, windowHeight, settings.NewMemorySettingsStore(settings.DefaultSettings()))
	app.start()
	return app
}

// openAvailable opens the details of a plugin and its available versions, which it waits for.
func (ta *testApp) openAvailable(plugin string) {
	ta.t.Helper()
	ta.button("SDKs")
	ta.item(plugin)
	ta.button("Available")
	ta.until("the versions", func() bool { return ta.app.detailFor(plugin).controller.Versions.HasValue() })
	ta.settle()
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
	ta.mustText("120 versions")
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
	ta.until("the installation", func() bool { return len(ta.runner.Calls("install --yes java@27+35-tem")) == 1 })
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
		if text == "120 versions" {
			t.Fatal("LTS narrows the list")
		}
	}
	ta.snapshot("java-chips-on")
	ta.button("Clear filters")
	ta.mustText("120 versions")
	// A search of every distribution says which one a version is of.
	ta.button("Distribution: Temurin")
	field, _ := ta.find(kit.RoleTextField, "Search by name or short name")
	ta.fill(field, "all dist", false)
	ta.item("All distributions")
	ta.until("the versions of every distribution", func() bool {
		return len(ta.runner.Calls("search java all")) == 1 && ta.app.detailFor("java").controller.Versions.HasValue()
	})
	ta.settle()
	ta.mustText("2,641 versions")
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
	ta.button("Install by name…")
	ta.mustText("Install nodejs by name")
	ta.mustText("What follows nodejs@ in vfox install")
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
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("maven")
	if _, ok := ta.find(kit.RoleButton, "Options"); ok {
		t.Error("maven has no options tab")
	}
	ta.button("Available")
	ta.until("the versions", func() bool { return ta.app.detailFor("maven").controller.Versions.HasValue() })
	ta.settle()
	if strings.Contains(strings.Join(ta.texts(), "|"), "Distribution") {
		t.Error("maven has no distribution to choose")
	}
	ta.button("Install by name…")
	ta.mustText("Install maven by name")
	if ta.text("Names to try") {
		t.Error("maven has no names to suggest")
	}
	field, _ := ta.find(kit.RoleTextField, "Version name")
	ta.fill(field, "3.9.16", true)
	ta.until("the installation", func() bool { return len(ta.runner.Calls("install --yes maven@3.9.16")) == 1 })
}

func TestAPluginAddedUnderAnAliasHasTheChoicesOfThePluginItWasMadeFrom(t *testing.T) {
	ta := javaApp(t)
	ta.runner.
		On("list", core.ProcessOutput{Stdout: "All installed sdk versions\n└─┬coffee\n  └──21.0.12.1+1-tem\n"}).
		On("current", core.ProcessOutput{Stdout: "coffee -> v21.0.12.1+1-tem\n"}).
		On("info coffee --format "+vfox.PluginInfoTemplate, core.ProcessOutput{
			Stdout: "java\x1f0.6.3\x1fhttps://github.com/version-fox/vfox-java\x1f/home/u/.vfox/cache/coffee\x1fSupport for multiple JDK distributions.",
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
	ta.mustText("Besides the versions in the list, vfox install accepts these names for coffee.")
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
	ta.mustText("457 versions")
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
	ta.mustText("311 versions")
	ta.snapshot("flutter-chips")
}

func TestTheOptionsOfNodejsSetTheMirrorAndShowTheVersionFiles(t *testing.T) {
	ta := defaultApp(t)
	ta.button("SDKs")
	ta.item("nodejs")
	ta.button("Options")
	for _, text := range []string{
		"Version names", "Downloads", "Download mirror", "C library", "Advanced", "Version files",
		".node-version", ".nvmrc", "Names you can type", "Good to know", "Documentation",
	} {
		ta.mustText(text)
	}
	ta.snapshot("options")
	ta.mustText("VFOX_NODEJS_MIRROR\nNot set. The plugin uses its own default.")

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
	ta.mustText("VFOX_NODEJS_MIRROR\nnpmmirror · https://npmmirror.com/mirrors/node")
	// vfox runs with it from now on, and the settings page shows it among the other variables.
	ta.until("vfox to run with the mirror", func() bool {
		return ta.app.model.Cli.LaunchConfig().ExtraEnvironment["VFOX_NODEJS_MIRROR"] == "https://npmmirror.com/mirrors/node"
	})
	// Taking it away goes back to the default.
	ta.button("Change")
	ta.button("Default (not set)")
	ta.button("Save")
	ta.mustText("VFOX_NODEJS_MIRROR\nNot set. The plugin uses its own default.")

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
	// Java reads the file as it is: the strategy of vfox makes no difference.
	ta.mustText("Version used from legacy files has no effect on this plugin: the version in the file is what counts.")
	if _, ok := ta.find(kit.RoleListItem, "Version used from legacy files"); ok {
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

func TestAShorthandIsTriedFromTheOptions(t *testing.T) {
	ta := javaApp(t)
	ta.button("SDKs")
	ta.item("java")
	ta.button("Options")
	ta.mustText("21-tem")
	ta.mustText("The newest Temurin 21")
	ta.buttonNear("21-tem", "Install…")
	field, ok := ta.find(kit.RoleTextField, "Version name")
	if !ok || field.Value() != "21-tem" {
		t.Fatalf("the dialog starts with the shorthand: %v %q", ok, ta.texts())
	}
	ta.mustText("The newest Temurin 21")
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
	for _, text := range []string{"版本名称", "下载", "下载镜像", "版本文件", "可以直接输入的名称", "须知", "文档"} {
		if !ta.text(text) && !strings.Contains(strings.Join(ta.texts(), "\n"), text) {
			t.Errorf("no %q: %q", text, ta.texts())
		}
	}
	ta.snapshot("options-zh")
}
