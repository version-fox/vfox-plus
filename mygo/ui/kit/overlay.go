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

// popIn gives the element that has just appeared, a dialog or a menu, the
// entrance of Material 3 Expressive: it grows from the scale from to its size
// on a spatial spring while it fades in. originX and originY, from 0 to 1, are
// where it grows from in its box. mygo animates an element from its first
// pose, so the element shows the pose of the beginning in the build that creates
// it, and moves to its size in the next one.
func popIn(c Ctx, el ui.Element, from, originX, originY float32, spring Spring) {
	shown := ui.Local(el, "popin", func() bool { return false })
	pose := ui.Pose{}
	if !*shown {
		*shown = true
		pose = ui.Pose{Scale: c.Env.scaled(from), Fade: 1}
		c.AnimationFrame()
	}
	el.TransformOrigin(originX, originY)
	el.AnimatePose(pose, spring.Anim())
}
