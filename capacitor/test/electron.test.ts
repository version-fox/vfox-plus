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

// The main process: the accent colour of the desktop and the desktop
// operations, with Electron replaced by test doubles.

import { beforeEach, describe, expect, it, vi } from 'vitest';

const electron = vi.hoisted(() => ({
  app: { exit: vi.fn() },
  clipboard: { writeText: vi.fn(() => Promise.resolve()) },
  dialog: { showOpenDialog: vi.fn() },
  shell: { openExternal: vi.fn(() => Promise.resolve()) },
}));
vi.mock('electron', () => electron);

import { accentColor, parseAccentColor } from '../electron/accent';
import { electronDesktop } from '../electron/desktop';

describe('accent colour', () => {
  it('reads the reply of the desktop portal', () => {
    expect(parseAccentColor('(<(0.2, 0.4, 0.6)>,)\n')).toBe(0xff336699);
    expect(parseAccentColor('(<<(1.0, 0.5490196078431373, 0.10588235294117647)>>,)')).toBe(0xffff8c1b);
    expect(parseAccentColor('(<(0, 0, 0)>,)')).toBe(0xff000000);
  });

  it('has none when the desktop sets none', () => {
    expect(parseAccentColor('(<(-1.0, -1.0, -1.0)>,)')).toBeNull();
    expect(parseAccentColor('(<(2, 0, 0)>,)')).toBeNull();
    expect(parseAccentColor('(<(nan, 0, 0)>,)')).toBeNull();
    expect(parseAccentColor('')).toBeNull();
  });

  it('asks the portal with gdbus and survives its absence', async () => {
    const run = vi.fn(() => Promise.resolve('(<(0.2, 0.4, 0.6)>,)'));
    expect(await accentColor(run)).toBe(0xff336699);
    expect(run).toHaveBeenCalledWith('gdbus', expect.arrayContaining(['org.freedesktop.appearance', 'accent-color']));
    expect(await accentColor(() => Promise.reject(new Error('no session bus')))).toBeNull();
  });
});

describe('desktop operations', () => {
  const window = { setContentSize: vi.fn() };
  const desktop = electronDesktop(() => window as never);

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('opens only web pages', async () => {
    await desktop.openUrl('https://vfox.dev/');
    await desktop.openUrl('file:///etc/passwd');
    expect(electron.shell.openExternal).toHaveBeenCalledTimes(1);
    expect(electron.shell.openExternal).toHaveBeenCalledWith('https://vfox.dev/');
  });

  it('chooses directories and files in dialogs of the window', async () => {
    electron.dialog.showOpenDialog.mockResolvedValueOnce({ canceled: false, filePaths: ['/home/me/project'] });
    expect(await desktop.pickDirectory('/home/me')).toBe('/home/me/project');
    expect(electron.dialog.showOpenDialog).toHaveBeenLastCalledWith(window, {
      properties: ['openDirectory', 'createDirectory'],
      defaultPath: '/home/me',
    });

    electron.dialog.showOpenDialog.mockResolvedValueOnce({ canceled: false, filePaths: ['/tmp/plugin.lua'] });
    expect(await desktop.pickFile(null, 'vfox plugins', ['zip', 'lua'])).toBe('/tmp/plugin.lua');
    expect(electron.dialog.showOpenDialog).toHaveBeenLastCalledWith(window, {
      properties: ['openFile'],
      filters: [{ name: 'vfox plugins', extensions: ['zip', 'lua'] }],
    });

    electron.dialog.showOpenDialog.mockResolvedValueOnce({ canceled: true, filePaths: [] });
    expect(await desktop.pickFile('/usr/local/bin', null, null)).toBeNull();
  });

  it('shows dialogs without a window and names filters by their extensions', async () => {
    const headless = electronDesktop(() => null);
    electron.dialog.showOpenDialog.mockResolvedValueOnce({ canceled: false, filePaths: [] });
    expect(await headless.pickFile(null, null, ['zip'])).toBeNull();
    expect(electron.dialog.showOpenDialog).toHaveBeenLastCalledWith({
      properties: ['openFile'],
      filters: [{ name: 'zip', extensions: ['zip'] }],
    });
    await headless.resizeWindow?.(412, 800);
  });

  it('copies, logs, resizes and exits', async () => {
    const write = vi.spyOn(process.stdout, 'write').mockImplementation(() => true);
    await desktop.copyText('eval "$(vfox activate bash)"');
    await desktop.log('e2e: home');
    await desktop.resizeWindow?.(412, 800);
    await desktop.exit(3);
    expect(electron.clipboard.writeText).toHaveBeenCalledWith('eval "$(vfox activate bash)"');
    expect(write).toHaveBeenCalledWith('e2e: home\n');
    expect(window.setContentSize).toHaveBeenCalledWith(412, 800);
    expect(electron.app.exit).toHaveBeenCalledWith(3);
    write.mockRestore();
  });
});
