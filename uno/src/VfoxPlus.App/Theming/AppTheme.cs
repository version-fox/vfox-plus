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

using VfoxPlus.Design.Color;
using Windows.UI;

namespace VfoxPlus.App.Theming;

/// <summary>
/// The colour scheme as brushes: one per role, created once. When the scheme changes the brushes change colour, and
/// everything painted with them follows, which is how the theme, the seed colour and the accent colour of the desktop
/// apply without a rebuild of the user interface.
/// </summary>
internal sealed class AppTheme
{
    private readonly List<(Func<ColorScheme, uint> Pick, double Alpha, SolidColorBrush Brush)> brushes = [];

    public AppTheme(uint seed, bool dark)
    {
        Scheme = ColorScheme.Generate(seed, dark);
        Dark = dark;

        Surface = Role(s => s.Surface);
        SurfaceContainerLowest = Role(s => s.SurfaceContainerLowest);
        SurfaceContainerLow = Role(s => s.SurfaceContainerLow);
        SurfaceContainer = Role(s => s.SurfaceContainer);
        SurfaceContainerHigh = Role(s => s.SurfaceContainerHigh);
        SurfaceContainerHighest = Role(s => s.SurfaceContainerHighest);
        OnSurface = Role(s => s.OnSurface);
        OnSurfaceVariant = Role(s => s.OnSurfaceVariant);
        Outline = Role(s => s.Outline);
        OutlineVariant = Role(s => s.OutlineVariant);
        InverseSurface = Role(s => s.InverseSurface);
        InverseOnSurface = Role(s => s.InverseOnSurface);
        InversePrimary = Role(s => s.InversePrimary);
        Primary = Role(s => s.Primary);
        OnPrimary = Role(s => s.OnPrimary);
        PrimaryContainer = Role(s => s.PrimaryContainer);
        OnPrimaryContainer = Role(s => s.OnPrimaryContainer);
        PrimaryFixedDim = Role(s => s.PrimaryFixedDim);
        OnPrimaryFixed = Role(s => s.OnPrimaryFixed);
        Secondary = Role(s => s.Secondary);
        OnSecondary = Role(s => s.OnSecondary);
        SecondaryContainer = Role(s => s.SecondaryContainer);
        OnSecondaryContainer = Role(s => s.OnSecondaryContainer);
        Tertiary = Role(s => s.Tertiary);
        OnTertiary = Role(s => s.OnTertiary);
        TertiaryContainer = Role(s => s.TertiaryContainer);
        OnTertiaryContainer = Role(s => s.OnTertiaryContainer);
        Error = Role(s => s.Error);
        OnError = Role(s => s.OnError);
        ErrorContainer = Role(s => s.ErrorContainer);
        OnErrorContainer = Role(s => s.OnErrorContainer);
        Scrim = Role(s => s.Scrim);
        Shadow = Role(s => s.Shadow);

        // The colours of states: what is painted on a disabled control.
        OnSurface12 = Role(s => s.OnSurface, 0.12);
        OnSurface38 = Role(s => s.OnSurface, 0.38);
        OnSurface04 = Role(s => s.OnSurface, 0.04);
        Scrim32 = Role(s => s.Scrim, 0.32);
        Transparent = new SolidColorBrush(Microsoft.UI.Colors.Transparent);
    }

    /// <summary>Gets the current colour scheme.</summary>
    public ColorScheme Scheme { get; private set; }

    /// <summary>Gets a value indicating whether the scheme is the dark one.</summary>
    public bool Dark { get; private set; }

    public SolidColorBrush Surface { get; }

    public SolidColorBrush SurfaceContainerLowest { get; }

    public SolidColorBrush SurfaceContainerLow { get; }

    public SolidColorBrush SurfaceContainer { get; }

    public SolidColorBrush SurfaceContainerHigh { get; }

    public SolidColorBrush SurfaceContainerHighest { get; }

    public SolidColorBrush OnSurface { get; }

    public SolidColorBrush OnSurfaceVariant { get; }

    public SolidColorBrush Outline { get; }

    public SolidColorBrush OutlineVariant { get; }

    public SolidColorBrush InverseSurface { get; }

    public SolidColorBrush InverseOnSurface { get; }

    public SolidColorBrush InversePrimary { get; }

    public SolidColorBrush Primary { get; }

    public SolidColorBrush OnPrimary { get; }

    public SolidColorBrush PrimaryContainer { get; }

    public SolidColorBrush OnPrimaryContainer { get; }

    public SolidColorBrush PrimaryFixedDim { get; }

    public SolidColorBrush OnPrimaryFixed { get; }

    public SolidColorBrush Secondary { get; }

    public SolidColorBrush OnSecondary { get; }

    public SolidColorBrush SecondaryContainer { get; }

    public SolidColorBrush OnSecondaryContainer { get; }

    public SolidColorBrush Tertiary { get; }

    public SolidColorBrush OnTertiary { get; }

    public SolidColorBrush TertiaryContainer { get; }

    public SolidColorBrush OnTertiaryContainer { get; }

    public SolidColorBrush Error { get; }

    public SolidColorBrush OnError { get; }

    public SolidColorBrush ErrorContainer { get; }

    public SolidColorBrush OnErrorContainer { get; }

    public SolidColorBrush Scrim { get; }

    public SolidColorBrush Shadow { get; }

    /// <summary>Gets the colour of the container of a disabled control.</summary>
    public SolidColorBrush OnSurface12 { get; }

    /// <summary>Gets the colour of the container of a disabled text field.</summary>
    public SolidColorBrush OnSurface04 { get; }

    /// <summary>Gets the colour of the content of a disabled control.</summary>
    public SolidColorBrush OnSurface38 { get; }

    /// <summary>Gets the colour behind a dialog.</summary>
    public SolidColorBrush Scrim32 { get; }

    /// <summary>Gets a brush that paints nothing but, unlike no brush, can be hit.</summary>
    public SolidColorBrush Transparent { get; }

    /// <summary>Converts a colour of the colour utilities.</summary>
    /// <param name="argb">The colour as 0xAARRGGBB.</param>
    /// <param name="alpha">The opacity to multiply the colour with.</param>
    /// <returns>The colour.</returns>
    public static Color ToColor(uint argb, double alpha = 1) => Color.FromArgb(
        (byte)Math.Round(((argb >> 24) & 0xFF) * alpha),
        (byte)((argb >> 16) & 0xFF),
        (byte)((argb >> 8) & 0xFF),
        (byte)(argb & 0xFF));

    /// <summary>Makes the brushes paint another colour scheme.</summary>
    /// <param name="seed">The seed colour.</param>
    /// <param name="dark">Whether the scheme is the dark one.</param>
    public void Apply(uint seed, bool dark)
    {
        Scheme = ColorScheme.Generate(seed, dark);
        Dark = dark;
        foreach ((Func<ColorScheme, uint> pick, double alpha, SolidColorBrush brush) in brushes)
        {
            brush.Color = ToColor(pick(Scheme), alpha);
        }
    }

    private SolidColorBrush Role(Func<ColorScheme, uint> pick, double alpha = 1)
    {
        SolidColorBrush brush = new(ToColor(pick(Scheme), alpha));
        brushes.Add((pick, alpha, brush));
        return brush;
    }
}
