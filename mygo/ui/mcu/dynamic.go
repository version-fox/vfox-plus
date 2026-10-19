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

package mcu

import "math"

// The dynamic colours of the 2021 specification of the Material colour
// utilities (Copyright 2021 Google LLC, Apache License 2.0), for the tonal
// spot scheme: every colour role is a tone of a palette, nudged until it has
// the contrast to its background that the contrast curve asks for.

// contrastCurve is the contrast ratio that a colour role needs, by contrast
// level: low (-1), normal (0), medium (0.5) and high (1).
type contrastCurve struct{ low, normal, medium, high float64 }

func (c contrastCurve) get(level float64) float64 {
	switch {
	case level <= -1:
		return c.low
	case level < 0:
		return lerp(c.low, c.normal, (level-(-1))/1)
	case level < 0.5:
		return lerp(c.normal, c.medium, (level-0)/0.5)
	case level < 1:
		return lerp(c.medium, c.high, (level-0.5)/0.5)
	default:
		return c.high
	}
}

// polarity says which of the two roles of a tone delta pair is lighter.
type polarity int

const (
	polarityNearer polarity = iota
	polarityLighter
)

// toneDeltaPair asks for two roles to differ by at least delta in tone.
type toneDeltaPair struct {
	roleA, roleB *role
	delta        float64
	polarity     polarity
	stayTogether bool
}

// scheme is a tonal spot scheme: the palettes of a seed colour, light or dark,
// at a contrast level.
type scheme struct {
	dark                                                       bool
	contrastLevel                                              float64
	primary, secondary, tertiary, neutral, neutralVariant, err *TonalPalette
}

func newScheme(seed Hct, dark bool, contrastLevel float64) *scheme {
	return &scheme{
		dark:           dark,
		contrastLevel:  contrastLevel,
		primary:        NewTonalPalette(seed.Hue, 36),
		secondary:      NewTonalPalette(seed.Hue, 16),
		tertiary:       NewTonalPalette(sanitizeDegrees(seed.Hue+60), 24),
		neutral:        NewTonalPalette(seed.Hue, 6),
		neutralVariant: NewTonalPalette(seed.Hue, 8),
		err:            NewTonalPalette(25, 84),
	}
}

// role is one colour role, such as `primary` or `onSurface`.
type role struct {
	name             string
	palette          func(*scheme) *TonalPalette
	tone             func(*scheme) float64
	isBackground     bool
	background       func(*scheme) *role
	secondBackground func(*scheme) *role
	contrast         *contrastCurve
	toneDeltaPair    func(*scheme) *toneDeltaPair
}

func (r *role) hct(s *scheme) Hct { return r.palette(s).Hct(r.getTone(s)) }

// getTone is the tone of the role in the scheme, after the adjustments for the
// contrast to its background and for its tone delta pair.
func (r *role) getTone(s *scheme) float64 {
	decreasingContrast := s.contrastLevel < 0
	if r.toneDeltaPair != nil {
		return r.toneOfPair(s, r.toneDeltaPair(s), decreasingContrast)
	}
	answer := r.tone(s)
	if r.background == nil || r.contrast == nil {
		// No adjustment for colours without a background.
		return answer
	}
	bgTone := r.background(s).getTone(s)
	desiredRatio := r.contrast.get(s.contrastLevel)
	if ratioOfTones(bgTone, answer) < desiredRatio {
		answer = foregroundTone(bgTone, desiredRatio)
	}
	if decreasingContrast {
		answer = foregroundTone(bgTone, desiredRatio)
	}
	// Tones between 50 and 60 have a poor contrast to both black and white
	// text: background colours skip them.
	if r.isBackground && answer >= 50 && answer < 60 {
		if ratioOfTones(49, bgTone) >= desiredRatio {
			answer = 49
		} else {
			answer = 60
		}
	}
	if r.secondBackground == nil {
		return answer
	}
	bgTone1 := r.background(s).getTone(s)
	bgTone2 := r.secondBackground(s).getTone(s)
	upper, lower := math.Max(bgTone1, bgTone2), math.Min(bgTone1, bgTone2)
	if ratioOfTones(upper, answer) >= desiredRatio && ratioOfTones(lower, answer) >= desiredRatio {
		return answer
	}
	// Neither can be met: choose the tone that contrasts with both.
	lightOption := lighterTone(upper, desiredRatio)
	darkOption := darkerTone(lower, desiredRatio)
	available := 0
	if lightOption != -1 {
		available++
	}
	if darkOption != -1 {
		available++
	}
	if tonePrefersLightForeground(bgTone1) || tonePrefersLightForeground(bgTone2) {
		if lightOption < 0 {
			return 100
		}
		return lightOption
	}
	if available == 1 {
		if lightOption != -1 {
			return lightOption
		}
		return darkOption
	}
	if darkOption < 0 {
		return 0
	}
	return darkOption
}

