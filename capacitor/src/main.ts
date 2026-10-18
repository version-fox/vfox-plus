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
 * The entry point of the page: connects to the host, wires the application
 * model of the web front end (../web/src) to it and mounts the Ionic user
 * interface.
 */

// The Material 3 Expressive elements that Ionic has no counterpart of.
import '@m3e/web/loading-indicator';
import '@m3e/web/progress-indicator';
import '@m3e/web/shape';
import '@m3e/web/tooltip';

import '@ionic/vue/css/core.css';
import '@ionic/vue/css/normalize.css';
import '@ionic/vue/css/structure.css';
import './ui/styles/document.css';
import './ui/styles/ionic.css';

import { IonicVue } from '@ionic/vue';
import { createApp } from 'vue';

import type { Host } from '@web/core/host';
import { systemLocales } from '@web/core/i18n/localizations';
import { JsonFileSettingsStore, settingsPath } from '@web/core/settings/settings-store';
import { AppModel } from '@web/state/app-model';
import { LocaleController, setApp, ShellController } from '@web/ui/context';
import { createCapacitorHost } from './capacitor/vfox-host';
import App from './ui/App.vue';

async function createHost(): Promise<Host> {
  if (import.meta.env.DEV && window.vfoxHost === undefined) {
    // `npm run dev` in a browser, with the development host of ../web.
    return (await import('@web/hosts/dev')).createDevHost();
  }
  return createCapacitorHost();
}

async function main(): Promise<void> {
  const host = await createHost();
  const environment = await host.environment();
  const temporary = await host.files.systemTemporaryDirectory();
  const model = new AppModel({
    runner: host,
    files: host.files,
    settingsStore: new JsonFileSettingsStore(host.files, settingsPath(environment, temporary)),
    environment,
  });
  setApp({
    model,
    host,
    shell: new ShellController(),
    locale: new LocaleController(systemLocales(environment, navigator.languages)),
  });
  // Material Design everywhere: Ionic would pick it on Linux anyway.
  createApp(App).use(IonicVue, { mode: 'md' }).mount('#app');
  await model.start();
  if (__VFOX_PLUS_E2E__) {
    await (await import('@web/e2e/runner')).runScenario(host);
  }
}

main().catch((error: unknown) => {
  console.error(error);
  document.body.textContent = `vfox+ could not start: ${String(error)}`;
});
