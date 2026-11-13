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
	"cmp"
	"fmt"
	"math"
	"os"
	"runtime"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	mui "github.com/egoist/mygo/ui"

	"github.com/version-fox/vfox-plus/mygo/ui/kit"
)

// The census in the widest fonts that the system has. The layout is not written for a font, but that is a thing to be shown and
// not to be said: a machine has the fonts that it has, and a font that is wider than the one that the census draws in is wider
// than a layout that was never tried in it. Nobody can name the widest font of every machine, so the test asks the machine: it
// measures the families that systems and desktops set their interface in, in the text engine itself, and draws the scenes in the
// widest that it has. (A font of handwriting, of symbols or of emoji is a font that no screen of the application is set in: the
// first draft took the widest of every family that the engines list, and found Segoe Script, Zapfino and Apple Color Emoji.)

// familyProbes are the lines that a family is measured by: a line of English, which is how wide the family sets text, and
// three of one letter, which tell the wide letters of a font from its narrow ones, and one font from another.
var familyProbes = [...]string{
	"The quick brown fox jumps over the lazy dog 0123456789",
	"WWWWWWWW",
	"iiiiiiii",
	"mmmmmmmm",
}

// measuredFamily is a font family of the system, as the text engine sets the probes in it.
type measuredFamily struct {
	name string
	// widths are those of the probes, in dp at 14 dp.
	widths [len(familyProbes)]float32
	// monospaced is what the engine says of the family: that its regular font gives every character the same advance.
	monospaced bool
}

// width is how wide the family sets a line of English.
func (f measuredFamily) width() float32 { return f.widths[0] }

// proportional is whether the family sets a font of text: a wide letter wider than a narrow one, which a font of symbols, whose
// letters are all the same, does not, and nor does a monospaced one.
func (f measuredFamily) proportional() bool { return !f.monospaced && f.widths[1] > 1.5*f.widths[2] }

// alike is whether two families set every probe the same: they are the same font to the screens, as a family that has no Latin
// letters is, which the engine sets in the font that it chooses for them.
func (f measuredFamily) alike(other measuredFamily) bool {
	for i, width := range f.widths {
		if math.Abs(float64(width-other.widths[i])) > 0.01 {
			return false
		}
	}
	return true
}

// widest picks the n proportional families that set English the widest. It leaves out those that are set like one of the defaults,
// which are the fonts that the other tests take, and takes one of those that are set alike (the shortest name of them, which is the
// font itself and not one that its glyphs are borrowed by). The monospaced fonts are not told apart: a character of every one of
// them is 0.5 to 0.62 em wide, and the one that the system gives for `monospace` is among the widest.
func widest(families []measuredFamily, defaults []measuredFamily, n int) []measuredFamily {
	var candidates []measuredFamily
	for _, f := range families {
		if f.proportional() && !slices.ContainsFunc(defaults, f.alike) {
			candidates = append(candidates, f)
		}
	}
	slices.SortFunc(candidates, func(a, b measuredFamily) int {
		return cmp.Or(cmp.Compare(b.width(), a.width()), cmp.Compare(len(a.name), len(b.name)), strings.Compare(a.name, b.name))
	})
	var chosen []measuredFamily
	for _, f := range candidates {
		if len(chosen) == n {
			break
		}
		if !slices.ContainsFunc(chosen, f.alike) {
			chosen = append(chosen, f)
		}
	}
	return chosen
}

