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

import { beforeEach, describe, expect, it } from 'vitest';

import { isValidPluginName, launchConfig, VfoxCli, vfoxEnvironment } from '../../src/core/vfox/cli';
import { commandLine, commandSubject, VfoxError, VfoxNotFoundError } from '../../src/core/vfox/command';
import { VfoxLocator } from '../../src/core/vfox/locator';
import { fixtureOutput, fixtureRunner, type FakeProcessRunner } from '../support/fakes';

let runner: FakeProcessRunner;
let cli: VfoxCli;
const listener = { stdout: () => undefined, stderr: () => undefined };

beforeEach(() => {
  runner = fixtureRunner();
  cli = new VfoxCli(
    runner,
    launchConfig({
      executable: '/usr/local/bin/vfox',
      baseEnvironment: {
        HOME: '/home/user',
        PATH: '/usr/bin',
        __VFOX_SHELL: 'bash',
        __VFOX_PID: '42',
        __VFOX_CURTMPPATH: '/home/user/.vfox/tmp/x',
      },
      extraEnvironment: { VFOX_NODEJS_MIRROR: 'https://mirror', PATH: '/opt/bin' },
    }),
  );
});

describe('launch configuration', () => {
  it('removes the variables of an activated shell and disables colours', () => {
    const env = vfoxEnvironment(cli.launchConfig);
    expect(env).not.toHaveProperty('__VFOX_SHELL');
    expect(env).not.toHaveProperty('__VFOX_PID');
    expect(env).not.toHaveProperty('__VFOX_CURTMPPATH');
    expect(env['NO_COLOR']).toBe('1');
    expect(env['HOME']).toBe('/home/user');
  });

  it('lets user variables override inherited ones', () => {
    const env = vfoxEnvironment(cli.launchConfig);
    expect(env['PATH']).toBe('/opt/bin');
    expect(env['VFOX_NODEJS_MIRROR']).toBe('https://mirror');
  });

  it('marks the hook environment only when asked to', () => {
    expect(vfoxEnvironment(cli.launchConfig, { hookShell: true })['__VFOX_SHELL']).toBe('bash');
  });
});

describe('queries', () => {
  it('run in the neutral directory with the configured executable', async () => {
    await cli.version();
    const call = runner.invocations[0];
    expect(runner.invocations).toHaveLength(1);
    expect(call?.executable).toBe('/usr/local/bin/vfox');
    expect(call?.workingDirectory).toBe('/');
    expect(call?.environment['NO_COLOR']).toBe('1');
  });

  it('version', async () => {
    expect(await cli.version()).toBe('1.0.13');
  });

  it('rejects an executable that does not print a version', async () => {
    runner.on('--version', { exitCode: 0, stdout: 'hello\n', stderr: '' });
    await expect(cli.version()).rejects.toThrow('Unrecognised version output');
  });

  it('installedSdks merges list and current', async () => {
    expect(await cli.installedSdks()).toEqual([
      { name: 'maven', versions: ['3.9.16', '3.8.9'], current: '3.9.16' },
      { name: 'mvn2', versions: [], current: null },
      { name: 'nodejs', versions: ['24.11.1', '22.21.0'], current: '24.11.1' },
    ]);
  });

  it('installedSdks is empty when no plugin is installed', async () => {
    runner.on('list', fixtureOutput('list-empty', 1)).on('current', fixtureOutput('current-empty'));
    expect(await cli.installedSdks()).toEqual([]);
  });

  it('installedSdks reports a failing list', async () => {
    runner.on('list', { exitCode: 1, stdout: 'boom\n', stderr: '' });
    const error: unknown = await cli.installedSdks().catch((reason: unknown) => reason);
    expect(error).toBeInstanceOf(VfoxError);
    expect((error as VfoxError).message).toBe('boom');
    expect((error as VfoxError).arguments).toEqual(['list']);
    expect((error as VfoxError).exitCode).toBe(1);
  });

  it('pluginInfo uses the format template', async () => {
    const info = await cli.pluginInfo('nodejs');
    expect(info.version).toBe('0.5.1');
    expect(runner.invocations[0]?.arguments.slice(0, 3)).toEqual(['info', 'nodejs', '--format']);
  });

  it('pluginInfo rejects output without fields', async () => {
    await expect(cli.pluginInfo('maven')).rejects.toThrow(VfoxError);
  });

  it('searchVersions', async () => {
    const versions = await cli.searchVersions('maven');
    expect(versions.map((version) => version.version)).toContain('3.9.16');
  });

  it('availablePlugins', async () => {
    expect(await cli.availablePlugins()).toHaveLength(43);
  });

  it('configuration is listed, set and reset', async () => {
    runner
      .on('config proxy.url http://proxy:8080', { exitCode: 0, stdout: '', stderr: '' })
      .on('config --unset proxy.url', { exitCode: 0, stdout: '', stderr: '' });
    expect(await cli.configEntries()).toHaveLength(10);
    await cli.setConfig('proxy.url', 'http://proxy:8080');
    await cli.unsetConfig('proxy.url');
    expect(runner.calls('config').map((call) => call.arguments.join(' '))).toEqual([
      'config --list',
      'config proxy.url http://proxy:8080',
      'config --unset proxy.url',
    ]);
  });

  it('a rejected value becomes a VfoxError with the message of vfox', async () => {
    runner.on('config cache.availableHookDuration notaduration', fixtureOutput('config-set-invalid-duration', 1));
    await expect(cli.setConfig('cache.availableHookDuration', 'notaduration')).rejects.toThrow(
      'time: invalid duration "notaduration"',
    );
  });

  it('reports the exit code when vfox prints nothing', async () => {
    runner.on('config --unset proxy.url', { exitCode: 3, stdout: '', stderr: '' });
    await expect(cli.unsetConfig('proxy.url')).rejects.toThrow('vfox exited with code 3');
  });

  it('installPath', async () => {
    runner
      .on('info nodejs@24.11.1', fixtureOutput('info-version-path'))
      .on('info nodejs@1.0.0', fixtureOutput('info-version-not-installed'));
    expect(await cli.installPath('nodejs', '24.11.1')).toBe('/tmp/vfox-sb/home/.vfox/cache/nodejs/v-24.11.1');
    expect(await cli.installPath('nodejs', '1.0.0')).toBeNull();
  });

  it('a missing executable becomes a VfoxNotFoundError', async () => {
    runner.missingExecutable = true;
    await expect(cli.version()).rejects.toBeInstanceOf(VfoxNotFoundError);
  });
});

