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
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"sort"
	"strconv"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// The census of a screen: what a person reads on it, and what could not be read. It is the census that
// `vfxtool webheap -audit` takes of the web interface (shared/tooling/internal/webheap/audit.js), taken of the
// nodes that a frame of the desktop registered (they say where each text, icon and control is, and how a text
// is set) and of the image of the window (which says what a text stands on). docs/en/dev/design-system.md has the
// budgets that a screen is held to, and Census.Over says which of them it is over. Only tests take it: the
// registry knows the labels and the icons only while it collects them (Registry.CollectText).

// Census is the census of one screen.
type Census struct {
	// Words are the words on the screen, a character of Han counting for one.
	Words int `json:"words"`
	// Prose are the texts of eight words or more: sentences, where a title or a label would do.
	Prose Prose `json:"prose"`
	// Helpers are the small lines (12 dp or less) of four words or more that stand outside every control: what
	// is said under a title. SmallLines are their texts.
	Helpers    int      `json:"helpers"`
	SmallLines []string `json:"smallLines"`
	// Contrast is how the texts stand out from what is behind them.
	Contrast Contrast `json:"contrast"`
	// RowsWithTwoActions are the rows of a list that have two buttons with words on them. A split button is one
	// action (its menu has an icon only), and so are the segments of a button group. TwoActionRows are the
	// names of the rows.
	RowsWithTwoActions int      `json:"rowsWithTwoActions"`
	TwoActionRows      []string `json:"twoActionRows"`
	// Wallpaper are the icons that three rows or more of a list start with, as "icon×rows": an icon that is
	// the same on every row says nothing.
	Wallpaper []string `json:"wallpaper"`
	// OffScale counts the texts by their size when it is not a size of the type scale, and OffScaleTexts says which
	// texts they are.
	OffScale      map[string]int `json:"offScale"`
	OffScaleTexts []string       `json:"offScaleTexts"`
	// Targets are the controls a pointer has to hit.
	Targets Targets `json:"targets"`
	// Overflow says that something is drawn across an edge of the window: a page that is wider than it. Overflowing
	// says what.
	Overflow    bool     `json:"overflow"`
	Overflowing []string `json:"overflowing"`
	// Taller says that the page is taller than its window, or as tall as it: what a page has below the fold is not drawn, so
	// it is not counted, and the census is of a part of the page. A page that has all the room that it needs ends above its
	// padding; one that is cut ends at the edge, or just short of it, in the space between two of its nodes. TallerBy says
	// what ends lowest. Scrolls is for whoever takes the census of a page that has a list with more rows than any window
	// shows, which is as tall as its window by what it is: the census is of the rows that it shows.
	Taller   bool   `json:"taller"`
	TallerBy string `json:"tallerBy,omitempty"`
	Scrolls  bool   `json:"scrolls,omitempty"`
	// Spilling says what of a dialog is where a person cannot reach it: its actions, which are not in it or not in the window
	// (a window that was shorter than the dialog took them, and nothing could close it), and what is drawn wholly outside it.
	// What the dialog cuts at the edge of the content that scrolls is across its edge, and is not spilling.
	Spilling []string `json:"spilling"`
	// Overlapping are the texts that are drawn over one another, by what shows of them inside the scroll containers around them: the
	// label of a text field over the line that it holds, a title over its subtitle. Lines that touch are not.
	Overlapping []string `json:"overlapping"`
	// Outside are the texts that are drawn beyond the control that they are the label of: a label that is as wide as its text wants and
	// the destination that it names is less wide, which the neighbours of it cover, or the edge of the window.
	Outside []string `json:"outside"`
	// Sticking are the controls and the texts that a row of a list holds and that are drawn beyond its padding, which a
	// row wide enough for them does not do: they are nearer the edge of the row than anything else in it. A word that is
	// longer than the room of its row does, and is clipped by the corner of the row, or hangs out of it.
	Sticking []string `json:"sticking"`
	// Cut are the titles (22 dp and more) and the texts of controls that do not fit where they are and end with an
	// ellipsis, in one line or in the lines that they may have. A name or a path that is data, in a line of its own, may
	// be cut short, and is not counted.
	Cut []string `json:"cut"`
	// MissingIcons are the icons that the code asks for and that the table of the icons does not have: they
	// draw an empty square.
	MissingIcons []string `json:"missingIcons"`
}

