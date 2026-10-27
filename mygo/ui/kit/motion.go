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

	"github.com/egoist/mygo/ui"
)

// The Material 3 Expressive motion system: springs instead of durations and
// curves (docs/en/dev/design-system.md#motion). Spatial springs move, resize and
// reshape and may overshoot; effects springs change colour and opacity and
// never do. All springs have unit mass.

// Spring is a damped spring of unit mass, described as the design system does:
// by its damping ratio and its stiffness.
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

// Anim is the spring as an animation of mygo's package ui. mygo's springs take
// the damping coefficient, which is twice the product of the damping ratio and
// the square root of the stiffness for a unit mass.
func (s Spring) Anim() ui.Animation {
	return ui.Animation{Spring: ui.SpringOptions{
		Mass: 1, Stiffness: s.Stiffness, Damping: 2 * s.Damping * math.Sqrt(s.Stiffness),
	}}
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
// rest. Transitions that hold on to what is leaving last that long.
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

// Flag is 1 for true and 0 for false, the target of the springs that switch
// something on and off.
func Flag(on bool) float32 {
	if on {
		return 1
	}
	return 0
}
