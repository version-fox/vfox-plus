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

package kit

import (
	"image"
	"image/color"
	"image/draw"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func rect(x, y, w, h float32) ui.Rect { return ui.Rect{X: x, Y: y, W: w, H: h} }

func TestTheCensusCountsWordsAndAHanCharacterIsOne(t *testing.T) {
	for text, want := range map[string]int{
		"":                         0,
		"Install":                  1,
		"Install and use":          3,
		"24.21.0 or latest":        3,
		"21.0.8-tem, such as j17.": 4,
		"未发现问题。":                   5,
		"已安装 (2)":                  4,
	} {
		if got := CountWords(text); got != want {
			t.Errorf("CountWords(%q) = %d, want %d", text, got, want)
		}
	}
}

func TestTheCensusLeavesOutWhatIsNotOnTheScreen(t *testing.T) {
	window := rect(0, 0, 400, 300)
	page := []*Node{
		{Role: RoleText, Text: "Seen in the window and read by a person", Rect: rect(10, 10, 200, 20), Size: 14},
		{Role: RoleText, Text: "Scrolled away below the window and not on the screen", Rect: rect(10, 500, 200, 20), Size: 14},
		{Role: RoleText, Text: "No frame has drawn this one yet", Size: 14},
	}
	if got := TakeCensus(page, window, nil); got.Words != 9 || got.Prose.Blocks != 1 {
		t.Errorf("only the text in the window is on the screen: %+v", got)
	}
	// A dialog is a screen of its own: what is behind it is dimmed, and is the census of its page.
	dialog := append(append([]*Node{}, page...),
		&Node{Role: RoleDialog, Name: "Title", Rect: rect(50, 50, 300, 200), Modal: true, Enabled: true},
		&Node{Role: RoleText, Text: "In the dialog", Rect: rect(60, 60, 100, 20), Size: 14, Modal: true},
	)
	if got := TakeCensus(dialog, window, nil); got.Words != 3 || got.Prose.Blocks != 0 {
		t.Errorf("the dialog is the screen: %+v", got)
	}
}

func TestTheCensusTakesASentenceForWhatItsWordsAreNotData(t *testing.T) {
	window := rect(0, 0, 600, 300)
	nodes := []*Node{
		{Role: RoleText, Text: "A sentence that has more than eight words in it", Rect: rect(0, 0, 400, 20), Size: 14},
		{Role: RoleText, Text: "Reachable at https://10.255.255.254:22150 and https://172.23.120.55:22150/auth", Rect: rect(0, 20, 500, 20), Size: 14},
		{Role: RoleText, Text: "15:88:99:D1:EF:6D:65:9E:37:6E:E0:42:4D:B8:69:65", Rect: rect(0, 40, 400, 20), Size: 12},
		{Role: RoleText, Text: "eval one two three four five six seven eight nine", Rect: rect(0, 60, 400, 20), Size: 14, Mono: true},
	}
	got := TakeCensus(nodes, window, nil)
	if got.Prose.Blocks != 1 || !strings.HasPrefix(got.Prose.Texts[0], "10: A sentence") {
		t.Errorf("an address, a fingerprint and a line of code are no sentence: %+v", got.Prose)
	}
}

func TestTheCensusReadsATextOnlyWhenMoreThanHalfOfItsBoxIsInTheWindow(t *testing.T) {
	window := rect(0, 0, 400, 300)
	page := []*Node{
		{Role: RoleText, Text: "Whole in the window", Rect: rect(10, 100, 200, 20), Size: 14},
		{Role: RoleText, Text: "Most in the window below", Rect: rect(10, 288, 200, 20), Size: 14},
		{Role: RoleText, Text: "Half in the window", Rect: rect(10, 290, 200, 20), Size: 14},
		{Role: RoleText, Text: "Mostly out of the window", Rect: rect(10, 295, 200, 20), Size: 14},
		{Role: RoleText, Text: "Most in the window above", Rect: rect(10, -8, 200, 20), Size: 14},
		{Role: RoleText, Text: "Half above the window", Rect: rect(10, -10, 200, 20), Size: 14},
	}
	if got := TakeCensus(page, window, nil); got.Words != 4+5+5 {
		t.Errorf("the three texts that are more than half in the window are read, and not the half or less: %d words", got.Words)
	}
}

func TestTheCensusHoldsTheControlsToTheirSizeAndThePageToItsWindow(t *testing.T) {
	window := rect(0, 0, 400, 300)
	nodes := []*Node{
		{Role: RoleButton, Name: "Fits", Rect: rect(10, 10, 80, 40), Enabled: true},
		{Role: RoleButton, Name: "Thin", Rect: rect(10, 60, 80, 20), Enabled: true},
		{Role: RoleSwitch, Name: "Switch", Rect: rect(10, 90, 52, 32), Enabled: true},
		{Role: RoleListItem, Name: "A row is not a control", Rect: rect(0, 130, 400, 20)},
	}
	got := TakeCensus(nodes, window, nil)
	if got.Targets.Total != 3 || got.Targets.Under24 != 1 || got.Overflow {
		t.Errorf("targets %+v, overflow %v", got.Targets, got.Overflow)
	}
	nodes = append(nodes, &Node{Role: RoleText, Text: "wider than the window", Rect: rect(300, 160, 200, 20), Size: 14})
	if got := TakeCensus(nodes, window, nil); !got.Overflow {
		t.Error("a text that goes across the edge of the window is an overflow")
	}
	if over := TakeCensus(nodes, window, nil).Over(); len(over) != 2 {
		t.Errorf("the budgets that it is over: %q", over)
	}
}

func TestTheCensusTakesASmallLineOnlyOutsideTheControls(t *testing.T) {
	window := rect(0, 0, 400, 300)
	nodes := []*Node{
		{Role: RoleButton, Name: "Open it", Rect: rect(10, 10, 200, 40), Enabled: true},
		{Role: RoleText, Text: "four small words here", Rect: rect(20, 20, 100, 16), Size: 12},
		{Role: RoleText, Text: "four small words again", Rect: rect(10, 100, 100, 16), Size: 12},
		{Role: RoleText, Text: "four small words mono", Rect: rect(10, 120, 100, 16), Size: 12, Mono: true},
		{Role: RoleText, Text: "four words at body size", Rect: rect(10, 140, 100, 20), Size: 14},
		{Role: RoleText, Text: "Short one", Rect: rect(10, 160, 100, 16), Size: 12},
	}
	if got := TakeCensus(nodes, window, nil); got.Helpers != 1 {
		t.Errorf("one small line stands under a title: %d", got.Helpers)
	}
}

func TestTheCensusMeasuresTheContrastOfATextAgainstWhatItStandsOn(t *testing.T) {
	window := rect(0, 0, 200, 100)
	img := image.NewRGBA(image.Rect(0, 0, 200, 100))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	gray := ui.Color{R: 140, G: 140, B: 140, A: 255}
	nodes := []*Node{
		{Role: RoleText, Text: "Black on white", Rect: rect(0, 0, 100, 20), Size: 14, Color: ui.Color{A: 255}},
		{Role: RoleText, Text: "Gray, small", Rect: rect(0, 20, 100, 20), Size: 14, Color: gray},
		{Role: RoleText, Text: "Gray, large", Rect: rect(0, 40, 100, 30), Size: 28, Color: gray},
		{Role: RoleText, Text: "Faint, by its opacity", Rect: rect(0, 70, 100, 20), Size: 14, Color: ui.Color{A: 40}},
		{Role: RoleButton, Name: "Off", Rect: rect(100, 0, 100, 40), Enabled: false},
		{Role: RoleText, Text: "Gray in a disabled button", Rect: rect(105, 5, 90, 20), Size: 14, Color: gray},
	}
	got := TakeCensus(nodes, window, img)
	// The gray is 3.4:1: enough for a large text and not for a small one. The text of a disabled control is not asked
	// to stand out.
	if got.Contrast.Measured != 4 || got.Contrast.Failing != 2 {
		t.Errorf("contrast %+v", got.Contrast)
	}
	if len(got.Contrast.Worst) != 2 || got.Contrast.Worst[0].Text != "Faint, by its opacity" || got.Contrast.Worst[1].Text != "Gray, small" {
		t.Errorf("the worst first: %+v", got.Contrast.Worst)
	}
}

func TestTheCensusReadsThePixelsOfAnyImage(t *testing.T) {
	want := [3]uint8{200, 100, 50}
	rgba := image.NewRGBA(image.Rect(0, 0, 2, 2))
	rgba.Set(1, 1, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	nrgba := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	nrgba.Set(1, 1, color.NRGBA{R: 200, G: 100, B: 50, A: 255})
	gray := image.NewGray(image.Rect(0, 0, 2, 2))
	gray.Set(1, 1, color.Gray{Y: 77})
	for name, got := range map[string][3]uint8{"RGBA": pixel(rgba, 1, 1), "NRGBA": pixel(nrgba, 1, 1)} {
		if got != want {
			t.Errorf("%s: %v, want %v", name, got, want)
		}
	}
	if got := pixel(gray, 1, 1); got != [3]uint8{77, 77, 77} {
		t.Errorf("gray: %v", got)
	}
	if got := pixel(rgba, 0, 0); got != [3]uint8{} {
		t.Errorf("nothing drawn: %v", got)
	}
}

func TestTheCensusReadsWhatATextStandsOnAndNotItsInk(t *testing.T) {
	// A glyph that fills 64 % of its box (a large bold letter does): the ink is the colour that most of the box has, and
	// the colour behind the text is the other.
	img := image.NewRGBA(image.Rect(0, 0, 40, 40))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(2, 2, 18, 18), &image.Uniform{color.Black}, image.Point{}, draw.Src)
	black := ui.Color{A: 255}
	got := TakeCensus([]*Node{{Role: RoleText, Text: "E", Rect: rect(0, 0, 20, 20), Size: 14, Color: black}}, rect(0, 0, 40, 40), img)
	if got.Contrast.Measured != 1 || got.Contrast.Failing != 0 {
		t.Errorf("black on white, however much of the box the ink fills: %+v", got.Contrast)
	}
	// On a colour that does not stand out the picture says so, too.
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Gray{Y: 20}}, image.Point{}, draw.Src)
	if got := TakeCensus([]*Node{{Role: RoleText, Text: "E", Rect: rect(0, 0, 20, 20), Size: 14, Color: black}}, rect(0, 0, 40, 40), img); got.Contrast.Failing != 1 {
		t.Errorf("black on a dark gray is hard to read: %+v", got.Contrast)
	}
	// A box as high as the control that holds it has the border of the control in it, 2 rows of 20, and the fill is the
	// rest: the border is not what the text stands on.
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 0, 20, 1), &image.Uniform{color.Gray{Y: 120}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(0, 19, 20, 20), &image.Uniform{color.Gray{Y: 120}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(2, 2, 12, 18), &image.Uniform{color.Black}, image.Point{}, draw.Src)
	if got := TakeCensus([]*Node{{Role: RoleText, Text: "R", Rect: rect(0, 0, 20, 20), Size: 14, Color: black}}, rect(0, 0, 40, 40), img); got.Contrast.Failing != 0 {
		t.Errorf("black on white, with a gray border across the box: %+v", got.Contrast)
	}
	// A box that is larger than what the text stands on, as the line of a large initial is higher than its avatar: light text on
	// a dark fill, in a page that is nearly as light as the text. The page outweighs the fill in the box, and is not what the
	// glyph stands on.
	cream := color.RGBA{R: 250, G: 245, B: 240, A: 255}
	draw.Draw(img, img.Bounds(), &image.Uniform{cream}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(15, 15, 35, 35), &image.Uniform{color.RGBA{R: 60, G: 40, B: 20, A: 255}}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(21, 19, 29, 31), &image.Uniform{color.White}, image.Point{}, draw.Src)
	if got := TakeCensus([]*Node{{Role: RoleText, Text: "E", Rect: rect(0, 0, 40, 40), Size: 14, Color: ui.Color{R: 255, G: 255, B: 255, A: 255}}}, rect(0, 0, 40, 40), img); got.Contrast.Measured != 1 || got.Contrast.Failing != 0 {
		t.Errorf("white on a dark fill, in a box that holds more of a light page: %+v", got.Contrast)
	}
	// Text of the colour of what it stands on is not to be read: nothing is left that is not ink.
	draw.Draw(img, img.Bounds(), &image.Uniform{color.Black}, image.Point{}, draw.Src)
	if got := TakeCensus([]*Node{{Role: RoleText, Text: "E", Rect: rect(0, 0, 20, 20), Size: 14, Color: black}}, rect(0, 0, 40, 40), img); got.Contrast.Failing != 1 {
		t.Errorf("black on black cannot be read: %+v", got.Contrast)
	}
}

