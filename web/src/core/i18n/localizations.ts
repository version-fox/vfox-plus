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
 * Translations read from the ARB catalogues in `shared/i18n`.
 *
 * The catalogues are bundled as they are (see `vite.config.ts`); nothing is
 * converted or copied. English is the template and the fallback.
 */

import type { LanguagePreference } from '../settings/app-settings';
import { LANGUAGE_CODES } from '../settings/app-settings';
import { formatMessage, parseMessage, type MessageArguments, type MessagePart } from './message-format';

/** An ARB catalogue: messages and their `@`-prefixed metadata. */
export type ArbCatalogue = Readonly<Record<string, unknown>>;

/** The languages with a catalogue, by language code. */
export const SUPPORTED_LANGUAGES = ['en', 'zh'] as const;

/** A language code with a catalogue. */
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];

/**
 * Picks the catalogue for `preference`. With `system`, the desktop locales are
 * matched against the catalogues; every Chinese locale gets Simplified
 * Chinese and English is the fallback.
 */
export function resolveLanguage(preference: LanguagePreference, systemLocales: readonly string[]): SupportedLanguage {
  const code = LANGUAGE_CODES[preference];
  if (code === 'zh' || code === 'en') {
    return code;
  }
  for (const locale of systemLocales) {
    const language = locale.toLowerCase().split(/[-_.@]/)[0];
    if (language === 'zh') {
      return 'zh';
    }
    if (language === 'en') {
      return 'en';
    }
  }
  return 'en';
}

/** The messages of one language. */
export class Localizations {
  private readonly cache = new Map<string, MessagePart[]>();

  constructor(
    /** The language code of {@link messages}. */
    readonly language: SupportedLanguage,
    private readonly messages: ArbCatalogue,
    private readonly fallback: ArbCatalogue,
  ) {}

  /** The BCP 47 tag of the language, for the `lang` attribute. */
  get languageTag(): string {
    return this.language === 'zh' ? 'zh-Hans' : 'en';
  }

  /**
   * The message `key` formatted with `args`.
   *
   * A key that the language lacks falls back to English; a key that no
   * catalogue has is returned as it is, which the tests catch.
   */
  t(key: string, args: MessageArguments = {}): string {
    let parts = this.cache.get(key);
    if (parts === undefined) {
      const source = this.messages[key] ?? this.fallback[key];
      parts = typeof source === 'string' ? parseMessage(source) : [{ kind: 'text', text: key }];
      this.cache.set(key, parts);
    }
    return formatMessage(parts, args, this.language);
  }
}

/** The message keys of a catalogue, without metadata. */
export function messageKeys(catalogue: ArbCatalogue): string[] {
  return Object.keys(catalogue).filter((key) => !key.startsWith('@'));
}
