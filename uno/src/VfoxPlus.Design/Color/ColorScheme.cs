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
/// A colour scheme: the colour of every role, as ARGB (<c>0xAARRGGBB</c> with an opaque alpha). The roles are
/// those of Material 3; the documentation of Material 3 describes what they are for.
/// </summary>
public sealed record ColorScheme
{
    /// <summary>Gets the colour <c>background</c>.</summary>
    public required uint Background { get; init; }

    /// <summary>Gets the colour <c>on-background</c>.</summary>
    public required uint OnBackground { get; init; }

    /// <summary>Gets the colour <c>surface</c>.</summary>
    public required uint Surface { get; init; }

    /// <summary>Gets the colour <c>surface-dim</c>.</summary>
    public required uint SurfaceDim { get; init; }

    /// <summary>Gets the colour <c>surface-bright</c>.</summary>
    public required uint SurfaceBright { get; init; }

    /// <summary>Gets the colour <c>surface-container-lowest</c>.</summary>
    public required uint SurfaceContainerLowest { get; init; }

    /// <summary>Gets the colour <c>surface-container-low</c>.</summary>
    public required uint SurfaceContainerLow { get; init; }

    /// <summary>Gets the colour <c>surface-container</c>.</summary>
    public required uint SurfaceContainer { get; init; }

    /// <summary>Gets the colour <c>surface-container-high</c>.</summary>
    public required uint SurfaceContainerHigh { get; init; }

    /// <summary>Gets the colour <c>surface-container-highest</c>.</summary>
    public required uint SurfaceContainerHighest { get; init; }

    /// <summary>Gets the colour <c>on-surface</c>.</summary>
    public required uint OnSurface { get; init; }

    /// <summary>Gets the colour <c>on-surface-variant</c>.</summary>
    public required uint OnSurfaceVariant { get; init; }

    /// <summary>Gets the colour <c>outline</c>.</summary>
    public required uint Outline { get; init; }

    /// <summary>Gets the colour <c>outline-variant</c>.</summary>
    public required uint OutlineVariant { get; init; }

    /// <summary>Gets the colour <c>inverse-surface</c>.</summary>
    public required uint InverseSurface { get; init; }

    /// <summary>Gets the colour <c>inverse-on-surface</c>.</summary>
    public required uint InverseOnSurface { get; init; }

    /// <summary>Gets the colour <c>inverse-primary</c>.</summary>
    public required uint InversePrimary { get; init; }

    /// <summary>Gets the colour <c>surface-variant</c>.</summary>
    public required uint SurfaceVariant { get; init; }

    /// <summary>Gets the colour <c>shadow</c>.</summary>
    public required uint Shadow { get; init; }

    /// <summary>Gets the colour <c>scrim</c>.</summary>
    public required uint Scrim { get; init; }

    /// <summary>Gets the colour <c>primary</c>.</summary>
    public required uint Primary { get; init; }

    /// <summary>Gets the colour <c>on-primary</c>.</summary>
    public required uint OnPrimary { get; init; }

    /// <summary>Gets the colour <c>primary-container</c>.</summary>
    public required uint PrimaryContainer { get; init; }

    /// <summary>Gets the colour <c>on-primary-container</c>.</summary>
    public required uint OnPrimaryContainer { get; init; }

    /// <summary>Gets the colour <c>primary-fixed</c>.</summary>
    public required uint PrimaryFixed { get; init; }

    /// <summary>Gets the colour <c>primary-fixed-dim</c>.</summary>
    public required uint PrimaryFixedDim { get; init; }

    /// <summary>Gets the colour <c>on-primary-fixed</c>.</summary>
    public required uint OnPrimaryFixed { get; init; }

    /// <summary>Gets the colour <c>on-primary-fixed-variant</c>.</summary>
    public required uint OnPrimaryFixedVariant { get; init; }

    /// <summary>Gets the colour <c>secondary</c>.</summary>
    public required uint Secondary { get; init; }

    /// <summary>Gets the colour <c>on-secondary</c>.</summary>
    public required uint OnSecondary { get; init; }

    /// <summary>Gets the colour <c>secondary-container</c>.</summary>
    public required uint SecondaryContainer { get; init; }

    /// <summary>Gets the colour <c>on-secondary-container</c>.</summary>
    public required uint OnSecondaryContainer { get; init; }

    /// <summary>Gets the colour <c>secondary-fixed</c>.</summary>
    public required uint SecondaryFixed { get; init; }

    /// <summary>Gets the colour <c>secondary-fixed-dim</c>.</summary>
    public required uint SecondaryFixedDim { get; init; }

    /// <summary>Gets the colour <c>on-secondary-fixed</c>.</summary>
    public required uint OnSecondaryFixed { get; init; }