// Prose is the sentences of a census.
type Prose struct {
	Blocks int `json:"blocks"`
	Words  int `json:"words"`
	// Texts are the sentences, the longest first, each with its number of words.
	Texts []string `json:"texts"`
}

// Contrast is the contrast of the texts of a census, with the ones that fail WCAG AA (4.5:1, and 3:1 for large text).
type Contrast struct {
	Measured int               `json:"measured"`
	Failing  int               `json:"failing"`
	Worst    []ContrastFailure `json:"worst"`
}

// ContrastFailure is a text that is hard to read.
type ContrastFailure struct {
	Ratio float64 `json:"ratio"`
	Text  string  `json:"text"`
}

// Targets are the controls of a census and the smallest of them.
type Targets struct {
	Total   int `json:"total"`
	Under24 int `json:"under24"`
}

var (
	latinWord = regexp.MustCompile(`[A-Za-z0-9][A-Za-z0-9'.\-/]*`)
	// dataToken is what a text holds that is data and no words: an address, a fingerprint of a certificate.
	dataToken = regexp.MustCompile(`https?://\S+|(?:[0-9A-Fa-f]{2}:){7,}[0-9A-Fa-f]{2}`)
	// controls are the nodes that a pointer has to hit.
	controls = map[Role]bool{RoleButton: true, RoleTab: true, RoleSwitch: true, RoleTextField: true, RoleMenuItem: true}
)

// CountWords counts the words of a text: a run of letters and digits is one, and so is a character of Han.
func CountWords(text string) int {
	n := len(latinWord.FindAllString(text, -1))
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			n++
		}
	}
	return n
}

// proseWords counts the words of a text as a sentence has them: the addresses and the fingerprints in it are data.
func proseWords(text string) int { return CountWords(dataToken.ReplaceAllString(text, " ")) }

