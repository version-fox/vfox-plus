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

// Package screenshots ports shared/tools/screenshots.sh: it starts the application
// against the fake vfox five times (plain, with a slow fake vfox, with Java and
// Flutter installed next to Node.js, and the last two in the window of the compact
// layout), drives it to every page through the automation plugin (see package
// appsession), and captures the screen with ImageMagick at each page: the window
// with its title bar, as a person sees it.
package screenshots

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/version-fox/vfox-plus/shared/tooling/internal/appsession"
	"github.com/version-fox/vfox-plus/shared/tooling/internal/automation"
)

// shooter captures the screen when a page has settled.
type shooter struct {
	dir string
}

// shot waits for the page to settle, and captures the screen.
func (c shooter) shot(s *appsession.Session, name string) {
	s.Settle()
	s.Pause(1500 * time.Millisecond)
	target := filepath.Join(c.dir, name+".png")
	command := exec.Command("import", "-window", "root", target)
	if out, err := command.CombinedOutput(); err != nil {
		s.Fail("capture screenshot %s: %v: %s", name, err, strings.TrimSpace(string(out)))
	}
	fmt.Println(name + ".png")
}

// A run of the application: what the fake vfox plays (its environment), the
// settings that it starts with besides the language and the vfox executable, and the
// scenario that shows the pages.
type run struct {
	name     string
	env      []string
	settings map[string]any
	scenario func(*appsession.Session, shooter)
}

// documentationAddress is the address of the computer in the pictures of the web interface: the
// one that the end-to-end build is told (VFOX_PLUS_E2E_ADDRESSES), an address of the kind that a home
// network has, the same at every run, and the address of no machine.
const documentationAddress = "192.168.1.20"

var runs = []run{
	{name: "pages", scenario: pages},
	// A slow fake vfox, so that an installation is in progress.
	{name: "installing", env: []string{"VFOX_PLUS_FAKE_DELAY=4"}, scenario: installing},
	// The pages of the plugins that have a profile need a machine that has them.
	{name: "profiles", env: []string{"VFOX_PLUS_FAKE_INVENTORY=profiles"}, scenario: profiles},
	// The compact layout is that of a window less than 600 dp wide.
	{name: "compact pages", env: []string{"VFOX_PLUS_E2E_WINDOW=412x800"}, scenario: compactPages},
	{name: "compact profiles", env: []string{"VFOX_PLUS_E2E_WINDOW=412x800", "VFOX_PLUS_FAKE_INVENTORY=profiles"}, scenario: compactProfiles},
	// The settings of the web interface: it is on, as a person turns it on, and the computer has an address to pair a phone with.
	// The pictures for the guide in Simplified Chinese are the same in that language, which the run chooses first, as a person does.
	{name: "web interface", env: []string{"VFOX_PLUS_E2E_ADDRESSES=" + documentationAddress},
		settings: map[string]any{"webInterface": true}, scenario: webInterface(english)},
	{name: "web interface in Simplified Chinese", env: []string{"VFOX_PLUS_E2E_ADDRESSES=" + documentationAddress},
		settings: map[string]any{"webInterface": true}, scenario: webInterface(chinese)},
}

