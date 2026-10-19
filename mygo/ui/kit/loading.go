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

	"github.com/version-fox/vfox-plus/mygo/ui/shapes"
)

// The loading indicator morphs through a sequence of shapes
// (docs/design-system.md#loading-indicator).
const (
	loadingMorphInterval  = 650 * time.Millisecond
	loadingRotationPeriod = 4666 * time.Millisecond
	loadingActiveScale    = 38.0 / 48.0
)

var loadingMorphSpring = Spring{Damping: 0.6, Stiffness: 200}

var (
	loadingMorphsOnce sync.Once
	loadingMorphs     []*shapes.Morph
)

// morphsOfLoading prepares the morphs between the shapes, once.
func morphsOfLoading() []*shapes.Morph {
	loadingMorphsOnce.Do(func() {
		sequence := []*shapes.Polygon{
			shapes.ShapeSoftBurst, shapes.ShapeCookie9, shapes.ShapePentagon, shapes.ShapePill,
			shapes.ShapeSunny, shapes.ShapeCookie4, shapes.ShapeOval,
		}
		fitted := make([]*shapes.Polygon, len(sequence))
		for i, shape := range sequence {
			fitted[i] = shape.FitToCircle()
		}
		for i := range fitted {
			loadingMorphs = append(loadingMorphs, shapes.NewMorph(fitted[i], fitted[(i+1)%len(fitted)]))
		}
	})
	return loadingMorphs
}

// LoadingSpec describes a loading indicator.
type LoadingSpec struct {
	// ID identifies the indicator among its siblings between builds.
	ID string
	// Size is the side of the indicator in dp; 48 by default.
	Size float32
	// Contained draws it on a circle of the primary container.
	Contained bool
	// Color overrides the colour of the shape when its alpha is not zero.
	Color ui.Color
	// Label names the indicator for assistive technology.
	Label string
}

// LoadingIndicator is the Material 3 Expressive loading indicator: a shape that
// morphs into the next while it turns. It draws itself while the window shows
// it, without building the view again.
func LoadingIndicator(c Ctx, l LoadingSpec) ui.Element {
	cs := c.Colors()
	size := l.Size
	if size == 0 {
		size = 48
	}
	col := cs.Primary
	if l.Contained {
		col = cs.OnPrimaryContainer
	}
	if l.Color.A != 0 {
		col = l.Color
	}
	reduced := c.Preferences().ReduceMotion || c.Env.Theme.ReducedMotion
	if hasKey(l.ID) {
		c.Key(l.ID)
	}
	box := ui.Box(c.Context).Size(size, size).Shrink(0)
	if l.Label != "" {
		box.Label(l.Label)
	}
	start := ui.Local(box, "start", func() time.Time { return time.Time{} })
	node := c.Env.Reg.Add(Node{Role: RoleText, Name: l.Label})
	background := cs.PrimaryContainer
	return box.Draw(func(p *ui.Painter, r ui.Rect) {
		node.Rect = r
		if l.Contained {
			p.Fill(r, background, min(r.W, r.H)/2)
		}
		var elapsed time.Duration
		if !reduced {
			if start.IsZero() {
				*start = p.Now()
			}
			elapsed = p.Now().Sub(*start)
			p.AnimationFrame()
		}
		millis := float64(elapsed) / float64(time.Millisecond)
		interval := float64(loadingMorphInterval / time.Millisecond)
		step := int(millis / interval)
		progress := loadingMorphSpring.Position((millis - float64(step)*interval) / 1000)
		morphs := morphsOfLoading()
		morph := morphs[step%len(morphs)]
		// A constant rotation, and a quarter turn per morph that follows the
		// (overshooting) spring.
		rotation := 2*math.Pi*(millis/float64(loadingRotationPeriod/time.Millisecond)) + (float64(step)+progress)*math.Pi/2
		side := min(r.W, r.H) * loadingActiveScale
		inner := ui.Rect{X: r.X + (r.W-side)/2, Y: r.Y + (r.H-side)/2, W: side, H: side}
		fillShape(p, morph.AsCubics(math.Min(1, math.Max(0, progress))), inner, rotation, col)
	})
}

// WavySpec describes a wavy progress indicator.
type WavySpec struct {
	// ID identifies the indicator among its siblings between builds.
	ID string
	// Value is the progress from 0 to 1; ignored when Indeterminate.
	Value float64
	// Indeterminate shows the cycling indicator that has no value.
	Indeterminate bool
	// Label names the indicator for assistive technology.
	Label string
}

const (
	wavyStroke    = 4.0
	wavyAmplitude = 3.0
	wavyGap       = 4.0
)

// cubicBezier is the easing curve of CSS and Flutter's Cubic: the y of the point
// of the Bézier curve through (0,0), (a,b), (c,d), (1,1) whose x is t.
func cubicBezier(a, b, cc, d, t float64) float64 {
	bezier := func(p1, p2, s float64) float64 {
		u := 1 - s
		return 3*u*u*s*p1 + 3*u*s*s*p2 + s*s*s
	}
	lo, hi := 0.0, 1.0
	s := t
	for range 24 {
		x := bezier(a, cc, s)
		if math.Abs(x-t) < 0.001 {
			break
		}
		if x < t {
			lo = s
		} else {
			hi = s
		}
		s = (lo + hi) / 2
	}
	return bezier(b, d, s)
}

