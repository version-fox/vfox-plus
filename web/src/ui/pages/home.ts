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

/** The overview: what is installed, what is in use and where to go next. */

import { css, html, LitElement, nothing, type TemplateResult } from 'lit';
import { customElement } from 'lit/decorators.js';

import { app, LINKS, t } from '../context';
import { errorMessage, type ShapeName } from '../components/parts';
import { iconTemplate, type IconName } from '../icons';
import { snackbar } from '../snackbar';
import { base, page, typeScale } from '../styles';
import { Watch } from '../watch';

import '../components/parts';

/** The line that activates vfox in Bash. */
export const ACTIVATION_COMMAND = 'eval "$(vfox activate bash)"';

interface Tile {
  readonly value: number;
  readonly label: string;
  readonly icon: IconName;
  readonly shape: ShapeName;
  readonly role: 'primary' | 'secondary' | 'tertiary';
}

@customElement('vfox-home-page')
export class HomePage extends LitElement {
  static override styles = [
    base,
    typeScale,
    page,
    css`
      .content {
        max-width: 1080px;
      }
      .stats {
        display: grid;
        grid-template-columns: repeat(3, minmax(0, 1fr));
        gap: var(--gap-m);
      }
      .tile {
        position: relative;
        overflow: hidden;
        height: 136px;
        padding: var(--gap-xl);
        border: none;
        border-radius: var(--corner-extra-large);
        background: var(--tile-container);
        color: var(--tile-on-container);
        text-align: start;
        font: inherit;
        cursor: pointer;
        display: flex;
        flex-direction: column;
        align-items: flex-start;
        transition: border-radius var(--md-sys-motion-spring-fast-spatial);
      }
      .tile::after {
        content: '';
        position: absolute;
        inset: 0;
        background: currentColor;
        opacity: 0;
        transition: opacity var(--md-sys-motion-spring-fast-effects);
      }
      .tile:hover::after {
        opacity: 0.08;
      }
      .tile:active {
        border-radius: var(--corner-large);
      }
      .tile:active::after {
        opacity: 0.1;
      }
      .tile:focus-visible {
        outline: 3px solid var(--md-sys-color-secondary);
        outline-offset: 2px;
      }
      .tile m3e-shape {
        position: absolute;
        inset-inline-end: -28px;
        bottom: -36px;
        --m3e-shape-size: 140px;
        --m3e-shape-container-color: color-mix(in srgb, currentColor 8%, transparent);
      }
      .tile .value {
        margin-top: auto;
        line-height: 1;
      }
      .tile .label {
        margin-top: var(--gap-xs);
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
      }
      .tile > * {
        position: relative;
      }
      .compact .stats {
        gap: var(--gap-s);
      }
      .compact .tile {
        padding: var(--gap-l);
      }
      .shortcuts {
        display: grid;
        grid-template-columns: 3fr 2fr;
        grid-template-rows: 1fr 1fr;
        gap: var(--gap-m);
        margin-top: var(--gap-xl);
      }
      .shortcuts .shell-card {
        grid-row: span 2;
      }
      .narrow .shortcuts {
        grid-template-columns: 1fr;
        grid-template-rows: none;
      }
      .narrow .shortcuts .shell-card {
        grid-row: auto;
      }
      .shell-card {
        padding: var(--gap-xl);
        border-radius: var(--corner-large);
        background: var(--md-sys-color-surface-container-highest);
      }
      .shell-card .heading {
        display: flex;
        align-items: center;
        gap: var(--gap-m);
        color: var(--md-sys-color-on-surface);
      }
      .shell-card .heading m3e-icon {
        color: var(--md-sys-color-primary);
      }
      .shell-card p {
        margin: var(--gap-s) 0 var(--gap-m);
      }
      .command {
        display: flex;
        align-items: center;
        gap: var(--gap-s);
        padding-inline: var(--gap-l) var(--gap-xs);
        min-height: 48px;
        border-radius: var(--corner-medium);
        background: var(--md-sys-color-surface-container-lowest);
      }
      .command code {
        flex: 1 1 auto;
        overflow-x: auto;
        white-space: nowrap;
        user-select: text;
        cursor: text;
      }
      .shortcut {
        display: flex;
        align-items: center;
        gap: var(--gap-l);
        padding: var(--gap-l);
        border: none;
        border-radius: var(--corner-large);
        background: var(--md-sys-color-surface-container);
        color: var(--md-sys-color-on-surface);
        font: inherit;
        text-align: start;
        cursor: pointer;
        position: relative;
        overflow: hidden;
        transition: border-radius var(--md-sys-motion-spring-fast-spatial);
      }
      .shortcut::after {
        content: '';
        position: absolute;
        inset: 0;
        background: var(--md-sys-color-on-surface);
        opacity: 0;
        transition: opacity var(--md-sys-motion-spring-fast-effects);
      }
      .shortcut:hover::after {
        opacity: 0.08;
      }
      .shortcut:active {
        border-radius: var(--corner-small);
      }
      .shortcut:focus-visible {
        outline: 3px solid var(--md-sys-color-secondary);
        outline-offset: 2px;
      }
      .shortcut .shape {
        position: relative;
        flex: none;
        width: 48px;
        height: 48px;
        display: grid;
        place-items: center;
        color: var(--md-sys-color-on-secondary-container);
      }
      .shortcut .shape m3e-shape {
        position: absolute;
        inset: 0;
        --m3e-shape-size: 48px;
        --m3e-shape-container-color: var(--md-sys-color-secondary-container);
      }
      .shortcut .shape m3e-icon {
        position: relative;
      }
      .shortcut .text {
        flex: 1 1 auto;
        min-width: 0;
      }
      .shortcut .text p {
        margin-top: var(--gap-xs);
      }
      .shortcut > m3e-icon {
        color: var(--md-sys-color-on-surface-variant);
      }
    `,
  ];