func TestTheCensusFindsTheSizesThatTheTypeScaleDoesNotHave(t *testing.T) {
	window := rect(0, 0, 400, 300)
	nodes := []*Node{
		{Role: RoleText, Text: "On the scale", Rect: rect(0, 0, 100, 20), Size: Type(BodyMedium).Size},
		{Role: RoleText, Text: "Emphasized, on it", Rect: rect(0, 20, 100, 20), Size: Emph(TitleLarge).Size},
		{Role: RoleText, Text: "Off it", Rect: rect(0, 40, 100, 20), Size: 13},
		{Role: RoleText, Text: "Off it again", Rect: rect(0, 60, 100, 20), Size: 13.4},
		{Role: RoleText, Text: "And another", Rect: rect(0, 80, 100, 20), Size: 20},
	}
	if got := TakeCensus(nodes, window, nil).OffScale; !reflect.DeepEqual(got, map[string]int{"13": 2, "20": 1}) {
		t.Errorf("sizes off the scale: %v", got)
	}
}

// A screen that is drawn, with what the budgets are about: the same icon on three rows, two buttons in each of them, a
// sentence, a small line, a text that cannot be read and a size that the scale does not have.
func TestTheCensusReadsAScreenThatIsDrawn(t *testing.T) {
	h := newTested(t, 560, 900, func(c Ctx) {
		cs := c.Colors()
		Column(c).Background(cs.Surface).Padding(16).Gap(12).Children(func() {
			var wall []ListItemSpec
			for _, name := range []string{"one", "two", "three"} {
				wall = append(wall, ListItemSpec{
					ID: name, Title: name, Leading: func(col ui.Color) { Icon(c, "info", 24, col, 0) },
					Trailing: func() {
						Row(c).Gap(8).Children(func() {
							Button(c, ButtonSpec{ID: "install", Label: "Install", Kind: TonalButton})
							Button(c, ButtonSpec{ID: "use", Label: "Use", Kind: OutlinedButton})
						})
					},
				})
			}
			ListGroup(c, wall...)
			// One action in a row, and in two ways: a split button, and a group of segments.
			ListGroup(c,
				ListItemSpec{ID: "split", Title: "split", Leading: func(col ui.Color) { Icon(c, "check", 24, col, 0) }, Trailing: func() {
					SplitButton(c, SplitButtonSpec{ID: "install", Label: "Install", MenuTooltip: "More", Items: []MenuItem{{Label: "Install and use"}}})
				}},
				ListItemSpec{ID: "segments", Title: "segments", Leading: func(col ui.Color) { Icon(c, "add", 24, col, 0) }, Trailing: func() {
					ButtonGroup(c, ButtonGroupSpec{ID: "g", Selected: "a", OnSelect: func(string) {}, Segments: []Segment{
						{Value: "a", Label: "One"}, {Value: "b", Label: "Two"}, {Value: "c", Label: "Three"},
					}})
				}},
			)
			Label(c, "This sentence has far more than eight words in it, to be a sentence.", Type(BodyMedium), cs.OnSurface)
			Label(c, "A small line under a title", Type(BodySmall), cs.OnSurfaceVariant)
			Label(c, "Faint text that nobody can read", Type(BodyMedium), cs.OnSurface.Alpha(0.12))
			Label(c, "Thirteen", TextStyle{Size: 13, Line: 18, Weight: 400}, cs.OnSurface)
		})
	})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	img := h.tester.Image()
	if img.Bounds().Dx() != 560 || img.Bounds().Dy() != 900 {
		t.Fatalf("a dp is a pixel here: %v", img.Bounds())
	}
	got := TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 560, 900), img)
	if got.RowsWithTwoActions != 3 {
		t.Errorf("three rows have two buttons with words: %d", got.RowsWithTwoActions)
	}
	if !reflect.DeepEqual(got.Wallpaper, []string{"info×3"}) {
		t.Errorf("the icon that three rows start with: %v", got.Wallpaper)
	}
	if got.Prose.Blocks != 1 || got.Helpers != 1 {
		t.Errorf("one sentence and one small line: %+v, %d", got.Prose, got.Helpers)
	}
	if got.Contrast.Failing != 1 || len(got.Contrast.Worst) != 1 || !strings.HasPrefix(got.Contrast.Worst[0].Text, "Faint text") {
		t.Errorf("only the faint text is hard to read: %+v", got.Contrast)
	}
	if !reflect.DeepEqual(got.OffScale, map[string]int{"13": 1}) {
		t.Errorf("one size off the scale: %v", got.OffScale)
	}
	if got.Overflow || got.Targets.Under24 != 0 || len(got.MissingIcons) != 0 {
		t.Errorf("nothing else is wrong: %+v", got)
	}
	if len(got.Over()) != 4 {
		t.Errorf("four budgets are exceeded (the text, the buttons, the icon and the size): %q", got.Over())
	}
}

