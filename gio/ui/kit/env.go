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

// Package kit is the Material 3 Expressive toolkit of the Gio user interface:
// the theme, the springs, a small layout library, and the components of
// docs/design-system.md, drawn with Gio.
//
// Gio is an immediate mode toolkit: a frame lays out the whole interface
// again. The toolkit keeps what must outlive a frame (animations, hover, text
// being edited) in a Store, under keys the caller gives, and what a frame
// knows about its widgets (where they are, what they are called, what a click
// on them does) in a Registry, which is how popups find their anchors and how
// the end-to-end tests drive the interface like a user.
package kit

import (
	"image"
	"reflect"
	"time"

	"gioui.org/layout"
	"gioui.org/op"
)

// Widget lays out a part of the interface within the constraints of c and
// returns its size.
type Widget func(c Ctx) layout.Dimensions

// Ctx is the context of a layout: the Gio context, and the environment of
// the window.
type Ctx struct {
	layout.Context
	// Env is shared by all widgets of the window.
	Env *Env
	// scope prefixes the keys of the widgets laid out below, so that lists can
	// repeat widgets that have the same ids.
	scope string
}

// Scope returns a context whose widget keys are prefixed with name.
func (c Ctx) Scope(name string) Ctx {
	c.scope += "/" + name
	return c
}

// key makes the key of the widget id in this scope.
func (c Ctx) key(id string) scopedKey { return scopedKey{c.scope, id} }

type scopedKey struct{ scope, id string }

// DP converts density independent pixels to pixels.
func (c Ctx) DP(v float32) int { return c.Dp(dpUnit(v)) }

// SP converts scaled pixels, which follow the text scale, to pixels.
func (c Ctx) SP(v float32) int { return c.Sp(spUnit(v)) }

// Services is what the window system provides, which the toolkit cannot do
// itself.
type Services interface {
	// Invalidate asks for another frame, from any goroutine.
	Invalidate()
	// Copy puts text on the clipboard.
	Copy(text string)
	// OpenURL opens a web address in the browser of the user.
	OpenURL(url string)
}

// Env is the environment of a window: what its widgets share.
type Env struct {
	// Theme holds the colours, fonts and springs. The application replaces it
	// when the settings change.
	Theme *Theme
	// Store keeps the state of widgets from frame to frame.
	Store *Store
	// Reg lists the interactive widgets and the text of the frame.
	Reg *Registry
	// Services are the window system services.
	Services Services
	// Window is the size of the window in pixels.
	Window image.Point
	// Keyboard says that the last input came from the keyboard, so that the
	// widget in focus shows its focus ring.
	Keyboard bool
	// BottomInset is the height in pixels of what the bottom of the window
	// holds, such as the navigation bar, which snack bars stay above.
	BottomInset int

	popups  []popup
	dialogs []*dialogEntry
	snacks  snackQueue
	hover   hoverState
	frame   uint64
}

// NewEnv creates the environment of a window.
func NewEnv(theme *Theme, services Services) *Env {
	return &Env{Theme: theme, Store: NewStore(), Reg: &Registry{}, Services: services}
}

// BeginFrame prepares a frame: the registry and the popups of the previous
// frame are forgotten.
func (e *Env) BeginFrame(size image.Point) {
	e.frame++
	e.Window = size
	e.Reg.begin()
	e.popups = e.popups[:0]
	e.Store.sweep(e.frame)
}

// Frame is the number of the frame in progress.
func (e *Env) Frame() uint64 { return e.frame }

// Reduced reports whether motion is reduced: springs jump to their targets.
func (e *Env) Reduced() bool { return e.Theme.ReducedMotion }

// Store keeps the state of widgets between frames. An entry that no frame
// used for a while is forgotten.
type Store struct {
	items map[any]*storeItem
}

type storeItem struct {
	value any
	used  uint64
}

// NewStore creates an empty store.
func NewStore() *Store { return &Store{items: make(map[any]*storeItem)} }

func (s *Store) sweep(frame uint64) {
	// Frames are cheap; sweep rarely. Entries that were not used for a few
	// hundred frames belong to something that is no longer on screen.
	if frame%600 != 0 {
		return
	}
	for key, item := range s.items {
		if frame-item.used > 1200 {
			delete(s.items, key)
		}
	}
}

// typedKey is a key of the store: the state of a widget of one type never
// replaces that of a widget of another, even when they have the same id.
type typedKey struct {
	key any
	typ reflect.Type
}

// Use returns the state of type T that the widget with key keeps; the first
// call creates it with its zero value.
func Use[T any](c Ctx, key any) *T {
	k := typedKey{key: key, typ: reflect.TypeFor[T]()}
	item, ok := c.Env.Store.items[k]
	if !ok {
		item = &storeItem{value: new(T)}
		c.Env.Store.items[k] = item
	}
	item.used = c.Env.frame
	return item.value.(*T)
}

// UseID is Use with the key of the widget id in the scope of c.
func UseID[T any](c Ctx, id string) *T { return Use[T](c, c.key(id)) }

// Invalidate asks for another frame as soon as possible.
func (c Ctx) Invalidate() { c.Execute(op.InvalidateCmd{}) }

// InvalidateAt asks for another frame at t.
func (c Ctx) InvalidateAt(t time.Time) { c.Execute(op.InvalidateCmd{At: t}) }
