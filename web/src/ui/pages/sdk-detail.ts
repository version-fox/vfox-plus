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
 * Everything about one installed plugin: its metadata, its installed versions
 * and the versions that can be installed.
 */

import '@lit-labs/virtualizer';

import { css, html, LitElement, nothing, type TemplateResult } from 'lit';
import { customElement, property, state } from 'lit/decorators.js';

import { formatAddition, type AvailableVersion, type InstalledSdk } from '../../core/models';
import type { SdkDetailController, VersionFilter } from '../../state/controllers';
import type { VfoxTask } from '../../state/task-queue';
import { app, t, taskTitle } from '../context';
import { askProjectDirectory, confirmAction } from '../components/dialogs';
import { choiceGroup, errorMessage, type Choice } from '../components/parts';
import { icon, iconTemplate } from '../icons';
import { snackbar } from '../snackbar';
import { base, typeScale } from '../styles';
import { Watch } from '../watch';

import '../components/parts';

type Tab = 'installed' | 'available';

let menus = 0;

@customElement('vfox-sdk-detail')
export class SdkDetail extends LitElement {
  static override styles = [
    base,
    typeScale,
    css`
      :host {
        display: block;
        height: 100%;
        overflow: hidden auto;
        scrollbar-gutter: stable;
      }
      .header {
        display: flex;
        align-items: flex-start;
        gap: var(--gap-l);
        padding: var(--gap-xl) var(--gap-l) var(--gap-xl) var(--gap-s);
      }
      .header.narrow {
        flex-direction: column;
        align-items: stretch;
        gap: var(--gap-s);
      }
      .header .identity {
        display: flex;
        align-items: flex-start;
        gap: var(--gap-l);
        flex: 1 1 auto;
        min-width: 0;
      }
      .header .text {
        flex: 1 1 auto;
        min-width: 0;
      }
      .header h1 {
        color: var(--md-sys-color-on-surface);
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
      .tags {
        display: flex;
        flex-wrap: wrap;
        gap: var(--gap-xs) var(--gap-s);
        margin-top: var(--gap-xs);
      }
      .description {
        margin-top: var(--gap-s);
        color: var(--md-sys-color-on-surface-variant);
        display: -webkit-box;
        -webkit-line-clamp: 3;
        -webkit-box-orient: vertical;
        overflow: hidden;
      }
      .actions {
        display: flex;
        align-items: center;
        gap: var(--gap-xs);
        flex: none;
      }
      .narrow .actions .spacer {
        flex: 1 1 auto;
      }
      .tabs {
        padding: 0 var(--gap-xl) var(--gap-l);
      }
      .list {
        padding: 0 var(--gap-xl) var(--gap-xl);
      }
      .toolbar {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--gap-s) var(--gap-m);
        padding: 0 var(--gap-xl) var(--gap-m);
      }
      .toolbar m3e-form-field {
        width: 280px;
        --m3e-form-field-container-color: var(--md-sys-color-surface-container-highest);
      }
      .toolbar .count {
        color: var(--md-sys-color-on-surface-variant);
      }
      .version-actions {
        display: flex;
        align-items: center;
        gap: var(--gap-xs);
      }
      .current-icon {
        color: var(--md-sys-color-primary);
      }
      m3e-list-item {
        --m3e-list-item-container-color: var(--md-sys-color-surface-container);
      }
      lit-virtualizer {
        display: block;
      }
      .available {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--gap-s);
      }
      .version-row {
        /* The virtualizer positions its items absolutely. */
        width: 100%;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--gap-s) var(--gap-l);
        min-height: 64px;
        padding: var(--gap-s) var(--gap-l);
        margin-bottom: var(--gap-hairline);
        background: var(--md-sys-color-surface-container);
        border-radius: var(--corner-extra-small);
        transition: border-radius var(--md-sys-motion-spring-fast-spatial);
      }
      .version-row:hover {
        border-radius: var(--corner-medium);
      }
      .version-row.first {
        border-top-left-radius: var(--corner-large-increased);
        border-top-right-radius: var(--corner-large-increased);
      }
      .version-row.last {
        border-bottom-left-radius: var(--corner-large-increased);
        border-bottom-right-radius: var(--corner-large-increased);
        margin-bottom: 0;
      }
      .version-row .text {
        flex: 1 1 200px;
        min-width: 0;
      }
      .version-row .title {
        color: var(--md-sys-color-on-surface);
      }
      .version-row .additions {
        color: var(--md-sys-color-on-surface-variant);
      }
      .progress {
        display: flex;
        align-items: center;
        gap: var(--gap-xs);
        width: 220px;
      }
      .progress .bar {
        flex: 1 1 auto;
        display: flex;
        flex-direction: column;
        gap: var(--gap-xs);
        color: var(--md-sys-color-on-surface-variant);
      }
      m3e-linear-progress-indicator {
        width: 100%;
      }
      m3e-loading-indicator.small {
        --m3e-loading-indicator-size: 40px;
      }
      .danger {
        color: var(--md-sys-color-error);
      }
    `,
  ];

