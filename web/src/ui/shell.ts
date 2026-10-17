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
 * The adaptive frame around the pages: a navigation bar at the bottom of
 * compact windows, the expressive navigation rail at the start of wider ones.
 * Switching destinations fades through.
 */

import { css, html, LitElement, nothing, type PropertyValues, type TemplateResult } from 'lit';
import { customElement, query, state } from 'lit/decorators.js';
import { repeat } from 'lit/directives/repeat.js';

import { app, DESTINATIONS, t, type Destination } from './context';
import { icon, iconTemplate, type IconName } from './icons';
import { EXPRESSIVE, prefersReducedMotion, settleDuration, springEasing } from './motion';
import { base } from './styles';
import { Watch } from './watch';

import './pages/home';
import './pages/plugins';
import './pages/sdks';
import './pages/settings';
import './pages/tasks';
import './pages/unavailable';

const NAVIGATION: Readonly<Record<Destination, { icon: IconName; label: string }>> = {
  home: { icon: 'home', label: 'navHome' },
  sdks: { icon: 'sdks', label: 'navSdks' },
  plugins: { icon: 'plugins', label: 'navPlugins' },
  tasks: { icon: 'tasks', label: 'navTasks' },
  settings: { icon: 'settings', label: 'navSettings' },
};

/** Material 3 window size classes: compact windows are narrower than 600 dp. */
const COMPACT_QUERY = '(max-width: 599.98px)';

@customElement('vfox-shell')
export class Shell extends LitElement {
  static override styles = [
    base,
    css`
      :host {
        display: block;
        height: 100%;
      }
      .shell {
        display: flex;
        height: 100%;
      }
      .shell.compact {
        flex-direction: column;
      }
      m3e-nav-rail {
        flex: none;
        height: 100%;
        overflow-y: auto;
        --m3e-nav-rail-compact-width: 96px;
        --m3e-nav-rail-expanded-width: 240px;
      }
      /*
       * The spacing that @m3e/web gives the buttons of the rail with a
       * stylesheet of the document, which does not reach into this shadow root.
       */
      m3e-nav-rail > m3e-icon-button,
      m3e-nav-rail > m3e-fab {
        margin-block-end: var(--m3e-nav-rail-button-item-space, var(--md-sys-measurement-space200, 16px));
      }
      m3e-nav-rail > m3e-icon-button {
        margin-inline-start: var(--m3e-nav-rail-icon-button-inset, var(--md-sys-measurement-space100, 8px));
      }
      m3e-nav-bar {
        flex: none;
        /* Five destinations share the width of compact windows equally. */
        --m3e-nav-bar-vertical-item-width: 0px;
        --m3e-nav-bar-container-color: var(--md-sys-color-surface-container);
      }
      m3e-nav-bar m3e-nav-item {
        flex: 1 1 0;
        min-width: 0;
      }
      .stage {
        position: relative;
        flex: 1 1 auto;
        min-width: 0;
        min-height: 0;
      }
      .shell:not(.compact) .stage {
        margin: var(--gap-l) var(--gap-l) var(--gap-l) 0;
      }
      .page {
        position: absolute;
        inset: 0;
      }
      .badge-host {
        position: relative;
        display: inline-flex;
      }
      .badge {
        position: absolute;
        top: -4px;
        inset-inline-start: 14px;
        min-width: 16px;
        height: 16px;
        padding: 0 4px;
        border-radius: 8px;
        background: var(--md-sys-color-error);
        color: var(--md-sys-color-on-error);
        font-size: 11px;
        font-weight: 500;
        line-height: 16px;
        text-align: center;
      }
    `,
  ];

  @state() private compact = matchMedia(COMPACT_QUERY).matches;
  /** The page that fades out, while the transition runs. */
  @state() private leaving: string | null = null;
  @query('.page.entering') private entering!: HTMLElement | null;
  @query('.page.leaving') private leavingPage!: HTMLElement | null;
  private shownPage = '';
  private animation: Animation | null = null;
  private readonly media = matchMedia(COMPACT_QUERY);
  private readonly onMedia = (): void => {
    this.compact = this.media.matches;
  };

  constructor() {
    super();
    const { model, shell, locale } = app();
    new Watch(this, () => [shell, model.vfox, model.tasks, model.settings, locale]);
  }

  override connectedCallback(): void {
    super.connectedCallback();
    this.media.addEventListener('change', this.onMedia);
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.media.removeEventListener('change', this.onMedia);
  }

  /** The page shown for `destination`; Settings stay reachable without vfox. */
  private pageFor(destination: Destination): string {
    if (app().model.vfox.availability !== 'ready' && destination !== 'settings') {
      return 'unavailable';
    }
    return destination;
  }

  protected override willUpdate(changed: PropertyValues): void {
    super.willUpdate(changed);
    const page = this.pageFor(app().shell.destination);
    if (this.shownPage !== '' && page !== this.shownPage) {
      // A transition that still runs ends at once.
      this.animation?.finish();
      this.leaving = this.shownPage;
      this.startTransition = true;
    }
    this.shownPage = page;
  }

