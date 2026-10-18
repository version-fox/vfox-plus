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
/// The dynamic colours of the 2021 specification of the Material colour utilities (Copyright 2021 Google LLC,
/// Apache License 2.0), for the tonal spot scheme: every colour role is a tone of a palette, nudged until it
/// has the contrast to its background that the contrast curve asks for.
/// </summary>
internal static class DynamicRoles
{
    private static readonly Roles All = new();

    /// <summary>Generates the ARGB colour of every role of the tonal spot scheme of a seed colour.</summary>
    /// <param name="seed">The seed colour as <c>0xAARRGGBB</c>.</param>
    /// <param name="dark">Whether the scheme is for a dark theme.</param>
    /// <returns>The scheme, at the standard contrast.</returns>
    internal static ColorScheme Generate(uint seed, bool dark)
    {
        Scheme scheme = new(Hct.FromArgb(seed), dark, 0);
        Roles r = All;
        uint Argb(ColorRole role) => role.HctIn(scheme).Argb;
        return new ColorScheme
        {
            Background = Argb(r.Background),
            OnBackground = Argb(r.OnBackground),
            Surface = Argb(r.Surface),
            SurfaceDim = Argb(r.SurfaceDim),
            SurfaceBright = Argb(r.SurfaceBright),
            SurfaceContainerLowest = Argb(r.SurfaceContainerLowest),
            SurfaceContainerLow = Argb(r.SurfaceContainerLow),
            SurfaceContainer = Argb(r.SurfaceContainer),
            SurfaceContainerHigh = Argb(r.SurfaceContainerHigh),
            SurfaceContainerHighest = Argb(r.SurfaceContainerHighest),
            OnSurface = Argb(r.OnSurface),
            OnSurfaceVariant = Argb(r.OnSurfaceVariant),
            Outline = Argb(r.Outline),
            OutlineVariant = Argb(r.OutlineVariant),
            InverseSurface = Argb(r.InverseSurface),
            InverseOnSurface = Argb(r.InverseOnSurface),
            InversePrimary = Argb(r.InversePrimary),
            SurfaceVariant = Argb(r.SurfaceVariant),
            Shadow = Argb(r.Shadow),
            Scrim = Argb(r.Scrim),
            Primary = Argb(r.Primary),
            OnPrimary = Argb(r.OnPrimary),
            PrimaryContainer = Argb(r.PrimaryContainer),
            OnPrimaryContainer = Argb(r.OnPrimaryContainer),
            PrimaryFixed = Argb(r.PrimaryFixed),
            PrimaryFixedDim = Argb(r.PrimaryFixedDim),
            OnPrimaryFixed = Argb(r.OnPrimaryFixed),
            OnPrimaryFixedVariant = Argb(r.OnPrimaryFixedVariant),
            Secondary = Argb(r.Secondary),
            OnSecondary = Argb(r.OnSecondary),
            SecondaryContainer = Argb(r.SecondaryContainer),
            OnSecondaryContainer = Argb(r.OnSecondaryContainer),
            SecondaryFixed = Argb(r.SecondaryFixed),
            SecondaryFixedDim = Argb(r.SecondaryFixedDim),
            OnSecondaryFixed = Argb(r.OnSecondaryFixed),
            OnSecondaryFixedVariant = Argb(r.OnSecondaryFixedVariant),
            Tertiary = Argb(r.Tertiary),
            OnTertiary = Argb(r.OnTertiary),
            TertiaryContainer = Argb(r.TertiaryContainer),
            OnTertiaryContainer = Argb(r.OnTertiaryContainer),
            TertiaryFixed = Argb(r.TertiaryFixed),
            TertiaryFixedDim = Argb(r.TertiaryFixedDim),
            OnTertiaryFixed = Argb(r.OnTertiaryFixed),
            OnTertiaryFixedVariant = Argb(r.OnTertiaryFixedVariant),
            Error = Argb(r.Error),
            OnError = Argb(r.OnError),
            ErrorContainer = Argb(r.ErrorContainer),
            OnErrorContainer = Argb(r.OnErrorContainer),
        };
    }

