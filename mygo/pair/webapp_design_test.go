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

package pair

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/version-fox/vfox-plus/mygo/core/settings"
	"github.com/version-fox/vfox-plus/mygo/ui/kit"
	"github.com/version-fox/vfox-plus/mygo/ui/mcu"
)

// The desktop interface, the web interface and the Android app are one design system
// (docs/en/dev/design-system.md): the same colours and the same springs. The desktop has
// them in code (mcu and kit), the page in CSS and the app in resources. These tests fail
// when one of them moves without the others, and they say what the others must be.

// hexOf is a colour the way the page and the resources write it.
func hexOf(argb uint32) string { return fmt.Sprintf("#%06x", argb&0xFFFFFF) }

// schemeRoles are the roles of the scheme of the default seed by the names of the web
// interface (surface-container-low), as hex.
func schemeRoles(dark bool) map[string]string {
	roles := map[string]string{}
	for _, token := range mcu.Generate(settings.DefaultSeedColor, dark).Tokens() {
		roles[token.Name] = hexOf(token.ARGB)
	}
	return roles
}

// sameRoles reports every role in which got differs from want.
func sameRoles(t *testing.T, where string, got, want map[string]string) {
	t.Helper()
	names := make([]string, 0, len(want))
	for name := range want {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		switch have, ok := got[name]; {
		case !ok:
			t.Errorf("%s has no %s", where, name)
		case !strings.EqualFold(have, want[name]):
			t.Errorf("%s: %s is %s, the scheme has %s", where, name, have, want[name])
		}
	}
	for name := range got {
		if _, ok := want[name]; !ok {
			t.Errorf("%s has %s, which is no role of the scheme", where, name)
		}
	}
}

func TestTheColoursOfThePageAreTheSchemeOfTheSeed(t *testing.T) {
	role := regexp.MustCompile(`--m3-([a-z0-9-]+):\s*(#[0-9a-fA-F]{6})`)
	for _, theme := range []string{"light", "dark"} {
		block := regexp.MustCompile(`(?s):root\[data-theme=` + theme + `\]\s*\{(.*?)\}`).FindSubmatch(webappHTML)
		if block == nil {
			t.Fatalf("the page has no colours for the %s theme", theme)
		}
		got := map[string]string{}
		for _, m := range role.FindAllSubmatch(block[1], -1) {
			got[string(m[1])] = string(m[2])
		}
		sameRoles(t, "the "+theme+" theme of the page", got, schemeRoles(theme == "dark"))
	}
}

func TestTheColoursOfTheAndroidAppAreTheSchemeOfTheSeed(t *testing.T) {
	role := regexp.MustCompile(`<color name="m3_([a-z0-9_]+)">(#[0-9a-fA-F]{6})</color>`)
	res := filepath.Join("..", "..", "android", "app", "src", "main", "res")
	for theme, dir := range map[string]string{"light": "values", "dark": "values-night"} {
		file := filepath.Join(res, dir, "colors.xml")
		data, err := os.ReadFile(file) //nolint:gosec // A file of the repository.
		if err != nil {
			t.Fatal(err)
		}
		got := map[string]string{}
		for _, m := range role.FindAllStringSubmatch(string(data), -1) {
			got[strings.ReplaceAll(m[1], "_", "-")] = m[2]
		}
		sameRoles(t, file, got, schemeRoles(theme == "dark"))
	}
}

// response is where a spring of unit mass is after t seconds when it starts at 0 and
// its end is at 1: the exact solution of the equation of the damped oscillator.
func response(s kit.Spring, t float64) float64 {
	omega := math.Sqrt(s.Stiffness)
	switch zeta := s.Damping; {
	case zeta < 1:
		damped := omega * math.Sqrt(1-zeta*zeta)
		return 1 - math.Exp(-zeta*omega*t)*(math.Cos(damped*t)+zeta*omega/damped*math.Sin(damped*t))
	case zeta == 1:
		return 1 - math.Exp(-omega*t)*(1+omega*t)
	}
	panic("the design system has no overdamped spring")
}

