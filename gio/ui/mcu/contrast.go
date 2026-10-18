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

// Contrast ratios between tones (Copyright 2021 Google LLC, Apache License
// 2.0), as WCAG defines them on the relative luminance.

// ratioOfTones is the contrast ratio, 1 to 21, of two tones.
func ratioOfTones(toneA, toneB float64) float64 {
	return ratioOfYs(yFromLstar(clamp(100, toneA)), yFromLstar(clamp(100, toneB)))
}

func ratioOfYs(y1, y2 float64) float64 {
	lighter := math.Max(y1, y2)
	darker := math.Min(y1, y2)
	return (lighter + 5) / (darker + 5)
}

// lighterTone is the lightest tone that has the contrast ratio with tone, or
// -1 if there is none.
func lighterTone(tone, ratio float64) float64 {
	if tone < 0 || tone > 100 {
		return -1
	}
	darkY := yFromLstar(tone)
	lightY := ratio*(darkY+5) - 5
	realContrast := ratioOfYs(lightY, darkY)
	delta := math.Abs(realContrast - ratio)
	if realContrast < ratio && delta > 0.04 {
		return -1
	}
	// Ensure the contrast is met after the conversion to 8 bits.
	result := lstarFromY(lightY) + 0.4
	if result < 0 || result > 100 {
		return -1
	}
	return result
}

// darkerTone is the darkest tone that has the contrast ratio with tone, or -1
// if there is none.
func darkerTone(tone, ratio float64) float64 {
	if tone < 0 || tone > 100 {
		return -1
	}
	lightY := yFromLstar(tone)
	darkY := (lightY+5)/ratio - 5
	realContrast := ratioOfYs(lightY, darkY)
	delta := math.Abs(realContrast - ratio)
	if realContrast < ratio && delta > 0.04 {
		return -1
	}
	result := lstarFromY(darkY) - 0.4
	if result < 0 || result > 100 {
		return -1
	}
	return result
}

// lighterToneUnsafe is lighterTone, but 100 instead of -1.
func lighterToneUnsafe(tone, ratio float64) float64 {
	if lighter := lighterTone(tone, ratio); lighter >= 0 {
		return lighter
	}
	return 100
}

// darkerToneUnsafe is darkerTone, but 0 instead of -1.
func darkerToneUnsafe(tone, ratio float64) float64 {
	if darker := darkerTone(tone, ratio); darker >= 0 {
		return darker
	}
	return 0
}
