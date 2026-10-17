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

/** The plugin registry: every plugin that `vfox add` can install. */

import { css, html, LitElement, nothing, type TemplateResult } from 'lit';
import { customElement, state } from 'lit/decorators.js';

import type { AvailablePlugin } from '../../core/models';
import type { VfoxTask } from '../../state/task-queue';
import { app, t } from '../context';
import { askCustomPlugin } from '../components/dialogs';
import { choiceGroup, errorMessage, type Choice } from '../components/parts';
import { iconTemplate } from '../icons';
import { base, page, typeScale } from '../styles';
import { Watch } from '../watch';

import '../components/parts';

type PluginFilter = 'all' | 'official' | 'community';

/** The repository path of a homepage, such as `version-fox/vfox-nodejs`. */
export function repositoryPath(homepage: string): string {
  try {
    const segments = new URL(homepage).pathname.split('/').filter((segment) => segment !== '');
    return segments.length === 0 ? homepage : segments.join('/');
  } catch {
    return homepage;
  }
}

@customElement('vfox-plugins-page')
export class PluginsPage extends LitElement {
  static override styles = [
    base,
    typeScale,
    page,
    css`
      .filters {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--gap-m);
        padding: 0 var(--gap-xl) var(--gap-l);
      }
      m3e-search-bar {
        flex: 0 1 420px;
        min-width: 240px;
        --m3e-search-bar-container-color: var(--md-sys-color-surface-container-high);
      }
      m3e-search-bar input {
        font: inherit;
      }
      .grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
        gap: var(--gap-m);
        padding: 0 var(--gap-xl) var(--gap-xl);
      }
      .card {
        display: flex;
        flex-direction: column;
        height: 152px;
        padding: var(--gap-l);
        border-radius: var(--corner-large-increased);
        background: var(--md-sys-color-surface-container);
      }
      .card.installed {
        background: var(--md-sys-color-surface-container-highest);
      }
      .identity {
        display: flex;
        align-items: center;
        gap: var(--gap-m);
        min-width: 0;
      }
      .names {
        flex: 1 1 auto;
        min-width: 0;
      }
      .name {
        display: flex;
        align-items: center;
        gap: var(--gap-xs);
        color: var(--md-sys-color-on-surface);
      }
      .verified {
        --m3e-icon-size: 18px;
        color: var(--md-sys-color-primary);
      }
      .source {
        color: var(--md-sys-color-on-surface-variant);
      }
      .bottom {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-top: auto;
      }
      m3e-loading-indicator {
        --m3e-loading-indicator-size: 40px;
      }
    `,
  ];

  @state() private query = '';
  @state() private filter: PluginFilter = 'all';

  constructor() {
    super();
    const { model, locale } = app();
    new Watch(this, () => [model.registry, model.installed, model.tasks, locale]);
  }

  override connectedCallback(): void {
    super.connectedCallback();
    void app().model.registry.ensureLoaded();
  }

  private async addCustom(): Promise<void> {
    const source = await askCustomPlugin();
    if (source !== null) {
      await app().model.addPluginFromSource(source.source, source.alias);
    }
  }