func TestTheWidestFamiliesAreChosenByWhatTheyMeasure(t *testing.T) {
	family := func(name string, line, wide, narrow, mid float32, monospaced bool) measuredFamily {
		return measuredFamily{name: name, widths: [len(familyProbes)]float32{line, wide, narrow, mid}, monospaced: monospaced}
	}
	standard := family("Standard", 400, 80, 20, 70, false)
	all := []measuredFamily{
		standard,
		family("Wide", 480, 90, 24, 80, false),
		family("Wider", 520, 96, 26, 86, false),
		// The same font under a longer name, and a family of a script that has no Latin letters, which is set in the font of the
		// system as the default is: neither is another font.
		family("Wider, with its glyphs borrowed", 520, 96, 26, 86, false),
		family("Script", 400, 80, 20, 70, false),
		family("Narrow", 360, 70, 16, 60, false),
		// A font of symbols is the widest line there is, and has no wide letters and narrow ones; nor has a monospaced font.
		family("Symbols", 700, 100, 100, 100, false),
		family("Code", 800, 56, 56, 56, true),
	}
	names := func(families []measuredFamily) []string {
		var out []string
		for _, f := range families {
			out = append(out, f.name)
		}
		return out
	}
	if got, want := names(widest(all, []measuredFamily{standard}, 5)), []string{"Wider", "Wide", "Narrow"}; !slices.Equal(got, want) {
		t.Errorf("the proportional families, widest first, one of each font, none of the default or of the symbols: %q, want %q", got, want)
	}
	if got, want := names(widest(all, []measuredFamily{standard}, 1)), []string{"Wider"}; !slices.Equal(got, want) {
		t.Errorf("the widest of them: %q, want %q", got, want)
	}
	if got := widest(all, []measuredFamily{standard, family("Wider again", 520, 96, 26, 86, false)}, 5); len(got) != 2 || got[0].name != "Wide" {
		t.Errorf("a family that is set as a default is, is not one of them: %q", names(got))
	}
}

// uiFamilies are the families that operating systems and desktops set their interface in, and the ones that sans-serif commonly
// stands for: the fonts that a person's screens are drawn in. Verdana is the widest of them on Windows and on macOS, and DejaVu
// Sans on Linux.
var uiFamilies = []string{
	"Verdana", "Tahoma", "Trebuchet MS", "Segoe UI", "Arial", "Calibri", "Geneva", "Lucida Grande", "Helvetica Neue", "Helvetica",
	"DejaVu Sans", "Noto Sans", "Open Sans", "Roboto", "Inter", "Adwaita Sans", "Cantarell", "Liberation Sans", "Lato", "FreeSans",
	"Droid Sans", "Fira Sans", "Source Sans 3", "Oxygen", "Ubuntu Sans", "Ubuntu",
}

// systemFamilies are the families of uiFamilies that the system has, measured, and what the engine gives for sans-serif.
type systemFamilies struct {
	all       []measuredFamily
	sansSerif measuredFamily
}

// measureSystemFamilies measures the families of uiFamilies that the text engine lists, in the text engine.
func measureSystemFamilies(t *testing.T) systemFamilies {
	t.Helper()
	var system systemFamilies
	measured := false
	list := mui.FontFamilies()
	tester := mui.NewTester(func(c *mui.Context) {
		if measured {
			return
		}
		measured = true
		measure := func(name string, monospaced bool) measuredFamily {
			f := measuredFamily{name: name, monospaced: monospaced}
			for i, probe := range familyProbes {
				f.widths[i], _ = c.MeasureText(0, mui.Span{Text: probe, Font: name, Size: 14, Weight: 400})
			}
			return f
		}
		for _, family := range list {
			if slices.Contains(uiFamilies, family.Name) {
				system.all = append(system.all, measure(family.Name, family.Monospaced))
			}
		}
		system.sansSerif = measure("sans-serif", false)
	}, 400, 100)
	tester.Frame()
	if !measured {
		t.Fatal("the text engine measured nothing")
	}
	return system
}