// Run implements `screenshots <output-dir> <app command...>`,
// mirroring shared/tools/screenshots.sh.
func Run(args []string) error {
	if len(args) < 2 {
		return errors.New("Usage: screenshots <output directory> <application command...>")
	}
	outputDir, command := args[0], args[1:]
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	root, err := appsession.FindRepoRoot()
	if err != nil {
		return err
	}
	for _, r := range runs {
		config := appsession.Config{Name: "screenshots", Command: command, Fixtures: appsession.FixturesDir(root), Env: r.env, Settings: r.settings}
		err := appsession.RunScenario(config, r.name, func(s *appsession.Session) {
			// The startup overlay fades out after vfox is found; the first shot must
			// wait for it, or the Home page is still dimmed.
			s.WaitForText("Versions in use")
			s.WaitUntil("the startup overlay to go away", 30*time.Second, func(t automation.Tree) bool { return !t.Shows("Looking for vfox…") })
			r.scenario(s, shooter{outputDir})
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// pages shows every page against the fake vfox, in the light theme and in the
// dark, and in Chinese.
func pages(s *appsession.Session, c shooter) {
	c.shot(s, "home")
	s.Navigate(appsession.SDKs)
	s.ClickItem("nodejs")
	s.WaitForText("Node.js runtime environment.")
	c.shot(s, "sdks")
	s.ClickButton("Available")
	s.WaitForText("872 versions")
	c.shot(s, "available")
	s.Fill("Filter versions", "24.11.0", false)
	s.WaitForText("1 version")
	s.ClickButtonNear("24.11.0", "Install")
	s.WaitForTask("Install nodejs@24.11.0")
	s.Navigate(appsession.Tasks)
	// A click on the row of a task shows what vfox printed.
	s.ClickItem("Install nodejs@24.11.0")
	c.shot(s, "tasks")
	s.Navigate(appsession.Plugins)
	s.WaitForText("dotnet")
	c.shot(s, "plugins")
	s.ClickButton("Add from source…")
	s.WaitForText("Add a plugin from source")
	c.shot(s, "dialog")
	s.TapDialog(false)
	s.Navigate(appsession.Settings)
	s.WaitForText("Plugin registry")
	c.shot(s, "settings")

	s.PressKey("dark")
	s.ClickButton("Show labels")
	s.Navigate(appsession.Home)
	c.shot(s, "dark-expanded")
	s.Navigate(appsession.SDKs)
	s.ClickItem("maven")
	s.WaitForText("3.8.9")
	c.shot(s, "dark-sdks")
	s.ClickItem("nodejs")
	s.WaitForText("Node.js runtime environment.")
	s.ClickButton("Available")
	s.WaitForText("872 versions")
	c.shot(s, "dark-available")

	s.Navigate(appsession.Settings)
	s.PressKey("light")
	s.ClickButton("Hide labels")
	s.PressKey("simplifiedChinese")
	s.Navigate(appsession.Home)
	s.WaitForText("正在使用的版本")
	c.shot(s, "zh-home")
	s.Navigate(appsession.Settings)
	s.WaitForText("插件仓库")
	c.shot(s, "zh-settings")
}

// scrollTo moves the page so that the element with the key is in view: a page that is longer than
// the window shows its top, and the picture of what is further down needs the page moved first.
func scrollTo(s *appsession.Session, key string) {
	var found automation.Node
	s.WaitUntil("the element with the key "+key, 30*time.Second, func(t automation.Tree) bool {
		node, ok := t.Find(func(n automation.Node) bool { return n.KeyText() == key })
		found = node
		return ok
	})
	if err := s.Client().Act(found.ID, automation.ScrollIntoView, ""); err != nil {
		s.Fail("scroll to %s: %v", key, err)
	}
	s.Settle()
}

// words are what the scenario of the web interface waits for and clicks, in the language of the run, and the
// prefix of the names of its pictures.
type words struct {
	prefix                                  string
	language                                string // the key of the button that chooses the language, or "" to keep English.
	section, allow, pair, showCode, newCode string
}

var (
	english = words{"", "", "Web interface", "Allow other devices?", "Pair a phone", "Show code", "New code"}
	chinese = words{"zh-", "simplifiedChinese", "网页界面", "允许其他设备？", "配对手机", "显示二维码", "换一个二维码"}
)

// webInterface shows what the settings say of the web interface, as a person goes through it: the
// switch that is on and the address that serves, the dialog that asks before other devices may connect,
// the settings that come with them, and the code that pairs a phone.
func webInterface(w words) func(*appsession.Session, shooter) {
	return func(s *appsession.Session, c shooter) {
		s.Navigate(appsession.Settings)
		if w.language != "" {
			s.PressKey(w.language)
		}
		s.WaitForText(w.section)
		// The page is scrolled to its end and back up to the row of the language, so that the window starts there, with
		// the group of the web interface after it, as the pictures that follow do: scrolling shows an element with
		// as little movement as it takes, which puts one that is above the window at the top of it.
		scrollTo(s, "licenses")
		scrollTo(s, "english")
		c.shot(s, w.prefix+"web-interface")

		s.PressKey("web-remote")
		s.WaitForText(w.allow)
		c.shot(s, w.prefix+"web-allow")
		s.TapDialog(true)
		s.WaitForText(w.pair)
		s.PressKey("lightweight")
		scrollTo(s, "web-quit-button")
		c.shot(s, w.prefix+"web-remote")

		s.ClickButton(w.showCode)
		s.WaitForText(w.newCode)
		c.shot(s, w.prefix+"web-pair")
	}
}

// installing shows an installation in progress.
func installing(s *appsession.Session, c shooter) {
	s.Navigate(appsession.SDKs)
	s.ClickItem("nodejs")
	s.ClickButton("Available")
	s.WaitForText("872 versions")
	s.ClickButtonNear("26.12.0", "Install")
	s.WaitUntil("the download to start", 30*time.Second, func(t automation.Tree) bool {
		percent, _ := t.Downloading()
		return percent > 0
	})
	c.shot(s, "installing")
}

// profiles shows what vfox-plus offers for the plugins that it knows: the
// distributions of Java, the filter chips of its versions and of Flutter's, the
// installation by name, and the options of Node.js. The fake vfox plays a machine
// that has Java and Flutter installed next to Node.js.
func profiles(s *appsession.Session, c shooter) {
	s.Navigate(appsession.SDKs)
	s.ClickItem("java")
	s.WaitForText("Support for multiple JDK distributions, such as: Oracle, Graalvm, Eclipse & more.")
	s.WaitForText("Liberica · JavaFX")
	c.shot(s, "java-installed")
	s.ClickButton("Available")
	s.WaitForText("75 versions")
	c.shot(s, "java-available")
	s.ClickButton("Distribution: OpenJDK")
	s.WaitForText("Distribution")
	c.shot(s, "java-choose")
	s.ClickItem("Temurin")
	s.WaitForText("120 versions")
	s.ClickButton("LTS")
	s.WaitForTextGone("120 versions")
	c.shot(s, "java-temurin")
	s.ClickButton("Install by name…")
	s.WaitForText("Install java by name")
	s.ClickButton("21-tem")
	s.WaitForText("The newest Temurin 21")
	c.shot(s, "java-install-name")
	s.TapDialog(false)
	s.ClickButton("Distribution: Temurin")
	s.WaitForText("Distribution")
	s.ClickItem("All distributions")
	s.WaitForText("2,641 versions")
	c.shot(s, "java-all")
	// The choice is kept, so Java would list every distribution again at the next visit:
	// take it back to the default, as a user does.
	s.ClickButton("Distribution: All distributions")
	s.WaitForText("Distribution")
	s.ClickItem("OpenJDK")
	s.WaitForText("75 versions")

	s.Navigate(appsession.SDKs)
	s.ClickItem("flutter")
	s.ClickButton("Available")
	s.WaitForText("457 versions")
	s.ClickButton("stable")
	s.WaitForTextGone("457 versions")
	c.shot(s, "flutter-available")

	s.Navigate(appsession.SDKs)
	s.ClickItem("nodejs")
	s.ClickButton("Options")
	s.WaitForText("Download mirror")
	c.shot(s, "options")
	s.ClickButton("Change")
	s.WaitForText("Address")
	s.ClickButton("npmmirror")
	c.shot(s, "options-mirror")
	s.TapDialog(false)

	s.Navigate(appsession.Settings)
	s.PressKey("simplifiedChinese")
	s.Navigate(appsession.SDKs)
	s.WaitForText("版本名称")
	c.shot(s, "zh-options")
	s.ClickItem("java")
	s.ClickButton("可安装")
	s.WaitForText("75 个版本")
	c.shot(s, "zh-java-available")
}

// compactPages shows the Home page and the versions of Node.js in a window of the
// compact layout, where the navigation is at the bottom.
func compactPages(s *appsession.Session, c shooter) {
	c.shot(s, "compact-home")
	// The versions in use on the Home page lead to their SDK.
	s.ClickItem("nodejs")
	s.WaitForText("Node.js runtime environment.")
	c.shot(s, "compact-detail")
}

// compactProfiles shows the distributions of Java in a window of the compact layout.
func compactProfiles(s *appsession.Session, c shooter) {
	s.ClickItem("java")
	s.ClickButton("Available")
	s.WaitForText("75 versions")
	s.ClickButton("Distribution: OpenJDK")
	s.ClickItem("Temurin")
	s.WaitForText("120 versions")
	c.shot(s, "compact-java")
}
