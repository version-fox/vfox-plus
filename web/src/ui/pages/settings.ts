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

/** Settings of the application and of vfox. */

import { css, html, LitElement, nothing, type TemplateResult } from 'lit';
import { customElement, state } from 'lit/decorators.js';

import {
  formatEnvironmentLines,
  parseEnvironmentLines,
  type AppSettings,
  type LanguagePreference,
  type ThemePreference,
} from '../../core/settings/app-settings';
import { APP_VERSION } from '../../version';
import { app, describeError, LINKS, t } from '../context';
import { showText } from '../components/dialogs';
import { choiceGroup, errorMessage } from '../components/parts';
import { iconTemplate, type IconName } from '../icons';
import { snackbar } from '../snackbar';
import { base, page, typeScale } from '../styles';
import { primaryOf, SEED_COLORS, hex } from '../theme';
import { Watch } from '../watch';
import { licenseText } from '../licenses';

import '../components/parts';

/** The configuration keys of vfox that get a dedicated editor. */
export const CONFIG_KEYS = {
  proxyEnable: 'proxy.enable',
  proxyUrl: 'proxy.url',
  registry: 'registry.address',
  storage: 'storage.sdkPath',
  legacyEnable: 'legacyVersionFile.enable',
  legacyStrategy: 'legacyVersionFile.strategy',
  cache: 'cache.availableHookDuration',
  httpTimeout: 'plugin.http.timeout',
  downloadTimeout: 'plugin.http.downloadTimeout',
  gitignore: 'gitignore.enable',
} as const;

const KNOWN_KEYS: ReadonlySet<string> = new Set(Object.values(CONFIG_KEYS));

/** Saves a vfox setting and reports the result in a snack bar. */
async function saveConfig(save: () => Promise<void>): Promise<void> {
  try {
    await save();
    snackbar(t('settingsSaved'));
  } catch (error) {
    snackbar(describeError(error));
  }
}

@customElement('vfox-settings-page')
export class SettingsPage extends LitElement {
  static override styles = [
    base,
    typeScale,
    page,
    css`
      .content {
        max-width: 840px;
        padding-bottom: var(--gap-xxl);
      }
      m3e-list {
        --m3e-segmented-list-item-container-color: var(--md-sys-color-surface-container);
      }
      .group + .group {
        margin-top: var(--gap-l);
      }
      .row-item {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: var(--gap-s) var(--gap-l);
        min-height: 56px;
        padding: var(--gap-m) var(--gap-l);
        margin-bottom: var(--gap-hairline);
        background: var(--md-sys-color-surface-container);
        border-radius: var(--corner-extra-small);
        transition: border-radius var(--md-sys-motion-spring-fast-spatial);
      }
      .row-item:hover {
        border-radius: var(--corner-medium);
      }
      .row-item:first-child {
        border-top-left-radius: var(--corner-large-increased);
        border-top-right-radius: var(--corner-large-increased);
      }
      .row-item:last-child {
        border-bottom-left-radius: var(--corner-large-increased);
        border-bottom-right-radius: var(--corner-large-increased);
        margin-bottom: 0;
      }
      .row-item.clickable {
        cursor: pointer;
      }
      .row-item > m3e-icon,
      .row-item > .icon-space {
        flex: none;
        width: 24px;
        color: var(--md-sys-color-on-surface-variant);
      }
      .row-item .text {
        flex: 1 1 220px;
        min-width: 0;
      }
      .row-item .title {
        color: var(--md-sys-color-on-surface);
      }
      .row-item .supporting {
        color: var(--md-sys-color-on-surface-variant);
        overflow-wrap: anywhere;
      }
      .row-item .trailing {
        display: flex;
        align-items: center;
        gap: var(--gap-xs);
        flex: 0 1 auto;
        margin-inline-start: auto;
      }
      .row-item .wide {
        flex: 1 1 100%;
        padding-inline-start: 40px;
      }
      .swatches {
        display: flex;
        flex-wrap: wrap;
        gap: var(--gap-s);
        margin-top: var(--gap-s);
      }
      .swatch {
        position: relative;
        width: 40px;
        height: 40px;
        padding: 0;
        border: none;
        background: none;
        cursor: pointer;
        display: grid;
        place-items: center;
      }
      .swatch:focus-visible {
        outline: 3px solid var(--md-sys-color-secondary);
        outline-offset: 2px;
        border-radius: 50%;
      }
      .swatch m3e-shape {
        position: absolute;
        inset: 0;
        --m3e-shape-size: 40px;
        --m3e-shape-transition: clip-path var(--md-sys-motion-spring-fast-spatial);
        transition: transform var(--md-sys-motion-spring-fast-spatial);
      }
      .swatch[aria-pressed='true'] m3e-shape {
        transform: rotate(34deg);
      }
      .swatch m3e-icon {
        position: relative;
        --m3e-icon-size: 20px;
        opacity: 0;
        transition: opacity var(--md-sys-motion-spring-fast-effects);
      }
      .swatch[aria-pressed='true'] m3e-icon {
        opacity: 1;
      }
      m3e-form-field {
        width: 300px;
        max-width: 100%;
        --m3e-form-field-container-color: var(--md-sys-color-surface-container-highest);
      }
      textarea {
        font: inherit;
        resize: vertical;
      }
      .environment m3e-form-field {
        width: 100%;
      }
      .environment .save {
        display: flex;
        justify-content: flex-end;
        margin-top: var(--gap-s);
      }
      .help {
        padding: 0 var(--gap-l) var(--gap-m);
        color: var(--md-sys-color-on-surface-variant);
      }
      m3e-loading-indicator.small {
        --m3e-loading-indicator-size: 36px;
      }
    `,
  ];

