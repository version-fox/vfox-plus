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

namespace VfoxPlus.Design.Tokens;

/// <summary>
/// The Material 3 Expressive corner radius scale, in dp. See
/// <see href="https://m3.material.io/styles/shape/corner-radius-scale"/>.
/// </summary>
public static class Corners
{
    /// <summary>0 dp.</summary>
    public const double None = 0;

    /// <summary>4 dp.</summary>
    public const double ExtraSmall = 4;

    /// <summary>8 dp.</summary>
    public const double Small = 8;

    /// <summary>12 dp.</summary>
    public const double Medium = 12;

    /// <summary>16 dp.</summary>
    public const double Large = 16;

    /// <summary>20 dp.</summary>
    public const double LargeIncreased = 20;

    /// <summary>28 dp.</summary>
    public const double ExtraLarge = 28;

    /// <summary>32 dp.</summary>
    public const double ExtraLargeIncreased = 32;

    /// <summary>48 dp.</summary>
    public const double ExtraExtraLarge = 48;

    /// <summary>Fully rounded: half of the shorter side.</summary>
    public const double Full = 9999;
}

/// <summary>Spacing between elements, on a 4 dp grid.</summary>
public static class Gaps
{
    /// <summary>2 dp: between the segments of connected groups.</summary>
    public const double Hairline = 2;

    /// <summary>4 dp.</summary>
    public const double Xs = 4;

    /// <summary>8 dp.</summary>
    public const double S = 8;

    /// <summary>12 dp.</summary>
    public const double M = 12;

    /// <summary>16 dp.</summary>
    public const double L = 16;

    /// <summary>24 dp.</summary>
    public const double Xl = 24;

    /// <summary>32 dp.</summary>
    public const double Xxl = 32;
}

/// <summary>
/// The Material 3 window size classes, by window width. See
/// <see href="https://m3.material.io/foundations/layout/applying-layout/window-size-classes"/>.
/// </summary>
public enum WindowSizeClass
{
    /// <summary>Narrower than 600 dp.</summary>
    Compact,

    /// <summary>600 dp to 839 dp.</summary>
    Medium,

    /// <summary>840 dp to 1199 dp.</summary>
    Expanded,

    /// <summary>1200 dp to 1599 dp.</summary>
    Large,

    /// <summary>1600 dp and wider.</summary>
    ExtraLarge,
}

/// <summary>Helpers for <see cref="WindowSizeClass"/>.</summary>
public static class WindowSizeClasses
{
    /// <summary>Returns the size class of a window.</summary>
    /// <param name="width">The width of the window in dp.</param>
    /// <returns>The class.</returns>
    public static WindowSizeClass FromWidth(double width) => width switch
    {
        < 600 => WindowSizeClass.Compact,
        < 840 => WindowSizeClass.Medium,
        < 1200 => WindowSizeClass.Expanded,
        < 1600 => WindowSizeClass.Large,
        _ => WindowSizeClass.ExtraLarge,
    };
}

/// <summary>One of the roles of the Material 3 type scale.</summary>
public enum TextRole
{
    /// <summary>Display large.</summary>
    DisplayLarge,

    /// <summary>Display medium.</summary>
    DisplayMedium,

    /// <summary>Display small.</summary>
    DisplaySmall,

    /// <summary>Headline large.</summary>
    HeadlineLarge,

    /// <summary>Headline medium.</summary>
    HeadlineMedium,

    /// <summary>Headline small.</summary>
    HeadlineSmall,

    /// <summary>Title large.</summary>
    TitleLarge,

    /// <summary>Title medium.</summary>
    TitleMedium,

    /// <summary>Title small.</summary>
    TitleSmall,

    /// <summary>Body large.</summary>
    BodyLarge,

    /// <summary>Body medium.</summary>
    BodyMedium,

    /// <summary>Body small.</summary>
    BodySmall,

    /// <summary>Label large.</summary>
    LabelLarge,

    /// <summary>Label medium.</summary>
    LabelMedium,

    /// <summary>Label small.</summary>
    LabelSmall,
}

