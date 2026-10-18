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

/** The controllers of the state layer, except the task queue. */

import type { AvailableVersion, ConfigEntry, PluginInfo } from '../core/models';
import { DEFAULT_SETTINGS, settingsEqual, type AppSettings } from '../core/settings/app-settings';
import type { SettingsStore } from '../core/settings/settings-store';
import { VFOX_EXECUTABLE_NAME, type VfoxCli } from '../core/vfox/cli';
import { VfoxError, VfoxNotFoundError } from '../core/vfox/command';
import type { VfoxLocator } from '../core/vfox/locator';
import { AsyncResource } from './async-resource';
import { ChangeNotifier } from './notifier';

/** Holds the {@link AppSettings} and persists every change. */
export class SettingsController extends ChangeNotifier {
  private current: AppSettings = DEFAULT_SETTINGS;

  constructor(private readonly store: SettingsStore) {
    super();
  }

  /** The current settings. */
  get settings(): AppSettings {
    return this.current;
  }

  /** Loads the persisted settings. */
  async load(): Promise<void> {
    this.current = await this.store.load();
    this.notifyListeners();
  }

  /** Applies `change` to the settings and saves the result. */
  async update(change: (settings: AppSettings) => AppSettings): Promise<void> {
    const updated = change(this.current);
    if (settingsEqual(updated, this.current)) {
      return;
    }
    this.current = updated;
    this.notifyListeners();
    await this.store.save(updated);
  }
}

/**
 * Whether vfox can be used:
 *
 * - `unknown`: not checked yet;
 * - `checking`: the executable is being probed;
 * - `ready`: vfox runs and reported its version;
 * - `notFound`: no vfox executable was found;
 * - `broken`: an executable was found but did not behave like vfox.
 */
export type VfoxAvailability = 'unknown' | 'checking' | 'ready' | 'notFound' | 'broken';

/** Locates vfox, configures the {@link VfoxCli} and checks that vfox works. */
export class VfoxController extends ChangeNotifier {
  private state: VfoxAvailability = 'unknown';
  private path: string | null = null;
  private custom = false;
  private detectedVersion: string | null = null;
  private lastError: unknown = null;

  constructor(
    private readonly cli: VfoxCli,
    private readonly locator: VfoxLocator,
    private readonly environment: Readonly<Record<string, string>>,
  ) {
    super();
  }

  /** Whether vfox can be used. */
  get availability(): VfoxAvailability {
    return this.state;
  }

  /** The executable in use, if one was found. */
  get executable(): string | null {
    return this.path;
  }

  /** Whether {@link executable} was chosen by the user rather than detected. */
  get isCustomExecutable(): boolean {
    return this.custom;
  }

  /** The vfox version, once vfox is ready. */
  get version(): string | null {
    return this.detectedVersion;
  }

  /** Why vfox is not available, if it is not. */
  get error(): unknown {
    return this.lastError;
  }

  /**
   * Points the CLI at the executable and environment from `settings` and
   * checks the result.
   */
  async apply(settings: AppSettings): Promise<void> {
    const custom = settings.vfoxExecutable;
    this.custom = custom !== null;
    this.path = custom ?? (await this.locator.locate());
    this.cli.launchConfig = {
      ...this.cli.launchConfig,
      executable: this.path ?? VFOX_EXECUTABLE_NAME,
      baseEnvironment: this.environment,
      extraEnvironment: settings.extraEnvironment,
    };
    await this.check();
  }

  /** Runs `vfox --version` to check that vfox works. */
  async check(): Promise<void> {
    this.state = 'checking';
    this.notifyListeners();
    try {
      this.detectedVersion = await this.cli.version();
      this.lastError = null;
      this.state = 'ready';
    } catch (error) {
      this.detectedVersion = null;
      this.lastError = error;
      this.state = error instanceof VfoxNotFoundError ? 'notFound' : 'broken';
      if (!(error instanceof VfoxError)) {
        console.error(error);
      }
    }
    this.notifyListeners();
  }
}

/** The vfox configuration (`vfox config`), editable key by key. */
export class VfoxConfigController extends AsyncResource<ConfigEntry[]> {
  private readonly saving = new Set<string>();

  constructor(private readonly cli: VfoxCli) {
    super(() => cli.configEntries());
  }

  /** The value of `key`, or `null` if vfox does not know the key. */
  valueFor(key: string): string | null {
    return this.value?.find((entry) => entry.key === key)?.value ?? null;
  }

  /** Whether `key` is being saved. */
  isSaving(key: string): boolean {
    return this.saving.has(key);
  }

  /**
   * Sets `key` to `newValue` and reloads the configuration.
   *
   * Rejects with a `VfoxError` if vfox rejects the value; the configuration is
   * reloaded either way.
   */
  set(key: string, newValue: string): Promise<void> {
    return this.save(key, () => this.cli.setConfig(key, newValue));
  }

  /** Resets `key` to its default and reloads the configuration. */
  reset(key: string): Promise<void> {
    return this.save(key, () => this.cli.unsetConfig(key));
  }

  private async save(key: string, action: () => Promise<void>): Promise<void> {
    this.saving.add(key);
    this.notifyListeners();
    try {
      await action();
    } finally {
      this.saving.delete(key);
      await this.load();
    }
  }
}

/**
 * Which versions are listed on the "available" tab: every version, versions
 * carrying a note such as `LTS`, or versions that are installed already.
 */
export type VersionFilter = 'all' | 'withNote' | 'installed';

/** State of the detail view of one installed plugin. */
export class SdkDetailController extends ChangeNotifier {
  /** Plugin metadata, loaded on demand. */
  readonly info: AsyncResource<PluginInfo>;
  /** Installable versions, loaded on demand because the search can be slow. */
  readonly versions: AsyncResource<AvailableVersion[]>;
  private text = '';
  private shown: VersionFilter = 'all';
  private readonly stopListening: () => void;

  constructor(
    cli: VfoxCli,
    /** The plugin name. */
    readonly name: string,
  ) {
    super();
    this.info = new AsyncResource(() => cli.pluginInfo(name));
    this.versions = new AsyncResource(() => cli.searchVersions(name));
    this.stopListening = this.versions.addListener(() => this.notifyListeners());
  }

  /** The text the available versions are filtered by. */
  get query(): string {
    return this.text;
  }

  set query(value: string) {
    if (value !== this.text) {
      this.text = value;
      this.notifyListeners();
    }
  }

  /** Which available versions are shown. */
  get filter(): VersionFilter {
    return this.shown;
  }

  set filter(value: VersionFilter) {
    if (value !== this.shown) {
      this.shown = value;
      this.notifyListeners();
    }
  }

  /**
   * The available versions matching {@link query} and {@link filter}.
   *
   * `installed` holds the installed versions. It is passed in rather than taken
   * from the search output, so that it stays current after installs without
   * searching again.
   */
  visibleVersions(installed: ReadonlySet<string>): AvailableVersion[] {
    const needle = this.text.trim().toLowerCase();
    return (this.versions.value ?? []).filter(
      (version) =>
        this.matchesFilter(version, installed) &&
        (needle === '' ||
          version.version.toLowerCase().includes(needle) ||
          (version.note?.toLowerCase().includes(needle) ?? false)),
    );
  }

  private matchesFilter(version: AvailableVersion, installed: ReadonlySet<string>): boolean {
    switch (this.shown) {
      case 'all':
        return true;
      case 'withNote':
        return version.note !== null;
      case 'installed':
        return installed.has(version.version);
    }
  }

  override dispose(): void {
    this.stopListening();
    this.info.dispose();
    this.versions.dispose();
    super.dispose();
  }
}
