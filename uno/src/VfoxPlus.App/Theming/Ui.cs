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

using VfoxPlus.Design.Tokens;
using Windows.UI.Text;

namespace VfoxPlus.App.Theming;

/// <summary>
/// What every part of the user interface shares: the theme and the helpers that make text.
/// </summary>
internal static class Ui
{
    private static AppTheme? theme;

    /// <summary>Gets or sets the theme of the application.</summary>
    public static AppTheme Theme
    {
        get => theme ?? throw new InvalidOperationException("The theme is not set up yet.");
        set => theme = value;
    }

    /// <summary>Applies a style of the type scale to a text block.</summary>
    /// <param name="block">The text block.</param>
    /// <param name="style">The style.</param>
    /// <returns>The text block.</returns>
    public static TextBlock Style(this TextBlock block, TextStyle style)
    {
        block.FontFamily = Fonts.Sans;
        block.FontSize = style.Size;
        block.FontWeight = new FontWeight((ushort)style.Weight);
        block.LineHeight = style.Line;
        block.LineStackingStrategy = LineStackingStrategy.BlockLineHeight;
        return block;
    }

    /// <summary>Sets the text of a text block, and takes the block out of the layout when there is none.</summary>
    /// <param name="block">The text block.</param>
    /// <param name="text">The text.</param>
    public static void SetText(this TextBlock block, string text)
    {
        block.Text = text;
        block.Visibility = text.Length == 0 ? Visibility.Collapsed : Visibility.Visible;
    }

    /// <summary>Makes a text block with a style of the type scale.</summary>
    /// <param name="text">The text.</param>
    /// <param name="style">The style.</param>
    /// <param name="foreground">The colour; the colour of text on a surface by default.</param>
    /// <returns>The text block.</returns>
    public static TextBlock Text(string text, TextStyle style, Brush? foreground = null) => new TextBlock
    {
        Text = text,
        Foreground = foreground ?? Theme.OnSurface,
    }.Style(style);

    /// <summary>Makes a text block with a role of the type scale.</summary>
    /// <param name="text">The text.</param>
    /// <param name="role">The role.</param>
    /// <param name="foreground">The colour; the colour of text on a surface by default.</param>
    /// <param name="emphasized">Whether to use the emphasized style of the role.</param>
    /// <returns>The text block.</returns>
    public static TextBlock Text(string text, TextRole role, Brush? foreground = null, bool emphasized = false) =>
        Text(text, emphasized ? Typography.Emphasized(role) : Typography.Of(role), foreground);
}
