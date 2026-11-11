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
	"testing"
	"time"
)

func TestScaleSizeKeepsToTheStepsOfTheTypeScale(t *testing.T) {
	for size, want := range map[float32]float32{
		4: 11, 11: 11, 12: 12, 13: 12, 16.8: 16, 23.5: 22, 30.2: 28, 57: 57, 90: 57,
	} {
		if got := ScaleSize(size); got != want {
			t.Errorf("ScaleSize(%v) = %v, want %v", size, got, want)
		}
	}
}

func TestTheInitialInAnAvatarKeepsToTheTypeScaleWhateverTheSizeOfTheAvatar(t *testing.T) {
	h := newTested(t, 420, 200, func(c Ctx) {
		Row(c).Padding(16).Gap(8).Children(func() {
			for _, size := range []float32{24, 32, 40, 56, 72} {
				Avatar(c, AvatarSpec{Name: "nodejs", Size: size})
			}
		})
	})
	h.tester.Frame()
	h.tester.Advance(time.Second)
	got := TakeCensus(h.env.Reg.Nodes(), rect(0, 0, 420, 200), nil)
	if len(got.OffScale) != 0 || got.Words != 5 {
		t.Errorf("five initials, all on the scale: %d words, off the scale %v", got.Words, got.OffScale)
	}
}
