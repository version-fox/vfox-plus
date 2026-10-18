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

	"gioui.org/gesture"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

type tipState struct{ hover gesture.Hover }

// Tip shows text in a tooltip when the pointer rests on w, which is not
// interactive itself, such as an icon that marks something.
func Tip(id, text string, w Widget) Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[tipState](c, id)
		hovered := st.hover.Update(c.Source)
		dims := w(c)
		rect := image.Rectangle{Max: dims.Size}
		area := clip.Rect(rect).Push(c.Ops)
		st.hover.Add(c.Ops)
		area.Pop()
		key := c.key(id)
		c.Env.Reg.Add(Node{Key: key, Role: RoleText, Name: text, Tooltip: text, Rect: rect, Hovered: hovered})
		c.Env.noteHover(key, text, hovered, rect, 0)
		return dims
	}
}
