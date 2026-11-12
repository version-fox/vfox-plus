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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/internal/testsupport"
	"github.com/version-fox/vfox-plus/mygo/pair"
	"github.com/version-fox/vfox-plus/mygo/state"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// The census of the screens of the desktop application, the way `vfxtool webheap -audit` takes the census of the web
// interface. Every screen is drawn without a window, in a window as wide as the one that the application opens with
// and in a compact one, in the light theme and the dark, and held to the budgets of docs/en/dev/design-system.md#budgets
// (kit.Census.Over): the words that it says, the rows that it gives two buttons, the icon that three of its rows start
// with, the texts that cannot be read. The windows are as tall as a page can be, so that what a page has below the fold
// is drawn, and counted, too.
//
//	VFOX_PLUS_CENSUS=/tmp/census go test ./ui -run TestTheScreensAreWithinTheBudgets
//
// writes the census of every screen to that directory, as JSON, and a table of them, summary.txt.

// censusWindow is a window that the census draws a screen in.
type censusWindow struct {
	name          string
	width, height int
	dark          bool
	// scale is the text scale of the system that the window is drawn with; 0 is 1.
	scale float32
	// shrunk is the height that the window is made when the scene has reached its screen, if it is not 0: a scene goes to its
	// screen in a window that is tall enough to press everything in, and a person who makes the window smaller sees what is left.
	shrunk int
	// language is the language that the screen is shown in, when it is not English, which the application starts in and the
	// scenes are told in: it is changed where the scene has reached the screen.
	language settings.LanguagePreference
}

// heightFor is the height of the window for a scene: its own, or, for a page that is taller than the windows of the census, as tall
// as the page is: more, by a fifth or so, than the tallest page needs at the size of text of the window, in the fonts of Linux,
// of Windows and of macOS (a list of the distributions of Java is 7 000 dp in the narrowest window and 9 000 at 200 %), so that
// what a page has below the fold is drawn and counted, too. What the census says of a page that is taller than that is
// kit.Census.Taller. A window as tall as that is much to draw, and no more is made of it than a page needs.
func (w censusWindow) heightFor(s censusScene) int {
	if !s.tall {
		return w.height
	}
	scale := float64(max(w.scale, 1))
	switch {
	case w.width >= 1000:
		return int(5500 * scale)
	case w.width >= 600:
		return int(8500 + 3000*(scale-1))
	default:
		return int(9000 + 3500*(scale-1))
	}
}

// over is what the census of a screen is over, in the budgets that the window is held to: all of them, and for a language that
// has no spaces those of the layout alone, which count no words (kit.Census.OverLayout).
func (w censusWindow) over(c kit.Census) []string {
	if w.language != "" {
		return c.OverLayout()
	}
	return c.Over()
}

