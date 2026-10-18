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
/// Colour conversions and maths of the Material colour utilities (Copyright 2021 Google LLC, Apache License
/// 2.0). Colours are ARGB values in a <see cref="uint"/>: <c>0xAARRGGBB</c>.
/// </summary>
internal static class ColorMath
{
    private static readonly double[][] SrgbToXyz =
    [
        [0.41233895, 0.35762064, 0.18051042],
        [0.2126, 0.7152, 0.0722],
        [0.01932141, 0.11916382, 0.95034478],
    ];

    /// <summary>Gets the D65 white point.</summary>
    internal static readonly double[] WhitePointD65 = [95.047, 100.0, 108.883];

    internal static double Signum(double value) => value < 0 ? -1 : value == 0 ? 0 : 1;

    internal static double Lerp(double start, double stop, double amount) => ((1 - amount) * start) + (amount * stop);

    /// <summary>Limits a value to the range from 0 to <paramref name="high"/>.</summary>
    internal static double Clamp(double high, double value) => Math.Min(Math.Max(value, 0), high);

    /// <summary>Rounds like JavaScript's <c>Math.round</c>, half towards positive infinity, which the reference implementation relies on.</summary>
    internal static double JsRound(double value) => Math.Floor(value + 0.5);

    /// <summary>Brings an angle into [0, 360).</summary>
    internal static double SanitizeDegrees(double degrees)
    {
        degrees %= 360;
        return degrees < 0 ? degrees + 360 : degrees;
    }

    internal static double[] MatrixMultiply(double[] row, double[][] matrix) =>
    [
        (row[0] * matrix[0][0]) + (row[1] * matrix[0][1]) + (row[2] * matrix[0][2]),
        (row[0] * matrix[1][0]) + (row[1] * matrix[1][1]) + (row[2] * matrix[1][2]),
        (row[0] * matrix[2][0]) + (row[1] * matrix[2][1]) + (row[2] * matrix[2][2]),
    ];

    internal static uint ArgbFromRgb(int red, int green, int blue) =>
        0xFF000000 | ((uint)(red & 255) << 16) | ((uint)(green & 255) << 8) | (uint)(blue & 255);

    internal static uint ArgbFromLinRgb(double[] linrgb) =>
        ArgbFromRgb(Delinearized(linrgb[0]), Delinearized(linrgb[1]), Delinearized(linrgb[2]));

    internal static int RedOf(uint argb) => (int)((argb >> 16) & 255);

    internal static int GreenOf(uint argb) => (int)((argb >> 8) & 255);

    internal static int BlueOf(uint argb) => (int)(argb & 255);

    internal static uint ArgbFromLstar(double lstar)
    {
        int component = Delinearized(YFromLstar(lstar));
        return ArgbFromRgb(component, component, component);
    }

    internal static double LstarFromArgb(uint argb)
    {
        double y = MatrixMultiply([Linearized(RedOf(argb)), Linearized(GreenOf(argb)), Linearized(BlueOf(argb))], SrgbToXyz)[1];
        return (116 * LabF(y / 100)) - 16;
    }

    internal static double YFromLstar(double lstar) => 100 * LabInvf((lstar + 16) / 116);

    internal static double LstarFromY(double y) => (LabF(y / 100) * 116) - 16;

    /// <summary>Converts a 0 to 255 channel value to linear light, 0 to 100.</summary>
    internal static double Linearized(int component)
    {
        double normalized = component / 255.0;
        return normalized <= 0.040449936
            ? normalized / 12.92 * 100
            : Math.Pow((normalized + 0.055) / 1.055, 2.4) * 100;
    }

    /// <summary>Converts linear light, 0 to 100, to a 0 to 255 channel value.</summary>
    internal static int Delinearized(double component)
    {
        double normalized = component / 100;
        double result = normalized <= 0.0031308
            ? normalized * 12.92
            : (1.055 * Math.Pow(normalized, 1.0 / 2.4)) - 0.055;
        return (int)Clamp(255, JsRound(result * 255));
    }

    internal static double TrueDelinearized(double component)
    {
        double normalized = component / 100;
        double result = normalized <= 0.0031308
            ? normalized * 12.92
            : (1.055 * Math.Pow(normalized, 1.0 / 2.4)) - 0.055;
        return result * 255;
    }

    private static double LabF(double t)
    {
        const double e = 216.0 / 24389.0;
        const double kappa = 24389.0 / 27.0;
        return t > e ? Math.Pow(t, 1.0 / 3.0) : ((kappa * t) + 16) / 116;
    }

    private static double LabInvf(double ft)
    {
        const double e = 216.0 / 24389.0;
        const double kappa = 24389.0 / 27.0;
        double ft3 = ft * ft * ft;
        return ft3 > e ? ft3 : ((116 * ft) - 16) / kappa;
    }
}
