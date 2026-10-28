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

// Package appicon draws the application icon of vfox+: a fox on an indigo plate
// with a Material 3 Expressive cookie shape as a halo.
//
// The master of the icon is the SVG of the Linux desktop entry,
// shared/packaging/linux/dev.vfox.VfoxPlus.svg. The Android launcher icon repeats its
// paths as vector drawables, and this package repeats them as Go data so that the Windows
// logos, which a build draws as PNG files because the repository holds no binary files,
// show the same fox. The tests compare all three, so that a change of the master that
// forgets one of the others fails.
package appicon

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"
	"strings"

	"golang.org/x/image/vector"
)

// Render draws the icon on a transparent image of width by height pixels: the square plate
// with its rounded corners, as big as the smaller side and centred.
func Render(width, height int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	side := min(width, height)
	scale := float64(side) / canvas
	base := affine{scale, scale, float64(width-side) / 2, float64(height-side) / 2}
	fill(img, plate, base)
	centred := canvas / 2 * (1 - haloScale)
	fill(img, halo, base.compose(affine{haloScale, haloScale, centred, centred}))
	fox := base.compose(affine{foxScale, foxScale, foxX, foxY})
	for _, s := range foxShapes {
		fill(img, s, fox)
	}
	return img
}

// shape is a closed outline with its fill.
type shape struct {
	id   string // the id of the path in the SVG
	fill string // "#RRGGBB", or "url(#id)" of one of the gradients
	d    string // SVG path data, with the absolute commands M, L, C and Z only
}

// gradient is a linear gradient in the units of the shape that it fills.
type gradient struct {
	id             string
	x1, y1, x2, y2 float64
	stops          []stop
}

type stop struct {
	offset  float64
	color   string // "#RRGGBB"
	opacity float64
}

// affine is a scale and a translation, which is all that the icon uses.
type affine struct{ sx, sy, tx, ty float64 }

func (a affine) point(x, y float64) (float64, float64) { return a.sx*x + a.tx, a.sy*y + a.ty }

// compose is the transform that applies inner first, then a.
func (a affine) compose(inner affine) affine {
	return affine{a.sx * inner.sx, a.sy * inner.sy, a.sx*inner.tx + a.tx, a.sy*inner.ty + a.ty}
}

// fill draws the shape on img, its units transformed by tr.
func fill(img *image.NRGBA, s shape, tr affine) {
	cmds, err := parsePath(s.d)
	if err != nil {
		panic(fmt.Sprintf("appicon: path %s: %v", s.id, err)) // The data is part of the source.
	}
	b := img.Bounds()
	z := vector.NewRasterizer(b.Dx(), b.Dy())
	for _, c := range cmds {
		var p [3][2]float32
		for i := 0; i < c.n; i++ {
			x, y := tr.point(c.pts[i][0], c.pts[i][1])
			p[i] = [2]float32{float32(x), float32(y)}
		}
		switch c.op {
		case 'M':
			z.MoveTo(p[0][0], p[0][1])
		case 'L':
			z.LineTo(p[0][0], p[0][1])
		case 'C':
			z.CubeTo(p[0][0], p[0][1], p[1][0], p[1][1], p[2][0], p[2][1])
		case 'Z':
			z.ClosePath()
		}
	}
	z.Draw(img, b, paintOf(s.fill, tr), image.Point{})
}

// paintOf is the colour or the gradient of a fill, placed on the image by tr.
func paintOf(fill string, tr affine) image.Image {
	if id, ok := strings.CutPrefix(fill, "url(#"); ok {
		id = strings.TrimSuffix(id, ")")
		for _, g := range gradients {
			if g.id == id {
				x1, y1 := tr.point(g.x1, g.y1)
				x2, y2 := tr.point(g.x2, g.y2)
				return &linearGradient{x1: x1, y1: y1, x2: x2, y2: y2, stops: g.stops}
			}
		}
		panic("appicon: no gradient " + id)
	}
	return image.NewUniform(parseColor(fill, 1))
}

