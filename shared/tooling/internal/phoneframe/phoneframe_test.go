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

package phoneframe

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var (
	cream = color.NRGBA{R: 255, G: 248, B: 245, A: 255}
	grey  = color.NRGBA{R: 0x7A, G: 0x7A, B: 0x84, A: 255}
)

// screen is a picture of one colour: what a phone shows, as far as the frame is concerned.
func screen(width, height int) *image.NRGBA {
	picture := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			picture.SetNRGBA(x, y, cream)
		}
	}
	return picture
}

func TestFrameRoundsTheCornersAndDrawsTheBorderOverTheEdge(t *testing.T) {
	framed := Frame(screen(200, 400), 20, 3, grey)
	if got := framed.Bounds(); got != image.Rect(0, 0, 200, 400) {
		t.Fatalf("the picture changed its size: %v", got)
	}
	// Outside the rounding the picture is transparent, and what colour it has there does not matter.
	for name, at := range map[string][2]int{"outside the rounding": {0, 0}, "outside, a little further in": {2, 2}} {
		if got := framed.NRGBAAt(at[0], at[1]); got.A != 0 {
			t.Errorf("%s, %v: %v is not transparent", name, at, got)
		}
	}
	for name, c := range map[string]struct {
		x, y int
		want color.NRGBA
	}{
		"the border on the arc":        {6, 6, grey},
		"the border at the top":        {100, 0, grey},
		"the border at the top, third": {100, 2, grey},
		"the screen under the border":  {100, 3, cream},
		"the border on the left":       {2, 200, grey},
		"the screen next to it":        {3, 200, cream},
		"the border on the right":      {197, 200, grey},
		"the border at the bottom":     {100, 399, grey},
		"the middle":                   {100, 200, cream},
	} {
		if got := framed.NRGBAAt(c.x, c.y); got != c.want {
			t.Errorf("%s, (%d,%d): %v, want %v", name, c.x, c.y, got, c.want)
		}
	}
}

func TestFrameAntiAliasesTheEdgeOfTheRounding(t *testing.T) {
	framed := Frame(screen(200, 400), 20, 3, grey)
	// On the arc the picture is neither transparent nor opaque: some pixel of the first row is in between.
	partial := false
	for x := range 20 {
		if alpha := framed.NRGBAAt(x, 0).A; alpha > 0 && alpha < 255 {
			partial = true
		}
	}
	if !partial {
		t.Error("the edge of the rounding is not smooth")
	}
}

func TestFrameKeepsWhatIsTransparentTransparent(t *testing.T) {
	picture := screen(100, 100)
	picture.SetNRGBA(50, 50, color.NRGBA{R: 1, G: 2, B: 3, A: 0})
	if got := Frame(picture, 10, 2, grey).NRGBAAt(50, 50).A; got != 0 {
		t.Errorf("a transparent pixel has alpha %d", got)
	}
}

func TestParseColor(t *testing.T) {
	for text, want := range map[string]color.NRGBA{"7A7A84": grey, "#7a7a84": grey, "000000": {A: 255}} {
		if got, err := parseColor(text); err != nil || got != want {
			t.Errorf("%q: %v, %v", text, got, err)
		}
	}
	for _, text := range []string{"", "7A7A8", "7A7A845", "#7A7A8", "gg0000", "#"} {
		if _, err := parseColor(text); err == nil {
			t.Errorf("%q is no colour", text)
		}
	}
}

func TestRunFramesAFile(t *testing.T) {
	dir := t.TempDir()
	input, output := filepath.Join(dir, "in.png"), filepath.Join(dir, "out.png")
	var raw bytes.Buffer
	if err := png.Encode(&raw, screen(300, 600)); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(input, raw.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	// The sizes of the guide, which are the defaults, in a picture that is big enough for them.
	if err := run([]string{input, output}, io.Discard); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(output) //nolint:gosec // The picture of this test.
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file.Close() }()
	framed, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}
	if framed.Bounds() != image.Rect(0, 0, 300, 600) {
		t.Errorf("the size is %v", framed.Bounds())
	}
	if _, _, _, alpha := framed.At(0, 0).RGBA(); alpha != 0 {
		t.Errorf("the corner is not transparent")
	}
	if got := color.NRGBAModel.Convert(framed.At(150, 0)); got != grey {
		t.Errorf("the border is %v", got)
	}
	if got := color.NRGBAModel.Convert(framed.At(150, 300)); got != cream {
		t.Errorf("the middle is %v", got)
	}
	// The options: a border that is another colour and wider.
	other := filepath.Join(dir, "other.png")
	if err := run([]string{"--radius", "0", "--border", "10", "--color", "#FF0000", input, other}, io.Discard); err != nil {
		t.Fatal(err)
	}
	file2, err := os.Open(other) //nolint:gosec // The picture of this test.
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = file2.Close() }()
	square, err := png.Decode(file2)
	if err != nil {
		t.Fatal(err)
	}
	if got := color.NRGBAModel.Convert(square.At(0, 0)); got != (color.NRGBA{R: 255, A: 255}) {
		t.Errorf("with no rounding the corner is %v", got)
	}
	if got := color.NRGBAModel.Convert(square.At(150, 9)); got != (color.NRGBA{R: 255, A: 255}) {
		t.Errorf("the border of 10 pixels is %v at 9", got)
	}
	if got := color.NRGBAModel.Convert(square.At(150, 10)); got != cream {
		t.Errorf("the screen after the border of 10 pixels is %v", got)
	}
}

func TestRunFails(t *testing.T) {
	dir := t.TempDir()
	notPNG := filepath.Join(dir, "text.png")
	if err := os.WriteFile(notPNG, []byte("not a picture"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, c := range map[string]struct {
		args []string
		want string
	}{
		"no files":                     {nil, "Usage:"},
		"one file":                     {[]string{"in.png"}, "Usage:"},
		"three files":                  {[]string{"a", "b", "c"}, "Usage:"},
		"a colour that is none":        {[]string{"--color", "red", "a", "b"}, "not a colour"},
		"a negative radius":            {[]string{"--radius", "-1", "a", "b"}, "not negative"},
		"a file that is not there":     {[]string{filepath.Join(dir, "nothing.png"), filepath.Join(dir, "out.png")}, "no such file"},
		"a file that is not a picture": {[]string{notPNG, filepath.Join(dir, "out.png")}, "text.png"},
		"an unknown option":            {[]string{"--shadow", "a", "b"}, "shadow"},
	} {
		if err := run(c.args, io.Discard); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want an error that says %q", name, err, c.want)
		}
	}
}