func TestTheCensusFindsATitleOrAControlThatIsCutShort(t *testing.T) {
	h := newTested(t, 300, 300, func(c Ctx) {
		cs := c.Colors()
		Column(c).Background(cs.Surface).Padding(16).Gap(8).Children(func() {
			// A title that is too long for its width, in one line: an ellipsis ends it.
			Line(c, "A title that is far too long for the room it has", Type(TitleLarge), cs.OnSurface).Width(100)
			// As long, in a line of its own that has the room.
			Line(c, "A title with room", Type(TitleLarge), cs.OnSurface)
			// A button that is too narrow for its label does not cut it: the label goes on in a second line.
			Button(c, ButtonSpec{ID: "wraps", Label: "A label that is far too long", Kind: TonalButton}).Width(90)
			// A button that has the room.
			Button(c, ButtonSpec{ID: "fits", Label: "Fits", Kind: TonalButton})
			// A name or a path that is data, in a line of its own, may be cut short.
			Line(c, "version-fox/vfox-a-name-that-is-too-long-for-the-line", Type(BodyLarge), cs.OnSurface).Width(100)
			// A sentence that is free to wrap: nothing is cut.
			Label(c, "A sentence that is far too long for the room it has", Type(TitleLarge), cs.OnSurface).Width(100)
		})
	})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	got := TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 300, 300), nil)
	if len(got.Cut) != 1 || !strings.Contains(got.Cut[0], "A title that is far too long") {
		t.Errorf("the title is cut, and the label of the button is not: %q", got.Cut)
	}
	for _, n := range h.env.Reg.Nodes() {
		if n.Role == RoleButton && n.Name == "A label that is far too long" && (n.Rect.W > 90.5 || n.Rect.H <= buttonHeight) {
			t.Errorf("a button that is too narrow for its label is as wide as it was made, and has a second line: %+v", n.Rect)
		}
	}
	// The label of a control that cuts it, which no button does now: the text of a control is held to its room, a text of 14 dp
	// that no control holds is not (it is a sentence, or data).
	control := &Node{Role: RoleButton, Name: "Item", Rect: rect(0, 0, 90, 40)}
	label := &Node{Role: RoleText, Text: "A label that is far too long", Rect: rect(16, 10, 58, 20), Size: 14, Natural: 160}
	if got := TakeCensus([]*Node{control, label}, rect(0, 0, 300, 100), nil); len(got.Cut) != 1 || !strings.Contains(got.Cut[0], "A label that is far too long") {
		t.Errorf("the label of a control that is cut is found: %q", got.Cut)
	}
	if got := TakeCensus([]*Node{label}, rect(0, 0, 300, 100), nil); len(got.Cut) != 0 {
		t.Errorf("and the same text that no control holds is not: %q", got.Cut)
	}
}

