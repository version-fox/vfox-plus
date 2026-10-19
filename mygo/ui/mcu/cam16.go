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

// The CAM16 colour appearance model of the Material colour utilities
// (Copyright 2021 Google LLC, Apache License 2.0), in the default viewing
// conditions, which is all the HCT colour space needs.

// viewingConditions are the conditions a colour is viewed in.
type viewingConditions struct {
	n, aw, nbb, ncb, c, nc float64
	rgbD                   [3]float64
	fl, flRoot, z          float64
}

// defaultViewingConditions are those of sRGB: the D65 white point, an
// adapting luminance of 11.72 cd/m², a background of L* 50 and an average
// surround.
var defaultViewingConditions = makeViewingConditions()

func makeViewingConditions() viewingConditions {
	adaptingLuminance := (200 / valuePi) * yFromLstar(50) / 100
	backgroundLstar, surround := 50.0, 2.0
	xyz := whitePointD65
	rW := xyz[0]*0.401288 + xyz[1]*0.650173 + xyz[2]*-0.051461
	gW := xyz[0]*-0.250268 + xyz[1]*1.204414 + xyz[2]*0.045854
	bW := xyz[0]*-0.002079 + xyz[1]*0.048952 + xyz[2]*0.953127
	f := 0.8 + surround/10
	var c float64
	if f >= 0.9 {
		c = lerp(0.59, 0.69, (f-0.9)*10)
	} else {
		c = lerp(0.525, 0.59, (f-0.8)*10)
	}
	d := f * (1 - (1/value36)*math.Exp((-adaptingLuminance-42)/92))
	d = clamp(1, d)
	nc := f
	rgbD := [3]float64{
		d*(100/rW) + 1 - d,
		d*(100/gW) + 1 - d,
		d*(100/bW) + 1 - d,
	}
	k := 1 / (5*adaptingLuminance + 1)
	k4 := k * k * k * k
	k4F := 1 - k4
	fl := k4*adaptingLuminance + 0.1*k4F*k4F*math.Cbrt(5*adaptingLuminance)
	n := yFromLstar(backgroundLstar) / whitePointD65[1]
	z := 1.48 + math.Sqrt(n)
	nbb := 0.725 / math.Pow(n, 0.2)
	ncb := nbb
	factors := [3]float64{
		math.Pow(fl*rgbD[0]*rW/100, 0.42),
		math.Pow(fl*rgbD[1]*gW/100, 0.42),
		math.Pow(fl*rgbD[2]*bW/100, 0.42),
	}
	rgbA := [3]float64{
		400 * factors[0] / (factors[0] + 27.13),
		400 * factors[1] / (factors[1] + 27.13),
		400 * factors[2] / (factors[2] + 27.13),
	}
	aw := (2*rgbA[0] + rgbA[1] + 0.05*rgbA[2]) * nbb
	return viewingConditions{
		n: n, aw: aw, nbb: nbb, ncb: ncb, c: c, nc: nc, rgbD: rgbD,
		fl: fl, flRoot: math.Pow(fl, 0.25), z: z,
	}
}

// cam16 is a colour in the CAM16 appearance model; only the hue and the
// chroma, which HCT keeps, are kept.
type cam16 struct {
	hue, chroma float64
}

// cam16FromARGB computes the CAM16 hue and chroma of a colour.
func cam16FromARGB(argb uint32) cam16 {
	vc := defaultViewingConditions
	redL := linearized(redOf(argb))
	greenL := linearized(greenOf(argb))
	blueL := linearized(blueOf(argb))
	x := 0.41233895*redL + 0.35762064*greenL + 0.18051042*blueL
	y := 0.2126*redL + 0.7152*greenL + 0.0722*blueL
	z := 0.01932141*redL + 0.11916382*greenL + 0.95034478*blueL

	rC := 0.401288*x + 0.650173*y - 0.051461*z
	gC := -0.250268*x + 1.204414*y + 0.045854*z
	bC := -0.002079*x + 0.048952*y + 0.953127*z

	rD := vc.rgbD[0] * rC
	gD := vc.rgbD[1] * gC
	bD := vc.rgbD[2] * bC

	rAF := math.Pow(vc.fl*math.Abs(rD)/100, 0.42)
	gAF := math.Pow(vc.fl*math.Abs(gD)/100, 0.42)
	bAF := math.Pow(vc.fl*math.Abs(bD)/100, 0.42)
	rA := signum(rD) * 400 * rAF / (rAF + 27.13)
	gA := signum(gD) * 400 * gAF / (gAF + 27.13)
	bA := signum(bD) * 400 * bAF / (bAF + 27.13)

	a := (11*rA + -12*gA + bA) / 11
	b := (rA + gA - 2*bA) / 9
	u := (20*rA + 20*gA + 21*bA) / 20
	p2 := (40*rA + 20*gA + bA) / 20

	hue := sanitizeDegrees(math.Atan2(b, a) * 180 / math.Pi)
	ac := p2 * vc.nbb
	j := 100 * math.Pow(ac/vc.aw, vc.c*vc.z)

	huePrime := hue
	if hue < 20.14 {
		huePrime = hue + 360
	}
	eHue := 0.25 * (math.Cos(huePrime*math.Pi/180+2) + 3.8)
	p1 := 50000.0 / 13.0 * eHue * vc.nc * vc.ncb
	t := p1 * math.Sqrt(a*a+b*b) / (u + 0.305)
	alpha := math.Pow(t, 0.9) * math.Pow(1.64-math.Pow(0.29, vc.n), 0.73)
	return cam16{hue: hue, chroma: alpha * math.Sqrt(j/100)}
}