  /** The plugin name. */
  @property() name = '';
  /** Whether to show a back button, which fires `back`. */
  @property({ type: Boolean }) back = false;
  @state() private tab: Tab = 'installed';
  @state() private width = 1000;
  private controller: SdkDetailController | null = null;
  private stopTasks: (() => void) | null = null;
  private readonly menuId = `sdk-menu-${++menus}`;
  private readonly resize = new ResizeObserver((entries) => {
    this.width = entries[0]?.contentRect.width ?? this.width;
  });

  constructor() {
    super();
    const { model, locale } = app();
    new Watch(this, () => [model.installed, model.tasks, locale, this.controller, this.controller?.info]);
  }

  override connectedCallback(): void {
    super.connectedCallback();
    const { model } = app();
    this.controller = model.sdkDetail(this.name);
    void this.controller.info.load();
    this.stopTasks = model.tasks.onFinished((task) => {
      if (task.command.sdk === this.name && task.command.operation === 'updatePlugin') {
        void this.controller?.info.load();
      }
    });
    this.resize.observe(this);
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.stopTasks?.();
    this.controller?.dispose();
    this.controller = null;
    this.resize.disconnect();
  }

  private showTab(tab: Tab): void {
    this.tab = tab;
    if (tab === 'available') {
      void this.controller?.versions.ensureLoaded();
    }
  }

  override render(): TemplateResult {
    const controller = this.controller;
    if (controller === null) {
      return html``;
    }
    const sdk: InstalledSdk = app().model.installedSdk(this.name) ?? { name: this.name, versions: [], current: null };
    return html`${this.renderHeader(sdk)}
      <div class="tabs">
        ${choiceGroup<Tab>(
          [
            ['installed', `${t('sdkTabInstalled')} (${sdk.versions.length})`, 'versions'],
            ['available', t('sdkTabAvailable'), 'install'],
          ],
          this.tab,
          (tab) => {
            this.showTab(tab);
          },
        )}
      </div>
      ${this.tab === 'installed' ? this.renderInstalled(sdk) : this.renderAvailable(sdk, controller)}`;
  }

