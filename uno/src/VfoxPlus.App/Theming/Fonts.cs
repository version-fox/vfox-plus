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

using SkiaSharp;

namespace VfoxPlus.App.Theming;

/// <summary>
/// The fonts of the user interface: the sans-serif font of the desktop, which is Ubuntu Sans on Ubuntu, and the icon
/// font. Uno takes the system fonts for Chinese.
/// </summary>
internal static class Fonts
{
    private const string OutlinedIcons = "ms-appx:///Assets/Fonts/MaterialSymbolsRounded-Outlined.ttf#Material Symbols Rounded";
    private const string FilledIcons = "ms-appx:///Assets/Fonts/MaterialSymbolsRounded-Filled.ttf#Material Symbols Rounded";

    // The families to try, in order. Uno does not read a list of families in one name.
    private static readonly string[] SansFamilies = ["Ubuntu Sans", "Ubuntu", "Noto Sans", "DejaVu Sans", "Liberation Sans"];

    /// <summary>Gets the font of the text.</summary>
    public static FontFamily Sans { get; } = FindSans();

    /// <summary>Gets the font of the icons at rest, drawn as outlines.</summary>
    public static FontFamily Outlined { get; } = new(OutlinedIcons);

    /// <summary>Gets the font of the icons that are selected, drawn filled.</summary>
    public static FontFamily Filled { get; } = new(FilledIcons);

    /// <summary>Gets the font of the code: the monospaced font of the desktop.</summary>
    public static FontFamily Mono { get; } = FindFamily(["Ubuntu Sans Mono", "Ubuntu Mono", "Noto Sans Mono", "DejaVu Sans Mono", "Liberation Mono"]) ?? new FontFamily("monospace");

    private static FontFamily FindSans() => FindFamily(SansFamilies) ?? FontFamily.XamlAutoFontFamily;

    private static FontFamily? FindFamily(string[] candidates)
    {
        HashSet<string> installed = [.. SKFontManager.Default.GetFontFamilies()];
        string? found = candidates.FirstOrDefault(installed.Contains);
        return found is null ? null : new FontFamily(found);
    }
}
