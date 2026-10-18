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
	"sync"
	"time"

	"gioui.org/op"
)

// The Material 3 Expressive motion system: springs instead of durations and
// curves (docs/design-system.md#motion). Spatial springs move, resize and
// reshape and may overshoot; effects springs change colour and opacity and
// never do. All springs have unit mass.

// Spring is a damped spring of unit mass.
type Spring struct {
	// Damping is the damping ratio: 1 is critically damped, less bounces.
	Damping float64
	// Stiffness is the stiffness; higher settles faster.
	Stiffness float64
}

// MotionScheme holds the six springs of a motion scheme.
type MotionScheme struct {
	FastSpatial, DefaultSpatial, SlowSpatial Spring
	FastEffects, DefaultEffects, SlowEffects Spring
}

// Expressive is the expressive scheme, which vfox-plus uses: bouncy spatial
// springs.
var Expressive = MotionScheme{
	FastSpatial:    Spring{0.6, 800},
	DefaultSpatial: Spring{0.8, 380},
	SlowSpatial:    Spring{0.8, 200},
	FastEffects:    Spring{1, 3800},
	DefaultEffects: Spring{1, 1600},
	SlowEffects:    Spring{1, 800},
}

// Tolerance of a spring at rest, as in Flutter's physics.
const (
	restDistance = 0.001
	restVelocity = 0.01
)

// step returns the position and velocity of the spring after dt seconds,
// when it started dx away from its target, moving at v. It is the exact
// solution of the equation of the damped oscillator.
func (s Spring) step(dx, v, dt float64) (float64, float64) {
	omega := math.Sqrt(s.Stiffness)
	zeta := s.Damping
	switch {
	case zeta < 1:
		omegaD := omega * math.Sqrt(1-zeta*zeta)
		decay := math.Exp(-zeta * omega * dt)
		cos, sin := math.Cos(omegaD*dt), math.Sin(omegaD*dt)
		b := (v + zeta*omega*dx) / omegaD
		x := decay * (dx*cos + b*sin)
		vel := decay * (v*cos - (dx*omegaD+zeta*omega*b)*sin)
		return x, vel
	case zeta == 1:
		decay := math.Exp(-omega * dt)
		b := v + omega*dx
		return decay * (dx + b*dt), decay * (v - omega*b*dt)
	default:
		root := math.Sqrt(zeta*zeta - 1)
		r1 := -omega * (zeta - root)
		r2 := -omega * (zeta + root)
		c2 := (v - r1*dx) / (r2 - r1)
		c1 := dx - c2
		e1, e2 := math.Exp(r1*dt), math.Exp(r2*dt)
		return c1*e1 + c2*e2, c1*r1*e1 + c2*r2*e2
	}
}

var settleTimes sync.Map

// SettleTime is how long the spring needs to travel from 0 to 1 and come to
// rest. Page transitions, which bring their own curves, last that long.
func (s Spring) SettleTime() time.Duration {
	if cached, ok := settleTimes.Load(s); ok {
		return cached.(time.Duration)
	}
	settled := s.computeSettleTime()
	settleTimes.Store(s, settled)
	return settled
}

func (s Spring) computeSettleTime() time.Duration {
	const stepSize = 1.0 / 240
	for t := 0.0; t < 5; t += stepSize {
		x, v := s.step(-1, 0, t)
		if math.Abs(x) < restDistance && math.Abs(v) < restVelocity {
			return time.Duration(t * float64(time.Second))
		}
	}
	return 5 * time.Second
}

// Position is the position of the spring t seconds after it started at 0,
// at rest, with 1 as its target.
func (s Spring) Position(t float64) float64 {
	x, _ := s.step(-1, 0, t)
	return 1 + x
}

// Anim is a value that follows a target on a spring. It keeps its velocity
// when the target changes, so that interrupted animations stay smooth. The zero
// value starts at the first target it is given, without animating.
type Anim struct {
	value, velocity float64
	target          float64
	last            time.Time
	started         bool
}

// To moves the value towards target with the spring and returns it. It asks for
// the next frame while the value is moving.
func (a *Anim) To(c Ctx, target float64, spring Spring) float64 {
	if !a.started || c.Env.Reduced() {
		a.value, a.velocity, a.target, a.started = target, 0, target, true
		a.last = c.Now
		return a.value
	}
	a.target = target
	dt := c.Now.Sub(a.last).Seconds()
	if dt <= 0 {
		// Laid out again within the same frame: the time has not moved.
		if a.value != target || a.velocity != 0 {
			c.Execute(op.InvalidateCmd{})
		}
		return a.value
	}
	a.last = c.Now
	if dt > 0.05 {
		// A long pause between frames (a hidden window) must not skip the
		// animation.
		dt = 0.05
	}
	dx, v := spring.step(a.value-target, a.velocity, dt)
	a.value, a.velocity = target+dx, v
	if math.Abs(dx) < restDistance && math.Abs(v) < restVelocity {
		a.value, a.velocity = target, 0
		return a.value
	}
	c.Execute(op.InvalidateCmd{})
	return a.value
}

// Snap puts the value on target at once.
func (a *Anim) Snap(c Ctx, target float64) {
	a.value, a.velocity, a.target, a.started = target, 0, target, true
	a.last = c.Now
}

// Value is the current value, without advancing it.
func (a *Anim) Value() float64 { return a.value }

// Moving reports whether the value has not reached its target.
func (a *Anim) Moving() bool { return a.started && (a.value != a.target || a.velocity != 0) }

// F is the value of To as a float32, which drawing uses.
func (a *Anim) F(c Ctx, target float64, spring Spring) float32 {
	return float32(a.To(c, target, spring))
}

// Flag animates a switch between 0 and 1: To with the target 1 for true, 0
// for false.
func (a *Anim) Flag(c Ctx, on bool, spring Spring) float32 {
	target := 0.0
	if on {
		target = 1
	}
	return float32(a.To(c, target, spring))
}

// Lerp interpolates between a and b.
func Lerp(a, b, t float32) float32 { return a + (b-a)*t }