var (
	censusWide = censusWindow{name: "wide", width: 1280, height: 1600}
	censusDark = censusWindow{name: "wide-dark", width: 1280, height: 1600, dark: true}
	// The windows in between have a rail and one pane at a time, and the narrowest that the application opens in is 360 dp.
	censusMedium  = censusWindow{name: "medium", width: 700, height: 1800}
	censusCompact = censusWindow{name: "compact", width: 412, height: 2400}
	censusNarrow  = censusWindow{name: "narrow", width: 360, height: 2400}
	// The text that the system makes larger is held up to 200 % (docs/en/dev/design-system.md#budgets): at 200 % in the
	// narrowest window, in one of the rail and in the widest, which are the three that lay out differently; and at 130 % and
	// 150 % in the narrowest, and at 150 % in the next, where the room is the least and what did not fit was found. The sizes in
	// between are taken by TestTheScreensAreWithinTheBudgetsAtEverySizeOfText, when it is asked to.
	censusNarrowMid   = censusWindow{name: "narrow-130", width: 360, height: 2400, scale: 1.3}
	censusNarrowMore  = censusWindow{name: "narrow-150", width: 360, height: 2400, scale: 1.5}
	censusCompactMore = censusWindow{name: "compact-150", width: 412, height: 2400, scale: 1.5}
	censusNarrowLarge = censusWindow{name: "narrow-200", width: 360, height: 2400, scale: 2}
	censusMediumLarge = censusWindow{name: "medium-200", width: 700, height: 1800, scale: 2}
	censusWideLarge   = censusWindow{name: "wide-200", width: 1280, height: 1600, scale: 2}
	// The least window that the application opens in is 360 by 480 dp (MinWidth and MinHeight of its window), and a window may be as
	// wide as a screen and as short as that. A page that the foot of the window cuts scrolls, so the pages are held to nothing of it;
	// the dialogs and the menus are, in these windows: their title and their actions are where a person can reach them.
	censusLeast      = censusWindow{name: "least", width: 360, height: 2400, shrunk: 480}
	censusLeastLarge = censusWindow{name: "least-200", width: 360, height: 2400, shrunk: 480, scale: 2}
	censusShallow    = censusWindow{name: "shallow", width: 1280, height: 1600, shrunk: 480}
	// The Chinese text, in the narrowest window, at 200 % in it, and in the widest: its words are not as long as the English
	// ones, and it has no spaces to break at, which is another way to be too wide for a room. They are held to the budgets of the
	// layout, and not to those of the copy, which count the words as English has them (kit.Census.OverLayout).
	censusNarrowZh      = censusWindow{name: "narrow-zh", width: 360, height: 2400, language: settings.LanguageSimplifiedChinese}
	censusNarrowLargeZh = censusWindow{name: "narrow-200-zh", width: 360, height: 2400, scale: 2, language: settings.LanguageSimplifiedChinese}
	censusWideZh        = censusWindow{name: "wide-zh", width: 1280, height: 1600, language: settings.LanguageSimplifiedChinese}
)

// censusAddresses are the addresses that the computer of the census has on its networks: one of IPv4 and one of IPv6, of the
// ranges that are kept for documentation. A computer at home has both, and an IPv6 address is the longest word that a screen
// of the interface has to hold, 36 characters with no space in them.
var censusAddresses = []string{"192.0.2.17", "2001:db8:85a3:8d3:1319:8a2e:370:7334"}

// chooseAddress picks an address of the dialog of the pairing: its segment where they fit the dialog, or, where they do not and
// are a dropdown that shows the first, its entry in the menu.
func (ta *testApp) chooseAddress(address string) {
	ta.t.Helper()
	if _, ok := ta.find(kit.RoleButton, address); ok {
		ta.button(address)
		return
	}
	ta.click(kit.RoleButton, censusAddresses[0])
	ta.click(kit.RoleMenuItem, address)
}

// censusScene is a screen that the census takes.
type censusScene struct {
	name string
	// java gives the machine Java next to Node.js, and missing takes vfox away.
	java, missing bool
	// tall says that the page is taller than the windows of the census are, as the settings are in the narrowest, and the lists of
	// the plugins and of the distributions of Java: it is drawn in a window as tall as it is (censusWindow.heightFor), so that what
	// it has below the fold is drawn and counted, too.
	tall bool
	// modal says that the scene ends in a dialog or in a menu, which the least windows hold to the budgets, and whose content is drawn at
	// its end in them, too.
	modal bool
	// tooltip says that the scene shows a tooltip, which is an overlay as a menu is: the least windows are drawn, too.
	tooltip bool
	// toEnd says that the screen has a list that scrolls, whose end is drawn and counted, too: its rows are those that a person
	// reaches by scrolling, and the last of them is not the first.
	toEnd bool
	// endless says that the page has a list that has more rows than any window shows (the versions of Node.js, a thousand of them),
	// which is as tall as its window by what it is: the census is of the rows that it shows (kit.Census.Scrolls).
	endless bool
	// windows are the windows to draw it in; every window of the census by default.
	windows []censusWindow
	// reach goes to the screen.
	reach func(ta *testApp)
	// open, if the scene has it, opens the menu that the scene is of, on the screen that reach went to, in the window as it is by
	// then: a menu is anchored to what opens it, and a person presses what they can reach, scrolling to it first.
	open func(ta *testApp)
}

