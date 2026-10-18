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
/// Contrast ratios between tones (Copyright 2021 Google LLC, Apache License 2.0), as WCAG defines them on
/// the relative luminance.
/// </summary>
internal static class Contrast
{
    /// <summary>Returns the contrast ratio, 1 to 21, of two tones.</summary>
    internal static double RatioOfTones(double toneA, double toneB) =>
        RatioOfYs(ColorMath.YFromLstar(ColorMath.Clamp(100, toneA)), ColorMath.YFromLstar(ColorMath.Clamp(100, toneB)));

    internal static double RatioOfYs(double y1, double y2)
    {
        double lighter = Math.Max(y1, y2);
        double darker = Math.Min(y1, y2);
        return (lighter + 5) / (darker + 5);
    }

    /// <summary>Returns the lightest tone that has the contrast ratio with a tone, or -1 if there is none.</summary>
    internal static double LighterTone(double tone, double ratio)
    {
        if (tone is < 0 or > 100)
        {
            return -1;
        }

        double darkY = ColorMath.YFromLstar(tone);
        double lightY = (ratio * (darkY + 5)) - 5;
        double realContrast = RatioOfYs(lightY, darkY);
        double delta = Math.Abs(realContrast - ratio);
        if (realContrast < ratio && delta > 0.04)
        {
            return -1;
        }

        // Ensure the contrast is met after the conversion to 8 bits.
        double result = ColorMath.LstarFromY(lightY) + 0.4;
        return result is < 0 or > 100 ? -1 : result;
    }

    /// <summary>Returns the darkest tone that has the contrast ratio with a tone, or -1 if there is none.</summary>
    internal static double DarkerTone(double tone, double ratio)
    {
        if (tone is < 0 or > 100)
        {
            return -1;
        }

        double lightY = ColorMath.YFromLstar(tone);
        double darkY = ((lightY + 5) / ratio) - 5;
        double realContrast = RatioOfYs(lightY, darkY);
        double delta = Math.Abs(realContrast - ratio);
        if (realContrast < ratio && delta > 0.04)
        {
            return -1;
        }

        double result = ColorMath.LstarFromY(darkY) - 0.4;
        return result is < 0 or > 100 ? -1 : result;
    }

    /// <summary>Like <see cref="LighterTone"/>, but 100 instead of -1.</summary>
    internal static double LighterToneUnsafe(double tone, double ratio)
    {
        double lighter = LighterTone(tone, ratio);
        return lighter >= 0 ? lighter : 100;
    }

    /// <summary>Like <see cref="DarkerTone"/>, but 0 instead of -1.</summary>
    internal static double DarkerToneUnsafe(double tone, double ratio)
    {
        double darker = DarkerTone(tone, ratio);
        return darker >= 0 ? darker : 0;
    }
}