  @state() private environmentError = '';
  private environmentText: string | null = null;
  /** Values typed into configuration fields and not saved yet, by key. */
  private readonly drafts = new Map<string, string>();

  constructor() {
    super();
    const { model, locale } = app();
    new Watch(this, () => [model.settings, model.vfox, model.vfoxConfig, locale]);
  }

  override connectedCallback(): void {
    super.connectedCallback();
    const { model } = app();
    if (model.vfox.availability === 'ready') {
      void model.vfoxConfig.load();
    }
  }

  private update_(change: (settings: AppSettings) => AppSettings): void {
    void app().model.settings.update(change);
  }

  override render(): TemplateResult {
    return html`<div class="pane">
      <header class="page-header">
        <div class="titles">
          <h1 class="type-headline-medium-emphasized" role="heading" aria-level="1">${t('navSettings')}</h1>
        </div>
      </header>
      <div class="content">
        ${this.renderAppearance()} ${this.renderVfox()} ${this.renderConfiguration()} ${this.renderAbout()}
      </div>
    </div>`;
  }

  private header(title: string): TemplateResult {
    return html`<h2 class="group-header" role="heading" aria-level="2">${title}</h2>`;
  }

  private renderAppearance(): TemplateResult {
    const settings = app().model.settings.settings;
    const dark = document.documentElement.style.colorScheme === 'dark';
    return html`${this.header(t('settingsAppearance'))}
      <div class="group">
        <div class="row-item">
          ${iconTemplate('darkMode')}
          <div class="text"><div class="title type-body-large">${t('settingsTheme')}</div></div>
          <div class="trailing">
            ${choiceGroup<ThemePreference>(
              [
                ['system', t('themeSystem'), 'systemMode'],
                ['light', t('themeLight'), 'lightMode'],
                ['dark', t('themeDark'), 'darkMode'],
              ],
              settings.theme,
              (theme) => {
                this.update_((s) => ({ ...s, theme }));
              },
            )}
          </div>
        </div>
        <div
          class="row-item clickable"
          @click=${() => {
            this.update_((s) => ({ ...s, useSystemColor: !s.useSystemColor }));
          }}
        >
          ${iconTemplate('palette')}
          <div class="text"><div class="title type-body-large">${t('settingsSystemColor')}</div></div>
          <div class="trailing">
            <m3e-switch
              .checked=${settings.useSystemColor}
              aria-label=${t('settingsSystemColor')}
              @click=${(event: Event) => event.stopPropagation()}
              @change=${(event: Event) => {
                const checked = (event.target as HTMLElement & { checked: boolean }).checked;
                this.update_((s) => ({ ...s, useSystemColor: checked }));
              }}
            ></m3e-switch>
          </div>
        </div>
        <div class="row-item">
          <span class="icon-space"></span>
          <div class="text">
            <div class="title type-body-large">${t('settingsColor')}</div>
            <div class="swatches">
              ${SEED_COLORS.map((seed, index) => {
                const selected = !settings.useSystemColor && settings.seedColor === seed;
                const label = t('settingsColorOption', { index: index + 1 });
                return html`<button
                  class="swatch"
                  aria-label=${label}
                  title=${label}
                  aria-pressed=${selected ? 'true' : 'false'}
                  data-seed=${hex(seed)}
                  @click=${() => {
                    this.update_((s) => ({ ...s, seedColor: seed, useSystemColor: false }));
                  }}
                >
                  <m3e-shape
                    name=${selected ? 'flower' : 'circle'}
                    style="--m3e-shape-container-color: ${primaryOf(seed, dark)}"
                  ></m3e-shape>
                  <m3e-icon variant="rounded" name="check" style="color: ${dark ? '#000' : '#fff'}"></m3e-icon>
                </button>`;
              })}
            </div>
          </div>
        </div>
        <div class="row-item">
          ${iconTemplate('language')}
          <div class="text"><div class="title type-body-large">${t('settingsLanguage')}</div></div>
          <div class="trailing">
            ${choiceGroup<LanguagePreference>(
              [
                ['system', t('languageSystem'), null],
                // Language names are always shown in their own language.
                ['english', 'English', null],
                ['simplifiedChinese', '简体中文', null],
              ],
              settings.language,
              (language) => {
                this.update_((s) => ({ ...s, language }));
              },
            )}
          </div>
        </div>
      </div>`;
  }

