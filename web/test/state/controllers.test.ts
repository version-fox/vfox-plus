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

import { afterEach, beforeEach, describe, expect, it } from 'vitest';

import { DEFAULT_SETTINGS } from '../../src/core/settings/app-settings';
import { MemorySettingsStore } from '../../src/core/settings/settings-store';
import { launchConfig, VfoxCli } from '../../src/core/vfox/cli';
import { VfoxError } from '../../src/core/vfox/command';
import { VfoxLocator } from '../../src/core/vfox/locator';
import { AppModel } from '../../src/state/app-model';
import { AsyncResource } from '../../src/state/async-resource';
import {
  SdkDetailController,
  SettingsController,
  VfoxConfigController,
  VfoxController,
} from '../../src/state/controllers';
import { listenAll } from '../../src/state/notifier';
import { fixtureOutput, fixtureRunner, MemoryFileSystem, settle, type FakeProcessRunner } from '../support/fakes';

describe('AsyncResource', () => {
  it('keeps the previous value while reloading and records errors', async () => {
    let calls = 0;
    const resource = new AsyncResource(() => {
      calls++;
      return calls === 2 ? Promise.reject(new Error('boom')) : Promise.resolve(calls);
    });
    await resource.load();
    expect(resource.value).toBe(1);
    expect(resource.hasValue).toBe(true);
    const reload = resource.load();
    expect(resource.isLoading).toBe(true);
    expect(resource.value).toBe(1);
    await reload;
    expect(resource.error).toEqual(new Error('boom'));
    expect(resource.value).toBe(1);
    expect(resource.isLoading).toBe(false);
  });

  it('lets the latest of overlapping loads win', async () => {
    const resolvers: ((value: string) => void)[] = [];
    const resource = new AsyncResource(
      () =>
        new Promise<string>((resolve) => {
          resolvers.push(resolve);
        }),
    );
    const first = resource.load();
    const second = resource.load();
    resolvers[1]?.('new');
    resolvers[0]?.('old');
    await Promise.all([first, second]);
    expect(resource.value).toBe('new');
  });

  it('ensureLoaded loads once, and setValue replaces the value', async () => {
    let calls = 0;
    const resource = new AsyncResource(() => Promise.resolve(++calls));
    await Promise.all([resource.ensureLoaded(), resource.ensureLoaded()]);
    await resource.ensureLoaded();
    expect(calls).toBe(1);
    resource.setValue(9);
    expect(resource.value).toBe(9);
  });

  it('notifies listeners until disposed', async () => {
    const resource = new AsyncResource(() => Promise.resolve(1));
    let notified = 0;
    const stop = listenAll([resource], () => notified++);
    await resource.load();
    expect(notified).toBe(2);
    stop();
    await resource.load();
    expect(notified).toBe(2);
    resource.dispose();
    expect(resource.isDisposed).toBe(true);
  });
});

describe('SettingsController', () => {
  it('persists changes and skips no-ops', async () => {
    const store = new MemorySettingsStore({ ...DEFAULT_SETTINGS, theme: 'dark' });
    const controller = new SettingsController(store);
    await controller.load();
    expect(controller.settings.theme).toBe('dark');
    await controller.update((s) => ({ ...s, theme: 'dark' }));
    expect(store.saveCount).toBe(0);
    await controller.update((s) => ({ ...s, theme: 'light' }));
    expect(store.saveCount).toBe(1);
    expect((await store.load()).theme).toBe('light');
  });
});

