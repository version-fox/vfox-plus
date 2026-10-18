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

import type { FileSystem } from '../core/host';
import type { AvailablePlugin, InstalledSdk } from '../core/models';
import type { ProcessRunner } from '../core/process';
import { environmentEqual, type AppSettings } from '../core/settings/app-settings';
import type { SettingsStore } from '../core/settings/settings-store';
import { VfoxCli, launchConfig, vfoxEnvironment } from '../core/vfox/cli';
import type { VfoxCommand } from '../core/vfox/command';
import { VfoxLocator } from '../core/vfox/locator';
import { preparePluginSource, vfoxTemporaryDirectory, type PreparedPluginSource } from '../core/vfox/plugin-source';
import { AsyncResource } from './async-resource';
import {
  SdkDetailController,
  SettingsController,
  VfoxConfigController,
  VfoxController,
  type VfoxAvailability,
} from './controllers';
import { TaskQueue, type VfoxTask } from './task-queue';

/** What the application model is made of. */
export interface AppModelOptions {
  /** Spawns vfox. */
  readonly runner: ProcessRunner;
  /** File system access, for the locator and for plugin sources. */
  readonly files: FileSystem;
  /** Persists the settings. */
  readonly settingsStore: SettingsStore;
  /** The environment of the application. */
  readonly environment: Readonly<Record<string, string>>;
  /** Finds vfox; defaults to a locator that searches `PATH` and the usual places. */
  readonly locator?: VfoxLocator;
}

/**
 * The application state: every controller, wired together.
 *
 * The model owns the {@link VfoxCli} and keeps the derived data consistent:
 * installed SDKs are reloaded whenever vfox becomes available or a task that
 * changes them finishes.
 */
export class AppModel {
  /** The vfox client shared by all controllers. */
  readonly cli: VfoxCli;
  /** Application preferences. */
  readonly settings: SettingsController;
  /** Availability of the vfox executable. */
  readonly vfox: VfoxController;
  /** Installed plugins and their versions. */
  readonly installed: AsyncResource<InstalledSdk[]>;
  /** Plugins available in the registry. */
  readonly registry: AsyncResource<AvailablePlugin[]>;
  /** The vfox configuration. */
  readonly vfoxConfig: VfoxConfigController;
  /** State-changing operations. */
  readonly tasks: TaskQueue;

  private readonly files: FileSystem;
  private readonly pluginCopies = new Map<VfoxTask, PreparedPluginSource>();
  private readonly stops: (() => void)[] = [];
  private lastAvailability: VfoxAvailability = 'unknown';
  private appliedSettings: AppSettings | null = null;

  constructor(options: AppModelOptions) {
    this.files = options.files;
    this.cli = new VfoxCli(options.runner, launchConfig({ baseEnvironment: options.environment }));
    this.settings = new SettingsController(options.settingsStore);
    this.vfox = new VfoxController(
      this.cli,
      options.locator ?? new VfoxLocator(options.environment, (path) => options.files.isExecutable(path)),
      options.environment,
    );
    this.installed = new AsyncResource(() => this.cli.installedSdks());
    this.registry = new AsyncResource(() => this.cli.availablePlugins());
    this.vfoxConfig = new VfoxConfigController(this.cli);
    this.tasks = new TaskQueue(this.cli);
    this.stops.push(
      this.tasks.onFinished((task) => this.onTaskFinished(task)),
      this.vfox.addListener(() => this.onVfoxChanged()),
      this.settings.addListener(() => this.onSettingsChanged()),
    );
  }

  /** Loads the settings and probes vfox. */
  async start(): Promise<void> {
    await this.settings.load();
    await this.applySettings();
  }

  /** The installed SDK called `name`, if it is installed. */
  installedSdk(name: string): InstalledSdk | null {
    return this.installed.value?.find((sdk) => sdk.name === name) ?? null;
  }

  /** Creates the controller for the detail view of `name`. */
  sdkDetail(name: string): SdkDetailController {
    return new SdkDetailController(this.cli, name);
  }

  /** Queues `command` and returns its task. */
  run(command: VfoxCommand): VfoxTask {
    return this.tasks.enqueue(command);
  }

  /**
   * Adds a plugin from `source`, a URL or a local `.zip` or `.lua` file,
   * optionally under `alias`.
   *
   * vfox moves a local `.lua` file away instead of reading it, so such a file
   * is copied into the temporary directory of vfox first and vfox gets the
   * copy; the copy is deleted once the task has finished.
   */
  async addPluginFromSource(source: string, alias: string | null): Promise<VfoxTask> {
    const temporaryRoot = await vfoxTemporaryDirectory(vfoxEnvironment(this.cli.launchConfig), this.files);
    const prepared = await preparePluginSource(source, temporaryRoot, this.files);
    const task = this.run(this.cli.addPlugin('', { source: prepared.path, alias }));
    if (prepared.isCopy) {
      this.pluginCopies.set(task, prepared);
    }
    return task;
  }

  private onVfoxChanged(): void {
    const availability = this.vfox.availability;
    if (availability === this.lastAvailability) {
      return;
    }
    this.lastAvailability = availability;
    if (availability === 'ready') {
      void this.installed.load();
      if (this.registry.hasValue || this.registry.error !== null) {
        void this.registry.load();
      }
      if (this.vfoxConfig.hasValue || this.vfoxConfig.error !== null) {
        void this.vfoxConfig.load();
      }
    }
  }

  private onSettingsChanged(): void {
    const previous = this.appliedSettings;
    const current = this.settings.settings;
    if (
      previous !== null &&
      (previous.vfoxExecutable !== current.vfoxExecutable ||
        !environmentEqual(previous.extraEnvironment, current.extraEnvironment))
    ) {
      void this.applySettings();
    }
  }

  private async applySettings(): Promise<void> {
    this.appliedSettings = this.settings.settings;
    // Force a reload of the dependent data even if vfox stays available.
    this.lastAvailability = 'unknown';
    await this.vfox.apply(this.settings.settings);
  }

  private onTaskFinished(task: VfoxTask): void {
    const copy = this.pluginCopies.get(task);
    if (copy !== undefined) {
      this.pluginCopies.delete(task);
      void copy.dispose();
    }
    if (task.status === 'cancelled' && task.startedAt === null) {
      return;
    }
    void this.installed.load();
  }

  /** Releases every resource. */
  dispose(): void {
    this.stops.forEach((stop) => stop());
    for (const copy of this.pluginCopies.values()) {
      void copy.dispose();
    }
    this.pluginCopies.clear();
    this.tasks.dispose();
    this.installed.dispose();
    this.registry.dispose();
    this.vfoxConfig.dispose();
    this.vfox.dispose();
    this.settings.dispose();
  }
}
