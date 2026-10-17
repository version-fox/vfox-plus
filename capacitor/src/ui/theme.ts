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
 * The Material 3 Expressive theme for Ionic. The colour scheme and the
 * springs are those of the web front end (../web/src/ui/theme.ts and
 * motion.ts); Ionic's own colours, the text and background colours and the
 * steps between them, are derived from the scheme, so that the components
 * that keep Ionic's styles use the colours of the scheme as well.
 */

import { EXPRESSIVE, motionTokens } from '@web/ui/motion';
import { colorTokens } from '@web/ui/theme';

/** `#rrggbb` as the `r, g, b` triple of Ionic's `-rgb` properties. */
export function rgbOf(color: string): string {
  const value = Number.parseInt(color.slice(1), 16);
  return `${(value >> 16) & 0xff}, ${(value >> 8) & 0xff}, ${value & 0xff}`;
}

/** `#rrggbb` of `from` mixed with `amount` (0–1) of `to`. */
export function mix(from: string, to: string, amount: number): string {
  const a = rgbOf(from).split(', ').map(Number);
  const b = rgbOf(to).split(', ').map(Number);
  return `#${a
    .map((channel, i) =>
      Math.round(channel + ((b[i] ?? channel) - channel) * amount)
        .toString(16)
        .padStart(2, '0'),
    )
    .join('')}`;
}

/** The Ionic colour `name` with `base` and the `contrast` on it. */
function ionicColor(name: string, base: string, contrast: string): Record<string, string> {
  return {
    [`--ion-color-${name}`]: base,
    [`--ion-color-${name}-rgb`]: rgbOf(base),
    [`--ion-color-${name}-contrast`]: contrast,
    [`--ion-color-${name}-contrast-rgb`]: rgbOf(contrast),
    // The state layers of Material 3: the content colour over the container.
    [`--ion-color-${name}-shade`]: mix(base, contrast, 0.1),
    [`--ion-color-${name}-tint`]: mix(base, contrast, 0.08),
  };
}

/** The custom properties of Ionic for the Material colour scheme `colors`. */
export function ionicTokens(colors: Readonly<Record<string, string>>): Record<string, string> {
  const color = (role: string): string => colors[`--md-sys-color-${role}`] ?? '#000000';
  const background = color('surface');
  const text = color('on-surface');
  const tokens: Record<string, string> = {
    ...ionicColor('primary', color('primary'), color('on-primary')),
    ...ionicColor('secondary', color('secondary'), color('on-secondary')),
    ...ionicColor('tertiary', color('tertiary'), color('on-tertiary')),
    ...ionicColor('danger', color('error'), color('on-error')),
    // Tonal buttons (`color="tonal"`), on the secondary container.
    ...ionicColor('tonal', color('secondary-container'), color('on-secondary-container')),
    '--ion-background-color': background,
    '--ion-background-color-rgb': rgbOf(background),
    '--ion-text-color': text,
    '--ion-text-color-rgb': rgbOf(text),
    '--ion-border-color': color('outline-variant'),
    '--ion-placeholder-color': color('on-surface-variant'),
    '--ion-item-background': color('surface-container'),
    '--ion-backdrop-color': color('scrim'),
    '--ion-overlay-background-color': color('surface-container-high'),
  };
  for (let step = 50; step < 1000; step += 50) {
    tokens[`--ion-background-color-step-${step}`] = mix(background, text, step / 1000);
    tokens[`--ion-text-color-step-${step}`] = mix(text, background, step / 1000);
  }
  return tokens;
}

/**
 * Applies the colours of `seed` in light or dark, for Material 3 and for
 * Ionic, and the springs of the expressive motion scheme, to the document.
 */
export function applyTheme(seed: number, dark: boolean, reducedMotion: boolean): void {
  const colors = colorTokens(seed, dark);
  const style = document.documentElement.style;
  for (const [name, value] of Object.entries({
    ...colors,
    ...ionicTokens(colors),
    ...motionTokens(EXPRESSIVE, reducedMotion),
  })) {
    style.setProperty(name, value);
  }
  style.colorScheme = dark ? 'dark' : 'light';
}