  private startTransition = false;

  protected override updated(): void {
    if (!this.startTransition) {
      return;
    }
    this.startTransition = false;
    const duration = prefersReducedMotion() ? 0 : settleDuration(EXPRESSIVE.slowEffects);
    // Fade through: the old page fades out during the first 30 %, then the new
    // one fades in and grows from 92 %.
    this.leavingPage?.animate([{ opacity: 1 }, { opacity: 0 }], { duration: duration * 0.3, fill: 'forwards' });
    const animation =
      this.entering?.animate(
        [
          { opacity: 0, transform: 'scale(0.92)' },
          { opacity: 1, transform: 'none' },
        ],
        {
          duration: duration * 0.7,
          delay: duration * 0.3,
          easing: springEasing(EXPRESSIVE.slowEffects),
          fill: 'backwards',
        },
      ) ?? null;
    this.animation = animation;
    const done = (): void => {
      if (this.animation === animation) {
        this.animation = null;
        this.leaving = null;
      }
    };
    if (animation === null) {
      done();
    } else {
      animation.onfinish = done;
      animation.oncancel = done;
    }
  }

  private renderPage(page: string): TemplateResult {
    switch (page) {
      case 'home':
        return html`<vfox-home-page></vfox-home-page>`;
      case 'sdks':
        return html`<vfox-sdks-page></vfox-sdks-page>`;
      case 'plugins':
        return html`<vfox-plugins-page></vfox-plugins-page>`;
      case 'tasks':
        return html`<vfox-tasks-page></vfox-tasks-page>`;
      case 'settings':
        return html`<vfox-settings-page></vfox-settings-page>`;
      default:
        return html`<vfox-unavailable-page></vfox-unavailable-page>`;
    }
  }

  private items(): TemplateResult[] {
    const { model, shell } = app();
    const active = model.tasks.activeCount;
    return DESTINATIONS.map((destination) => {
      const item = NAVIGATION[destination];
      const badge = destination === 'tasks' && active > 0 ? html`<span class="badge">${active}</span>` : nothing;
      return html`<m3e-nav-item
        data-destination=${destination}
        ?selected=${shell.destination === destination}
        aria-label=${t(item.label)}
      >
        <span class="badge-host" slot="icon">${iconTemplate(item.icon)}${badge}</span>
        <span class="badge-host" slot="selected-icon">${iconTemplate(item.icon, { filled: true })}${badge}</span>
        ${t(item.label)}
      </m3e-nav-item>`;
    });
  }

  private select(event: Event): void {
    const bar = event.currentTarget as HTMLElement & { selected: HTMLElement | null };
    const destination = bar.selected?.dataset['destination'] as Destination | undefined;
    if (destination !== undefined) {
      app().shell.go(destination);
    }
  }

  override render(): TemplateResult {
    const { model, shell } = app();
    const expanded = model.settings.settings.railExpanded;
    const page = this.shownPage === '' ? this.pageFor(shell.destination) : this.shownPage;
    // Keyed, so that the element of the old page fades out as it is.
    const pages = this.leaving === null || this.leaving === page ? [page] : [this.leaving, page];
    const stage = html`<div class="stage">
      ${repeat(
        pages,
        (key) => key,
        (key) =>
          html`<div
            class="page ${key === page ? 'entering' : 'leaving'}"
            aria-hidden=${key === page ? 'false' : 'true'}
            ?inert=${key !== page}
          >
            ${this.renderPage(key)}
          </div>`,
      )}
    </div>`;

    if (this.compact) {
      return html`<div class="shell compact">
        ${stage}
        <m3e-nav-bar @change=${this.select}>${this.items()}</m3e-nav-bar>
      </div>`;
    }
    const ready = model.vfox.availability === 'ready';
    return html`<div class="shell">
      <m3e-nav-rail mode=${expanded ? 'expanded' : 'compact'} @change=${this.select}>
        <m3e-icon-button
          toggle
          ?selected=${expanded}
          aria-label=${expanded ? t('navCollapse') : t('navExpand')}
          @click=${() => void model.settings.update((settings) => ({ ...settings, railExpanded: !expanded }))}
        >
          <m3e-icon variant="rounded" name=${icon('menu')}></m3e-icon>
          <m3e-icon variant="rounded" slot="selected" name=${icon('menuOpen')}></m3e-icon>
        </m3e-icon-button>
        ${
          ready
            ? html`<m3e-fab size="small" aria-label=${t('navAddPlugin')} @click=${() => shell.go('plugins')}>
                ${iconTemplate('add')}<span slot="label">${t('navAddPlugin')}</span>
              </m3e-fab>`
            : nothing
        }
        ${this.items()}
      </m3e-nav-rail>
      ${stage}
    </div>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-shell': Shell;
  }
}
