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
/// A colour in the HCT colour space of Material 3: the hue and the chroma of CAM16 and the tone, which is the
/// L* of CIE L*a*b*. It is perceptually uniform, and a tone of 50 or less always has a contrast ratio of 3 to
/// the tone of 100, whatever the hue and the chroma.
/// </summary>
/// <param name="Hue">The CAM16 hue in degrees, from 0 to 360.</param>
/// <param name="Chroma">The CAM16 chroma, roughly 0 to 120.</param>
/// <param name="Tone">The L* of CIE L*a*b*, from 0 (black) to 100 (white).</param>
/// <param name="Argb">The sRGB colour, <c>0xAARRGGBB</c> with an opaque alpha.</param>
public readonly record struct Hct(double Hue, double Chroma, double Tone, uint Argb)
{
    /// <summary>Converts an sRGB colour to HCT.</summary>
    /// <param name="argb">The colour as <c>0xAARRGGBB</c>.</param>
    /// <returns>The colour in HCT.</returns>
    public static Hct FromArgb(uint argb)
    {
        (double hue, double chroma) = Cam16.FromArgb(argb);
        return new Hct(hue, chroma, ColorMath.LstarFromArgb(argb), argb);
    }

    /// <summary>
    /// Finds the colour with the given hue, chroma and tone that sRGB can show; the chroma is reduced when the
    /// gamut does not reach it.
    /// </summary>
    /// <param name="hue">The hue in degrees.</param>
    /// <param name="chroma">The chroma.</param>
    /// <param name="tone">The tone, from 0 to 100.</param>
    /// <returns>The colour.</returns>
    public static Hct From(double hue, double chroma, double tone) =>
        FromArgb(HctSolver.SolveToArgb(hue, chroma, tone));

    /// <summary>Determines whether a hue is in the range where the tone 99 of a palette is not representable.</summary>
    /// <param name="hue">The hue in degrees.</param>
    /// <returns><see langword="true"/> for the yellows.</returns>
    internal static bool IsYellow(double hue) => hue is >= 105 and < 125;
}
