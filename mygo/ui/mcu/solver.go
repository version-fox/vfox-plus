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

// The HCT solver of the Material colour utilities (Copyright 2021 Google LLC,
// Apache License 2.0): finds the sRGB colour closest to a hue, a chroma and a
// tone (L*), by first trying to solve for the CAM16 lightness and then
// bisecting along the edges of the sRGB gamut.

var scaledDiscountFromLinRGB = [3][3]float64{
	{0.001200833568784504, 0.002389694492170889, 0.0002795742885861124},
	{0.0005891086651375999, 0.0029785502573438758, 0.0003270666104008398},
	{0.00010146692491640572, 0.0005364214359186694, 0.0032979401770712076},
}

var linRGBFromScaledDiscount = [3][3]float64{
	{1373.2198709594231, -1100.4251190754821, -7.278681089101213},
	{-271.815969077903, 559.6580465940733, -32.46047482791194},
	{1.9622899599665666, -57.173814538844006, 308.7233197812385},
}

var yFromLinRGB = [3]float64{0.2126, 0.7152, 0.0722}

type vec3 = [3]float64

func sanitizeRadians(angle float64) float64 {
	return math.Mod(angle+math.Pi*8, math.Pi*2)
}

func trueDelinearized(component float64) float64 {
	normalized := component / 100
	var result float64
	if normalized <= 0.0031308 {
		result = normalized * 12.92
	} else {
		result = 1.055*math.Pow(normalized, reciprocal24) - 0.055
	}
	return result * 255
}

func chromaticAdaptation(component float64) float64 {
	af := math.Pow(math.Abs(component), value042)
	return signum(component) * 400 * af / (af + 27.13)
}

// hueOf is the hue, in radians, of a linear RGB colour.
func hueOf(linrgb vec3) float64 {
	scaled := matrixMultiply(linrgb, scaledDiscountFromLinRGB)
	rA := chromaticAdaptation(scaled[0])
	gA := chromaticAdaptation(scaled[1])
	bA := chromaticAdaptation(scaled[2])
	a := (11*rA + -12*gA + bA) / 11
	b := (rA + gA - 2*bA) / 9
	return math.Atan2(b, a)
}

func areInCyclicOrder(a, b, c float64) bool {
	return sanitizeRadians(b-a) < sanitizeRadians(c-a)
}

// intercept is how far mid lies between source and target.
func intercept(source, mid, target float64) float64 {
	return (mid - source) / (target - source)
}

func lerpPoint(source vec3, t float64, target vec3) vec3 {
	return vec3{
		source[0] + (target[0]-source[0])*t,
		source[1] + (target[1]-source[1])*t,
		source[2] + (target[2]-source[2])*t,
	}
}

// setCoordinate finds the point between source and target whose coordinate on
// axis is the given one.
func setCoordinate(source vec3, coordinate float64, target vec3, axis int) vec3 {
	t := intercept(source[axis], coordinate, target[axis])
	return lerpPoint(source, t, target)
}

func isBounded(x float64) bool { return 0 <= x && x <= 100 }

// nthVertex is the nth vertex of the intersection of the plane of the given
// luminance and the RGB cube, or (-1, -1, -1) if there is none.
func nthVertex(y float64, n int) vec3 {
	kR, kG, kB := yFromLinRGB[0], yFromLinRGB[1], yFromLinRGB[2]
	coordA := 100.0
	if n%4 <= 1 {
		coordA = 0
	}
	coordB := 100.0
	if n%2 == 0 {
		coordB = 0
	}
	none := vec3{-1, -1, -1}
	switch {
	case n < 4:
		g, b := coordA, coordB
		r := (y - g*kG - b*kB) / kR
		if isBounded(r) {
			return vec3{r, g, b}
		}
	case n < 8:
		b, r := coordA, coordB
		g := (y - r*kR - b*kB) / kG
		if isBounded(g) {
			return vec3{r, g, b}
		}
	default:
		r, g := coordA, coordB
		b := (y - r*kR - g*kG) / kB
		if isBounded(b) {
			return vec3{r, g, b}
		}
	}
	return none
}

// bisectToSegment finds the segment of the polygon of equal luminance that
// contains the target hue.
func bisectToSegment(y, targetHue float64) (vec3, vec3) {
	left := vec3{-1, -1, -1}
	right := left
	var leftHue, rightHue float64
	initialized := false
	uncut := true
	for n := 0; n < 12; n++ {
		mid := nthVertex(y, n)
		if mid[0] < 0 {
			continue
		}
		midHue := hueOf(mid)
		if !initialized {
			left, right = mid, mid
			leftHue, rightHue = midHue, midHue
			initialized = true
			continue
		}
		if uncut || areInCyclicOrder(leftHue, midHue, rightHue) {
			uncut = false
			if areInCyclicOrder(leftHue, targetHue, midHue) {
				right, rightHue = mid, midHue
			} else {
				left, leftHue = mid, midHue
			}
		}
	}
	return left, right
}