func TestTheCensusFindsAControlThatIsDrawnBeyondThePaddingOfItsRow(t *testing.T) {
	// A window of 360 dp: the row is 312 dp wide, from 24, its padding is 16 dp, and under the title, which starts at 80
	// after the icon, 240 dp are left up to the padding.
	census := func(width float32) Census {
		h := newTested(t, 360, 300, func(c Ctx) {
			Column(c).Background(c.Colors().Surface).Padding(24).Children(func() {
				ListGroup(c, ListItemSpec{
					ID: "item", Title: "Item", Wrap: true,
					Leading: func(col ui.Color) { Icon(c, "check", 24, col, 0) },
					Trailing: func() {
						Button(c, ButtonSpec{ID: "wide", Label: "A button", Kind: TonalButton}).Width(width).Shrink(0)
					},
				})
			})
		})
		h.tester.Frame()
		h.tester.Advance(time.Second)
		return TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 360, 300), nil)
	}
	if got := census(240); len(got.Sticking) != 0 || got.Overflow {
		t.Errorf("a button that fills what is left is inside the padding: %q", got.Sticking)
	}
	if got := census(248); len(got.Sticking) != 1 || !strings.Contains(got.Sticking[0], "A button") || got.Overflow {
		t.Errorf("a button that is 8 dp too wide is in the padding: %q", got.Sticking)
	}
	// Too wide for the row, too: it ends beyond the row and beyond the window.
	if got := census(330); len(got.Sticking) != 1 || !got.Overflow {
		t.Errorf("a button across the edge: %q, overflow %v", got.Sticking, got.Overflow)
	}
}

func TestTheCensusFindsATextThatIsDrawnBeyondThePaddingOfItsRow(t *testing.T) {
	// The same row: the inside of it is 280 dp wide, from 40 to 320 in a window of 360 dp. A word that is as wide as the inside is
	// within it, and one that is 20 dp wider hangs out of the row, as a word that is longer than the room of its row does.
	census := func(width float32) Census {
		h := newTested(t, 360, 300, func(c Ctx) {
			Column(c).Background(c.Colors().Surface).Padding(24).Children(func() {
				ListGroup(c, ListItemSpec{
					ID: "item", Title: "Item",
					SubtitleWidget: func(col ui.Color) { Label(c, "A word", Type(BodyMedium), col).Width(width).Shrink(0) },
				})
			})
		})
		h.tester.Frame()
		h.tester.Advance(time.Second)
		return TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 360, 300), nil)
	}
	if got := census(280); len(got.Sticking) != 0 {
		t.Errorf("a text as wide as the inside of the row is inside its padding: %q", got.Sticking)
	}
	if got := census(300); len(got.Sticking) != 1 || !strings.Contains(got.Sticking[0], "A word") || got.Overflow {
		t.Errorf("a text that is 20 dp too wide is in the padding, and not across the edge of the window: %q", got.Sticking)
	}
}

