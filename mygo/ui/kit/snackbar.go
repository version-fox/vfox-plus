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
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Snack is a message in a snack bar.
type Snack struct {
	// Text is the message.
	Text string
	// Action is the label of a button next to the message, if any.
	Action string
	// OnAction runs when the action button is pressed.
	OnAction func()
	// Persist keeps the snack bar until the user dismisses it; otherwise it goes
	// after four seconds.
	Persist bool
	// Closable adds a button that dismisses the snack bar.
	Closable bool
	// CloseLabel names the close button.
	CloseLabel string
}

// ShowSnackbar shows a snack bar, replacing the one that is showing. It may be
// called from event handlers: the snack bar shows in the next build of the view,
// which the application asks for.
func (e *Env) ShowSnackbar(s Snack) {
	e.snacks = append(e.snacks, s)
}

// closeSnack closes the snack bar with the id in the next build.
func (e *Env) closeSnack(id string) {
	e.snackClose = append(e.snackClose, id)
}

// PendingSnack reports whether a snack bar waits for the next build: the
// application asks for a frame then.
func (e *Env) PendingSnack() bool { return len(e.snacks) > 0 || len(e.snackClose) > 0 }

// LayoutSnackbars shows the snack bars that were asked for and builds the one
// that is showing, above everything else. The application calls it once in every
// build of the view. mygo keeps their time (4 seconds, stopped while the
// pointer or the focus is in one), their stacking and their Escape.
func (e *Env) LayoutSnackbars(c Ctx) {
	for _, id := range e.snackClose {
		c.CloseToast(id)
	}
	e.snackClose = nil
	for _, s := range e.snacks {
		// One snack bar at a time: a new one replaces the one that shows.
		c.CloseToast("")
		e.snackCount++
		timeout := time.Duration(0)
		if s.Persist {
			timeout = -1
		}
		c.AddToast(ui.Toast{
			ID: "snack-" + strconv.Itoa(e.snackCount), Title: s.Text, Timeout: timeout,
			Action: s.Action, OnAction: s.OnAction, Data: s,
		})
	}
	e.snacks = nil
	cs := c.Colors()
	ui.ToastViewportBase(c.Context, func(viewport ui.Element, toasts []ui.Toast) {
		bottom := float32(16)
		if e.Compact {
			bottom += NavBarHeight
		}
		viewport.Padding(16, 16, bottom, 16).AlignItems(ui.Center).Gap(8)
		for _, t := range toasts {
			snack, _ := t.Data.(Snack)
			parts := ui.ToastBase(c.Context, t)
			parts.Root.Row().Gap(8).Padding(6, 8, 6, 16).Radius(CornerMedium).Background(cs.InverseSurface).
				MinHeight(48).MinWidth(min(480, c.windowWidth()-32)).MaxWidth(480)
			Elevation(c, parts.Root, 3)
			slide := c.Env.Theme.Motion.DefaultSpatial.Anim()
			parts.Root.Transition(ui.ElementTransition{
				Animation: &slide,
				Enter:     &ui.Motion{Y: 72},
				Exit:      &ui.Motion{Y: 72},
			})
			parts.Root.Children(func() {
				Label(c, t.Title, Type(BodyMedium), cs.InverseOnSurface).Grow(1).Shrink(1)
				if t.Action != "" {
					action := parts.ActionButton()
					actionNode := c.Env.Reg.Add(Node{
						Role: RoleButton, Name: t.Action, Enabled: true,
						Click: func() {
							if t.OnAction != nil {
								t.OnAction()
							}
							e.closeSnack(t.ID)
						},
					})
					action.Height(40).PaddingX(12).Radius(CornerFull).Justify(ui.Center).
						TextColor(cs.InversePrimary).Draw(func(p *ui.Painter, r ui.Rect) { actionNode.Rect = r })
					action.Children(func() {
						Line(c, t.Action, Type(LabelLarge), cs.InversePrimary)
					})
				}
				if snack.Closable {
					closeButton := parts.CloseButton().Label(snack.CloseLabel).Size(32, 32).Radius(CornerFull).Center().
						TextColor(cs.InverseOnSurface)
					closeNode := c.Env.Reg.Add(Node{
						Role: RoleButton, Name: snack.CloseLabel, Enabled: true,
						Click: func() { e.closeSnack(t.ID) },
					})
					closeButton.Draw(func(p *ui.Painter, r ui.Rect) { closeNode.Rect = r })
					closeButton.Children(func() {
						Icon(c, "close", 20, cs.InverseOnSurface, 0)
					})
				}
			})
		}
	})
}

// windowWidth is the width of the window in dp.
func (c Ctx) windowWidth() float32 {
	w, _ := c.Size()
	return w
}