func (r *role) toneOfPair(s *scheme, pair *toneDeltaPair, decreasingContrast bool) float64 {
	aIsNearer := pair.polarity == polarityNearer ||
		pair.polarity == polarityLighter && !s.dark
	nearer, farther := pair.roleB, pair.roleA
	if aIsNearer {
		nearer, farther = pair.roleA, pair.roleB
	}
	amNearer := r.name == nearer.name
	expansionDir := -1.0
	if s.dark {
		expansionDir = 1
	}
	nTone := nearer.tone(s)
	fTone := farther.tone(s)
	if r.background != nil && nearer.contrast != nil && farther.contrast != nil {
		bgTone := r.background(s).getTone(s)
		nContrast := nearer.contrast.get(s.contrastLevel)
		fContrast := farther.contrast.get(s.contrastLevel)
		if ratioOfTones(bgTone, nTone) < nContrast {
			nTone = foregroundTone(bgTone, nContrast)
		}
		if ratioOfTones(bgTone, fTone) < fContrast {
			fTone = foregroundTone(bgTone, fContrast)
		}
		if decreasingContrast {
			nTone = foregroundTone(bgTone, nContrast)
			fTone = foregroundTone(bgTone, fContrast)
		}
	}
	// Expand the farther role away from the nearer one until the delta holds.
	if (fTone-nTone)*expansionDir < pair.delta {
		fTone = clamp(100, nTone+pair.delta*expansionDir)
		if (fTone-nTone)*expansionDir < pair.delta {
			nTone = clamp(100, fTone-pair.delta*expansionDir)
		}
	}
	// Avoid the tones between 50 and 60.
	switch {
	case nTone >= 50 && nTone < 60:
		if expansionDir > 0 {
			nTone = 60
			fTone = math.Max(fTone, nTone+pair.delta*expansionDir)
		} else {
			nTone = 49
			fTone = math.Min(fTone, nTone+pair.delta*expansionDir)
		}
	case fTone >= 50 && fTone < 60:
		switch {
		case pair.stayTogether && expansionDir > 0:
			nTone = 60
			fTone = math.Max(fTone, nTone+pair.delta*expansionDir)
		case pair.stayTogether:
			nTone = 49
			fTone = math.Min(fTone, nTone+pair.delta*expansionDir)
		case expansionDir > 0:
			fTone = 60
		default:
			fTone = 49
		}
	}
	if amNearer {
		return nTone
	}
	return fTone
}

// foregroundTone is the tone for text on a background of bgTone that has the
// contrast ratio, preferring the lighter or the darker one as the tone of the
// background suggests.
func foregroundTone(bgTone, ratio float64) float64 {
	lighter := lighterToneUnsafe(bgTone, ratio)
	darker := darkerToneUnsafe(bgTone, ratio)
	lighterRatio := ratioOfTones(lighter, bgTone)
	darkerRatio := ratioOfTones(darker, bgTone)
	if tonePrefersLightForeground(bgTone) {
		negligibleDifference := math.Abs(lighterRatio-darkerRatio) < 0.1 && lighterRatio < ratio && darkerRatio < ratio
		if lighterRatio >= ratio || lighterRatio >= darkerRatio || negligibleDifference {
			return lighter
		}
		return darker
	}
	if darkerRatio >= ratio || darkerRatio >= lighterRatio {
		return darker
	}
	return lighter
}

func tonePrefersLightForeground(tone float64) bool { return jsRound(tone) < 60 }

// roles are all colour roles of the 2021 specification that the tonal spot
// scheme uses.
type roles struct {
	background, onBackground                                                      *role
	surface, surfaceDim, surfaceBright                                            *role
	surfaceContainerLowest, surfaceContainerLow, surfaceContainer                 *role
	surfaceContainerHigh, surfaceContainerHighest                                 *role
	onSurface, surfaceVariant, onSurfaceVariant, inverseSurface, inverseOnSurface *role
	outline, outlineVariant, shadow, scrim                                        *role
	primary, onPrimary, primaryContainer, onPrimaryContainer, inversePrimary      *role
	secondary, onSecondary, secondaryContainer, onSecondaryContainer              *role
	tertiary, onTertiary, tertiaryContainer, onTertiaryContainer                  *role
	error, onError, errorContainer, onErrorContainer                              *role
	primaryFixed, primaryFixedDim, onPrimaryFixed, onPrimaryFixedVariant          *role
	secondaryFixed, secondaryFixedDim, onSecondaryFixed, onSecondaryFixedVariant  *role
	tertiaryFixed, tertiaryFixedDim, onTertiaryFixed, onTertiaryFixedVariant      *role
}