  private async chooseExecutable(): Promise<void> {
    const { host } = app();
    const path = await host.pickFile('/usr/local/bin');
    if (path !== null) {
      this.update_((s) => ({ ...s, vfoxExecutable: path }));
    }
  }

  private async saveEnvironment(): Promise<void> {
    const text = this.environmentText ?? formatEnvironmentLines(app().model.settings.settings.extraEnvironment);
    const parsed = parseEnvironmentLines(text);
    if (parsed.invalidLines.length > 0) {
      this.environmentError = t('settingsEnvironmentInvalid', {
        count: parsed.invalidLines.length,
        lines: parsed.invalidLines.join(', '),
      });
      return;
    }
    this.environmentError = '';
    await app().model.settings.update((s) => ({ ...s, extraEnvironment: parsed.variables }));
    snackbar(t('settingsSaved'));
  }

  private renderVfox(): TemplateResult {
    const { model } = app();
    const vfox = model.vfox;
    const source =
      vfox.executable === null
        ? t('settingsExecutableMissing')
        : vfox.isCustomExecutable
          ? t('settingsExecutableCustom')
          : t('settingsExecutableDetected');
    const details = [
      vfox.executable ?? 'vfox',
      source,
      ...(vfox.version === null ? [] : [t('settingsVfoxVersion', { version: vfox.version })]),
    ].join(' · ');
    const environment = this.environmentText ?? formatEnvironmentLines(model.settings.settings.extraEnvironment);
    return html`${this.header(t('settingsVfox'))}
      <div class="group">
        <div class="row-item">
          ${iconTemplate('terminal')}
          <div class="text">
            <div class="title type-body-large">${t('settingsExecutable')}</div>
            <div class="supporting type-body-medium">${details}</div>
          </div>
          <div class="trailing">
            ${
              vfox.availability === 'checking'
                ? html`<m3e-loading-indicator
                    class="small"
                    aria-label=${t('vfoxCheckingTitle')}
                  ></m3e-loading-indicator>`
                : vfox.isCustomExecutable
                  ? html`<vfox-icon-button
                      icon="refresh"
                      .label=${t('settingsExecutableDetect')}
                      @click=${() => {
                        this.update_((s) => ({ ...s, vfoxExecutable: null }));
                      }}
                    ></vfox-icon-button>`
                  : nothing
            }
            <m3e-button variant="tonal" @click=${() => void this.chooseExecutable()}>${t('actionBrowse')}</m3e-button>
          </div>
        </div>
        <div class="row-item environment">
          ${iconTemplate('environment')}
          <div class="text">
            <div class="title type-body-large">${t('settingsEnvironment')}</div>
            <div class="supporting type-body-medium">${t('settingsEnvironmentHelp')}</div>
          </div>
          <div class="wide">
            <m3e-form-field variant="outlined" ?error=${this.environmentError !== ''}>
              <textarea
                class="mono"
                rows="3"
                aria-label=${t('settingsEnvironment')}
                placeholder="VFOX_NODEJS_MIRROR=https://npmmirror.com/mirrors/node"
                .value=${environment}
                @input=${(event: Event) => {
                  this.environmentText = (event.target as HTMLTextAreaElement).value;
                }}
              ></textarea>
              <span slot="error">${this.environmentError}</span>
            </m3e-form-field>
            <div class="save">
              <m3e-button variant="filled" @click=${() => void this.saveEnvironment()}>${t('actionSave')}</m3e-button>
            </div>
          </div>
        </div>
      </div>`;
  }

