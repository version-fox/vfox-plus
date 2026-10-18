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
 * The host of `npm run dev`: the development host of the Vite server (see
 * `dev-host/plugin.ts`), reached over a WebSocket. Dialogs, links and the
 * clipboard use what the browser offers.
 */

import type { Host } from '../core/host';
import { channelHost } from './channel-host';
import type { HostEvent, HostResponse } from './protocol';

/** Connects to the development host of the page's server. */
export async function createDevHost(): Promise<Host> {
  const socket = new WebSocket(`${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/__vfox-plus-host`);
  await new Promise<void>((resolve, reject) => {
    socket.addEventListener('open', () => {
      resolve();
    });
    socket.addEventListener('error', () => {
      reject(new Error('The development host is not reachable.'));
    });
  });
  const host = channelHost('browser', {
    send: (request) => {
      socket.send(JSON.stringify(request));
    },
    onMessage: (receive) => {
      socket.addEventListener('message', (event: MessageEvent<string>) => {
        receive(JSON.parse(event.data) as HostResponse | HostEvent);
      });
    },
  });
  return {
    ...host,
    openUrl: (url) => {
      window.open(url, '_blank', 'noopener');
      return Promise.resolve();
    },
    pickDirectory: (initialDirectory) => Promise.resolve(window.prompt('Directory', initialDirectory ?? '')),
    pickFile: (initialDirectory) => Promise.resolve(window.prompt('File', initialDirectory ?? '')),
    copyText: (text) => navigator.clipboard.writeText(text),
  };
}