// TakeCensus takes the census of the nodes of a frame, drawn in a window of that size, as the image shows it. A
// screen with a dialog or a menu open is the dialog or the menu: what is behind them is dimmed, and is the census
// of its own screen. A node that no frame has drawn, or that lies outside the window (scrolled away), is not on
// the screen. The image may be nil, and then the contrast is not measured.
func TakeCensus(nodes []*Node, window ui.Rect, img image.Image) Census {
	census := Census{OffScale: map[string]int{}, Wallpaper: []string{}, MissingIcons: []string{}, Cut: []string{}, Sticking: []string{}, Overlapping: []string{}, Outside: []string{}}
	modal := false
	for _, n := range nodes {
		modal = modal || n.Modal
	}
	var shown []*Node
	for _, n := range nodes {
		if n.Visible() && (n.Modal || !modal) && onScreen(n, window) {
			shown = append(shown, n)
		}
	}

	allowed := map[float32]bool{}
	for _, style := range scale {
		allowed[style.Size] = true
	}
	var prose []*Node
	for _, n := range shown {
		switch n.Role {
		case RoleText:
			if n.Text == "" {
				continue
			}
			words := CountWords(n.Text)
			census.Words += words
			if !n.Mono && proseWords(n.Text) >= 8 {
				prose = append(prose, n)
			}
			counted := n.Size >= 22 || within(n, shown, func(o *Node) bool { return controls[o.Role] })
			if n.Natural > 0 && n.Rect.W < n.Natural-cutSlack && counted {
				census.Cut = append(census.Cut, fmt.Sprintf("%q is %.0f dp wide where it has %.0f", truncate(n.Text, 30), n.Natural, n.Rect.W))
			}
			if n.Lines > 1 && n.Needs > n.Lines && counted {
				census.Cut = append(census.Cut, fmt.Sprintf("%q needs %d lines where it has %d", truncate(n.Text, 30), n.Needs, n.Lines))
			}
			if n.Size > 0 && !allowed[n.Size] {
				size := strconv.Itoa(int(math.Round(float64(n.Size))))
				census.OffScale[size]++
				census.OffScaleTexts = append(census.OffScaleTexts, size+": "+truncate(n.Text, 30))
			}
			if n.Size <= 12 && words >= 4 && !n.Mono && !within(n, shown, func(o *Node) bool { return controls[o.Role] }) {
				census.Helpers++
				census.SmallLines = append(census.SmallLines, truncate(n.Text, 60))
			}
			// A glyph that the edge of the window cuts, or a scroll container, is not what a person reads, and what is left of it may be none
			// of its ink.
			if img != nil && wholly(n.Rect, window) && !n.Clipped && !within(n, shown, func(o *Node) bool { return controls[o.Role] && !o.Enabled }) {
				census.measure(n, img)
			}
		case RoleIcon:
			if !HasIcon(n.Name) {
				census.MissingIcons = append(census.MissingIcons, n.Name)
			}
		}
		if controls[n.Role] {
			census.Targets.Total++
			if math.Min(float64(n.Rect.W), float64(n.Rect.H)) < 24 {
				census.Targets.Under24++
			}
		}
		if n.Rect.X < window.X-0.5 || n.Rect.X+n.Rect.W > window.X+window.W+0.5 {
			census.Overflow = true
			census.Overflowing = append(census.Overflowing, fmt.Sprintf("%s %q x %.0f to %.0f of %.0f", n.Role, truncate(n.Text+n.Name, 30), n.Rect.X, n.Rect.X+n.Rect.W, window.W))
		}
	}
	if !modal {
		census.taller(shown, window)
	} else {
		census.spilling(nodes, shown, window)
	}
	census.overlap(shown)
	census.outside(shown)
	sort.SliceStable(prose, func(i, j int) bool { return proseWords(prose[i].Text) > proseWords(prose[j].Text) })
	census.Prose.Blocks = len(prose)
	for _, n := range prose {
		census.Prose.Words += proseWords(n.Text)
		census.Prose.Texts = append(census.Prose.Texts, fmt.Sprintf("%d: %s", proseWords(n.Text), truncate(n.Text, 70)))
	}
	sort.Slice(census.Contrast.Worst, func(i, j int) bool { return census.Contrast.Worst[i].Ratio < census.Contrast.Worst[j].Ratio })
	if len(census.Contrast.Worst) > 4 {
		census.Contrast.Worst = census.Contrast.Worst[:4]
	}
	census.rows(shown)
	sort.Strings(census.MissingIcons)
	return census
}

// pageEndRoom is how far above the bottom of its room the last thing of a page ends, in dp, where the page has all the room that
// it needs: more than its padding at the end, and more than the space between any two of its nodes.
const pageEndRoom = 48

// taller says whether the page, which is every node of the screen that is not in a dialog, goes beyond its window or up to it.
// The room of a page is the window, or what the bar of the navigation leaves of it: a button as high as the bar (BarHeight, which is
// more than NavBarHeight where the text is larger), whose bottom is the bottom of the window.
func (c *Census) taller(shown []*Node, window ui.Rect) {
	room := window.Y + window.H
	for _, n := range shown {
		if n.Role == RoleButton && n.Rect.H >= NavBarHeight-0.5 && n.Rect.H <= NavBarHeight+barGrowth()+0.5 && math.Abs(float64(n.Rect.Y+n.Rect.H-room)) < 0.5 {
			room = min(room, n.Rect.Y)
		}
	}
	var lowest *Node
	for _, n := range shown {
		if n.Rect.Y >= room-0.5 {
			continue // the bar of the navigation, and what is in it
		}
		if lowest == nil || n.Rect.Y+n.Rect.H > lowest.Rect.Y+lowest.Rect.H {
			lowest = n
		}
	}
	if lowest != nil && lowest.Rect.Y+lowest.Rect.H > room-pageEndRoom {
		c.Taller = true
		c.TallerBy = fmt.Sprintf("%s %q ends at %.0f of the %.0f that the page has", lowest.Role, truncate(lowest.Text+lowest.Name, 30), lowest.Rect.Y+lowest.Rect.H, room)
	}
}