    internal static double ForegroundTone(double bgTone, double ratio)
    {
        double lighter = Contrast.LighterToneUnsafe(bgTone, ratio);
        double darker = Contrast.DarkerToneUnsafe(bgTone, ratio);
        double lighterRatio = Contrast.RatioOfTones(lighter, bgTone);
        double darkerRatio = Contrast.RatioOfTones(darker, bgTone);
        if (TonePrefersLightForeground(bgTone))
        {
            bool negligibleDifference = Math.Abs(lighterRatio - darkerRatio) < 0.1 && lighterRatio < ratio && darkerRatio < ratio;
            return lighterRatio >= ratio || lighterRatio >= darkerRatio || negligibleDifference ? lighter : darker;
        }

        return darkerRatio >= ratio || darkerRatio >= lighterRatio ? darker : lighter;
    }

    internal static bool TonePrefersLightForeground(double tone) => ColorMath.JsRound(tone) < 60;

    /// <summary>The contrast ratio that a colour role needs, by contrast level: low (-1), normal (0), medium (0.5) and high (1).</summary>
    internal sealed record ContrastCurve(double Low, double Normal, double Medium, double High)
    {
        internal double Get(double level) => level switch
        {
            <= -1 => Low,
            < 0 => ColorMath.Lerp(Low, Normal, level - -1),
            < 0.5 => ColorMath.Lerp(Normal, Medium, (level - 0) / 0.5),
            < 1 => ColorMath.Lerp(Medium, High, (level - 0.5) / 0.5),
            _ => High,
        };
    }

    /// <summary>Which of the two roles of a tone delta pair is lighter.</summary>
    internal enum Polarity
    {
        Nearer,
        Lighter,
    }

    /// <summary>Asks for two roles to differ by at least a delta in tone.</summary>
    internal sealed record ToneDeltaPair(ColorRole RoleA, ColorRole RoleB, double Delta, Polarity Polarity, bool StayTogether);

    /// <summary>A tonal spot scheme: the palettes of a seed colour, light or dark, at a contrast level.</summary>
    internal sealed class Scheme
    {
        internal Scheme(Hct seed, bool dark, double contrastLevel)
        {
            Dark = dark;
            ContrastLevel = contrastLevel;
            Primary = new TonalPalette(seed.Hue, 36);
            Secondary = new TonalPalette(seed.Hue, 16);
            Tertiary = new TonalPalette(ColorMath.SanitizeDegrees(seed.Hue + 60), 24);
            Neutral = new TonalPalette(seed.Hue, 6);
            NeutralVariant = new TonalPalette(seed.Hue, 8);
            Error = new TonalPalette(25, 84);
        }

        internal bool Dark { get; }

        internal double ContrastLevel { get; }

        internal TonalPalette Primary { get; }

        internal TonalPalette Secondary { get; }

        internal TonalPalette Tertiary { get; }

        internal TonalPalette Neutral { get; }

        internal TonalPalette NeutralVariant { get; }

        internal TonalPalette Error { get; }
    }

    /// <summary>One colour role, such as <c>primary</c> or <c>onSurface</c>.</summary>
    internal sealed class ColorRole
    {
        internal required string Name { get; init; }

        internal required Func<Scheme, TonalPalette> Palette { get; init; }

        internal required Func<Scheme, double> ToneOf { get; init; }

        internal bool IsBackground { get; init; }

        internal Func<Scheme, ColorRole>? Background { get; init; }

        internal Func<Scheme, ColorRole>? SecondBackground { get; init; }

        internal ContrastCurve? Contrast { get; init; }

        internal Func<Scheme, ToneDeltaPair>? Pair { get; init; }

        internal Hct HctIn(Scheme scheme) => Palette(scheme).HctAt(GetTone(scheme));