func (r *roles) highestSurface(s *scheme) *role {
	if s.dark {
		return r.surfaceBright
	}
	return r.surfaceDim
}

func fixed(light, dark float64) func(*scheme) float64 {
	return func(s *scheme) float64 {
		if s.dark {
			return dark
		}
		return light
	}
}

func always(tone float64) func(*scheme) float64 {
	return func(*scheme) float64 { return tone }
}

func curve(low, normal, medium, high float64) *contrastCurve {
	return &contrastCurve{low, normal, medium, high}
}

// byLevel picks the tone of a surface role from a contrast curve.
func byLevel(light, dark contrastCurve) func(*scheme) float64 {
	return func(s *scheme) float64 {
		if s.dark {
			return dark.get(s.contrastLevel)
		}
		return light.get(s.contrastLevel)
	}
}

// newRoles builds the roles. They refer to each other, so they are built
// together.
func newRoles() *roles {
	r := &roles{}
	neutral := func(s *scheme) *TonalPalette { return s.neutral }
	neutralVariant := func(s *scheme) *TonalPalette { return s.neutralVariant }
	primaryPalette := func(s *scheme) *TonalPalette { return s.primary }
	secondaryPalette := func(s *scheme) *TonalPalette { return s.secondary }
	tertiaryPalette := func(s *scheme) *TonalPalette { return s.tertiary }
	errorPalette := func(s *scheme) *TonalPalette { return s.err }
	onHighestSurface := func(s *scheme) *role { return r.highestSurface(s) }
	of := func(target **role) func(*scheme) *role { return func(*scheme) *role { return *target } }

	r.background = &role{name: "background", palette: neutral, tone: fixed(98, 6), isBackground: true}
	r.onBackground = &role{
		name: "on_background", palette: neutral, tone: fixed(10, 90),
		background: of(&r.background), contrast: curve(3, 3, 4.5, 7),
	}
	r.surface = &role{name: "surface", palette: neutral, tone: fixed(98, 6), isBackground: true}
	r.surfaceDim = &role{
		name: "surface_dim", palette: neutral, isBackground: true,
		tone: byLevel(contrastCurve{87, 87, 80, 75}, contrastCurve{6, 6, 6, 6}),
	}
	r.surfaceBright = &role{
		name: "surface_bright", palette: neutral, isBackground: true,
		tone: byLevel(contrastCurve{98, 98, 98, 98}, contrastCurve{24, 24, 29, 34}),
	}
	r.surfaceContainerLowest = &role{
		name: "surface_container_lowest", palette: neutral, isBackground: true,
		tone: byLevel(contrastCurve{100, 100, 100, 100}, contrastCurve{4, 4, 2, 0}),
	}
	r.surfaceContainerLow = &role{
		name: "surface_container_low", palette: neutral, isBackground: true,
		tone: byLevel(contrastCurve{96, 96, 96, 95}, contrastCurve{10, 10, 11, 12}),
	}
	r.surfaceContainer = &role{
		name: "surface_container", palette: neutral, isBackground: true,
		tone: byLevel(contrastCurve{94, 94, 92, 90}, contrastCurve{12, 12, 16, 20}),
	}
	r.surfaceContainerHigh = &role{
		name: "surface_container_high", palette: neutral, isBackground: true,
		tone: byLevel(contrastCurve{92, 92, 88, 85}, contrastCurve{17, 17, 21, 25}),
	}
	r.surfaceContainerHighest = &role{
		name: "surface_container_highest", palette: neutral, isBackground: true,
		tone: byLevel(contrastCurve{90, 90, 84, 80}, contrastCurve{22, 22, 26, 30}),
	}
	r.onSurface = &role{
		name: "on_surface", palette: neutral, tone: fixed(10, 90),
		background: onHighestSurface, contrast: curve(4.5, 7, 11, 21),
	}
	r.surfaceVariant = &role{name: "surface_variant", palette: neutralVariant, tone: fixed(90, 30), isBackground: true}
	r.onSurfaceVariant = &role{
		name: "on_surface_variant", palette: neutralVariant, tone: fixed(30, 80),
		background: onHighestSurface, contrast: curve(3, 4.5, 7, 11),
	}
	r.inverseSurface = &role{name: "inverse_surface", palette: neutral, tone: fixed(20, 90), isBackground: true}
	r.inverseOnSurface = &role{
		name: "inverse_on_surface", palette: neutral, tone: fixed(95, 20),
		background: of(&r.inverseSurface), contrast: curve(4.5, 7, 11, 21),
	}
	r.outline = &role{
		name: "outline", palette: neutralVariant, tone: fixed(50, 60),
		background: onHighestSurface, contrast: curve(1.5, 3, 4.5, 7),
	}
	r.outlineVariant = &role{
		name: "outline_variant", palette: neutralVariant, tone: fixed(80, 30),
		background: onHighestSurface, contrast: curve(1, 1, 3, 4.5),
	}
	r.shadow = &role{name: "shadow", palette: neutral, tone: always(0)}
	r.scrim = &role{name: "scrim", palette: neutral, tone: always(0)}

	// accent builds the roles of an accent colour: the colour, the colour on
	// it, the container and the colour on the container.
	accent := func(prefix string, palette func(*scheme) *TonalPalette,
		main, onMain, container, onContainer **role,
	) {
		*main = &role{
			name: prefix, palette: palette, tone: fixed(40, 80), isBackground: true,
			background: onHighestSurface, contrast: curve(3, 4.5, 7, 7),
			toneDeltaPair: func(*scheme) *toneDeltaPair {
				return &toneDeltaPair{roleA: *container, roleB: *main, delta: 10, polarity: polarityNearer}
			},
		}
		*onMain = &role{
			name: "on_" + prefix, palette: palette, tone: fixed(100, 20),
			background: of(main), contrast: curve(4.5, 7, 11, 21),
		}
		*container = &role{
			name: prefix + "_container", palette: palette, tone: fixed(90, 30), isBackground: true,
			background: onHighestSurface, contrast: curve(1, 1, 3, 4.5),
			toneDeltaPair: func(*scheme) *toneDeltaPair {
				return &toneDeltaPair{roleA: *container, roleB: *main, delta: 10, polarity: polarityNearer}
			},
		}
		*onContainer = &role{
			name: "on_" + prefix + "_container", palette: palette, tone: fixed(30, 90),
			background: of(container), contrast: curve(3, 4.5, 7, 11),
		}
	}
	accent("primary", primaryPalette, &r.primary, &r.onPrimary, &r.primaryContainer, &r.onPrimaryContainer)
	accent("secondary", secondaryPalette, &r.secondary, &r.onSecondary, &r.secondaryContainer, &r.onSecondaryContainer)
	accent("tertiary", tertiaryPalette, &r.tertiary, &r.onTertiary, &r.tertiaryContainer, &r.onTertiaryContainer)
	accent("error", errorPalette, &r.error, &r.onError, &r.errorContainer, &r.onErrorContainer)
	r.inversePrimary = &role{
		name: "inverse_primary", palette: primaryPalette, tone: fixed(80, 40),
		background: of(&r.inverseSurface), contrast: curve(3, 4.5, 7, 7),
	}

	// fixedRoles builds the roles of an accent colour that does not change
	// with the theme.
	fixedRoles := func(prefix string, palette func(*scheme) *TonalPalette,
		fixedRole, dim, onFixed, onVariant **role,
	) {
		pair := func(*scheme) *toneDeltaPair {
			return &toneDeltaPair{roleA: *fixedRole, roleB: *dim, delta: 10, polarity: polarityLighter, stayTogether: true}
		}
		*fixedRole = &role{
			name: prefix + "_fixed", palette: palette, tone: always(90), isBackground: true,
			background: onHighestSurface, contrast: curve(1, 1, 3, 4.5), toneDeltaPair: pair,
		}
		*dim = &role{
			name: prefix + "_fixed_dim", palette: palette, tone: always(80), isBackground: true,
			background: onHighestSurface, contrast: curve(1, 1, 3, 4.5), toneDeltaPair: pair,
		}
		*onFixed = &role{
			name: "on_" + prefix + "_fixed", palette: palette, tone: always(10),
			background: of(dim), secondBackground: of(fixedRole), contrast: curve(4.5, 7, 11, 21),
		}
		*onVariant = &role{
			name: "on_" + prefix + "_fixed_variant", palette: palette, tone: always(30),
			background: of(dim), secondBackground: of(fixedRole), contrast: curve(3, 4.5, 7, 11),
		}
	}
	fixedRoles("primary", primaryPalette, &r.primaryFixed, &r.primaryFixedDim, &r.onPrimaryFixed, &r.onPrimaryFixedVariant)
	fixedRoles("secondary", secondaryPalette, &r.secondaryFixed, &r.secondaryFixedDim, &r.onSecondaryFixed, &r.onSecondaryFixedVariant)
	fixedRoles("tertiary", tertiaryPalette, &r.tertiaryFixed, &r.tertiaryFixedDim, &r.onTertiaryFixed, &r.onTertiaryFixedVariant)
	return r
}

var colorRoles = newRoles()