/// <summary>
/// A size, a line height and a weight.
/// </summary>
/// <param name="Size">The font size in dp.</param>
/// <param name="Line">The line height in dp.</param>
/// <param name="Weight">The weight, from 100 to 900.</param>
public readonly record struct TextStyle(double Size, double Line, int Weight)
{
    /// <summary>Returns the style with another weight.</summary>
    /// <param name="weight">The weight.</param>
    /// <returns>The style.</returns>
    public TextStyle WithWeight(int weight) => this with { Weight = weight };

    /// <summary>Returns the style with another size, which keeps the proportion of the line height.</summary>
    /// <param name="size">The size.</param>
    /// <returns>The style.</returns>
    public TextStyle WithSize(double size) => this with { Size = size, Line = Line * size / Size };
}

/// <summary>
/// The Material 3 type scale with the baseline weights of Flutter's <c>Typography.material2021</c>, and the
/// <em>emphasized</em> variant of each role that Material 3 Expressive adds (see <c>docs/design-system.md#typography</c>).
/// </summary>
public static class Typography
{
    private static readonly TextStyle[] Scale =
    [
        new(57, 64, 400),
        new(45, 52, 400),
        new(36, 44, 400),
        new(32, 40, 400),
        new(28, 36, 400),
        new(24, 32, 400),
        new(22, 28, 400),
        new(16, 24, 500),
        new(14, 20, 500),
        new(16, 24, 400),
        new(14, 20, 400),
        new(12, 16, 400),
        new(14, 20, 500),
        new(12, 16, 500),
        new(11, 16, 500),
    ];

    private static readonly int[] EmphasizedWeights =
    [
        500, 500, 500,
        500, 500, 500,
        500, 700, 700,
        500, 500, 500,
        700, 700, 700,
    ];

    /// <summary>Returns the style of a role of the type scale.</summary>
    /// <param name="role">The role.</param>
    /// <returns>The style.</returns>
    public static TextStyle Of(TextRole role) => Scale[(int)role];

    /// <summary>
    /// Returns the emphasized style of a role of the type scale, which Material 3 Expressive uses for the text that
    /// matters most on a screen.
    /// </summary>
    /// <param name="role">The role.</param>
    /// <returns>The style.</returns>
    public static TextStyle Emphasized(TextRole role) => Scale[(int)role].WithWeight(EmphasizedWeights[(int)role]);
}

/// <summary>
/// The avatar of a plugin: its shape and colours come from a hash of its name that is the same on every run and
/// in every implementation.
/// </summary>
public static class Avatars
{
    /// <summary>
    /// Returns a hash of a text that is the same on every run and in every implementation of vfox-plus (FNV-1a
    /// over the UTF-16 code units).
    /// </summary>
    /// <param name="text">The text.</param>
    /// <returns>The hash.</returns>
    public static uint StableHash(string text)
    {
        uint hash = 0x811C9DC5;
        foreach (char unit in text)
        {
            hash ^= unit;
            hash *= 0x01000193;
        }

        return hash;
    }

    /// <summary>Returns the index in <see cref="Shapes.MaterialShapes.AvatarShapes"/> of the shape that a name gets.</summary>
    /// <param name="name">The name.</param>
    /// <returns>The index.</returns>
    public static int ShapeIndex(string name) => (int)(StableHash(name) % (uint)Shapes.MaterialShapes.AvatarShapes.Count);

    /// <summary>
    /// Returns which of the three colour pairs a name gets: 0 for primary fixed dim, 1 for secondary and 2 for the
    /// tertiary container. Not the secondary container, which marks selected list items, nor the primary container,
    /// which some seeds (the vfox orange among them) make just as light.
    /// </summary>
    /// <param name="name">The name.</param>
    /// <returns>The index of the pair.</returns>
    public static int ColorIndex(string name) => (int)(StableHash(name) % 3);

    /// <summary>Returns the first character of a name, in upper case, or <c>?</c> for an empty name.</summary>
    /// <param name="name">The name.</param>
    /// <returns>The initial.</returns>
    public static string Initial(string name)
    {
        if (name.Length == 0)
        {
            return "?";
        }

        int length = char.IsHighSurrogate(name[0]) && name.Length > 1 ? 2 : 1;
        return name[..length].ToUpperInvariant();
    }
}
