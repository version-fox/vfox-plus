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
 * The dialogs of the application, on `m3e-dialog`. Each dialog is a short-lived
 * element added to the application root and removed once it has closed; the
 * functions resolve with what the user chose.
 */

import { css, html, LitElement, nothing, type TemplateResult } from 'lit';
import { customElement, property, query, state } from 'lit/decorators.js';

import { isValidPluginName } from '../../core/vfox/cli';
import { app, t } from '../context';
import { iconTemplate, type IconName } from '../icons';
import { base, typeScale } from '../styles';

import './parts';

const dialogStyles = css`
  m3e-dialog {
    --m3e-dialog-shape: var(--corner-extra-large);
    --m3e-dialog-container-color: var(--md-sys-color-surface-container-high);
  }
  .header {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--gap-l);
    text-align: center;
  }
  .header m3e-icon {
    color: var(--md-sys-color-secondary);
  }
  .header.destructive m3e-icon {
    color: var(--md-sys-color-error);
  }
  .body {
    max-width: 480px;
    color: var(--md-sys-color-on-surface-variant);
  }
  .actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--gap-s);
  }
  .field-row {
    display: flex;
    align-items: flex-start;
    gap: var(--gap-s);
    margin-top: var(--gap-l);
  }
  .field-row m3e-form-field {
    flex: 1 1 auto;
  }
  .field-row vfox-icon-button {
    margin-top: var(--gap-s);
  }
  m3e-form-field {
    width: 100%;
  }
  .destructive-button {
    --m3e-filled-button-container-color: var(--md-sys-color-error);
    --m3e-filled-button-label-text-color: var(--md-sys-color-on-error);
    --m3e-filled-button-icon-color: var(--md-sys-color-on-error);
  }
`;

/** Shows `dialog` and resolves with its result once it has closed. */
function show<T>(dialog: HTMLElement & { result: Promise<T> }): Promise<T> {
  (document.querySelector('m3e-theme') ?? document.body).append(dialog);
  return dialog.result.finally(() => {
    dialog.remove();
  });
}

/** A base for dialogs: an open `m3e-dialog` that settles `result` when it closes. */
abstract class DialogElement<T> extends LitElement {
  readonly result: Promise<T>;
  protected settle: (value: T) => void = () => undefined;
  @query('m3e-dialog') protected dialog!: (HTMLElement & { open: boolean }) | null;
  private value: T;

  constructor(dismissed: T) {
    super();
    this.value = dismissed;
    this.result = new Promise<T>((resolve) => {
      this.settle = resolve;
    });
  }

  /** Closes the dialog with `value`. */
  protected close(value: T): void {
    this.value = value;
    if (this.dialog !== null) {
      this.dialog.open = false;
    }
  }

  protected onClosed(): void {
    this.settle(this.value);
  }
}

@customElement('vfox-confirm-dialog')
class ConfirmDialog extends DialogElement<boolean> {
  static override styles = [base, typeScale, dialogStyles];

  @property() heading = '';
  @property() body = '';
  @property() confirmLabel = '';
  @property() icon: IconName | null = null;
  @property({ type: Boolean }) destructive = false;

  constructor() {
    super(false);
  }

  override render(): TemplateResult {
    return html`<m3e-dialog open @closed=${this.onClosed}>
      <div slot="header" class="header ${this.destructive ? 'destructive' : ''}">
        ${this.icon === null ? nothing : iconTemplate(this.icon)}
        <span class="type-headline-small">${this.heading}</span>
      </div>
      <p class="body type-body-medium">${this.body}</p>
      <div slot="actions" class="actions">
        <m3e-button variant="text" @click=${() => this.close(false)}>${t('actionCancel')}</m3e-button>
        <m3e-button
          variant="filled"
          class=${this.destructive ? 'destructive-button' : ''}
          autofocus
          @click=${() => this.close(true)}
          >${this.confirmLabel}</m3e-button
        >
      </div>
    </m3e-dialog>`;
  }
}