// cutSlack is how much narrower than the width that the engine measures for a text of a single line its room may be, in dp, before the
// text is said to be cut: what a float of the layout leaves. The engine ends a line with an ellipsis for any width that it has less
// than the text wants, and a slack of a dp, which the census had, hid a label that was short by 0.4 dp ("Settin..." on macOS, in Verdana at
// 200 %, in a slot of 72 dp where the label wants 72.4).
const cutSlack = 0.05

// lineSlack is how far, in dp, the line of a text may go beyond its control: the engine makes a line as high as a whole dp, and the
// font that Han characters fall back to has a line a little higher than the one that the label was given (19 dp for 18.4, in a bar at
// 115 %), where the letters of it are far from the edge. A label that was cut by the foot of the window was 5 dp beyond.
const lineSlack = 1

// outside says which texts of the screen are drawn beyond the control that holds them, which is the control whose box has the centre
// of the text in it (within): a text is the label of its control, and is in it, where its line is wide enough for the text, or ends
// with an ellipsis where it is not.
func (c *Census) outside(shown []*Node) {
	for _, n := range shown {
		// A text that a scroll container clips away entirely is not drawn, and one that shows in part is held by what is under the
		// part that shows: a row that goes under the bar of the navigation, or under the field of a search, is not in them.
		if n.Role != RoleText || n.Text == "" || n.Seen.W <= 0 || n.Seen.H <= 0 {
			continue
		}
		x, y := n.Seen.X+n.Seen.W/2, n.Seen.Y+n.Seen.H/2
		for _, o := range shown {
			if o == n || !controls[o.Role] || !o.Visible() || x < o.Rect.X || x > o.Rect.X+o.Rect.W || y < o.Rect.Y || y > o.Rect.Y+o.Rect.H {
				continue
			}
			if n.Rect.X < o.Rect.X-0.5 || n.Rect.X+n.Rect.W > o.Rect.X+o.Rect.W+0.5 ||
				n.Rect.Y < o.Rect.Y-lineSlack || n.Rect.Y+n.Rect.H > o.Rect.Y+o.Rect.H+lineSlack {
				c.Outside = append(c.Outside, fmt.Sprintf("%q is at x %.0f to %.0f and y %.0f to %.0f, where its %s is at %.0f to %.0f and %.0f to %.0f", truncate(n.Text, 30), n.Rect.X, n.Rect.X+n.Rect.W, n.Rect.Y, n.Rect.Y+n.Rect.H, o.Role, o.Rect.X, o.Rect.X+o.Rect.W, o.Rect.Y, o.Rect.Y+o.Rect.H))
			}
			break
		}
	}
}

// overlapShare is how much of the lower of two lines of text the other may be drawn over, before their letters touch. A line of
// text has room above its letters and below them, a quarter of its height each, so that two lines may be over one another by what
// the rooms of both are, which is the half of the lower, and the letters are apart: the label of a field is over the line that it
// holds by a fifth of its own height, as Material draws it, and the badge of a destination of the rail is over the label above it by
// a quarter of that label.
const overlapShare = 0.4

// overlap says which texts of the screen are drawn over one another: the lines of text that the labels draw and those that the text
// fields hold. They overlap when the part that shows of both, which is inside the scroll containers around them, is wider than a
// stroke of a letter, and taller than overlapShare of the lower of the lines, where the line is whole: a line that a scroll
// container cuts is as low as what shows, and is no more over a header than what shows of it.
func (c *Census) overlap(shown []*Node) {
	var texts []*Node
	for _, n := range shown {
		// A tooltip is shown over what is under it.
		if (n.Role == RoleText || n.Role == RoleInput) && n.Text != "" && n.Seen.W > 0 && n.Seen.H > 0 && n.Scope != "tooltip" {
			texts = append(texts, n)
		}
	}
	for i, a := range texts {
		for _, b := range texts[i+1:] {
			common := intersection(a.Seen, b.Seen)
			if lower := min(a.Rect.H, b.Rect.H); common.W > 2 && common.H > overlapShare*lower {
				c.Overlapping = append(c.Overlapping, fmt.Sprintf("%q is drawn over %q, by %.0f dp of %.0f", truncate(a.Text, 30), truncate(b.Text, 30), common.H, lower))
			}
		}
	}
}