  private width = 1080;
  private readonly resize = new ResizeObserver((entries) => {
    const width = entries[0]?.contentRect.width ?? this.width;
    if (Math.abs(width - this.width) > 1) {
      this.width = width;
      this.requestUpdate();
    }
  });

  constructor() {
    super();
    const { model, locale } = app();
    new Watch(this, () => [model.installed, model.vfox, locale]);
  }

  override connectedCallback(): void {
    super.connectedCallback();
    this.resize.observe(this);
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.resize.disconnect();
  }

  private async copyCommand(): Promise<void> {
    await app().host.copyText(ACTIVATION_COMMAND);
    snackbar(t('actionCopied'));
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
    const active = sdks.filter((sdk) => sdk.current !== null);
    const versions = sdks.reduce((sum, sdk) => sum + sdk.versions.length, 0);
    const subtitle = [t('homeSubtitle', { version: model.vfox.version ?? '?' }), model.vfox.executable]
      .filter((part) => part !== null)
      .join(' · ');
    const tiles: Tile[] = [
      {
        value: sdks.length,
        label: t('homeStatPlugins', { count: sdks.length }),
        icon: 'plugins',
        shape: '9-sided-cookie',
        role: 'primary',
      },
      {
        value: versions,
        label: t('homeStatVersions', { count: versions }),
        icon: 'versions',
        shape: '4-leaf-clover',
        role: 'secondary',
      },
      { value: active.length, label: t('homeStatActive'), icon: 'current', shape: 'sunny', role: 'tertiary' },
    ];
    // Narrow windows keep the three tiles in a row, in a compact size.
    const compact = this.width < 560 + 48;
    const narrow = this.width < 760 + 48;

    return html`<div class="pane">
      <header class="page-header">
        <div class="titles">
          <h1 class="type-headline-medium-emphasized" role="heading" aria-level="1">${t('appTitle')}</h1>
          <p class="subtitle type-body-medium">${subtitle}</p>
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
      <div class="content ${compact ? 'compact' : ''} ${narrow ? 'narrow' : ''}">
        <div class="stats">
          ${tiles.map(
            (tile) =>
              html`<button
                class="tile"
                style="--tile-container: var(--md-sys-color-${tile.role}-container); --tile-on-container: var(--md-sys-color-on-${tile.role}-container)"
                @click=${() => {
                  shell.showSdk(null);
                }}
              >
                <m3e-shape name=${tile.shape} aria-hidden="true"></m3e-shape>
                ${iconTemplate(tile.icon, { filled: true })}
                <span class="value ${compact ? 'type-headline-large-emphasized' : 'type-display-small-emphasized'}"
                  >${tile.value}</span
                >
                <span class="label type-label-large-emphasized">${tile.label}</span>
              </button>`,
          )}
        </div>
        <h2 class="group-header" role="heading" aria-level="2">${t('homeActiveTitle')}</h2>
        <m3e-action-list variant="segmented">
          ${
            active.length === 0
              ? html`<m3e-list-action
                  @click=${() => {
                    shell.showSdk(null);
                  }}
                >
                  ${iconTemplate('info', { slot: 'leading' })}${t('homeActiveEmpty')}
                </m3e-list-action>`
              : active.map(
                  (sdk) =>
                    html`<m3e-list-action
                      @click=${() => {
                        shell.showSdk(sdk.name);
                      }}
                    >
                      <vfox-avatar slot="leading" .name=${sdk.name}></vfox-avatar>
                      ${sdk.name}
                      <span slot="supporting-text">${sdk.current}</span>
                      ${iconTemplate('chevronRight', { slot: 'trailing' })}
                    </m3e-list-action>`,
                )
          }
        </m3e-action-list>
        <div class="shortcuts">
          <section class="shell-card">
            <div class="heading">
              ${iconTemplate('terminal')}
              <h2 class="type-title-medium-emphasized">${t('homeShellTitle')}</h2>
            </div>
            <p class="type-body-medium muted">${t('homeShellBody')}</p>
            <div class="command">
              <code class="mono type-body-medium">${ACTIVATION_COMMAND}</code>
              <vfox-icon-button
                icon="copy"
                .label=${t('actionCopy')}
                @click=${() => void this.copyCommand()}
              ></vfox-icon-button>
            </div>
          </section>
          ${this.shortcut('explore', t('homeExploreTitle'), t('homeExploreBody'), 'chevronRight', () => {
            shell.go('plugins');
          })}
          ${this.shortcut('docs', t('homeDocsTitle'), t('homeDocsBody'), 'openInNew', () => {
            void app().host.openUrl(LINKS.vfoxDocs);
          })}
        </div>
      </div>
    </div>`;
  }

  private shortcut(
    iconName: IconName,
    title: string,
    body: string,
    trailing: IconName,
    activate: () => void,
  ): TemplateResult {
    return html`<button class="shortcut" @click=${activate}>
      <span class="shape" aria-hidden="true"
        ><m3e-shape name="6-sided-cookie"></m3e-shape>${iconTemplate(iconName)}</span
      >
      <span class="text">
        <span class="type-title-small-emphasized">${title}</span>
        <p class="type-body-small muted">${body}</p>
      </span>
      ${iconTemplate(trailing)}${nothing}
    </button>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-home-page': HomePage;
  }
}
