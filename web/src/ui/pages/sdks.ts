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
 * Installed SDKs, in the list-detail canonical layout: wide pages show the
 * list and the selected SDK side by side, and the detail fades through when
 * the selection changes; narrow pages show one of them at a time and move
 * between them along the horizontal shared axis.
 */

import { css, html, LitElement, nothing, type TemplateResult } from 'lit';
import { customElement, query, state } from 'lit/decorators.js';
import { keyed } from 'lit/directives/keyed.js';

import type { InstalledSdk } from '../../core/models';
import { app, t } from '../context';
import { errorMessage } from '../components/parts';
import { iconTemplate } from '../icons';
import { EXPRESSIVE, prefersReducedMotion, settleDuration, springEasing } from '../motion';
import { base, page, typeScale } from '../styles';
import { Watch } from '../watch';

import '../components/parts';
import './sdk-detail';

/** The narrowest page that shows list and detail side by side. */
export const TWO_PANE_WIDTH = 840;

@customElement('vfox-sdks-page')
export class SdksPage extends LitElement {
  static override styles = [
    base,
    typeScale,
    page,
    css`
      .layout {
        display: flex;
        gap: var(--gap-xl);
        height: 100%;
      }
      .list-pane {
        flex: none;
        width: 360px;
      }
      .detail-pane {
        flex: 1 1 auto;
        min-width: 0;
      }
      .single {
        height: 100%;
      }
      .search {
        padding: 0 var(--gap-l) var(--gap-m);
      }
      m3e-search-bar {
        --m3e-search-bar-container-color: var(--md-sys-color-surface-container-high);
      }
      m3e-search-bar input {
        font: inherit;
      }
      .items {
        padding: 0 var(--gap-l) var(--gap-l);
      }
      .two-lines {
        white-space: pre-line;
      }
      m3e-list-action[selected] {
        --m3e-list-item-container-color: var(--md-sys-color-secondary-container);
        --m3e-segmented-list-item-container-color: var(--md-sys-color-secondary-container);
        --m3e-list-item-label-text-color: var(--md-sys-color-on-secondary-container);
        --m3e-list-item-supporting-text-color: var(--md-sys-color-on-secondary-container);
        --m3e-segmented-list-item-container-shape: var(--corner-medium);
      }
    `,
  ];

  @state() private query = '';
  @state() private width = 0;
  @query('.detail-host') private detailHost!: HTMLElement | null;
  private shownSelection: string | null = null;
  private narrowShown: 'list' | 'detail' | null = null;
  private readonly resize = new ResizeObserver((entries) => {
    this.width = entries[0]?.contentRect.width ?? this.width;
  });

  constructor() {
    super();
    const { model, shell, locale } = app();
    new Watch(this, () => [model.installed, shell, locale]);
  }

  override connectedCallback(): void {
    super.connectedCallback();
    this.resize.observe(this);
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.resize.disconnect();
  }

  /** The selected SDK, if it is installed. */
  private selected(sdks: readonly InstalledSdk[]): string | null {
    const name = app().shell.selectedSdk;
    return sdks.some((sdk) => sdk.name === name) ? name : null;
  }

  protected override updated(): void {
    const sdks = app().model.installed.value;
    if (sdks === null || this.width === 0) {
      return;
    }
    const selected = this.selected(sdks);
    const duration = prefersReducedMotion() ? 0 : settleDuration(EXPRESSIVE.slowEffects);
    const easing = springEasing(EXPRESSIVE.slowEffects);
    if (this.width >= TWO_PANE_WIDTH) {
      this.narrowShown = null;
      if (selected !== this.shownSelection) {
        // Fade through between SDKs in the detail pane.
        this.detailHost?.animate(
          [
            { opacity: 0, transform: 'scale(0.92)' },
            { opacity: 1, transform: 'none' },
          ],
          { duration: duration * 0.7, delay: duration * 0.3, easing, fill: 'backwards' },
        );
      }
    } else {
      const shown = selected === null ? 'list' : 'detail';
      if (this.narrowShown !== null && shown !== this.narrowShown) {
        // Shared axis, horizontal: forwards into the detail, backwards out of it.
        const direction = shown === 'detail' ? 1 : -1;
        this.renderRoot.querySelector('.single')?.animate(
          [
            { opacity: 0, transform: `translateX(${30 * direction}px)` },
            { opacity: 1, transform: 'none' },
          ],
          { duration, easing },
        );
      }
      this.narrowShown = shown;
    }
    this.shownSelection = selected;
  }