func TestTheCensusFindsAnIconThatTheTableDoesNotHave(t *testing.T) {
	h := newTested(t, 300, 100, func(c Ctx) {
		Row(c).Padding(16).Gap(8).Children(func() {
			Icon(c, "check", 24, c.Colors().OnSurface, 0)
			Icon(c, "no-such-icon", 24, c.Colors().OnSurface, 0)
		})
	})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	got := TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 300, 100), nil)
	if !reflect.DeepEqual(got.MissingIcons, []string{"no-such-icon"}) {
		t.Errorf("missing icons: %v", got.MissingIcons)
	}
}

func TestTheCensusMeasuresTheContrastOfATextThatIsWhollyInTheWindow(t *testing.T) {
	// White on white cannot be read. The text at the bottom of the window has most of its box in it, and is counted as words, but
	// its glyph is cut by the edge of the window, and whatever is left of it is not measured.
	img := image.NewRGBA(image.Rect(0, 0, 40, 40))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	white := ui.Color{R: 255, G: 255, B: 255, A: 255}
	whole := TakeCensus([]*Node{{Role: RoleText, Text: "E", Rect: rect(0, 0, 20, 20), Size: 14, Color: white}}, rect(0, 0, 40, 40), img)
	if whole.Contrast.Measured != 1 || whole.Contrast.Failing != 1 {
		t.Errorf("white text on white in the window is measured and fails: %+v", whole.Contrast)
	}
	cut := TakeCensus([]*Node{{Role: RoleText, Text: "E", Rect: rect(0, 28, 20, 20), Size: 14, Color: white}}, rect(0, 0, 40, 40), img)
	if cut.Words != 1 || cut.Contrast.Measured != 0 {
		t.Errorf("a text that the edge of the window cuts is read as words and not measured: %d words, %+v", cut.Words, cut.Contrast)
	}
}

func TestTheCensusReadsAThinGlyphAgainstItsBackgroundAndNotAgainstItsOwnStrokes(t *testing.T) {
	// A Han character at 14 dp: a dozen pixels of the colour of the text, in the middle of strokes that are all blends of it
	// with the page, in a box that is the page for the most part. The pixels around the dozen are blends, which are not what
	// the text stands on: the page is.
	page := color.RGBA{R: 255, G: 241, B: 232, A: 255}
	blend := color.RGBA{R: 177, G: 136, B: 98, A: 255}
	img := image.NewRGBA(image.Rect(0, 0, 300, 40))
	draw.Draw(img, img.Bounds(), &image.Uniform{page}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(100, 6, 124, 26), &image.Uniform{blend}, image.Point{}, draw.Src)
	text := func(ink color.RGBA) []*Node {
		for y := 14; y < 17; y++ {
			for x := 108; x < 112; x++ {
				img.SetRGBA(x, y, ink)
			}
		}
		return []*Node{{Role: RoleText, Text: "网页界面", Rect: rect(0, 0, 292, 24), Size: 14, Color: ui.Color{R: ink.R, G: ink.G, B: ink.B, A: 255}}}
	}
	dark := color.RGBA{R: 136, G: 82, B: 28, A: 255}
	if got := TakeCensus(text(dark), rect(0, 0, 300, 40), img); got.Contrast.Measured != 1 || got.Contrast.Failing != 0 {
		t.Errorf("dark brown on the page, whatever blends the strokes have: %+v", got.Contrast)
	}
	// A thin text that is nearly the colour of the page is still found out.
	pale := color.RGBA{R: 240, G: 228, B: 218, A: 255}
	if got := TakeCensus(text(pale), rect(0, 0, 300, 40), img); got.Contrast.Failing != 1 {
		t.Errorf("a text that is nearly the colour of the page cannot be read: %+v", got.Contrast)
	}
}

func TestTheCensusKnowsTheTextOfAControlThatIsDisabledWhenItsLineIsTallerThanTheControl(t *testing.T) {
	// The line of a text that is set in the font of Han characters (Noto Sans CJK has a taller line than the font of the
	// Latin letters) is 42 dp high where the button that it is the label of is 40: the rectangles of a real screen. The button
	// is its control all the same, and the text of what is disabled is not held to the contrast of a text; where no control
	// holds it, the same text is measured and found out.
	img := image.NewRGBA(image.Rect(0, 0, 300, 200))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{R: 255, G: 241, B: 232, A: 255}}, image.Point{}, draw.Src)
	button := &Node{Role: RoleButton, Name: "清除已完成", Rect: rect(32, 114, 200, 40)}
	text := &Node{Role: RoleText, Text: "清除已完成", Rect: rect(76, 113, 140, 42), Size: 14, Color: ui.Color{R: 34, G: 26, B: 20, A: 97}}
	if got := TakeCensus([]*Node{button, text}, rect(0, 0, 300, 200), img); got.Contrast.Measured != 0 {
		t.Errorf("the text of a disabled button is not measured, though its line is taller than the button: %+v", got.Contrast)
	}
	if got := TakeCensus([]*Node{text}, rect(0, 0, 300, 200), img); got.Contrast.Measured != 1 || got.Contrast.Failing != 1 {
		t.Errorf("and without a button it is measured, and as dim as it is it fails: %+v", got.Contrast)
	}
	// A text beside the button, whose line overlaps it but whose centre does not, is not the label of the button.
	beside := &Node{Role: RoleText, Text: "next to it", Rect: rect(230, 113, 60, 42), Size: 14, Color: ui.Color{R: 34, G: 26, B: 20, A: 97}}
	if got := TakeCensus([]*Node{button, beside}, rect(0, 0, 300, 200), img); got.Contrast.Measured != 1 {
		t.Errorf("a text that stands by a button is its own: %+v", got.Contrast)
	}
}

