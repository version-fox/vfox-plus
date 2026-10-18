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

using System.Numerics;
using VfoxPlus.App.Controls.Overlays;

namespace VfoxPlus.App.Theming;

/// <summary>
/// The shadows of Material 3: a surface at a higher level casts a larger, softer shadow.
/// </summary>
internal static class Elevation
{
    // The depth that Uno draws a shadow of, per level of Material.
    private const float DepthPerLevel = 5;

    /// <summary>Lifts an element to a level of elevation.</summary>
    /// <param name="element">The element.</param>
    /// <param name="level">The level, 1 to 5; Material 3 uses 1 for cards that rest, 3 for menus, dialogs and snack bars.</param>
    public static void Set(FrameworkElement element, double level)
    {
        ThemeShadow shadow = new();
        if (OverlayHost.Current is { } host)
        {
            shadow.Receivers.Add(host);
        }

        element.Shadow = shadow;
        element.Translation = new Vector3(0, 0, (float)(level * DepthPerLevel));
    }
}