        // The tone of the role in the scheme, after the adjustments for the contrast to its background and for
        // its tone delta pair.
        internal double GetTone(Scheme s)
        {
            bool decreasingContrast = s.ContrastLevel < 0;
            if (Pair is not null)
            {
                return ToneOfPair(s, Pair(s), decreasingContrast);
            }

            double answer = ToneOf(s);
            if (Background is null || Contrast is null)
            {
                // No adjustment for colours without a background.
                return answer;
            }

            double bgTone = Background(s).GetTone(s);
            double desiredRatio = Contrast.Get(s.ContrastLevel);
            if (Design.Color.Contrast.RatioOfTones(bgTone, answer) < desiredRatio)
            {
                answer = ForegroundTone(bgTone, desiredRatio);
            }

            if (decreasingContrast)
            {
                answer = ForegroundTone(bgTone, desiredRatio);
            }

            // Tones between 50 and 60 have a poor contrast to both black and white text: background colours skip
            // them.
            if (IsBackground && answer is >= 50 and < 60)
            {
                answer = Design.Color.Contrast.RatioOfTones(49, bgTone) >= desiredRatio ? 49 : 60;
            }

            if (SecondBackground is null)
            {
                return answer;
            }

            double bgTone1 = Background(s).GetTone(s);
            double bgTone2 = SecondBackground(s).GetTone(s);
            double upper = Math.Max(bgTone1, bgTone2);
            double lower = Math.Min(bgTone1, bgTone2);
            if (Design.Color.Contrast.RatioOfTones(upper, answer) >= desiredRatio && Design.Color.Contrast.RatioOfTones(lower, answer) >= desiredRatio)
            {
                return answer;
            }

            // Neither can be met: choose the tone that contrasts with both.
            double lightOption = Design.Color.Contrast.LighterTone(upper, desiredRatio);
            double darkOption = Design.Color.Contrast.DarkerTone(lower, desiredRatio);
            int available = (lightOption != -1 ? 1 : 0) + (darkOption != -1 ? 1 : 0);
            if (TonePrefersLightForeground(bgTone1) || TonePrefersLightForeground(bgTone2))
            {
                return lightOption < 0 ? 100 : lightOption;
            }

            if (available == 1)
            {
                return lightOption != -1 ? lightOption : darkOption;
            }

            return darkOption < 0 ? 0 : darkOption;
        }

        private double ToneOfPair(Scheme s, ToneDeltaPair pair, bool decreasingContrast)
        {
            bool aIsNearer = pair.Polarity == Polarity.Nearer || (pair.Polarity == Polarity.Lighter && !s.Dark);
            ColorRole nearer = aIsNearer ? pair.RoleA : pair.RoleB;
            ColorRole farther = aIsNearer ? pair.RoleB : pair.RoleA;
            bool amNearer = Name == nearer.Name;
            double expansionDir = s.Dark ? 1 : -1.0;
            double nTone = nearer.ToneOf(s);
            double fTone = farther.ToneOf(s);
            if (Background is not null && nearer.Contrast is not null && farther.Contrast is not null)
            {
                double bgTone = Background(s).GetTone(s);
                double nContrast = nearer.Contrast.Get(s.ContrastLevel);
                double fContrast = farther.Contrast.Get(s.ContrastLevel);
                if (Design.Color.Contrast.RatioOfTones(bgTone, nTone) < nContrast)
                {
                    nTone = ForegroundTone(bgTone, nContrast);
                }

                if (Design.Color.Contrast.RatioOfTones(bgTone, fTone) < fContrast)
                {
                    fTone = ForegroundTone(bgTone, fContrast);
                }

                if (decreasingContrast)
                {
                    nTone = ForegroundTone(bgTone, nContrast);
                    fTone = ForegroundTone(bgTone, fContrast);
                }
            }

            // Expand the farther role away from the nearer one until the delta holds.
            if ((fTone - nTone) * expansionDir < pair.Delta)
            {
                fTone = ColorMath.Clamp(100, nTone + (pair.Delta * expansionDir));
                if ((fTone - nTone) * expansionDir < pair.Delta)
                {
                    nTone = ColorMath.Clamp(100, fTone - (pair.Delta * expansionDir));
                }
            }

            // Avoid the tones between 50 and 60.
            if (nTone is >= 50 and < 60)
            {
                if (expansionDir > 0)
                {
                    nTone = 60;
                    fTone = Math.Max(fTone, nTone + (pair.Delta * expansionDir));
                }
                else
                {
                    nTone = 49;
                    fTone = Math.Min(fTone, nTone + (pair.Delta * expansionDir));
                }
            }
            else if (fTone is >= 50 and < 60)
            {
                if (pair.StayTogether && expansionDir > 0)
                {
                    nTone = 60;
                    fTone = Math.Max(fTone, nTone + (pair.Delta * expansionDir));
                }
                else if (pair.StayTogether)
                {
                    nTone = 49;
                    fTone = Math.Min(fTone, nTone + (pair.Delta * expansionDir));
                }
                else
                {
                    fTone = expansionDir > 0 ? 60 : 49;
                }
            }

            return amNearer ? nTone : fTone;
        }
    }

