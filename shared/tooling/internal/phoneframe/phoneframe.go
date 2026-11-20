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

// Package phoneframe implements `vfxtool frame-phone`: it frames the screenshot of a phone for the
// documentation. The pictures of the phone app in the user guide have the rounded corners of a phone
// display, transparent outside, and a thin grey border over the edge of the picture: the cream screens of the
// light theme have no edge against the white page of GitHub without it. With --top it first cuts the rows at
// the top that hold the status bar of the device: what the guide shows is the app, and not the clock, the
// battery and the signal of the emulator that it was taken on.
package phoneframe

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"math"
	"os"
	"strconv"
)

// What the pictures of the guide have: the radius of the corners and the width of the border, in pixels of a
// screenshot of 1080 by 2400, and the colour of the border (#7A7A84).
const (
	defaultRadius = 96
	defaultBorder = 5
	defaultColor  = "7A7A84"
)

const usage = "Usage: frame-phone [--top PIXELS] [--radius PIXELS] [--border PIXELS] [--color RRGGBB] INPUT.png OUTPUT.png"

// Run implements `frame-phone INPUT.png OUTPUT.png`.
func Run(args []string) error {
	return run(args, os.Stderr)
}

func run(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("frame-phone", flag.ContinueOnError)
	flags.SetOutput(stderr)
	top := flags.Int("top", 0, "rows to cut from the top of the picture before it is framed, in pixels: the status bar of the device")
	radius := flags.Int("radius", defaultRadius, "radius of the corners, in pixels")
	border := flags.Int("border", defaultBorder, "width of the border, in pixels")
	hex := flags.String("color", defaultColor, "colour of the border, as RRGGBB")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 2 {
		return errors.New(usage)
	}
	if *top < 0 || *radius < 0 || *border < 0 {
		return errors.New("frame-phone: the top, the radius and the border are not negative")
	}
	edge, err := parseColor(*hex)
	if err != nil {
		return err
	}
	input, err := os.Open(flags.Arg(0))
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()
	source, err := png.Decode(input)
	if err != nil {
		return fmt.Errorf("%s: %w", flags.Arg(0), err)
	}
	source, err = CutTop(source, *top)
	if err != nil {
		return fmt.Errorf("%s: %w", flags.Arg(0), err)
	}
	output, err := os.Create(flags.Arg(1))
	if err != nil {
		return err
	}
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(output, Frame(source, *radius, *border, edge)); err != nil {
		_ = output.Close()
		return err
	}
	return output.Close()
}

// parseColor reads RRGGBB, with or without a leading #.
func parseColor(text string) (color.NRGBA, error) {
	if len(text) == 7 && text[0] == '#' {
		text = text[1:]
	}
	value, err := strconv.ParseUint(text, 16, 32)
	if err != nil || len(text) != 6 {
		return color.NRGBA{}, fmt.Errorf("frame-phone: %q is not a colour, which is RRGGBB", text)
	}
	return color.NRGBA{R: uint8(value >> 16), G: uint8(value >> 8), B: uint8(value), A: 255}, nil
}

// CutTop returns the picture without its first rows, which hold the status bar of the device. It fails when
// nothing would be left of the picture.
func CutTop(source image.Image, rows int) (image.Image, error) {
	if rows == 0 {
		return source, nil
	}
	bounds := source.Bounds()
	if rows >= bounds.Dy() {
		return nil, fmt.Errorf("--top %d leaves nothing of a picture that is %d rows high", rows, bounds.Dy())
	}
	cut := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()-rows))
	draw.Draw(cut, cut.Bounds(), source, image.Pt(bounds.Min.X, bounds.Min.Y+rows), draw.Src)
	return cut, nil
}

// Frame returns the picture with the corners rounded, which are transparent outside the rounding, and a border
// of the colour over the edge of it. The edges are anti-aliased.
func Frame(source image.Image, radius, border int, edge color.NRGBA) *image.NRGBA {
	bounds := source.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	framed := image.NewNRGBA(image.Rect(0, 0, width, height))
	draw.Draw(framed, framed.Bounds(), source, bounds.Min, draw.Src)
	reach := float64(border) + 1
	for y := range height {
		for x := range width {
			depth := inside(x, y, width, height, radius)
			if depth > reach {
				continue // the middle of the picture, which nothing touches.
			}
			alpha := clamp(depth + 0.5)
			ring := clamp(float64(border) - depth + 0.5)
			pixel := framed.NRGBAAt(x, y)
			pixel.R = mix(pixel.R, edge.R, ring)
			pixel.G = mix(pixel.G, edge.G, ring)
			pixel.B = mix(pixel.B, edge.B, ring)
			pixel.A = uint8(math.Round(float64(pixel.A) * alpha))
			framed.SetNRGBA(x, y, pixel)
		}
	}
	return framed
}

// inside says how far inside the rounded rectangle the centre of a pixel is, in pixels: negative outside it.
func inside(x, y, width, height, radius int) float64 {
	px, py := float64(x)+0.5, float64(y)+0.5
	w, h, r := float64(width), float64(height), float64(radius)
	cx, cy, rounded := 0.0, 0.0, 0
	switch {
	case px < r:
		cx, rounded = r, rounded+1
	case px > w-r:
		cx, rounded = w-r, rounded+1
	}
	switch {
	case py < r:
		cy, rounded = r, rounded+1
	case py > h-r:
		cy, rounded = h-r, rounded+1
	}
	if rounded == 2 {
		return r - math.Hypot(px-cx, py-cy)
	}
	return math.Min(math.Min(px, py), math.Min(w-px, h-py))
}

func clamp(value float64) float64 { return math.Max(0, math.Min(1, value)) }

// mix is the colour that is from of the way from a to b.
func mix(a, b uint8, from float64) uint8 {
	return uint8(math.Round(float64(a)*(1-from) + float64(b)*from))
}