// The scenes in the narrowest window at 100 % and at 200 %, and in the widest, in the families that are the widest of those that
// the system has, with `monospace` beside each; and in Chinese, in the widest family of them, where the words of a screen that are
// not Chinese (the names of the SDKs, the versions, the paths) are set in it, and its Han characters in the font that the system
// chooses for them. The family that the system gives for `sans-serif` is that of
// TestTheScreensAreWithinTheBudgetsInTheFontsThatTheSystemChooses, and is left out here.
//
// It takes the time of that test for each family, so a run that asks for it takes it (VFOX_PLUS_CENSUS_SWEEP=1), as CI does on
// every platform.
func TestTheScreensAreWithinTheBudgetsInTheWidestFontsThatTheSystemHas(t *testing.T) {
	if os.Getenv("VFOX_PLUS_CENSUS_SWEEP") == "" {
		t.Skip("VFOX_PLUS_CENSUS_SWEEP=1 takes the widest fonts that the system has: the time of the fonts that it chooses, for each")
	}
	if raceDetector {
		t.Skip("the census does not need the race detector, which makes it three times slower")
	}
	system := measureSystemFamilies(t)
	proportional := widest(system.all, []measuredFamily{system.sansSerif}, 3)
	if len(proportional) == 0 {
		t.Fatalf("the system has none of the families that screens are set in other than the one that it gives for sans-serif: %d", len(system.all))
	}
	// Verdana is on every Windows and every Mac, and is the widest font of them that is of text: a machine that has not got it, or
	// that has fonts that are wider, is not the one that this test is told of, and the test says so, where it would have drawn the
	// screens in narrower fonts.
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		if !slices.ContainsFunc(proportional, func(f measuredFamily) bool { return f.name == "Verdana" }) {
			t.Fatalf("Verdana is the widest font of %s, and is not among those that are tried: %d", runtime.GOOS, len(proportional))
		}
	}
	against := system.sansSerif.width()
	t.Logf("a line of English is %.0f dp wide in sans-serif (%s)", against, fontOf("sans-serif"))
	for _, f := range proportional {
		t.Logf("%-28s %.0f dp, %+.0f %% of sans-serif", f.name, f.width(), (f.width()/against-1)*100)
	}
	for i, family := range proportional {
		t.Run(family.name, func(t *testing.T) {
			interfaceFamilies, monospaceFamilies := kit.UIFamilies, kit.MonoFamilies
			kit.UIFamilies, kit.MonoFamilies = family.name, "monospace"
			t.Cleanup(func() { kit.UIFamilies, kit.MonoFamilies = interfaceFamilies, monospaceFamilies })
			windows := []censusWindow{censusNarrow, censusNarrowLarge, censusWide}
			if i == 0 {
				windows = append(windows, censusNarrowZh, censusNarrowLargeZh, censusWideZh)
			}
			for _, scene := range censusScenes {
				for _, window := range windows {
					t.Run(scene.name+"."+window.name, func(t *testing.T) {
						ta := takeScene(t, scene, window)
						census := ta.census()
						if i == 0 && window.name == censusNarrowLarge.name {
							ta.snapshot(fmt.Sprintf("widest.%s.%s", scene.name, window.name))
						}
						if over := window.over(census); len(over) > 0 {
							t.Errorf("over the budgets of the design system:\n  %s", strings.Join(over, "\n  "))
						}
					})
				}
			}
		})
	}
}

// trackingFor is the tracking, in em, that makes a line of English wider by the share that is asked (0.4 is two fifths), in the font
// that the interface is set in: the width of the probe at 14 dp, and a character's worth of spacing for every character of it.
func trackingFor(t *testing.T, share float32) float32 {
	t.Helper()
	probe := familyProbes[0]
	var width float32
	measured := false
	tester := mui.NewTester(func(c *mui.Context) {
		if !measured {
			measured = true
			width, _ = c.MeasureText(0, mui.Span{Text: probe, Font: kit.UIFamilies, Size: 14, Weight: 400})
		}
	}, 400, 100)
	tester.Frame()
	if !measured || width <= 0 {
		t.Fatal("the text engine measured nothing")
	}
	return share * width / (float32(utf8.RuneCountInString(probe)) * 14)
}