describe('VfoxController', () => {
  let runner: FakeProcessRunner;
  let cli: VfoxCli;

  beforeEach(() => {
    runner = fixtureRunner();
    cli = new VfoxCli(runner, launchConfig({ executable: 'vfox' }));
  });

  const controller = (found?: string): VfoxController =>
    new VfoxController(cli, new VfoxLocator({}, (path) => Promise.resolve(path === found)), { HOME: '/home/u' });

  it('uses the detected executable and reads the version', async () => {
    const vfox = controller('/usr/local/bin/vfox');
    await vfox.apply({ ...DEFAULT_SETTINGS, extraEnvironment: { A: '1' } });
    expect(vfox.availability).toBe('ready');
    expect(vfox.version).toBe('1.0.13');
    expect(vfox.executable).toBe('/usr/local/bin/vfox');
    expect(vfox.isCustomExecutable).toBe(false);
    expect(vfox.error).toBeNull();
    expect(cli.launchConfig.extraEnvironment).toEqual({ A: '1' });
    expect(runner.invocations.map((call) => call.executable)).toEqual(['/usr/local/bin/vfox']);
  });

  it('prefers the executable chosen by the user', async () => {
    const vfox = controller('/usr/local/bin/vfox');
    await vfox.apply({ ...DEFAULT_SETTINGS, vfoxExecutable: '/opt/vfox' });
    expect(vfox.executable).toBe('/opt/vfox');
    expect(vfox.isCustomExecutable).toBe(true);
  });

  it('reports a missing and a broken executable', async () => {
    const vfox = controller();
    runner.missingExecutable = true;
    await vfox.apply(DEFAULT_SETTINGS);
    expect(vfox.availability).toBe('notFound');
    expect(vfox.executable).toBeNull();
    runner.missingExecutable = false;
    runner.on('--version', { exitCode: 0, stdout: 'hello', stderr: '' });
    await vfox.check();
    expect(vfox.availability).toBe('broken');
    expect(vfox.error).toBeInstanceOf(VfoxError);
  });
});

describe('SdkDetailController', () => {
  it('filters available versions by text, note and installation', async () => {
    const controller = new SdkDetailController(new VfoxCli(fixtureRunner(), launchConfig()), 'nodejs');
    await controller.versions.load();
    const installed = new Set(['24.11.1']);
    expect(controller.visibleVersions(installed)).toHaveLength(872);
    controller.filter = 'withNote';
    expect(controller.visibleVersions(installed).every((version) => version.note === 'LTS')).toBe(true);
    controller.filter = 'all';
    controller.query = '24.11.';
    expect(controller.query).toBe('24.11.');
    expect(controller.visibleVersions(installed).map((version) => version.version)).toEqual(['24.11.1', '24.11.0']);
    controller.query = 'lts';
    expect(controller.visibleVersions(installed).length).toBeGreaterThan(10);
    controller.query = '';
    controller.filter = 'installed';
    expect(controller.filter).toBe('installed');
    expect(controller.visibleVersions(installed).map((version) => version.version)).toEqual(['24.11.1']);
    controller.dispose();
  });
});

describe('VfoxConfigController', () => {
  it('saves and reloads, and exposes values by key', async () => {
    const runner = fixtureRunner().on('config gitignore.enable false', { exitCode: 0, stdout: '', stderr: '' });
    const controller = new VfoxConfigController(new VfoxCli(runner, launchConfig()));
    await controller.load();
    expect(controller.valueFor('gitignore.enable')).toBe('true');
    expect(controller.valueFor('missing')).toBeNull();
    const saving = controller.set('gitignore.enable', 'false');
    expect(controller.isSaving('gitignore.enable')).toBe(true);
    await saving;
    expect(controller.isSaving('gitignore.enable')).toBe(false);
    expect(runner.calls('config --list')).toHaveLength(2);
    runner.on('config --unset proxy.url', { exitCode: 0, stdout: '', stderr: '' });
    await controller.reset('proxy.url');
    expect(runner.calls('config --unset proxy.url')).toHaveLength(1);
  });

  it('rethrows rejected values after reloading', async () => {
    const runner = fixtureRunner().on(
      'config cache.availableHookDuration x',
      fixtureOutput('config-set-invalid-duration', 1),
    );
    const controller = new VfoxConfigController(new VfoxCli(runner, launchConfig()));
    await expect(controller.set('cache.availableHookDuration', 'x')).rejects.toBeInstanceOf(VfoxError);
    expect(controller.hasValue).toBe(true);
  });
});

