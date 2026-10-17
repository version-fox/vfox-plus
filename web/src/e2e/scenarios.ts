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
 * The end-to-end scenarios, the same as those of the Flutter implementation
 * (`flutter/integration_test`). `e2e/run.sh` prepares the environment and
 * starts the application with `VFOX_PLUS_E2E_SCENARIO` set to one of them.
 */

import type { Host } from '../core/host';
import { app, taskTitle } from '../ui/context';
import {
  check,
  clickButton,
  clickButtonNear,
  clickElement,
  clickText,
  fill,
  findText,
  ownText,
  pause,
  waitFor,
  waitForText,
} from './driver';

/** What a scenario can use. */
export interface ScenarioContext {
  readonly host: Host;
  readonly environment: Readonly<Record<string, string>>;
  /** Reports progress on the standard output of the application. */
  step(message: string): Promise<void>;
}

/** Runs vfox as the tests see it: in the sandbox, without colours. */
async function vfox(context: ScenarioContext, args: string[]): Promise<string> {
  const executable = context.environment['VFOX_PLUS_E2E_VFOX'] ?? 'vfox';
  const output = await context.host.run(executable, args, {
    workingDirectory: '/',
    environment: { ...context.environment, NO_COLOR: '1' },
  });
  return output.stdout;
}

/** Waits until the task with `title` (in English) has succeeded; fails if it failed. */
async function waitForTask(title: string): Promise<void> {
  await waitFor(`the task "${title}" to finish`, () => {
    const task = app().model.tasks.tasks.find((candidate) => taskTitle(candidate.command) === title);
    if (task?.status === 'failed') {
      throw new Error(`${title} failed: ${task.errorMessage ?? ''}`);
    }
    return task?.status === 'succeeded';
  });
}

/** The application against the fake vfox of `shared/fixtures/fake-vfox.sh`, which replays the fixtures. */
export async function fakeScenario(context: ScenarioContext): Promise<void> {
  const { model } = app();
  await waitForText('Versions in use');
  await context.step('home');
  await waitForText(`vfox 1.0.13 · ${model.vfox.executable ?? ''}`);
  check(
    model.vfox.executable === context.environment['VFOX_PLUS_E2E_VFOX'],
    'the executable from the settings is used',
  );

  await clickButton('SDKs');
  await clickElement('the SDK nodejs', 'm3e-list-action', (element) => ownText(element).startsWith('nodejs'));
  await waitForText('Node.js runtime environment.');
  await context.step('detail');

  // Select 22.21.0 as the global default.
  await clickButton('Use');
  await waitForTask('Use nodejs@22.21.0');

  // Install a version from the available tab.
  await clickButton('Available');
  await waitForText('872 versions');
  await fill('Filter versions', '24.11.0');
  await waitForText('1 version');
  await clickButtonNear('24.11.0', 'Install');
  await waitForTask('Install nodejs@24.11.0');

  const tasks = model.tasks.tasks;
  check(
    tasks.every((task) => task.status === 'succeeded'),
    'every task succeeded',
  );
  check(tasks[0]?.progress === 1, 'the download progress reached 100 %');
  check(tasks[0]?.log.includes('Install nodejs@24.11.0 success!') === true, 'the output of vfox is kept');
}

