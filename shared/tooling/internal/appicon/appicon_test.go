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

package appicon

import (
	"encoding/xml"
	"fmt"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func pixel(t *testing.T, w, h, x, y int) color.NRGBA {
	t.Helper()
	img := Render(w, h)
	return img.NRGBAAt(x, y)
}

// near reports whether every channel of c is within tolerance of the colour of want.
func near(c color.NRGBA, want string, tolerance int) bool {
	w := parseColor(want, 1)
	diff := func(a, b uint8) bool { return math.Abs(float64(a)-float64(b)) <= float64(tolerance) }
	return diff(c.R, w.R) && diff(c.G, w.G) && diff(c.B, w.B) && c.A == 255
}

func TestRender(t *testing.T) {
	img := Render(256, 256)
	if b := img.Bounds(); b.Dx() != 256 || b.Dy() != 256 {
		t.Fatalf("size = %v", b)
	}
	// What the pixels are follows from the paths: where the corner of the plate is rounded away,
	// the eyes and the nose of the fox, the fur, the cheeks and the plate.
	if c := img.NRGBAAt(1, 1); c.A != 0 {
		t.Errorf("corner = %v, want transparent", c)
	}
	for name, at := range map[string][2]int{"left eye": {99, 139}, "right eye": {157, 139}, "nose": {128, 182}} {
		if c := img.NRGBAAt(at[0], at[1]); !near(c, "#17123F", 6) {
			t.Errorf("%s at %v = %v, want the ink", name, at, c)
		}
	}
	if c := img.NRGBAAt(128, 150); c.A != 255 || c.R < 0xF0 || c.G < 0x80 || c.G > 0xB8 || c.B > 0x45 {
		t.Errorf("fur = %v, want orange", c)
	}
	if c := img.NRGBAAt(59, 135); !near(c, "#FFEFDD", 14) {
		t.Errorf("cheek = %v, want cream", c)
	}
	if c := img.NRGBAAt(12, 128); c.A != 255 || c.B < 0x90 || c.R > 0x60 {
		t.Errorf("plate = %v, want indigo", c)
	}
	// The halo is a lighter plate: it lifts the plate near the fox.
	plate, haloed := img.NRGBAAt(14, 200), img.NRGBAAt(40, 128)
	if int(haloed.R)+int(haloed.G) <= int(plate.R)+int(plate.G) {
		t.Errorf("halo %v is not lighter than the plate %v", haloed, plate)
	}
}

func TestRenderWide(t *testing.T) {
	// The wide logo has the icon of its height in the middle and nothing around it.
	if c := pixel(t, 310, 150, 3, 75); c.A != 0 {
		t.Errorf("left of the icon = %v, want transparent", c)
	}
	wide, square := Render(310, 150), Render(150, 150)
	if got, want := wide.NRGBAAt(155, 75), square.NRGBAAt(75, 75); got != want {
		t.Errorf("centre = %v, want %v", got, want)
	}
	for _, size := range []int{16, 24, 44, 50} {
		if c := pixel(t, size, size, size/2, size/2); c.A != 255 {
			t.Errorf("centre at %d px = %v, want opaque", size, c)
		}
	}
}

func TestParsePath(t *testing.T) {
	cmds, err := parsePath("M1,2 L3.5,-4 C1,2 3,4 5,6 Z M0 0 1 1")
	if err != nil {
		t.Fatal(err)
	}
	var ops []byte
	for _, c := range cmds {
		ops = append(ops, c.op)
	}
	if string(ops) != "MLCZML" { // Further pairs of a move are lines.
		t.Errorf("commands = %s", ops)
	}
	if cmds[1].pts[0] != [2]float64{3.5, -4} || cmds[2].pts[2] != [2]float64{5, 6} {
		t.Errorf("points = %v %v", cmds[1].pts, cmds[2].pts)
	}
	for _, bad := range []string{"M1", "M1,2 H3", "1,2 M3,4", "M1,2 C1,2 3,4 Z", "M1,x"} {
		if _, err := parsePath(bad); err == nil {
			t.Errorf("parsePath(%q) succeeded", bad)
		}
	}
}

func TestPathsAreAbsolute(t *testing.T) {
	all := append([]shape{plate, halo, {id: "android", d: androidBackground}, {id: "mono", d: maskMono}}, foxShapes...)
	for _, s := range all {
		if _, err := parsePath(s.d); err != nil && s.id != "android" { // H and V are for Android only.
			t.Errorf("%s: %v", s.id, err)
		}
	}
}

// repoFile reads a file of the repository, found from the directory of the test.
func repoFile(t *testing.T, rel string) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel))) //nolint:gosec // The repository's own files.
		if err == nil {
			return string(data)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("%s is not in a parent directory of the test", rel)
		}
		dir = parent
	}
}

type svgPath struct {
	ID        string `xml:"id,attr"`
	Fill      string `xml:"fill,attr"`
	D         string `xml:"d,attr"`
	Transform string `xml:"transform,attr"`
}