  private renderHeader(sdk: InstalledSdk): TemplateResult {
    const { model, host } = app();
    const info = this.controller?.info.value ?? null;
    const updating = model.tasks.activeTask({ operation: 'updatePlugin', sdk: sdk.name });
    const narrow = this.width < 520;
    const back = this.back
      ? html`<vfox-icon-button
          icon="back"
          .label=${t('actionClose')}
          @click=${() => this.dispatchEvent(new Event('back'))}
        ></vfox-icon-button>`
      : nothing;
    const actions = html`
      ${
        updating === null
          ? html`<vfox-icon-button
              icon="update"
              .label=${t('sdkUpdatePlugin')}
              @click=${() => model.run(model.cli.updatePlugin(sdk.name))}
            ></vfox-icon-button>`
          : html`<m3e-loading-indicator class="small" aria-label=${t('loading')}></m3e-loading-indicator>`
      }
      <vfox-icon-button icon="more" .label=${t('sdkMoreActions')}>
        <m3e-menu-trigger for=${this.menuId}></m3e-menu-trigger>
      </vfox-icon-button>
      <m3e-menu id=${this.menuId} position-x="before">
        ${
          info !== null && info.homepage !== ''
            ? html`<m3e-menu-item @click=${() => void host.openUrl(info.homepage)}>
                ${iconTemplate('openInNew', { slot: 'icon' })}${t('actionOpenHomepage')}
              </m3e-menu-item>`
            : nothing
        }
        <m3e-menu-item class="danger" @click=${() => void this.removePlugin(sdk)}>
          ${iconTemplate('removePlugin', { slot: 'icon' })}${t('sdkRemovePlugin')}
        </m3e-menu-item>
      </m3e-menu>
    `;
    const text = html`<div class="text">
      <h1 class="type-headline-medium-emphasized" role="heading" aria-level="1">${sdk.name}</h1>
      <div class="tags">
        <span class="tag ${sdk.current === null ? '' : 'emphasized'}">
          ${sdk.current === null ? nothing : iconTemplate('current', { filled: true })}
          ${sdk.current === null ? t('sdksNoCurrent') : t('sdksCurrent', { version: sdk.current })}
        </span>
        ${
          info !== null && info.version !== ''
            ? html`<span class="tag">${t('sdkPluginVersion', { version: info.version })}</span>`
            : nothing
        }
      </div>
      ${
        info !== null && info.description !== ''
          ? html`<p class="description type-body-medium">${info.description}</p>`
          : nothing
      }
    </div>`;
    if (narrow) {
      return html`<header class="header narrow">
        <div class="actions">${back}<span class="spacer"></span>${actions}</div>
        <div class="identity"><vfox-avatar .name=${sdk.name} size="56"></vfox-avatar>${text}</div>
      </header>`;
    }
    return html`<header class="header">
      <div class="identity">${back}<vfox-avatar .name=${sdk.name} size="72"></vfox-avatar>${text}</div>
      <div class="actions">${actions}</div>
    </header>`;
  }

  private async removePlugin(sdk: InstalledSdk): Promise<void> {
    const { model, shell } = app();
    const confirmed = await confirmAction({
      title: t('sdkRemoveTitle', { name: sdk.name }),
      body: t('sdkRemoveBody', { count: sdk.versions.length }),
      confirmLabel: t('sdkRemoveConfirm'),
      icon: 'removePlugin',
      destructive: true,
    });
    if (confirmed) {
      model.run(model.cli.removePlugin(sdk.name));
      shell.showSdk(null);
    }
  }

  private renderInstalled(sdk: InstalledSdk): TemplateResult {
    if (sdk.versions.length === 0) {
      return html`<vfox-status icon="install" shape="pill" .heading=${t('sdkInstalledEmpty', { name: sdk.name })}>
        <m3e-button variant="filled" @click=${() => this.showTab('available')}>
          ${iconTemplate('install', { slot: 'icon' })}${t('sdkBrowseVersions')}
        </m3e-button>
      </vfox-status>`;
    }
    return html`<div class="list">
      <m3e-list variant="segmented"
        >${sdk.versions.map((version, index) => this.renderInstalledVersion(sdk, version, index))}</m3e-list
      >
    </div>`;
  }