    /// <summary>All colour roles of the 2021 specification that the tonal spot scheme uses. They refer to each other, so they are built together.</summary>
    internal sealed class Roles
    {
        internal Roles()
        {
            Func<Scheme, TonalPalette> neutral = s => s.Neutral;
            Func<Scheme, TonalPalette> neutralVariant = s => s.NeutralVariant;
            Func<Scheme, TonalPalette> primaryPalette = s => s.Primary;
            Func<Scheme, TonalPalette> secondaryPalette = s => s.Secondary;
            Func<Scheme, TonalPalette> tertiaryPalette = s => s.Tertiary;
            Func<Scheme, TonalPalette> errorPalette = s => s.Error;
            Func<Scheme, ColorRole> onHighestSurface = OnHighestSurface;

            Background = new ColorRole { Name = "background", Palette = neutral, ToneOf = Fixed(98, 6), IsBackground = true };
            OnBackground = new ColorRole
            {
                Name = "on_background",
                Palette = neutral,
                ToneOf = Fixed(10, 90),
                Background = _ => Background,
                Contrast = new ContrastCurve(3, 3, 4.5, 7),
            };
            Surface = new ColorRole { Name = "surface", Palette = neutral, ToneOf = Fixed(98, 6), IsBackground = true };
            SurfaceDim = Surfaced("surface_dim", neutral, new ContrastCurve(87, 87, 80, 75), new ContrastCurve(6, 6, 6, 6));
            SurfaceBright = Surfaced("surface_bright", neutral, new ContrastCurve(98, 98, 98, 98), new ContrastCurve(24, 24, 29, 34));
            SurfaceContainerLowest = Surfaced("surface_container_lowest", neutral, new ContrastCurve(100, 100, 100, 100), new ContrastCurve(4, 4, 2, 0));
            SurfaceContainerLow = Surfaced("surface_container_low", neutral, new ContrastCurve(96, 96, 96, 95), new ContrastCurve(10, 10, 11, 12));
            SurfaceContainer = Surfaced("surface_container", neutral, new ContrastCurve(94, 94, 92, 90), new ContrastCurve(12, 12, 16, 20));
            SurfaceContainerHigh = Surfaced("surface_container_high", neutral, new ContrastCurve(92, 92, 88, 85), new ContrastCurve(17, 17, 21, 25));
            SurfaceContainerHighest = Surfaced("surface_container_highest", neutral, new ContrastCurve(90, 90, 84, 80), new ContrastCurve(22, 22, 26, 30));
            OnSurface = new ColorRole
            {
                Name = "on_surface",
                Palette = neutral,
                ToneOf = Fixed(10, 90),
                Background = onHighestSurface,
                Contrast = new ContrastCurve(4.5, 7, 11, 21),
            };
            SurfaceVariant = new ColorRole { Name = "surface_variant", Palette = neutralVariant, ToneOf = Fixed(90, 30), IsBackground = true };
            OnSurfaceVariant = new ColorRole
            {
                Name = "on_surface_variant",
                Palette = neutralVariant,
                ToneOf = Fixed(30, 80),
                Background = onHighestSurface,
                Contrast = new ContrastCurve(3, 4.5, 7, 11),
            };
            InverseSurface = new ColorRole { Name = "inverse_surface", Palette = neutral, ToneOf = Fixed(20, 90), IsBackground = true };
            InverseOnSurface = new ColorRole
            {
                Name = "inverse_on_surface",
                Palette = neutral,
                ToneOf = Fixed(95, 20),
                Background = _ => InverseSurface,
                Contrast = new ContrastCurve(4.5, 7, 11, 21),
            };
            Outline = new ColorRole
            {
                Name = "outline",
                Palette = neutralVariant,
                ToneOf = Fixed(50, 60),
                Background = onHighestSurface,
                Contrast = new ContrastCurve(1.5, 3, 4.5, 7),
            };
            OutlineVariant = new ColorRole
            {
                Name = "outline_variant",
                Palette = neutralVariant,
                ToneOf = Fixed(80, 30),
                Background = onHighestSurface,
                Contrast = new ContrastCurve(1, 1, 3, 4.5),
            };
            Shadow = new ColorRole { Name = "shadow", Palette = neutral, ToneOf = _ => 0 };
            Scrim = new ColorRole { Name = "scrim", Palette = neutral, ToneOf = _ => 0 };

            // The roles of an accent colour: the colour, the colour on it, the container and the colour on the
            // container.
            (Primary, OnPrimary, PrimaryContainer, OnPrimaryContainer) = Accent("primary", primaryPalette, onHighestSurface);
            (Secondary, OnSecondary, SecondaryContainer, OnSecondaryContainer) = Accent("secondary", secondaryPalette, onHighestSurface);
            (Tertiary, OnTertiary, TertiaryContainer, OnTertiaryContainer) = Accent("tertiary", tertiaryPalette, onHighestSurface);
            (Error, OnError, ErrorContainer, OnErrorContainer) = Accent("error", errorPalette, onHighestSurface);
            InversePrimary = new ColorRole
            {
                Name = "inverse_primary",
                Palette = primaryPalette,
                ToneOf = Fixed(80, 40),
                Background = _ => InverseSurface,
                Contrast = new ContrastCurve(3, 4.5, 7, 7),
            };
            (PrimaryFixed, PrimaryFixedDim, OnPrimaryFixed, OnPrimaryFixedVariant) = FixedRoles("primary", primaryPalette, onHighestSurface);
            (SecondaryFixed, SecondaryFixedDim, OnSecondaryFixed, OnSecondaryFixedVariant) = FixedRoles("secondary", secondaryPalette, onHighestSurface);
            (TertiaryFixed, TertiaryFixedDim, OnTertiaryFixed, OnTertiaryFixedVariant) = FixedRoles("tertiary", tertiaryPalette, onHighestSurface);
        }

