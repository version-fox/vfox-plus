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

package appicon

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

// The icons of the phone app. mygo reads them from the resources directory of the app at build
// time (mygo/phone/mygo.json names them), and the repository holds no binary files, so `vfxtool
// appicon` writes them, in the build and in the dev container, instead of a commit.
const (
	// IconFile is the icon of the app, which the launchers of the systems without adaptive icons show.
	IconFile = "icon.png"
	// ForegroundFile is the foreground of the adaptive icon of Android 8.0 and later.
	ForegroundFile = "icon-foreground.png"

	// PhoneIconSize is the side of IconFile: the desktop icon (Render), as big as mygo wants it.
	PhoneIconSize = 1024
	// AdaptiveForegroundSize is the side of ForegroundFile: 108 dp at 4 pixels a dp, the size of the
	// densest screens. mygo scales it to the others.
	AdaptiveForegroundSize = 432
	// AdaptiveBackground is the colour under the foreground, which shows where a launcher moves the
	// foreground: the darkest colour of the plate.
	AdaptiveBackground = "#1B1268"
)

// RenderAdaptiveForeground draws the foreground of the adaptive icon on a size by size image: the
// plate over the whole canvas, which the launcher's mask shows in its middle 72 of 108 dp, the halo
// and the fox, at the scales of the Android icon. It is opaque, so that AdaptiveBackground only shows
// where a launcher moves the layer.
func RenderAdaptiveForeground(size int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	base := affine{float64(size) / canvas, float64(size) / canvas, 0, 0}
	fillWith(img, shape{id: "adaptive-plate", d: androidBackground}, base, bleedPaint(base))
	centred := canvas / 2 * (1 - androidHaloScale)
	fill(img, halo, base.compose(affine{androidHaloScale, androidHaloScale, centred, centred}))
	fox := base.compose(affine{androidFoxScale, androidFoxScale, androidFoxX, androidFoxY})
	for _, s := range foxShapes {
		fill(img, s, fox)
	}
	return img
}

// bleedPaint is the gradient of the plate stretched over the whole canvas, as the adaptive icon of
// the phone app draws it: the colours of the plate, from its corner to the opposite one.
func bleedPaint(tr affine) image.Image {
	for _, g := range gradients {
		if g.id == "plate" {
			x1, y1 := tr.point(0, 0)
			x2, y2 := tr.point(canvas, canvas)
			return &linearGradient{x1: x1, y1: y1, x2: x2, y2: y2, stops: g.stops}
		}
	}
	panic("appicon: no plate gradient") // The data is part of the source.
}

// WritePhoneIcons writes the icons of the phone app into dir, which it creates: IconFile and
// ForegroundFile.
func WritePhoneIcons(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := writePNG(filepath.Join(dir, IconFile), Render(PhoneIconSize, PhoneIconSize)); err != nil {
		return err
	}
	return writePNG(filepath.Join(dir, ForegroundFile), RenderAdaptiveForeground(AdaptiveForegroundSize))
}

// writePNG writes img to path as a PNG file.
func writePNG(path string, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// Run implements `appicon <directory>`: it writes the icons of the phone app into the directory,
// for example `vfxtool appicon mygo/phone/resources` before `mygo build`.
func Run(args []string) error {
	if len(args) != 1 {
		return errors.New("Usage: appicon <directory>")
	}
	return WritePhoneIcons(args[0])
}
