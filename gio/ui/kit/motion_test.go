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

package kit_test

import (
	"math"
	"testing"
	"time"

	"gioui.org/layout"

	"github.com/version-fox/vfox-plus/gio/ui/kit"
)

func TestSpringsReachTheirTargetAndOnlyBounceWhenUnderdamped(t *testing.T) {
	bouncy := kit.Spring{Damping: 0.5, Stiffness: 400}
	critical := kit.Spring{Damping: 1, Stiffness: 400}
	heavy := kit.Spring{Damping: 2, Stiffness: 400}
	for name, spring := range map[string]kit.Spring{"bouncy": bouncy, "critical": critical, "heavy": heavy} {
		if got := spring.Position(0); math.Abs(got) > 1e-9 {
			t.Errorf("%s starts at 0, got %v", name, got)
		}
		if got := spring.Position(10); math.Abs(got-1) > 1e-6 {
			t.Errorf("%s settles at 1, got %v", name, got)
		}
	}
	overshoot := func(s kit.Spring) float64 {
		peak := 0.0
		for i := 0; i < 400; i++ {
			peak = math.Max(peak, s.Position(float64(i)/200))
		}
		return peak
	}
	if overshoot(bouncy) <= 1.1 {
		t.Error("an underdamped spring overshoots")
	}
	if overshoot(critical) > 1+1e-9 || overshoot(heavy) > 1+1e-9 {
		t.Error("the others do not")
	}
}

func TestSettleTimeGrowsWithSoftnessAndIsRemembered(t *testing.T) {
	stiff := kit.Spring{Damping: 1, Stiffness: 3800}.SettleTime()
	soft := kit.Spring{Damping: 1, Stiffness: 200}.SettleTime()
	if stiff <= 0 || soft <= stiff {
		t.Errorf("got %v and %v", stiff, soft)
	}
	if again := (kit.Spring{Damping: 1, Stiffness: 200}).SettleTime(); again != soft {
		t.Errorf("got %v and %v", again, soft)
	}
	if never := (kit.Spring{Damping: 0.001, Stiffness: 1}).SettleTime(); never != 5*time.Second {
		t.Errorf("a spring that does not settle is cut off, got %v", never)
	}
	if kit.Expressive.FastSpatial.Damping >= 1 || kit.Expressive.DefaultEffects.Damping != 1 {
		t.Error("spatial springs bounce, effects springs do not")
	}
}

func TestAnimStartsAtItsFirstTargetThenFollowsItAndStepsOncePerFrame(t *testing.T) {
	h := newHarness(t)
	var anim kit.Anim
	var values []float64
	var second float64
	probe := func(target float64) kit.Widget {
		return func(c kit.Ctx) layout.Dimensions {
			values = append(values, anim.To(c, target, kit.Expressive.DefaultSpatial))
			second = anim.To(c, target, kit.Expressive.DefaultSpatial)
			return layout.Dimensions{}
		}
	}
	h.frame(probe(5))
	if values[0] != 5 || anim.Moving() {
		t.Errorf("the first target is not animated, got %v", values)
	}
	h.frame(probe(10))
	first := values[1]
	if first <= 5 || first >= 10 || !anim.Moving() {
		t.Errorf("got %v", values)
	}
	if second != first {
		t.Errorf("laid out twice in a frame, it moves once: %v and %v", first, second)
	}
	h.invalidations = 0
	for range 200 {
		h.frame(probe(10))
	}
	if got := anim.Value(); got != 10 || anim.Moving() {
		t.Errorf("got %v", got)
	}
	anim.Snap(h.ctx(layout.Constraints{}), 3)
	if anim.Value() != 3 || anim.F(h.ctx(layout.Constraints{}), 3, kit.Expressive.FastEffects) != 3 {
		t.Error("snap")
	}
	if kit.Lerp(2, 4, 0.25) != 2.5 {
		t.Error("lerp")
	}
}

func TestAnimJumpsWhenMotionIsReduced(t *testing.T) {
	h := newHarness(t)
	h.env.Theme.ReducedMotion = true
	var anim kit.Anim
	run := func(on bool) float32 {
		var value float32
		h.frame(func(c kit.Ctx) layout.Dimensions {
			value = anim.Flag(c, on, kit.Expressive.SlowSpatial)
			return layout.Dimensions{}
		})
		return value
	}
	if run(false) != 0 || run(true) != 1 || run(false) != 0 {
		t.Error("with reduced motion, every change is immediate")
	}
}

func TestLongPausesDoNotSkipAnAnimation(t *testing.T) {
	h := newHarness(t)
	var anim kit.Anim
	step := func(target float64) float64 {
		var value float64
		h.frame(func(c kit.Ctx) layout.Dimensions {
			value = anim.To(c, target, kit.Expressive.SlowSpatial)
			return layout.Dimensions{}
		})
		return value
	}
	step(0)
	// A frame after a long pause moves the value by one short step (50 ms of
	// the spring) only, rather than to the target.
	h.now = h.now.Add(10 * time.Second)
	if got := step(1); got <= 0 || got > 0.3 {
		t.Errorf("got %v", got)
	}
}
