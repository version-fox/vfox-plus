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

/** Running, queued and finished operations with their output. */

import { css, html, LitElement, nothing, type PropertyValues, type TemplateResult } from 'lit';
import { customElement, property, query } from 'lit/decorators.js';
import { repeat } from 'lit/directives/repeat.js';

import { commandLine } from '../../core/vfox/command';
import { isActive, type VfoxTask } from '../../state/task-queue';
import { app, t, taskTitle } from '../context';
import { iconTemplate } from '../icons';
import { EXPRESSIVE, prefersReducedMotion, settleDuration, springEasing } from '../motion';
import { snackbar } from '../snackbar';
import { base, page, typeScale } from '../styles';
import { Watch } from '../watch';

import '../components/parts';

@customElement('vfox-tasks-page')
export class TasksPage extends LitElement {
  static override styles = [
    base,
    typeScale,
    page,
    css`
      .list {
        display: flex;
        flex-direction: column;
        gap: var(--gap-hairline);
        padding: 0 var(--gap-xl) var(--gap-xl);
      }
    `,
  ];

  constructor() {
    super();
    const { model, locale } = app();
    new Watch(this, () => [model.tasks, locale]);
  }

  override render(): TemplateResult {
    const { model } = app();
    const tasks = model.tasks.tasks;
    return html`<div class="pane">
      <header class="page-header">
        <div class="titles">
          <h1 class="type-headline-medium-emphasized" role="heading" aria-level="1">${t('navTasks')}</h1>
        </div>
        <div class="actions">
          <m3e-button variant="text" ?disabled=${!model.tasks.hasFinished} @click=${() => model.tasks.clearFinished()}>
            ${iconTemplate('succeeded', { slot: 'icon' })}${t('tasksClearFinished')}
          </m3e-button>
        </div>
      </header>
      ${
        tasks.length === 0
          ? html`<vfox-status
              icon="tasks"
              shape="sunny"
              .heading=${t('tasksEmptyTitle')}
              .body=${t('tasksEmptyBody')}
            ></vfox-status>`
          : html`<div class="list">
              ${repeat(
                tasks,
                (task) => task.id,
                (task, index) =>
                  html`<vfox-task-item
                    .task=${task}
                    ?first=${index === 0}
                    ?last=${index === tasks.length - 1}
                  ></vfox-task-item>`,
              )}
            </div>`
      }
    </div>`;
  }
}

/** One task of the list: status, title, progress, actions and its output. */
@customElement('vfox-task-item')
export class TaskItem extends LitElement {
  static override styles = [
    base,
    typeScale,
    css`
      :host {
        display: block;
        --outer: var(--corner-large-increased);
        --inner: var(--corner-extra-small);
      }
      .item {
        display: flex;
        align-items: center;
        gap: var(--gap-l);
        padding: var(--gap-m) var(--gap-l);
        background: var(--md-sys-color-surface-container);
        border-radius: var(--inner);
        cursor: pointer;
        transition: border-radius var(--md-sys-motion-spring-fast-spatial);
        position: relative;
      }
      .item:hover {
        --inner: var(--corner-medium);
      }
      :host([first]) .item {
        border-top-left-radius: var(--outer);
        border-top-right-radius: var(--outer);
      }
      :host([last]:not([expanded])) .item {
        border-bottom-left-radius: var(--outer);
        border-bottom-right-radius: var(--outer);
      }
      .status {
        flex: none;
        width: 32px;
        height: 32px;
        display: grid;
        place-items: center;
        color: var(--md-sys-color-on-surface-variant);
      }
      .status.succeeded {
        color: var(--md-sys-color-primary);
      }
      .status.failed {
        color: var(--md-sys-color-error);
      }
      m3e-loading-indicator {
        --m3e-loading-indicator-size: 32px;
      }
      .text {
        flex: 1 1 auto;
        min-width: 0;
      }
      .title {
        color: var(--md-sys-color-on-surface);
      }
      .detail {
        color: var(--md-sys-color-on-surface-variant);
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
        overflow: hidden;
        overflow-wrap: anywhere;
      }
      .detail.failed {
        color: var(--md-sys-color-error);
      }
      m3e-linear-progress-indicator {
        display: block;
        max-width: 480px;
        margin-top: var(--gap-s);
      }
      .actions {
        display: flex;
        align-items: center;
        flex: none;
      }
      .toggle {
        transition: transform var(--md-sys-motion-spring-fast-spatial);
      }
      :host([expanded]) .toggle {
        transform: rotate(180deg);
      }
      .output-frame {
        overflow: hidden;
      }
      .output {
        display: flex;
        align-items: flex-start;
        gap: var(--gap-xs);
        margin-top: var(--gap-hairline);
        padding: var(--gap-s) var(--gap-xs) var(--gap-m) var(--gap-l);
        background: var(--md-sys-color-surface-container-highest);
        border-radius: var(--inner);
      }
      :host([last]) .output {
        border-bottom-left-radius: var(--outer);
        border-bottom-right-radius: var(--outer);
      }
      pre {
        flex: 1 1 auto;
        margin: 0;
        max-height: 320px;
        overflow: auto;
        white-space: pre-wrap;
        overflow-wrap: anywhere;
        line-height: 1.5;
        user-select: text;
        cursor: text;
        color: var(--md-sys-color-on-surface);
      }
    `,
  ];

