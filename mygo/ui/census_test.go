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
	"sort"
	"strings"
	"testing"

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
)

// censusAddresses are the addresses that the computer of the census has on its networks: one of IPv4 and one of IPv6, of the
// ranges that are kept for documentation. A computer at home has both, and an IPv6 address is the longest word that a screen
// of the interface has to hold, 36 characters with no space in them.
var censusAddresses = []string{"192.0.2.17", "2001:db8:85a3:8d3:1319:8a2e:370:7334"}

// censusScene is a screen that the census takes.
type censusScene struct {
	name string
	// java gives the machine Java next to Node.js, and missing takes vfox away.
	java, missing bool
	// windows are the windows to draw it in; every window of the census by default.
	windows []censusWindow
	// reach goes to the screen.
	reach func(ta *testApp)
}

func (s censusScene) in() []censusWindow {
	if s.windows != nil {
		return s.windows
	}
	return []censusWindow{
		censusWide, censusDark, censusMedium, censusCompact, censusNarrow,
		censusNarrowMid, censusNarrowMore, censusCompactMore, censusNarrowLarge, censusMediumLarge, censusWideLarge,
	}
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
	{name: "available", reach: func(ta *testApp) { ta.openAvailable("nodejs") }},
	{name: "java-available", java: true, reach: func(ta *testApp) { ta.openAvailable("java") }},
	{name: "java-choose", java: true, reach: func(ta *testApp) { ta.openAvailable("java"); ta.button("Distribution: OpenJDK") }},
	{name: "java-install-name", java: true, reach: func(ta *testApp) { ta.openAvailable("java"); ta.addMore("Install by name…") }},
	{name: "link-manual", reach: func(ta *testApp) { ta.openAvailable("nodejs"); ta.addMore("Link existing…") }},
	{name: "options", reach: optionsOf("nodejs")},
	{name: "options-java", java: true, reach: optionsOf("java")},
	{name: "alias-set", reach: func(ta *testApp) { optionsOf("nodejs")(ta); ta.button("Name alias…") }},
	{name: "mirror", reach: func(ta *testApp) { optionsOf("nodejs")(ta); ta.button("Change") }},
	{name: "use-in-project", reach: func(ta *testApp) {
		ta.button("SDKs")
		ta.item("nodejs")
		ta.click(kit.RoleButton, "More ways to use this version")
		ta.click(kit.RoleMenuItem, "Use in a project…")
	}},
	{name: "remove-plugin", reach: func(ta *testApp) {
		ta.button("SDKs")
		ta.item("nodejs")
		ta.click(kit.RoleButton, "More actions")
		ta.click(kit.RoleMenuItem, "Remove plugin")
	}},
	{name: "plugins", reach: func(ta *testApp) {
		ta.button("Plugins")
		ta.until("the registry", func() bool { return ta.app.model.Registry.HasValue() })
		ta.settle()
	}},
	{name: "tasks", reach: func(ta *testApp) {
		ta.button("SDKs")
		ta.item("nodejs")
		ta.button("Use")
		ta.until("the use task", func() bool { return len(ta.app.model.Tasks.Tasks()) == 1 })
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
	{name: "settings", reach: func(ta *testApp) { ta.button("Settings") }},
	{name: "settings-web", reach: func(ta *testApp) {
		ta.updateSettings(func(s *settings.AppSettings) {
			s.WebInterface, s.WebRemote, s.WebPort = true, true, testsupport.FreePort(ta.t)
		})
		ta.waitFor("other devices", func(status pair.Status) bool { return status.Remote })
		ta.openSettings()
	}},
	{name: "pairing", reach: func(ta *testApp) {
		ta.updateSettings(func(s *settings.AppSettings) {
			s.WebInterface, s.WebRemote, s.WebPort = true, true, testsupport.FreePort(ta.t)
		})
		ta.waitFor("other devices", func(status pair.Status) bool { return status.Remote })
		ta.openSettings()
		ta.button("Show code")
		ta.mustText("Pair a phone")
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
	preferences := settings.DefaultSettings()
	if window.dark {
		preferences.Theme = settings.ThemeDark
	}
	ta := newTestApp(t, runner, window.width, window.height, settings.NewMemorySettingsStore(preferences))
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

// census takes the census of the screen that the window shows.
func (ta *testApp) census() kit.Census {
	ta.t.Helper()
	ta.settle()
	img := ta.tester.Image()
	return kit.TakeCensus(ta.nodes(), mui.Rect{W: float32(img.Bounds().Dx()), H: float32(img.Bounds().Dy())}, img)
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
				ta := censusApp(t, scene, window)
				scene.reach(ta)
				census := ta.census()
				ta.snapshot("census." + name)
				report[name] = census
				t.Logf("%s", censusLine(census))
				if over := census.Over(); len(over) > 0 {
					t.Errorf("over the budgets of the design system:\n  %s", strings.Join(over, "\n  "))
				}
			})
		}
	}
	writeCensus(t, report)
}

// The sizes of text between 100 % and 200 % that the census does not take, in the four widths of the windows: what holds at 200 %
// holds below it is a thing that the layouts do not promise, where a row wraps at one width and a group of segments is a dropdown at
// another, and this is how it is looked at. It takes about as long as the census does, twice, so only a run that asks for it
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
			window := base
			window.scale, window.name = scale, fmt.Sprintf("%s-%d", base.name, int(scale*100+0.5))
			for _, scene := range censusScenes {
				t.Run(scene.name+"."+window.name, func(t *testing.T) {
					ta := censusApp(t, scene, window)
					scene.reach(ta)
					if over := ta.census().Over(); len(over) > 0 {
						t.Errorf("over the budgets of the design system:\n  %s", strings.Join(over, "\n  "))
					}
				})
			}
		}
	}
}

// fontOf is the family that the system gives for a name, for the log of a test.
func fontOf(name string) string {
	out, err := exec.Command("fc-match", "-f", "%{family}", name).Output() //nolint:gosec // The names are the two that the test asks for.
	if err != nil {
		return name + " (fc-match is not there)"
	}
	return strings.TrimSpace(string(out))
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
				ta := censusApp(t, scene, window)
				scene.reach(ta)
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