type svgIcon struct {
	Gradients []struct {
		ID    string  `xml:"id,attr"`
		X1    float64 `xml:"x1,attr"`
		Y1    float64 `xml:"y1,attr"`
		X2    float64 `xml:"x2,attr"`
		Y2    float64 `xml:"y2,attr"`
		Stops []struct {
			Offset  float64 `xml:"offset,attr"`
			Color   string  `xml:"stop-color,attr"`
			Opacity float64 `xml:"stop-opacity,attr"`
		} `xml:"stop"`
	} `xml:"defs>linearGradient"`
	Paths []svgPath `xml:"path"`
	Fox   struct {
		Transform string    `xml:"transform,attr"`
		Paths     []svgPath `xml:"path"`
	} `xml:"g"`
}

func TestMatchesSVG(t *testing.T) {
	var icon svgIcon
	if err := xml.Unmarshal([]byte(repoFile(t, "shared/packaging/linux/dev.vfox.VfoxPlus.svg")), &icon); err != nil {
		t.Fatal(err)
	}
	if len(icon.Gradients) != len(gradients) {
		t.Fatalf("the SVG has %d gradients, appicon %d", len(icon.Gradients), len(gradients))
	}
	for i, g := range gradients {
		s := icon.Gradients[i]
		if s.ID != g.id || s.X1 != g.x1 || s.Y1 != g.y1 || s.X2 != g.x2 || s.Y2 != g.y2 || len(s.Stops) != len(g.stops) {
			t.Errorf("gradient %s differs: SVG %+v, appicon %+v", g.id, s, g)
			continue
		}
		for k, st := range g.stops {
			sv := s.Stops[k]
			if sv.Opacity == 0 {
				sv.Opacity = 1 // The attribute is left out for opaque stops.
			}
			if sv.Offset != st.offset || !strings.EqualFold(sv.Color, st.color) || sv.Opacity != st.opacity {
				t.Errorf("gradient %s, stop %d: SVG %+v, appicon %+v", g.id, k, sv, st)
			}
		}
	}
	if len(icon.Paths) != 2 || len(icon.Fox.Paths) != len(foxShapes) {
		t.Fatalf("the SVG has %d paths and %d in the fox, appicon 2 and %d", len(icon.Paths), len(icon.Fox.Paths), len(foxShapes))
	}
	check := func(got svgPath, want shape) {
		if got.ID != want.id || got.D != want.d || got.Fill != want.fill {
			t.Errorf("%s differs between the SVG and appicon:\nSVG     %+v\nappicon %+v", want.id, got, want)
		}
	}
	check(icon.Paths[0], plate)
	check(icon.Paths[1], halo)
	if want := fmt.Sprintf("translate(128 128) scale(%v) translate(-128 -128)", haloScale); icon.Paths[1].Transform != want {
		t.Errorf("halo transform = %q, want %q", icon.Paths[1].Transform, want)
	}
	if want := fmt.Sprintf("translate(%v %v) scale(%v)", foxX, foxY, foxScale); icon.Fox.Transform != want {
		t.Errorf("fox transform = %q, want %q", icon.Fox.Transform, want)
	}
	for i, s := range foxShapes {
		check(icon.Fox.Paths[i], s)
	}
}

var (
	pathDataRe  = regexp.MustCompile(`android:pathData="([^"]*)"`)
	fillColorRe = regexp.MustCompile(`android:fillColor="([^"]*)"`)
	gradientRe  = regexp.MustCompile(`(?s)<gradient\s+android:type="linear"\s+android:startX="([^"]*)" android:startY="([^"]*)"\s+android:endX="([^"]*)" android:endY="([^"]*)">(.*?)</gradient>`)
	itemRe      = regexp.MustCompile(`<item android:offset="([^"]*)" android:color="([^"]*)"/>`)
)

// androidColor is the colour as an Android resource writes it, #RRGGBB or #AARRGGBB.
func androidColor(color string, opacity float64) string {
	color = strings.ToUpper(color)
	if opacity >= 1 {
		return color
	}
	return fmt.Sprintf("#%02X%s", int(math.Round(opacity*255)), strings.TrimPrefix(color, "#"))
}

// androidGradient is how a gradient of the XML reads: its line and its stops.
func androidGradient(g gradient) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%v,%v,%v,%v", g.x1, g.y1, g.x2, g.y2)
	for _, s := range g.stops {
		fmt.Fprintf(&b, " %v=%s", s.offset, androidColor(s.color, s.opacity))
	}
	return b.String()
}

func gradientByID(id string) gradient {
	for _, g := range gradients {
		if g.id == id {
			return g
		}
	}
	panic("no gradient " + id)
}

// fullBleedPlate is the gradient of the plate of the adaptive icon, which fills the canvas
// instead of the rounded square of the other icons.
func fullBleedPlate() gradient {
	g := gradientByID("plate")
	g.x1, g.y1, g.x2, g.y2 = 0, 0, canvas, canvas
	return g
}