func TestTheCensusFindsAPageThatIsTallerThanItsWindow(t *testing.T) {
	// What a page has below the fold is not drawn, and so it is not counted: a census of a page that is cut is of a part of it, and
	// says nothing of the rest. The nodes are those of a window of 360 by 600 dp, with a bar of the navigation at its bottom (a
	// button as high as the bar, whose bottom is the bottom of the window), which leaves the page 536 dp.
	window := rect(0, 0, 360, 600)
	bar := &Node{Role: RoleButton, Name: "Settings", Rect: rect(270, 536, 90, NavBarHeight)}
	line := func(y float32) *Node {
		return &Node{Role: RoleText, Text: "a line of the page", Rect: rect(24, y, 200, 24), Size: 14}
	}
	if got := TakeCensus([]*Node{line(100), line(400), bar}, window, nil); got.Taller || len(got.Over()) != 0 {
		t.Errorf("a page that ends well above its room is not taller than its window: %+v %q", got.TallerBy, got.Over())
	}
	// It ends in the space between two nodes, at the edge of the room; or one of its buttons is cut by the edge, and the bar is
	// drawn over the rest of it (the label of the Save button of the settings, behind the bar, was white on the bar: 1.16).
	cut := []*Node{line(100), line(490), bar}
	if got := TakeCensus(cut, window, nil); !got.Taller || !strings.Contains(got.TallerBy, "ends at 514 of the 536") {
		t.Errorf("a page that ends at the edge of its room is cut: %+v", got)
	}
	save := &Node{Role: RoleButton, Name: "Save", Rect: rect(200, 528, 64, 40)}
	if got := TakeCensus([]*Node{line(100), save, bar}, window, nil); !got.Taller || !strings.Contains(got.TallerBy, `"Save" ends at 568 of the 536`) {
		t.Errorf("a button that the bar is drawn over is cut: %+v", got)
	}
	// Without a bar the window is the room.
	if got := TakeCensus([]*Node{line(100), line(560)}, window, nil); !got.Taller {
		t.Errorf("a page that ends at the bottom of its window is cut: %+v", got)
	}
	if got := TakeCensus([]*Node{line(100), line(500)}, window, nil); got.Taller {
		t.Errorf("a page that ends 76 dp above the bottom of its window is not: %+v", got)
	}
	// The budget names it, unless the page has a list that has more rows than any window shows.
	census := TakeCensus(cut, window, nil)
	if over := census.Over(); len(over) != 1 || !strings.Contains(over[0], "taller than its window") {
		t.Errorf("a page that is cut is over a budget: %q", over)
	}
	census.Scrolls = true
	if over := census.Over(); len(over) != 0 {
		t.Errorf("and is not, where its list is what scrolls: %q", over)
	}
	// A dialog is as tall as its window where the window is narrow, and has its actions at the end: it is the census of its own
	// screen, and is not cut by what it is.
	dialog := []*Node{{Role: RoleDialog, Modal: true, Rect: rect(0, 0, 360, 600)}, {Role: RoleButton, Name: "Close", Modal: true, Rect: rect(250, 540, 90, 40)}}
	if got := TakeCensus(dialog, window, nil); got.Taller {
		t.Errorf("a dialog is not a page that is cut: %+v", got)
	}
}

func TestTheCensusFindsWhatOfADialogIsWhereNobodyCanReachIt(t *testing.T) {
	// A dialog in a window of 360 by 480 dp. Its actions are what closes it: out of the dialog or out of the window, nothing does,
	// and what it holds that is drawn wholly outside it is a thing that no person can read or press.
	window := rect(0, 0, 360, 480)
	dialog := &Node{Role: RoleDialog, Name: "Download mirror", Modal: true, Rect: rect(24, 24, 312, 432)}
	node := func(role Role, name string, r ui.Rect) *Node {
		return &Node{Role: role, Name: name, Modal: true, Scope: "dialog", Rect: r, Enabled: true}
	}
	spilling := func(extra ...*Node) []string {
		return TakeCensus(append([]*Node{dialog}, extra...), window, nil).Spilling
	}
	// The actions are in it, and the content that scrolls is cut by the edge of what scrolls: across the edge of the dialog,
	// and not outside it.
	if got := spilling(node(RoleActions, "Download mirror", rect(48, 392, 264, 40)), node(RoleButton, "A chip", rect(48, 440, 120, 32))); len(got) != 0 {
		t.Errorf("a dialog that holds its actions and a content that is cut at its edge spills nothing: %q", got)
	}
	// The actions are below it, beyond the window.
	if got := spilling(node(RoleActions, "Download mirror", rect(48, 470, 264, 40))); len(got) != 1 || !strings.Contains(got[0], "the actions of the dialog") {
		t.Errorf("the actions that are below the dialog and the window: %q", got)
	}
	// The actions were not drawn: they are in the registry, with no place.
	if got := spilling(&Node{Role: RoleActions, Name: "Download mirror", Modal: true, Scope: "dialog", Enabled: true}); len(got) != 1 || !strings.Contains(got[0], "are not drawn") {
		t.Errorf("the actions that nothing drew: %q", got)
	}
	// A chip that is drawn under the dialog, in the window and wholly out of it.
	if got := spilling(node(RoleActions, "Download mirror", rect(48, 392, 264, 40)), node(RoleButton, "Alibaba Cloud", rect(48, 458, 120, 20))); len(got) != 1 || !strings.Contains(got[0], "Alibaba Cloud") {
		t.Errorf("a chip wholly outside the dialog: %q", got)
	}
	// A dialog that is taller than its window.
	dialog = &Node{Role: RoleDialog, Name: "Download mirror", Modal: true, Rect: rect(24, 24, 312, 560)}
	if got := spilling(); len(got) != 1 || !strings.Contains(got[0], "in a window that is 480 tall") {
		t.Errorf("a dialog that is taller than its window: %q", got)
	}
	// A menu has no dialog.
	menu := []*Node{{Role: RoleMenuItem, Name: "Use", Modal: true, Rect: rect(100, 100, 200, 48)}}
	if got := TakeCensus(menu, window, nil).Spilling; len(got) != 0 {
		t.Errorf("a menu is not a dialog: %q", got)
	}
	// The budget names it.
	if over := TakeCensus([]*Node{dialog, node(RoleActions, "Download mirror", rect(48, 392, 264, 40))}, window, nil).Over(); len(over) == 0 {
		t.Error("a dialog that spills is over a budget")
	}
}

