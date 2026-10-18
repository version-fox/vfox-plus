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

package mcu

// Colors is a colour scheme: the colour of every role, as ARGB (0xAARRGGBB
// with an opaque alpha). The roles are those of Material 3; the
// documentation of Material 3 describes what they are for.
type Colors struct {
	Background, OnBackground uint32

	Surface, SurfaceDim, SurfaceBright                            uint32
	SurfaceContainerLowest, SurfaceContainerLow, SurfaceContainer uint32
	SurfaceContainerHigh, SurfaceContainerHighest                 uint32
	OnSurface, OnSurfaceVariant, Outline, OutlineVariant          uint32
	InverseSurface, InverseOnSurface, InversePrimary              uint32
	SurfaceVariant, Shadow, Scrim                                 uint32

	Primary, OnPrimary, PrimaryContainer, OnPrimaryContainer         uint32
	PrimaryFixed, PrimaryFixedDim, OnPrimaryFixed                    uint32
	OnPrimaryFixedVariant                                            uint32
	Secondary, OnSecondary, SecondaryContainer, OnSecondaryContainer uint32
	SecondaryFixed, SecondaryFixedDim, OnSecondaryFixed              uint32
	OnSecondaryFixedVariant                                          uint32
	Tertiary, OnTertiary, TertiaryContainer, OnTertiaryContainer     uint32
	TertiaryFixed, TertiaryFixedDim, OnTertiaryFixed                 uint32
	OnTertiaryFixedVariant                                           uint32
	Error, OnError, ErrorContainer, OnErrorContainer                 uint32
}

// Generate creates the tonal spot scheme of the seed colour (0xAARRGGBB), for
// a light or a dark theme, at the standard contrast.
func Generate(seed uint32, dark bool) Colors {
	s := newScheme(HctFromARGB(seed), dark, 0)
	r := colorRoles
	argb := func(c *role) uint32 { return c.hct(s).ARGB() }
	return Colors{
		Background: argb(r.background), OnBackground: argb(r.onBackground),

		Surface: argb(r.surface), SurfaceDim: argb(r.surfaceDim), SurfaceBright: argb(r.surfaceBright),
		SurfaceContainerLowest:  argb(r.surfaceContainerLowest),
		SurfaceContainerLow:     argb(r.surfaceContainerLow),
		SurfaceContainer:        argb(r.surfaceContainer),
		SurfaceContainerHigh:    argb(r.surfaceContainerHigh),
		SurfaceContainerHighest: argb(r.surfaceContainerHighest),
		OnSurface:               argb(r.onSurface), OnSurfaceVariant: argb(r.onSurfaceVariant),
		Outline: argb(r.outline), OutlineVariant: argb(r.outlineVariant),
		InverseSurface: argb(r.inverseSurface), InverseOnSurface: argb(r.inverseOnSurface),
		InversePrimary: argb(r.inversePrimary),
		SurfaceVariant: argb(r.surfaceVariant), Shadow: argb(r.shadow), Scrim: argb(r.scrim),

		Primary: argb(r.primary), OnPrimary: argb(r.onPrimary),
		PrimaryContainer: argb(r.primaryContainer), OnPrimaryContainer: argb(r.onPrimaryContainer),
		PrimaryFixed: argb(r.primaryFixed), PrimaryFixedDim: argb(r.primaryFixedDim),
		OnPrimaryFixed: argb(r.onPrimaryFixed), OnPrimaryFixedVariant: argb(r.onPrimaryFixedVariant),
		Secondary: argb(r.secondary), OnSecondary: argb(r.onSecondary),
		SecondaryContainer: argb(r.secondaryContainer), OnSecondaryContainer: argb(r.onSecondaryContainer),
		SecondaryFixed: argb(r.secondaryFixed), SecondaryFixedDim: argb(r.secondaryFixedDim),
		OnSecondaryFixed: argb(r.onSecondaryFixed), OnSecondaryFixedVariant: argb(r.onSecondaryFixedVariant),
		Tertiary: argb(r.tertiary), OnTertiary: argb(r.onTertiary),
		TertiaryContainer: argb(r.tertiaryContainer), OnTertiaryContainer: argb(r.onTertiaryContainer),
		TertiaryFixed: argb(r.tertiaryFixed), TertiaryFixedDim: argb(r.tertiaryFixedDim),
		OnTertiaryFixed: argb(r.onTertiaryFixed), OnTertiaryFixedVariant: argb(r.onTertiaryFixedVariant),
		Error: argb(r.error), OnError: argb(r.onError),
		ErrorContainer: argb(r.errorContainer), OnErrorContainer: argb(r.onErrorContainer),
	}
}

// Token is a colour role by its name in the CSS custom properties of the web
// front end, such as `surface-container-low`.
type Token struct {
	Name string
	ARGB uint32
}

// Tokens lists the roles with the names of the web front end, in the order of
// its specification.
func (c Colors) Tokens() []Token {
	return []Token{
		{"background", c.Background}, {"on-background", c.OnBackground},
		{"surface", c.Surface}, {"surface-dim", c.SurfaceDim}, {"surface-bright", c.SurfaceBright},
		{"surface-container-lowest", c.SurfaceContainerLowest}, {"surface-container-low", c.SurfaceContainerLow},
		{"surface-container", c.SurfaceContainer}, {"surface-container-high", c.SurfaceContainerHigh},
		{"surface-container-highest", c.SurfaceContainerHighest},
		{"on-surface", c.OnSurface}, {"on-surface-variant", c.OnSurfaceVariant},
		{"outline", c.Outline}, {"outline-variant", c.OutlineVariant},
		{"inverse-surface", c.InverseSurface}, {"inverse-on-surface", c.InverseOnSurface},
		{"primary", c.Primary}, {"on-primary", c.OnPrimary},
		{"primary-container", c.PrimaryContainer}, {"on-primary-container", c.OnPrimaryContainer},
		{"primary-fixed", c.PrimaryFixed}, {"primary-fixed-dim", c.PrimaryFixedDim},
		{"on-primary-fixed", c.OnPrimaryFixed}, {"on-primary-fixed-variant", c.OnPrimaryFixedVariant},
		{"inverse-primary", c.InversePrimary},
		{"secondary", c.Secondary}, {"on-secondary", c.OnSecondary},
		{"secondary-container", c.SecondaryContainer}, {"on-secondary-container", c.OnSecondaryContainer},
		{"secondary-fixed", c.SecondaryFixed}, {"secondary-fixed-dim", c.SecondaryFixedDim},
		{"on-secondary-fixed", c.OnSecondaryFixed}, {"on-secondary-fixed-variant", c.OnSecondaryFixedVariant},
		{"tertiary", c.Tertiary}, {"on-tertiary", c.OnTertiary},
		{"tertiary-container", c.TertiaryContainer}, {"on-tertiary-container", c.OnTertiaryContainer},
		{"tertiary-fixed", c.TertiaryFixed}, {"tertiary-fixed-dim", c.TertiaryFixedDim},
		{"on-tertiary-fixed", c.OnTertiaryFixed}, {"on-tertiary-fixed-variant", c.OnTertiaryFixedVariant},
		{"error", c.Error}, {"on-error", c.OnError},
		{"error-container", c.ErrorContainer}, {"on-error-container", c.OnErrorContainer},
		{"surface-variant", c.SurfaceVariant}, {"shadow", c.Shadow}, {"scrim", c.Scrim},
	}
}
