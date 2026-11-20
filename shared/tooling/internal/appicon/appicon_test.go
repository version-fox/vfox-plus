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
	"bytes"
	"encoding/xml"
	"fmt"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
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

func TestRenderFox(t *testing.T) {
	for _, size := range []int{16, 32, 64} {
		img := RenderFox(size)
		if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
			t.Fatalf("size = %v, want %d", b, size)
		}
		// The fox is centred, as big as the air around it allows, and nothing else is drawn:
		// the plate is indigo and the fox has none of that.
		minX, minY, maxX, maxY := size, size, -1, -1
		var ink, orange int
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				c := img.NRGBAAt(x, y)
				if c.A == 0 {
					continue
				}
				minX, maxX = min(minX, x), max(maxX, x)
				minY, maxY = min(minY, y), max(maxY, y)
				if c.A == 255 && c.B > 0x90 && c.R < 0x60 {
					t.Fatalf("%d px: pixel %d,%d = %v is the plate", size, x, y, c)
				}
				if near(c, "#17123F", 16) {
					ink++
				}
				if c.A == 255 && c.R > 0xF0 && c.G > 0x70 && c.G < 0xC0 && c.B < 0x60 {
					orange++
				}
			}
		}
		if maxX < minX {
			t.Fatalf("%d px: nothing is drawn", size)
		}
		air := float64(size) * foxAir
		if float64(minX) > air+1.5 || float64(size-1-maxX) > air+1.5 {
			t.Errorf("%d px: the fox spans x %d to %d, want it to fill the width but the air of %.1f px", size, minX, maxX, air)
		}
		if d := (minX + maxX) - (size - 1); d < -1 || d > 1 {
			t.Errorf("%d px: the fox spans x %d to %d, which is not centred", size, minX, maxX)
		}
		if d := (minY + maxY) - (size - 1); d < -2 || d > 2 {
			t.Errorf("%d px: the fox spans y %d to %d, which is not centred", size, minY, maxY)
		}
		if size >= 32 && (ink == 0 || orange == 0) {
			t.Errorf("%d px: %d pixels of ink and %d of orange, want both: the eyes and the fur", size, ink, orange)
		}
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
	all := append([]shape{plate, halo, {id: "adaptive-plate", d: androidBackground}}, foxShapes...)
	for _, s := range all {
		if _, err := parsePath(s.d); err != nil {
			t.Errorf("%s: %v", s.id, err)
		}
	}
}

func TestRenderAdaptiveForeground(t *testing.T) {
	img := RenderAdaptiveForeground(432)
	// The plate covers the whole canvas, so the corners are opaque, and the colour runs from the
	// top colour of the plate to the bottom one.
	if c := img.NRGBAAt(0, 0); !near(c, "#5444E4", 12) {
		t.Errorf("top-left corner = %v, want the top colour of the plate", c)
	}
	if c := img.NRGBAAt(431, 431); !near(c, "#1B1268", 16) {
		t.Errorf("bottom-right corner = %v, want the bottom colour of the plate", c)
	}
}

func TestWritePhoneIcons(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "resources")
	if err := WritePhoneIcons(dir); err != nil {
		t.Fatalf("WritePhoneIcons: %v", err)
	}
	for file, side := range map[string]int{IconFile: PhoneIconSize, ForegroundFile: AdaptiveForegroundSize} {
		data, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		img, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		if b := img.Bounds(); b.Dx() != side || b.Dy() != side {
			t.Errorf("%s is %v, want %d by %d", file, b, side, side)
		}
	}
	// The icon keeps the rounded corners of the desktop icon, transparent outside; the foreground is
	// opaque to its corners.
	icon, err := os.ReadFile(filepath.Join(dir, IconFile))
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := png.Decode(bytes.NewReader(icon))
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, a := decoded.At(0, 0).RGBA(); a != 0 {
		t.Errorf("the corner of the icon has alpha %d, want 0", a>>8)
	}
}

func TestRunNeedsADirectory(t *testing.T) {
	if err := Run(nil); err == nil {
		t.Fatal("Run without a directory succeeded")
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
