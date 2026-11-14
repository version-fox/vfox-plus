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
	"math"
	"testing"
	"time"
)

// A group that has less width than it wants gives up its icons, and then part of the padding of its segments.
func TestAButtonGroupGivesUpItsIconsAndThenItsPaddingToFitTheWidthItHas(t *testing.T) {
	segments := []Segment{
		{Value: "system", Label: "System", Icon: "systemMode"},
		{Value: "light", Label: "Light", Icon: "lightMode"},
		{Value: "dark", Label: "Dark", Icon: "darkMode"},
	}
	drawn := func(room float32) (width float32, icons int) {
		h := newTested(t, 600, 120, func(c Ctx) {
			Row(c).Padding(16).Children(func() {
				ButtonGroup(c, ButtonGroupSpec{ID: "group", Segments: segments, Selected: "system", OnSelect: func(string) {}, MaxWidth: room})
			})
		})
		h.tester.Frame()
		h.tester.Advance(time.Second)
		left, right := float32(1e9), float32(0)
		for _, n := range h.env.Reg.Nodes() {
			switch n.Role {
			case RoleButton:
				left, right = min(left, n.Rect.X), max(right, n.Rect.X+n.Rect.W)
			case RoleIcon:
				icons++
			}
		}
		return right - left, icons
	}
	// The widths are sums of measured text, which the engines give to a hundredth of a dp and not to the last bit of a float:
	// they are the same when they are as near as that, and a room that is exactly the width that a group wants is told from the
	// next one by a hundredth.
	near := func(a, b float32) bool { return math.Abs(float64(a-b)) < 0.01 }
	best, icons := drawn(0)
	if icons != 3 {
		t.Fatalf("a group with all the room has the icons of its three segments: %d", icons)
	}
	if same, _ := drawn(best + 0.01); !near(same, best) {
		t.Errorf("a group that has just the width that it wants is as it is: %v and %v", same, best)
	}
	// One dp short: the padding goes first, 4 dp from each side of a segment, and the icons stay.
	padded12, icons := drawn(best - 1)
	if icons != 3 || !near(padded12, best-3*8) {
		t.Errorf("with 12 dp of padding the group is %v dp wide and has %d icons, want %v and 3", padded12, icons, best-3*8)
	}
	// Then the icons (a segment is 28 dp narrower without its icon), and the padding is what it was.
	noIcons, icons := drawn(padded12 - 1)
	if icons != 0 || !near(noIcons, best-3*28) {
		t.Errorf("without icons the group is %v dp wide and has %d icons, want %v and none", noIcons, icons, best-3*28)
	}
	// Then the padding again: 12 dp, and 8 dp (a segment is 48 dp wide at the least, so a short label may give up
	// less than 4 dp the second time).
	noIcons12, icons := drawn(noIcons - 1)
	if icons != 0 || !near(noIcons12, noIcons-3*8) {
		t.Errorf("without icons and with 12 dp of padding the group is %v dp wide and has %d icons, want %v and none", noIcons12, icons, noIcons-3*8)
	}
	noIcons8, icons := drawn(noIcons12 - 1)
	if icons != 0 || noIcons8 >= noIcons12 {
		t.Errorf("with 8 dp of padding the group is %v dp wide and has %d icons, want less than %v and none", noIcons8, icons, noIcons12)
	}
	// Less than it can be: it is the least that it can be.
	if smaller, _ := drawn(10); !near(smaller, noIcons8) {
		t.Errorf("a group that cannot fit is as narrow as it gets: %v, want %v", smaller, noIcons8)
	}
}
