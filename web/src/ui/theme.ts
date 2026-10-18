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

/**
 * Colour and motion of the Material 3 Expressive theme
 * (`docs/design-system.md`).
 *
 * The colour scheme is the tonal spot scheme of the Material colour utilities
 * for the seed colour, in the 2021 specification that the other
 * implementations use. It is written to the root element, where it overrides
 * the scheme of `m3e-theme` and reaches every component, including the
 * snack bars that `@m3e/web` adds to the document body.
 */

import { Hct, hexFromArgb, MaterialDynamicColors, SchemeTonalSpot } from '@material/material-color-utilities';

import { DEFAULT_SEED_COLOR } from '../core/settings/app-settings';
import { EXPRESSIVE, motionTokens } from './motion';

/** Seed colours offered when the desktop accent colour is not used. */
export const SEED_COLORS: readonly number[] = [
  DEFAULT_SEED_COLOR,
  0xffd9473b,
  0xffc2407a,
  0xff7e57c2,
  0xff3f6fd8,
  0xff0b8a9a,
  0xff3e8e41,
  0xff8d7b1e,
];

/** The `--md-sys-color-*` custom properties of the scheme for `seed`. */
export function colorTokens(seed: number, dark: boolean): Record<string, string> {
  const scheme = new SchemeTonalSpot(Hct.fromInt(seed >>> 0), dark, 0);
  const colors = new MaterialDynamicColors();
  const tokens: Record<string, string> = {};
  for (const color of [...colors.allColors, colors.surfaceVariant(), colors.shadow(), colors.scrim()]) {
    tokens[`--md-sys-color-${color.name.replace(/_/g, '-').toLowerCase()}`] = hexFromArgb(color.getArgb(scheme));
  }
  return tokens;
}

/** The primary colour of the scheme for `seed`, as `#rrggbb`. */
export function primaryOf(seed: number, dark: boolean): string {
  return colorTokens(seed, dark)['--md-sys-color-primary'] ?? '#000000';
}

/** `#rrggbb` of an ARGB colour. */
export function hex(argb: number): string {
  return hexFromArgb(argb >>> 0);
}

/**
 * Applies the colours of `seed` in light or dark, and the springs of the
 * expressive motion scheme, to the document.
 */
export function applyTheme(seed: number, dark: boolean, reducedMotion: boolean): void {
  const style = document.documentElement.style;
  for (const [name, value] of Object.entries({
    ...colorTokens(seed, dark),
    ...motionTokens(EXPRESSIVE, reducedMotion),
  })) {
    style.setProperty(name, value);
  }
  style.colorScheme = dark ? 'dark' : 'light';
}