// spilling says what of the dialog or the menu on the screen is out of its reach: for a menu, that its panel is not in the window,
// for a dialog, the actions that are not drawn, or not in the dialog and in the window, and the nodes of it that are wholly outside
// it. A dialog is the node of its panel, and its actions are the node of their row, which is there whether it was drawn or not.
func (c *Census) spilling(nodes, shown []*Node, window ui.Rect) {
	inside := func(r, outer ui.Rect) bool {
		return r.X >= outer.X-0.5 && r.Y >= outer.Y-0.5 && r.X+r.W <= outer.X+outer.W+0.5 && r.Y+r.H <= outer.Y+outer.H+0.5
	}
	for _, n := range nodes {
		if n.Role == RoleMenu && n.Visible() && !inside(n.Rect, window) {
			c.Spilling = append(c.Spilling, fmt.Sprintf("the menu is at %.0f to %.0f and %.0f to %.0f in a window that is %.0f by %.0f", n.Rect.X, n.Rect.X+n.Rect.W, n.Rect.Y, n.Rect.Y+n.Rect.H, window.W, window.H))
		}
	}
	var dialog *Node
	for _, n := range shown {
		if n.Role == RoleDialog {
			dialog = n
		}
	}
	if dialog == nil {
		return
	}
	name := truncate(dialog.Name, 30)
	if !inside(dialog.Rect, window) {
		c.Spilling = append(c.Spilling, fmt.Sprintf("the dialog %q is at %.0f to %.0f in a window that is %.0f tall", name, dialog.Rect.Y, dialog.Rect.Y+dialog.Rect.H, window.H))
	}
	for _, n := range nodes {
		if n.Role == RoleActions && n.Modal {
			switch {
			case !n.Visible():
				c.Spilling = append(c.Spilling, fmt.Sprintf("the actions of the dialog %q are not drawn", name))
			case !inside(n.Rect, dialog.Rect) || !inside(n.Rect, window):
				c.Spilling = append(c.Spilling, fmt.Sprintf("the actions of the dialog %q are at %.0f to %.0f where it is %.0f to %.0f", name, n.Rect.Y, n.Rect.Y+n.Rect.H, dialog.Rect.Y, dialog.Rect.Y+dialog.Rect.H))
			}
		}
	}
	for _, n := range shown {
		if n.Scope == "dialog" && n != dialog && n.Role != RoleActions && !overlaps(n.Rect, dialog.Rect) {
			c.Spilling = append(c.Spilling, fmt.Sprintf("%s %q is drawn at %.0f to %.0f, outside the dialog", n.Role, truncate(n.Text+n.Name, 30), n.Rect.Y, n.Rect.Y+n.Rect.H))
		}
	}
}

