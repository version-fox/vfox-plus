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

using VfoxPlus.App.Motion;
using VfoxPlus.App.Theming;
using VfoxPlus.Design.Motion;

namespace VfoxPlus.App.Controls;

/// <summary>
/// An icon of Material Symbols Rounded. It is drawn as an outline, or filled: the two are different fonts, so that
/// the fill, which animates, is a cross-fade between them.
/// </summary>
internal sealed class M3Icon : Grid
{
    private readonly TextBlock outlined = Glyph(Fonts.Outlined);
    private TextBlock? filled;
    private Animated? fillAnimation;
    private double size;
    private string symbol = string.Empty;

    public M3Icon(string symbol, double size = 24, Brush? foreground = null)
    {
        this.size = size;
        Children.Add(outlined);
        outlined.Foreground = foreground ?? Ui.Theme.OnSurfaceVariant;
        IsHitTestVisible = false;
        Symbol = symbol;
        Size = size;
    }

    /// <summary>Gets or sets what the icon means, as in <c>shared/icons.json</c>.</summary>
    public string Symbol
    {
        get => symbol;
        set
        {
            symbol = value;
            outlined.Text = Icons.Glyph(value);
            if (filled is not null)
            {
                filled.Text = outlined.Text;
            }
        }
    }

    /// <summary>Gets or sets the side of the icon in dp.</summary>
    public double Size
    {
        get => size;
        set
        {
            size = value;
            Width = value;
            Height = value;
            Resize(outlined);
            if (filled is not null)
            {
                Resize(filled);
            }
        }
    }

    /// <summary>Gets or sets the colour of the icon.</summary>
    public Brush Foreground
    {
        get => outlined.Foreground;
        set
        {
            outlined.Foreground = value;
            if (filled is not null)
            {
                filled.Foreground = value;
            }
        }
    }

    /// <summary>Gets the fill of the icon, from 0 (an outline) to 1 (filled).</summary>
    public double Fill => fillAnimation?.Value ?? 0;

    /// <summary>Fills the icon, or empties it, on the fast effects spring.</summary>
    /// <param name="target">The fill, from 0 to 1.</param>
    /// <param name="animate">Whether to animate the change.</param>
    public void FillTo(double target, bool animate = true)
    {
        if (filled is null)
        {
            if (target == 0)
            {
                return;
            }

            filled = Glyph(Fonts.Filled);
            filled.Text = outlined.Text;
            filled.Foreground = outlined.Foreground;
            filled.Opacity = 0;
            Resize(filled);
            Children.Add(filled);
            fillAnimation = new Animated(MotionScheme.Expressive.FastEffects, ApplyFill);
        }

        if (animate)
        {
            fillAnimation!.To(target);
        }
        else
        {
            fillAnimation!.Snap(target);
        }
    }

    private static TextBlock Glyph(FontFamily family) => new()
    {
        FontFamily = family,
        HorizontalAlignment = HorizontalAlignment.Center,
        VerticalAlignment = VerticalAlignment.Center,
        IsHitTestVisible = false,
    };

    // The line of the icon font is 1.2 em tall, with the em square in its middle, so the glyph is centred by
    // centring the line.
    private void Resize(TextBlock glyph) => glyph.FontSize = size;

    private void ApplyFill(double value)
    {
        filled!.Opacity = value;
        outlined.Opacity = 1 - value;
    }
}
