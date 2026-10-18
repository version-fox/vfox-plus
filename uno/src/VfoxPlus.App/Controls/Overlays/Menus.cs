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

namespace VfoxPlus.App.Controls.Overlays;

/// <summary>Buttons that open menus.</summary>
internal static class Menus
{
    /// <summary>
    /// Makes an icon button open a menu below it. The entries are made when the menu opens, so that they know what is
    /// true then.
    /// </summary>
    /// <param name="button">The button.</param>
    /// <param name="items">Makes the entries.</param>
    /// <param name="alignEnd">Whether the menu ends where the button ends, for buttons at the end of a row.</param>
    public static void Attach(M3IconButton button, Func<IReadOnlyList<MenuItem>> items, bool alignEnd = true) =>
        button.Click += (_, _) => OverlayHost.Current?.ShowMenu(button, items(), new MenuOptions(AlignEnd: alignEnd));
}
