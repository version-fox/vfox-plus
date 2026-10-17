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

/** Persistence of {@link AppSettings}. */

import type { FileSystem } from '../host';
import { DEFAULT_SETTINGS, settingsFromJson, settingsToJson, type AppSettings } from './app-settings';

/** Loads and saves {@link AppSettings}. */
export interface SettingsStore {
  /** Loads the settings, or returns the defaults if none are stored. */
  load(): Promise<AppSettings>;
  /** Persists `settings`. */
  save(settings: AppSettings): Promise<void>;
}

/**
 * The settings file for `environment`: `$XDG_CONFIG_HOME/vfox-plus/settings.json`,
 * where `XDG_CONFIG_HOME` defaults to `~/.config` as defined by the XDG Base
 * Directory Specification.
 */
export function settingsPath(environment: Readonly<Record<string, string>>, fallbackHome = '/tmp'): string {
  const xdgConfigHome = environment['XDG_CONFIG_HOME'];
  const base =
    xdgConfigHome?.startsWith('/') === true ? xdgConfigHome : `${environment['HOME'] ?? fallbackHome}/.config`;
  return `${base}/vfox-plus/settings.json`;
}

/**
 * Stores settings as JSON in a file, by default the one of
 * {@link settingsPath}.
 *
 * Corrupted files are ignored rather than fatal: the application then starts
 * with the defaults.
 */
export class JsonFileSettingsStore implements SettingsStore {
  constructor(
    private readonly files: FileSystem,
    /** The settings file. */
    readonly path: string,
  ) {}

  async load(): Promise<AppSettings> {
    try {
      const text = await this.files.readTextFile(this.path);
      return text === null ? DEFAULT_SETTINGS : settingsFromJson(JSON.parse(text));
    } catch {
      return DEFAULT_SETTINGS;
    }
  }

  async save(settings: AppSettings): Promise<void> {
    // The host writes a temporary file first, so that a crash never leaves a
    // truncated settings file behind.
    await this.files.writeTextFile(this.path, `${JSON.stringify(settingsToJson(settings), null, 2)}\n`);
  }
}

/** Keeps settings in memory. Used by tests and as a fallback. */
export class MemorySettingsStore implements SettingsStore {
  /** How many times {@link save} was called. */
  saveCount = 0;

  constructor(private settings: AppSettings = DEFAULT_SETTINGS) {}

  load(): Promise<AppSettings> {
    return Promise.resolve(this.settings);
  }

  save(settings: AppSettings): Promise<void> {
    this.saveCount++;
    this.settings = settings;
    return Promise.resolve();
  }
}