  override render(): TemplateResult {
    const { model } = app();
    const plugins = model.registry.value;
    const installed = new Set((model.installed.value ?? []).map((sdk) => sdk.name));
    const query = this.query.trim().toLowerCase();
    const visible = (plugins ?? []).filter(
      (plugin) =>
        (this.filter === 'all' || (this.filter === 'official') === plugin.official) &&
        (query === '' || plugin.name.toLowerCase().includes(query) || plugin.homepage.includes(query)),
    );
    const updatingAll = model.tasks.activeTask({ operation: 'updateAllPlugins' });
    const filters: Choice<PluginFilter>[] = [
      ['all', t('pluginsFilterAll'), null],
      ['official', t('pluginsFilterOfficial'), 'verified'],
      ['community', t('pluginsFilterCommunity'), null],
    ];

    let body: TemplateResult;
    if (plugins === null) {
      const error = model.registry.error;
      body =
        error === null
          ? html`<vfox-loading></vfox-loading>`
          : errorMessage(t('pluginsLoadError'), error, () => void model.registry.load());
    } else if (visible.length === 0) {
      body = html`<vfox-status icon="notFound" .heading=${t('pluginsNoMatch')}></vfox-status>`;
    } else {
      body = html`<div class="grid">
        ${visible.map((plugin) =>
          this.renderCard(
            plugin,
            installed.has(plugin.name),
            model.tasks.activeTask({ operation: 'addPlugin', sdk: plugin.name }),
          ),
        )}
      </div>`;
    }

    return html`<div class="pane">
      <header class="page-header">
        <div class="titles">
          <h1 class="type-headline-medium-emphasized" role="heading" aria-level="1">${t('pluginsTitle')}</h1>
          <p class="subtitle type-body-medium">${t('pluginsSubtitle')}</p>
        </div>
        <div class="actions">
          ${
            updatingAll === null
              ? html`<vfox-icon-button
                  icon="update"
                  .label=${t('pluginsUpdateAll')}
                  ?disabled=${installed.size === 0}
                  @click=${() => model.run(model.cli.updateAllPlugins())}
                ></vfox-icon-button>`
              : html`<m3e-loading-indicator aria-label=${t('pluginsUpdateAll')}></m3e-loading-indicator>`
          }
          <vfox-icon-button
            icon="refresh"
            .label=${t('actionRefresh')}
            ?disabled=${model.registry.isLoading}
            @click=${() => void model.registry.load()}
          ></vfox-icon-button>
          <m3e-button variant="tonal" @click=${() => void this.addCustom()}>
            ${iconTemplate('add', { slot: 'icon' })}${t('pluginsAddCustom')}
          </m3e-button>
        </div>
      </header>
      <div class="filters">
        <m3e-search-bar clearable .clearLabel=${t('actionClose')} @clear=${() => (this.query = '')}>
          ${iconTemplate('search', { slot: 'leading' })}
          <input
            slot="input"
            placeholder=${t('pluginsSearchHint')}
            aria-label=${t('pluginsSearchHint')}
            .value=${this.query}
            @input=${(event: Event) => (this.query = (event.target as HTMLInputElement).value)}
          />
        </m3e-search-bar>
        ${choiceGroup<PluginFilter>(filters, this.filter, (filter) => {
          this.filter = filter;
        })}
      </div>
      ${body}
    </div>`;
  }

  private renderCard(plugin: AvailablePlugin, installed: boolean, task: VfoxTask | null): TemplateResult {
    const { model, shell, host } = app();
    let action: TemplateResult;
    if (task !== null) {
      action = html`<m3e-loading-indicator aria-label=${t('loading')}></m3e-loading-indicator>`;
    } else if (installed) {
      action = html`<m3e-button
        variant="text"
        @click=${() => {
          shell.showSdk(plugin.name);
        }}
        >${iconTemplate('check', { slot: 'icon' })}${t('pluginsAdded')}</m3e-button
      >`;
    } else {
      action = html`<m3e-button variant="tonal" @click=${() => model.run(model.cli.addPlugin(plugin.name))}>
        ${iconTemplate('add', { slot: 'icon' })}${t('pluginsAdd')}
      </m3e-button>`;
    }
    const official = `official-${plugin.name}`;
    return html`<article class="card ${installed ? 'installed' : ''}">
      <div class="identity">
        <vfox-avatar .name=${plugin.name} size="44"></vfox-avatar>
        <div class="names">
          <div class="name">
            <span class="type-title-medium-emphasized ellipsis">${plugin.name}</span>
            ${
              plugin.official
                ? html`<m3e-icon
                      id=${official}
                      class="verified"
                      variant="rounded"
                      filled
                      name="verified"
                      aria-label=${t('pluginsOfficial')}
                    ></m3e-icon>
                    <m3e-tooltip for=${official}>${t('pluginsOfficial')}</m3e-tooltip>`
                : nothing
            }
          </div>
          <div class="source type-body-small ellipsis">${repositoryPath(plugin.homepage)}</div>
        </div>
      </div>
      <div class="bottom">
        <vfox-icon-button
          icon="openInNew"
          .label=${t('actionOpenHomepage')}
          @click=${() => void host.openUrl(plugin.homepage)}
        ></vfox-icon-button>
        ${action}
      </div>
    </article>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-plugins-page': PluginsPage;
  }
}
