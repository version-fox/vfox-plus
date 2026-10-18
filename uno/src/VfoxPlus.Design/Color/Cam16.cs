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

namespace VfoxPlus.Design.Color;

/// <summary>
/// The CAM16 colour appearance model of the Material colour utilities (Copyright 2021 Google LLC, Apache
/// License 2.0), in the default viewing conditions, which is all the HCT colour space needs.
/// </summary>
internal static class Cam16
{
    /// <summary>
    /// The viewing conditions of sRGB: the D65 white point, an adapting luminance of 11.72 cd/m², a
    /// background of L* 50 and an average surround.
    /// </summary>
    internal static readonly ViewingConditions Default = MakeViewingConditions();

    /// <summary>Computes the CAM16 hue and chroma of a colour.</summary>
    /// <param name="argb">The colour.</param>
    /// <returns>The hue in degrees and the chroma.</returns>
    internal static (double Hue, double Chroma) FromArgb(uint argb)
    {
        ViewingConditions vc = Default;
        double redL = ColorMath.Linearized(ColorMath.RedOf(argb));
        double greenL = ColorMath.Linearized(ColorMath.GreenOf(argb));
        double blueL = ColorMath.Linearized(ColorMath.BlueOf(argb));
        double x = (0.41233895 * redL) + (0.35762064 * greenL) + (0.18051042 * blueL);
        double y = (0.2126 * redL) + (0.7152 * greenL) + (0.0722 * blueL);
        double z = (0.01932141 * redL) + (0.11916382 * greenL) + (0.95034478 * blueL);

        double rC = (0.401288 * x) + (0.650173 * y) - (0.051461 * z);
        double gC = (-0.250268 * x) + (1.204414 * y) + (0.045854 * z);
        double bC = (-0.002079 * x) + (0.048952 * y) + (0.953127 * z);

        double rD = vc.RgbD[0] * rC;
        double gD = vc.RgbD[1] * gC;
        double bD = vc.RgbD[2] * bC;

        double rAF = Math.Pow(vc.Fl * Math.Abs(rD) / 100, 0.42);
        double gAF = Math.Pow(vc.Fl * Math.Abs(gD) / 100, 0.42);
        double bAF = Math.Pow(vc.Fl * Math.Abs(bD) / 100, 0.42);
        double rA = ColorMath.Signum(rD) * 400 * rAF / (rAF + 27.13);
        double gA = ColorMath.Signum(gD) * 400 * gAF / (gAF + 27.13);
        double bA = ColorMath.Signum(bD) * 400 * bAF / (bAF + 27.13);

        double a = ((11 * rA) + (-12 * gA) + bA) / 11;
        double b = (rA + gA - (2 * bA)) / 9;
        double u = ((20 * rA) + (20 * gA) + (21 * bA)) / 20;
        double p2 = ((40 * rA) + (20 * gA) + bA) / 20;

        double hue = ColorMath.SanitizeDegrees(Math.Atan2(b, a) * 180 / Math.PI);
        double ac = p2 * vc.Nbb;
        double j = 100 * Math.Pow(ac / vc.Aw, vc.C * vc.Z);

        double huePrime = hue < 20.14 ? hue + 360 : hue;
        double eHue = 0.25 * (Math.Cos((huePrime * Math.PI / 180) + 2) + 3.8);
        double p1 = 50000.0 / 13.0 * eHue * vc.Nc * vc.Ncb;
        double t = p1 * Math.Sqrt((a * a) + (b * b)) / (u + 0.305);
        double alpha = Math.Pow(t, 0.9) * Math.Pow(1.64 - Math.Pow(0.29, vc.N), 0.73);
        return (hue, alpha * Math.Sqrt(j / 100));
    }

    private static ViewingConditions MakeViewingConditions()
    {
        double adaptingLuminance = 200 / Math.PI * ColorMath.YFromLstar(50) / 100;
        const double backgroundLstar = 50.0;
        const double surround = 2.0;
        double[] xyz = ColorMath.WhitePointD65;
        double rW = (xyz[0] * 0.401288) + (xyz[1] * 0.650173) + (xyz[2] * -0.051461);
        double gW = (xyz[0] * -0.250268) + (xyz[1] * 1.204414) + (xyz[2] * 0.045854);
        double bW = (xyz[0] * -0.002079) + (xyz[1] * 0.048952) + (xyz[2] * 0.953127);
        double f = 0.8 + (surround / 10);
        double c = f >= 0.9
            ? ColorMath.Lerp(0.59, 0.69, (f - 0.9) * 10)
            : ColorMath.Lerp(0.525, 0.59, (f - 0.8) * 10);
        double d = f * (1 - (1 / 3.6 * Math.Exp((-adaptingLuminance - 42) / 92)));
        d = ColorMath.Clamp(1, d);
        double nc = f;
        double[] rgbD =
        [
            (d * (100 / rW)) + 1 - d,
            (d * (100 / gW)) + 1 - d,
            (d * (100 / bW)) + 1 - d,
        ];
        double k = 1 / ((5 * adaptingLuminance) + 1);
        double k4 = k * k * k * k;
        double k4F = 1 - k4;
        double fl = (k4 * adaptingLuminance) + (0.1 * k4F * k4F * Math.Cbrt(5 * adaptingLuminance));
        double n = ColorMath.YFromLstar(backgroundLstar) / ColorMath.WhitePointD65[1];
        double z = 1.48 + Math.Sqrt(n);
        double nbb = 0.725 / Math.Pow(n, 0.2);
        double ncb = nbb;
        double[] factors =
        [
            Math.Pow(fl * rgbD[0] * rW / 100, 0.42),
            Math.Pow(fl * rgbD[1] * gW / 100, 0.42),
            Math.Pow(fl * rgbD[2] * bW / 100, 0.42),
        ];
        double[] rgbA =
        [
            400 * factors[0] / (factors[0] + 27.13),
            400 * factors[1] / (factors[1] + 27.13),
            400 * factors[2] / (factors[2] + 27.13),
        ];
        double aw = ((2 * rgbA[0]) + rgbA[1] + (0.05 * rgbA[2])) * nbb;
        return new ViewingConditions(n, aw, nbb, ncb, c, nc, rgbD, fl, Math.Pow(fl, 0.25), z);
    }

    /// <summary>The conditions a colour is viewed in.</summary>
    internal sealed record ViewingConditions(
        double N,
        double Aw,
        double Nbb,
        double Ncb,
        double C,
        double Nc,
        double[] RgbD,
        double Fl,
        double FlRoot,
        double Z);
}