  private renderConfiguration(): TemplateResult {
    const { model } = app();
    const config = model.vfoxConfig;
    let body: TemplateResult | typeof nothing;
    if (model.vfox.availability !== 'ready') {
      body = nothing;
    } else if (!config.hasValue) {
      body =
        config.error === null
          ? html`<vfox-loading></vfox-loading>`
          : errorMessage(t('settingsVfoxConfigError'), config.error, () => void config.load());
    } else {
      const other = (config.value ?? []).filter((entry) => !KNOWN_KEYS.has(entry.key));
      body = html`<div class="group">
          ${this.configSwitch(CONFIG_KEYS.proxyEnable, t('configProxyEnable'), 'proxy')}
          ${this.configText(CONFIG_KEYS.proxyUrl, t('configProxyUrl'), null, 'http://127.0.0.1:7890')}
          ${this.configText(CONFIG_KEYS.registry, t('configRegistry'), 'registry', t('configRegistryHint'))}
          ${this.configText(CONFIG_KEYS.storage, t('configStorage'), 'storage', t('configStorageHint'), null, true)}
        </div>
        <div class="group">
          ${this.configSwitch(CONFIG_KEYS.legacyEnable, t('configLegacyEnable'), 'legacyFiles', t('configLegacyEnableHelp'))}
          ${this.configChoice(CONFIG_KEYS.legacyStrategy, t('configLegacyStrategy'), [
            ['specified', t('strategySpecified')],
            ['latest_installed', t('strategyLatestInstalled')],
            ['latest_available', t('strategyLatestAvailable')],
          ])}
          ${this.configSwitch(CONFIG_KEYS.gitignore, t('configGitignore'), 'gitignore', t('configGitignoreHelp'))}
        </div>
        <div class="group">
          ${this.configText(CONFIG_KEYS.cache, t('configCache'), 'cache', '12h', t('configCacheHelp'))}
          ${this.configText(CONFIG_KEYS.httpTimeout, t('configHttpTimeout'), 'timeout', '30s', t('configTimeoutHelp'))}
          ${this.configText(CONFIG_KEYS.downloadTimeout, t('configDownloadTimeout'), null, '30m', t('configTimeoutHelp'))}
        </div>
        ${
          other.length === 0
            ? nothing
            : html`${this.header(t('configOther'))}
                <div class="group">${other.map((entry) => this.configText(entry.key, entry.key, null, ''))}</div>`
        }`;
    }
    return html`${this.header(t('settingsVfoxConfig'))}
      <p class="help type-body-small">${t('settingsVfoxConfigHelp')}</p>
      ${body}`;
  }

  private configSwitch(key: string, title: string, iconName: IconName, help?: string): TemplateResult {
    const config = app().model.vfoxConfig;
    const value = config.valueFor(key) === 'true';
    const saving = config.isSaving(key);
    const toggle = (): void => {
      if (!saving) {
        void saveConfig(() => config.set(key, String(!value)));
      }
    };
    return html`<div class="row-item clickable" @click=${toggle}>
      ${iconTemplate(iconName)}
      <div class="text">
        <div class="title type-body-large">${title}</div>
        ${help === undefined ? nothing : html`<div class="supporting type-body-medium">${help}</div>`}
      </div>
      <div class="trailing">
        <m3e-switch
          .checked=${value}
          ?disabled=${saving}
          aria-label=${title}
          @click=${(event: Event) => {
            event.stopPropagation();
            event.preventDefault();
            toggle();
          }}
        ></m3e-switch>
      </div>
    </div>`;
  }

  private configChoice(key: string, title: string, choices: readonly [string, string][]): TemplateResult {
    const config = app().model.vfoxConfig;
    const value = config.valueFor(key) ?? '';
    return html`<div class="row-item">
      <span class="icon-space"></span>
      <div class="text"><div class="title type-body-large">${title}</div></div>
      <div class="trailing">
        <m3e-form-field variant="outlined" hide-subscript="always">
          <m3e-select
            aria-label=${title}
            ?disabled=${config.isSaving(key)}
            @change=${(event: Event) => {
              const selection = (event.target as HTMLElement & { value: string }).value;
              if (selection !== '' && selection !== value) {
                void saveConfig(() => config.set(key, selection));
              }
            }}
          >
            ${choices.map(
              ([choice, label]) =>
                html`<m3e-option value=${choice} ?selected=${choice === value}>${label}</m3e-option>`,
            )}
          </m3e-select>
        </m3e-form-field>
      </div>
    </div>`;
  }

