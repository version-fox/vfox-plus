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
	"image/color"

	"gioui.org/text"

	"github.com/version-fox/vfox-plus/gio/ui/mcu"
)

// Palette is the colour scheme as colours Gio draws with.
type Palette struct {
	Background, OnBackground                                             color.NRGBA
	Surface, SurfaceDim, SurfaceBright                                   color.NRGBA
	SurfaceContainerLowest, SurfaceContainerLow, SurfaceContainer        color.NRGBA
	SurfaceContainerHigh, SurfaceContainerHighest                        color.NRGBA
	OnSurface, OnSurfaceVariant, Outline, OutlineVariant                 color.NRGBA
	InverseSurface, InverseOnSurface, InversePrimary                     color.NRGBA
	SurfaceVariant, Shadow, Scrim                                        color.NRGBA
	Primary, OnPrimary, PrimaryContainer, OnPrimaryContainer             color.NRGBA
	PrimaryFixed, PrimaryFixedDim, OnPrimaryFixed, OnPrimaryFixedVariant color.NRGBA
	Secondary, OnSecondary, SecondaryContainer, OnSecondaryContainer     color.NRGBA
	Tertiary, OnTertiary, TertiaryContainer, OnTertiaryContainer         color.NRGBA
	Error, OnError, ErrorContainer, OnErrorContainer                     color.NRGBA
}

// ARGB converts a colour of the colour utilities.
func ARGB(v uint32) color.NRGBA {
	return color.NRGBA{R: uint8(v >> 16 & 0xFF), G: uint8(v >> 8 & 0xFF), B: uint8(v & 0xFF), A: uint8(v >> 24 & 0xFF)}
}

// PaletteFor generates the palette of the tonal spot scheme of seed.
func PaletteFor(seed uint32, dark bool) Palette {
	c := mcu.Generate(seed, dark)
	return Palette{
		Background: ARGB(c.Background), OnBackground: ARGB(c.OnBackground),
		Surface: ARGB(c.Surface), SurfaceDim: ARGB(c.SurfaceDim), SurfaceBright: ARGB(c.SurfaceBright),
		SurfaceContainerLowest:  ARGB(c.SurfaceContainerLowest),
		SurfaceContainerLow:     ARGB(c.SurfaceContainerLow),
		SurfaceContainer:        ARGB(c.SurfaceContainer),
		SurfaceContainerHigh:    ARGB(c.SurfaceContainerHigh),
		SurfaceContainerHighest: ARGB(c.SurfaceContainerHighest),
		OnSurface:               ARGB(c.OnSurface), OnSurfaceVariant: ARGB(c.OnSurfaceVariant),
		Outline: ARGB(c.Outline), OutlineVariant: ARGB(c.OutlineVariant),
		InverseSurface: ARGB(c.InverseSurface), InverseOnSurface: ARGB(c.InverseOnSurface),
		InversePrimary: ARGB(c.InversePrimary),
		SurfaceVariant: ARGB(c.SurfaceVariant), Shadow: ARGB(c.Shadow), Scrim: ARGB(c.Scrim),
		Primary: ARGB(c.Primary), OnPrimary: ARGB(c.OnPrimary),
		PrimaryContainer: ARGB(c.PrimaryContainer), OnPrimaryContainer: ARGB(c.OnPrimaryContainer),
		PrimaryFixed: ARGB(c.PrimaryFixed), PrimaryFixedDim: ARGB(c.PrimaryFixedDim),
		OnPrimaryFixed: ARGB(c.OnPrimaryFixed), OnPrimaryFixedVariant: ARGB(c.OnPrimaryFixedVariant),
		Secondary: ARGB(c.Secondary), OnSecondary: ARGB(c.OnSecondary),
		SecondaryContainer: ARGB(c.SecondaryContainer), OnSecondaryContainer: ARGB(c.OnSecondaryContainer),
		Tertiary: ARGB(c.Tertiary), OnTertiary: ARGB(c.OnTertiary),
		TertiaryContainer: ARGB(c.TertiaryContainer), OnTertiaryContainer: ARGB(c.OnTertiaryContainer),
		Error: ARGB(c.Error), OnError: ARGB(c.OnError),
		ErrorContainer: ARGB(c.ErrorContainer), OnErrorContainer: ARGB(c.OnErrorContainer),
	}
}