func TestTheCensusFindsAMenuThatGoesOutOfItsWindow(t *testing.T) {
	// A menu that is in the window is where a person can reach every entry of it, by scrolling when it is shorter than they are;
	// one that goes below the window, or beyond the side of it, has entries that nothing can press.
	window := rect(0, 0, 360, 480)
	spilling := func(menu ui.Rect) []string {
		nodes := []*Node{
			{Role: RoleMenu, Modal: true, Enabled: true, Rect: menu},
			{Role: RoleMenuItem, Name: "Use", Modal: true, Enabled: true, Rect: rect(menu.X, menu.Y, menu.W, 48)},
		}
		return TakeCensus(nodes, window, nil).Spilling
	}
	if got := spilling(rect(96, 96, 208, 144)); len(got) != 0 {
		t.Errorf("a menu in its window spills nothing: %q", got)
	}
	if got := spilling(rect(8, 8, 344, 464)); len(got) != 0 {
		t.Errorf("a menu that takes the window up to its margin spills nothing: %q", got)
	}
	if got := spilling(rect(96, 400, 208, 144)); len(got) != 1 || !strings.Contains(got[0], "the menu is at") {
		t.Errorf("a menu below the window: %q", got)
	}
	if got := spilling(rect(200, 96, 208, 144)); len(got) != 1 {
		t.Errorf("a menu beyond the side of the window: %q", got)
	}
	// A menu that no frame has drawn is not on the screen, and is not out of it.
	if got := TakeCensus([]*Node{{Role: RoleMenu, Modal: true, Enabled: true}}, window, nil).Spilling; len(got) != 0 {
		t.Errorf("a menu that was not drawn: %q", got)
	}
}

func TestTheCensusFindsTextsThatAreDrawnOverOneAnother(t *testing.T) {
	// The label of a text field and the line that it holds: at 200 % the field was as high as it was at 100 %, and the label was drawn
	// over the text. Two lines of one text, one under the other, touch and do not overlap, and the label of a field is a little over
	// the text, as Material draws it.
	window := rect(0, 0, 360, 480)
	text := func(name string, r ui.Rect) *Node {
		return &Node{Role: RoleText, Text: name, Size: 14, Enabled: true, Rect: r, Seen: r}
	}
	input := func(r ui.Rect) *Node {
		return &Node{Role: RoleInput, Text: "17-system", Enabled: true, Rect: r, Seen: r}
	}
	overlapping := func(nodes ...*Node) []string { return TakeCensus(nodes, window, nil).Overlapping }
	if got := overlapping(text("Version name", rect(16, 8, 200, 16)), input(rect(16, 22, 200, 24))); len(got) != 0 {
		t.Errorf("a label a little over the text of its field, as Material draws it (2 dp of 16): %q", got)
	}
	if got := overlapping(text("A title", rect(16, 8, 200, 24)), text("A subtitle", rect(16, 32, 200, 20))); len(got) != 0 {
		t.Errorf("two lines that touch: %q", got)
	}
	if got := overlapping(text("A title", rect(16, 8, 100, 24)), text("Another", rect(130, 8, 100, 24))); len(got) != 0 {
		t.Errorf("two texts side by side: %q", got)
	}
	if got := overlapping(text("Version name", rect(16, 8, 200, 32)), input(rect(16, 22, 200, 48))); len(got) != 1 || !strings.Contains(got[0], "Version name") {
		t.Errorf("a label that is 18 dp of its 32 over the text of its field: %q", got)
	}
	if got := overlapping(text("One", rect(16, 8, 100, 24)), text("Two", rect(60, 8, 100, 24))); len(got) != 1 {
		t.Errorf("two texts that are drawn over one another by the half of them: %q", got)
	}
	// What a scroll container clips is not drawn: a row that is scrolled half under a header has the text that shows of it, and
	// the header's text is not drawn over by what is under it.
	row := text("A row", rect(16, 20, 200, 24))
	row.Seen = rect(16, 40, 200, 4)
	if got := overlapping(text("A header", rect(16, 8, 200, 24)), row); len(got) != 0 {
		t.Errorf("a text that is clipped away from what it was over: %q", got)
	}
	// A text that no frame has drawn is not on the screen, and a field that is empty has no text to be drawn over.
	if got := overlapping(text("One", rect(16, 8, 100, 24)), &Node{Role: RoleText, Text: "Two"}); len(got) != 0 {
		t.Errorf("a text that was not drawn: %q", got)
	}
	empty := input(rect(16, 8, 200, 24))
	empty.Text = ""
	if got := overlapping(text("Version name", rect(16, 8, 200, 24)), empty); len(got) != 0 {
		t.Errorf("the label of a field that rests, over the line that has no text: %q", got)
	}
	// A tooltip is over what is under it, and the badge of a destination of the rail is over the label above it by a quarter of the
	// label: no letters touch.
	tip := text("A tooltip", rect(30, 12, 100, 24))
	tip.Scope = "tooltip"
	if got := overlapping(text("A title", rect(16, 8, 200, 24)), tip); len(got) != 0 {
		t.Errorf("a tooltip is over what is under it: %q", got)
	}
	if got := overlapping(text("Plugins", rect(16, 8, 100, 32)), text("1", rect(60, 32, 20, 32))); len(got) != 0 {
		t.Errorf("a badge over a quarter of the label above it: %q", got)
	}
	// The budget names it.
	if over := TakeCensus([]*Node{text("One", rect(16, 8, 100, 24)), text("Two", rect(60, 8, 100, 24))}, window, nil).Over(); len(over) == 0 {
		t.Error("texts that are drawn over one another are over a budget")
	}
}