// parseColor reads "#RRGGBB" with the given opacity.
func parseColor(s string, opacity float64) color.NRGBA {
	v, err := strconv.ParseUint(strings.TrimPrefix(s, "#"), 16, 32)
	if err != nil || len(s) != 7 {
		panic("appicon: colour " + s)
	}
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), uint8(math.Round(opacity * 255))}
}

// linearGradient is an image.Image of a gradient between two points in pixels, which is
// padded with its first and last colour, as in SVG.
type linearGradient struct {
	x1, y1, x2, y2 float64
	stops          []stop
}

func (g *linearGradient) ColorModel() color.Model { return color.NRGBAModel }

func (g *linearGradient) Bounds() image.Rectangle {
	return image.Rect(math.MinInt32, math.MinInt32, math.MaxInt32, math.MaxInt32)
}

func (g *linearGradient) At(x, y int) color.Color {
	dx, dy := g.x2-g.x1, g.y2-g.y1
	t := ((float64(x)+0.5-g.x1)*dx + (float64(y)+0.5-g.y1)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	prev := g.stops[0]
	for _, next := range g.stops[1:] {
		if t <= next.offset {
			f := (t - prev.offset) / (next.offset - prev.offset)
			a, b := parseColor(prev.color, prev.opacity), parseColor(next.color, next.opacity)
			mix := func(p, q uint8) uint8 { return uint8(math.Round(float64(p) + (float64(q)-float64(p))*f)) }
			return color.NRGBA{mix(a.R, b.R), mix(a.G, b.G), mix(a.B, b.B), mix(a.A, b.A)}
		}
		prev = next
	}
	return parseColor(prev.color, prev.opacity)
}

// command is one command of path data: M and L have one point, C three, Z none.
type command struct {
	op  byte
	n   int
	pts [3][2]float64
}

// parsePath reads SVG path data made of the absolute commands M, L, C and Z, which is all
// that the icon uses (the same data is valid in an Android vector drawable).
func parsePath(d string) ([]command, error) {
	var cmds []command
	var nums []float64
	op := byte(0)
	arity := map[byte]int{'M': 2, 'L': 2, 'C': 6, 'Z': 0}
	flush := func() error {
		if op == 0 {
			if len(nums) > 0 {
				return fmt.Errorf("numbers before a command")
			}
			return nil
		}
		n := arity[op]
		if op == 'Z' {
			if len(nums) > 0 {
				return fmt.Errorf("Z with numbers")
			}
			cmds = append(cmds, command{op: 'Z'})
			return nil
		}
		if len(nums) == 0 || len(nums)%n != 0 {
			return fmt.Errorf("%c has %d numbers", op, len(nums))
		}
		for i := 0; i < len(nums); i += n {
			c := command{op: op, n: n / 2}
			for k := 0; k < n/2; k++ {
				c.pts[k] = [2]float64{nums[i+2*k], nums[i+2*k+1]}
			}
			cmds = append(cmds, c)
			if op == 'M' { // Further pairs of a move are lines.
				op = 'L'
			}
		}
		return nil
	}
	token := ""
	endToken := func() error {
		if token == "" {
			return nil
		}
		v, err := strconv.ParseFloat(token, 64)
		if err != nil {
			return err
		}
		nums = append(nums, v)
		token = ""
		return nil
	}
	for i := 0; i < len(d); i++ {
		ch := d[i]
		switch {
		case ch >= '0' && ch <= '9', ch == '.', ch == '-' && token == "":
			token += string(ch)
		case ch == ' ', ch == ',', ch == '\n', ch == '\t':
			if err := endToken(); err != nil {
				return nil, err
			}
		case ch == 'M' || ch == 'L' || ch == 'C' || ch == 'Z':
			if err := endToken(); err != nil {
				return nil, err
			}
			if err := flush(); err != nil {
				return nil, err
			}
			op, nums = ch, nil
		default:
			return nil, fmt.Errorf("unsupported %q in path data", ch)
		}
	}
	if err := endToken(); err != nil {
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return cmds, nil
}
