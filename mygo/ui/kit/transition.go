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

// Transition is how a view gives way to another one
// (docs/en/dev/design-system.md#motion).
type Transition int

const (
	// FadeThrough fades the old view out while the new one fades in and grows a
	// little: for views that have nothing to do with each other, such as the
	// pages of the navigation.
	FadeThrough Transition = iota
	// SharedAxisX slides the views along the horizontal axis while they fade: for
	// views that belong together, such as a list and the details of one of its
	// items.
	SharedAxisX
)

// SwitchView builds the view that belongs to key. When the key changes, the
// view of the old key gives way to the new one on a slow effects spring: mygo
// keeps a copy of what the old view last showed, under the new one, until it has
// gone.
//
// side tells where a view of SharedAxisX lives, so that it comes from there and
// goes back there: -1 for the list at the start, 1 for the details at the end.
// Keys must be comparable.
func SwitchView(c Ctx, kind Transition, side float32, key any, build func()) ui.Element {
	spring := c.Env.Theme.Motion.SlowEffects.Anim()
	var enter, exit ui.Motion
	switch kind {
	case SharedAxisX:
		enter, exit = ui.Motion{X: 30 * side}, ui.Motion{X: 30 * side}
	default:
		enter, exit = ui.Motion{Scale: 0.92}, ui.Motion{}
	}
	return ui.Column(c.Key(key)).Transition(ui.ElementTransition{
		Animation: &spring, Enter: &enter, Exit: &exit,
	}).Children(build)
}

// Collapsible builds what the function builds under a height that follows a
// spring between 0 and the height of the content, so that content appears and
// disappears by growing. The content is not built while it is closed.
func Collapsible(c Ctx, id string, open bool, build func()) ui.Element {
	spring := c.Env.Theme.Motion.DefaultSpatial.Anim()
	el := ui.Column(c.Key(id)).ClipY().Transition(ui.ElementTransition{Size: true, Animation: &spring})
	if !open {
		el.Height(0)
		if !el.Transitioning() {
			return el
		}
	}
	return el.Children(build)
}