/** Asks the user to confirm an action. Resolves with whether they did. */
export function confirmAction(options: {
  title: string;
  body: string;
  confirmLabel: string;
  icon?: IconName;
  destructive?: boolean;
}): Promise<boolean> {
  const dialog = new ConfirmDialog();
  dialog.heading = options.title;
  dialog.body = options.body;
  dialog.confirmLabel = options.confirmLabel;
  dialog.icon = options.icon ?? null;
  dialog.destructive = options.destructive ?? false;
  return show(dialog);
}

@customElement('vfox-project-dialog')
class ProjectDialog extends DialogElement<string | null> {
  static override styles = [base, typeScale, dialogStyles];

  @property() name = '';
  @property() version = '';
  @state() private error = '';
  @query('input') private input!: HTMLInputElement | null;

  constructor() {
    super(null);
  }

  private async browse(): Promise<void> {
    const { host, model } = app();
    const current = this.input?.value.trim() ?? '';
    const home = model.cli.launchConfig.baseEnvironment['HOME'];
    const directory = await host.pickDirectory(current === '' ? home : current);
    if (directory !== null && this.input !== null) {
      this.input.value = directory;
      this.error = '';
    }
  }

  private async submit(): Promise<void> {
    const path = this.input?.value.trim() ?? '';
    if (path === '' || !(await app().host.files.isDirectory(path))) {
      this.error = t('useProjectInvalid');
      return;
    }
    this.close(path);
  }

  override render(): TemplateResult {
    return html`<m3e-dialog open @closed=${this.onClosed}>
      <div slot="header" class="header">
        ${iconTemplate('projectScope')}
        <span class="type-headline-small">${t('useProjectTitle', { name: this.name, version: this.version })}</span>
      </div>
      <p class="body type-body-medium">${t('useProjectBody')}</p>
      <div class="field-row">
        <m3e-form-field variant="outlined" ?error=${this.error !== ''}>
          <label slot="label" for="directory">${t('useProjectDirectory')}</label>
          <input
            id="directory"
            autofocus
            @keydown=${(event: KeyboardEvent) => {
              if (event.key === 'Enter') {
                void this.submit();
              }
            }}
            @input=${() => (this.error = '')}
          />
          <span slot="error">${this.error}</span>
        </m3e-form-field>
        <vfox-icon-button
          icon="folder"
          variant="tonal"
          .label=${t('actionBrowse')}
          @click=${() => void this.browse()}
        ></vfox-icon-button>
      </div>
      <div slot="actions" class="actions">
        <m3e-button variant="text" @click=${() => this.close(null)}>${t('actionCancel')}</m3e-button>
        <m3e-button variant="filled" @click=${() => void this.submit()}>${t('useProjectConfirm')}</m3e-button>
      </div>
    </m3e-dialog>`;
  }
}

/** Asks for the project directory in which to pin `name` `version`. */
export function askProjectDirectory(name: string, version: string): Promise<string | null> {
  const dialog = new ProjectDialog();
  dialog.name = name;
  dialog.version = version;
  return show(dialog);
}

/** The source and optional alias of a plugin added from a URL or a path. */
export interface CustomPluginSource {
  readonly source: string;
  readonly alias: string | null;
}

@customElement('vfox-custom-plugin-dialog')
class CustomPluginDialog extends DialogElement<CustomPluginSource | null> {
  static override styles = [base, typeScale, dialogStyles];

  @state() private sourceError = '';
  @state() private aliasError = '';
  @query('#source') private source!: HTMLInputElement | null;
  @query('#alias') private alias!: HTMLInputElement | null;

  constructor() {
    super(null);
  }

  private async browse(): Promise<void> {
    const { host, model } = app();
    const file = await host.pickFile(model.cli.launchConfig.baseEnvironment['HOME'], {
      label: t('pluginsCustomFileType'),
      extensions: ['zip', 'lua'],
    });
    if (file !== null && this.source !== null) {
      this.source.value = file;
      this.sourceError = '';
    }
  }