  private renderInstalledVersion(sdk: InstalledSdk, version: string, index: number): TemplateResult {
    const { model } = app();
    const current = sdk.current === version;
    const task = model.tasks.activeTask({ sdk: sdk.name, version });
    // Identifiers of menus must be valid in CSS selectors, which versions are not.
    const menu = `${this.menuId}-version-${index}`;
    const useMenu = `${menu}-use`;
    let actions: TemplateResult;
    if (task !== null) {
      actions = html`<m3e-loading-indicator
        class="small"
        aria-label=${taskTitle(task.command)}
      ></m3e-loading-indicator>`;
    } else {
      actions = html`<div class="version-actions">
        ${
          current
            ? html`<span class="tag emphasized">${iconTemplate('current', { filled: true })}${t('versionInUse')}</span>`
            : html`<m3e-split-button variant="tonal">
                  <m3e-button slot="leading-button" @click=${() => model.run(model.cli.use(sdk.name, version))}>
                    ${iconTemplate('check', { slot: 'icon' })}${t('versionUse')}
                  </m3e-button>
                  <m3e-icon-button slot="trailing-button" aria-label=${t('versionUseOptions')}>
                    <m3e-icon variant="rounded" name=${icon('expandMore')}></m3e-icon>
                    <m3e-menu-trigger for=${useMenu}></m3e-menu-trigger>
                  </m3e-icon-button>
                </m3e-split-button>
                <m3e-menu id=${useMenu} position-x="before">
                  <m3e-menu-item @click=${() => model.run(model.cli.use(sdk.name, version))}>
                    ${iconTemplate('globalScope', { slot: 'icon' })}${t('versionUseGlobally')}
                  </m3e-menu-item>
                  <m3e-menu-item @click=${() => void this.useInProject(sdk.name, version)}>
                    ${iconTemplate('projectScope', { slot: 'icon' })}${t('versionUseInProject')}
                  </m3e-menu-item>
                </m3e-menu>`
        }
        <vfox-icon-button icon="more" .label=${t('versionMoreActions', { version })}>
          <m3e-menu-trigger for=${menu}></m3e-menu-trigger>
        </vfox-icon-button>
        <m3e-menu id=${menu} position-x="before">
          ${
            current
              ? html`<m3e-menu-item @click=${() => void this.useInProject(sdk.name, version)}>
                    ${iconTemplate('projectScope', { slot: 'icon' })}${t('versionUseInProject')}
                  </m3e-menu-item>
                  <m3e-menu-item @click=${() => model.run(model.cli.unuse(sdk.name))}>
                    ${iconTemplate('cancel', { slot: 'icon' })}${t('versionUnuse')}
                  </m3e-menu-item>`
              : nothing
          }
          <m3e-menu-item @click=${() => void this.copyPath(sdk.name, version)}>
            ${iconTemplate('copy', { slot: 'icon' })}${t('versionCopyPath')}
          </m3e-menu-item>
          <m3e-menu-item class="danger" @click=${() => void this.uninstall(sdk.name, version, current)}>
            ${iconTemplate('uninstall', { slot: 'icon' })}${t('versionUninstall')}
          </m3e-menu-item>
        </m3e-menu>
      </div>`;
    }
    return html`<m3e-list-item>
      <m3e-icon
        slot="leading"
        variant="rounded"
        class=${current ? 'current-icon' : ''}
        name=${icon(current ? 'current' : 'versions')}
        ?filled=${current}
      ></m3e-icon>
      <span class=${current ? 'type-body-large-emphasized' : ''}>${version}</span>
      <div slot="trailing">${actions}</div>
    </m3e-list-item>`;
  }

  private async useInProject(name: string, version: string): Promise<void> {
    const { model } = app();
    const directory = await askProjectDirectory(name, version);
    if (directory !== null) {
      model.run(model.cli.use(name, version, 'project', directory));
    }
  }

  private async copyPath(name: string, version: string): Promise<void> {
    const { model, host } = app();
    const path = await model.cli.installPath(name, version);
    if (path !== null) {
      await host.copyText(path);
      snackbar(t('actionCopied'));
    }
  }

  private async uninstall(name: string, version: string, current: boolean): Promise<void> {
    const { model } = app();
    const confirmed = await confirmAction({
      title: t('versionUninstallTitle', { name, version }),
      body: current ? t('versionUninstallCurrentBody') : t('versionUninstallBody'),
      confirmLabel: t('versionUninstall'),
      icon: 'uninstall',
      destructive: true,
    });
    if (confirmed) {
      model.run(model.cli.uninstall(name, version));
    }
  }

