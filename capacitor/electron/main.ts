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
 * The Electron main process: one window with the Ionic front end of src/,
 * and the host of the host protocol (../web/src/hosts/protocol.ts), which
 * runs vfox and reads and writes files with Node.js on behalf of the page,
 * like the development host of ../web. The page reaches it through the
 * VfoxHost Capacitor plugin (src/capacitor/) and the bridge of preload.ts.
 */

import { homedir } from 'node:os';
import { join, normalize, sep } from 'node:path';
import { pathToFileURL } from 'node:url';

import { app, BrowserWindow, ipcMain, Menu, net, protocol, session, type IpcMainEvent } from 'electron';

import { NodeHost } from '../../web/dev-host/node-host.ts';
import type { HostRequest } from '../../web/src/hosts/protocol.ts';
import { HOST_CHANNEL } from './channel.ts';
import { electronDesktop } from './desktop.ts';

/** The application ID, after which the desktop entry and the icon are named. */
const APP_ID = 'dev.vfox.VfoxPlus';
/** The page is served from this origin, not from file: URLs. */
const SCHEME = 'app';
const ORIGIN = `${SCHEME}://vfox-plus`;
const RENDERER = join(__dirname, 'renderer');

protocol.registerSchemesAsPrivileged([
  { scheme: SCHEME, privileges: { standard: true, secure: true, supportFetchAPI: true } },
]);
// No hardware acceleration: Skia rasterizes on the CPU, which keeps Mesa's software
// OpenGL and its compiler out of the GPU process when there is no real GPU.
app.disableHardwareAcceleration();
// And fewer processes with memory of their own: the GPU and the network service run
// in the browser process, and V8 leaves out its optimizing compiler.
app.commandLine.appendSwitch('in-process-gpu');
app.commandLine.appendSwitch('enable-features', 'NetworkServiceInProcess2');
app.commandLine.appendSwitch('js-flags', '--lite-mode');
// X11 takes WM_CLASS from the name, which the desktop entry names (StartupWMClass).
app.setName(APP_ID);
// Chromium keeps its profile apart from the settings of vfox+, among the caches.
const cache = process.env['XDG_CACHE_HOME'];
app.setPath(
  'userData',
  join(cache === undefined || cache === '' ? join(homedir(), '.cache') : cache, 'vfox-plus', 'electron'),
);

/** Serves the files of the page; nothing outside of it. */
function serve(request: Request): Promise<Response> | Response {
  const url = new URL(request.url);
  const file = normalize(join(RENDERER, decodeURIComponent(url.pathname)));
  // The URL class of Node.js gives schemes that it does not know no origin.
  if (`${url.protocol}//${url.host}` !== ORIGIN || !file.startsWith(RENDERER + sep)) {
    return new Response(null, { status: 404 });
  }
  return net.fetch(pathToFileURL(file).toString());
}

function createWindow(): void {
  const window = new BrowserWindow({
    width: 1280,
    height: 820,
    minWidth: 360,
    minHeight: 480,
    title: 'vfox+',
    show: false,
    autoHideMenuBar: true,
    webPreferences: {
      preload: join(__dirname, 'preload.cjs'),
      sandbox: true,
      contextIsolation: true,
      nodeIntegration: false,
      spellcheck: false,
    },
  });
  const contents = window.webContents;
  const send = (message: unknown): void => {
    if (!contents.isDestroyed()) {
      contents.send(HOST_CHANNEL, message);
    }
  };
  const host = new NodeHost(
    electronDesktop(() => (window.isDestroyed() ? null : window)),
    send,
  );
  const onRequest = (event: IpcMainEvent, request: HostRequest): void => {
    if (event.sender === contents && event.senderFrame?.origin === ORIGIN) {
      void host.handle(request).then(send);
    }
  };
  ipcMain.on(HOST_CHANNEL, onRequest);
  window.on('closed', () => {
    ipcMain.off(HOST_CHANNEL, onRequest);
    // Operations of vfox that still run end with the window.
    host.close();
  });
  window.once('ready-to-show', () => {
    window.show();
  });
  // The page never navigates; links open in the browser through the host.
  contents.on('will-navigate', (event) => {
    event.preventDefault();
  });
  contents.setWindowOpenHandler(() => ({ action: 'deny' }));
  // Errors of the page belong in the output of the application.
  contents.on('console-message', (event) => {
    if (event.level === 'error') {
      process.stderr.write(`${event.message} (${event.sourceId}:${event.lineNumber})\n`);
    }
  });
  contents.on('render-process-gone', (_event, details) => {
    process.stderr.write(`The page ended: ${details.reason} (exit code ${details.exitCode})\n`);
    app.exit(1);
  });
  void window.loadURL(`${ORIGIN}/index.html`);
}

void app.whenReady().then(() => {
  Menu.setApplicationMenu(null);
  session.defaultSession.setPermissionRequestHandler((_contents, _permission, callback) => {
    callback(false);
  });
  protocol.handle(SCHEME, serve);
  createWindow();
});

app.on('window-all-closed', () => {
  app.quit();
});