// rows takes the census of the rows of the lists: the buttons that each has, and the icon that each starts with.
func (c *Census) rows(shown []*Node) {
	var lists [][]*Node
	var current []*Node
	flush := func() {
		if len(current) > 0 {
			lists = append(lists, current)
			current = nil
		}
	}
	for _, row := range shown {
		if row.Role != RoleListItem {
			continue
		}
		actions := 0
		for _, n := range shown {
			if n.Role == RoleButton && n.Tooltip == "" && !n.Choice && CountWords(n.Name) > 0 && heldBy(n, row) {
				actions++
			}
		}
		if actions >= 2 {
			c.RowsWithTwoActions++
			c.TwoActionRows = append(c.TwoActionRows, truncate(row.Name, 40))
		}
		for _, n := range shown {
			// A text is held to the padding as a control is, unless a control holds it: that is the control's to be inside.
			in := controls[n.Role] || (n.Role == RoleText && !within(n, shown, func(o *Node) bool { return controls[o.Role] }))
			if in && startsInRow(n, row) && (n.Rect.X < row.Rect.X+row.PadLeft-0.5 || n.Rect.X+n.Rect.W > row.Rect.X+row.Rect.W-row.PadRight+0.5) {
				c.Sticking = append(c.Sticking, fmt.Sprintf("%s %q at %.0f to %.0f in the row %q, whose inside is %.0f to %.0f", n.Role, truncate(n.Name+n.Text, 30), n.Rect.X, n.Rect.X+n.Rect.W,
					truncate(row.Name, 30), row.Rect.X+row.PadLeft, row.Rect.X+row.Rect.W-row.PadRight))
			}
		}
		// A list runs from its first item to its last; the items of the registry are in the order they were built.
		switch row.Position {
		case PosSingle:
			flush()
			current = []*Node{row}
			flush()
		case PosFirst:
			flush()
			current = []*Node{row}
		case PosLast:
			current = append(current, row)
			flush()
		default:
			current = append(current, row)
		}
	}
	flush()
	for _, list := range lists {
		names := map[string]int{}
		for _, row := range list {
			if icon := leading(row, shown); icon != "" {
				names[icon]++
			}
		}
		var walls []string
		for name, n := range names {
			if n >= 3 {
				walls = append(walls, name+"×"+strconv.Itoa(n))
			}
		}
		sort.Strings(walls)
		c.Wallpaper = append(c.Wallpaper, walls...)
	}
}

// heldBy reports whether the widget is in the row: it is drawn inside it and was built in it, which tells it from a bar
// that is drawn over the last row of a page.
func heldBy(n, row *Node) bool { return n.Inside(row) && (row.Holds == "" || n.Scope == row.Holds) }

// startsInRow reports whether the widget was built in the row and starts inside it, in the band of the row: it may end
// beyond the row, which is what a widget that is too wide for it does.
func startsInRow(n, row *Node) bool {
	return (row.Holds == "" || n.Scope == row.Holds) && n.Visible() &&
		n.Rect.Y >= row.Rect.Y-0.5 && n.Rect.Y+n.Rect.H <= row.Rect.Y+row.Rect.H+0.5 &&
		n.Rect.X >= row.Rect.X-0.5 && n.Rect.X < row.Rect.X+row.Rect.W
}

// leading is the icon that a row starts with, or "" for a row that starts with something else (an avatar, a text).
func leading(row *Node, shown []*Node) string {
	for _, n := range shown {
		if n.Role == RoleIcon && n.Inside(row) && n.Rect.X-row.Rect.X < 40 && !within(n, shown, func(o *Node) bool { return controls[o.Role] }) {
			return n.Name
		}
	}
	return ""
}

// inkShare is how small a part of the box of a text, one pixel in so many, may be of the colour of the text for those pixels to
// say where its glyphs are.
const inkShare = 20