func midpoint(a, b vec3) vec3 {
	return vec3{(a[0] + b[0]) / 2, (a[1] + b[1]) / 2, (a[2] + b[2]) / 2}
}

func criticalPlaneBelow(x float64) int { return int(math.Floor(x - 0.5)) }

func criticalPlaneAbove(x float64) int { return int(math.Ceil(x - 0.5)) }

// bisectToLimit finds a color with the given luminance and hue that is as
// chromatic as possible within the gamut, to the precision of the critical
// planes.
func bisectToLimit(y, targetHue float64) vec3 {
	left, right := bisectToSegment(y, targetHue)
	leftHue := hueOf(left)
	for axis := 0; axis < 3; axis++ {
		if left[axis] == right[axis] {
			continue
		}
		var lPlane, rPlane int
		if left[axis] < right[axis] {
			lPlane = criticalPlaneBelow(trueDelinearized(left[axis]))
			rPlane = criticalPlaneAbove(trueDelinearized(right[axis]))
		} else {
			lPlane = criticalPlaneAbove(trueDelinearized(left[axis]))
			rPlane = criticalPlaneBelow(trueDelinearized(right[axis]))
		}
		for i := 0; i < 8; i++ {
			if abs(rPlane-lPlane) <= 1 {
				break
			}
			mPlane := int(math.Floor(float64(lPlane+rPlane) / 2))
			mid := setCoordinate(left, criticalPlanes[mPlane], right, axis)
			midHue := hueOf(mid)
			if areInCyclicOrder(leftHue, targetHue, midHue) {
				right, rPlane = mid, mPlane
			} else {
				left, leftHue, lPlane = mid, midHue, mPlane
			}
		}
	}
	return midpoint(left, right)
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func inverseChromaticAdaptation(adapted float64) float64 {
	adaptedAbs := math.Abs(adapted)
	base := math.Max(0, 27.13*adaptedAbs/(400-adaptedAbs))
	return signum(adapted) * math.Pow(base, reciprocal042)
}

// findResultByJ solves for the colour of the given hue, chroma and luminance
// by iterating on the CAM16 lightness J. It returns 0 if there is none inside
// the gamut.
func findResultByJ(hueRadians, chroma, y float64) uint32 {
	j := math.Sqrt(y) * 11
	vc := defaultViewingConditions
	tInnerCoeff := 1 / math.Pow(1.64-math.Pow(0.29, vc.n), 0.73)
	eHue := 0.25 * (math.Cos(hueRadians+2) + 3.8)
	p1 := eHue * (50000.0 / 13.0) * vc.nc * vc.ncb
	hSin, hCos := math.Sin(hueRadians), math.Cos(hueRadians)
	for round := 0; round < 5; round++ {
		jNormalized := j / 100
		alpha := 0.0
		if chroma != 0 && j != 0 {
			alpha = chroma / math.Sqrt(jNormalized)
		}
		t := math.Pow(alpha*tInnerCoeff, reciprocal09)
		ac := vc.aw * math.Pow(jNormalized, 1/vc.c/vc.z)
		p2 := ac / vc.nbb
		gamma := 23 * (p2 + 0.305) * t / (23*p1 + 11*t*hCos + 108*t*hSin)
		a := gamma * hCos
		b := gamma * hSin
		rA := (460*p2 + 451*a + 288*b) / 1403
		gA := (460*p2 - 891*a - 261*b) / 1403
		bA := (460*p2 - 220*a - 6300*b) / 1403
		linrgb := matrixMultiply(vec3{
			inverseChromaticAdaptation(rA),
			inverseChromaticAdaptation(gA),
			inverseChromaticAdaptation(bA),
		}, linRGBFromScaledDiscount)
		if linrgb[0] < 0 || linrgb[1] < 0 || linrgb[2] < 0 {
			return 0
		}
		fnj := yFromLinRGB[0]*linrgb[0] + yFromLinRGB[1]*linrgb[1] + yFromLinRGB[2]*linrgb[2]
		if fnj <= 0 {
			return 0
		}
		if round == 4 || math.Abs(fnj-y) < 0.002 {
			if linrgb[0] > 100.01 || linrgb[1] > 100.01 || linrgb[2] > 100.01 {
				return 0
			}
			return argbFromLinRGB(linrgb)
		}
		// Iterate with Newton's method on the lightness.
		j -= (fnj - y) * j / (2 * fnj)
	}
	return 0
}

// solveToARGB is the sRGB colour with the given hue (degrees), chroma and
// tone (L*), or the closest one inside the gamut.
func solveToARGB(hueDegrees, chroma, lstar float64) uint32 {
	if chroma < 0.0001 || lstar < 0.0001 || lstar > 99.9999 {
		return argbFromLstar(lstar)
	}
	hueDegrees = sanitizeDegrees(hueDegrees)
	hueRadians := hueDegrees / 180 * math.Pi
	y := yFromLstar(lstar)
	if exact := findResultByJ(hueRadians, chroma, y); exact != 0 {
		return exact
	}
	return argbFromLinRGB(bisectToLimit(y, hueRadians))
}
