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
 * What every element of the user interface needs: the application model, the
 * host, the navigation state and the translations. There is one application
 * per page, so these live in a module rather than in a context protocol.
 */

import en from '@shared/i18n/app_en.arb';
import zh from '@shared/i18n/app_zh.arb';

import type { Host } from '../core/host';
import { Localizations, resolveLanguage } from '../core/i18n/localizations';
import type { MessageArguments } from '../core/i18n/message-format';
import type { AppModel } from '../state/app-model';
import { ChangeNotifier } from '../state/notifier';
import { commandSubject, VfoxError, type VfoxCommand } from '../core/vfox/command';

/** The top-level destinations of the application. */
export type Destination = 'home' | 'sdks' | 'plugins' | 'tasks' | 'settings';

/** The destinations in the order of the navigation. */
export const DESTINATIONS: readonly Destination[] = ['home', 'sdks', 'plugins', 'tasks', 'settings'];

/** Navigation state of the application shell. */
export class ShellController extends ChangeNotifier {
  private visible: Destination = 'home';
  private sdk: string | null = null;

  /** The visible destination. */
  get destination(): Destination {
    return this.visible;
  }

  /** The SDK shown on the SDKs page, if any. */
  get selectedSdk(): string | null {
    return this.sdk;
  }

  /** Shows `destination`. */
  go(destination: Destination): void {
    if (destination !== this.visible) {
      this.visible = destination;
      this.notifyListeners();
    }
  }

  /** Shows the SDKs page with `name` selected, or with no selection. */
  showSdk(name: string | null): void {
    this.visible = 'sdks';
    this.sdk = name;
    this.notifyListeners();
  }
}

/** The translations of the current language; notifies when it changes. */
export class LocaleController extends ChangeNotifier {
  private current: Localizations;

  constructor(private readonly systemLocales: readonly string[]) {
    super();
    this.current = this.create('system');
  }

  /** The translations in effect. */
  get l10n(): Localizations {
    return this.current;
  }

  /** Switches to the language of `preference`. */
  apply(preference: Parameters<typeof resolveLanguage>[0]): void {
    const next = this.create(preference);
    if (next.language !== this.current.language) {
      this.current = next;
      this.notifyListeners();
    }
  }

  private create(preference: Parameters<typeof resolveLanguage>[0]): Localizations {
    const language = resolveLanguage(preference, this.systemLocales);
    return new Localizations(language, language === 'zh' ? zh : en, en);
  }
}

/** The pieces of the running application. */
export interface App {
  readonly model: AppModel;
  readonly host: Host;
  readonly shell: ShellController;
  readonly locale: LocaleController;
}

let current: App | null = null;

/** Makes `app` the running application. */
export function setApp(app: App): void {
  current = app;
}

/** The running application. */
export function app(): App {
  if (current === null) {
    throw new Error('The application has not been started.');
  }
  return current;
}

/** The message `key` of the current language, formatted with `args`. */
export function t(key: string, args: MessageArguments = {}): string {
  return app().locale.l10n.t(key, args);
}

/** The localized title of a task that runs `command`. */
export function taskTitle(command: VfoxCommand): string {
  const subject = commandSubject(command) ?? '';
  switch (command.operation) {
    case 'install':
      return t('taskInstall', { subject });
    case 'uninstall':
      return t('taskUninstall', { subject });
    case 'use':
      return t('taskUse', { subject });
    case 'unuse':
      return t('taskUnuse', { subject });
    case 'addPlugin':
      return subject === '' ? t('taskAddPluginFromSource') : t('taskAddPlugin', { subject });
    case 'removePlugin':
      return t('taskRemovePlugin', { subject });
    case 'updatePlugin':
      return t('taskUpdatePlugin', { subject });
    case 'updateAllPlugins':
      return t('taskUpdateAllPlugins');
  }
}

/** A short, user-facing description of `error`. */
export function describeError(error: unknown): string {
  if (error instanceof VfoxError || error instanceof Error) {
    return error.message;
  }
  return String(error);
}

/** Links that the user interface opens. */
export const LINKS = {
  vfoxInstall: 'https://vfox.dev/guides/quick-start.html',
  vfoxDocs: 'https://vfox.dev/',
  repository: 'https://github.com/version-fox/vfox-plus',
} as const;