// measure takes the contrast of a text against what stands behind it: the colour that most of the pixels around its ink
// have in the image, which the colour of the text, with its opacity, is laid over. The ink of an opaque text is the pixels
// that have the colour of the text itself, and what is behind it is read in the 3 px around them, which is what the glyphs
// stand on whatever the box of the line holds: a large or a bold glyph fills more than half of its box; the box of a line
// of large text is as high as the control that holds it, so that the border is a part of it; and it is higher than an avatar,
// so that the page is. A text with no pixel of its own colour (a thin one, or one with an opacity) has the box.
func (c *Census) measure(n *Node, img image.Image) {
	box := image.Rect(int(math.Floor(float64(n.Rect.X))), int(math.Floor(float64(n.Rect.Y))),
		int(math.Ceil(float64(n.Rect.X+n.Rect.W))), int(math.Ceil(float64(n.Rect.Y+n.Rect.H)))).Intersect(img.Bounds())
	if box.Empty() {
		return
	}
	ink := [3]uint8{n.Color.R, n.Color.G, n.Color.B}
	opaque := n.Color.A == 255
	region := box
	if opaque {
		found, count := image.Rectangle{}, 0
		for y := box.Min.Y; y < box.Max.Y; y++ {
			for x := box.Min.X; x < box.Max.X; x++ {
				if pixel(img, x, y) == ink {
					found = found.Union(image.Rect(x, y, x+1, y+1))
					count++
				}
			}
		}
		// The pixels of the colour of the text say where its glyphs are when there are enough of them. A thin glyph, a Han
		// character at 14 dp, has a few of them in the middle of its strokes, all of them blends, and what is around those
		// is blends, too: its box is read whole, where the background is most of it.
		if count*inkShare >= box.Dx()*box.Dy() {
			region = found.Inset(-3).Intersect(box)
		}
	}
	counts := map[[3]uint8]int{}
	// Text that nothing but its own colour stands on is not to be read, so it stays its own background.
	behind, most := ink, 0
	for y := region.Min.Y; y < region.Max.Y; y++ {
		for x := region.Min.X; x < region.Max.X; x++ {
			key := pixel(img, x, y)
			if opaque && key == ink {
				continue
			}
			counts[key]++
			if counts[key] > most {
				most, behind = counts[key], key
			}
		}
	}
	alpha := float64(n.Color.A) / 255
	var shown [3]float64
	for i, v := range [3]uint8{n.Color.R, n.Color.G, n.Color.B} {
		shown[i] = float64(v)*alpha + float64(behind[i])*(1-alpha)
	}
	found := contrast(shown, [3]float64{float64(behind[0]), float64(behind[1]), float64(behind[2])})
	large := n.Size >= 24 || (n.Size >= 18.66 && n.Weight >= 700)
	limit := 4.5
	if large {
		limit = 3
	}
	c.Contrast.Measured++
	if found < limit {
		c.Contrast.Failing++
		c.Contrast.Worst = append(c.Contrast.Worst, ContrastFailure{Ratio: math.Round(found*100) / 100, Text: truncate(n.Text, 34)})
	}
}

// pixel is the colour of a pixel of the image, in 8 bits to a channel. The window of a test is an *image.RGBA, which is
// read without a detour.
func pixel(img image.Image, x, y int) [3]uint8 {
	if rgba, ok := img.(*image.RGBA); ok {
		i := rgba.PixOffset(x, y)
		return [3]uint8{rgba.Pix[i], rgba.Pix[i+1], rgba.Pix[i+2]}
	}
	if c, ok := color.RGBAModel.Convert(img.At(x, y)).(color.RGBA); ok {
		return [3]uint8{c.R, c.G, c.B}
	}
	return [3]uint8{}
}

// contrast is the WCAG contrast ratio of two colours.
func contrast(a, b [3]float64) float64 {
	light, dim := luminance(a), luminance(b)
	if dim > light {
		light, dim = dim, light
	}
	return (light + 0.05) / (dim + 0.05)
}

