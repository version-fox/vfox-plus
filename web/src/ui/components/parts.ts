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

/** Small building blocks shared by the pages. */

import { css, html, LitElement, nothing, type TemplateResult } from 'lit';
import { customElement, property } from 'lit/decorators.js';

import { iconTemplate, type IconName } from '../icons';
import { base, typeScale } from '../styles';
import { describeError, t } from '../context';

/**
 * The shapes of the Material 3 Expressive shape library that look good as
 * decorative containers of small icons and initials, as named by `@m3e/web`.
 */
export const DECORATIVE_SHAPES = [
  '9-sided-cookie',
  '4-leaf-clover',
  'sunny',
  'soft-burst',
  '6-sided-cookie',
  'gem',
  'pentagon',
  'flower',
  'puffy-diamond',
  '4-sided-cookie',
  'arch',
  '8-leaf-clover',
] as const;

/** A shape name of `@m3e/web`. */
export type ShapeName = (typeof DECORATIVE_SHAPES)[number] | 'pill' | 'circle' | '12-sided-cookie';

/**
 * A deterministic, platform-independent string hash (FNV-1a, 32 bit) of the
 * UTF-16 code units of `text`, the same as in the other implementations.
 */
export function stableHash(text: string): number {
  let hash = 0x811c9dc5;
  for (let i = 0; i < text.length; i++) {
    hash ^= text.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193) >>> 0;
  }
  return hash >>> 0;
}

/**
 * A plugin avatar: an initial on a decorative Material 3 Expressive shape.
 *
 * The shape and the colour are derived from the name, so every plugin keeps
 * its look. Hovering spins the shape by 30° on the fast spatial spring.
 */
@customElement('vfox-avatar')
export class Avatar extends LitElement {
  static override styles = [
    base,
    css`
      :host {
        display: inline-flex;
        flex: none;
        --size: 40px;
      }
      .avatar {
        position: relative;
        width: var(--size);
        height: var(--size);
        display: grid;
        place-items: center;
      }
      m3e-shape {
        position: absolute;
        inset: 0;
        --m3e-shape-size: var(--size);
        transition: transform var(--md-sys-motion-spring-fast-spatial);
      }
      .avatar:hover m3e-shape {
        transform: rotate(30deg);
      }
      .initial {
        position: relative;
        font-weight: 700;
        font-size: calc(var(--size) * 0.42);
        line-height: 1;
      }
      m3e-icon {
        position: relative;
        --m3e-icon-size: calc(var(--size) * 0.5);
      }
    `,
  ];

  /** The plugin name the avatar represents. */
  @property() name = '';
  /** Width and height, in pixels. */
  @property({ type: Number }) size = 40;
  /** An icon to show instead of the initial. */
  @property() icon: IconName | null = null;

  override render(): TemplateResult {
    const hash = stableHash(this.name);
    // Selected list items use the secondary container colour. Avatars avoid it,
    // and the primary container too, which some seeds (the vfox orange among
    // them) make just as light, so that they stay visible on selected items.
    const [background, foreground] = [
      ['primary-fixed-dim', 'on-primary-fixed'],
      ['secondary', 'on-secondary'],
      ['tertiary-container', 'on-tertiary-container'],
    ][hash % 3] ?? ['primary-fixed-dim', 'on-primary-fixed'];
    const first = new Intl.Segmenter().segment(this.name)[Symbol.iterator]().next().value?.segment;
    const initial = (first ?? '?').toUpperCase();
    return html`<div
      class="avatar"
      style="--size: ${this.size}px; color: var(--md-sys-color-${foreground})"
      aria-hidden="true"
    >
      <m3e-shape
        name=${DECORATIVE_SHAPES[hash % DECORATIVE_SHAPES.length] ?? '9-sided-cookie'}
        style="--m3e-shape-container-color: var(--md-sys-color-${background})"
      ></m3e-shape>
      ${this.icon === null ? html`<span class="initial">${initial}</span>` : iconTemplate(this.icon)}
    </div>`;
  }
}

/**
 * A friendly placeholder for empty, unavailable or failed content, with an
 * expressive shape behind its icon. Buttons go into the default slot.
 */
@customElement('vfox-status')
export class StatusMessage extends LitElement {
  static override styles = [
    base,
    typeScale,
    css`
      :host {
        display: grid;
        place-items: center;
        padding: var(--gap-xxl);
        min-height: 100%;
      }
      .status {
        max-width: 440px;
        display: flex;
        flex-direction: column;
        align-items: center;
        text-align: center;
      }
      .shape {
        position: relative;
        width: 112px;
        height: 112px;
        display: grid;
        place-items: center;
        margin-bottom: var(--gap-xl);
      }
      m3e-shape {
        position: absolute;
        inset: 0;
        --m3e-shape-size: 112px;
        --m3e-shape-container-color: var(--md-sys-color-secondary-container);
      }
      m3e-icon {
        position: relative;
        --m3e-icon-size: 48px;
        color: var(--md-sys-color-on-secondary-container);
      }
      :host([error]) m3e-shape {
        --m3e-shape-container-color: var(--md-sys-color-error-container);
      }
      :host([error]) m3e-icon {
        color: var(--md-sys-color-on-error-container);
      }
      h2 {
        color: var(--md-sys-color-on-surface);
      }
      p {
        margin-top: var(--gap-s);
        color: var(--md-sys-color-on-surface-variant);
        white-space: pre-line;
        overflow-wrap: anywhere;
      }
      .actions {
        display: flex;
        flex-wrap: wrap;
        justify-content: center;
        gap: var(--gap-s);
        margin-top: var(--gap-xl);
      }
      .actions:not(:has(*)) {
        display: none;
      }
    `,
  ];

