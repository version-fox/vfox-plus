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
/// The colours of one hue and chroma at every tone, from 0 to 100.
/// </summary>
public sealed class TonalPalette
{
    private readonly Lock gate = new();
    private readonly Dictionary<double, uint> cache = [];

    /// <summary>Initializes a new instance of the <see cref="TonalPalette"/> class.</summary>
    /// <param name="hue">The hue in degrees.</param>
    /// <param name="chroma">The chroma requested for the colours; light and dark tones may have less.</param>
    public TonalPalette(double hue, double chroma)
    {
        Hue = hue;
        Chroma = chroma;
    }

    /// <summary>Gets the hue in degrees.</summary>
    public double Hue { get; }

    /// <summary>Gets the chroma requested for the colours.</summary>
    public double Chroma { get; }

    /// <summary>Returns the colour of the palette at a tone.</summary>
    /// <param name="tone">The tone, from 0 to 100.</param>
    /// <returns>The colour as ARGB.</returns>
    public uint Tone(double tone)
    {
        lock (gate)
        {
            if (cache.TryGetValue(tone, out uint cached))
            {
                return cached;
            }

            // Tone 99 of a yellow is not representable: use the average of the neighbours.
            uint argb = tone == 99 && Hct.IsYellow(Hue)
                ? Average(Solve(98), Solve(100))
                : Solve(tone);
            cache[tone] = argb;
            return argb;
        }
    }

    /// <summary>Returns the colour of the palette at a tone, in HCT.</summary>
    /// <param name="tone">The tone, from 0 to 100.</param>
    /// <returns>The colour.</returns>
    public Hct HctAt(double tone) => Hct.FromArgb(Tone(tone));

    private static uint Average(uint a, uint b)
    {
        int red = (int)ColorMath.JsRound((ColorMath.RedOf(a) + ColorMath.RedOf(b)) / 2.0);
        int green = (int)ColorMath.JsRound((ColorMath.GreenOf(a) + ColorMath.GreenOf(b)) / 2.0);
        int blue = (int)ColorMath.JsRound((ColorMath.BlueOf(a) + ColorMath.BlueOf(b)) / 2.0);
        return ColorMath.ArgbFromRgb(red, green, blue);
    }

    // Must be called with the lock held.
    private uint Solve(double tone)
    {
        if (!cache.TryGetValue(tone, out uint argb))
        {
            argb = HctSolver.SolveToArgb(Hue, Chroma, tone);
            cache[tone] = argb;
        }

        return argb;
    }
}