    /// <summary>Gets the colour <c>on-secondary-fixed-variant</c>.</summary>
    public required uint OnSecondaryFixedVariant { get; init; }

    /// <summary>Gets the colour <c>tertiary</c>.</summary>
    public required uint Tertiary { get; init; }

    /// <summary>Gets the colour <c>on-tertiary</c>.</summary>
    public required uint OnTertiary { get; init; }

    /// <summary>Gets the colour <c>tertiary-container</c>.</summary>
    public required uint TertiaryContainer { get; init; }

    /// <summary>Gets the colour <c>on-tertiary-container</c>.</summary>
    public required uint OnTertiaryContainer { get; init; }

    /// <summary>Gets the colour <c>tertiary-fixed</c>.</summary>
    public required uint TertiaryFixed { get; init; }

    /// <summary>Gets the colour <c>tertiary-fixed-dim</c>.</summary>
    public required uint TertiaryFixedDim { get; init; }

    /// <summary>Gets the colour <c>on-tertiary-fixed</c>.</summary>
    public required uint OnTertiaryFixed { get; init; }

    /// <summary>Gets the colour <c>on-tertiary-fixed-variant</c>.</summary>
    public required uint OnTertiaryFixedVariant { get; init; }

    /// <summary>Gets the colour <c>error</c>.</summary>
    public required uint Error { get; init; }

    /// <summary>Gets the colour <c>on-error</c>.</summary>
    public required uint OnError { get; init; }

    /// <summary>Gets the colour <c>error-container</c>.</summary>
    public required uint ErrorContainer { get; init; }

    /// <summary>Gets the colour <c>on-error-container</c>.</summary>
    public required uint OnErrorContainer { get; init; }

    /// <summary>
    /// Creates the tonal spot scheme of a seed colour, in the 2021 specification of the Material colour utilities
    /// that Flutter's <c>ColorScheme.fromSeed</c> uses, at the standard contrast.
    /// </summary>
    /// <param name="seed">The seed colour as <c>0xAARRGGBB</c>.</param>
    /// <param name="dark">Whether the scheme is for a dark theme.</param>
    /// <returns>The scheme.</returns>
    public static ColorScheme Generate(uint seed, bool dark) => DynamicRoles.Generate(seed, dark);

    /// <summary>Lists the roles with the names of the web front end, such as <c>surface-container-low</c>, in the order of its specification.</summary>
    /// <returns>The names and the colours.</returns>
    public IReadOnlyList<(string Name, uint Argb)> Tokens() =>
    [
        ("background", Background), ("on-background", OnBackground),
        ("surface", Surface), ("surface-dim", SurfaceDim), ("surface-bright", SurfaceBright),
        ("surface-container-lowest", SurfaceContainerLowest), ("surface-container-low", SurfaceContainerLow),
        ("surface-container", SurfaceContainer), ("surface-container-high", SurfaceContainerHigh),
        ("surface-container-highest", SurfaceContainerHighest),
        ("on-surface", OnSurface), ("on-surface-variant", OnSurfaceVariant),
        ("outline", Outline), ("outline-variant", OutlineVariant),
        ("inverse-surface", InverseSurface), ("inverse-on-surface", InverseOnSurface),
        ("primary", Primary), ("on-primary", OnPrimary),
        ("primary-container", PrimaryContainer), ("on-primary-container", OnPrimaryContainer),
        ("primary-fixed", PrimaryFixed), ("primary-fixed-dim", PrimaryFixedDim),
        ("on-primary-fixed", OnPrimaryFixed), ("on-primary-fixed-variant", OnPrimaryFixedVariant),
        ("inverse-primary", InversePrimary),
        ("secondary", Secondary), ("on-secondary", OnSecondary),
        ("secondary-container", SecondaryContainer), ("on-secondary-container", OnSecondaryContainer),
        ("secondary-fixed", SecondaryFixed), ("secondary-fixed-dim", SecondaryFixedDim),
        ("on-secondary-fixed", OnSecondaryFixed), ("on-secondary-fixed-variant", OnSecondaryFixedVariant),
        ("tertiary", Tertiary), ("on-tertiary", OnTertiary),
        ("tertiary-container", TertiaryContainer), ("on-tertiary-container", OnTertiaryContainer),
        ("tertiary-fixed", TertiaryFixed), ("tertiary-fixed-dim", TertiaryFixedDim),
        ("on-tertiary-fixed", OnTertiaryFixed), ("on-tertiary-fixed-variant", OnTertiaryFixedVariant),
        ("error", Error), ("on-error", OnError),
        ("error-container", ErrorContainer), ("on-error-container", OnErrorContainer),
        ("surface-variant", SurfaceVariant), ("shadow", Shadow), ("scrim", Scrim),
    ];
}