describe('commands', () => {
  it('install answers prompts with --yes', () => {
    const command = cli.install('nodejs', '24.11.1');
    expect(command.arguments).toEqual(['install', '--yes', 'nodejs@24.11.1']);
    expect(command.operation).toBe('install');
    expect(commandSubject(command)).toBe('nodejs@24.11.1');
    expect(commandLine(command)).toBe('vfox install --yes nodejs@24.11.1');
    expect(command.workingDirectory).toBeNull();
  });

  it('use in the global scope needs the hook environment', () => {
    const command = cli.use('nodejs', '24.11.1');
    expect(command.arguments).toEqual(['use', '--global', 'nodejs@24.11.1']);
    expect(command.requiresHookEnvironment).toBe(true);
  });

  it('use in a project runs in the project directory', () => {
    const command = cli.use('java', '21', 'project', '/work/app');
    expect(command.arguments).toEqual(['use', '--project', 'java@21']);
    expect(command.workingDirectory).toBe('/work/app');
  });

  it('the project scope requires a directory', () => {
    expect(() => cli.use('java', '21', 'project')).toThrow(RangeError);
    expect(() => cli.unuse('java', 'project', ' ')).toThrow(RangeError);
  });

  it('unuse, uninstall, remove and update', () => {
    expect(cli.unuse('java').arguments).toEqual(['unuse', '--global', 'java']);
    expect(cli.unuse('java', 'project', '/p').workingDirectory).toBe('/p');
    expect(cli.uninstall('java', '21').arguments).toEqual(['uninstall', 'java@21']);
    expect(cli.removePlugin('java').arguments).toEqual(['remove', '--yes', 'java']);
    expect(cli.updatePlugin('java').arguments).toEqual(['update', 'java']);
    expect(cli.updateAllPlugins().arguments).toEqual(['update', '--all']);
    expect(commandSubject(cli.updateAllPlugins())).toBeNull();
  });

  it('addPlugin from the registry or from a source', () => {
    expect(cli.addPlugin('nodejs').arguments).toEqual(['add', 'nodejs']);
    const custom = cli.addPlugin('', { source: ' https://x/p.zip ', alias: 'mine' });
    expect(custom.arguments).toEqual(['add', '--source', 'https://x/p.zip', '--alias', 'mine']);
    expect(custom.sdk).toBe('mine');
    expect(cli.addPlugin('', { source: '/p.lua' }).sdk).toBeNull();
    expect(() => cli.addPlugin('  ')).toThrow(RangeError);
  });

  it('rejects names and versions that look like options', () => {
    expect(() => cli.install('--all', '1')).toThrow(RangeError);
    expect(() => cli.install('java', '-1')).toThrow(RangeError);
    expect(() => cli.install('java', '1 2')).toThrow(RangeError);
    expect(() => cli.install('java', 'a@b')).toThrow(RangeError);
    expect(() => cli.install('java', '')).toThrow(RangeError);
    expect(isValidPluginName('my-sdk.2')).toBe(true);
    expect(isValidPluginName('-x')).toBe(false);
  });

  it('rejects malformed configuration keys', async () => {
    await expect(cli.setConfig('proxy..url', 'x')).rejects.toThrow(RangeError);
  });

  it('start launches the command with the right environment', async () => {
    await cli.start(cli.use('java', '21', 'project', '/work/app'), listener);
    const call = runner.invocations[0];
    expect(call?.streamed).toBe(true);
    expect(call?.workingDirectory).toBe('/work/app');
    expect(call?.environment['__VFOX_SHELL']).toBe('bash');
  });

  it('start reports a missing executable', async () => {
    runner.missingExecutable = true;
    await expect(cli.start(cli.updateAllPlugins(), listener)).rejects.toBeInstanceOf(VfoxNotFoundError);
  });
});

describe('VfoxLocator', () => {
  it('searches PATH first, then the usual install locations, without duplicates', () => {
    const locator = new VfoxLocator({ PATH: '/opt/bin::/usr/bin', HOME: '/home/me' }, () => Promise.resolve(false));
    expect(locator.searchDirectories).toEqual([
      '/opt/bin',
      '/usr/bin',
      '/usr/local/bin',
      '/home/me/.local/bin',
      '/home/me/bin',
      '/home/me/.linuxbrew/bin',
      '/home/linuxbrew/.linuxbrew/bin',
      '/snap/bin',
    ]);
  });

  it('finds the first executable', async () => {
    const locator = new VfoxLocator({ HOME: '/home/me' }, (path) =>
      Promise.resolve(path === '/home/me/.local/bin/vfox' || path === '/snap/bin/vfox'),
    );
    expect(await locator.locate()).toBe('/home/me/.local/bin/vfox');
  });

  it('returns null without vfox', async () => {
    expect(await new VfoxLocator({}, () => Promise.resolve(false)).locate()).toBeNull();
    expect(new VfoxLocator({}, () => Promise.resolve(false)).fallbackDirectories).not.toContain('undefined/bin');
  });
});