        internal ColorRole Background { get; }

        internal ColorRole OnBackground { get; }

        internal ColorRole Surface { get; }

        internal ColorRole SurfaceDim { get; }

        internal ColorRole SurfaceBright { get; }

        internal ColorRole SurfaceContainerLowest { get; }

        internal ColorRole SurfaceContainerLow { get; }

        internal ColorRole SurfaceContainer { get; }

        internal ColorRole SurfaceContainerHigh { get; }

        internal ColorRole SurfaceContainerHighest { get; }

        internal ColorRole OnSurface { get; }

        internal ColorRole SurfaceVariant { get; }

        internal ColorRole OnSurfaceVariant { get; }

        internal ColorRole InverseSurface { get; }

        internal ColorRole InverseOnSurface { get; }

        internal ColorRole Outline { get; }

        internal ColorRole OutlineVariant { get; }

        internal ColorRole Shadow { get; }

        internal ColorRole Scrim { get; }

        internal ColorRole Primary { get; }

        internal ColorRole OnPrimary { get; }

        internal ColorRole PrimaryContainer { get; }

        internal ColorRole OnPrimaryContainer { get; }

        internal ColorRole InversePrimary { get; }

        internal ColorRole Secondary { get; }

        internal ColorRole OnSecondary { get; }

        internal ColorRole SecondaryContainer { get; }

        internal ColorRole OnSecondaryContainer { get; }

        internal ColorRole Tertiary { get; }

        internal ColorRole OnTertiary { get; }

        internal ColorRole TertiaryContainer { get; }

        internal ColorRole OnTertiaryContainer { get; }

        internal ColorRole Error { get; }

        internal ColorRole OnError { get; }

        internal ColorRole ErrorContainer { get; }

        internal ColorRole OnErrorContainer { get; }

        internal ColorRole PrimaryFixed { get; }

        internal ColorRole PrimaryFixedDim { get; }

        internal ColorRole OnPrimaryFixed { get; }

        internal ColorRole OnPrimaryFixedVariant { get; }

        internal ColorRole SecondaryFixed { get; }

        internal ColorRole SecondaryFixedDim { get; }

        internal ColorRole OnSecondaryFixed { get; }

        internal ColorRole OnSecondaryFixedVariant { get; }

        internal ColorRole TertiaryFixed { get; }

        internal ColorRole TertiaryFixedDim { get; }

        internal ColorRole OnTertiaryFixed { get; }

