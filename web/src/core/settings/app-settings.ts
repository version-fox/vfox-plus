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
 * Preferences of the application itself (not of vfox).
 *
 * The JSON keys are part of the shared specification in
 * `docs/architecture.md#settings`, so that every implementation of the GUI can
 * read the same file.
 */

/** Light or dark appearance; `system` follows the desktop. */
export type ThemePreference = 'system' | 'light' | 'dark';

/** The user's language choice; `system` follows the desktop locale. */
export type LanguagePreference = 'system' | 'english' | 'simplifiedChinese';

/** The language codes of the translation catalogues, by preference. */
export const LANGUAGE_CODES: Readonly<Record<LanguagePreference, string | null>> = {
  system: null,
  english: 'en',
  simplifiedChinese: 'zh',
};

/**
 * The brand colour used when the desktop accent colour is not used or not
 * available: the orange of the vfox logo.
 */
export const DEFAULT_SEED_COLOR = 0xfff28c1b;

/** Preferences of the application, persisted as JSON. */
export interface AppSettings {
  /** Light or dark appearance. */
  readonly theme: ThemePreference;
  /** Whether to derive the colour scheme from the desktop accent colour. */
  readonly useSystemColor: boolean;
  /** ARGB seed of the colour scheme when {@link useSystemColor} is off. */
  readonly seedColor: number;
  /** The user interface language. */
  readonly language: LanguagePreference;
  /** Path of the vfox executable chosen by the user, or `null` to detect it. */
  readonly vfoxExecutable: string | null;
  /** Additional environment variables for vfox, such as mirror settings. */
  readonly extraEnvironment: Readonly<Record<string, string>>;
  /** Whether the navigation rail shows labels next to the icons. */
  readonly railExpanded: boolean;
  /**
   * Keys of the settings file that this version does not know, kept as they
   * are so that settings written by another implementation survive.
   */
  readonly unknown: Readonly<Record<string, unknown>>;
}

/** The settings of a fresh installation. */
export const DEFAULT_SETTINGS: AppSettings = Object.freeze({
  theme: 'system',
  useSystemColor: true,
  seedColor: DEFAULT_SEED_COLOR,
  language: 'system',
  vfoxExecutable: null,
  extraEnvironment: Object.freeze({}),
  railExpanded: false,
  unknown: Object.freeze({}),
});

const KNOWN_KEYS = new Set([
  'theme',
  'useSystemColor',
  'seedColor',
  'language',
  'vfoxExecutable',
  'extraEnvironment',
  'railExpanded',
]);

/** Restores settings from `json`, ignoring malformed values. */
export function settingsFromJson(json: unknown): AppSettings {
  if (typeof json !== 'object' || json === null || Array.isArray(json)) {
    return DEFAULT_SETTINGS;
  }
  const map = json as Record<string, unknown>;
  const environment: Record<string, string> = {};
  const rawEnvironment = map['extraEnvironment'];
  if (typeof rawEnvironment === 'object' && rawEnvironment !== null && !Array.isArray(rawEnvironment)) {
    for (const [key, value] of Object.entries(rawEnvironment)) {
      if (typeof value === 'string') {
        environment[key] = value;
      }
    }
  }
  const unknown: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(map)) {
    if (!KNOWN_KEYS.has(key)) {
      unknown[key] = value;
    }
  }
  const executable = map['vfoxExecutable'];
  const seed = map['seedColor'];
  return {
    theme: oneOf(map['theme'], ['system', 'light', 'dark'], 'system'),
    useSystemColor: typeof map['useSystemColor'] === 'boolean' ? map['useSystemColor'] : true,
    seedColor: typeof seed === 'number' && Number.isInteger(seed) ? seed : DEFAULT_SEED_COLOR,
    language: oneOf(map['language'], ['system', 'english', 'simplifiedChinese'], 'system'),
    vfoxExecutable: typeof executable === 'string' && executable !== '' ? executable : null,
    extraEnvironment: environment,
    railExpanded: map['railExpanded'] === true,
    unknown,
  };
}

/** Serialises the settings, with the keys in the order of the specification. */
export function settingsToJson(settings: AppSettings): Record<string, unknown> {
  return {
    theme: settings.theme,
    useSystemColor: settings.useSystemColor,
    seedColor: settings.seedColor,
    language: settings.language,
    ...(settings.vfoxExecutable === null ? {} : { vfoxExecutable: settings.vfoxExecutable }),
    extraEnvironment: settings.extraEnvironment,
    railExpanded: settings.railExpanded,
    ...settings.unknown,
  };
}

/** Whether two settings are equal, comparing environments by value. */
export function settingsEqual(a: AppSettings, b: AppSettings): boolean {
  return JSON.stringify(settingsToJson(a)) === JSON.stringify(settingsToJson(b));
}

/** Whether two sets of environment variables are equal. */
export function environmentEqual(a: Readonly<Record<string, string>>, b: Readonly<Record<string, string>>): boolean {
  const keys = Object.keys(a);
  return keys.length === Object.keys(b).length && keys.every((key) => b[key] === a[key]);
}

/** The result of {@link parseEnvironmentLines}. */
export interface ParsedEnvironment {
  /** The variables of the valid lines. */
  readonly variables: Record<string, string>;
  /** The 1-based numbers of the lines that could not be parsed. */
  readonly invalidLines: number[];
}

/**
 * Parses `KEY=VALUE` lines into environment variables.
 *
 * Blank lines and lines starting with `#` are ignored. Keys must be valid POSIX
 * variable names.
 */
export function parseEnvironmentLines(text: string): ParsedEnvironment {
  const variables: Record<string, string> = {};
  const invalidLines: number[] = [];
  const keyPattern = /^[A-Za-z_][A-Za-z0-9_]*$/;
  text.split('\n').forEach((rawLine, index) => {
    const line = rawLine.trim();
    if (line === '' || line.startsWith('#')) {
      return;
    }
    const equals = line.indexOf('=');
    const key = equals < 0 ? line : line.slice(0, equals).trim();
    if (equals < 0 || !keyPattern.test(key)) {
      invalidLines.push(index + 1);
      return;
    }
    variables[key] = line.slice(equals + 1).trim();
  });
  return { variables, invalidLines };
}

/** Formats environment variables as `KEY=VALUE` lines, sorted by key. */
export function formatEnvironmentLines(variables: Readonly<Record<string, string>>): string {
  return Object.keys(variables)
    .sort()
    .map((key) => `${key}=${variables[key]}`)
    .join('\n');
}

function oneOf<T extends string>(value: unknown, choices: readonly T[], fallback: T): T {
  return choices.find((choice) => choice === value) ?? fallback;
}
