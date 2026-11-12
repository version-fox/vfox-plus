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
// the system has, with `monospace` beside each. The family that the system gives for `sans-serif` is that of
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
			for _, scene := range censusScenes {
				for _, window := range []censusWindow{censusNarrow, censusNarrowLarge, censusWide} {
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