  override render(): TemplateResult {
    const { model, shell } = app();
    const sdks = model.installed.value;
    if (sdks === null) {
      const error = model.installed.error;
      return html`<div class="pane">
        ${
          error === null
            ? html`<vfox-loading></vfox-loading>`
            : errorMessage(t('sdksLoadError'), error, () => void model.installed.load())
        }
      </div>`;
    }
    const selected = this.selected(sdks);
    if (this.width >= TWO_PANE_WIDTH || this.width === 0) {
      return html`<div class="layout">
        <div class="pane list-pane">${this.renderList(sdks, selected, false)}</div>
        <div class="pane detail-pane">
          <div class="detail-host" style="height: 100%">
            ${
              selected === null
                ? html`<vfox-status icon="sdks" shape="gem" .heading=${t('sdksSelectPrompt')}></vfox-status>`
                : keyed(selected, html`<vfox-sdk-detail .name=${selected}></vfox-sdk-detail>`)
            }
          </div>
        </div>
      </div>`;
    }
    return html`<div class="single">
      ${
        selected === null
          ? html`<div class="pane">${this.renderList(sdks, null, true)}</div>`
          : keyed(
              selected,
              html`<div class="pane">
                <vfox-sdk-detail
                  .name=${selected}
                  back
                  @back=${() => {
                    shell.showSdk(null);
                  }}
                ></vfox-sdk-detail>
              </div>`,
            )
      }
    </div>`;
  }

  private renderList(sdks: readonly InstalledSdk[], selected: string | null, chevrons: boolean): TemplateResult {
    const { model, shell } = app();
    if (sdks.length === 0) {
      return html`<vfox-status icon="empty" shape="flower" .heading=${t('sdksEmptyTitle')} .body=${t('sdksEmptyBody')}>
        <m3e-button variant="filled" @click=${() => shell.go('plugins')}>
          ${iconTemplate('add', { slot: 'icon' })}${t('navAddPlugin')}
        </m3e-button>
      </vfox-status>`;
    }
    const query = this.query.trim().toLowerCase();
    const visible = sdks.filter((sdk) => query === '' || sdk.name.toLowerCase().includes(query));
    return html`<header class="page-header">
        <div class="titles">
          <h1 class="type-headline-medium-emphasized" role="heading" aria-level="1">${t('navSdks')}</h1>
        </div>
        <div class="actions">
          <vfox-icon-button
            icon="refresh"
            .label=${t('actionRefresh')}
            ?disabled=${model.installed.isLoading}
            @click=${() => void model.installed.load()}
          ></vfox-icon-button>
        </div>
      </header>
      <div class="search">
        <m3e-search-bar clearable .clearLabel=${t('actionClose')} @clear=${() => (this.query = '')}>
          ${iconTemplate('search', { slot: 'leading' })}
          <input
            slot="input"
            placeholder=${t('sdksSearchHint')}
            aria-label=${t('sdksSearchHint')}
            .value=${this.query}
            @input=${(event: Event) => (this.query = (event.target as HTMLInputElement).value)}
          />
        </m3e-search-bar>
      </div>
      ${
        visible.length === 0
          ? html`<vfox-status icon="notFound" .heading=${t('sdksNoMatch', { query: this.query.trim() })}></vfox-status>`
          : html`<div class="items">
              <m3e-action-list variant="segmented">
                ${visible.map(
                  (sdk) =>
                    html`<m3e-list-action
                      ?selected=${sdk.name === selected}
                      aria-current=${sdk.name === selected ? 'true' : 'false'}
                      @click=${() => {
                        shell.showSdk(sdk.name);
                      }}
                    >
                      <vfox-avatar slot="leading" .name=${sdk.name}></vfox-avatar>
                      ${sdk.name}
                      <span slot="supporting-text" class="two-lines"
                        >${sdk.current === null ? t('sdksNoCurrent') : t('sdksCurrent', { version: sdk.current })}
                        ${t('sdksVersionCount', { count: sdk.versions.length })}</span
                      >
                      ${chevrons ? iconTemplate('chevronRight', { slot: 'trailing' }) : nothing}
                    </m3e-list-action>`,
                )}
              </m3e-action-list>
            </div>`
      }`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-sdks-page': SdksPage;
  }
}
