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

const (
	chipHeight = 32
	chipIcon   = 18
)

// ChipSpec describes a filter chip.
type ChipSpec struct {
	// ID identifies the chip among its siblings between builds.
	ID string
	// Label is the text of the chip.
	Label string
	// Selected tells whether the filter is on.
	Selected bool
	// OnClick runs when the user flips the filter.
	OnClick func()
	// Disabled makes the chip inert.
	Disabled bool
}

// FilterChip is a Material 3 filter chip: an 8 dp pill with an outline while
// the filter is off, and a secondary container with a check mark while it is on
// (docs/en/dev/design-system.md#components). It morphs with a spatial spring: the
// corners grow to the full height when the filter comes on and pinch to 4 dp
// while the chip is pressed, and the check mark slides in by widening the chip.
func FilterChip(c Ctx, s ChipSpec) ui.Element {
	cs := c.Colors()
	el, in, node := press(c, pressSpec{
		id: s.ID, role: RoleButton, name: s.Label, selected: s.Selected, disabled: s.Disabled, onClick: s.OnClick,
	})
	motion := c.Env.Theme.Motion.FastSpatial.Anim()
	selection := el.AnimateTo("selection", Flag(s.Selected), motion)
	pressed := el.AnimateTo("shape", Flag(in.pressed), motion)
	const full = chipHeight / 2
	radius := min(max(Lerp(Lerp(CornerSmall, full, selection), CornerExtraSmall, pressed), 0), full)
	content := cs.OnSurfaceVariant.Mix(cs.OnSecondaryContainer, selection)
	border := cs.OutlineVariant.Alpha(1 - selection)
	if s.Disabled {
		content, border = cs.OnSurface.Alpha(0.38), cs.OnSurface.Alpha(0.12)
	}
	el.Height(chipHeight).PaddingX(12).Radius(radius).TextColor(content).
		Background(cs.SecondaryContainer.Alpha(selection)).Border(1, border)
	paintPress(c, el, node, Uniform(radius), stateLayer(c, el, in), content, in.keyboard)
	return el.Children(func() {
		// The check mark and the space after it have no width while the filter is
		// off, and the row grows to its width on the spring that morphs the chip.
		check := ui.Row(c.Key("check")).ClipX().Transition(ui.ElementTransition{Size: true, Animation: &motion})
		if !s.Selected {
			check.Width(0)
		}
		if s.Selected || check.Transitioning() {
			check.Children(func() {
				Icon(c, "check", chipIcon, content, 0).Margin(0, 8, 0, 0)
			})
		}
		Line(c, s.Label, Type(LabelLarge), content).Shrink(1)
	})
}