func (s censusScene) in() []censusWindow {
	if s.windows != nil {
		return s.windows
	}
	windows := []censusWindow{
		censusWide, censusDark, censusMedium, censusCompact, censusNarrow,
		censusNarrowMid, censusNarrowMore, censusCompactMore, censusNarrowLarge, censusMediumLarge, censusWideLarge,
		censusNarrowZh, censusNarrowLargeZh, censusWideZh,
	}
	if s.modal || s.tooltip {
		windows = append(windows, censusLeast, censusLeastLarge, censusShallow)
	}
	return windows
}

// optionsOf opens the options of a plugin, with the configuration of vfox that its version files need.
func optionsOf(plugin string) func(ta *testApp) {
	return func(ta *testApp) {
		ta.button("SDKs")
		ta.item(plugin)
		ta.openTab("Options")
		ta.until("the configuration", func() bool { return ta.app.model.VfoxConfig.HasValue() })
		ta.settle()
	}
}

var censusScenes = []censusScene{
	{name: "home", reach: func(ta *testApp) {}},
	{name: "sdks", reach: func(ta *testApp) { ta.button("SDKs") }},
	{name: "installed", reach: func(ta *testApp) { ta.button("SDKs"); ta.item("nodejs") }},
	{name: "available", endless: true, toEnd: true, reach: func(ta *testApp) { ta.openAvailable("nodejs") }},
	{name: "java-available", java: true, tall: true, reach: func(ta *testApp) { ta.openAvailable("java") }},
	{name: "java-choose", java: true, modal: true, toEnd: true, reach: func(ta *testApp) { ta.openAvailable("java"); ta.button("Distribution: OpenJDK") }},
	{name: "java-install-name", java: true, modal: true, reach: func(ta *testApp) { ta.openAvailable("java"); ta.addMore("Install by name…") }},
	{name: "link-manual", modal: true, reach: func(ta *testApp) { ta.openAvailable("nodejs"); ta.addMore("Link existing…") }},
	{name: "options", tall: true, reach: optionsOf("nodejs")},
	{name: "options-java", java: true, tall: true, reach: optionsOf("java")},
	{name: "tooltip", tooltip: true, reach: func(ta *testApp) { ta.button("SDKs"); ta.item("nodejs") }, open: func(ta *testApp) {
		ta.hover(kit.RoleButton, "More ways to use this version")
	}},
	{name: "alias-set", modal: true, reach: func(ta *testApp) { optionsOf("nodejs")(ta); ta.button("Name alias…") }},
	{name: "mirror", modal: true, reach: func(ta *testApp) { optionsOf("nodejs")(ta); ta.button("Change") }},
	{name: "use-in-project", modal: true, reach: func(ta *testApp) {
		ta.button("SDKs")
		ta.item("nodejs")
		ta.click(kit.RoleButton, "More ways to use this version")
		ta.click(kit.RoleMenuItem, "Use in a project…")
	}},
	{name: "remove-plugin", modal: true, reach: func(ta *testApp) {
		ta.button("SDKs")
		ta.item("nodejs")
		ta.click(kit.RoleButton, "More actions")
		ta.click(kit.RoleMenuItem, "Remove plugin")
	}},
	{name: "menu-use", modal: true, reach: func(ta *testApp) { ta.button("SDKs"); ta.item("nodejs") }, open: pressing(kit.RoleButton, "More ways to use this version")},
	{name: "menu-actions", modal: true, reach: func(ta *testApp) { ta.button("SDKs"); ta.item("nodejs") }, open: pressing(kit.RoleButton, "More actions")},
	{name: "menu-add", modal: true, reach: func(ta *testApp) { ta.openAvailable("nodejs") }, open: pressing(kit.RoleButton, "More ways to add")},
	{name: "plugins", tall: true, reach: func(ta *testApp) {
		ta.button("Plugins")
		ta.until("the registry", func() bool { return ta.app.model.Registry.HasValue() })
		ta.settle()
	}},
	{name: "tasks", reach: func(ta *testApp) {
		ta.button("SDKs")
		ta.item("nodejs")
		ta.button("Use")
		ta.until("the use task", func() bool { return len(ta.app.model.Tasks.Tasks()) == 1 })
		// A task that ends says so in a snack bar, which stays four seconds above the bar of the navigation, and a process of
		// the fixtures ends when it ends: whether the snack bar was on the page when the census was taken depended on that, and one
		// run of CI had it (a text 22 dp above the bottom of the room: a page that is cut). The scene waits for the end of the
		// task and for the snack bar, and lets the clock go past it.
		ta.until("the use task to end", func() bool {
			tasks := ta.app.model.Tasks.Tasks()
			return len(tasks) == 1 && tasks[0].Status() == state.TaskSucceeded
		})
		for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); ta.frame() {
			said := false
			for _, n := range ta.nodes() {
				said = said || (n.Role == kit.RoleText && strings.HasSuffix(n.Text, ": done"))
			}
			if said {
				break
			}
			time.Sleep(5 * time.Millisecond)
		}
		ta.tester.Advance(5 * time.Second)
		ta.settle()
		ta.button("Tasks")
	}},
	{name: "task-output", reach: func(ta *testApp) {
		// What vfox prints names links and paths, which are one word each, longer than a window of 360 dp.
		process := testsupport.NewFakeRunningProcess()
		ta.runner.OnStart("install --yes maven@3.9.16", func() *testsupport.FakeRunningProcess { return process })
		ta.t.Cleanup(func() { process.Exit(0) })
		command, err := ta.app.model.Cli.Install("maven", "3.9.16")
		if err != nil {
			ta.t.Fatal(err)
		}
		running := ta.app.model.Run(command)
		ta.button("Tasks")
		process.WriteStdout("Downloading https://repo.maven.apache.org/maven2/org/apache/maven/apache-maven/3.9.16/apache-maven-3.9.16-bin.tar.gz\n")
		process.WriteStdout("Unpacking to /home/someone/.version-fox/cache/maven/v-3.9.16/maven-3.9.16/apache-maven-3.9.16\n")
		ta.until("the output", func() bool { return len(running.Log()) >= 2 })
		ta.settle()
		ta.buttonNear("Install maven@3.9.16", "Show output")
	}},
	{name: "settings", tall: true, reach: func(ta *testApp) { ta.button("Settings") }},
	{name: "settings-web", tall: true, reach: func(ta *testApp) {
		ta.updateSettings(func(s *settings.AppSettings) {
			s.WebInterface, s.WebRemote, s.WebPort = true, true, testsupport.FreePort(ta.t)
		})
		ta.waitFor("other devices", func(status pair.Status) bool { return status.Remote })
		ta.openSettings()
	}},
	{name: "pairing", modal: true, reach: func(ta *testApp) {
		ta.updateSettings(func(s *settings.AppSettings) {
			s.WebInterface, s.WebRemote, s.WebPort = true, true, testsupport.FreePort(ta.t)
		})
		ta.waitFor("other devices", func(status pair.Status) bool { return status.Remote })
		ta.openSettings()
		ta.button("Show code")
		ta.mustText("Pair a phone")
		// The IPv6 address is the longest word of the screen: the choice, and the link of the code, are in it.
		ta.chooseAddress(censusAddresses[1])
	}},
	{name: "vfox-missing", missing: true, reach: func(ta *testApp) {}},
}

