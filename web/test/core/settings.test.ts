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

import { describe, expect, it } from 'vitest';

import {
  DEFAULT_SEED_COLOR,
  DEFAULT_SETTINGS,
  environmentEqual,
  formatEnvironmentLines,
  parseEnvironmentLines,
  settingsEqual,
  settingsFromJson,
  settingsToJson,
  type AppSettings,
} from '../../src/core/settings/app-settings';
import { JsonFileSettingsStore, settingsPath } from '../../src/core/settings/settings-store';
import {
  isRemotePluginSource,
  preparePluginSource,
  vfoxMovesPluginSource,
  vfoxTemporaryDirectory,
} from '../../src/core/vfox/plugin-source';
import { MemoryFileSystem } from '../support/fakes';

describe('AppSettings', () => {
  it('round-trips through JSON', () => {
    const settings: AppSettings = {
      theme: 'dark',
      useSystemColor: false,
      seedColor: 0xff123456,
      language: 'simplifiedChinese',
      vfoxExecutable: '/opt/vfox',
      extraEnvironment: { A: '1' },
      railExpanded: true,
      unknown: {},
    };
    expect(settingsFromJson(JSON.parse(JSON.stringify(settingsToJson(settings))))).toEqual(settings);
  });

  it('writes the keys of the specification in order', () => {
    expect(Object.keys(settingsToJson(DEFAULT_SETTINGS))).toEqual([
      'theme',
      'useSystemColor',
      'seedColor',
      'language',
      'extraEnvironment',
      'railExpanded',
    ]);
  });

  it('ignores malformed values and keeps unknown keys', () => {
    const settings = settingsFromJson({
      theme: 'purple',
      useSystemColor: 'yes',
      seedColor: '#fff',
      language: 42,
      vfoxExecutable: '',
      extraEnvironment: { A: '1', B: 2 },
      future: true,
    });
    expect(settings.theme).toBe('system');
    expect(settings.useSystemColor).toBe(true);
    expect(settings.seedColor).toBe(DEFAULT_SEED_COLOR);
    expect(settings.language).toBe('system');
    expect(settings.vfoxExecutable).toBeNull();
    expect(settings.extraEnvironment).toEqual({ A: '1' });
    expect(settingsToJson(settings)['future']).toBe(true);
    expect(settingsFromJson([1, 2])).toBe(DEFAULT_SETTINGS);
    expect(settingsFromJson({ extraEnvironment: ['x'] }).extraEnvironment).toEqual({});
  });

  it('compares by value', () => {
    expect(settingsEqual(DEFAULT_SETTINGS, { ...DEFAULT_SETTINGS, extraEnvironment: {} })).toBe(true);
    expect(settingsEqual(DEFAULT_SETTINGS, { ...DEFAULT_SETTINGS, railExpanded: true })).toBe(false);
    expect(environmentEqual({ A: '1' }, { A: '1' })).toBe(true);
    expect(environmentEqual({ A: '1' }, { A: '2' })).toBe(false);
    expect(environmentEqual({ A: '1' }, { A: '1', B: '2' })).toBe(false);
  });
});

describe('environment lines', () => {
  it('parse KEY=VALUE, skipping comments and blank lines', () => {
    const result = parseEnvironmentLines('# mirrors\nA=1\n\n B = x=y \nnot valid\n1X=2\n');
    expect(result.variables).toEqual({ A: '1', B: 'x=y' });
    expect(result.invalidLines).toEqual([5, 6]);
  });

  it('format sorted by key', () => {
    expect(formatEnvironmentLines({ B: '2', A: '1' })).toBe('A=1\nB=2');
  });
});

describe('JsonFileSettingsStore', () => {
  it('uses XDG_CONFIG_HOME and falls back to ~/.config', () => {
    expect(settingsPath({ XDG_CONFIG_HOME: '/x', HOME: '/h' })).toBe('/x/vfox-plus/settings.json');
    expect(settingsPath({ XDG_CONFIG_HOME: 'relative', HOME: '/h' })).toBe('/h/.config/vfox-plus/settings.json');
    expect(settingsPath({}, '/fallback')).toBe('/fallback/.config/vfox-plus/settings.json');
  });

  it('returns defaults without a file, and what was saved afterwards', async () => {
    const files = new MemoryFileSystem();
    const store = new JsonFileSettingsStore(files, '/h/nested/settings.json');
    expect(await store.load()).toEqual(DEFAULT_SETTINGS);
    const settings: AppSettings = { ...DEFAULT_SETTINGS, theme: 'light' };
    await store.save(settings);
    expect(await store.load()).toEqual(settings);
    expect(files.files.get('/h/nested/settings.json')?.endsWith('}\n')).toBe(true);
  });

  it('returns defaults for a corrupted file', async () => {
    const files = new MemoryFileSystem();
    files.files.set('/h/settings.json', '{not json');
    expect(await new JsonFileSettingsStore(files, '/h/settings.json').load()).toEqual(DEFAULT_SETTINGS);
  });
});

describe('plugin sources', () => {
  it('only local .lua files are moved by vfox', () => {
    expect(vfoxMovesPluginSource('/home/u/plugin.lua')).toBe(true);
    expect(vfoxMovesPluginSource('/home/u/PLUGIN.LUA')).toBe(true);
    expect(vfoxMovesPluginSource('/home/u/plugin.zip')).toBe(false);
    expect(vfoxMovesPluginSource('https://example.com/plugin.lua')).toBe(false);
    expect(isRemotePluginSource('http://127.0.0.1:8080/p.zip')).toBe(true);
  });

  it('the temporary directory of vfox is in its home directory', async () => {
    const files = new MemoryFileSystem();
    expect(await vfoxTemporaryDirectory({ HOME: '/h' }, files)).toBe('/h/.vfox/tmp');
    await files.createDirectory('/h/.version-fox');
    expect(await vfoxTemporaryDirectory({ HOME: '/h' }, files)).toBe('/h/.version-fox/tmp');
    expect(await vfoxTemporaryDirectory({}, files)).toBe('/tmp');
  });

  it('URLs and archives are passed through', async () => {
    const files = new MemoryFileSystem();
    for (const source of ['https://example.com/p.lua', '/h/p.zip']) {
      const prepared = await preparePluginSource(source, '/h/.vfox/tmp', files);
      expect(prepared.path).toBe(source);
      expect(prepared.isCopy).toBe(false);
      await prepared.dispose();
    }
  });

  it('a local .lua file is copied and the original is kept', async () => {
    const files = new MemoryFileSystem();
    files.files.set('/h/mine.lua', 'PLUGIN = {}');
    const prepared = await preparePluginSource('/h/mine.lua', '/h/.vfox/tmp', files);
    expect(prepared.isCopy).toBe(true);
    expect(prepared.path).not.toBe('/h/mine.lua');
    expect(prepared.path.endsWith('/mine.lua')).toBe(true);
    expect(files.files.get(prepared.path)).toBe('PLUGIN = {}');
    await prepared.dispose();
    expect(files.files.has(prepared.path)).toBe(false);
    expect(files.files.has('/h/mine.lua')).toBe(true);
    expect([...files.directories].filter((path) => path.startsWith('/h/.vfox/tmp/'))).toEqual([]);
  });

  it('a file that cannot be copied is left for vfox to report', async () => {
    const files = new MemoryFileSystem();
    const prepared = await preparePluginSource('/h/missing.lua', '/h/tmp', files);
    expect(prepared.path).toBe('/h/missing.lua');
    expect(prepared.isCopy).toBe(false);
    expect([...files.directories].filter((path) => path.startsWith('/h/tmp/'))).toEqual([]);
  });
});
