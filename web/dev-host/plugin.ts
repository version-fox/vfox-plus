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
 * The development host: lets the front end run in a browser during
 * development, with `npm run dev`. The Vite server answers the host protocol
 * of `src/hosts/protocol.ts` on a WebSocket, with Node.js doing what Tauri or
 * Wails do in the real applications.
 *
 * vfox runs with a private home directory (`VFOX_PLUS_DEV_HOME`, by default a
 * new temporary directory), so the vfox installation of the developer is never
 * touched; `VFOX_PLUS_DEV_VFOX` chooses the vfox executable, by default the
 * fake vfox of `e2e/fake-vfox.sh`, which replays `shared/fixtures`. Never
 * expose the development server beyond the machine: anybody who reaches it
 * can run vfox.
 */

import { mkdirSync, mkdtempSync, symlinkSync } from 'node:fs';
import type { IncomingMessage } from 'node:http';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import type { Duplex } from 'node:stream';
import { fileURLToPath } from 'node:url';
import type { Plugin } from 'vite';
import { WebSocketServer, type WebSocket } from 'ws';

import type { HostRequest } from '../src/hosts/protocol.ts';
import { environmentOf, NodeHost } from './node-host.ts';

/** The path of the WebSocket of the development host. */
export const DEV_HOST_PATH = '/__vfox-plus-host';

function prepareHome(): { home: string; vfox: string } {
  const home = process.env['VFOX_PLUS_DEV_HOME'] ?? mkdtempSync(join(tmpdir(), 'vfox-plus-dev-'));
  const vfox = process.env['VFOX_PLUS_DEV_VFOX'] ?? fileURLToPath(new URL('../e2e/fake-vfox.sh', import.meta.url));
  // The locator finds vfox in the PATH of the front end.
  const bin = join(home, 'bin');
  mkdirSync(bin, { recursive: true });
  try {
    symlinkSync(vfox, join(bin, 'vfox'));
  } catch {
    // Already linked by an earlier run.
  }
  return { home, vfox };
}

/** The Vite plugin of the development host. */
export function devHost(): Plugin {
  return {
    name: 'vfox-plus-dev-host',
    configureServer(server) {
      const { home } = prepareHome();
      const fixtures = fileURLToPath(new URL('../../shared/fixtures/vfox-1.0.13', import.meta.url));
      const environment = {
        ...environmentOf(process.env),
        HOME: home,
        PATH: `${join(home, 'bin')}:${process.env['PATH'] ?? '/usr/bin:/bin'}`,
        VFOX_PLUS_FIXTURES: process.env['VFOX_PLUS_FIXTURES'] ?? fixtures,
      };
      delete (environment as Record<string, string | undefined>)['VFOX_HOME'];
      delete (environment as Record<string, string | undefined>)['XDG_CONFIG_HOME'];
      server.config.logger.info(`vfox+ development host: HOME=${home}`);

      const sockets = new WebSocketServer({ noServer: true });
      server.httpServer?.on('upgrade', (request: IncomingMessage, socket: Duplex, head: Buffer) => {
        if (request.url !== DEV_HOST_PATH) {
          return;
        }
        sockets.handleUpgrade(request, socket, head, (ws: WebSocket) => {
          const host = new NodeHost(
            {
              environment: () => environment,
              accentColor: () => Promise.resolve(null),
              // The browser opens links, shows prompts and copies by itself.
              openUrl: () => Promise.resolve(),
              pickDirectory: () => Promise.resolve(null),
              pickFile: () => Promise.resolve(null),
              copyText: () => Promise.resolve(),
              exit: (code) => {
                server.config.logger.info(`vfox+ asked to exit with code ${code}`);
                return Promise.resolve();
              },
              log: (message) => {
                server.config.logger.info(`vfox+: ${message}`);
                return Promise.resolve();
              },
            },
            (event) => {
              ws.send(JSON.stringify(event));
            },
          );
          ws.on('message', (data: Buffer) => {
            const request = JSON.parse(data.toString('utf8')) as HostRequest;
            void host.handle(request).then((response) => {
              ws.send(JSON.stringify(response));
            });
          });
          ws.on('close', () => {
            host.close();
          });
        });
      });
    },
  };
}