// censusApp is the application in a window, at the screen before the scene goes to it.
func censusApp(t *testing.T, scene censusScene, window censusWindow) *testApp {
	t.Helper()
	runner := testsupport.FixtureRunner()
	if scene.java {
		runner = javaRunner()
	}
	runner.SetMissingExecutable(scene.missing)
	window.height = window.heightFor(scene)
	preferences := settings.DefaultSettings()
	if window.dark {
		preferences.Theme = settings.ThemeDark
	}
	ta := newTestApp(t, runner, window.width, window.height, settings.NewMemorySettingsStore(preferences))
	ta.endless = scene.endless
	ta.setAddresses(censusAddresses...)
	if window.scale > 0 {
		ta.tester.SetPreferences(mui.Preferences{ReduceMotion: true, TextScale: window.scale})
	}
	if scene.missing {
		ta.app.model.Start()
		ta.until("vfox to be missing", func() bool { return ta.app.model.Vfox.Availability() == state.VfoxNotFound })
		ta.settle()
		return ta
	}
	ta.start()
	return ta
}

// takeScene is the application at the screen that a scene goes to, in the window: in the language of the window, which is
// changed where the scene has reached the screen, as the scenes are told by the words of English.
func takeScene(t *testing.T, scene censusScene, window censusWindow) *testApp {
	t.Helper()
	ta := censusApp(t, scene, window)
	scene.reach(ta)
	if window.shrunk > 0 {
		ta.tester.SetSize(window.width, window.shrunk)
		ta.settle()
		ta.settle()
	}
	if scene.open != nil {
		scene.open(ta)
	}
	if window.language != "" {
		ta.updateSettings(func(s *settings.AppSettings) { s.Language = window.language })
		ta.settle()
	}
	return ta
}

