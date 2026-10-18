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

	"gioui.org/io/key"
	"gioui.org/layout"
)

// Dialog is a Material 3 dialog: a modal surface with an icon, a title, content
// and actions. It scales in on a spring and fades in.
type Dialog struct {
	// Icon is shown above the title, if any.
	Icon string
	// IconError draws the icon in the colour of errors instead of the secondary
	// colour, for destructive actions.
	IconError bool
	// Title is the headline.
	Title string
	// Content is the body of the dialog, built with the colour of the
	// text on the dialog.
	Content Widget
	// Actions are the buttons at the bottom, from the left.
	Actions []Widget
	// MaxWidth is the widest the dialog gets, in dp; 560 by default.
	MaxWidth float32
	// Tall makes the dialog as high as the window allows, with the content
	// taking the space between the title and the actions: for content that
	// scrolls.
	Tall bool
}

// dialogEntry is a dialog that is open.
type dialogEntry struct {
	id      string
	build   func(c Ctx, close func()) Dialog
	scale   Anim
	closing bool
	shown   bool
}

// ShowDialog opens a dialog. id identifies it: a dialog with the id that is
// already open is left as it is. build is called every frame, with a function
// that closes the dialog.
func (e *Env) ShowDialog(id string, build func(c Ctx, close func()) Dialog) {
	for _, d := range e.dialogs {
		if d.id == id && !d.closing {
			return
		}
	}
	e.dialogs = append(e.dialogs, &dialogEntry{id: id, build: build})
	e.Services.Invalidate()
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

// layoutDialogs draws the open dialogs on top of everything else.
func (e *Env) layoutDialogs(c Ctx) {
	for i := 0; i < len(e.dialogs); i++ {
		d := e.dialogs[i]
		if d.layout(c) {
			e.dialogs = append(e.dialogs[:i], e.dialogs[i+1:]...)
			i--
		}
	}
}

// layout draws a dialog; it reports that the dialog is gone.
func (d *dialogEntry) layout(c Ctx) bool {
	theme := c.Env.Theme
	colors := theme.Colors
	scope := c.Scope("dialog:" + d.id)
	closeDialog := func() {
		d.closing = true
		c.Env.Services.Invalidate()
	}
	spec := d.build(scope, closeDialog)
	if !d.shown {
		d.shown = true
		d.scale.Snap(c, 0)
	}
	target := 1.0
	if d.closing {
		target = 0
	}
	progress := d.scale.To(c, target, theme.Motion.DefaultSpatial)
	effects := minf(1, maxf(0, float32(progress)))
	if d.closing && progress < 0.02 {
		return true
	}
	if d.closing && c.Env.Reduced() {
		return true
	}

	reg := c.Env.Reg
	reg.SetModal(true)
	defer reg.SetModal(false)
	if !d.closing {
		for {
			if _, ok := c.Event(key.Filter{Name: key.NameEscape}); !ok {
				break
			}
			closeDialog()
		}
	}
	// The barrier dims the window and closes the dialog when it is pressed.
	Opacity(c.Ops, effects, func() {
		FillRRect(c.Ops, image.Rectangle{Max: c.Env.Window}, Radii{}, Alpha(colors.Scrim, 0.32))
	})
	scrim(c, d, closeDialog)

	maxWidth := spec.MaxWidth
	if maxWidth == 0 {
		maxWidth = 560
	}
	width := minInt(c.DP(maxWidth), c.Env.Window.X-c.DP(48))
	var parts []FlexChild
	if spec.Icon != "" {
		iconColor := colors.Secondary
		if spec.IconError {
			iconColor = colors.Error
		}
		parts = append(parts, Rigid(Center(Icon(spec.Icon, 24, iconColor, 0))), Rigid(Space(0, 16)))
	}
	titleOptions := []LabelOption{}
	if spec.Icon != "" {
		titleOptions = append(titleOptions, Centered())
	}
	parts = append(parts, Rigid(Label(spec.Title, Type(HeadlineSmall), colors.OnSurface, titleOptions...)))
	if spec.Content != nil {
		if spec.Tall {
			parts = append(parts, Rigid(Space(0, 16)), Expanded(1, spec.Content))
		} else {
			parts = append(parts, Rigid(Space(0, 16)), Rigid(spec.Content))
		}
	}
	if len(spec.Actions) > 0 {
		parts = append(parts, Rigid(Space(0, 24)),
			Rigid(Row(FlexOpts{Main: MainEnd, Gap: 8, Cross: CrossCenter}, actionChildren(spec.Actions)...)))
	}
	body := Pad(All(24), Column(FlexOpts{Cross: CrossStretch, Min: !spec.Tall}, parts...))
	// The content is laid out in the scope of the dialog, in which build
	// finds its fields.
	p := scope.measure(layout.Constraints{Min: image.Pt(minInt(width, c.DP(280)), 0), Max: image.Pt(width, c.Env.Window.Y-c.DP(48))}, body)
	size := p.size()
	at := image.Pt((c.Env.Window.X-size.X)/2, (c.Env.Window.Y-size.Y)/2)
	scale := Lerp(0.8, 1, float32(progress))
	rect := image.Rectangle{Min: at, Max: at.Add(size)}
	Opacity(c.Ops, effects, func() {
		defer scaleAbout(c, image.Pt(at.X+size.X/2, at.Y+size.Y/2), scale)()
		r := c.Round(CornerExtraLarge)
		c.Elevation(rect, r, 3)
		FillRRect(c.Ops, rect, r, colors.SurfaceContainerHigh)
		p.put(c, at)
	})
	reg.Add(Node{Role: RoleDialog, Name: spec.Title, Rect: rect, Enabled: true, Modal: true})
	return false
}

func actionChildren(actions []Widget) []FlexChild {
	children := make([]FlexChild, len(actions))
	for i, a := range actions {
		children[i] = Rigid(a)
	}
	return children
}
