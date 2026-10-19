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

// Hct is a colour in the HCT colour space of Material 3: the hue and the
// chroma of CAM16 and the tone, which is the L* of CIE L*a*b*. It is
// perceptually uniform, and a tone of 50 or less always has a contrast ratio
// of 3 to the tone of 100, whatever the hue and the chroma.
type Hct struct {
	// Hue is the CAM16 hue in degrees, from 0 to 360.
	Hue float64
	// Chroma is the CAM16 chroma, roughly 0 to 120.
	Chroma float64
	// Tone is the L* of CIE L*a*b*, from 0 (black) to 100 (white).
	Tone float64
	argb uint32
}

// HctFromARGB converts an sRGB colour, 0xAARRGGBB, to HCT.
func HctFromARGB(argb uint32) Hct {
	cam := cam16FromARGB(argb)
	return Hct{Hue: cam.hue, Chroma: cam.chroma, Tone: lstarFromARGB(argb), argb: argb}
}

// HctFrom finds the colour with the given hue, chroma and tone that sRGB can
// show; the chroma is reduced when the gamut does not reach it.
func HctFrom(hue, chroma, tone float64) Hct {
	return HctFromARGB(solveToARGB(hue, chroma, tone))
}

// ARGB is the sRGB colour, 0xAARRGGBB with an opaque alpha.
func (h Hct) ARGB() uint32 { return h.argb }

// isYellow reports whether hue is in the range where the tone 99 of a palette
// is not representable.
func isYellow(hue float64) bool { return hue >= 105 && hue < 125 }
