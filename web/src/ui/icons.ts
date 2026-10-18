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
 * The Material Symbols of the user interface, by meaning, read from
 * `shared/icons.json`. The icon font (`npm run icons`) contains exactly these
 * symbols and their ligatures, so `<m3e-icon name=${icon('home')}>` draws the
 * symbol `home`.
 */

import { html, type TemplateResult } from 'lit';
import { ifDefined } from 'lit/directives/if-defined.js';

import manifest from '@shared/icons.json';

/** The meanings of the icons, such as `sdks`. */
export type IconName = keyof typeof manifest.icons;

/** The Material Symbol name of the icon for `meaning`. */
export function icon(meaning: IconName): string {
  return manifest.icons[meaning];
}

/** An `m3e-icon` for `meaning`, filled when `filled` is set. */
export function iconTemplate(
  meaning: IconName,
  { filled = false, slot }: { filled?: boolean; slot?: string } = {},
): TemplateResult {
  return html`<m3e-icon variant="rounded" name=${icon(meaning)} ?filled=${filled} slot=${ifDefined(slot)}></m3e-icon>`;
}