// interval maps t, from 0 to 1, to the progress of an animation that begins at
// begin and ends at end, eased by the curve.
func interval(t, begin, end float64, curve func(float64) float64) float64 {
	t = math.Min(1, math.Max(0, (t-begin)/(end-begin)))
	if t == 0 || t == 1 {
		return t
	}
	return curve(t)
}

// WavyProgress is the wavy linear progress indicator of Material 3 Expressive
// (docs/design-system.md#wavy-progress-indicator), determinate or
// indeterminate. It is as wide as its parent leaves it, and draws the wave
// while the window shows it, without building the view again.
func WavyProgress(c Ctx, w WavySpec) ui.Element {
	cs := c.Colors()
	theme := c.Env.Theme
	reduced := c.Preferences().ReduceMotion || theme.ReducedMotion
	if hasKey(w.ID) {
		c.Key(w.ID)
	}
	box := ui.Box(c.Context).Height(wavyStroke + wavyAmplitude*2).Shrink(0)
	if w.Label != "" {
		box.Label(w.Label)
	}
	value := math.Min(1, math.Max(0, w.Value))
	targetAmplitude := float32(1)
	if !w.Indeterminate && (value <= 0.1 || value >= 0.95) {
		targetAmplitude = 0
	}
	amplitude := wavyAmplitude * clamp01(box.AnimateTo("amplitude", targetAmplitude, theme.Motion.DefaultEffects.Anim()))
	shown := float64(clamp01(box.AnimateTo("value", float32(value), theme.Motion.DefaultSpatial.Anim())))
	wavelength := float32(40)
	if w.Indeterminate {
		wavelength = 20
	}
	start := ui.Local(box, "start", func() time.Time { return time.Time{} })
	node := c.Env.Reg.Add(Node{Role: RoleText, Name: w.Label})
	primary, track := cs.Primary, cs.SecondaryContainer
	return box.Draw(func(p *ui.Painter, r ui.Rect) {
		node.Rect = r
		var clock float64
		if !reduced {
			if start.IsZero() {
				*start = p.Now()
			}
			clock = float64(p.Now().Sub(*start)%(2*time.Second)) / float64(2*time.Second)
			p.AnimationFrame()
		}
		// The wave travels one wavelength per second.
		phase := float32(clock * 2 * 2 * math.Pi)
		width := r.W
		cy := r.Y + r.H/2
		const stroke, capRadius = float32(wavyStroke), float32(wavyStroke) / 2

		drawTrack := func(from, to float32) {
			if to-from <= stroke {
				return
			}
			p.Fill(ui.Rect{X: r.X + from, Y: cy - capRadius, W: to - from, H: stroke}, track, capRadius)
		}
		drawActive := func(from, to float32) {
			if to-from <= 0 {
				return
			}
			var path ui.Path
			begin := from + capRadius
			end := max(begin, to-capRadius)
			const step = 2
			for x := begin; ; x += step {
				x = min(x, end)
				y := cy + amplitude*float32(math.Sin(float64(2*math.Pi*x/wavelength-phase)))
				if x == begin {
					path.MoveTo(r.X+x, y)
				} else {
					path.LineTo(r.X+x, y)
				}
				if x >= end {
					break
				}
			}
			if end == begin {
				path.LineTo(r.X+end+0.01, cy)
			}
			p.StrokePath(&path, stroke, primary)
		}

		if !w.Indeterminate {
			activeEnd := width * float32(shown)
			drawActive(0, activeEnd)
			if shown > 0 {
				drawTrack(activeEnd+wavyGap, width)
			} else {
				drawTrack(0, width)
			}
			if shown < 1 {
				// The stop indicator at the end of the track.
				p.Fill(ui.Rect{X: r.X + width - capRadius - 1, Y: cy - 1, W: 2, H: 2}, primary, 1)
			}
			return
		}
		t := clock
		curve := func(a, b, cc, d float64) func(float64) float64 {
			return func(x float64) float64 { return cubicBezier(a, b, cc, d, x) }
		}
		head1 := width * float32(interval(t, 0, 0.375, curve(0.2, 0, 0.8, 1)))
		tail1 := width * float32(interval(t, 0.1665, 0.5415, curve(0.4, 0, 1, 1)))
		head2 := width * float32(interval(t, 0.5, 0.7835, curve(0, 0, 0.65, 1)))
		tail2 := width * float32(interval(t, 0.6335, 0.9, curve(0.1, 0, 0.45, 1)))
		cursor := float32(0)
		if head2 > tail2 {
			drawTrack(cursor, tail2-wavyGap)
			drawActive(tail2, head2)
			cursor = head2 + wavyGap
		}
		if head1 > tail1 {
			drawTrack(cursor, tail1-wavyGap)
			drawActive(tail1, head1)
			cursor = head1 + wavyGap
		}
		drawTrack(cursor, width)
	})
}
