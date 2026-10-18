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
	"image/color"
	"math"
	"sync"
	"time"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"

	"github.com/version-fox/vfox-plus/gio/ui/shapes"
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

// LoadingIndicator is the Material 3 Expressive loading indicator: a shape that
// morphs into the next while it turns.
type LoadingIndicator struct {
	// ID identifies the indicator between frames, within its scope.
	ID string
	// Size is the side of the indicator in dp; 48 by default.
	Size float32
	// Contained draws it on a circle of the primary container.
	Contained bool
	// Color overrides the colour of the shape.
	Color *color.NRGBA
	// Label names the indicator for the accessibility tree.
	Label string
}

type loadingState struct {
	start time.Time
}

// W is the indicator as a widget.
func (l LoadingIndicator) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[loadingState](c, l.ID)
		colors := c.Env.Theme.Colors
		sizeDp := l.Size
		if sizeDp == 0 {
			sizeDp = 48
		}
		px := c.DP(sizeDp)
		size := c.Constraints.Constrain(image.Pt(px, px))
		col := colors.Primary
		if l.Contained {
			col = colors.OnPrimaryContainer
			FillCircle(c.Ops, image.Pt(size.X/2, size.Y/2), minInt(size.X, size.Y)/2, colors.PrimaryContainer)
		}
		if l.Color != nil {
			col = *l.Color
		}
		var elapsed time.Duration
		if !c.Env.Reduced() {
			if st.start.IsZero() {
				st.start = c.Now
			}
			elapsed = c.Now.Sub(st.start)
			c.Invalidate()
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
		side := float32(minInt(size.X, size.Y)) * loadingActiveScale
		center := f32.Pt(float32(size.X)/2, float32(size.Y)/2)
		cubics := morph.AsCubics(math.Min(1, math.Max(0, progress)))
		defer clip.Outline{Path: ShapePath(c.Ops, cubics, center, side, rotation)}.Op().Push(c.Ops).Pop()
		paint.Fill(c.Ops, col)
		c.Env.Reg.Add(Node{Role: RoleText, Name: l.Label, Rect: image.Rectangle{Max: size}})
		return layout.Dimensions{Size: size}
	}
}

// WavyProgress is the wavy linear progress indicator of Material 3 Expressive
// (docs/design-system.md#wavy-progress-indicator), determinate or
// indeterminate.
type WavyProgress struct {
	// ID identifies the indicator between frames, within its scope.
	ID string
	// Value is the progress from 0 to 1; ignored when Indeterminate.
	Value float64
	// Indeterminate shows the cycling indicator that has no value.
	Indeterminate bool
	// Label names the indicator for the accessibility tree.
	Label string
}

type wavyState struct {
	start     time.Time
	amplitude Anim
	value     Anim
}

const (
	wavyStroke    = 4.0
	wavyAmplitude = 3.0
	wavyGap       = 4.0
)

