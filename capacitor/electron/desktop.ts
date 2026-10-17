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
 * What the host of ../web/dev-host/node-host.ts needs from the desktop, with
 * the native dialogs, the clipboard and the browser that Electron offers.
 */

import { app, clipboard, dialog, shell, type BrowserWindow, type OpenDialogOptions } from 'electron';

import type { DesktopOperations } from '../../web/dev-host/node-host.ts';
import { accentColor } from './accent.ts';

/** Shows an open dialog, attached to `window` if there is one, and resolves with the chosen path. */
async function choosePath(window: BrowserWindow | null, options: OpenDialogOptions): Promise<string | null> {
  const result = window === null ? await dialog.showOpenDialog(options) : await dialog.showOpenDialog(window, options);
  return result.canceled ? null : (result.filePaths[0] ?? null);
}

/** The desktop operations for the window that `window` returns. */
export function electronDesktop(window: () => BrowserWindow | null): DesktopOperations {
  return {
    accentColor: () => accentColor(),
    openUrl: async (url) => {
      // Only web pages: the front end never asks for anything else.
      if (/^https?:\/\//i.test(url)) {
        await shell.openExternal(url);
      }
    },
    pickDirectory: (initialDirectory) =>
      choosePath(window(), {
        properties: ['openDirectory', 'createDirectory'],
        ...(initialDirectory === null ? {} : { defaultPath: initialDirectory }),
      }),
    pickFile: (initialDirectory, label, extensions) =>
      choosePath(window(), {
        properties: ['openFile'],
        ...(initialDirectory === null ? {} : { defaultPath: initialDirectory }),
        ...(extensions === null ? {} : { filters: [{ name: label ?? extensions.join(', '), extensions }] }),
      }),
    copyText: (text) => clipboard.writeText(text),
    exit: (code) => {
      app.exit(code);
      return Promise.resolve();
    },
    log: (message) => {
      process.stdout.write(`${message}\n`);
      return Promise.resolve();
    },
    resizeWindow: (width, height) => {
      window()?.setContentSize(width, height);
      return Promise.resolve();
    },
  };
}