/** Every operation against the real vfox and the offline test plugin of `shared/test-plugin`. */
export async function realScenario(context: ScenarioContext): Promise<void> {
  const { model } = app();
  const { host, environment } = context;
  const server = environment['VFOX_PLUS_TEST_SERVER'] ?? '';
  const home = environment['HOME'] ?? '';
  const localPlugin = environment['VFOX_PLUS_E2E_LOCAL_PLUGIN'] ?? '';
  const control = (action: string): Promise<unknown> =>
    host.run('curl', ['-fsS', `${server}/control/${action}`], { workingDirectory: '/', environment });

  await waitForText('Versions in use');
  check(model.installed.value?.length === 0, 'no plugin is installed yet');

  // Point vfox to the local registry through the vfox configuration editor.
  await context.step('configure the registry');
  await clickButton('Settings');
  await fill('Plugin registry', `${server}/registry`, { submit: true });
  await waitForText('Saved');
  check(
    (await vfox(context, ['config', 'registry.address'])).trim() === `${server}/registry`,
    'vfox stored the registry',
  );

  // Add the plugin from the registry.
  await context.step('add a plugin from the registry');
  await clickButton('Plugins');
  await waitForText('testsdk');
  await clickButton('Add');
  await waitForTask('Add plugin testsdk');
  await waitForText('Added');

  // Add a local copy of the plugin under another name. vfox moves local .lua
  // files away, so the application must hand it a copy.
  await context.step('add a plugin from a local file');
  await clickButton('Add from source…');
  await waitForText('Add a plugin from source');
  await fill('Source', localPlugin);
  await fill('Name (optional)', 'localsdk');
  await clickButton('Add plugin');
  await waitForTask('Add plugin localsdk');
  check(await host.files.exists(localPlugin), 'the source file survives vfox add --source');

  // Search and install both versions.
  await context.step('install');
  await clickButton('SDKs');
  await clickElement('the SDK testsdk', 'm3e-list-action', (element) => ownText(element).startsWith('testsdk'));
  await waitForText('An offline SDK for the vfox-plus end-to-end tests.');
  await clickButton('Available');
  await waitForText('2 versions');
  // The list is virtualised and renders its items after the count.
  await waitForText('LTS');
  await waitForText('testtool 2.0');
  check(findText('LTS').length === 1, 'the note is shown once');
  await clickButtonNear('2.0.0', 'Install');
  await waitForTask('Install testsdk@2.0.0');
  check(model.tasks.tasks[0]?.progress === 1, 'vfox reports the download progress on standard error');

  // Cancel the installation of 1.0.0 during the download. From 1.0.13 vfox
  // leaves the partial installation behind, but must neither list it nor
  // refuse to install the version again.
  await context.step('cancel an installation');
  await control('hold');
  await clickButtonNear('1.0.0', 'Install');
  await waitFor('the download to start', () => (model.tasks.tasks[0]?.progress ?? 0) > 0);
  await clickButton('Cancel');
  await waitFor('vfox to exit', () => model.tasks.tasks[0]?.finishedAt !== null);
  check(model.tasks.tasks[0]?.status === 'cancelled', 'the task is cancelled');
  check(!(await vfox(context, ['list', 'testsdk'])).includes('1.0.0'), 'vfox does not list the partial version');
  await control('release');
  await pause(500);
  await clickButtonNear('1.0.0', 'Install');
  await waitForTask('Install testsdk@1.0.0');

  // Make 2.0.0 the global default.
  await context.step('use');
  await clickButton('Installed (2)');
  await waitForText('1.0.0');
  await clickButtonNear('2.0.0', 'Use');
  await waitForTask('Use testsdk@2.0.0');
  await waitForText('In use: 2.0.0');
  check(
    ((await host.files.readTextFile(`${home}/.vfox/.vfox.toml`)) ?? '').includes('testsdk = "2.0.0"'),
    'the global .vfox.toml pins 2.0.0',
  );
  check(await host.files.exists(`${home}/.vfox/sdks/testsdk`), 'the SDK is linked');

  // Uninstall 1.0.0.
  await context.step('uninstall');
  await clickButton('More actions for 1.0.0');
  await clickElement('the menu item Uninstall', 'm3e-menu-item', (element) => ownText(element) === 'Uninstall');
  await waitForText('Uninstall testsdk 1.0.0?');
  await clickElement('the confirmation', 'm3e-button', (element) => ownText(element) === 'Uninstall');
  await waitForTask('Uninstall testsdk@1.0.0');
  await waitForText('Installed (1)');

  // Remove the second plugin.
  await context.step('remove');
  await clickElement('the SDK localsdk', 'm3e-list-action', (element) => ownText(element).startsWith('localsdk'));
  await waitForText('localsdk runtime environment.', 5000).catch(() => waitForText('localsdk'));
  await clickButton('More actions');
  await clickElement('the menu item Remove plugin', 'm3e-menu-item', (element) => ownText(element) === 'Remove plugin');
  await waitForText('Remove localsdk?');
  await clickElement('the confirmation', 'm3e-button', (element) => ownText(element) === 'Remove');
  await waitForTask('Remove plugin localsdk');
  await waitForText('Select an SDK to manage its versions.');
  check(!app().model.installed.value?.some((sdk) => sdk.name === 'localsdk'), 'the removed plugin is no longer listed');

  // Every task but the cancelled one succeeded, and vfox agrees with what
  // the application shows.
  const statuses = model.tasks.tasks.map((task) => task.status).filter((status) => status !== 'succeeded');
  check(
    statuses.length === 1 && statuses[0] === 'cancelled',
    `only the cancelled task did not succeed: ${statuses.join(', ')}`,
  );
  check((await vfox(context, ['current'])) === 'testsdk -> v2.0.0\n', 'vfox uses testsdk 2.0.0');
  check((await vfox(context, ['list', 'testsdk'])).includes('2.0.0'), 'vfox lists testsdk 2.0.0');
  // The copy of the local plugin is deleted once its task has finished.
  const copies = await host.run('find', [`${home}/.vfox/tmp`, '-maxdepth', '1', '-name', 'vfox-plus-plugin-*'], {
    workingDirectory: '/',
    environment,
  });
  check(copies.stdout.trim() === '', `the copy of the local plugin is deleted: ${copies.stdout}`);
}

/**
 * Opens the Home page and the versions of Node.js, so that the harness can
 * measure the memory of the application at both points (see
 * `docs/new-implementation.md`).
 */
export async function measureScenario(context: ScenarioContext): Promise<void> {
  await waitForText('Versions in use');
  await pause(3000);
  await context.step('MEASURE home');
  await pause(5000);
  await clickButton('SDKs');
  await clickElement('the SDK nodejs', 'm3e-list-action', (element) => ownText(element).startsWith('nodejs'));
  await clickButton('Available');
  await waitForText('872 versions');
  await pause(3000);
  await context.step('MEASURE nodejs');
  await pause(5000);
  await clickText('Home').catch(() => undefined);
}