// cubicBezier is the easing curve of CSS and Flutter's Cubic: the y of the
// point of the Bézier curve through (0,0), (a,b), (c,d), (1,1) whose x is t.
func cubicBezier(a, b, cc, d, t float64) float64 {
	bezier := func(p1, p2, s float64) float64 {
		u := 1 - s
		return 3*u*u*s*p1 + 3*u*s*s*p2 + s*s*s
	}
	lo, hi := 0.0, 1.0
	s := t
	for i := 0; i < 24; i++ {
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

// W is the indicator as a widget.
func (w WavyProgress) W() Widget {
	return func(c Ctx) layout.Dimensions {
		st := UseID[wavyState](c, w.ID)
		theme := c.Env.Theme
		colors := theme.Colors
		stroke := float32(c.DP(wavyStroke))
		height := c.DP(wavyStroke + wavyAmplitude*2)
		width := c.Constraints.Max.X
		if unbounded(width) {
			width = c.DP(200)
		}
		size := c.Constraints.Constrain(image.Pt(width, height))
		width = size.X
		value := math.Min(1, math.Max(0, w.Value))
		targetAmplitude := 1.0
		if !w.Indeterminate && (value <= 0.1 || value >= 0.95) {
			targetAmplitude = 0
		}
		amplitude := float32(c.DP(wavyAmplitude)) * float32(math.Min(1, math.Max(0, st.amplitude.To(c, targetAmplitude, theme.Motion.DefaultEffects))))
		shown := math.Min(1, math.Max(0, st.value.To(c, value, theme.Motion.DefaultSpatial)))
		wavelength := float32(c.DP(40))
		if w.Indeterminate {
			wavelength = float32(c.DP(20))
		}
		var clock float64
		if !c.Env.Reduced() {
			if st.start.IsZero() {
				st.start = c.Now
			}
			clock = float64(c.Now.Sub(st.start)%(2*time.Second)) / float64(2*time.Second)
			c.Invalidate()
		}
		// The wave travels one wavelength per second.
		phase := float32(clock * 2 * 2 * math.Pi)
		cy := float32(size.Y) / 2
		cap := stroke / 2
		gap := float32(c.DP(wavyGap))

		drawTrack := func(from, to float32) {
			if to-from <= stroke {
				return
			}
			var p clip.Path
			p.Begin(c.Ops)
			p.MoveTo(f32.Pt(from+cap, cy))
			p.LineTo(f32.Pt(to-cap, cy))
			fillStroke(c, p.End(), stroke, colors.SecondaryContainer)
		}
		drawActive := func(from, to float32) {
			if to-from <= 0 {
				return
			}
			var p clip.Path
			p.Begin(c.Ops)
			start := from + cap
			end := maxf(start, to-cap)
			for x := start; x <= end; x++ {
				y := cy + amplitude*float32(math.Sin(float64(2*math.Pi*x/wavelength-phase)))
				if x == start {
					p.MoveTo(f32.Pt(x, y))
				} else {
					p.LineTo(f32.Pt(x, y))
				}
			}
			if end == start {
				p.LineTo(f32.Pt(end+0.01, cy))
			}
			fillStroke(c, p.End(), stroke, colors.Primary)
		}

		if !w.Indeterminate {
			activeEnd := float32(width) * float32(shown)
			drawActive(0, activeEnd)
			if shown > 0 {
				drawTrack(activeEnd+gap, float32(width))
			} else {
				drawTrack(0, float32(width))
			}
			if shown < 1 {
				FillCircle(c.Ops, image.Pt(width-int(cap), int(cy)), int(cap/2+0.5), colors.Primary)
			}
		} else {
			t := clock
			curve := func(a, b, cc, d float64) func(float64) float64 {
				return func(x float64) float64 { return cubicBezier(a, b, cc, d, x) }
			}
			head1 := float32(width) * float32(interval(t, 0, 0.375, curve(0.2, 0, 0.8, 1)))
			tail1 := float32(width) * float32(interval(t, 0.1665, 0.5415, curve(0.4, 0, 1, 1)))
			head2 := float32(width) * float32(interval(t, 0.5, 0.7835, curve(0, 0, 0.65, 1)))
			tail2 := float32(width) * float32(interval(t, 0.6335, 0.9, curve(0.1, 0, 0.45, 1)))
			cursor := float32(0)
			if head2 > tail2 {
				drawTrack(cursor, tail2-gap)
				drawActive(tail2, head2)
				cursor = head2 + gap
			}
			if head1 > tail1 {
				drawTrack(cursor, tail1-gap)
				drawActive(tail1, head1)
				cursor = head1 + gap
			}
			drawTrack(cursor, float32(width))
		}
		c.Env.Reg.Add(Node{Role: RoleText, Name: w.Label, Rect: image.Rectangle{Max: size}})
		return layout.Dimensions{Size: size}
	}
}

// fillStroke draws a path as a line with round ends.
func fillStroke(c Ctx, path clip.PathSpec, width float32, col color.NRGBA) {
	defer clip.Stroke{Path: path, Width: width}.Op().Push(c.Ops).Pop()
	paint.Fill(c.Ops, col)
}