  @property({ attribute: false }) task: VfoxTask | null = null;
  @property({ type: Boolean, reflect: true }) first = false;
  @property({ type: Boolean, reflect: true }) last = false;
  @property({ type: Boolean, reflect: true }) expanded = false;
  private animateOutput = false;
  @query('pre') private output!: HTMLElement | null;
  @query('.output-frame') private frame!: HTMLElement | null;

  constructor() {
    super();
    new Watch(this, () => [this.task, app().locale]);
  }

  private toggle(): void {
    this.expanded = !this.expanded;
    this.animateOutput = true;
  }

  protected override updated(changed: PropertyValues): void {
    // Show the end of the output, where vfox writes.
    if (this.output !== null) {
      this.output.scrollTop = this.output.scrollHeight;
    }
    if (changed.has('expanded') && this.animateOutput && this.frame !== null) {
      this.animateOutput = false;
      if (!prefersReducedMotion()) {
        const height = this.frame.scrollHeight;
        this.frame.animate([{ height: '0px' }, { height: `${height}px` }], {
          duration: settleDuration(EXPRESSIVE.defaultSpatial),
          easing: springEasing(EXPRESSIVE.defaultSpatial),
        });
      }
    }
  }

  private async copyOutput(text: string): Promise<void> {
    await app().host.copyText(text);
    snackbar(t('actionCopied'));
  }

  override render(): TemplateResult {
    const task = this.task;
    if (task === null) {
      return html``;
    }
    const { model } = app();
    const status = task.status;
    const leading = {
      queued: iconTemplate('queued'),
      running: html`<m3e-loading-indicator aria-label=${t('taskStatusRunning')}></m3e-loading-indicator>`,
      succeeded: iconTemplate('succeeded', { filled: true }),
      failed: iconTemplate('error', { filled: true }),
      cancelled: iconTemplate('cancel'),
    }[status];
    const progress = task.progress;
    const detail = {
      queued: t('taskStatusQueued'),
      running:
        progress === null ? t('taskStatusRunning') : t('taskDownloading', { percent: Math.round(progress * 100) }),
      succeeded: t('taskStatusSucceeded'),
      failed: `${t('taskStatusFailed')} · ${task.errorMessage ?? ''}`,
      cancelled: t('taskStatusCancelled'),
    }[status];
    const command = `$ ${commandLine(task.command)}`;
    const text = [command, ...task.log].join('\n');

    return html`<div class="item" @click=${() => this.toggle()}>
        <div class="status ${status}">${leading}</div>
        <div class="text">
          <div class="title type-body-large-emphasized">${taskTitle(task.command)}</div>
          <div class="detail type-body-medium ${status === 'failed' ? 'failed' : ''}">${detail}</div>
          ${
            status === 'running' && task.command.operation === 'install'
              ? html`<m3e-linear-progress-indicator
                  variant="wavy"
                  mode=${progress === null ? 'indeterminate' : 'determinate'}
                  value=${Math.round((progress ?? 0) * 100)}
                ></m3e-linear-progress-indicator>`
              : nothing
          }
        </div>
        <div class="actions" @click=${(event: Event) => event.stopPropagation()}>
          ${
            isActive(status)
              ? html`<vfox-icon-button
                  icon="cancel"
                  .label=${t('actionCancel')}
                  @click=${() => model.tasks.cancel(task)}
                ></vfox-icon-button>`
              : status === 'failed' || status === 'cancelled'
                ? html`<vfox-icon-button
                    icon="retry"
                    .label=${t('actionRetry')}
                    @click=${() => model.tasks.retry(task)}
                  ></vfox-icon-button>`
                : nothing
          }
          <vfox-icon-button
            class="toggle"
            icon="expandMore"
            .label=${this.expanded ? t('taskHideOutput') : t('taskShowOutput')}
            @click=${() => this.toggle()}
          ></vfox-icon-button>
        </div>
      </div>
      ${
        this.expanded
          ? html`<div class="output-frame">
              <div class="output">
                <pre class="mono type-body-small">
${task.log.length === 0 ? `${command}\n${t('taskNoOutput')}` : text}</pre>
                <vfox-icon-button
                  icon="copy"
                  .label=${t('taskCopyOutput')}
                  @click=${() => void this.copyOutput(text)}
                ></vfox-icon-button>
              </div>
            </div>`
          : nothing
      }`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-tasks-page': TasksPage;
    'vfox-task-item': TaskItem;
  }
}
