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

// Colour conversions and maths of the Material colour utilities (Copyright
// 2021 Google LLC, Apache License 2.0). Colours are ARGB values in a uint32:
// 0xAARRGGBB.

var srgbToXYZ = [3][3]float64{
	{0.41233895, 0.35762064, 0.18051042},
	{0.2126, 0.7152, 0.0722},
	{0.01932141, 0.11916382, 0.95034478},
}

var whitePointD65 = [3]float64{95.047, 100.0, 108.883}

// The reference implementation divides these double precision numbers at run
// time, while Go evaluates a constant expression exactly and rounds once,
// which can differ in the last bit; variables keep the arithmetic the same.
var (
	valuePi, value24, value042, value09, value36 = math.Pi, 2.4, 0.42, 0.9, 3.6
	reciprocal24                                 = 1 / value24
	reciprocal042                                = 1 / value042
	reciprocal09                                 = 1 / value09
)

func signum(value float64) float64 {
	switch {
	case value < 0:
		return -1
	case value == 0:
		return 0
	default:
		return 1
	}
}

func lerp(start, stop, amount float64) float64 {
	return (1-amount)*start + amount*stop
}

// clamp limits value to the range from 0 to high.
func clamp(high, value float64) float64 {
	return math.Min(math.Max(value, 0), high)
}

// jsRound rounds like JavaScript's Math.round, half towards positive
// infinity, which the reference implementation relies on.
func jsRound(value float64) float64 { return math.Floor(value + 0.5) }

// sanitizeDegrees brings an angle into [0, 360).
func sanitizeDegrees(degrees float64) float64 {
	degrees = math.Mod(degrees, 360)
	if degrees < 0 {
		degrees += 360
	}
	return degrees
}

func matrixMultiply(row [3]float64, matrix [3][3]float64) [3]float64 {
	return [3]float64{
		row[0]*matrix[0][0] + row[1]*matrix[0][1] + row[2]*matrix[0][2],
		row[0]*matrix[1][0] + row[1]*matrix[1][1] + row[2]*matrix[1][2],
		row[0]*matrix[2][0] + row[1]*matrix[2][1] + row[2]*matrix[2][2],
	}
}

func argbFromRGB(red, green, blue int) uint32 {
	return 0xFF000000 | uint32(red&255)<<16 | uint32(green&255)<<8 | uint32(blue&255)
}

func argbFromLinRGB(linrgb [3]float64) uint32 {
	return argbFromRGB(delinearized(linrgb[0]), delinearized(linrgb[1]), delinearized(linrgb[2]))
}

func redOf(argb uint32) int   { return int(argb >> 16 & 255) }
func greenOf(argb uint32) int { return int(argb >> 8 & 255) }
func blueOf(argb uint32) int  { return int(argb & 255) }

func argbFromLstar(lstar float64) uint32 {
	component := delinearized(yFromLstar(lstar))
	return argbFromRGB(component, component, component)
}

func lstarFromARGB(argb uint32) float64 {
	y := matrixMultiply([3]float64{
		linearized(redOf(argb)), linearized(greenOf(argb)), linearized(blueOf(argb)),
	}, srgbToXYZ)[1]
	return 116*labF(y/100) - 16
}

func yFromLstar(lstar float64) float64 { return 100 * labInvf((lstar+16)/116) }

func lstarFromY(y float64) float64 { return labF(y/100)*116 - 16 }

// linearized converts a 0 to 255 channel value to linear light, 0 to 100.
func linearized(component int) float64 {
	normalized := float64(component) / 255
	if normalized <= 0.040449936 {
		return normalized / 12.92 * 100
	}
	return math.Pow((normalized+0.055)/1.055, 2.4) * 100
}

// delinearized converts linear light, 0 to 100, to a 0 to 255 channel value.
func delinearized(component float64) int {
	normalized := component / 100
	var result float64
	if normalized <= 0.0031308 {
		result = normalized * 12.92
	} else {
		result = 1.055*math.Pow(normalized, reciprocal24) - 0.055
	}
	return int(clamp(255, jsRound(result*255)))
}

func labF(t float64) float64 {
	const e = 216.0 / 24389.0
	const kappa = 24389.0 / 27.0
	if t > e {
		return math.Pow(t, 1.0/3.0)
	}
	return (kappa*t + 16) / 116
}

func labInvf(ft float64) float64 {
	const e = 216.0 / 24389.0
	const kappa = 24389.0 / 27.0
	ft3 := ft * ft * ft
	if ft3 > e {
		return ft3
	}
	return (116*ft - 16) / kappa
}