// A text that is wider than the font that the screens are drawn in. A translation is: German, Finnish and Russian set the same
// words in a third to a half more characters than English does (the W3C's guidance to designers of interfaces is to leave a third,
// and more for what is short), and a font may be wider than the ones that the other tests draw in. No machine has the widest font
// there is, so the test makes the text wider by letter spacing (kit.Tracking), as the tools of pseudo-localization do, which the
// labels draw and the layouts measure; the text of the interface is made wider, whatever it says and in what font, by two fifths at every
// size of text: the W3C's guidance, and what a label of a field that has not the room at rest needs (it is set at the size that fits).
//
// It takes the time of the census for each of its windows, so a run that asks for it takes it (VFOX_PLUS_CENSUS_SWEEP=1), as CI does.
func TestTheScreensAreWithinTheBudgetsInATextThatIsWider(t *testing.T) {
	if os.Getenv("VFOX_PLUS_CENSUS_SWEEP") == "" {
		t.Skip("VFOX_PLUS_CENSUS_SWEEP=1 takes the text that is wider than the font of the interface: the time of the census, for its windows")
	}
	if raceDetector {
		t.Skip("the census does not need the race detector, which makes it three times slower")
	}
	t.Cleanup(func() { kit.Tracking = 0 })
	for _, wider := range []struct {
		window censusWindow
		share  float32
	}{
		{censusNarrow, 0.4}, {censusNarrowMore, 0.4}, {censusNarrowLarge, 0.4}, {censusNarrowLargest, 0.4},
		{censusMedium, 0.4}, {censusMediumLarge, 0.4},
		{censusWide, 0.4}, {censusWideLarge, 0.4},
		{censusNarrowZh, 0.4}, {censusWideZh, 0.4},
	} {
		kit.Tracking = trackingFor(t, wider.share)
		t.Logf("%s: a tracking of %.3f em makes a line of English %.0f %% wider in %s", wider.window.name, kit.Tracking, wider.share*100, fontOf(kit.UIFamilies))
		for _, scene := range censusScenes {
			t.Run(scene.name+"."+wider.window.name, func(t *testing.T) {
				ta := takeScene(t, scene, wider.window)
				census := ta.census()
				ta.snapshot(fmt.Sprintf("wider.%s.%s", scene.name, wider.window.name))
				if over := wider.window.over(census); len(over) > 0 {
					t.Errorf("over the budgets of the design system:\n  %s", strings.Join(over, "\n  "))
				}
			})
		}
		kit.Tracking = 0
	}
}

// measuredLabel is a label of a navigation bar, as the engine measures it and as it draws it: as wide as it wants, and in boxes of
// the width that the engine measured and a little more, where it is a single line that an ellipsis ends if the box is too narrow.
type measuredLabel struct {
	family, text string
	weight       int
	size         float32
	measured     float32
	free         mui.Rect
	boxed        [len(boxExtras)]mui.Rect
	// needs is how much wider than the measured width the box has to be for the label to be drawn as the free label is: the first
	// of boxExtras at which it is, or -1 where none is.
	needs float32
}

// boxExtras are the widths in dp, more than what the engine measured, that the boxes of a label have.
var boxExtras = [...]float32{0, 0.5, 1, 2, 4}

// rowHeight is the height of the row of a label in the picture: a whole number of dp, so that the lines of every row are drawn at the
// same fraction of a pixel and the pictures of two rows are alike where the labels are.
const rowHeight = 40

