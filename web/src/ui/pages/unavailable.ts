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

/** Shown instead of the pages while vfox is missing or broken. */

import { html, LitElement, type TemplateResult } from 'lit';
import { customElement } from 'lit/decorators.js';

import { app, LINKS, t } from '../context';
import { iconTemplate } from '../icons';
import { base, page } from '../styles';
import { Watch } from '../watch';

import '../components/parts';

/** Lets the user choose the vfox executable and stores the choice. */
export async function locateVfox(): Promise<void> {
  const { host, model } = app();
  const path = await host.pickFile('/usr/local/bin');
  if (path !== null) {
    await model.settings.update((settings) => ({ ...settings, vfoxExecutable: path }));
  }
}

@customElement('vfox-unavailable-page')
export class UnavailablePage extends LitElement {
  static override styles = [base, page];

  constructor() {
    super();
    const { model, locale } = app();
    new Watch(this, () => [model.vfox, locale]);
  }

  override render(): TemplateResult {
    const { model, host } = app();
    const vfox = model.vfox;
    if (vfox.availability === 'unknown' || vfox.availability === 'checking' || vfox.availability === 'ready') {
      return html`<div class="pane"><vfox-loading .label=${t('vfoxCheckingTitle')}></vfox-loading></div>`;
    }
    const missing = vfox.availability === 'notFound';
    return html`<div class="pane">
      <vfox-status
        icon=${missing ? 'notFound' : 'warning'}
        shape=${missing ? '4-leaf-clover' : 'soft-burst'}
        ?error=${!missing}
        .heading=${missing ? t('vfoxMissingTitle') : t('vfoxBrokenTitle')}
        .body=${missing ? t('vfoxMissingBody') : t('vfoxBrokenBody', { path: vfox.executable ?? 'vfox' })}
      >
        <m3e-button variant="filled" @click=${() => void host.openUrl(LINKS.vfoxInstall)}>
          ${iconTemplate('openInNew', { slot: 'icon' })}${t('vfoxInstallGuide')}
        </m3e-button>
        <m3e-button variant="tonal" @click=${() => void locateVfox()}>
          ${iconTemplate('folder', { slot: 'icon' })}${t('vfoxLocate')}
        </m3e-button>
        <m3e-button variant="outlined" @click=${() => void vfox.apply(model.settings.settings)}>
          ${iconTemplate('retry', { slot: 'icon' })}${t('actionRetry')}
        </m3e-button>
      </vfox-status>
    </div>`;
  }
}

declare global {
  interface HTMLElementTagNameMap {
    'vfox-unavailable-page': UnavailablePage;
  }
}
