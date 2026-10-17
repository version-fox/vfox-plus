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

/** The root element: theme, language, the shell and the task snack bars. */

import { css, html, LitElement, type TemplateResult } from 'lit';
import { customElement } from 'lit/decorators.js';

import type { VfoxTask } from '../state/task-queue';
import { app, t, taskTitle } from './context';
import { prefersReducedMotion } from './motion';
import { snackbar } from './snackbar';
import { base } from './styles';
import { applyTheme } from './theme';
import { Watch } from './watch';

import './shell';

const DARK_QUERY = '(prefers-color-scheme: dark)';
const REDUCED_MOTION_QUERY = '(prefers-reduced-motion: reduce)';

@customElement('vfox-plus-app')
export class VfoxPlusApp extends LitElement {
  static override styles = [
    base,
    css`
      :host {
        display: block;
        height: 100vh;
        background: var(--md-sys-color-surface);
        color: var(--md-sys-color-on-surface);
      }
    `,
  ];

  private accent: number | null = null;
  private readonly stops: (() => void)[] = [];
  private readonly dark = matchMedia(DARK_QUERY);
  private readonly reducedMotion = matchMedia(REDUCED_MOTION_QUERY);
  private readonly onMedia = (): void => {
    this.applySettings();
  };

  constructor() {
    super();
    new Watch(this, () => [app().locale]);
  }

  override connectedCallback(): void {
    super.connectedCallback();
    const { model, host } = app();
    this.stops.push(
      model.settings.addListener(() => {
        this.applySettings();
      }),
      model.tasks.onFinished((task) => {
        this.announce(task);
      }),
    );
    this.dark.addEventListener('change', this.onMedia);
    this.reducedMotion.addEventListener('change', this.onMedia);
    this.applySettings();
    void host.accentColor().then(
      (color) => {
        this.accent = color;
        this.applySettings();
      },
      () => undefined,
    );
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.stops.forEach((stop) => stop());
    this.dark.removeEventListener('change', this.onMedia);
    this.reducedMotion.removeEventListener('change', this.onMedia);
  }

  private applySettings(): void {
    const { model, locale } = app();
    const settings = model.settings.settings;
    const seed = settings.useSystemColor && this.accent !== null ? this.accent : settings.seedColor;
    const dark = settings.theme === 'dark' || (settings.theme === 'system' && this.dark.matches);
    applyTheme(seed, dark, prefersReducedMotion());
    locale.apply(settings.language);
    document.documentElement.lang = locale.l10n.languageTag;
    document.title = t('appTitle');
  }

  /** Tells the user how a task ended, unless they are watching the task list. */
  private announce(task: VfoxTask): void {
    const { shell } = app();
    if (task.status === 'cancelled') {
      return;
    }
    const title = taskTitle(task.command);
    if (task.status === 'failed') {
      // Failures stay until dismissed, so that they can be read in full.
      snackbar(`${t('taskFailed', { title })}\n${task.errorMessage ?? ''}`.trim(), {
        action: t('taskView'),
        onAction: () => {
          shell.go('tasks');
        },
        persistent: true,
      });
    } else if (shell.destination !== 'tasks') {
      snackbar(t('taskSucceeded', { title }));
    }
  }

  override render(): TemplateResult {
    return html`<vfox-shell></vfox-shell>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-plus-app': VfoxPlusApp;
  }
}