describe('AppModel', () => {
  let runner: FakeProcessRunner;
  let files: MemoryFileSystem;
  let model: AppModel;

  const create = (environment: Record<string, string>): AppModel =>
    new AppModel({
      runner,
      files,
      settingsStore: new MemorySettingsStore(),
      environment,
      locator: new VfoxLocator({}, (path) => Promise.resolve(path === '/usr/bin/vfox')),
    });

  beforeEach(() => {
    runner = fixtureRunner();
    files = new MemoryFileSystem();
    model = create({ HOME: '/home/u' });
  });

  afterEach(() => {
    model.dispose();
  });

  it('starts by probing vfox and loading the installed SDKs', async () => {
    await model.start();
    await settle();
    expect(model.vfox.availability).toBe('ready');
    expect(model.installed.value?.map((sdk) => sdk.name)).toEqual(['maven', 'mvn2', 'nodejs']);
    expect(model.installedSdk('nodejs')?.current).toBe('24.11.1');
    expect(model.installedSdk('python')).toBeNull();
    expect(model.sdkDetail('nodejs').name).toBe('nodejs');
  });

  it('reloads the installed SDKs after a task', async () => {
    await model.start();
    await settle();
    const before = runner.calls('list').length;
    model.run(model.cli.use('nodejs', '22.21.0'));
    await settle();
    expect(runner.calls('list')).toHaveLength(before + 1);
    expect(model.tasks.tasks[0]?.status).toBe('succeeded');
  });

  it('reloads the registry and the configuration when vfox becomes available again', async () => {
    await model.start();
    await settle();
    await model.registry.load();
    await model.vfoxConfig.load();
    await model.settings.update((s) => ({ ...s, extraEnvironment: { X: '1' } }));
    await settle();
    expect(runner.calls('available')).toHaveLength(2);
    expect(runner.calls('config --list')).toHaveLength(2);
    expect(runner.invocations.at(-1)?.environment['X']).toBe('1');
  });

  it('hands vfox a copy of a local .lua plugin and deletes it afterwards', async () => {
    files.files.set('/home/u/mine.lua', 'PLUGIN = {}');
    await model.start();
    await settle();
    const task = await model.addPluginFromSource('/home/u/mine.lua', 'mine');
    const args = runner.calls('add')[0]?.arguments ?? [];
    const copy = args[args.indexOf('--source') + 1] ?? '';
    // vfox renames the file into its temporary directory, which only works on
    // the same filesystem.
    expect(copy.startsWith('/home/u/.vfox/tmp/vfox-plus-plugin-')).toBe(true);
    expect(args.slice(-2)).toEqual(['--alias', 'mine']);
    await settle();
    expect(task.status).toBe('succeeded');
    expect(files.files.has('/home/u/mine.lua')).toBe(true);
    expect(files.files.has(copy)).toBe(false);
  });

  it('passes URLs to vfox as they are', async () => {
    await model.start();
    await model.addPluginFromSource('https://example.com/p.zip', null);
    expect(runner.calls('add')[0]?.arguments).toEqual(['add', '--source', 'https://example.com/p.zip']);
  });

  it('re-applies the settings when the executable changes', async () => {
    await model.start();
    await settle();
    await model.settings.update((s) => ({ ...s, vfoxExecutable: '/opt/vfox' }));
    await settle();
    expect(model.vfox.executable).toBe('/opt/vfox');
    expect(runner.invocations.at(-1)?.executable).toBe('/opt/vfox');
  });

  it('does not reload after a task that never ran', async () => {
    await model.start();
    await settle();
    const before = runner.calls('list').length;
    const running = model.run(model.cli.updatePlugin('a'));
    const queued = model.run(model.cli.updatePlugin('b'));
    model.tasks.cancel(queued);
    await settle();
    expect(running.status).toBe('succeeded');
    expect(runner.calls('list')).toHaveLength(before + 1);
  });
});