  private submit(): void {
    const source = this.source?.value.trim() ?? '';
    const alias = this.alias?.value.trim() ?? '';
    this.sourceError = source === '' ? t('pluginsCustomRequired') : '';
    this.aliasError = alias !== '' && !isValidPluginName(alias) ? t('pluginsCustomInvalidName') : '';
    if (this.sourceError === '' && this.aliasError === '') {
      this.close({ source, alias: alias === '' ? null : alias });
    }
  }

  private onKey(event: KeyboardEvent): void {
    if (event.key === 'Enter') {
      this.submit();
    }
  }

  override render(): TemplateResult {
    return html`<m3e-dialog open @closed=${this.onClosed}>
      <div slot="header" class="header">
        ${iconTemplate('plugins')}
        <span class="type-headline-small">${t('pluginsCustomTitle')}</span>
      </div>
      <div class="field-row">
        <m3e-form-field variant="outlined" ?error=${this.sourceError !== ''}>
          <label slot="label" for="source">${t('pluginsCustomSource')}</label>
          <input id="source" autofocus @keydown=${this.onKey} @input=${() => (this.sourceError = '')} />
          <span slot="hint">${t('pluginsCustomSourceHelp')}</span>
          <span slot="error">${this.sourceError}</span>
        </m3e-form-field>
        <vfox-icon-button
          icon="folder"
          variant="tonal"
          .label=${t('actionBrowse')}
          @click=${() => void this.browse()}
        ></vfox-icon-button>
      </div>
      <div class="field-row">
        <m3e-form-field variant="outlined" ?error=${this.aliasError !== ''}>
          <label slot="label" for="alias">${t('pluginsCustomAlias')}</label>
          <input id="alias" @keydown=${this.onKey} @input=${() => (this.aliasError = '')} />
          <span slot="hint">${t('pluginsCustomAliasHelp')}</span>
          <span slot="error">${this.aliasError}</span>
        </m3e-form-field>
      </div>
      <div slot="actions" class="actions">
        <m3e-button variant="text" @click=${() => this.close(null)}>${t('actionCancel')}</m3e-button>
        <m3e-button variant="filled" @click=${() => this.submit()}>${t('pluginsCustomConfirm')}</m3e-button>
      </div>
    </m3e-dialog>`;
  }
}

/** Asks for the source of a plugin that is not in the registry. */
export function askCustomPlugin(): Promise<CustomPluginSource | null> {
  return show(new CustomPluginDialog());
}

@customElement('vfox-text-dialog')
class TextDialog extends DialogElement<null> {
  static override styles = [
    base,
    typeScale,
    dialogStyles,
    css`
      pre {
        margin: 0;
        max-width: min(720px, 80vw);
        max-height: 60vh;
        overflow: auto;
        white-space: pre-wrap;
        user-select: text;
        color: var(--md-sys-color-on-surface-variant);
      }
    `,
  ];

  @property() heading = '';
  @property() text = '';

  constructor() {
    super(null);
  }

  override render(): TemplateResult {
    return html`<m3e-dialog open @closed=${this.onClosed}>
      <div slot="header" class="header"><span class="type-headline-small">${this.heading}</span></div>
      <pre class="mono type-body-small">${this.text}</pre>
      <div slot="actions" class="actions">
        <m3e-button variant="text" @click=${() => this.close(null)}>${t('actionClose')}</m3e-button>
      </div>
    </m3e-dialog>`;
  }
}

/** Shows a long, monospaced text such as the open-source licenses. */
export async function showText(title: string, text: string): Promise<void> {
  const dialog = new TextDialog();
  dialog.heading = title;
  dialog.text = text;
  await show(dialog);
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-confirm-dialog': ConfirmDialog;
    'vfox-project-dialog': ProjectDialog;
    'vfox-custom-plugin-dialog': CustomPluginDialog;
    'vfox-text-dialog': TextDialog;
  }
}