// pressing is what opens a menu: the scene scrolls to the button, as a person does, and presses it.
func pressing(role kit.Role, name string) func(ta *testApp) {
	return func(ta *testApp) {
		ta.t.Helper()
		ta.reveal(role, name)
		ta.click(role, name)
	}
}

// hover rests the pointer on the button, as a person does, and waits for its tooltip.
func (ta *testApp) hover(role kit.Role, name string) {
	ta.t.Helper()
	ta.reveal(role, name)
	n, ok := ta.find(role, name)
	if !ok || !n.Visible() {
		ta.t.Fatalf("%s %q is not drawn, to rest the pointer on it (%q)", role, name, ta.texts())
	}
	ta.tester.Move(n.Rect.X+n.Rect.W/2, n.Rect.Y+n.Rect.H/2)
	ta.settle()
	ta.settle()
}

// reveal scrolls the page, by what a wheel does, until the node is wholly in the room of the page, which the bar of the navigation takes
// the foot of; the pointer is taken away after it, as it is by scrollToEnd. What is not on the screen is not in the registry, so a node
// that is not found is looked for by a step of a quarter of the window; one that is found is brought into the room by the distance that
// it is out of it, and no more, where a step would take a row of a list that is built out of it.
func (ta *testApp) reveal(role kit.Role, name string) {
	ta.t.Helper()
	img := ta.tester.Image()
	width, height := float32(img.Bounds().Dx()), float32(img.Bounds().Dy())
	room := height
	if ta.app.env.Compact {
		room -= float32(kit.NavBarHeight)
	}
	defer func() {
		ta.tester.Move(-1, -1)
		ta.settle()
	}()
	for range 200 {
		n, ok := ta.find(role, name)
		if ok && n.Visible() {
			switch {
			case n.Rect.Y >= 0 && n.Rect.Y+n.Rect.H <= room:
				return
			case n.Rect.Y < 0:
				ta.tester.Scroll(width/2, height/2, 0, n.Rect.Y-8)
			default:
				ta.tester.Scroll(width/2, height/2, 0, n.Rect.Y+n.Rect.H-room+8)
			}
			ta.settle()
			continue
		}
		ta.tester.Scroll(width/2, height/2, 0, height/4)
		ta.settle()
	}
	ta.t.Fatalf("%s %q is not in the room of the page, however far it is scrolled (%q)", role, name, ta.texts())
}

// scrollToEnd scrolls what is under the middle of the window, a list or a dialog, to its end, and takes the pointer away, as a person
// does who is done: left where the wheel was, it rests on a button, and the tooltip of the button is drawn over the text that is under
// it, which is not the text that the census reads (the last row of a list was 1.4 on Windows, against the tooltip that is dark).
func (ta *testApp) scrollToEnd() {
	ta.t.Helper()
	img := ta.tester.Image()
	ta.tester.Scroll(float32(img.Bounds().Dx())/2, float32(img.Bounds().Dy())/2, 0, 1e6)
	ta.settle()
	ta.tester.Move(-1, -1)
	ta.settle()
	ta.settle()
}

// census takes the census of the screen that the window shows.
func (ta *testApp) census() kit.Census {
	ta.t.Helper()
	ta.settle()
	img := ta.tester.Image()
	census := kit.TakeCensus(ta.nodes(), mui.Rect{W: float32(img.Bounds().Dx()), H: float32(img.Bounds().Dy())}, img)
	census.Scrolls = ta.endless
	return census
}