func luminance(c [3]float64) float64 {
	channel := func(v float64) float64 {
		v /= 255
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*channel(c[0]) + 0.7152*channel(c[1]) + 0.0722*channel(c[2])
}

// Over names the budgets of docs/en/dev/design-system.md#budgets that the census is over, each with its number. A
// screen within the budgets gives none.
func (c Census) Over() []string { return c.over(true) }

// OverLayout is Over without the budgets of the copy, the sentences and the small lines, which count the words as English
// has them. A language that has no spaces is held to the rest: a character of Han is a word for the census, and a sentence
// of eight of them is a short one.
func (c Census) OverLayout() []string { return c.over(false) }

func (c Census) over(withCopy bool) []string {
	var over []string
	add := func(format string, args ...any) { over = append(over, fmt.Sprintf(format, args...)) }
	if c.Contrast.Failing > 0 {
		add("%d texts under the contrast of WCAG AA (%v)", c.Contrast.Failing, c.Contrast.Worst)
	}
	if c.Overflow {
		add("something is drawn across an edge of the window: %q", c.Overflowing)
	}
	if c.Taller && !c.Scrolls {
		add("the page is taller than its window, and what it has below the fold is not counted: %s", c.TallerBy)
	}
	if len(c.Spilling) > 0 {
		add("%d things of a dialog are where nobody can reach them: %q", len(c.Spilling), c.Spilling)
	}
	if c.Targets.Under24 > 0 {
		add("%d controls under 24 dp", c.Targets.Under24)
	}
	for size, n := range c.OffScale {
		add("%d texts of %s dp, which is not a size of the type scale: %v", n, size, c.OffScaleTexts)
	}
	if c.RowsWithTwoActions > 0 {
		add("%d rows with two buttons that have words on them: %q", c.RowsWithTwoActions, c.TwoActionRows)
	}
	if len(c.Wallpaper) > 0 {
		add("rows that start with the same icon: %v", c.Wallpaper)
	}
	if withCopy && c.Prose.Blocks > 3 {
		add("%d sentences of eight words or more (at most 3): %q", c.Prose.Blocks, c.Prose.Texts)
	}
	if withCopy && c.Helpers > 3 {
		add("%d small lines of four words or more (at most 3): %q", c.Helpers, c.SmallLines)
	}
	if len(c.Overlapping) > 0 {
		add("%d texts are drawn over one another: %q", len(c.Overlapping), c.Overlapping)
	}
	if len(c.Outside) > 0 {
		add("%d texts are drawn beyond their control: %q", len(c.Outside), c.Outside)
	}
	if len(c.Sticking) > 0 {
		add("%d controls and texts drawn beyond the padding of their row: %q", len(c.Sticking), c.Sticking)
	}
	if len(c.Cut) > 0 {
		add("%d texts that do not fit and are cut: %q", len(c.Cut), c.Cut)
	}
	if len(c.MissingIcons) > 0 {
		add("icons that the table does not have: %v", c.MissingIcons)
	}
	sort.Strings(over)
	return over
}

// onScreen reports whether a node is where a person sees it: a text that is drawn, with more than half of its box in the
// window (the line that begins below the last one that fits is not read, and its ink is not drawn), and any other node
// that is partly in it.
func onScreen(n *Node, window ui.Rect) bool {
	if !overlaps(n.Rect, window) {
		return false
	}
	if n.Role != RoleText {
		return true
	}
	inside := min(n.Rect.Y+n.Rect.H, window.Y+window.H) - max(n.Rect.Y, window.Y)
	return inside > n.Rect.H/2
}

// wholly reports whether the rectangle is wholly in the window.
func wholly(r, window ui.Rect) bool {
	return r.X >= window.X-0.5 && r.Y >= window.Y-0.5 && r.X+r.W <= window.X+window.W+0.5 && r.Y+r.H <= window.Y+window.H+0.5
}

// overlaps reports whether the rectangle is at least partly in the window.
func overlaps(r, window ui.Rect) bool {
	return r.X < window.X+window.W && r.X+r.W > window.X && r.Y < window.Y+window.H && r.Y+r.H > window.Y
}

// within reports whether another node of the screen, of the kind that the test says, holds the node: the centre of the node is
// in it. A text is held by the control that it is the label of, whatever the height of its line: where a Han character is set
// in another font than the Latin letters around it, the line is taller than the font of the control, and a text that was 42 dp high
// in a button of 40 was not inside it.
func within(n *Node, shown []*Node, kind func(*Node) bool) bool {
	x, y := n.Rect.X+n.Rect.W/2, n.Rect.Y+n.Rect.H/2
	for _, o := range shown {
		if o != n && kind(o) && o.Visible() &&
			x >= o.Rect.X && x <= o.Rect.X+o.Rect.W && y >= o.Rect.Y && y <= o.Rect.Y+o.Rect.H {
			return true
		}
	}
	return false
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
