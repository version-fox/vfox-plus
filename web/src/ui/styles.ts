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
 * Styles shared by the elements of the user interface: the type scale with
 * its emphasized variant, the shape and spacing tokens of
 * `docs/design-system.md`, and the page parts (panes, headers, tags).
 */

import { css, unsafeCSS, type CSSResult } from 'lit';

/**
 * The Material 3 type scale: size and line height in pixels, the baseline and
 * the emphasized weight, and the tracking in pixels. The emphasized weights
 * are those of `docs/design-system.md#typography`.
 */
const TYPE_SCALE: Readonly<Record<string, readonly [number, number, number, number, number]>> = {
  'display-large': [57, 64, 400, 500, -0.25],
  'display-medium': [45, 52, 400, 500, 0],
  'display-small': [36, 44, 400, 500, 0],
  'headline-large': [32, 40, 400, 500, 0],
  'headline-medium': [28, 36, 400, 500, 0],
  'headline-small': [24, 32, 400, 500, 0],
  'title-large': [22, 28, 400, 500, 0],
  'title-medium': [16, 24, 500, 700, 0.15],
  'title-small': [14, 20, 500, 700, 0.1],
  'body-large': [16, 24, 400, 500, 0.5],
  'body-medium': [14, 20, 400, 500, 0.25],
  'body-small': [12, 16, 400, 500, 0.4],
  'label-large': [14, 20, 500, 700, 0.1],
  'label-medium': [12, 16, 500, 700, 0.5],
  'label-small': [11, 16, 500, 700, 0.5],
};

function typeClasses(): CSSResult {
  const rules: string[] = [];
  for (const [name, [size, lineHeight, weight, emphasizedWeight, tracking]] of Object.entries(TYPE_SCALE)) {
    for (const emphasized of [false, true]) {
      const token = `--md-sys-typescale-${emphasized ? 'emphasized-' : ''}${name}`;
      rules.push(`.type-${name}${emphasized ? '-emphasized' : ''} {
        font-size: var(${token}-font-size, ${size}px);
        font-weight: var(${token}-font-weight, ${emphasized ? emphasizedWeight : weight});
        line-height: var(${token}-line-height, ${lineHeight}px);
        letter-spacing: var(${token}-tracking, ${tracking}px);
      }`);
    }
  }
  return unsafeCSS(rules.join('\n'));
}

/** The type scale as classes: `.type-title-medium`, `.type-title-medium-emphasized`, … */
export const typeScale = typeClasses();

/** Styles every element starts from. */
export const base = css`
  :host {
    box-sizing: border-box;
    --gap-hairline: 2px;
    --gap-xs: 4px;
    --gap-s: 8px;
    --gap-m: 12px;
    --gap-l: 16px;
    --gap-xl: 24px;
    --gap-xxl: 32px;
    --corner-extra-small: 4px;
    --corner-small: 8px;
    --corner-medium: 12px;
    --corner-large: 16px;
    --corner-large-increased: 20px;
    --corner-extra-large: 28px;
  }
  *,
  *::before,
  *::after {
    box-sizing: inherit;
  }
  [hidden] {
    display: none !important;
  }
  h1,
  h2,
  h3,
  p {
    margin: 0;
  }
  .muted {
    color: var(--md-sys-color-on-surface-variant);
  }
  .mono {
    font-family: 'Ubuntu Mono', 'Noto Sans Mono', monospace;
  }
  .row {
    display: flex;
    align-items: center;
    gap: var(--gap-s);
  }
  .grow {
    flex: 1 1 auto;
    min-width: 0;
  }
  .ellipsis {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  /* A small label such as "In use: 24.11.1" or "LTS". */
  .tag {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 3px 10px;
    border-radius: 9999px;
    background: var(--md-sys-color-surface-container-highest);
    color: var(--md-sys-color-on-surface-variant);
    font-size: var(--md-sys-typescale-label-medium-font-size);
    font-weight: var(--md-sys-typescale-label-medium-font-weight);
    line-height: var(--md-sys-typescale-label-medium-line-height);
    letter-spacing: var(--md-sys-typescale-label-medium-tracking);
    max-width: 100%;
    white-space: nowrap;
  }
  .tag.emphasized {
    background: var(--md-sys-color-tertiary-container);
    color: var(--md-sys-color-on-tertiary-container);
  }
  .tag m3e-icon {
    --m3e-icon-size: 16px;
  }
  /* The heading of a group of list items. */
  .group-header {
    display: flex;
    align-items: center;
    padding: var(--gap-xl) var(--gap-xs) var(--gap-s) var(--gap-l);
    color: var(--md-sys-color-primary);
    font-size: var(--md-sys-typescale-title-small-font-size);
    font-weight: var(--md-sys-typescale-title-small-font-weight);
    line-height: var(--md-sys-typescale-title-small-line-height);
    letter-spacing: var(--md-sys-typescale-title-small-tracking);
  }
  /* Segmented lists: items on their own surfaces, 2 dp apart. */
  m3e-list[variant='segmented'],
  m3e-action-list[variant='segmented'] {
    --m3e-list-item-container-color: var(--md-sys-color-surface-container);
    --m3e-segmented-list-gap: var(--gap-hairline);
    padding: 0;
  }
  /* Hints in the colour of Material 3. */
  m3e-form-field input::placeholder,
  m3e-form-field textarea::placeholder {
    color: var(--md-sys-color-on-surface-variant);
  }
  @media (prefers-reduced-motion: reduce) {
    * {
      transition: none !important;
      animation: none !important;
    }
  }
`;

/** The rounded surface that holds the content of a page, and its header. */
export const page = css`
  :host {
    display: block;
    height: 100%;
    min-height: 0;
  }
  .pane {
    height: 100%;
    overflow: hidden auto;
    background: var(--md-sys-color-surface-container-low);
    border-radius: var(--corner-extra-large);
    scrollbar-gutter: stable;
  }
  .page-header {
    display: flex;
    align-items: center;
    gap: var(--gap-s);
    padding: var(--gap-xl) var(--gap-l) var(--gap-l) var(--gap-xl);
  }
  .page-header .titles {
    flex: 1 1 auto;
    min-width: 0;
  }
  .page-header h1 {
    color: var(--md-sys-color-on-surface);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .page-header .subtitle {
    margin-top: var(--gap-xs);
    color: var(--md-sys-color-on-surface-variant);
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .page-header .actions {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--gap-s);
  }
  .content {
    padding: 0 var(--gap-xl) var(--gap-xl);
    margin: 0 auto;
  }
`;