// cssNumber is a number the way the page writes the stops of a curve: three decimals
// at most, and no zero in front of the point.
func cssNumber(x float64) string {
	text := strings.TrimRight(strings.TrimRight(strconv.FormatFloat(x, 'f', 3, 64), "0"), ".")
	if text = strings.TrimPrefix(text, "0"); text == "" {
		return "0"
	}
	return text
}

// worstGap is how far the line through the stops, which CSS draws evenly over the
// time, is from the curve of the spring.
func worstGap(s kit.Spring, settle float64, stops []float64) float64 {
	worst := 0.0
	for k := 0; k <= 2000; k++ {
		at := float64(k) / 2000
		position := at * float64(len(stops)-1)
		i := int(math.Min(math.Floor(position), float64(len(stops)-2)))
		line := stops[i]*(1-(position-float64(i))) + stops[i+1]*(position-float64(i))
		worst = math.Max(worst, math.Abs(line-response(s, settle*at)))
	}
	return worst
}

// springCSS is a spring as the page has it: the milliseconds that it needs to settle,
// and its response sampled into linear() over that time, with as few stops as keep the
// line within 0.006 of the curve (less than a third of a pixel on a 56 px pill).
func springCSS(s kit.Spring) (millis string, linear string) {
	settle := s.SettleTime().Seconds()
	for n := 8; ; n += 2 {
		stops := make([]float64, n)
		for i := range stops {
			stops[i] = math.Round(response(s, settle*float64(i)/float64(n-1))*1000) / 1000
		}
		stops[n-1] = 1
		if worstGap(s, settle, stops) > 0.006 {
			continue
		}
		text := make([]string, n)
		for i, stop := range stops {
			text[i] = cssNumber(stop)
		}
		return strconv.Itoa(int(math.Round(settle*1000))) + "ms", "linear(" + strings.Join(text, ",") + ")"
	}
}

// The springs of the page by the names of its variables, from the motion scheme of the
// desktop interface.
var pageSprings = map[string]kit.Spring{
	"fast-spatial":    kit.Expressive.FastSpatial,
	"default-spatial": kit.Expressive.DefaultSpatial,
	"slow-spatial":    kit.Expressive.SlowSpatial,
	"fast-effects":    kit.Expressive.FastEffects,
	"default-effects": kit.Expressive.DefaultEffects,
	"slow-effects":    kit.Expressive.SlowEffects,
}

func TestTheSpringsOfThePageAreTheExpressiveSprings(t *testing.T) {
	page := string(webappHTML)
	names := make([]string, 0, len(pageSprings))
	for name := range pageSprings {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		millis, linear := springCSS(pageSprings[name])
		wantTime, wantCurve := "--sp-"+name+"-t:"+millis+";", "--sp-"+name+":"+linear+";"
		if !strings.Contains(page, wantTime) || !strings.Contains(page, wantCurve) {
			t.Errorf("the page does not have the spring %s of mygo/ui/kit (damping %v, stiffness %v); in :root and in the @supports block of its CSS they must be\n  %s\n  %s",
				name, pageSprings[name].Damping, pageSprings[name].Stiffness, wantTime, wantCurve)
		}
	}
}

func TestThePageMovesOnItsSprings(t *testing.T) {
	page := string(webappHTML)
	// A transition or an animation takes its time and its curve from a spring. The ones that
	// run until the work is done (the wavy progress, the loading indicator) are the exception.
	timed := regexp.MustCompile(`(?:transition|animation):[^;}]*`)
	literalDuration := regexp.MustCompile(`(?:^|[\s,])\d*\.?\d+m?s\b`)
	for _, declaration := range timed.FindAllString(page, -1) {
		if strings.Contains(declaration, "infinite") {
			continue
		}
		if literalDuration.MatchString(declaration) {
			t.Errorf("%q has a duration of its own: use the --sp-* variables of the springs", declaration)
		}
		if strings.Contains(declaration, "cubic-bezier") {
			t.Errorf("%q has a curve of its own: use the --sp-* variables of the springs", declaration)
		}
	}
	// The curves that earlier pages had stay as what a browser without linear() gets.
	for _, line := range strings.Split(page, "\n") {
		if strings.Contains(line, "cubic-bezier(") && !strings.HasPrefix(line, "  --sp-") {
			t.Errorf("a curve of its own outside the fallbacks of the springs: %s", strings.TrimSpace(line))
		}
	}
}