func TestTheCensusFindsATextThatSticksOutOfItsControl(t *testing.T) {
	// A label that is wider than the destination that it names, with no ellipsis to cut it, is drawn over its neighbours: its
	// element is as wide as the text wants, and the width of it is what the census would compare with that.
	window := rect(0, 0, 360, 480)
	item := func(r ui.Rect) *Node { return &Node{Role: RoleButton, Name: "Settings", Enabled: true, Rect: r} }
	label := func(r ui.Rect) *Node {
		return &Node{Role: RoleText, Text: "Settings", Size: 12, Enabled: true, Rect: r, Seen: r}
	}
	outside := func(nodes ...*Node) []string { return TakeCensus(nodes, window, nil).Outside }
	if got := outside(item(rect(288, 416, 72, 64)), label(rect(296, 448, 56, 16))); len(got) != 0 {
		t.Errorf("a label in its destination: %q", got)
	}
	if got := outside(item(rect(288, 416, 72, 64)), label(rect(288, 448, 72, 16))); len(got) != 0 {
		t.Errorf("a label as wide as its destination: %q", got)
	}
	if got := outside(item(rect(288, 416, 72, 64)), label(rect(281, 448, 86, 16))); len(got) != 1 || !strings.Contains(got[0], "Settings") {
		t.Errorf("a label that is 86 dp wide in a destination of 72: %q", got)
	}
	// A text that is held by no control is not held by any.
	if got := outside(label(rect(281, 448, 86, 16))); len(got) != 0 {
		t.Errorf("a text that no control holds: %q", got)
	}
	// A row that goes under the bar of the navigation is clipped by the page: what is under the bar is not drawn, and what shows of
	// a row is above it.
	under := label(rect(288, 408, 56, 24))
	under.Seen = rect(288, 408, 56, 8)
	if got := outside(item(rect(288, 416, 72, 64)), under); len(got) != 0 {
		t.Errorf("a row that is clipped above the bar: %q", got)
	}
	gone := label(rect(288, 430, 56, 24))
	gone.Seen = ui.Rect{}
	if got := outside(item(rect(288, 416, 72, 64)), gone); len(got) != 0 {
		t.Errorf("a row that is clipped away: %q", got)
	}
	// A label whose line goes below its destination: the line of it is as high as the text wants, 21 dp at 130 %, in a bar of 64.
	if got := outside(item(rect(288, 400, 72, 64)), label(rect(296, 448, 56, 21))); len(got) != 1 {
		t.Errorf("a label of 21 dp from 448 in a destination that ends at 464: %q", got)
	}
	// The budget names it.
	if over := TakeCensus([]*Node{item(rect(288, 416, 72, 64)), label(rect(281, 448, 86, 16))}, window, nil).Over(); len(over) == 0 {
		t.Error("a text that sticks out of its control is over a budget")
	}
}

func TestTheBudgetsOfTheCopyAreNotThoseOfALanguageWithoutSpaces(t *testing.T) {
	// Five sentences of eight words and five small lines are over the budgets of the copy; a page that is across the edge
	// of its window is over a budget of the layout. A language without spaces is held to the second.
	census := Census{
		Overflow: true, Overflowing: []string{"a page"},
		Prose: Prose{Blocks: 5, Texts: []string{"eight words"}}, Helpers: 5, SmallLines: []string{"four words"},
	}
	if got := census.Over(); len(got) != 3 {
		t.Errorf("the page, the sentences and the small lines: %q", got)
	}
	if got := census.OverLayout(); len(got) != 1 || !strings.Contains(got[0], "across an edge of the window") {
		t.Errorf("the page only: %q", got)
	}
}

func TestTheCensusFindsALabelThatNeedsMoreLinesThanItMayHave(t *testing.T) {
	// A word of 48 characters that is broken at characters, in a control whose label may have two lines: where the room has
	// three lines of it the third is an ellipsis, and where it has two it is not cut. The rooms are those of the width of the
	// word, which is measured, so that the test does not depend on the font. (No component of the kit limits the lines of a
	// label that is data; the control is made here.)
	const word = "VFOX_JAVA_GITHUB_MIRROR_VFOX_JAVA_GITHUB_MIRROR_"
	build := func(room float32) func(Ctx) {
		return func(ctx Ctx) {
			ui.Row(ctx.Context).Background(ctx.Colors().Surface).Padding(16).Children(func() {
				ui.Row(ctx.Context).Width(room).Children(func() {
					el, in, node := press(ctx, pressSpec{id: "data", role: RoleButton, name: word})
					el.MinWidth(0).PaddingX(buttonPadding).Shrink(1)
					paintPress(ctx, el, node, Uniform(8), stateLayer(ctx, el, in), ctx.Colors().OnSurface, in.keyboard)
					el.Children(func() {
						Label(ctx, word, Type(LabelLarge), ctx.Colors().OnSurface, Lines(2), Breakable()).Shrink(1)
					})
				})
			})
		}
	}
	width := drawnAtScale(t, 1200, 1, build(1000)).rectOf(RoleText, word).W
	for _, c := range []struct {
		name string
		room float32
		cut  bool
	}{{"three lines", 2*buttonPadding + 0.4*width, true}, {"two lines", 2*buttonPadding + 0.62*width, false}} {
		h := drawnAtScale(t, 1200, 1, build(c.room))
		got := TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 1200, 300), nil)
		if cut := len(got.Cut) == 1; cut != c.cut {
			t.Errorf("in the room of %s the label is cut: %v, want %v (%q)", c.name, cut, c.cut, got.Cut)
		}
	}
}