  @property() icon: IconName = 'info';
  @property() shape: ShapeName = '9-sided-cookie';
  @property() heading = '';
  @property() body = '';
  /** Whether to use error colours. */
  @property({ type: Boolean, reflect: true }) error = false;

  override render(): TemplateResult {
    return html`<div class="status" role="status">
      <div class="shape" aria-hidden="true">
        <m3e-shape name=${this.shape}></m3e-shape>
        ${iconTemplate(this.icon, { filled: true })}
      </div>
      <h2 class="type-title-large-emphasized">${this.heading}</h2>
      ${this.body === '' ? nothing : html`<p class="type-body-medium">${this.body}</p>`}
      <div class="actions"><slot></slot></div>
    </div>`;
  }
}

/** A centred loading indicator with an optional label. */
@customElement('vfox-loading')
export class LoadingMessage extends LitElement {
  static override styles = [
    base,
    typeScale,
    css`
      :host {
        display: grid;
        place-items: center;
        padding: var(--gap-xxl);
        min-height: 100%;
      }
      .loading {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: var(--gap-l);
        text-align: center;
        color: var(--md-sys-color-on-surface-variant);
      }
      m3e-loading-indicator {
        --m3e-loading-indicator-size: 64px;
      }
    `,
  ];

  @property() label = '';

  override render(): TemplateResult {
    return html`<div class="loading" role="progressbar" aria-label=${this.label === '' ? t('loading') : this.label}>
      <m3e-loading-indicator variant="contained"></m3e-loading-indicator>
      ${this.label === '' ? nothing : html`<span class="type-body-medium">${this.label}</span>`}
    </div>`;
  }
}

/** One choice of a {@link choiceGroup}: its value, label and optional icon. */
export type Choice<T> = readonly [value: T, label: string, icon: IconName | null];

/**
 * A Material 3 Expressive connected button group with single selection. The
 * selected button cannot be unselected by clicking it again.
 */
export function choiceGroup<T>(choices: readonly Choice<T>[], selected: T, choose: (value: T) => void): TemplateResult {
  return html`<m3e-button-group variant="connected">
    ${choices.map(
      ([value, label, icon]) =>
        html`<m3e-button
          variant="tonal"
          toggle
          .selected=${value === selected}
          @beforeinput=${(event: Event) => {
            if ((event.currentTarget as HTMLElement & { selected: boolean }).selected) {
              event.preventDefault();
            }
          }}
          @click=${() => {
            if (value !== selected) {
              choose(value);
            }
          }}
          >${icon === null ? nothing : iconTemplate(icon, { slot: 'icon' })}${label}</m3e-button
        >`,
    )}
  </m3e-button-group>`;
}

/** A status message for an error, with the error text and a retry button. */
export function errorMessage(heading: string, error: unknown, retry?: () => void): TemplateResult {
  return html`<vfox-status icon="error" shape="soft-burst" error .heading=${heading} .body=${describeError(error)}>
    ${
      retry === undefined
        ? nothing
        : html`<m3e-button variant="tonal" @click=${retry}
            >${iconTemplate('retry', { slot: 'icon' })}${t('actionRetry')}</m3e-button
          >`
    }
  </vfox-status>`;
}

/**
 * An icon button with a tooltip, which also serves as its accessible name.
 * Every icon-only button of the application is one.
 */
@customElement('vfox-icon-button')
export class IconButton extends LitElement {
  static override styles = css`
    :host {
      display: inline-flex;
    }
  `;

  private static counter = 0;
  private readonly buttonId = `icon-button-${++IconButton.counter}`;

  @property() icon: IconName = 'info';
  @property() label = '';
  @property() variant: 'standard' | 'filled' | 'tonal' | 'outlined' = 'standard';
  @property({ type: Boolean }) disabled = false;
  @property({ type: Boolean }) filled = false;

  override render(): TemplateResult {
    return html`<m3e-icon-button
        id=${this.buttonId}
        variant=${this.variant}
        aria-label=${this.label}
        ?disabled=${this.disabled}
        >${iconTemplate(this.icon, { filled: this.filled })}<slot></slot
      ></m3e-icon-button>
      <m3e-tooltip for=${this.buttonId} show-delay="500">${this.label}</m3e-tooltip>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-avatar': Avatar;
    'vfox-status': StatusMessage;
    'vfox-loading': LoadingMessage;
    'vfox-icon-button': IconButton;
  }
}