        internal ColorRole OnTertiaryFixedVariant { get; }

        private static Func<Scheme, double> Fixed(double light, double dark) => s => s.Dark ? dark : light;

        // The brightest surface in a dark scheme and the dimmest in a light one: what the colours on a surface
        // are measured against.
        private ColorRole OnHighestSurface(Scheme s) => s.Dark ? SurfaceBright : SurfaceDim;

        private static ColorRole Surfaced(string name, Func<Scheme, TonalPalette> palette, ContrastCurve light, ContrastCurve dark) => new()
        {
            Name = name,
            Palette = palette,
            IsBackground = true,
            ToneOf = s => s.Dark ? dark.Get(s.ContrastLevel) : light.Get(s.ContrastLevel),
        };

        private (ColorRole Main, ColorRole OnMain, ColorRole Container, ColorRole OnContainer) Accent(
            string prefix,
            Func<Scheme, TonalPalette> palette,
            Func<Scheme, ColorRole> onHighestSurface)
        {
            ColorRole? main = null;
            ColorRole? onMain = null;
            ColorRole? container = null;
            ColorRole? onContainer = null;
            ToneDeltaPair Pair(Scheme scheme) => new(container!, main!, 10, Polarity.Nearer, false);
            main = new ColorRole
            {
                Name = prefix,
                Palette = palette,
                ToneOf = Fixed(40, 80),
                IsBackground = true,
                Background = onHighestSurface,
                Contrast = new ContrastCurve(3, 4.5, 7, 7),
                Pair = Pair,
            };
            onMain = new ColorRole
            {
                Name = "on_" + prefix,
                Palette = palette,
                ToneOf = Fixed(100, 20),
                Background = _ => main,
                Contrast = new ContrastCurve(4.5, 7, 11, 21),
            };
            container = new ColorRole
            {
                Name = prefix + "_container",
                Palette = palette,
                ToneOf = Fixed(90, 30),
                IsBackground = true,
                Background = onHighestSurface,
                Contrast = new ContrastCurve(1, 1, 3, 4.5),
                Pair = Pair,
            };
            onContainer = new ColorRole
            {
                Name = "on_" + prefix + "_container",
                Palette = palette,
                ToneOf = Fixed(30, 90),
                Background = _ => container,
                Contrast = new ContrastCurve(3, 4.5, 7, 11),
            };
            return (main, onMain, container, onContainer);
        }

        // The roles of an accent colour that does not change with the theme.
        private (ColorRole Fixed, ColorRole Dim, ColorRole OnFixed, ColorRole OnVariant) FixedRoles(
            string prefix,
            Func<Scheme, TonalPalette> palette,
            Func<Scheme, ColorRole> onHighestSurface)
        {
            ColorRole? fixedRole = null;
            ColorRole? dim = null;
            ColorRole? onFixed = null;
            ColorRole? onVariant = null;
            ToneDeltaPair Pair(Scheme scheme) => new(fixedRole!, dim!, 10, Polarity.Lighter, true);
            fixedRole = new ColorRole
            {
                Name = prefix + "_fixed",
                Palette = palette,
                ToneOf = _ => 90,
                IsBackground = true,
                Background = onHighestSurface,
                Contrast = new ContrastCurve(1, 1, 3, 4.5),
                Pair = Pair,
            };
            dim = new ColorRole
            {
                Name = prefix + "_fixed_dim",
                Palette = palette,
                ToneOf = _ => 80,
                IsBackground = true,
                Background = onHighestSurface,
                Contrast = new ContrastCurve(1, 1, 3, 4.5),
                Pair = Pair,
            };
            onFixed = new ColorRole
            {
                Name = "on_" + prefix + "_fixed",
                Palette = palette,
                ToneOf = _ => 10,
                Background = _ => dim,
                SecondBackground = _ => fixedRole,
                Contrast = new ContrastCurve(4.5, 7, 11, 21),
            };
            onVariant = new ColorRole
            {
                Name = "on_" + prefix + "_fixed_variant",
                Palette = palette,
                ToneOf = _ => 30,
                Background = _ => dim,
                SecondBackground = _ => fixedRole,
                Contrast = new ContrastCurve(3, 4.5, 7, 11),
            };
            return (fixedRole, dim, onFixed, onVariant);
        }
    }
}
