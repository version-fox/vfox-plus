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
	"time"

	"gioui.org/layout"
)

// Snack is a message in a snack bar.
type Snack struct {
	// Text is the message.
	Text string
	// Action is the label of a button next to the message, if any.
	Action string
	// OnAction runs when the action button is pressed.
	OnAction func()
	// Persist keeps the snack bar until the user dismisses it; otherwise it
	// goes after four seconds.
	Persist bool
	// Closable adds a button that dismisses the snack bar.
	Closable bool
	// CloseLabel names the close button.
	CloseLabel string
}

// snackQueue holds the snack bar that is showing.
type snackQueue struct {
	current *snackEntry
}

type snackEntry struct {
	Snack
	shown   time.Time
	slide   Anim
	closing bool
}

// ShowSnackbar shows a snack bar, replacing the one that is showing.
func (e *Env) ShowSnackbar(s Snack) {
	e.snacks.current = &snackEntry{Snack: s}
	e.Services.Invalidate()
}

// snackDuration is how long a snack bar stays that does not persist.
const snackDuration = 4 * time.Second

// layoutSnackbar draws the snack bar, above everything else.
func (e *Env) layoutSnackbar(c Ctx) {
	s := e.snacks.current
	if s == nil {
		return
	}
	theme := c.Env.Theme
	colors := theme.Colors
	if s.shown.IsZero() {
		s.shown = c.Now
		s.slide.Snap(c, 0)
	}
	if !s.Persist && !s.closing && c.Now.Sub(s.shown) > snackDuration {
		s.closing = true
	}
	target := 1.0
	if s.closing {
		target = 0
	}
	progress := s.slide.To(c, target, theme.Motion.DefaultSpatial)
	if !s.Persist && !s.closing {
		c.InvalidateAt(s.shown.Add(snackDuration + time.Millisecond))
	}
	if s.closing && progress < 0.02 {
		e.snacks.current = nil
		return
	}
	dismiss := func() { s.closing = true; c.Env.Services.Invalidate() }

	scope := c.Scope("snackbar")
	margin := c.DP(16)
	width := minInt(c.DP(480), c.Env.Window.X-2*margin)
	parts := []FlexChild{Expanded(1, Label(s.Text, Type(BodyMedium), colors.InverseOnSurface))}
	if s.Action != "" {
		parts = append(parts, Rigid(Space(8, 0)), Rigid(snackAction(s.Action, colors, func() {
			if s.OnAction != nil {
				s.OnAction()
			}
			dismiss()
		})))
	}
	if s.Closable {
		parts = append(parts, Rigid(IconButton{ID: "close", Icon: "close", Tooltip: s.CloseLabel,
			OnClick: dismiss, Color: &colors.InverseOnSurface, Size: 32}.W()))
	}
	body := Pad(Inset{16, 6, 8, 6}, Row(FlexOpts{Cross: CrossCenter}, parts...))
	p := scope.measure(layout.Constraints{Min: image.Pt(width, c.DP(48)), Max: image.Pt(width, c.Env.Window.Y)}, body)
	size := p.size()
	rest := c.Env.Window.Y - c.Env.BottomInset - margin - size.Y
	hidden := c.Env.Window.Y + c.DP(8)
	at := image.Pt((c.Env.Window.X-size.X)/2, int(Lerp(float32(hidden), float32(rest), float32(progress))))
	rect := image.Rectangle{Min: at, Max: at.Add(size)}
	Opacity(c.Ops, minf(1, maxf(0, float32(progress)*1.5)), func() {
		r := c.Round(CornerMedium)
		c.Elevation(rect, r, 3)
		FillRRect(c.Ops, rect, r, colors.InverseSurface)
		p.put(c, at)
	})
}

func snackAction(label string, colors Palette, onClick func()) Widget {
	return Button{ID: "action", Kind: TextButton, Label: label, OnClick: onClick}.withContent(colors.InversePrimary).W()
}

// withContent is the button with the colour of its content replaced.
func (b Button) withContent(content colorValue) Button {
	b.content = &content
	return b
}
