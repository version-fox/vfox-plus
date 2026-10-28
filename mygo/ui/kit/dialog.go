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
	"github.com/egoist/mygo/ui"
)

// DialogSpec describes a Material 3 dialog: a modal surface with an icon, a title,
// content and actions. It scales in on a spring and fades in.
type DialogSpec struct {
	// Icon is shown above the title, if any.
	Icon string
	// IconError draws the icon in the colour of errors instead of the secondary
	// colour, for destructive actions.
	IconError bool
	// Title is the headline.
	Title string
	// Content builds the body of the dialog.
	Content func()
	// Actions build the buttons at the bottom, from the left.
	Actions []func()
	// MaxWidth is the widest the dialog gets, in dp; 560 by default.
	MaxWidth float32
	// Tall makes the dialog as high as the window allows, with the content taking
	// the space between the title and the actions: for content that scrolls.
	Tall bool
}

// dialogEntry is a dialog that is open.
type dialogEntry struct {
	id      string
	build   func(c Ctx, close func()) DialogSpec
	closing bool
	shown   bool
}

// ShowDialog opens a dialog. id identifies it: a dialog with the id that is
// already open is left as it is. build is called for every build of the view,
// with a function that closes the dialog; the application builds the dialogs
// with LayoutDialogs.
func (e *Env) ShowDialog(id string, build func(c Ctx, close func()) DialogSpec) {
	for _, d := range e.dialogs {
		if d.id == id && !d.closing {
			return
		}
	}
	e.dialogs = append(e.dialogs, &dialogEntry{id: id, build: build})
}

// HasDialog reports whether a dialog is open.
func (e *Env) HasDialog() bool {
	for _, d := range e.dialogs {
		if !d.closing {
			return true
		}
	}
	return false
}

// LayoutDialogs builds the open dialogs, above everything else. The application
// calls it once in every build of the view, after its pages.
func (e *Env) LayoutDialogs(c Ctx) {
	for i := 0; i < len(e.dialogs); i++ {
		if e.dialogs[i].layout(c) {
			e.dialogs = append(e.dialogs[:i], e.dialogs[i+1:]...)
			i--
		}
	}
}

// layout builds a dialog; it reports that the dialog is gone.
func (d *dialogEntry) layout(c Ctx) bool {
	cs := c.Colors()
	closeDialog := func() {
		d.closing = true
		c.Invalidate()
	}
	spec := d.build(c, closeDialog)
	motion := c.Env.Theme.Motion
	reg := c.Env.Reg
	open, arrived := true, false
	dialog := ui.DialogBase(c.Context, &open, func(backdrop, panel ui.Element) {
		reg.SetModal(true)
		defer reg.SetModal(false)
		visible := d.shown && !d.closing
		fade := backdrop.AnimateTo("fade", Flag(visible), motion.DefaultEffects.Anim())
		arrived = d.closing && backdrop.Arrived("fade")
		backdrop.Background(cs.Scrim.Alpha(0.32 * fade))
		pose := ui.Pose{Scale: c.Env.scaled(0.8), Fade: 1}
		if visible {
			pose = ui.Pose{}
		}
		panel.AnimatePose(pose, motion.DefaultSpatial.Anim())

		maxWidth := spec.MaxWidth
		if maxWidth == 0 {
			maxWidth = 560
		}
		windowWidth, windowHeight := c.Size()
		width := min(maxWidth, windowWidth-48)
		node := reg.Add(Node{Role: RoleDialog, Name: spec.Title, Enabled: true, Modal: true})
		panel.MinWidth(min(width, 280)).MaxWidth(width).Padding(24).Radius(CornerExtraLarge).
			Background(cs.SurfaceContainerHigh).MaxHeight(windowHeight - 48).Gap(0).Label(spec.Title).
			Draw(func(_ *ui.Painter, r ui.Rect) { node.Rect = r })
		if spec.Tall {
			panel.Height(windowHeight - 48)
		}
		Elevation(c, panel, 3)
		panel.Children(func() {
			defer reg.EnterScope("dialog")()
			centred := spec.Icon != ""
			if centred {
				iconColor := cs.Secondary
				if spec.IconError {
					iconColor = cs.Error
				}
				Icon(c, spec.Icon, 24, iconColor, 0).AlignSelf(ui.Center).Margin(0, 0, 16, 0)
			}
			var title []LabelOption
			if centred {
				title = append(title, Centered())
			}
			Label(c, spec.Title, Type(HeadlineSmall), cs.OnSurface, title...).Role(ui.RoleHeading).Level(2)
			if spec.Content != nil {
				content := ui.Column(c.Context).Margin(16, 0, 0, 0).Children(spec.Content)
				if spec.Tall {
					content.Grow(1).Shrink(1)
				}
			}
			if len(spec.Actions) > 0 {
				ui.Row(c.Context).Justify(ui.End).Gap(8).Margin(24, 0, 0, 0).Children(func() {
					for _, action := range spec.Actions {
						action()
					}
				})
			}
		})
	})
	// Escape and a press on the scrim close the dialog, after the spring has
	// taken it out.
	if !open || dialog.Dismissed() {
		d.closing = true
		c.Invalidate()
	}
	if !d.shown {
		d.shown = true
		c.AnimationFrame()
	}
	return arrived
}