func TestMatchesAndroid(t *testing.T) {
	type layer struct {
		file      string
		paths     []string   // the path data of the file, in order
		gradients []gradient // the gradients, in order
		solids    []string   // the solid colours, in order
		groups    []string   // the transforms of the groups, in order
	}
	foxPaths, foxSolids := []string{}, []string{}
	for _, s := range foxShapes {
		foxPaths = append(foxPaths, s.d)
		if !strings.HasPrefix(s.fill, "url(") {
			foxSolids = append(foxSolids, strings.ToUpper(s.fill))
		}
	}
	foxGradients := []gradient{gradientByID("cream"), gradientByID("fur")}
	foxAndroid := fmt.Sprintf("%v,%v,%v", androidFoxX, androidFoxY, androidFoxScale)
	layers := []layer{
		{"ic_launcher_background.xml", []string{androidBackground, halo.d}, []gradient{fullBleedPlate(), gradientByID("halo")}, nil,
			[]string{fmt.Sprint(androidHaloScale)}},
		{"ic_launcher_foreground.xml", foxPaths, foxGradients, foxSolids, []string{foxAndroid}},
		{"ic_mark.xml", append([]string{plate.d, halo.d}, foxPaths...), append([]gradient{gradientByID("plate"), gradientByID("halo")}, foxGradients...), foxSolids,
			[]string{fmt.Sprint(haloScale), fmt.Sprintf("%v,%v,%v", foxX, foxY, foxScale)}},
	}
	for _, l := range layers {
		text := repoFile(t, "android/app/src/main/res/drawable/"+l.file)
		var got []string
		for _, m := range pathDataRe.FindAllStringSubmatch(text, -1) {
			got = append(got, m[1])
		}
		if strings.Join(got, "\n") != strings.Join(l.paths, "\n") {
			t.Errorf("%s: the path data differs from appicon (%d paths in the file, %d in appicon)", l.file, len(got), len(l.paths))
		}
		var solids []string
		for _, m := range fillColorRe.FindAllStringSubmatch(text, -1) {
			solids = append(solids, strings.ToUpper(m[1]))
		}
		if strings.Join(solids, " ") != strings.Join(l.solids, " ") {
			t.Errorf("%s: solid colours %v, want %v", l.file, solids, l.solids)
		}
		var grads []string
		for _, m := range gradientRe.FindAllStringSubmatch(text, -1) {
			var stops []string
			for _, it := range itemRe.FindAllStringSubmatch(m[5], -1) {
				stops = append(stops, it[1]+"="+strings.ToUpper(it[2]))
			}
			grads = append(grads, fmt.Sprintf("%s,%s,%s,%s %s", m[1], m[2], m[3], m[4], strings.Join(stops, " ")))
		}
		var want []string
		for _, g := range l.gradients {
			want = append(want, androidGradient(g))
		}
		if strings.Join(grads, "\n") != strings.Join(want, "\n") {
			t.Errorf("%s: gradients\n%s\nwant\n%s", l.file, strings.Join(grads, "\n"), strings.Join(want, "\n"))
		}
		var groups []string
		for _, m := range regexp.MustCompile(`(?s)<group(.*?)>`).FindAllStringSubmatch(text, -1) {
			attr := func(name string) string {
				a := regexp.MustCompile(`android:` + name + `="([^"]*)"`).FindStringSubmatch(m[1])
				if a == nil {
					return ""
				}
				return a[1]
			}
			if attr("pivotX") != "" { // The halo: a scale around the centre.
				groups = append(groups, attr("scaleX"))
			} else {
				groups = append(groups, attr("translateX")+","+attr("translateY")+","+attr("scaleX"))
			}
		}
		if strings.Join(groups, " ") != strings.Join(l.groups, " ") {
			t.Errorf("%s: group transforms %v, want %v", l.file, groups, l.groups)
		}
	}

	// The monochrome icon draws the cheeks, with the eyes and the nose cut out, below the fur,
	// with the eyes and the ears cut out.
	text := repoFile(t, "android/app/src/main/res/drawable/ic_launcher_monochrome.xml")
	var mono []string
	for _, m := range pathDataRe.FindAllStringSubmatch(text, -1) {
		mono = append(mono, m[1])
	}
	byID := map[string]string{}
	for _, s := range foxShapes {
		byID[s.id] = s.d
	}
	wantMono := []string{
		strings.Join([]string{byID["cheeks"], byID["eye-left"], byID["eye-right"], byID["nose"]}, " "),
		strings.Join([]string{maskMono, byID["eye-left"], byID["eye-right"], byID["ear-left"], byID["ear-right"]}, " "),
	}
	if strings.Join(mono, "\n") != strings.Join(wantMono, "\n") {
		t.Errorf("ic_launcher_monochrome.xml differs from appicon:\n%v\nwant\n%v", mono, wantMono)
	}
	for _, f := range []string{"ic_launcher.xml", "ic_launcher_round.xml"} {
		adaptive := repoFile(t, "android/app/src/main/res/mipmap-anydpi-v26/"+f)
		for _, ref := range []string{"@drawable/ic_launcher_background", "@drawable/ic_launcher_foreground", "@drawable/ic_launcher_monochrome"} {
			if !strings.Contains(adaptive, ref) {
				t.Errorf("%s does not use %s", f, ref)
			}
		}
	}
}