func TestTheScreensAreWithinTheBudgets(t *testing.T) {
	if raceDetector {
		t.Skip("the census does not need the race detector, which makes it three times slower: `go test ./ui -run TestTheScreensAreWithinTheBudgets` takes it")
	}
	report := map[string]kit.Census{}
	for _, scene := range censusScenes {
		for _, window := range scene.in() {
			name := scene.name + "." + window.name
			t.Run(name, func(t *testing.T) {
				ta := takeScene(t, scene, window)
				census := ta.census()
				ta.snapshot("census." + name)
				report[name] = census
				t.Logf("%s", censusLine(census))
				if over := window.over(census); len(over) > 0 {
					t.Errorf("over the budgets of the design system:\n  %s", strings.Join(over, "\n  "))
				}
				if !scene.toEnd && (!scene.modal || window.shrunk == 0) {
					return
				}
				// The rows that a person reaches by scrolling are not the rows that the screen starts with: the end of the list, or of the
				// content of a dialog in a window that is shorter than it, is drawn, and counted, too, and the census of it is of what is
				// shown (kit.Census.Scrolls).
				ta.scrollToEnd()
				end := ta.census()
				end.Scrolls = true
				ta.snapshot("census." + name + ".end")
				report[name+".end"] = end
				t.Logf("at the end: %s", censusLine(end))
				if over := window.over(end); len(over) > 0 {
					t.Errorf("at the end of its list, over the budgets of the design system:\n  %s", strings.Join(over, "\n  "))
				}
			})
		}
	}
	writeCensus(t, report)
}

// The sizes of text between 100 % and 200 % that the census does not take, in the four widths of the windows, in English and in
// Chinese: what holds at 200 % holds below it is a thing that the layouts do not promise, where a row wraps at one width and a
// group of segments is a dropdown at another, and the Chinese text has its own: no spaces to break at, and a word that is a run of
// characters. This is how they are looked at. It takes about as long as the census does, four times, so only a run that asks for it
// (VFOX_PLUS_CENSUS_SWEEP=1) takes it; the census holds the sizes where the room was the least.
func TestTheScreensAreWithinTheBudgetsAtEverySizeOfText(t *testing.T) {
	if os.Getenv("VFOX_PLUS_CENSUS_SWEEP") == "" {
		t.Skip("VFOX_PLUS_CENSUS_SWEEP=1 takes the sizes of text between those of the census: it takes a few minutes")
	}
	if raceDetector {
		t.Skip("the census does not need the race detector, which makes it three times slower")
	}
	for _, base := range []censusWindow{censusNarrow, censusCompact, censusMedium, censusWide} {
		for _, scale := range []float32{1.15, 1.3, 1.5, 1.75} {
			for _, language := range []settings.LanguagePreference{"", settings.LanguageSimplifiedChinese} {
				window := base
				window.scale, window.language = scale, language
				window.name = fmt.Sprintf("%s-%d", base.name, int(scale*100+0.5))
				if language != "" {
					window.name += "-zh"
				}
				for _, scene := range censusScenes {
					t.Run(scene.name+"."+window.name, func(t *testing.T) {
						ta := takeScene(t, scene, window)
						if over := window.over(ta.census()); len(over) > 0 {
							t.Errorf("over the budgets of the design system:\n  %s", strings.Join(over, "\n  "))
						}
					})
				}
			}
		}
	}
}

// fontOf is the family that the system gives for a name, for the log of a test; on a platform that has no fontconfig, what the
// name stands for in the text engine of the platform.
func fontOf(name string) string {
	// Only Linux asks fontconfig. A runner of macOS has one too (Homebrew's), and it says Verdana, where Core Text gives the
	// system font; DirectWrite has its own list for sans-serif and monospace: the engines of Windows and macOS choose.
	if runtime.GOOS == "linux" {
		if out, err := exec.Command("fc-match", "-f", "%{family}", name).Output(); err == nil { //nolint:gosec // The names are the two that the test asks for.
			return strings.TrimSpace(string(out))
		}
	}
	return name + ", as the text engine of " + runtime.GOOS + " has it"
}

