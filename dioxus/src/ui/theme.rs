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

//! The colours of the Material 3 Expressive theme (`docs/design-system.md`):
//! the tonal spot scheme of the Material colour utilities for the seed
//! colour, in the 2021 specification that the other implementations use.
//! They are written to the root element of the document, where they override
//! the scheme of `m3e-theme` and reach every component, including the snack
//! bars that `@m3e/web` adds to the body.

use material_colors::color::Rgb;
use material_colors::dynamic_color::{DynamicScheme, Platform, SpecVersion, Variant};
use material_colors::hct::Hct;

use crate::core::settings::DEFAULT_SEED_COLOR;

/// Seed colours offered when the desktop accent colour is not used.
pub const SEED_COLORS: [u32; 8] =
    [DEFAULT_SEED_COLOR, 0xFFD9_473B, 0xFFC2_407A, 0xFF7E_57C2, 0xFF3F_6FD8, 0xFF0B_8A9A, 0xFF3E_8E41, 0xFF8D_7B1E];

/// `#rrggbb` of an ARGB colour.
pub fn hex(argb: u32) -> String {
    format!("#{:06x}", argb & 0x00FF_FFFF)
}

fn rgb_hex(color: Rgb) -> String {
    format!("#{:02x}{:02x}{:02x}", color.red, color.green, color.blue)
}

fn scheme(seed: u32, dark: bool) -> DynamicScheme {
    let source = Hct::new(Rgb::from_u32(seed & 0x00FF_FFFF));
    DynamicScheme::from_spec(source, Variant::TonalSpot, dark, Some(0.0), Platform::Phone, SpecVersion::Spec2021)
}

/// The `--md-sys-color-*` custom properties of the scheme for `seed`, the
/// same as those of the web front end. The dim roles, which only the 2025
/// specification defines and the components do not use, are left out.
pub fn color_tokens(seed: u32, dark: bool) -> Vec<(&'static str, String)> {
    let s = scheme(seed, dark);
    let roles: [(&'static str, Rgb); 48] = [
        ("background", s.background()),
        ("on-background", s.on_background()),
        ("surface", s.surface()),
        ("surface-dim", s.surface_dim()),
        ("surface-bright", s.surface_bright()),
        ("surface-container-lowest", s.surface_container_lowest()),
        ("surface-container-low", s.surface_container_low()),
        ("surface-container", s.surface_container()),
        ("surface-container-high", s.surface_container_high()),
        ("surface-container-highest", s.surface_container_highest()),
        ("on-surface", s.on_surface()),
        ("on-surface-variant", s.on_surface_variant()),
        ("outline", s.outline()),
        ("outline-variant", s.outline_variant()),
        ("inverse-surface", s.inverse_surface()),
        ("inverse-on-surface", s.inverse_on_surface()),
        ("primary", s.primary()),
        ("on-primary", s.on_primary()),
        ("primary-container", s.primary_container()),
        ("on-primary-container", s.on_primary_container()),
        ("primary-fixed", s.primary_fixed()),
        ("primary-fixed-dim", s.primary_fixed_dim()),
        ("on-primary-fixed", s.on_primary_fixed()),
        ("on-primary-fixed-variant", s.on_primary_fixed_variant()),
        ("inverse-primary", s.inverse_primary()),
        ("secondary", s.secondary()),
        ("on-secondary", s.on_secondary()),
        ("secondary-container", s.secondary_container()),
        ("on-secondary-container", s.on_secondary_container()),
        ("secondary-fixed", s.secondary_fixed()),
        ("secondary-fixed-dim", s.secondary_fixed_dim()),
        ("on-secondary-fixed", s.on_secondary_fixed()),
        ("on-secondary-fixed-variant", s.on_secondary_fixed_variant()),
        ("tertiary", s.tertiary()),
        ("on-tertiary", s.on_tertiary()),
        ("tertiary-container", s.tertiary_container()),
        ("on-tertiary-container", s.on_tertiary_container()),
        ("tertiary-fixed", s.tertiary_fixed()),
        ("tertiary-fixed-dim", s.tertiary_fixed_dim()),
        ("on-tertiary-fixed", s.on_tertiary_fixed()),
        ("on-tertiary-fixed-variant", s.on_tertiary_fixed_variant()),
        ("error", s.error()),
        ("on-error", s.on_error()),
        ("error-container", s.error_container()),
        ("on-error-container", s.on_error_container()),
        ("surface-variant", s.surface_variant()),
        ("shadow", s.shadow()),
        ("scrim", s.scrim()),
    ];
    roles.into_iter().map(|(name, color)| (name, rgb_hex(color))).collect()
}

/// The primary colour of the scheme for `seed`, as `#rrggbb`.
pub fn primary_of(seed: u32, dark: bool) -> String {
    rgb_hex(scheme(seed, dark).primary())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn token(seed: u32, dark: bool, name: &str) -> String {
        color_tokens(seed, dark).into_iter().find(|(role, _)| *role == name).map(|(_, value)| value).unwrap()
    }

    /// The values of the web front end, from `@material/material-color-utilities` 0.4.0.
    #[test]
    fn the_scheme_of_the_default_seed_matches_the_web_front_end() {
        let light = [
            ("surface", "#fff8f5"),
            ("surface-container-low", "#fff1e8"),
            ("on-surface", "#221a14"),
            ("primary", "#88521c"),
            ("primary-container", "#ffdcc1"),
            ("primary-fixed-dim", "#ffb779"),
            ("secondary-container", "#ffdcc1"),
            ("tertiary-container", "#e0e7b1"),
            ("surface-variant", "#f3dfd1"),
            ("scrim", "#000000"),
        ];
        let dark = [
            ("surface", "#19120c"),
            ("surface-container-low", "#221a14"),
            ("on-surface", "#efe0d6"),
            ("primary", "#ffb779"),
            ("primary-container", "#6b3b04"),
            ("primary-fixed-dim", "#ffb779"),
            ("secondary-container", "#5a422e"),
            ("tertiary-container", "#434a22"),
            ("surface-variant", "#51443b"),
            ("shadow", "#000000"),
        ];
        for (name, value) in light {
            assert_eq!(token(DEFAULT_SEED_COLOR, false, name), value, "light {name}");
        }
        for (name, value) in dark {
            assert_eq!(token(DEFAULT_SEED_COLOR, true, name), value, "dark {name}");
        }
        assert_eq!(color_tokens(DEFAULT_SEED_COLOR, false).len(), 48);
        assert_eq!(primary_of(DEFAULT_SEED_COLOR, true), "#ffb779");
        assert_eq!(hex(DEFAULT_SEED_COLOR), "#f28c1b");
    }
}