  private renderAvailable(sdk: InstalledSdk, controller: SdkDetailController): TemplateResult {
    const versions = controller.versions;
    if (!versions.hasValue) {
      const error = versions.error;
      return error === null
        ? html`<vfox-loading .label=${t('availableLoading')}></vfox-loading>`
        : errorMessage(t('availableError', { name: sdk.name }), error, () => void versions.load());
    }
    const installed = new Set(sdk.versions);
    const visible = controller.visibleVersions(installed);
    const filters: Choice<VersionFilter>[] = [
      ['all', t('availableFilterAll'), null],
      ['withNote', t('availableFilterTagged'), null],
      ['installed', t('availableFilterInstalled'), null],
    ];
    return html`<div class="toolbar">
        <m3e-form-field variant="outlined" hide-subscript="always">
          ${iconTemplate('search', { slot: 'prefix' })}
          <input
            placeholder=${t('availableSearchHint')}
            aria-label=${t('availableSearchHint')}
            .value=${controller.query}
            @input=${(event: Event) => (controller.query = (event.target as HTMLInputElement).value)}
          />
        </m3e-form-field>
        ${choiceGroup<VersionFilter>(filters, controller.filter, (filter) => {
          controller.filter = filter;
        })}
        <span class="count type-label-large">${t('availableCount', { count: visible.length })}</span>
      </div>
      ${
        visible.length === 0
          ? html`<vfox-status icon="notFound" .heading=${t('availableEmpty')}></vfox-status>`
          : html`<div class="list">
              <lit-virtualizer
                .items=${visible}
                .renderItem=${(version: AvailableVersion, index: number) =>
                  this.renderAvailableVersion(sdk.name, version, installed.has(version.version), index, visible.length)}
              ></lit-virtualizer>
            </div>`
      }`;
  }

  private renderAvailableVersion(
    name: string,
    version: AvailableVersion,
    installed: boolean,
    index: number,
    count: number,
  ): TemplateResult {
    const { model } = app();
    const task = model.tasks.activeTask({ operation: 'install', sdk: name, version: version.version });
    const additions = version.additions.map(formatAddition).join(' · ');
    let trailing: TemplateResult;
    if (task !== null) {
      trailing = html`<vfox-install-progress .task=${task}></vfox-install-progress>`;
    } else if (installed) {
      trailing = html`<m3e-button variant="text" disabled
        >${iconTemplate('check', { slot: 'icon' })}${t('versionInstalled')}</m3e-button
      >`;
    } else {
      trailing = html`<m3e-button variant="tonal" @click=${() => model.run(model.cli.install(name, version.version))}>
        ${iconTemplate('install', { slot: 'icon' })}${t('versionInstall')}
      </m3e-button>`;
    }
    return html`<div class="version-row ${index === 0 ? 'first' : ''} ${index === count - 1 ? 'last' : ''}">
      <div class="text">
        <div class="available">
          <span class="title type-body-large">${version.version}</span>
          ${version.note === null ? nothing : html`<span class="tag emphasized">${version.note}</span>`}
        </div>
        ${additions === '' ? nothing : html`<div class="additions type-body-medium">${additions}</div>`}
      </div>
      ${trailing}
    </div>`;
  }
}

/** The download progress of an installation, with a cancel button. */
@customElement('vfox-install-progress')
export class InstallProgress extends LitElement {
  static override styles = [
    base,
    typeScale,
    css`
      .progress {
        display: flex;
        align-items: center;
        gap: var(--gap-xs);
        width: 220px;
      }
      .bar {
        flex: 1 1 auto;
        display: flex;
        flex-direction: column;
        gap: var(--gap-xs);
        color: var(--md-sys-color-on-surface-variant);
      }
      m3e-linear-progress-indicator {
        width: 100%;
      }
    `,
  ];

  @property({ attribute: false }) task: VfoxTask | null = null;

  constructor() {
    super();
    new Watch(this, () => [this.task, app().locale]);
  }

  override render(): TemplateResult {
    const task = this.task;
    if (task === null) {
      return html``;
    }
    const progress = task.progress;
    const label =
      progress === null
        ? task.status === 'queued'
          ? t('taskStatusQueued')
          : t('taskStatusRunning')
        : t('taskDownloading', { percent: Math.round(progress * 100) });
    return html`<div class="progress">
      <div class="bar">
        <m3e-linear-progress-indicator
          variant="wavy"
          mode=${progress === null ? 'indeterminate' : 'determinate'}
          value=${Math.round((progress ?? 0) * 100)}
          aria-label=${taskTitle(task.command)}
        ></m3e-linear-progress-indicator>
        <span class="type-label-small">${label}</span>
      </div>
      <vfox-icon-button
        icon="cancel"
        .label=${t('actionCancel')}
        @click=${() => app().model.tasks.cancel(task)}
      ></vfox-icon-button>
    </div>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-sdk-detail': SdkDetail;
    'vfox-install-progress': InstallProgress;
  }
}