// Alpha is the colour with its opacity multiplied by a (0 to 1).
func Alpha(c color.NRGBA, a float32) color.NRGBA {
	c.A = uint8(float32(c.A)*clamp01(a) + 0.5)
	return c
}

// Mix is the colour between a and b: a for t 0, b for t 1.
func Mix(a, b color.NRGBA, t float32) color.NRGBA {
	t = clamp01(t)
	mix := func(x, y uint8) uint8 { return uint8(float32(x)*(1-t) + float32(y)*t + 0.5) }
	return color.NRGBA{R: mix(a.R, b.R), G: mix(a.G, b.G), B: mix(a.B, b.B), A: mix(a.A, b.A)}
}

func clamp01(v float32) float32 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}

// Theme is the look of the interface: colours, type, motion.
type Theme struct {
	// Colors is the colour scheme.
	Colors Palette
	// Dark says that the scheme is the dark one.
	Dark bool
	// Motion holds the springs.
	Motion MotionScheme
	// ReducedMotion makes springs jump to their targets and indeterminate
	// indicators stop.
	ReducedMotion bool
	// Fonts holds the faces and the shaper.
	Fonts *Fonts
}

// NewTheme creates the theme for a seed colour, with the expressive springs.
func NewTheme(seed uint32, dark, reducedMotion bool, fonts *Fonts) *Theme {
	return &Theme{
		Colors: PaletteFor(seed, dark), Dark: dark, Motion: Expressive,
		ReducedMotion: reducedMotion, Fonts: fonts,
	}
}

// Shaper is the text shaper.
func (t *Theme) Shaper() *text.Shaper { return t.Fonts.Shaper }

// TextRole is one of the roles of the Material 3 type scale.
type TextRole int

// The roles of the type scale.
const (
	DisplayLarge TextRole = iota
	DisplayMedium
	DisplaySmall
	HeadlineLarge
	HeadlineMedium
	HeadlineSmall
	TitleLarge
	TitleMedium
	TitleSmall
	BodyLarge
	BodyMedium
	BodySmall
	LabelLarge
	LabelMedium
	LabelSmall
)

// TextStyle is a size, a line height and a weight.
type TextStyle struct {
	// Size is the font size in sp.
	Size float32
	// Line is the line height in sp.
	Line float32
	// Weight is the weight, from 100 to 900.
	Weight int
}

// The Material 3 type scale, with the baseline weights of Flutter's
// Typography.material2021.
var scale = [...]TextStyle{
	DisplayLarge:   {57, 64, 400},
	DisplayMedium:  {45, 52, 400},
	DisplaySmall:   {36, 44, 400},
	HeadlineLarge:  {32, 40, 400},
	HeadlineMedium: {28, 36, 400},
	HeadlineSmall:  {24, 32, 400},
	TitleLarge:     {22, 28, 400},
	TitleMedium:    {16, 24, 500},
	TitleSmall:     {14, 20, 500},
	BodyLarge:      {16, 24, 400},
	BodyMedium:     {14, 20, 400},
	BodySmall:      {12, 16, 400},
	LabelLarge:     {14, 20, 500},
	LabelMedium:    {12, 16, 500},
	LabelSmall:     {11, 16, 500},
}

// The weights of the emphasized variants of the roles of the type scale.
var emphasizedWeight = [...]int{
	DisplayLarge: 500, DisplayMedium: 500, DisplaySmall: 500,
	HeadlineLarge: 500, HeadlineMedium: 500, HeadlineSmall: 500,
	TitleLarge: 500, TitleMedium: 700, TitleSmall: 700,
	BodyLarge: 500, BodyMedium: 500, BodySmall: 500,
	LabelLarge: 700, LabelMedium: 700, LabelSmall: 700,
}

// Type is the style of a role of the type scale.
func Type(role TextRole) TextStyle { return scale[role] }

// Emph is the emphasized style of a role of the type scale, which Material 3
// Expressive uses for the text that matters most on a screen.
func Emph(role TextRole) TextStyle {
	style := scale[role]
	style.Weight = emphasizedWeight[role]
	return style
}

// WithWeight is the style with another weight.
func (s TextStyle) WithWeight(weight int) TextStyle {
	s.Weight = weight
	return s
}

// WithSize is the style with another size, which keeps the proportion of the
// line height.
func (s TextStyle) WithSize(size float32) TextStyle {
	s.Line = s.Line * size / s.Size
	s.Size = size
	return s
}