// The screens hold in the fonts that the system chooses where it has not those of the interface: nothing of the layout is
// written for the width of Ubuntu Sans. The scenes are taken in the narrowest window, at 100 % and at 200 %, and in the widest,
// with `sans-serif` and `monospace`, which are Noto Sans or DejaVu Sans, as they are installed.
func TestTheScreensAreWithinTheBudgetsInTheFontsThatTheSystemChooses(t *testing.T) {
	if raceDetector {
		t.Skip("the census does not need the race detector, which makes it three times slower: `go test ./ui -run TestTheScreensAreWithinTheBudgets` takes it")
	}
	interfaceFamilies, monospaceFamilies := kit.UIFamilies, kit.MonoFamilies
	kit.UIFamilies, kit.MonoFamilies = "sans-serif", "monospace"
	t.Cleanup(func() { kit.UIFamilies, kit.MonoFamilies = interfaceFamilies, monospaceFamilies })
	t.Logf("the interface is set in %s, and the monospaced text in %s", fontOf("sans-serif"), fontOf("monospace"))
	for _, scene := range censusScenes {
		for _, window := range []censusWindow{censusNarrow, censusNarrowLarge, censusWide} {
			t.Run(scene.name+"."+window.name, func(t *testing.T) {
				ta := takeScene(t, scene, window)
				if over := ta.census().Over(); len(over) > 0 {
					t.Errorf("over the budgets of the design system:\n  %s", strings.Join(over, "\n  "))
				}
			})
		}
	}
}

// censusLine is a census in one line.
func censusLine(c kit.Census) string {
	return fmt.Sprintf("%d words, %d sentences, %d small lines, %d rows with two actions, walls %v, contrast %d/%d, targets %d/%d, off scale %v, overflow %v",
		c.Words, c.Prose.Blocks, c.Helpers, c.RowsWithTwoActions, c.Wallpaper, c.Contrast.Failing, c.Contrast.Measured, c.Targets.Under24, c.Targets.Total, c.OffScale, c.Overflow)
}

// writeCensus saves the census of every screen to the directory that VFOX_PLUS_CENSUS names, if it is set.
func writeCensus(t *testing.T, report map[string]kit.Census) {
	t.Helper()
	dir := os.Getenv("VFOX_PLUS_CENSUS")
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // The directory is the choice of whoever takes the census.
		t.Fatal(err)
	}
	names := make([]string, 0, len(report))
	for name := range report {
		names = append(names, name)
	}
	sort.Strings(names)
	var table strings.Builder
	fmt.Fprintf(&table, "%-34s %6s %6s %6s %6s %6s %6s %6s %6s\n", "screen", "words", "sent.", "small", "2 act.", "walls", "contr.", "<24dp", "off")
	var words, sentences, small, actions, walls, failing, under, off int
	for _, name := range names {
		c := report[name]
		data, err := json.MarshalIndent(c, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name+".json"), append(data, '\n'), 0o600); err != nil { //nolint:gosec // Likewise.
			t.Fatal(err)
		}
		offScale := 0
		for _, n := range c.OffScale {
			offScale += n
		}
		fmt.Fprintf(&table, "%-34s %6d %6d %6d %6d %6d %6d %6d %6d\n", name, c.Words, c.Prose.Blocks, c.Helpers, c.RowsWithTwoActions, len(c.Wallpaper), c.Contrast.Failing, c.Targets.Under24, offScale)
		words, sentences, small, actions, walls = words+c.Words, sentences+c.Prose.Blocks, small+c.Helpers, actions+c.RowsWithTwoActions, walls+len(c.Wallpaper)
		failing, under, off = failing+c.Contrast.Failing, under+c.Targets.Under24, off+offScale
	}
	fmt.Fprintf(&table, "%-34s %6d %6d %6d %6d %6d %6d %6d %6d\n", "all", words, sentences, small, actions, walls, failing, under, off)
	if err := os.WriteFile(filepath.Join(dir, "summary.txt"), []byte(table.String()), 0o600); err != nil { //nolint:gosec // Likewise.
		t.Fatal(err)
	}
}
