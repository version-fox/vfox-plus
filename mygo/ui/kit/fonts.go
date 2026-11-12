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
	"slices"

	"github.com/egoist/mygo/ui"
)

// The font families of the interface. mygo draws text with the text engine of
// the desktop, which finds these through fontconfig: the sans-serif font of the
// desktop is the font of Material 3 on Linux (Ubuntu Sans on Ubuntu), and what
// it lacks, such as Chinese, comes from the next family and then from the
// fonts the system chooses. They are variables so that a test can put the fonts
// that the system chooses in their place, and see that the screens do not
// depend on the width of Ubuntu Sans; nothing else writes them.
var (
	// UIFamilies is the font of the interface.
	UIFamilies = "Ubuntu Sans, Ubuntu, Noto Sans, sans-serif"
	// MonoFamilies is the monospaced font.
	MonoFamilies = "Ubuntu Sans Mono, Ubuntu Mono, Noto Sans Mono, monospace"
)

// Tracking is the width, in em, that is added after every character of the text of the interface, and is 0 where nothing asks for it.
// A test sets it to see that the screens hold in a text that is wider than the fonts it is drawn in, as a translation is, or a font
// that no system sets its interface in; nothing else writes it. The labels draw with it, and what measures a text, to make room for
// it, measures with it (Ctx.MeasureText), so that the widths that the layouts decide by are those that are drawn.
var Tracking float32

// MeasureText is the width and the height of the spans set in the room of width dp (no limit for 0), as the labels set them: with the
// tracking that a test asks for, in the spans that have not their own letter spacing.
func (c Ctx) MeasureText(width float32, spans ...ui.Span) (w, h float32) {
	if Tracking != 0 {
		spans = slices.Clone(spans)
		for i := range spans {
			if spans[i].LetterSpacing == 0 {
				spans[i].LetterSpacing = Tracking * spans[i].Size
			}
		}
	}
	return c.Context.MeasureText(width, spans...)
}
