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
 * The entry point: connects to the host of the shell this front end runs in,
 * wires the application model to it and starts the user interface.
 */

// The components of @m3e/web the application uses; the rest is not bundled.
import '@m3e/web/button';
import '@m3e/web/button-group';
import '@m3e/web/card';
import '@m3e/web/dialog';
import '@m3e/web/fab';
import '@m3e/web/form-field';
import '@m3e/web/icon';
import '@m3e/web/icon-button';
import '@m3e/web/list';
import '@m3e/web/loading-indicator';
import '@m3e/web/menu';
import '@m3e/web/nav-bar';
import '@m3e/web/nav-rail';
import '@m3e/web/option';
import '@m3e/web/progress-indicator';
import '@m3e/web/search';
import '@m3e/web/select';
import '@m3e/web/shape';
import '@m3e/web/snackbar';
import '@m3e/web/split-button';
import '@m3e/web/switch';
import '@m3e/web/theme';
import '@m3e/web/tooltip';

import type { Host } from './core/host';
import { JsonFileSettingsStore, settingsPath } from './core/settings/settings-store';
import { AppModel } from './state/app-model';
import { LocaleController, setApp, ShellController } from './ui/context';

async function createHost(): Promise<Host> {
  switch (import.meta.env.MODE) {
    case 'tauri':
      return (await import('./hosts/tauri')).createTauriHost();
    case 'wails':
      return (await import('./hosts/wails')).createWailsHost();
    default:
      return (await import('./hosts/dev')).createDevHost();
  }
}

/**
 * The locales of the desktop, most preferred first, as GLib reads them, then
 * those of the web view.
 */
function systemLocales(environment: Readonly<Record<string, string>>): string[] {
  const locales = [
    ...(environment['LANGUAGE'] ?? '').split(':'),
    environment['LC_ALL'] ?? '',
    environment['LC_MESSAGES'] ?? '',
    environment['LANG'] ?? '',
  ].filter((locale) => locale !== '' && locale !== 'C' && locale !== 'POSIX' && !locale.startsWith('C.'));
  return [...locales, ...navigator.languages];
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
  setApp({ model, host, shell: new ShellController(), locale: new LocaleController(systemLocales(environment)) });
  // Defines <vfox-plus-app>, which the page already contains.
  await import('./ui/app');
  await model.start();
  if (__VFOX_PLUS_E2E__) {
    await (await import('./e2e/runner')).runScenario(host);
  }
}

main().catch((error: unknown) => {
  console.error(error);
  document.body.textContent = `vfox+ could not start: ${String(error)}`;
});
