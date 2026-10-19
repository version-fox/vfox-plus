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

// Package kit is the Material 3 Expressive toolkit of the mygo user interface:
// the theme, the springs, and the components of docs/design-system.md, built
// on the elements of mygo's package ui.
//
// mygo gives widgets without a look (buttons, switches, text inputs, popovers,
// dialogs, tooltips, toasts) and flexbox layout; this package gives them the
// look: colours from the Material colour utilities, shapes that morph on
// interaction, state layers, and springs. Components are functions of a Ctx
// that build their elements in place and return the outermost one, which the
// caller may place with the usual element methods (Grow, Margin, Width, ...).
//
// What a frame knows about its widgets (what they are called, what a click on
// them does) goes into a Registry, which is how the end-to-end scenarios drive
// the interface like a user.
package kit

import (
	"github.com/egoist/mygo/ui"
)

// Ctx is the context of a build: mygo's context, and the environment of the
// window.
type Ctx struct {
	*ui.Context
	// Env is shared by all widgets of the window.
	Env *Env
}

// Colors is the colour scheme of the theme.
func (c Ctx) Colors() *Palette { return &c.Env.Theme.Colors }

// Theme is the theme of the window.
func (c Ctx) Theme() *Theme { return c.Env.Theme }

// Row is a row of mygo, which centres its children vertically.
func Row(c Ctx) ui.Element { return ui.Row(c.Context) }

// Column is a column of mygo, which stretches its children to its width.
func Column(c Ctx) ui.Element { return ui.Column(c.Context) }

// Box is a column of mygo.
func Box(c Ctx) ui.Element { return ui.Box(c.Context) }

// Spacer takes the free space of its row or column.
func Spacer(c Ctx) ui.Element { return ui.Spacer(c.Context) }

// Gap is an empty space of the given width and height in dp.
func Gap(c Ctx, width, height float32) ui.Element {
	return ui.Box(c.Context).Size(width, height).Shrink(0)
}

// Env is the environment of a window: what its widgets share.
type Env struct {
	// Theme holds the colours and springs. The application replaces it when
	// the settings change.
	Theme *Theme
	// Reg lists the interactive widgets and the text of the frame.
	Reg *Registry
	// Compact says that the window is narrow, which the navigation shows as a
	// bar at the bottom and which snack bars keep above.
	Compact bool

	frame      uint64
	dialogs    []*dialogEntry
	snacks     []Snack
	snackClose []string
	snackCount int
}

// NewEnv creates the environment of a window.
func NewEnv(theme *Theme) *Env {
	return &Env{Theme: theme, Reg: &Registry{}}
}

// Begin prepares a build of the view: the registry of the previous build is
// forgotten, and the context of mygo is joined with the environment.
func (e *Env) Begin(c *ui.Context) Ctx {
	e.frame++
	e.Reg.begin()
	return Ctx{Context: c, Env: e}
}

// Frame is the number of the build in progress.
func (e *Env) Frame() uint64 { return e.frame }
