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
 * Gives the page the message channel of the host protocol, and nothing else:
 * the page runs sandboxed, without Node.js.
 */

import { contextBridge, ipcRenderer } from 'electron';

import { HOST_CHANNEL, type HostBridge } from './channel.ts';

const bridge: HostBridge = {
  send(request) {
    ipcRenderer.send(HOST_CHANNEL, request);
  },
  onMessage(receive) {
    ipcRenderer.on(HOST_CHANNEL, (_event, message: unknown) => {
      receive(message);
    });
  },
};

contextBridge.exposeInMainWorld('vfoxHost', bridge);