  private saveText(key: string, input: HTMLInputElement): void {
    const config = app().model.vfoxConfig;
    const value = input.value.trim();
    const stored = config.valueFor(key) ?? '';
    this.drafts.delete(key);
    if (value === stored) {
      return;
    }
    // Clearing a field resets the key. If vfox rejects the value, the reload
    // shows the stored value again.
    void saveConfig(() => (value === '' ? config.reset(key) : config.set(key, value)));
  }

  private async browseStorage(key: string, input: HTMLInputElement): Promise<void> {
    const directory = await app().host.pickDirectory(input.value === '' ? undefined : input.value);
    if (directory !== null) {
      input.value = directory;
      this.saveText(key, input);
    }
  }

  private configText(
    key: string,
    title: string,
    iconName: IconName | null,
    hint: string,
    help: string | null = null,
    pickDirectory = false,
  ): TemplateResult {
    const config = app().model.vfoxConfig;
    const value = this.drafts.get(key) ?? config.valueFor(key) ?? '';
    const id = `config-${key.replaceAll('.', '-')}`;
    return html`<div class="row-item">
      ${iconName === null ? html`<span class="icon-space"></span>` : iconTemplate(iconName)}
      <div class="text">
        <div class="title type-body-large">${title}</div>
        ${help === null ? nothing : html`<div class="supporting type-body-medium">${help}</div>`}
      </div>
      <div class="trailing">
        <m3e-form-field variant="outlined" hide-subscript="always">
          <input
            id=${id}
            aria-label=${title}
            placeholder=${hint}
            .value=${value}
            ?disabled=${config.isSaving(key)}
            @input=${(event: Event) => this.drafts.set(key, (event.target as HTMLInputElement).value)}
            @keydown=${(event: KeyboardEvent) => {
              if (event.key === 'Enter') {
                this.saveText(key, event.target as HTMLInputElement);
              }
            }}
            @change=${(event: Event) => {
              this.saveText(key, event.target as HTMLInputElement);
            }}
          />
          ${
            pickDirectory
              ? html`<vfox-icon-button
                  slot="suffix"
                  icon="folder"
                  .label=${t('actionBrowse')}
                  @click=${(event: Event) => {
                    const input = (event.currentTarget as HTMLElement).parentElement?.querySelector('input');
                    if (input !== null && input !== undefined) {
                      void this.browseStorage(key, input);
                    }
                  }}
                ></vfox-icon-button>`
              : nothing
          }
        </m3e-form-field>
      </div>
    </div>`;
  }

  private renderAbout(): TemplateResult {
    const { host } = app();
    const link = (iconName: IconName, title: string, url: string): TemplateResult =>
      html`<div
        class="row-item clickable"
        role="link"
        tabindex="0"
        @click=${() => void host.openUrl(url)}
        @keydown=${(event: KeyboardEvent) => {
          if (event.key === 'Enter') {
            void host.openUrl(url);
          }
        }}
      >
        ${iconTemplate(iconName)}
        <div class="text">
          <div class="title type-body-large">${title}</div>
          <div class="supporting type-body-medium">${url}</div>
        </div>
        <div class="trailing">${iconTemplate('openInNew')}</div>
      </div>`;
    return html`${this.header(t('settingsAbout'))}
      <div class="group">
        <div class="row-item">
          <vfox-avatar name="vfox+" icon="sdks" size="40"></vfox-avatar>
          <div class="text">
            <div class="title type-body-large">${t('appTitle')}</div>
            <div class="supporting type-body-medium">
              ${t('aboutVersion', { version: APP_VERSION })} · ${t('aboutLicense')} · ${host.name}
            </div>
          </div>
        </div>
        ${link('sourceCode', t('aboutSource'), LINKS.repository)} ${link('docs', t('aboutDocs'), LINKS.vfoxDocs)}
        <div
          class="row-item clickable"
          role="button"
          tabindex="0"
          @click=${() => void licenseText().then((text) => showText(t('aboutLicenses'), text))}
        >
          ${iconTemplate('license')}
          <div class="text"><div class="title type-body-large">${t('aboutLicenses')}</div></div>
          <div class="trailing">${iconTemplate('chevronRight')}</div>
        </div>
      </div>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-settings-page': SettingsPage;
  }
}
