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

namespace VfoxPlus.Design.Platform;

/// <summary>What the XDG desktop portal reports about the appearance of the desktop, as the values that the application uses.</summary>
public static class PortalValues
{
    /// <summary>
    /// Converts the accent colour of the portal, a triple of components from 0 to 1, to ARGB. The portal reports
    /// components outside that range when no accent colour is set.
    /// </summary>
    /// <param name="components">The red, green and blue components.</param>
    /// <returns>The colour as 0xAARRGGBB, or <see langword="null"/> when the desktop has no accent colour.</returns>
    public static uint? ParseAccent(IReadOnlyList<double> components)
    {
        if (components.Count != 3)
        {
            return null;
        }

        uint argb = 0xFF000000;
        for (int i = 0; i < 3; i++)
        {
            double component = components[i];
            if (double.IsNaN(component) || component < 0 || component > 1)
            {
                return null;
            }

            argb |= (uint)Math.Round(component * 255) << (16 - (8 * i));
        }

        return argb;
    }

    /// <summary>Tells whether the desktop prefers a dark appearance, from its <c>color-scheme</c>.</summary>
    /// <param name="scheme">1 prefers dark, 2 prefers light, and 0 has no preference.</param>
    /// <returns><see langword="true"/> for dark, <see langword="false"/> for light, and <see langword="null"/> for no preference.</returns>
    public static bool? ParseColorScheme(uint scheme) => scheme switch
    {
        1 => true,
        2 => false,
        _ => null,
    };
}