// The census says that a label is cut when it has less room than the width that the text engine measures it at (kit.Node.Natural),
// and what it trusts is that a label that has that width is not cut. On macOS, in Verdana, the label of the navigation bar "Settings"
// was drawn as "Settin..." in a bar that the census said was not cut. A width of the engine that is less than what the engine draws is
// how that could be, and this is how it is looked at: the labels of the bar, at the weights that it uses and the sizes that it has, in
// the default family and in the widest that the system has, drawn as wide as they want and in boxes of the measured width and a
// little more. The log says what the engine needs beyond what it measures, and the test fails where the width that the label has
// when it wants none is not what is measured, by more than a dp, and where a box that is 2 dp wider than the measure cuts a label.
func TestTheScreensAreWithinTheBudgetsMeasureWhatTheyDraw(t *testing.T) {
	if raceDetector {
		t.Skip("the census does not need the race detector")
	}
	system := measureSystemFamilies(t)
	families := []string{kit.UIFamilies, "sans-serif"}
	for _, f := range widest(system.all, []measuredFamily{system.sansSerif}, 3) {
		families = append(families, f.name)
	}
	var labels []*measuredLabel
	for _, family := range families {
		for _, weight := range []int{400, 500, 700} {
			for _, size := range []float32{12, 15.6, 24} {
				for _, text := range []string{"Home", "SDKs", "Plugins", "Tasks", "Settings"} {
					labels = append(labels, &measuredLabel{family: family, weight: weight, size: size, text: text})
				}
			}
		}
	}
	rows := 1 + len(boxExtras)
	tester := mui.NewTester(func(c *mui.Context) {
		mui.Column(c).Padding(8).Gap(4).Children(func() {
			for _, l := range labels {
				l.measured, _ = c.MeasureText(0, mui.Span{Text: l.text, Font: l.family, Size: l.size, Weight: l.weight})
				mui.Row(c).Height(rowHeight).Children(func() {
					mui.Text(c, l.text).Font(l.family).FontSize(l.size).FontWeight(l.weight).SingleLine().
						DrawOver(func(_ *mui.Painter, r mui.Rect) { l.free = r })
				})
				for i, extra := range boxExtras {
					mui.Row(c).Height(rowHeight).Children(func() {
						mui.Text(c, l.text).Font(l.family).FontSize(l.size).FontWeight(l.weight).SingleLine().Width(l.measured + extra).
							DrawOver(func(_ *mui.Painter, r mui.Rect) { l.boxed[i] = r })
					})
				}
			}
		})
	}, 240, (rowHeight+4)*rows*len(labels)+16)
	tester.Frame()
	img := tester.Image()
	// ink is the picture of the row that a rect is in: as wide as the label may be, and as high as the row.
	ink := func(r mui.Rect, width int) []int {
		var pixels []int
		top := int(math.Floor(float64(r.Y))) / (rowHeight + 4) * (rowHeight + 4)
		for y := top; y < top+rowHeight; y++ {
			for x := 0; x < width; x++ {
				red, green, blue, _ := img.At(int(r.X)+x, y).RGBA()
				pixels = append(pixels, int(red>>8), int(green>>8), int(blue>>8))
			}
		}
		return pixels
	}
	differ := func(a, b []int) int {
		count := 0
		for i := range min(len(a), len(b)) {
			if d := a[i] - b[i]; d > 24 || d < -24 {
				count++
			}
		}
		return count
	}
	var mismatched, cutAtMeasure, cutAtTwo int
	var needing []string
	var widest, narrowest, most float32
	narrowest = 1e9
	for _, l := range labels {
		width := int(math.Ceil(float64(l.measured+boxExtras[len(boxExtras)-1]))) + 4
		free := ink(l.free, width)
		l.needs = -1
		for i, extra := range boxExtras {
			if differ(free, ink(l.boxed[i], width)) == 0 {
				l.needs = extra
				break
			}
		}
		delta := l.free.W - l.measured
		widest, narrowest = max(widest, delta), min(narrowest, delta)
		if l.needs > 0 || l.needs < 0 {
			cutAtMeasure++
			most = max(most, l.needs)
			if len(needing) < 12 {
				needing = append(needing, fmt.Sprintf("%s %d %.1f %q: measured %.2f, drawn free %.2f, needs %+.1f", l.family, l.weight, l.size, l.text, l.measured, l.free.W, l.needs))
			}
		}
		if delta > 1 || delta < -1 {
			mismatched++
			t.Errorf("%-24s %3d %5.1f %-9s the engine measures %.2f dp, and the label that is as wide as it wants is %.2f dp", l.family, l.weight, l.size, l.text, l.measured, l.free.W)
		}
		if l.needs < 0 || l.needs > 2 {
			cutAtTwo++
			t.Errorf("%-24s %3d %5.1f %-9s a box that is 2 dp wider than the %.2f dp that the engine measured still cuts the label", l.family, l.weight, l.size, l.text, l.measured)
		}
	}
	// The label that was drawn as "Settin..." on macOS: a fifth of a window of 360 dp is 72, and the bar read its labels at 130 %, 15.6 sp.
	for _, l := range labels {
		if l.text == "Settings" && l.size == 15.6 {
			t.Logf("%q at 15.6 sp, weight %d, in %s: the engine measures %.2f dp, and a destination of the bar of a window of 360 dp has 72", l.text, l.weight, l.family, l.measured)
		}
	}
	t.Logf("%d labels in %d families, at 3 weights and 3 sizes: a label that wants no room is %.2f to %.2f dp wider than the engine measures it (%d by more than a dp); "+
		"a box of the measured width cuts %d of them, %.1f dp more is the most that one needs, and %d are cut by a box that is 2 dp wider",
		len(labels), len(families), narrowest, widest, mismatched, cutAtMeasure, most, cutAtTwo)
	for _, line := range needing {
		t.Logf("a box of the measured width cuts: %s", line)
	}
}
